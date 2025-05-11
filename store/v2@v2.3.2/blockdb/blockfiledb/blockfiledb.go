/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0

*/

package blockfiledb

import (
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"runtime"
	"strconv"
	"sync"
	"time"

	bytesconv "chainmaker.org/chainmaker/common/v2/bytehelper"
	commonPb "chainmaker.org/chainmaker/pb-go/v2/common"
	storePb "chainmaker.org/chainmaker/pb-go/v2/store"
	"chainmaker.org/chainmaker/protocol/v2"
	"chainmaker.org/chainmaker/store/v2/binlog"
	"chainmaker.org/chainmaker/store/v2/cache"
	"chainmaker.org/chainmaker/store/v2/conf"
	"chainmaker.org/chainmaker/store/v2/serialization"
	"chainmaker.org/chainmaker/store/v2/types"
	su "chainmaker.org/chainmaker/store/v2/utils"
	"chainmaker.org/chainmaker/utils/v2"
	"github.com/gogo/protobuf/proto"
	"golang.org/x/sync/semaphore"
)

// nolint
var DbType_Mysql = "mysql"

const (
	blockNumIdxKeyPrefix  = 'n'
	blockHashIdxKeyPrefix = 'h'

	blockTxIDIdxKeyPrefix   = 'b'
	blockIndexKeyPrefix     = "ib"
	blockMetaIndexKeyPrefix = "im"
	lastBlockNumKeyStr      = "lastBlockNumKey"
	lastConfigBlockNumKey   = "lastConfigBlockNumKey"
	archivedPivotKey        = "archivedPivotKey"
)

var (
	errValueNotFound = errors.New("value not found")
	errGetBatchPool  = errors.New("get updatebatch error")
)

// BlockFileDB provider a implementation of `blockdb.BlockDB`
// This implementation provides a key-value based data model
type BlockFileDB struct {
	dbHandle         protocol.DBHandle
	workersSemaphore *semaphore.Weighted
	worker           int64
	cache            *cache.StoreCacheMgr
	archivedPivot    uint64
	logger           protocol.Logger
	sync.Mutex
	batchPool   sync.Pool
	storeConfig *conf.StorageConfig
	fileStore   binlog.BinLogger
}

// NewBlockFileDB construct BlockFileDB
//  @Description:
//  @param chainId
//  @param dbHandle
//  @param logger
//  @param storeConfig
//  @param fileStore
//  @return *BlockFileDB
func NewBlockFileDB(chainId string, dbHandle protocol.DBHandle, logger protocol.Logger,
	storeConfig *conf.StorageConfig, fileStore binlog.BinLogger) *BlockFileDB {
	nWorkers := int64(runtime.NumCPU())
	b := &BlockFileDB{
		dbHandle:         dbHandle,
		worker:           nWorkers,
		workersSemaphore: semaphore.NewWeighted(nWorkers),
		cache:            cache.NewStoreCacheMgr(chainId, 10, logger),
		archivedPivot:    0,
		logger:           logger,
		batchPool:        sync.Pool{},
		storeConfig:      storeConfig,
		fileStore:        fileStore,
	}
	b.batchPool.New = func() interface{} {
		return types.NewUpdateBatch()
	}

	return b
}

// InitGenesis init genesis block
//  @Description:
//  @receiver b
//  @param genesisBlock
//  @return error
func (b *BlockFileDB) InitGenesis(genesisBlock *serialization.BlockWithSerializedInfo) error {
	//update Cache
	err := b.CommitBlock(genesisBlock, true)
	if err != nil {
		return err
	}
	//update BlockFileDB
	return b.CommitBlock(genesisBlock, false)
}

// CommitBlock commits the block and the corresponding rwsets in an atomic operation
//  @Description:
//  @receiver b
//  @param blockInfo
//  @param isCache
//  @return error
func (b *BlockFileDB) CommitBlock(blockInfo *serialization.BlockWithSerializedInfo, isCache bool) error {
	//原则上，写db失败，重试一定次数后，仍然失败，panic

	//如果是更新cache，则 直接更新 然后返回，不写 blockdb
	if isCache {
		return b.CommitCache(blockInfo)
	}
	//如果不是更新cache，则 直接更新 kvdb，然后返回
	return b.CommitDB(blockInfo)
}

// CommitCache 提交数据到cache
//  @Description:
//  @receiver b
//  @param blockInfo
//  @return error
func (b *BlockFileDB) CommitCache(blockInfo *serialization.BlockWithSerializedInfo) error {
	//如果是更新cache，则 直接更新 然后返回，不写 blockdb
	//从对象池中取一个对象,并重置
	start := time.Now()
	batch, ok := b.batchPool.Get().(*types.UpdateBatch)
	if !ok {
		b.logger.Errorf("chain[%s]: blockInfo[%d] get updatebatch error",
			blockInfo.Block.Header.ChainId, blockInfo.Block.Header.BlockHeight)
		return errGetBatchPool
	}
	batch.ReSet()
	dbType := b.dbHandle.GetDbType()
	//batch := types.NewUpdateBatch()
	// 1. last blockInfo height
	block := blockInfo.Block
	batch.Put([]byte(lastBlockNumKeyStr), encodeBlockNum(block.Header.BlockHeight))

	// 2. height-> blockInfo
	if blockInfo.Index == nil || blockInfo.MetaIndex == nil {
		return errors.New("blockInfo.Index and blockInfo.MetaIndex must not nil while block rfile db enabled")

	}
	blockIndexKey := constructBlockIndexKey(dbType, block.Header.BlockHeight)
	blockIndexInfo, err := proto.Marshal(blockInfo.Index)
	if err != nil {
		return err
	}
	batch.Put(blockIndexKey, blockIndexInfo)
	b.logger.Debugf("put block[%d] rfile index:%v", block.Header.BlockHeight, blockInfo.Index)
	//save block meta index to db
	metaIndexKey := constructBlockMetaIndexKey(dbType, block.Header.BlockHeight)
	metaIndexInfo := su.ConstructDBIndexInfo(blockInfo.Index, blockInfo.MetaIndex.Offset,
		blockInfo.MetaIndex.ByteLen)
	batch.Put(metaIndexKey, metaIndexInfo)

	// 4. hash-> height
	hashKey := constructBlockHashKey(b.dbHandle.GetDbType(), block.Header.BlockHash)
	batch.Put(hashKey, encodeBlockNum(block.Header.BlockHeight))

	// 4. txid -> tx,  txid -> blockHeight
	// 5. Concurrency batch Put
	//并发更新cache,ConcurrencyMap,提升并发更新能力
	//groups := len(blockInfo.SerializedContractEvents)
	wg := &sync.WaitGroup{}
	wg.Add(len(blockInfo.SerializedTxs))

	for index, txBytes := range blockInfo.SerializedTxs {
		go func(index int, txBytes []byte, batch protocol.StoreBatcher, wg *sync.WaitGroup) {
			defer wg.Done()
			tx := blockInfo.Block.Txs[index]

			// if block rfile db disable, save tx data to db
			txFileIndex := blockInfo.TxsIndex[index]

			// 把tx的地址写入数据库
			blockTxIdKey := constructBlockTxIDKey(tx.Payload.TxId)
			txBlockInf := constructTxIDBlockInfo(block.Header.BlockHeight, block.Header.BlockHash, uint32(index),
				block.Header.BlockTimestamp, blockInfo.Index, txFileIndex)

			batch.Put(blockTxIdKey, txBlockInf)
			//b.logger.Debugf("put tx[%s] rfile index:%v", tx.Payload.TxId, txFileIndex)
		}(index, txBytes, batch, wg)
	}

	wg.Wait()

	// last configBlock height
	if utils.IsConfBlock(block) || block.Header.BlockHeight == 0 {
		batch.Put([]byte(lastConfigBlockNumKey), encodeBlockNum(block.Header.BlockHeight))
		b.logger.Infof("chain[%s]: commit config blockInfo[%d]", block.Header.ChainId, block.Header.BlockHeight)
	}

	// 6. 增加cache,注意这个batch放到cache中了，正在使用，不能放到batchPool中
	b.cache.AddBlock(block.Header.BlockHeight, batch)

	b.logger.Debugf("chain[%s]: commit cache block[%d] blockfiledb, batch[%d], time used: %d",
		block.Header.ChainId, block.Header.BlockHeight, batch.Len(),
		time.Since(start).Milliseconds())
	return nil
}

// CommitDB 提交数据到kvdb
//  @Description:
//  @receiver b
//  @param blockInfo
//  @return error
func (b *BlockFileDB) CommitDB(blockInfo *serialization.BlockWithSerializedInfo) error {
	//1. 从缓存中取 batch
	start := time.Now()
	block := blockInfo.Block
	cacheBatch, err := b.cache.GetBatch(block.Header.BlockHeight)
	if err != nil {
		b.logger.Errorf("chain[%s]: commit blockInfo[%d] delete cacheBatch error",
			block.Header.ChainId, block.Header.BlockHeight)
		panic(err)
	}

	batchDur := time.Since(start)
	//2. write blockDb
	err = b.writeBatch(block.Header.BlockHeight, cacheBatch)
	if err != nil {
		return err
	}

	//3. Delete block from Cache, put batch to batchPool
	//把cacheBatch 从Cache 中删除
	b.cache.DelBlock(block.Header.BlockHeight)

	//再把cacheBatch放回到batchPool 中 (注意这两步前后不能反了)
	b.batchPool.Put(cacheBatch)
	writeDur := time.Since(start)

	b.logger.Debugf("chain[%s]: commit block[%d] kv blockfiledb, time used (batch[%d]:%d, "+
		"write:%d, total:%d)", block.Header.ChainId, block.Header.BlockHeight, cacheBatch.Len(),
		batchDur.Milliseconds(), (writeDur - batchDur).Milliseconds(), time.Since(start).Milliseconds())
	return nil
}

// GetArchivedPivot return archived pivot
//  @Description:
//  @receiver b
//  @return uint64
//  @return error
func (b *BlockFileDB) GetArchivedPivot() (uint64, error) {
	heightBytes, err := b.dbHandle.Get([]byte(archivedPivotKey))
	if err != nil {
		return 0, err
	}

	// heightBytes can be nil while db do not has archive pivot, we use pivot 1 as default
	dbHeight := uint64(0)
	if heightBytes != nil {
		dbHeight = decodeBlockNumKey(b.dbHandle.GetDbType(), heightBytes)
	}

	if dbHeight != b.archivedPivot {
		b.logger.Warnf("DB archivedPivot:[%d] is not match using archivedPivot:[%d], use write DB overwrite it!")
		b.archivedPivot = dbHeight
	}

	return b.archivedPivot, nil
}

// ShrinkBlocks remove ranged txid--SerializedTx from kvdb
//  @Description:
//  @receiver b
//  @param startHeight
//  @param endHeight
//  @return map[uint64][]string
//  @return error
func (b *BlockFileDB) ShrinkBlocks(startHeight uint64, endHeight uint64) (map[uint64][]string, error) {
	panic("block filedb not implement shrink")
}

// RestoreBlocks restore block data from outside to kvdb: txid--SerializedTx
//  @Description:
//  @receiver b
//  @param blockInfos
//  @return error
func (b *BlockFileDB) RestoreBlocks(blockInfos []*serialization.BlockWithSerializedInfo) error {
	panic("block filedb not implement restore blocks")
}

// BlockExists returns true if the block hash exist, or returns false if none exists.
//  @Description:
//  @receiver b
//  @param blockHash
//  @return bool
//  @return error
func (b *BlockFileDB) BlockExists(blockHash []byte) (bool, error) {
	hashKey := constructBlockHashKey(b.dbHandle.GetDbType(), blockHash)
	return b.has(hashKey)
}

// GetBlockByHash returns a block given it's hash, or returns nil if none exists.
//  @Description:
//  @receiver b
//  @param blockHash
//  @return *commonPb.Block
//  @return error
func (b *BlockFileDB) GetBlockByHash(blockHash []byte) (*commonPb.Block, error) {
	hashKey := constructBlockHashKey(b.dbHandle.GetDbType(), blockHash)
	heightBytes, err := b.get(hashKey)
	if err != nil {
		return nil, err
	}
	height := decodeBlockNum(heightBytes)
	return b.GetBlock(height)
}

// GetHeightByHash returns a block height given it's hash, or returns nil if none exists.
//  @Description:
//  @receiver b
//  @param blockHash
//  @return uint64
//  @return error
func (b *BlockFileDB) GetHeightByHash(blockHash []byte) (uint64, error) {
	hashKey := constructBlockHashKey(b.dbHandle.GetDbType(), blockHash)
	heightBytes, err := b.get(hashKey)
	if err != nil {
		return 0, err
	}

	if heightBytes == nil {
		b.logger.Warnf("get blockHash: %s empty", string(blockHash))
		return 0, errValueNotFound
	}

	return decodeBlockNum(heightBytes), nil
}

// GetBlockHeaderByHeight returns a block header by given it's height, or returns nil if none exists.
//  @Description:
//  @receiver b
//  @param height
//  @return *commonPb.BlockHeader
//  @return error
func (b *BlockFileDB) GetBlockHeaderByHeight(height uint64) (*commonPb.BlockHeader, error) {
	storeInfo, err := b.GetBlockMetaIndex(height)
	if err != nil {
		return nil, err
	}
	if storeInfo == nil {
		return nil, nil
	}
	vBytes, err := b.fileStore.ReadFileSection(storeInfo)
	if err != nil {
		return nil, err
	}
	var blockStoreInfo storePb.SerializedBlock
	err = proto.Unmarshal(vBytes, &blockStoreInfo)
	if err != nil {
		return nil, err
	}

	return blockStoreInfo.Header, nil
}

// GetBlock returns a block given it's block height, or returns nil if none exists.
//  @Description:
//  @receiver b
//  @param height
//  @return *commonPb.Block
//  @return error
func (b *BlockFileDB) GetBlock(height uint64) (*commonPb.Block, error) {

	// if block rfile db is enable and db is kvdb, we will get data from block rfile db
	index, err := b.GetBlockIndex(height)
	if err != nil {
		return nil, err
	}
	if index == nil {
		return nil, nil
	}
	data, err := b.fileStore.ReadFileSection(index)
	if err != nil {
		return nil, err
	}
	brw, err := serialization.DeserializeBlock(data)
	if err != nil {
		return nil, err
	}

	return brw.Block, nil
}

// GetLastBlock returns the last block.
//  @Description:
//  @receiver b
//  @return *commonPb.Block
//  @return error
func (b *BlockFileDB) GetLastBlock() (*commonPb.Block, error) {
	num, err := b.GetLastSavepoint()
	if err != nil {
		return nil, err
	}
	return b.GetBlock(num)
}

// GetLastConfigBlock returns the last config block.
//  @Description:
//  @receiver b
//  @return *commonPb.Block
//  @return error
func (b *BlockFileDB) GetLastConfigBlock() (*commonPb.Block, error) {
	height, err := b.GetLastConfigBlockHeight()
	if err != nil {
		return nil, err
	}
	b.logger.Debugf("configBlock height:%v", height)
	return b.GetBlock(height)
}

// GetLastConfigBlockHeight returns the last config block height.
//  @Description:
//  @receiver b
//  @return uint64
//  @return error
func (b *BlockFileDB) GetLastConfigBlockHeight() (uint64, error) {
	heightBytes, err := b.get([]byte(lastConfigBlockNumKey))
	if err != nil {
		return math.MaxUint64, err
	}
	b.logger.Debugf("configBlock height:%v", heightBytes)
	return decodeBlockNum(heightBytes), nil
}

// GetFilteredBlock returns a filtered block given it's block height, or return nil if none exists.
//  @Description:
//  @receiver b
//  @param height
//  @return *storePb.SerializedBlock
//  @return error
func (b *BlockFileDB) GetFilteredBlock(height uint64) (*storePb.SerializedBlock, error) {
	heightKey := constructBlockNumKey(b.dbHandle.GetDbType(), uint64(height))
	bytes, err := b.get(heightKey)
	if err != nil {
		return nil, err
	} else if bytes == nil {
		return nil, nil
	}
	var blockStoreInfo storePb.SerializedBlock
	err = proto.Unmarshal(bytes, &blockStoreInfo)
	if err != nil {
		return nil, err
	}
	return &blockStoreInfo, nil
}

// GetLastSavepoint returns the last block height
//  @Description:
//  @receiver b
//  @return uint64
//  @return error
func (b *BlockFileDB) GetLastSavepoint() (uint64, error) {
	bytes, err := b.get([]byte(lastBlockNumKeyStr))
	if err != nil {
		return 0, err
	} else if bytes == nil {
		return 0, nil
	}

	return decodeBlockNum(bytes), nil
}

// GetBlockByTx returns a block which contains a tx.
//  @Description:
//  @receiver b
//  @param txId
//  @return *commonPb.Block
//  @return error
func (b *BlockFileDB) GetBlockByTx(txId string) (*commonPb.Block, error) {
	txInfo, err := b.getTxInfoOnly(txId)
	if err != nil {
		return nil, err
	}
	if txInfo == nil {
		return nil, nil
	}
	return b.GetBlock(txInfo.BlockHeight)
}

// GetTxHeight retrieves a transaction height by txid, or returns nil if none exists.
//  @Description:
//  @receiver b
//  @param txId
//  @return uint64
//  @return error
func (b *BlockFileDB) GetTxHeight(txId string) (uint64, error) {
	blockTxIdKey := constructBlockTxIDKey(txId)
	txIdBlockInfoBytes, err := b.get(blockTxIdKey)
	if err != nil {
		return 0, err
	}

	if txIdBlockInfoBytes == nil {
		return 0, errValueNotFound
	}
	height, _, _, _, _, err := parseTxIdBlockInfo(txIdBlockInfoBytes)
	//if err != nil {
	//	b.logger.Infof("chain[%s]: put block[%d] hash[%x] (txs:%d bytes:%d), ",
	//		block.Header.ChainId, block.Header.BlockHeight, block.Header.BlockHash, len(block.Txs), len(blockBytes))
	//}
	return height, err
}

// GetTx retrieves a transaction by txid, or returns nil if none exists.
//  @Description:
//  @receiver b
//  @param txId
//  @return *commonPb.Transaction
//  @return error
func (b *BlockFileDB) GetTx(txId string) (*commonPb.Transaction, error) {

	// get tx from block rfile db
	index, err := b.GetTxIndex(txId)
	if err != nil {
		return nil, err
	}
	if index == nil {
		return nil, nil
	}
	data, err := b.fileStore.ReadFileSection(index)
	if err != nil {
		return nil, err
	}

	tx := &commonPb.Transaction{}
	if err = proto.Unmarshal(data, tx); err != nil {
		return nil, err
	}
	return tx, nil
}

// GetTxWithBlockInfo add next time
//  @Description:
//  @receiver b
//  @param txId
//  @return *storePb.TransactionStoreInfo
//  @return error
func (b *BlockFileDB) GetTxWithBlockInfo(txId string) (*storePb.TransactionStoreInfo, error) {
	txInfo, err := b.getTxInfoOnly(txId)
	if err != nil {
		return nil, err
	}
	if txInfo == nil { //查不到对应的Tx，返回nil,nil
		return nil, nil
	}
	tx, err := b.GetTx(txId)
	if err != nil {
		return nil, err
	}
	txInfo.Transaction = tx
	return txInfo, nil
}

//  getTxInfoOnly 获得除Tx之外的其他TxInfo信息
//  @Description:
//  @receiver b
//  @param txId
//  @return *storePb.TransactionStoreInfo
//  @return error
func (b *BlockFileDB) getTxInfoOnly(txId string) (*storePb.TransactionStoreInfo, error) {
	txIDBlockInfoBytes, err := b.get(constructBlockTxIDKey(txId))
	if err != nil {
		return nil, err
	}
	if txIDBlockInfoBytes == nil {
		return nil, nil
	}
	txi := &storePb.TransactionStoreInfo{}
	err = txi.Unmarshal(txIDBlockInfoBytes)
	return txi, err
}

// GetTxInfoOnly 获得除Tx之外的其他TxInfo信息
//  @Description:
//  @receiver b
//  @param txId
//  @return *storePb.TransactionStoreInfo
//  @return error
func (b *BlockFileDB) GetTxInfoOnly(txId string) (*storePb.TransactionStoreInfo, error) {
	return b.getTxInfoOnly(txId)
}

// TxExists returns true if the tx exist, or returns false if none exists.
//  @Description:
//  @receiver b
//  @param txId
//  @return bool
//  @return error
func (b *BlockFileDB) TxExists(txId string) (bool, error) {
	txHashKey := constructBlockTxIDKey(txId)
	exist, err := b.has(txHashKey)
	if err != nil {
		return false, err
	}
	return exist, nil
}

// TxArchived returns true if the tx archived, or returns false.
//  @Description:
//  @receiver b
//  @param txId
//  @return bool
//  @return error
func (b *BlockFileDB) TxArchived(txId string) (bool, error) {
	txIdBlockInfoBytes, err := b.dbHandle.Get(constructBlockTxIDKey(txId))
	if len(txIdBlockInfoBytes) == 0 {
		b.logger.Infof("get value []byte ,TxArchived txid[%s] txIdBlockInfoBytes[%s]",
			txId, txIdBlockInfoBytes)
	}

	if err != nil {
		return false, err
	}

	if txIdBlockInfoBytes == nil {
		return false, errValueNotFound
	}

	archivedPivot, err := b.GetArchivedPivot()
	if err != nil {
		return false, err
	}
	height, _, _, _, _, err := parseTxIdBlockInfo(txIdBlockInfoBytes)
	if err != nil {
		return false, err
	}
	if height <= archivedPivot {
		return true, nil
	}

	return false, nil
}

// GetTxConfirmedTime returns the confirmed time of a given tx
//  @Description:
//  @receiver b
//  @param txId
//  @return int64
//  @return error
func (b *BlockFileDB) GetTxConfirmedTime(txId string) (int64, error) {
	txInfo, err := b.getTxInfoOnly(txId)
	if err != nil {
		return 0, err
	}
	if txInfo == nil {
		return 0, nil
	}
	//如果是新版本，有BlockTimestamp。直接返回
	if txInfo.BlockTimestamp > 0 {
		return txInfo.BlockTimestamp, nil
	}
	//从TxInfo拿不到Timestamp，那么就从Block去拿
	block, err := b.GetBlockMeta(txInfo.BlockHeight)
	if err != nil {
		return 0, err
	}
	return block.Header.BlockTimestamp, nil
}

// GetBlockIndex returns the offset of the block in the rfile
//  @Description:
//  @receiver b
//  @param height
//  @return *storePb.StoreInfo
//  @return error
func (b *BlockFileDB) GetBlockIndex(height uint64) (*storePb.StoreInfo, error) {
	blockIndexByte, err := b.get(constructBlockIndexKey(b.dbHandle.GetDbType(), height))
	if err != nil {
		return nil, err
	}
	if blockIndexByte == nil {
		return nil, nil
	}
	return su.DecodeValueToIndex(blockIndexByte)
}

// GetBlockMeta add next time
//  @Description:
//  @receiver b
//  @param height
//  @return *storePb.SerializedBlock
//  @return error
func (b *BlockFileDB) GetBlockMeta(height uint64) (*storePb.SerializedBlock, error) {
	si, err := b.GetBlockMetaIndex(height)
	if err != nil {
		return nil, err
	}
	if si == nil {
		return nil, nil
	}
	data, err := b.fileStore.ReadFileSection(si)
	if err != nil {
		return nil, err
	}
	bm := &storePb.SerializedBlock{}
	err = bm.Unmarshal(data)
	if err != nil {
		return nil, err
	}
	return bm, nil
}

// GetBlockMetaIndex returns the offset of the block in the rfile
//  @Description:
//  @receiver b
//  @param height
//  @return *storePb.StoreInfo
//  @return error
func (b *BlockFileDB) GetBlockMetaIndex(height uint64) (*storePb.StoreInfo, error) {
	index, err := b.get(constructBlockMetaIndexKey(b.dbHandle.GetDbType(), height))
	if err != nil {
		return nil, err
	}
	if index == nil {
		return nil, nil
	}
	return su.DecodeValueToIndex(index)
}

// GetTxIndex returns the offset of the transaction in the rfile
//  @Description:
//  @receiver b
//  @param txId
//  @return *storePb.StoreInfo
//  @return error
func (b *BlockFileDB) GetTxIndex(txId string) (*storePb.StoreInfo, error) {
	txIDBlockInfoBytes, err := b.get(constructBlockTxIDKey(txId))
	if err != nil {
		return nil, err
	}
	if len(txIDBlockInfoBytes) == 0 {
		return nil, nil //not found
	}
	_, _, _, _, txIndex, err := parseTxIdBlockInfo(txIDBlockInfoBytes)
	b.logger.Debugf("read tx[%s] rfile index get:%v", txId, txIndex)
	return txIndex, err
}

// Close is used to close database
//  @Description:
//  @receiver b
func (b *BlockFileDB) Close() {
	//获得所有信号量，表示没有任何写操作了
	//b.logger.Infof("wait semaphore[%d]", b.worker)
	//err := b.workersSemaphore.Acquire(context.Background(), b.worker)
	//if err != nil {
	//	b.logger.Errorf("semaphore Acquire error:%s", err)
	//}
	b.logger.Info("close block rfile db")
	b.dbHandle.Close()
	b.cache.Clear()
}

//  writeBatch add next time
//  @Description:
//  @receiver b
//  @param blockHeight
//  @param batch
//  @return error
func (b *BlockFileDB) writeBatch(blockHeight uint64, batch protocol.StoreBatcher) error {
	//start := time.Now()
	//batches := batch.SplitBatch(102400)
	//batchDur := time.Since(start)
	//
	//wg := &sync.WaitGroup{}
	//wg.Add(len(batches))
	//for i := 0; i < len(batches); i++ {
	//	go func(index int) {
	//		defer wg.Done()
	//		if err := b.dbHandle.WriteBatch(batches[index], false); err != nil {
	//			panic(fmt.Sprintf("Error writing block rfile db: %s", err))
	//		}
	//	}(i)
	//}
	//wg.Wait()
	//writeDur := time.Since(start)
	//
	//b.logger.Infof("write block rfile db, block[%d], time used: (batchSplit[%d]:%d, "+
	//	"write:%d, total:%d)", blockHeight, len(batches), batchDur.Milliseconds(),
	//	(writeDur - batchDur).Milliseconds(), time.Since(start).Milliseconds())
	//return nil

	startWriteBatchTime := utils.CurrentTimeMillisSeconds()
	err := b.dbHandle.WriteBatch(batch, false)
	endWriteBatchTime := utils.CurrentTimeMillisSeconds()
	b.logger.Debugf("write block db, block[%d], current batch cnt: %d, time used: %d",
		blockHeight, batch.Len(), endWriteBatchTime-startWriteBatchTime)
	return err
}

//  get first from cache;if not found,from db
//  @Description:
//  @receiver b
//  @param key
//  @return []byte
//  @return error
func (b *BlockFileDB) get(key []byte) ([]byte, error) {
	//get from cache
	value, exist := b.cache.Get(string(key))
	if exist {
		b.logger.Debugf("get content: [%x] by [%d] in cache", value, key)
		if len(value) == 0 {
			b.logger.Debugf("get value []byte is empty in cache, key[%s], value: %s, ", key, value)
		}
		return value, nil
	}
	//如果从db 中未找到，会返回 (val,err) 为 (nil,nil)
	//由调用get的上层函数再做判断
	//因为存储系统，对一个key/value 做删除操作，是 直接把 value改为 nil,不做物理删除
	//get from database
	val, err := b.dbHandle.Get(key)
	b.logger.Debugf("get content: [%x] by [%d] in database", val, key)
	if err != nil {
		if len(val) == 0 {
			b.logger.Debugf("get value []byte is empty in database, key[%s], value: %s, ", key, value)
		}
		//
	}
	if err == nil {
		if len(val) == 0 {
			b.logger.Debugf("get value []byte is empty in database,err == nil, key[%s], value: %s, ", key, value)
		}
	}
	return val, err
}

//  has first from cache;if not found,from db
//  @Description:
//  @receiver b
//  @param key
//  @return bool
//  @return error
func (b *BlockFileDB) has(key []byte) (bool, error) {
	//check has from cache
	isDelete, exist := b.cache.Has(string(key))
	if exist {
		return !isDelete, nil
	}
	return b.dbHandle.Has(key)
}

//  constructBlockNumKey
// format n{blockNum}
//  @Description:
//  @param dbType
//  @param blockNum
//  @return []byte
func constructBlockNumKey(dbType string, blockNum uint64) []byte {
	var blkNumBytes []byte
	if dbType == DbType_Mysql {
		blkNumBytes = []byte(fmt.Sprint(blockNum))
	} else {
		blkNumBytes = encodeBlockNum(blockNum)
	}
	return append([]byte{blockNumIdxKeyPrefix}, blkNumBytes...)
}

//  constructBlockMetaIndexKey
// format: im{blockNum}
//  @Description:
//  @param dbType
//  @param blockNum
//  @return []byte
func constructBlockMetaIndexKey(dbType string, blockNum uint64) []byte {
	var blkNumBytes []byte
	if dbType == DbType_Mysql {
		blkNumBytes = []byte(fmt.Sprint(blockNum))
	} else {
		blkNumBytes = encodeBlockNum(blockNum)
	}
	key := append([]byte{}, blockMetaIndexKeyPrefix...)
	return append(key, blkNumBytes...)
}

//  constructBlockIndexKey
// format: ib{blockNum}
//  @Description:
//  @param dbType
//  @param blockNum
//  @return []byte
func constructBlockIndexKey(dbType string, blockNum uint64) []byte {
	var blkNumBytes []byte
	if dbType == DbType_Mysql {
		blkNumBytes = []byte(fmt.Sprint(blockNum))
	} else {
		blkNumBytes = encodeBlockNum(blockNum)
	}
	key := append([]byte{}, blockIndexKeyPrefix...)
	return append(key, blkNumBytes...)
}

//  constructBlockHashKey
// format: h{blockHash}
//  @Description:
//  @param dbType
//  @param blockHash
//  @return []byte
func constructBlockHashKey(dbType string, blockHash []byte) []byte {
	if dbType == DbType_Mysql {
		bString := hex.EncodeToString(blockHash)
		return append([]byte{blockHashIdxKeyPrefix}, bString...)
	}
	return append([]byte{blockHashIdxKeyPrefix}, blockHash...)
}

//  constructBlockTxIDKey
// format: b{txId}
//  @Description:
//  @param txID
//  @return []byte
func constructBlockTxIDKey(txID string) []byte {
	return append([]byte{blockTxIDIdxKeyPrefix}, bytesconv.StringToBytes(txID)...)
}

//  encodeBlockNum varint encode
//  @Description:
//  @param blockNum
//  @return []byte
func encodeBlockNum(blockNum uint64) []byte {
	return proto.EncodeVarint(blockNum)
}

//  decodeBlockNumKey trim prefix,decode blockNum
//  @Description:
//  @param dbType
//  @param blkNumBytes
//  @return uint64
func decodeBlockNumKey(dbType string, blkNumBytes []byte) uint64 {
	blkNumBytes = blkNumBytes[len([]byte{blockNumIdxKeyPrefix}):]
	if dbType == DbType_Mysql {
		intNum, _ := strconv.Atoi(string(blkNumBytes))
		return uint64(intNum)
	}
	return decodeBlockNum(blkNumBytes)
}

//  decodeBlockNum decode blockNum
//  @Description:
//  @param blockNumBytes
//  @return uint64
func decodeBlockNum(blockNumBytes []byte) uint64 {
	blockNum, _ := proto.DecodeVarint(blockNumBytes)
	return blockNum
}

//  constructTxIDBlockInfo marshal TransactionStoreInfo
//  @Description:
//  @param height
//  @param blockHash
//  @param txIndex
//  @param timestamp
//  @param blockFileIndex
//  @param txFileIndex
//  @return []byte
func constructTxIDBlockInfo(height uint64, blockHash []byte, txIndex uint32, timestamp int64,
	blockFileIndex, txFileIndex *storePb.StoreInfo) []byte {
	//value := fmt.Sprintf("%d,%x,%d", height, blockHash, txIndex)
	//return []byte(value)
	var transactionFileIndex *storePb.StoreInfo
	if txFileIndex != nil {
		transactionFileIndex = &storePb.StoreInfo{
			FileName: blockFileIndex.FileName,
			Offset:   blockFileIndex.Offset + txFileIndex.Offset,
			ByteLen:  txFileIndex.ByteLen,
		}
	}
	txInf := &storePb.TransactionStoreInfo{
		BlockHeight:          height,
		BlockHash:            nil, //for performance, set nil
		TxIndex:              txIndex,
		BlockTimestamp:       timestamp,
		TransactionStoreInfo: transactionFileIndex,
	}
	data, _ := txInf.Marshal()
	return data
}

//  parseTxIdBlockInfo retrieve TransactionStoreInfo
//  @Description:
//  @param value
//  @return height
//  @return blockHash
//  @return txIndex
//  @return timestamp
//  @return txFIleIndex
//  @return err
func parseTxIdBlockInfo(value []byte) (height uint64, blockHash []byte, txIndex uint32, timestamp int64,
	txFIleIndex *storePb.StoreInfo, err error) {
	if len(value) == 0 {
		err = errors.New("input is empty,in parseTxIdBlockInfo")
		return
	}
	//新版本使用TransactionInfo，是因为经过BenchmarkTest速度会快很多
	var txInfo storePb.TransactionStoreInfo
	err = txInfo.Unmarshal(value)
	if err != nil {
		return
	}
	height = txInfo.BlockHeight
	blockHash = txInfo.BlockHash
	txIndex = txInfo.TxIndex
	timestamp = txInfo.BlockTimestamp
	txFIleIndex = txInfo.TransactionStoreInfo
	err = nil
	return
}
