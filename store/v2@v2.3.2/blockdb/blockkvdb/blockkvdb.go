/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0

*/

package blockkvdb

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	bytesconv "chainmaker.org/chainmaker/common/v2/bytehelper"
	commonPb "chainmaker.org/chainmaker/pb-go/v2/common"
	storePb "chainmaker.org/chainmaker/pb-go/v2/store"
	"chainmaker.org/chainmaker/protocol/v2"
	"chainmaker.org/chainmaker/store/v2/archive"
	"chainmaker.org/chainmaker/store/v2/cache"
	"chainmaker.org/chainmaker/store/v2/conf"
	"chainmaker.org/chainmaker/store/v2/serialization"
	"chainmaker.org/chainmaker/store/v2/types"
	"chainmaker.org/chainmaker/utils/v2"
	"github.com/gogo/protobuf/proto"
	"golang.org/x/sync/semaphore"
)

// nolint
var DbType_Mysql = "mysql"

const (
	blockNumIdxKeyPrefix  = 'n'
	blockHashIdxKeyPrefix = 'h'
	txIDIdxKeyPrefix      = 't'
	//txConfirmedTimeKeyPrefix = 'c'
	blockTxIDIdxKeyPrefix = 'b'
	lastBlockNumKeyStr    = "lastBlockNumKey"
	lastConfigBlockNumKey = "lastConfigBlockNumKey"
	archivedPivotKey      = "archivedPivotKey"
)

var (
	errValueNotFound = errors.New("value not found")
	errGetBatchPool  = errors.New("get updatebatch error")
)

// BlockKvDB provider a implementation of `blockdb.BlockDB`
//  @Description:
// This implementation provides a key-value based data model
type BlockKvDB struct {
	dbHandle         protocol.DBHandle
	workersSemaphore *semaphore.Weighted
	worker           int64
	cache            *cache.StoreCacheMgr
	archivedPivot    uint64
	logger           protocol.Logger
	sync.Mutex
	batchPool   sync.Pool
	storeConfig *conf.StorageConfig
}

// NewBlockKvDB 创建blockKvDB
//  @Description:
//  @param chainId
//  @param dbHandle
//  @param logger
//  @param storeConfig
//  @return *BlockKvDB
func NewBlockKvDB(chainId string, dbHandle protocol.DBHandle, logger protocol.Logger,
	storeConfig *conf.StorageConfig) *BlockKvDB {
	nWorkers := int64(runtime.NumCPU())
	b := &BlockKvDB{
		dbHandle:         dbHandle,
		worker:           nWorkers,
		workersSemaphore: semaphore.NewWeighted(nWorkers),
		cache:            cache.NewStoreCacheMgr(chainId, 10, logger),
		archivedPivot:    0,
		logger:           logger,
		batchPool:        sync.Pool{},
		storeConfig:      storeConfig,
	}
	b.batchPool.New = func() interface{} {
		return types.NewUpdateBatch()
	}

	return b
}

// InitGenesis 初始化创世区块
//  @Description:
//  @receiver b
//  @param genesisBlock
//  @return error
func (b *BlockKvDB) InitGenesis(genesisBlock *serialization.BlockWithSerializedInfo) error {
	//update Cache
	err := b.CommitBlock(genesisBlock, true)
	if err != nil {
		return err
	}
	//update BlockKvDB
	return b.CommitBlock(genesisBlock, false)
}

// CommitBlock commits the block and the corresponding rwsets in an atomic operation
//  @Description:
//  @receiver b
//  @param blockInfo
//  @param isCache
//  @return error
func (b *BlockKvDB) CommitBlock(blockInfo *serialization.BlockWithSerializedInfo, isCache bool) error {
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
func (b *BlockKvDB) CommitCache(blockInfo *serialization.BlockWithSerializedInfo) error {
	b.logger.Debugf("[blockdb]start CommitCache currtime[%d]", utils.CurrentTimeMillisSeconds())
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

	// 1. last blockInfo height
	block := blockInfo.Block
	lastBlockNumBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(lastBlockNumBytes, block.Header.BlockHeight)
	batch.Put([]byte(lastBlockNumKeyStr), lastBlockNumBytes)

	// 2. height-> blockInfo
	heightKey := constructBlockNumKey(b.dbHandle.GetDbType(), block.Header.BlockHeight)

	batch.Put(heightKey, blockInfo.SerializedMeta)

	// 4. hash-> height
	hashKey := constructBlockHashKey(b.dbHandle.GetDbType(), block.Header.BlockHash)
	batch.Put(hashKey, heightKey)

	// 4. txid -> tx,  txid -> blockHeight
	txConfirmedTime := make([]byte, 8)
	binary.BigEndian.PutUint64(txConfirmedTime, uint64(block.Header.BlockTimestamp))
	b.logger.Debugf("[blockdb]1 CommitCache currtime[%d]", utils.CurrentTimeMillisSeconds())
	// 5. Concurrency batch Put
	//并发更新cache,ConcurrencyMap,提升并发更新能力
	wg := &sync.WaitGroup{}
	wg.Add(len(blockInfo.SerializedTxs))
	for index, txBytes := range blockInfo.SerializedTxs {
		go func(index int, txBytes []byte, batch protocol.StoreBatcher, wg *sync.WaitGroup) {
			defer wg.Done()
			tx := blockInfo.Block.Txs[index]

			// if block file db disable, save tx data to db
			txIdKey := constructTxIDKey(tx.Payload.TxId)
			batch.Put(txIdKey, txBytes)

			// 把tx的地址写入数据库
			blockTxIdKey := constructBlockTxIDKey(tx.Payload.TxId)
			//txBlockInf := constructTxIDBlockInfo(block.Header.BlockHeight, block.Header.BlockHash, uint32(index))
			txBlockInf := constructTxIDBlockInfo(block.Header.BlockHeight, block.Header.BlockHash, uint32(index),
				block.Header.BlockTimestamp)

			batch.Put(blockTxIdKey, txBlockInf)
			//b.logger.Debugf("put tx[%s]", tx.Payload.TxId)
		}(index, txBytes, batch, wg)
	}
	b.logger.Debugf("[blockdb]2 CommitCache currtime[%d]", utils.CurrentTimeMillisSeconds())
	wg.Wait()
	b.logger.Debugf("[blockdb]3 CommitCache currtime[%d]", utils.CurrentTimeMillisSeconds())
	// last configBlock height
	if utils.IsConfBlock(block) {
		batch.Put([]byte(lastConfigBlockNumKey), heightKey)
		b.logger.Infof("chain[%s]: commit config blockInfo[%d]", block.Header.ChainId, block.Header.BlockHeight)
	}

	// 6. 增加cache,注意这个batch放到cache中了，正在使用，不能放到batchPool中
	b.cache.AddBlock(block.Header.BlockHeight, batch)

	b.logger.Debugf("chain[%s]: commit cache block[%d] blockdb, batch[%d], time used: %d",
		block.Header.ChainId, block.Header.BlockHeight, batch.Len(),
		time.Since(start).Milliseconds())
	return nil
}

// CommitDB 提交数据到kvdb
//  @Description:
//  @receiver b
//  @param blockInfo
//  @return error
func (b *BlockKvDB) CommitDB(blockInfo *serialization.BlockWithSerializedInfo) error {
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

	b.logger.Debugf("chain[%s]: commit block[%d] kv blockdb, time used (batch[%d]:%d, "+
		"write:%d, total:%d)", block.Header.ChainId, block.Header.BlockHeight, cacheBatch.Len(),
		batchDur.Milliseconds(), (writeDur - batchDur).Milliseconds(), time.Since(start).Milliseconds())
	return nil
}

// GetArchivedPivot return archived pivot
//  @Description:
//  @receiver b
//  @return uint64
//  @return error
func (b *BlockKvDB) GetArchivedPivot() (uint64, error) {
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
func (b *BlockKvDB) ShrinkBlocks(startHeight uint64, endHeight uint64) (map[uint64][]string, error) {
	var (
		block *commonPb.Block
		err   error
	)

	if block, err = b.getBlockByHeightKey(constructBlockNumKey(b.dbHandle.GetDbType(), endHeight), true); err != nil {
		return nil, err
	}

	if utils.IsConfBlock(block) {
		return nil, archive.ErrConfigBlockArchive
	}

	txIdsMap := make(map[uint64][]string)
	startTime := utils.CurrentTimeMillisSeconds()
	for height := startHeight; height <= endHeight; height++ {
		heightKey := constructBlockNumKey(b.dbHandle.GetDbType(), height)
		blk, err1 := b.getBlockByHeightKey(heightKey, true)
		if err1 != nil {
			return nil, err1
		}

		if utils.IsConfBlock(blk) {
			b.logger.Infof("skip shrink conf block: [%d]", block.Header.BlockHeight)
			continue
		}

		batch := types.NewUpdateBatch()
		txIds := make([]string, 0, len(blk.Txs))
		for _, tx := range blk.Txs {
			// delete tx data
			batch.Delete(constructTxIDKey(tx.Payload.TxId))
			txIds = append(txIds, tx.Payload.TxId)
		}
		txIdsMap[height] = txIds
		//set archivedPivotKey to db
		batch.Put([]byte(archivedPivotKey), constructBlockNumKey(b.dbHandle.GetDbType(), height))

		//先删除cache，再更新db,删除key/value,就是把 value 重置成 nil
		b.cache.DelBlock(height)

		if err = b.dbHandle.WriteBatch(batch, false); err != nil {
			return nil, err
		}

		b.archivedPivot = height
	}

	go b.compactRange()

	usedTime := utils.CurrentTimeMillisSeconds() - startTime
	b.logger.Infof("shrink block from [%d] to [%d] time used: %d",
		startHeight, endHeight, usedTime)
	return txIdsMap, nil
}

// RestoreBlocks restore block data from outside to kvdb: txid--SerializedTx
//  @Description:
//  @receiver b
//  @param blockInfos
//  @return error
func (b *BlockKvDB) RestoreBlocks(blockInfos []*serialization.BlockWithSerializedInfo) error {
	startTime := utils.CurrentTimeMillisSeconds()
	archivePivot := uint64(0)
	for i := len(blockInfos) - 1; i >= 0; i-- {
		blockInfo := blockInfos[i]

		//check whether block can be archived
		if utils.IsConfBlock(blockInfo.Block) {
			b.logger.Infof("skip store conf block: [%d]", blockInfo.Block.Header.BlockHeight)
			continue
		}

		//check block hash
		sBlock, err := b.GetFilteredBlock(blockInfo.Block.Header.BlockHeight)
		if err != nil {
			return err
		}

		if !bytes.Equal(blockInfo.Block.Header.BlockHash, sBlock.Header.BlockHash) {
			return archive.ErrInvalidateRestoreBlocks
		}

		batch := types.NewUpdateBatch()
		//verify imported block txs
		for index, stx := range blockInfo.SerializedTxs {
			// put tx data
			batch.Put(constructTxIDKey(blockInfo.Block.Txs[index].Payload.TxId), stx)
		}

		archivePivot, err = b.getNextArchivePivot(blockInfo.Block)
		if err != nil {
			return err
		}

		batch.Put([]byte(archivedPivotKey), constructBlockNumKey(b.dbHandle.GetDbType(), archivePivot))
		err = b.dbHandle.WriteBatch(batch, false)
		if err != nil {
			return err
		}
		b.archivedPivot = archivePivot
	}

	go b.compactRange()

	usedTime := utils.CurrentTimeMillisSeconds() - startTime
	b.logger.Infof("shrink block from [%d] to [%d] time used: %d",
		blockInfos[len(blockInfos)-1].Block.Header.BlockHeight, blockInfos[0].Block.Header.BlockHeight, usedTime)
	return nil
}

// BlockExists returns true if the block hash exist, or returns false if none exists.
//  @Description:
//  @receiver b
//  @param blockHash
//  @return bool
//  @return error
func (b *BlockKvDB) BlockExists(blockHash []byte) (bool, error) {
	hashKey := constructBlockHashKey(b.dbHandle.GetDbType(), blockHash)
	return b.has(hashKey)
}

// GetBlockByHash returns a block given it's hash, or returns nil if none exists.
//  @Description:
//  @receiver b
//  @param blockHash
//  @return *commonPb.Block
//  @return error
func (b *BlockKvDB) GetBlockByHash(blockHash []byte) (*commonPb.Block, error) {
	hashKey := constructBlockHashKey(b.dbHandle.GetDbType(), blockHash)
	heightBytes, err := b.get(hashKey)
	if err != nil {
		return nil, err
	}

	return b.getBlockByHeightKey(heightBytes, true)
}

// GetHeightByHash returns a block height given it's hash, or returns nil if none exists.
//  @Description:
//  @receiver b
//  @param blockHash
//  @return uint64
//  @return error
func (b *BlockKvDB) GetHeightByHash(blockHash []byte) (uint64, error) {
	hashKey := constructBlockHashKey(b.dbHandle.GetDbType(), blockHash)
	heightBytes, err := b.get(hashKey)
	if err != nil {
		return 0, err
	}

	if heightBytes == nil {
		return 0, errValueNotFound
	}

	return decodeBlockNumKey(b.dbHandle.GetDbType(), heightBytes), nil
}

// GetBlockHeaderByHeight returns a block header by given it's height, or returns nil if none exists.
//  @Description:
//  @receiver b
//  @param height
//  @return *commonPb.BlockHeader
//  @return error
func (b *BlockKvDB) GetBlockHeaderByHeight(height uint64) (*commonPb.BlockHeader, error) {
	vBytes, err := b.get(constructBlockNumKey(b.dbHandle.GetDbType(), uint64(height)))
	if err != nil {
		return nil, err
	}

	if vBytes == nil {
		return nil, errValueNotFound
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
func (b *BlockKvDB) GetBlock(height uint64) (*commonPb.Block, error) {
	heightBytes := constructBlockNumKey(b.dbHandle.GetDbType(), height)
	return b.getBlockByHeightKey(heightBytes, true)
}

// GetLastBlock returns the last block.
//  @Description:
//  @receiver b
//  @return *commonPb.Block
//  @return error
func (b *BlockKvDB) GetLastBlock() (*commonPb.Block, error) {
	num, err := b.GetLastSavepoint()
	if err != nil {
		return nil, err
	}

	heightBytes := constructBlockNumKey(b.dbHandle.GetDbType(), num)
	return b.getBlockByHeightKey(heightBytes, true)
}

// GetLastConfigBlock returns the last config block.
//  @Description:
//  @receiver b
//  @return *commonPb.Block
//  @return error
func (b *BlockKvDB) GetLastConfigBlock() (*commonPb.Block, error) {
	heightKey, err := b.get([]byte(lastConfigBlockNumKey))
	if err != nil {
		return nil, err
	}
	b.logger.Debugf("configBlock height:%v", heightKey)
	return b.getBlockByHeightKey(heightKey, true)
}

// GetLastConfigBlockHeight returns the last config block height.
//  @Description:
//  @receiver b
//  @return uint64
//  @return error
func (b *BlockKvDB) GetLastConfigBlockHeight() (uint64, error) {
	heightKey, err := b.get([]byte(lastConfigBlockNumKey))
	if err != nil {
		return math.MaxUint64, err
	}
	b.logger.Debugf("configBlock height:%v", heightKey)
	return decodeBlockNumKey(b.dbHandle.GetDbType(), heightKey), nil
}

// GetFilteredBlock returns a filtered block given it's block height, or return nil if none exists.
//  @Description:
//  @receiver b
//  @param height
//  @return *storePb.SerializedBlock
//  @return error
func (b *BlockKvDB) GetFilteredBlock(height uint64) (*storePb.SerializedBlock, error) {
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
func (b *BlockKvDB) GetLastSavepoint() (uint64, error) {
	bytes, err := b.get([]byte(lastBlockNumKeyStr))
	if err != nil {
		return 0, err
	} else if bytes == nil {
		return 0, nil
	}

	num := binary.BigEndian.Uint64(bytes)
	return num, nil
}

// GetBlockByTx returns a block which contains a tx.
//  @Description:
//  @receiver b
//  @param txId
//  @return *commonPb.Block
//  @return error
func (b *BlockKvDB) GetBlockByTx(txId string) (*commonPb.Block, error) {
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
func (b *BlockKvDB) GetTxHeight(txId string) (uint64, error) {
	blockTxIdKey := constructBlockTxIDKey(txId)
	txIdBlockInfoBytes, err := b.get(blockTxIdKey)
	if err != nil {
		return 0, err
	}

	if txIdBlockInfoBytes == nil {
		return 0, errValueNotFound
	}
	height, _, _, _, err := parseTxIdBlockInfo(txIdBlockInfoBytes)
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
func (b *BlockKvDB) GetTx(txId string) (*commonPb.Transaction, error) {
	txIdKey := constructTxIDKey(txId)
	bytes, err := b.get(txIdKey)
	if err != nil {
		return nil, err
	} else if len(bytes) == 0 {
		isArchived, erra := b.TxArchived(txId)
		if erra == nil && isArchived {
			return nil, archive.ErrArchivedTx
		}

		return nil, nil
	}

	var tx commonPb.Transaction
	err = proto.Unmarshal(bytes, &tx)
	if err != nil {
		return nil, err
	}

	return &tx, nil
}

// GetTxWithBlockInfo 根据txId获得交易
//  @Description:
//  @receiver b
//  @param txId
//  @return *storePb.TransactionStoreInfo
//  @return error
func (b *BlockKvDB) GetTxWithBlockInfo(txId string) (*storePb.TransactionStoreInfo, error) {
	txIdKey := constructTxIDKey(txId)
	vBytes, err := b.get(txIdKey)
	if err != nil {
		return nil, err
	} else if len(vBytes) == 0 {
		isArchived, erra := b.TxArchived(txId)
		if erra == nil && isArchived {
			return nil, archive.ErrArchivedTx
		}
		return nil, nil
	}

	var tx commonPb.Transaction
	err = proto.Unmarshal(vBytes, &tx)
	if err != nil {
		return nil, err
	}
	txInfo, err := b.getTxInfoOnly(txId)
	if err != nil {
		return nil, err
	}
	txInfo.Transaction = &tx
	return txInfo, nil
}

//  getTxInfoOnly 获得除Tx之外的其他TxInfo信息
//  @Description:
//  @receiver b
//  @param txId
//  @return *storePb.TransactionStoreInfo
//  @return error
func (b *BlockKvDB) getTxInfoOnly(txId string) (*storePb.TransactionStoreInfo, error) {
	txIDBlockInfoBytes, err := b.get(constructBlockTxIDKey(txId))
	if err != nil {
		return nil, err
	}
	if txIDBlockInfoBytes == nil {
		return nil, nil
	}
	height, blockHash, txIndex, timestamp, err := parseTxIdBlockInfo(txIDBlockInfoBytes)
	if err != nil {
		return nil, err
	}
	return &storePb.TransactionStoreInfo{
			BlockHeight:    height,
			BlockHash:      blockHash,
			TxIndex:        txIndex,
			BlockTimestamp: timestamp},
		nil
}

// GetTxInfoOnly 获得除Tx之外的其他TxInfo信息
//  @Description:
//  @receiver b
//  @param txId
//  @return *storePb.TransactionStoreInfo
//  @return error
func (b *BlockKvDB) GetTxInfoOnly(txId string) (*storePb.TransactionStoreInfo, error) {
	return b.getTxInfoOnly(txId)
}

// TxExists returns true if the tx exist, or returns false if none exists.
//  @Description:
//  @receiver b
//  @param txId
//  @return bool
//  @return error
func (b *BlockKvDB) TxExists(txId string) (bool, error) {
	txHashKey := constructBlockTxIDKey(txId)
	exist, err := b.has(txHashKey)
	if err != nil {
		b.logger.Errorf("check tx exist by txid:[%s] in blockkvdb error:[%s]", txId, err)
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
func (b *BlockKvDB) TxArchived(txId string) (bool, error) {
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
	height, _, _, _, err := parseTxIdBlockInfo(txIdBlockInfoBytes)
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
func (b *BlockKvDB) GetTxConfirmedTime(txId string) (int64, error) {
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
	heightBytes := constructBlockNumKey(b.dbHandle.GetDbType(), txInfo.BlockHeight)
	block, err := b.getBlockByHeightKey(heightBytes, false)
	if err != nil {
		return 0, err
	}
	return block.Header.BlockTimestamp, nil
}

// Close is used to close database
//  @Description:
//  @receiver b
func (b *BlockKvDB) Close() {
	//获得所有信号量，表示没有任何写操作了
	//b.logger.Infof("wait semaphore[%d]", b.worker)
	//err := b.workersSemaphore.Acquire(context.Background(), b.worker)
	//if err != nil {
	//	b.logger.Errorf("semaphore Acquire error:%s", err)
	//}
	b.logger.Info("close block kv db")
	b.dbHandle.Close()
	b.cache.Clear()
}

//  getBlockByHeightKey 根据区块高度Key获得区块信息和其中的交易信息
//  @Description:
//  @receiver b
//  @param heightKey
//  @param includeTxs
//  @return *commonPb.Block
//  @return error
func (b *BlockKvDB) getBlockByHeightKey(heightKey []byte, includeTxs bool) (*commonPb.Block, error) {
	if heightKey == nil {
		return nil, nil
	}

	vBytes, err := b.get(heightKey)
	if err != nil || vBytes == nil {
		return nil, err
	}

	var blockStoreInfo storePb.SerializedBlock
	err = proto.Unmarshal(vBytes, &blockStoreInfo)
	if err != nil {
		return nil, err
	}

	var block = commonPb.Block{
		Header:         blockStoreInfo.Header,
		Dag:            blockStoreInfo.Dag,
		AdditionalData: blockStoreInfo.AdditionalData,
	}
	if includeTxs {
		//var batchWG sync.WaitGroup
		//batchWG.Add(len(blockStoreInfo.TxIds))
		//errsChan := make(chan error, len(blockStoreInfo.TxIds))
		block.Txs = make([]*commonPb.Transaction, len(blockStoreInfo.TxIds))
		for index, txid := range blockStoreInfo.TxIds {
			//used to limit the num of concurrency goroutine
			//b.workersSemaphore.Acquire(context.Background(), 1)
			//go func(i int, txid string) {
			//	defer b.workersSemaphore.Release(1)
			//	defer batchWG.Done()
			tx, err1 := b.GetTx(txid)
			if err1 != nil {
				if err1 == archive.ErrArchivedTx {
					return nil, archive.ErrArchivedBlock
				}
				//errsChan <- err
				return nil, err1
			}

			block.Txs[index] = tx
			//}(index, txid)
		}
		//batchWG.Wait()
		//if len(errsChan) > 0 {
		//	return nil, <-errsChan
		//}
	}
	b.logger.Debugf("chain[%s]: get block[%d] with transactions[%d]",
		block.Header.ChainId, block.Header.BlockHeight, len(block.Txs))
	return &block, nil
}

//  writeBatch 将batch中的区块，写入db中
//  @Description:
//  @receiver b
//  @param blockHeight
//  @param batch
//  @return error
func (b *BlockKvDB) writeBatch(blockHeight uint64, batch protocol.StoreBatcher) error {
	startWriteBatchTime := utils.CurrentTimeMillisSeconds()
	err := b.dbHandle.WriteBatch(batch, false)
	endWriteBatchTime := utils.CurrentTimeMillisSeconds()
	b.logger.Debugf("write block db, block[%d], current batch cnt: %d, time used: %d",
		blockHeight, batch.Len(), endWriteBatchTime-startWriteBatchTime)
	return err
}

//  get 获得key对应value
//  @Description:
//  @receiver b
//  @param key
//  @return []byte
//  @return error
func (b *BlockKvDB) get(key []byte) ([]byte, error) {
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

//  has 判断 key 是否存在
//  @Description:
//  @receiver b
//  @param key
//  @return bool
//  @return error
func (b *BlockKvDB) has(key []byte) (bool, error) {
	//check has from cache
	isDelete, exist := b.cache.Has(string(key))
	if exist {
		return !isDelete, nil
	}
	return b.dbHandle.Has(key)
}

//  getNextArchivePivot get next archive pivot
//  @Description:
//  @receiver b
//  @param pivotBlock
//  @return uint64
//  @return error
func (b *BlockKvDB) getNextArchivePivot(pivotBlock *commonPb.Block) (uint64, error) {
	curIsConf := true
	archivedPivot := uint64(pivotBlock.Header.BlockHeight)
	for curIsConf {
		//consider restore height 1 and height 0 block
		//1. height 1: this is a config block, archivedPivot should be 0
		//2. height 1: this is not a config block, archivedPivot should be 0
		//3. height 0: archivedPivot should be 0
		if archivedPivot < 2 {
			archivedPivot = 0
			break
		}

		//we should not get block data only if it is config block
		archivedPivot = archivedPivot - 1
		_, errb := b.GetBlock(archivedPivot)
		if errb == archive.ErrArchivedBlock {
			//curIsConf = false
			break
		} else if errb != nil {
			return 0, errb
		}
	}
	return archivedPivot, nil
}

//  constructBlockNumKey construct blockNum key
// format : n{blockNum}
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

//  constructBlockHashKey construct block hash key
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

//  constructTxIDKey construct tx id key
// format : t{txId}
//  @Description:
//  @param txId
//  @return []byte
func constructTxIDKey(txId string) []byte {
	return append([]byte{txIDIdxKeyPrefix}, bytesconv.StringToBytes(txId)...)
}

//func constructTxConfirmedTimeKey(txId string) []byte {
//	return append([]byte{txConfirmedTimeKeyPrefix}, txId...)
//}

//  constructBlockTxIDKey construct block tx id key
// format : b{txID}
//  @Description:
//  @param txID
//  @return []byte
func constructBlockTxIDKey(txID string) []byte {
	//return append([]byte{blockTxIDIdxKeyPrefix}, []byte(txID)...)
	return append([]byte{blockTxIDIdxKeyPrefix}, bytesconv.StringToBytes(txID)...)
}

//  encodeBlockNum varint encode blockNum
//  @Description:
//  @param blockNum
//  @return []byte
func encodeBlockNum(blockNum uint64) []byte {
	return proto.EncodeVarint(blockNum)
}

//  decodeBlockNumKey decode
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

//  decodeBlockNum varint decode
//  @Description:
//  @param blockNumBytes
//  @return uint64
func decodeBlockNum(blockNumBytes []byte) uint64 {
	blockNum, _ := proto.DecodeVarint(blockNumBytes)
	return blockNum
}

//  compactRange compact underly db
//  @Description:
//  @receiver b
func (b *BlockKvDB) compactRange() {
	//trigger level compact
	for i := 1; i <= 1; i++ {
		b.logger.Infof("Do %dst time CompactRange", i)
		if err := b.dbHandle.CompactRange(nil, nil); err != nil {
			b.logger.Warnf("blockdb level compact failed: %v", err)
		}
		//time.Sleep(2 * time.Second)
	}
}

//  constructTxIDBlockInfo marshal TransactionStoreInfo
//  @Description:
//  @param height
//  @param blockHash
//  @param txIndex
//  @param timestamp
//  @return []byte
func constructTxIDBlockInfo(height uint64, blockHash []byte, txIndex uint32, timestamp int64) []byte {
	//value := fmt.Sprintf("%d,%x,%d", height, blockHash, txIndex)
	//return []byte(value)
	txInf := &storePb.TransactionStoreInfo{
		BlockHeight:    height,
		BlockHash:      blockHash,
		TxIndex:        txIndex,
		BlockTimestamp: timestamp,
	}
	data, _ := txInf.Marshal()
	return data
}

//  parseTxIdBlockInfoOld return height,blockHash,txIndex
//  @Description:
//  @param value
//  @return height
//  @return blockHash
//  @return txIndex
//  @return err
func parseTxIdBlockInfoOld(value []byte) (height uint64, blockHash []byte, txIndex uint32, err error) {
	strArray := strings.Split(string(value), ",")
	if len(strArray) != 3 {
		err = fmt.Errorf("invalid input[%s]", value)
		return
	}
	height, err = strconv.ParseUint(strArray[0], 10, 64)
	if err != nil {
		return 0, nil, 0, err
	}
	blockHash, err = hex.DecodeString(strArray[1])
	if err != nil {
		return 0, nil, 0, err
	}
	idx, err := strconv.Atoi(strArray[2])
	txIndex = uint32(idx)
	return
}

//  parseTxIdBlockInfo return height,blockHash,txIndex，timestamp
//  @Description:
//  @param value
//  @return height
//  @return blockHash
//  @return txIndex
//  @return timestamp
//  @return err
func parseTxIdBlockInfo(value []byte) (height uint64, blockHash []byte, txIndex uint32, timestamp int64, err error) {
	if len(value) == 0 {
		err = errors.New("input is empty,in parseTxIdBlockInfo")
		return
	}
	//新版本使用TransactionInfo，是因为经过BenchmarkTest速度会快很多
	var txInfo storePb.TransactionStoreInfo
	err = txInfo.Unmarshal(value)
	if err != nil {
		//兼容老版本存储
		height, blockHash, txIndex, err = parseTxIdBlockInfoOld(value)
		return
	}
	height = txInfo.BlockHeight
	blockHash = txInfo.BlockHash
	txIndex = txInfo.TxIndex
	timestamp = txInfo.BlockTimestamp
	err = nil
	return
}

// GetBlockMetaIndex not implement
//  @Description:
//  @receiver b
//  @param height
//  @return *storePb.StoreInfo
//  @return error
func (b *BlockKvDB) GetBlockMetaIndex(height uint64) (*storePb.StoreInfo, error) {
	//TODO implement me
	panic("implement GetBlockMetaIndex")
}

// GetTxIndex not implement
//  @Description:
//  @receiver b
//  @param txId
//  @return *storePb.StoreInfo
//  @return error
func (b *BlockKvDB) GetTxIndex(txId string) (*storePb.StoreInfo, error) {
	//TODO implement me
	panic("implement GetTxIndex")
}

// GetBlockIndex not implement
//  @Description:
//  @receiver b
//  @param height
//  @return *storePb.StoreInfo
//  @return error
func (b *BlockKvDB) GetBlockIndex(height uint64) (*storePb.StoreInfo, error) {
	//TODO implement me
	panic("implement GetBlockIndex")
}
