/*
 * Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
 *
 * SPDX-License-Identifier: Apache-2.0
 *
 */

package store

import (
	"encoding/json"
	"errors"
	"fmt"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	commonErr "chainmaker.org/chainmaker/common/v2/errors"
	"chainmaker.org/chainmaker/common/v2/wal"
	"chainmaker.org/chainmaker/pb-go/v2/accesscontrol"
	commonPb "chainmaker.org/chainmaker/pb-go/v2/common"
	configPb "chainmaker.org/chainmaker/pb-go/v2/config"
	storePb "chainmaker.org/chainmaker/pb-go/v2/store"
	"chainmaker.org/chainmaker/protocol/v2"
	"chainmaker.org/chainmaker/store/v2/archive"
	"chainmaker.org/chainmaker/store/v2/bigfilterdb"
	"chainmaker.org/chainmaker/store/v2/binlog"
	"chainmaker.org/chainmaker/store/v2/blockdb"
	"chainmaker.org/chainmaker/store/v2/conf"
	"chainmaker.org/chainmaker/store/v2/contracteventdb"
	"chainmaker.org/chainmaker/store/v2/historydb"
	"chainmaker.org/chainmaker/store/v2/resultdb"
	"chainmaker.org/chainmaker/store/v2/rolling_window_cache"
	"chainmaker.org/chainmaker/store/v2/serialization"
	"chainmaker.org/chainmaker/store/v2/statedb"
	"chainmaker.org/chainmaker/store/v2/txexistdb"
	"chainmaker.org/chainmaker/store/v2/types"
	"chainmaker.org/chainmaker/utils/v2"
	"github.com/gogo/protobuf/proto"
	"golang.org/x/sync/semaphore"
)

const (
	logPath = "bfdb"
	//logDBBlockKeyPrefix = 'n'
)

const (
	blockFilePath = "bfdb"
	walLogPath    = "wal"
)

var (
	//errGetObjPool = errors.New("get obj error from syncPool")
	errGetBufPool = errors.New("get bufer error from syncPool")
)

type writeBatch struct {
	//blockBytes              []byte
	//block                   *commonPb.Block
	blockWithSerializedInfo *serialization.BlockWithSerializedInfo
}

// BlockStoreImpl provides an implementation of `protocol.BlockchainStore`.
// @Description:
type BlockStoreImpl struct {
	blockDB         blockdb.BlockDB
	stateDB         statedb.StateDB
	historyDB       historydb.HistoryDB
	resultDB        resultdb.ResultDB
	contractEventDB contracteventdb.ContractEventDB
	txExistDB       txexistdb.TxExistDB
	blockFileDB     binlog.BinLogger
	walLog          *wal.Log
	bigFilterDB     bigfilterdb.BigFilterDB
	//一个本地数据库，用于对外提供一些本节点的数据存储服务
	commonDB           protocol.DBHandle
	ArchiveMgr         *archive.ArchiveMgr
	workersSemaphore   *semaphore.Weighted
	logger             protocol.Logger
	storeConfig        *conf.StorageConfig
	rollingWindowCache rolling_window_cache.RollingWindowCache
	//writeBatchChan   chan *serialization.BlockWithSerializedInfo
	writeBatchChan          chan writeBatch
	blockSerializedInfoPool sync.Pool
	protoBufferPool         sync.Pool
	//记录慢日志的阈值，默认0表示不记录
	SlowLogThreshold int64
}

// NewBlockStoreImpl constructs new `BlockStoreImpl`
// @Description:
// @param chainId
// @param storeConfig
// @param blockDB
// @param stateDB
// @param historyDB
// @param contractEventDB
// @param resultDB
// @param txExistDB
// @param commonDB
// @param logger
// @param bfdb
// @param walLog
// @param bigFilterDB
// @param rwCache
// @return *BlockStoreImpl
// @return error
func NewBlockStoreImpl(chainId string,
	storeConfig *conf.StorageConfig,
	blockDB blockdb.BlockDB,
	stateDB statedb.StateDB,
	historyDB historydb.HistoryDB,
	contractEventDB contracteventdb.ContractEventDB,
	resultDB resultdb.ResultDB,
	txExistDB txexistdb.TxExistDB,
	commonDB protocol.DBHandle,
	logger protocol.Logger,
	bfdb binlog.BinLogger,
	walLog *wal.Log,
	bigFilterDB bigfilterdb.BigFilterDB,
	rwCache rolling_window_cache.RollingWindowCache) (*BlockStoreImpl, error) {
	nWorkers := runtime.NumCPU()

	blockStore := &BlockStoreImpl{
		blockDB:                 blockDB,
		stateDB:                 stateDB,
		historyDB:               historyDB,
		contractEventDB:         contractEventDB,
		resultDB:                resultDB,
		txExistDB:               txExistDB,
		blockFileDB:             bfdb,
		walLog:                  walLog,
		bigFilterDB:             bigFilterDB,
		rollingWindowCache:      rwCache,
		commonDB:                commonDB,
		workersSemaphore:        semaphore.NewWeighted(int64(nWorkers)),
		logger:                  logger,
		storeConfig:             storeConfig,
		writeBatchChan:          make(chan writeBatch, 10),
		blockSerializedInfoPool: sync.Pool{},
		protoBufferPool:         sync.Pool{},
		SlowLogThreshold:        storeConfig.SlowLog,
	}
	blockStore.blockSerializedInfoPool.New = func() interface{} {
		//return &serialization.BlockWithSerializedInfo{}
		// new blockSerializedInfo.Meta is not nil
		return serialization.NewBlockSerializedInfo()
	}

	blockStore.protoBufferPool.New = func() interface{} {
		return proto.NewBuffer(nil)
	}

	if err := blockStore.InitArchiveMgr(chainId); err != nil {
		return nil, err
	}

	//binlog 有SavePoint，不是空数据库，进行数据恢复
	if i, errbs := blockStore.getLastFileSavepoint(); errbs == nil && i > 0 {
		//check savepoint and recover
		errbs = blockStore.recover()
		if errbs != nil {
			return nil, errbs
		}
	} else {
		logger.Info("binlog is empty, don't need recover")
	}

	//存储模块的写入逻辑:先同步写wal,和chan，成功后立刻返回，其他的db写入 通过消费chan,异步写完成
	//启动一个groutine 去消费chan，完成异步 写 blockDB,stateDB,historyDB,resultDB,contractEventDB,
	go blockStore.WriteBatchFromChanToDB()

	return blockStore, nil
}

// InitGenesis 初始化创世区块到数据库，对应的数据库必须为空数据库，否则报错
// @Description:
// @receiver bs
// @param genesisBlock
// @return error
func (bs *BlockStoreImpl) InitGenesis(genesisBlock *storePb.BlockWithRWSet) error {
	bs.logger.Debug("start initial genesis block to database...")
	bs.logger.InfoDynamic(func() string {
		j, _ := json.Marshal(genesisBlock)
		return "Genesis JSON:" + string(j)
	})
	//1.检查创世区块是否有异常
	if err := checkGenesis(genesisBlock); err != nil {
		return err
	}
	//创世区块只执行一次，而且可能涉及到创建创建数据库，所以串行执行，而且无法启用事务
	blockBytes, blockWithSerializedInfo, err := serialization.SerializeBlock(genesisBlock)
	if err != nil {
		return err
	}
	block := genesisBlock.Block
	blockIndex, err := bs.writeBlockToFile(block.Header.BlockHeight, blockBytes)
	if err != nil {
		return err
	}
	blockWithSerializedInfo.Index = blockIndex

	//2.初始化BlockDB
	err = bs.blockDB.InitGenesis(blockWithSerializedInfo)
	if err != nil {
		bs.logger.Errorf("chain[%s] failed to write blockDB, block[%d]",
			block.Header.ChainId, block.Header.BlockHeight)
		return err
	}
	//3. 初始化StateDB
	err = bs.stateDB.InitGenesis(blockWithSerializedInfo)
	if err != nil {
		bs.logger.Errorf("chain[%s] failed to write stateDB, block[%d]",
			block.Header.ChainId, block.Header.BlockHeight)
		return err
	}
	//4. 初始化历史数据库
	if !bs.storeConfig.DisableHistoryDB {
		err = bs.historyDB.InitGenesis(blockWithSerializedInfo)
		if err != nil {
			bs.logger.Errorf("chain[%s] failed to write historyDB, block[%d]",
				block.Header.ChainId, block.Header.BlockHeight)
			return err
		}
	}
	//5. 初始化Result数据库
	if !bs.storeConfig.DisableResultDB {
		err = bs.resultDB.InitGenesis(blockWithSerializedInfo)
		if err != nil {
			bs.logger.Errorf("chain[%s] failed to write resultDB, block[%d]",
				block.Header.ChainId, block.Header.BlockHeight)
			return err
		}
	}
	//6. init contract event db
	if !bs.storeConfig.DisableContractEventDB {
		//if parseEngineType(bs.storeConfig.ContractEventDbConfig.SqlDbConfig.SqlDbType) == types.MySQL &&
		//	bs.storeConfig.ContractEventDbConfig.Provider == localconf.DbconfigProviderSql {
		err = bs.contractEventDB.InitGenesis(blockWithSerializedInfo)
		if err != nil {
			bs.logger.Errorf("chain[%s] failed to write event db, block[%d]",
				block.Header.ChainId, block.Header.BlockHeight)
			return err
		}
		//} else {
		//	return errors.New("contract event db config err")
		//}
	}
	//7. 初始化TxExistDB数据库
	err = bs.txExistDB.InitGenesis(blockWithSerializedInfo)
	if err != nil {
		bs.logger.Errorf("chain[%s] failed to write txExistDB, block[%d]",
			block.Header.ChainId, block.Header.BlockHeight)
		return err
	}

	//8. 初始化TxExistDB数据库
	if bs.storeConfig.EnableBigFilter {
		err = bs.bigFilterDB.InitGenesis(blockWithSerializedInfo)
		if err != nil {
			bs.logger.Errorf("chain[%s] failed to write bigFilterDB, block[%d]",
				block.Header.ChainId, block.Header.BlockHeight)
			return err
		}
	}

	//9. 初始化rollingWindowCache

	err = bs.rollingWindowCache.InitGenesis(blockWithSerializedInfo)
	if err != nil {
		bs.logger.Errorf("chain[%s] failed to write rollingWindowCache, block[%d]",
			block.Header.ChainId, block.Header.BlockHeight)
		return err
	}

	bs.logger.Infof("chain[%s]: put block[%d] hash[%x] (txs:%d bytes:%d), ",
		block.Header.ChainId, block.Header.BlockHeight, block.Header.BlockHash, len(block.Txs), len(blockBytes))

	//10. init archive manager
	err = bs.InitArchiveMgr(block.Header.ChainId)
	if err != nil {
		return err
	}

	return err
}
func checkGenesis(genesisBlock *storePb.BlockWithRWSet) error {
	if genesisBlock.Block.Header.BlockHeight != 0 {
		return errors.New("genesis block height must be 0")
	}
	return nil
}

// PutBlock commits the block and the corresponding rwsets in an atomic operation
// @Description:
// 如果是普通写入模式，先后写 kvCache,wal,kvdb 然后返回
// 如果是快速写模式，先写 kvCache,wal,chan 然后返回 ，chan中数据由单独的groutine负责完成 消费写到 db中
// @receiver bs
// @param block
// @param txRWSets
// @return error
func (bs *BlockStoreImpl) PutBlock(block *commonPb.Block, txRWSets []*commonPb.TxRWSet) error {
	var err error
	switch bs.storeConfig.WriteBlockType {
	case conf.CommonWriteBlockType:
		//普通写模式
		err = bs.CommonPutBlock(block, txRWSets)
	case conf.QuickWriteBlockType:
		//快速写模式
		err = bs.QuickPutBlock(block, txRWSets)
	default:
		err = errors.New("config error,write_block_type: " + strconv.Itoa(bs.storeConfig.WriteBlockType))
	}

	if err != nil && strings.Contains(err.Error(), "Canceled or timeout") {
		bs.logger.Errorf(err.Error())
		return commonErr.ErrStoreServiceNeedRestarted
	}

	return err
}

// WriteKvDbCacheSqlDb commit block to kvdb cache and sqldb
// @Description:
// 写block,state,history,result,bigfilter 5种kvdb cache或者对应的sqldb，
// 写1个 contractEventDB(sqldb), 1个 txExistDB(kvdb)  txExistDB不支持sql型,
// 写1个 rollingWindowCache
// 一共8个groutine
// @receiver bs
// @param blockWithSerializedInfo
// @param errsChan
// @return error
func (bs *BlockStoreImpl) WriteKvDbCacheSqlDb(blockWithSerializedInfo *serialization.BlockWithSerializedInfo,
	errsChan chan error) error {

	wg := sync.WaitGroup{}
	wg.Add(8)

	// 1.blockDB
	go func(bs *BlockStoreImpl, errsChan chan error) {
		defer wg.Done()

		if bs.storeConfig.BlockDbConfig.IsKVDB() {
			// update blockDB Cache
			bs.putBlock2DB(blockWithSerializedInfo, errsChan, bs.blockDB.CommitBlock, true)
		}
		if bs.storeConfig.BlockDbConfig.IsSqlDB() {
			// update blockDB SqlDB
			bs.putBlock2DB(blockWithSerializedInfo, errsChan, bs.blockDB.CommitBlock, false)
		}
	}(bs, errsChan)

	// 2.stateDB
	go func(bs *BlockStoreImpl, errsChan chan error) {
		defer wg.Done()
		if bs.storeConfig.StateDbConfig.IsKVDB() {
			// update stateDB Cache
			bs.putBlock2DB(blockWithSerializedInfo, errsChan, bs.stateDB.CommitBlock, true)
		}
		if bs.storeConfig.StateDbConfig.IsSqlDB() {
			bs.putBlock2DB(blockWithSerializedInfo, errsChan, bs.stateDB.CommitBlock, false)
		}
	}(bs, errsChan)

	// 3.historyDB
	go func(bs *BlockStoreImpl, errsChan chan error) {
		defer wg.Done()
		if !bs.storeConfig.DisableHistoryDB {
			if bs.storeConfig.HistoryDbConfig.IsKVDB() {
				// update stateDB Cache
				bs.putBlock2DB(blockWithSerializedInfo, errsChan, bs.historyDB.CommitBlock, true)
			}
			if bs.storeConfig.HistoryDbConfig.IsSqlDB() {
				bs.putBlock2DB(blockWithSerializedInfo, errsChan, bs.historyDB.CommitBlock, false)
			}
		}
	}(bs, errsChan)

	// 4.resultDB
	go func(bs *BlockStoreImpl, errsChan chan error) {
		defer wg.Done()
		if !bs.storeConfig.DisableResultDB {
			if bs.storeConfig.ResultDbConfig.IsKVDB() {
				// update stateDB Cache
				bs.putBlock2DB(blockWithSerializedInfo, errsChan, bs.resultDB.CommitBlock, true)
			}
			if bs.storeConfig.ResultDbConfig.IsSqlDB() {
				bs.putBlock2DB(blockWithSerializedInfo, errsChan, bs.resultDB.CommitBlock, false)
			}
		}
	}(bs, errsChan)

	// 5.contractEventDB ,only DB ,contractEventDB has't Cache
	go func(bs *BlockStoreImpl, errsChan chan error) {
		defer wg.Done()
		if !bs.storeConfig.DisableContractEventDB {
			bs.putBlock2DB(blockWithSerializedInfo, errsChan, bs.contractEventDB.CommitBlock, false)
		}
	}(bs, errsChan)

	// 6.txExistDB ,only DB ,txExistDB has't Cache
	go func(bs *BlockStoreImpl, errsChan chan error) {
		defer wg.Done()
		//if !bs.storeConfig.DisableTxExistDB {
		bs.putBlock2DB(blockWithSerializedInfo, errsChan, bs.txExistDB.CommitBlock, false)
		//}
	}(bs, errsChan)

	// 7.bigFilterDB
	go func(bs *BlockStoreImpl, errsChan chan error) {
		defer wg.Done()
		if bs.storeConfig.EnableBigFilter {
			bs.putBlock2DB(blockWithSerializedInfo, errsChan, bs.bigFilterDB.CommitBlock, true)
		}
	}(bs, errsChan)

	// 8.rollingWindowCache
	go func(bs *BlockStoreImpl, errsChan chan error) {
		defer wg.Done()
		bs.putBlock2DB(blockWithSerializedInfo, errsChan, bs.rollingWindowCache.CommitBlock, true)

	}(bs, errsChan)

	wg.Wait()

	//block := blockWithSerializedInfo.Block
	//bs.logger.Debugf("chain[%s]: start put block[%d] (txs:%d) start writeBatchChan currtime[%d]",
	//	block.Header.ChainId, block.Header.BlockHeight, len(block.Txs), utils.CurrentTimeMillisSeconds())
	return nil
}

// WriteKvDb commit block to kvdb
// @Description:
// 写 block,state,history,result,bigfilter 5种kvdb,不包含contractevent db, 合约db只有 sql型，没有kv型.
// @receiver bs
// @param blockWithSerializedInfo
// @param errsChan
// @return error
func (bs *BlockStoreImpl) WriteKvDb(blockWithSerializedInfo *serialization.BlockWithSerializedInfo,
	errsChan chan error) error {

	wg := sync.WaitGroup{}
	wg.Add(5)
	start := time.Now()
	var endBlock, endState, endHistory, endResult time.Time //用于统计耗时
	// 1.blockDB
	go func(bs *BlockStoreImpl, errsChan chan error) {
		defer func() {
			endBlock = time.Now()
			wg.Done()
		}()
		if bs.storeConfig.BlockDbConfig.IsKVDB() {
			// update blockDB
			bs.putBlock2DB(blockWithSerializedInfo, errsChan, bs.blockDB.CommitBlock, false)
		}
	}(bs, errsChan)

	// 2.stateDB
	go func(bs *BlockStoreImpl, errsChan chan error) {
		defer func() {
			endState = time.Now()
			wg.Done()
		}()
		if bs.storeConfig.StateDbConfig.IsKVDB() {
			// update stateDB
			bs.putBlock2DB(blockWithSerializedInfo, errsChan, bs.stateDB.CommitBlock, false)
		}
	}(bs, errsChan)

	// 3.historyDB
	go func(bs *BlockStoreImpl, errsChan chan error) {
		defer func() {
			endHistory = time.Now()
			wg.Done()
		}()
		if !bs.storeConfig.DisableHistoryDB {
			if bs.storeConfig.HistoryDbConfig.IsKVDB() {
				// update stateDB
				bs.putBlock2DB(blockWithSerializedInfo, errsChan, bs.historyDB.CommitBlock, false)
			}
		}
	}(bs, errsChan)

	// 4.resultDB
	go func(bs *BlockStoreImpl, errsChan chan error) {
		defer func() {
			endResult = time.Now()
			wg.Done()
		}()
		if !bs.storeConfig.DisableResultDB {
			if bs.storeConfig.ResultDbConfig.IsKVDB() {
				// update stateDB
				bs.putBlock2DB(blockWithSerializedInfo, errsChan, bs.resultDB.CommitBlock, false)
			}
		}
	}(bs, errsChan)

	// 5.bigFilterDB
	go func(bs *BlockStoreImpl, errsChan chan error) {
		defer func() {
			endResult = time.Now()
			wg.Done()
		}()
		if bs.storeConfig.EnableBigFilter {
			bs.putBlock2DB(blockWithSerializedInfo, errsChan, bs.bigFilterDB.CommitBlock, false)

		}
	}(bs, errsChan)

	wg.Wait()
	bs.logger.InfoDynamic(func() string {
		block := blockWithSerializedInfo.Block
		return fmt.Sprintf(
			"chain[%s]: write block[%d] (txs:%d) kvdb spend: blockdb:%d,statedb:%d,historydb:%d,resultdb:%d,total:%d",
			block.Header.ChainId, block.Header.BlockHeight, len(block.Txs),
			endBlock.Sub(start).Milliseconds(), endState.Sub(start).Milliseconds(),
			endHistory.Sub(start).Milliseconds(), endResult.Sub(start).Milliseconds(),
			time.Since(start).Milliseconds())
	})

	return nil
}

// CommonPutBlock add next time
// @Description:
// 普通写模式，占用资源少，写入慢
// 1.写wal
// 2.写kvdb cache 或者 sql
// 3.写kvdb 或者什么都不做
// 4.删除过期的wal
// @receiver bs
// @param block
// @param txRWSets
// @return error
func (bs *BlockStoreImpl) CommonPutBlock(block *commonPb.Block, txRWSets []*commonPb.TxRWSet) error {
	startTime := time.Now()
	//序列化
	blockWithRWSet := &storePb.BlockWithRWSet{
		Block:    block,
		TxRWSets: txRWSets,
	}
	//blockBytes, blockWithSerializedInfo, err := serialization.SerializeBlock(blockWithRWSet)
	buf, ok := bs.protoBufferPool.Get().(*proto.Buffer)
	if !ok {
		bs.logger.Errorf("chain[%s]: put block[%d] (txs:%d) when proto buffer pool get",
			block.Header.ChainId, block.Header.BlockHeight, len(block.Txs))
		return errGetBufPool
	}
	buf.Reset()

	blockBytes, blockWithSerializedInfo, err := serialization.SerializeBlock(blockWithRWSet)
	if err != nil {
		bs.logger.Errorf("chain[%s] failed to write log, block[%d], err:%s",
			block.Header.ChainId, block.Header.BlockHeight, err)
		return err
	}

	marshalDur := time.Since(startTime)
	errsChan := make(chan error, 6)

	// 1.write wal
	blockIndex, err := bs.writeBlockToFile(block.Header.BlockHeight, blockBytes)
	blockWithSerializedInfo.Index = blockIndex
	writeFileDur := time.Since(startTime)
	//放回对象池
	bs.protoBufferPool.Put(buf)
	if err != nil {
		bs.logger.Errorf("chain[%s] failed to write log, block[%d], err:%s",
			block.Header.ChainId, block.Header.BlockHeight, err)
		return err
	}

	// 2.写 kvdb Cache 或者 写sql
	//err = bs.WriteKvDbCacheSqlDb(blockWithSerializedInfo, errsChan)
	err = bs.WriteKvDbCacheSqlDb(blockWithSerializedInfo, errsChan)
	if err != nil {
		bs.logger.Errorf("chain[%s] failed to write KvDbCacheSqldb, block[%d], err:%s",
			block.Header.ChainId, block.Header.BlockHeight, err)
		return err
	}
	writeCacheDur := time.Since(startTime)
	//以上写WriteKvDbCacheSqlDb,有一个写入失败，返回第一个错误
	if len(errsChan) > 0 {
		return <-errsChan
	}

	// 3.写 kvdb
	err = bs.WriteKvDb(blockWithSerializedInfo, errsChan)
	if err != nil {
		bs.logger.Errorf("chain[%s] failed to write WriteKvDb, block[%d], err:%s",
			block.Header.ChainId, block.Header.BlockHeight, err)
		return err
	}

	writeKvDBDur := time.Since(startTime)
	//WriteKvDb,有一个写入失败，返回第一个错误
	if len(errsChan) > 0 {
		return <-errsChan
	}
	// 4.删除wal,每100个block删除一次
	go func() {
		err = bs.deleteBlockFromLog(block.Header.BlockHeight)
		if err != nil {
			bs.logger.Warnf("chain[%s]: failed to clean log, block[%d], err:%s",
				block.Header.ChainId, block.Header.BlockHeight, err)
		}
	}()

	bs.logger.Infof("chain[%s]: put block[%d] common (txs:%d, bytes: %d), time used: "+
		"marshal: %v, writeFile: %v, writeCache: %v, writeKvDB: %v, total: %v", block.Header.ChainId,
		block.Header.BlockHeight, len(block.Txs), len(blockBytes), marshalDur.Milliseconds(),
		(writeFileDur - marshalDur).Milliseconds(), (writeCacheDur - writeFileDur).Milliseconds(),
		(writeKvDBDur - writeCacheDur).Milliseconds(), time.Since(startTime).Milliseconds())

	return nil
}

// QuickPutBlock 模式，写入和读取性能更好，占用内存更多
// @Description:
// 1.写wal
// 2.写kvdb cache 或者 sql
// 3.写channel
//	判断5种db,是 kv型，还是sql型，kv型 则 因为 kv 型 有对应Cache，可以直接同步更新Cache
//	如果是sql型，则 因为 sql 型 没有Cache，直接同步更新db
//	再写 data 到 writeBatchChan
//	同理，消费 writeBatchChan时，也要 判断，如果是 sql 型，则不需要 消费chan了，因为前面已经 同步更新过了
//	如果 是 kv 型，则 消费 chan ，然后 同步更新
//	依次判断 blockDB,stateDB,historyDB,resultDB,contractEventDB 对应是 sql型存储还是 kv型存储
//	如果是 kv型存储，则直接更新其对应的Cache，如果(是sql型)不是(kv型)，则同步更新
//	根据配置，同步写入对应db或Cache,blockDB,stateDB,historyDB,resultDB,contractEventDB,如果写入失败，直接panic
// @receiver bs
// @param block
// @param txRWSets
// @return error
func (bs *BlockStoreImpl) QuickPutBlock(block *commonPb.Block, txRWSets []*commonPb.TxRWSet) error {
	startTime := time.Now()
	errsChan := make(chan error, 6)
	//序列化数据
	blockWithRWSet := &storePb.BlockWithRWSet{
		Block:    block,
		TxRWSets: txRWSets,
	}
	//blockBytes, blockWithSerializedInfo, err := serialization.SerializeBlock(blockWithRWSet)
	buf, ok := bs.protoBufferPool.Get().(*proto.Buffer)
	if !ok {
		bs.logger.Errorf("chain[%s]: put block[%d] (txs:%d) when proto buffer pool get",
			block.Header.ChainId, block.Header.BlockHeight, len(block.Txs))
		return errGetBufPool
	}
	buf.Reset()

	blockBytes, blockWithSerializedInfo, err := serialization.SerializeBlock(blockWithRWSet)
	if err != nil {
		bs.logger.Errorf("chain[%s] failed to write log, block[%d], err:%s",
			block.Header.ChainId, block.Header.BlockHeight, err)
		return err
	}
	marshalDur := time.Since(startTime)

	// 1.write wal
	blockIndex, err := bs.writeBlockToFile(block.Header.BlockHeight, blockBytes)
	blockWithSerializedInfo.Index = blockIndex
	//放回对象池
	bs.protoBufferPool.Put(buf)
	if err != nil {
		bs.logger.Errorf("chain[%s] failed to write log, block[%d], err:%s",
			block.Header.ChainId, block.Header.BlockHeight, err)
		return err
	}
	writeFileDur := time.Since(startTime)

	// 2.写 kvdb Cache 或者 写sql
	err = bs.WriteKvDbCacheSqlDb(blockWithSerializedInfo, errsChan)
	if err != nil {
		bs.logger.Errorf("chain[%s] failed to write KvDbCacheSqldb, block[%d], err:%s",
			block.Header.ChainId, block.Header.BlockHeight, err)
		return err
	}
	writeCacheDur := time.Since(startTime)

	//以上写WriteKvDbCacheSqlDb,有一个写入失败，返回第一个错误
	if len(errsChan) > 0 {
		return <-errsChan
	}

	// 3. writeBatchChan
	//同步写入WAL成功之后，再写入writeBatchChan，供后续blockDB,stateDB,historyDB,resultDB,contractEventDB异步 消费
	wBatch := writeBatch{
		blockWithSerializedInfo: blockWithSerializedInfo,
	}
	bs.writeBatchChan <- wBatch

	writeBatchChanDur := time.Since(startTime)

	bs.logger.Infof("chain[%s]: put block[%d] quick (txs:%d, bytes: %d, writeBatchChanLen: %d), "+
		"time used: marshal: %v, writeFile: %v, writeCache: %v, writeBatchChan: %v, total: %v", block.Header.ChainId,
		block.Header.BlockHeight, len(block.Txs), len(blockBytes), len(bs.writeBatchChan), marshalDur.Milliseconds(),
		(writeFileDur - marshalDur).Milliseconds(), (writeCacheDur - writeFileDur).Milliseconds(),
		(writeBatchChanDur - writeCacheDur).Milliseconds(), time.Since(startTime).Milliseconds())

	return nil
}

// WriteBatchFromChanToDB 消费chan 中数据，同步写到db
// @Description:
// 从一个chan中，消费需要批量写入的序列化好的块
// 1.写kvdb
// 2.删除wal中，当前block前10个block
// 3.blockWithSerializedInfo 放回对象池
// @receiver bs
func (bs *BlockStoreImpl) WriteBatchFromChanToDB() {
	for wBatch := range bs.writeBatchChan {
		start := time.Now()
		block := wBatch.blockWithSerializedInfo.Block
		blockWithSerializedInfo := wBatch.blockWithSerializedInfo

		//commit db concurrently

		//the amount of commit db work
		numBatches := 4
		errsChan := make(chan error, numBatches)

		//1.直接写kvdb, 包括block,state,history,result,bigfilter
		err := bs.WriteKvDb(blockWithSerializedInfo, errsChan)
		if err != nil {
			bs.logger.Errorf("chain[%s] failed to write WriteKvDb, block[%d], err:%s",
				block.Header.ChainId, block.Header.BlockHeight, err)
		}
		//WriteKvDb,有一个db写入失败，则直接panic
		if len(errsChan) > 0 {
			bs.logger.Errorf("chain[%s]: put block[%d] WriteBatchToDB error ",
				block.Header.ChainId, block.Header.BlockHeight)
			panic(<-errsChan)
		}
		writeKvDbDur := time.Since(start)
		//2. clean wal, 删除当前block，之前的10个块
		//err := bs.deleteBlockFromLog(block.Header.BlockHeight)
		//closeChan := make(chan struct{}, 1)
		go func(index uint64, lastN uint64) {
			//协程结束关闭 closeChan
			//defer close(closeChan)
			if err1 := bs.deleteLastNumBlockFromLog(index, lastN); err1 != nil {
				bs.logger.Warnf("chain[%s]: failed to clean log, block[%d], err:%s",
					block.Header.ChainId, block.Header.BlockHeight, err1)
			}
		}(block.Header.BlockHeight, 100)

		//<-closeChan

		//3. 将 blockWithSerializedInfo 重置数据，放到 pool 里面
		//blockWithSerializedInfo.ReSet()
		//bs.blockSerializedInfoPool.Put(blockWithSerializedInfo)

		bs.logger.Infof("chain[%s]: put block[%d] chan (txs:%d, chanLen: %d) write batch "+
			"from chan to db, time used: writeKvDb: %d, total: %d", block.Header.ChainId,
			block.Header.BlockHeight, len(block.Txs), len(bs.writeBatchChan), writeKvDbDur.Milliseconds(),
			time.Since(start).Milliseconds())
	}
}

// GetArchivedPivot  return archived pivot
// @Description:
// @receiver bs
// @return uint64
func (bs *BlockStoreImpl) GetArchivedPivot() uint64 {
	if !bs.isSupportArchive() {
		return 0
	}
	height, _ := bs.ArchiveMgr.GetArchivedPivot()
	return height
}

// ArchiveBlock the block after backup
// @Description:
// @receiver bs
// @param archiveHeight
// @return error
func (bs *BlockStoreImpl) ArchiveBlock(archiveHeight uint64) error {
	if !bs.isSupportArchive() {
		return nil
	}
	return bs.ArchiveMgr.ArchiveBlock(archiveHeight)
}

// RestoreBlocks restore blocks from outside serialized block data
// @Description:
// @receiver bs
// @param serializedBlocks
// @return error
func (bs *BlockStoreImpl) RestoreBlocks(serializedBlocks [][]byte) error {
	if !bs.isSupportArchive() {
		return nil
	}
	blockInfos := make([]*serialization.BlockWithSerializedInfo, 0, len(serializedBlocks))
	for _, blockInfo := range serializedBlocks {
		bwsInfo, err := serialization.DeserializeBlock(blockInfo)
		if err != nil {
			return err
		}
		_, s, err := serialization.SerializeBlock(bwsInfo)
		if err != nil {
			return err
		}
		blockInfos = append(blockInfos, s)
	}

	return bs.ArchiveMgr.RestoreBlock(blockInfos)
}

type commitBlock func(blockInfo *serialization.BlockWithSerializedInfo, isCache bool) error

// putBlock2DB add next time
// @Description:
// @receiver bs
// @param blockWithSerializedInfo
// @param errsChan
// @param commit
// @param isCache
func (bs *BlockStoreImpl) putBlock2DB(blockWithSerializedInfo *serialization.BlockWithSerializedInfo,
	errsChan chan error, commit commitBlock, isCache bool) {
	err := commit(blockWithSerializedInfo, isCache)
	block := blockWithSerializedInfo.Block
	if err != nil {
		bs.logger.Errorf("chain[%s] failed to write DB, block[%d]",
			block.Header.ChainId, block.Header.BlockHeight)
		errsChan <- err
	}
}

// BlockExists returns true if the black hash exist, or returns false if none exists.
// @Description:
// @receiver bs
// @param blockHash
// @return bool
// @return error
func (bs *BlockStoreImpl) BlockExists(blockHash []byte) (bool, error) {
	return bs.blockDB.BlockExists(blockHash)
}

// GetBlockByHash returns a block given it's hash, or returns nil if none exists.
// @Description:
// @receiver bs
// @param blockHash
// @return *commonPb.Block
// @return error
func (bs *BlockStoreImpl) GetBlockByHash(blockHash []byte) (*commonPb.Block, error) {
	//return bs.blockDB.GetBlockByHash(blockHash)
	height, err := bs.blockDB.GetHeightByHash(blockHash)
	if err != nil {
		return nil, err
	}
	return bs.GetBlock(height)
}

// GetHeightByHash returns a block height given it's hash, or returns nil if none exists.
// @Description:
// @receiver bs
// @param blockHash
// @return uint64
// @return error
func (bs *BlockStoreImpl) GetHeightByHash(blockHash []byte) (uint64, error) {
	return bs.blockDB.GetHeightByHash(blockHash)
}

// GetBlockHeaderByHeight returns a block header by given it's height, or returns nil if none exists.
// @Description:
// @receiver bs
// @param height
// @return *commonPb.BlockHeader
// @return error
func (bs *BlockStoreImpl) GetBlockHeaderByHeight(height uint64) (*commonPb.BlockHeader, error) {
	meta, err := bs.getBlockMetaFromFile(height)
	if err != nil {
		return nil, err
	}
	if meta != nil {
		return meta.Header, nil
	}

	blk, err := bs.GetBlock(height)
	if err != nil || blk == nil {
		return nil, err
	}
	return blk.Header, nil
}

// GetBlock returns a block given it's block height, or returns nil if none exists.
// @Description:
// @receiver bs
// @param height
// @return *commonPb.Block
// @return error
func (bs *BlockStoreImpl) GetBlock(height uint64) (*commonPb.Block, error) {

	return bs.blockDB.GetBlock(height)
}

// GetLastBlock returns the last block.
// @Description:
// @receiver bs
// @return *commonPb.Block
// @return error
func (bs *BlockStoreImpl) GetLastBlock() (*commonPb.Block, error) {
	var (
		err        error
		blockPoint uint64
		statePoint uint64
	)
	if blockPoint, err = bs.blockDB.GetLastSavepoint(); err != nil {
		return nil, err
	}
	if statePoint, err = bs.stateDB.GetLastSavepoint(); err != nil {
		return nil, err
	}

	if blockPoint != statePoint {
		usePoint := blockPoint
		if usePoint > statePoint {
			usePoint = statePoint
		}
		bs.logger.Warnf("block LastSavepoint: %d is not match state LastSavepoint: %d, will use: %d",
			blockPoint, statePoint, usePoint)
		return bs.GetBlock(usePoint)
	}
	return bs.GetBlock(blockPoint)
}

// GetLastConfigBlock returns the last config block.
// @Description:
// @receiver bs
// @return *commonPb.Block
// @return error
func (bs *BlockStoreImpl) GetLastConfigBlock() (*commonPb.Block, error) {
	return bs.blockDB.GetLastConfigBlock()
}

// GetLastChainConfig returns the last chain config
// @Description:
// @receiver bs
// @return *configPb.ChainConfig
// @return error
func (bs *BlockStoreImpl) GetLastChainConfig() (*configPb.ChainConfig, error) {
	return bs.stateDB.GetChainConfig()
}

// GetBlockByTx returns a block which contains a tx.
// @Description:
// @receiver bs
// @param txId
// @return *commonPb.Block
// @return error
func (bs *BlockStoreImpl) GetBlockByTx(txId string) (*commonPb.Block, error) {
	height, err := bs.blockDB.GetTxHeight(txId)
	if err != nil {
		return nil, err
	}
	return bs.GetBlock(height)
	//return bs.blockDB.GetBlockByTx(txId)
}

// GetTx retrieves a transaction by txid, or returns nil if none exists.
// @Description:
// @receiver bs
// @param txId
// @return *commonPb.Transaction
// @return error
func (bs *BlockStoreImpl) GetTx(txId string) (*commonPb.Transaction, error) {
	if len(txId) == 0 {
		return nil, errors.New("input txid is empty")
	}
	return bs.blockDB.GetTx(txId)
}

// GetTxWithRWSet return tx and it's rw set
// @Description:
// @receiver bs
// @param txId
// @return *commonPb.TransactionWithRWSet
// @return error
func (bs *BlockStoreImpl) GetTxWithRWSet(txId string) (*commonPb.TransactionWithRWSet, error) {
	if len(txId) == 0 {
		return nil, errors.New("input txid is empty")
	}
	tx, err := bs.blockDB.GetTx(txId)
	if err != nil {
		return nil, err
	}
	if tx == nil { //如果找不到，那么返回nil,nil
		return nil, nil
	}
	rwset, err := bs.resultDB.GetTxRWSet(txId)
	if err != nil {
		return nil, err
	}
	return &commonPb.TransactionWithRWSet{
		Transaction: tx,
		RwSet:       rwset,
	}, nil
}

// GetTxInfoWithRWSet return tx and tx info and rw set
// @Description:
// @receiver bs
// @param txId
// @return *commonPb.TransactionInfoWithRWSet
// @return error
func (bs *BlockStoreImpl) GetTxInfoWithRWSet(txId string) (*commonPb.TransactionInfoWithRWSet, error) {
	txInfo, err := bs.blockDB.GetTxWithBlockInfo(txId)
	if err != nil {
		return nil, err
	}
	if txInfo == nil { //如果找不到，那么返回nil,nil
		return nil, nil
	}
	rwset, err := bs.resultDB.GetTxRWSet(txId)
	if err != nil {
		return nil, err
	}
	return &commonPb.TransactionInfoWithRWSet{
		Transaction:    txInfo.Transaction,
		BlockHeight:    txInfo.BlockHeight,
		BlockHash:      txInfo.BlockHash,
		TxIndex:        txInfo.TxIndex,
		BlockTimestamp: txInfo.BlockTimestamp,
		RwSet:          rwset,
	}, nil
}

// GetTxWithInfo add next time
// @Description:
// @receiver d
// @param txId
// @return *commonPb.TransactionInfo
// @return error
func (d *BlockStoreImpl) GetTxWithInfo(txId string) (*commonPb.TransactionInfo, error) {
	txInfo, err := d.blockDB.GetTxWithBlockInfo(txId)
	return convertTxInfo(txInfo), err
}

// convertTxInfo add next time
// @Description:
// @param txInfo
// @return *commonPb.TransactionInfo
func convertTxInfo(txInfo *storePb.TransactionStoreInfo) *commonPb.TransactionInfo {
	if txInfo == nil {
		return nil
	}
	return &commonPb.TransactionInfo{
		Transaction:    txInfo.Transaction,
		BlockHeight:    txInfo.BlockHeight,
		BlockHash:      txInfo.BlockHash,
		TxIndex:        txInfo.TxIndex,
		BlockTimestamp: txInfo.BlockTimestamp,
	}
}

// GetTxInfoOnly add next time
// @Description:
// @receiver d
// @param txId
// @return *commonPb.TransactionInfo
// @return error
func (d *BlockStoreImpl) GetTxInfoOnly(txId string) (*commonPb.TransactionInfo, error) {
	txInfo, err := d.blockDB.GetTxInfoOnly(txId)
	return convertTxInfo(txInfo), err
}

// GetTxHeight retrieves a transaction height by txid, or returns nil if none exists.
// @Description:
// @receiver bs
// @param txId
// @return uint64
// @return error
func (bs *BlockStoreImpl) GetTxHeight(txId string) (uint64, error) {
	return bs.blockDB.GetTxHeight(txId)
}

// GetTxConfirmedTime returns the confirmed time of a given tx
// @Description:
// @receiver bs
// @param txId
// @return int64
// @return error
func (bs *BlockStoreImpl) GetTxConfirmedTime(txId string) (int64, error) {
	return bs.blockDB.GetTxConfirmedTime(txId)
}

// TxExists returns true if the tx exist, or returns false if none exists.
// @Description:
// @receiver bs
// @param txId
// @return bool
// @return error
func (bs *BlockStoreImpl) TxExists(txId string) (bool, error) {
	if bs.SlowLogThreshold > 0 {
		start := time.Now()
		defer func() {
			spend := time.Since(start).Milliseconds()
			if spend > bs.SlowLogThreshold {
				bs.logger.Infof("slow log: TxExists(%s) spend time %d", txId, spend)
			}
		}()
	}
	//return bs.blockDB.TxExists(txId)
	// bigfilter 未开启则直接查db
	//if bs.bigFilterDB == nil {
	//	return bs.txExistDB.TxExists(txId)
	//}
	if !bs.storeConfig.EnableBigFilter {
		return bs.txExistDB.TxExists(txId)
	}
	exists, b, err := bs.bigFilterDB.TxExists(txId)
	//如果从bigfilter查询出错，直接查db
	if err != nil {
		bs.logger.Errorf("check tx exist by txid:[%s] in bigfilter error:[%s], we will try check from db",
			txId, err)
		return bs.txExistDB.TxExists(txId)
	}
	// 在bigFilter中的cache存在
	if exists {
		return true, nil
	}
	// 在bigFilter中的cache不存在，bigFilter存储中也不存在
	if !b {
		return false, nil
	}
	// 返回假阳性，查 txExistDB
	if b {
		return bs.txExistDB.TxExists(txId)
	}

	return false, nil

}

// TxExistsInFullDB returns true and the latest committed block height in db if the tx exist,
// or returns false and math.MaxUint64 if none exists.
// @Description:
// @receiver bs
// @param txId
// @return bool
// @return uint64
// @return error
func (bs *BlockStoreImpl) TxExistsInFullDB(txId string) (bool, uint64, error) {
	//todo: 不加锁，加锁会阻塞写，影响大
	lastHeight, err := bs.stateDB.GetLastSavepoint()
	if err != nil {
		return false, 0, err
	}
	exists, err := bs.TxExists(txId)
	if err != nil {
		return false, 0, err
	}
	return exists, lastHeight, nil

}

// TxExistsInIncrementDB returns true if the tx exist from starHeight to the latest committed block,
// or returns false if none exists.
// @Description:
// @receiver bs
// @param txId
// @param startHeight
// @return bool
// @return error
func (bs *BlockStoreImpl) TxExistsInIncrementDB(txId string, startHeight uint64) (bool, error) {
	if !bs.storeConfig.EnableRWC {
		return bs.TxExists(txId)
	}
	isInner, b, err := bs.rollingWindowCache.Has(txId, startHeight)
	//在窗口内，返回结果
	if isInner {
		return b, err
	}
	//不在窗口内 直接查bs
	return bs.TxExists(txId)

}

// TxExistsInIncrementDBState returns true if the tx exist from starHeight to the latest committed block,
// or returns false if none exists.
// @Description:
// @receiver bs
// @param txId
// @param startHeight
// @return bool
// @return bool   ,true is inside the window, false is outside the window.
// @return error
func (bs *BlockStoreImpl) TxExistsInIncrementDBState(txId string, startHeight uint64) (bool, bool, error) {

	var b bool
	var err error

	if !bs.storeConfig.EnableRWC {
		b, err = bs.TxExists(txId)
		return b, false, err
	}
	isInner, b, err := bs.rollingWindowCache.Has(txId, startHeight)
	//在窗口内，返回结果
	if isInner {
		return b, true, err
	}
	//不在窗口内 直接查bs
	b, err = bs.TxExists(txId)
	return b, false, err
}

// ReadObject returns the state value for given contract name and key, or returns nil if none exists.
// @Description:
// @receiver bs
// @param contractName
// @param key
// @return []byte
// @return error
func (bs *BlockStoreImpl) ReadObject(contractName string, key []byte) ([]byte, error) {
	if bs.SlowLogThreshold > 0 {
		start := time.Now()
		defer func() {
			spend := time.Since(start).Milliseconds()
			if spend > bs.SlowLogThreshold {
				bs.logger.Infof("slow log: ReadObject(%s,%s) ,spend time %d", contractName, string(key), spend)
			}
		}()
	}
	return bs.stateDB.ReadObject(contractName, key)
}

// ReadObjects add next time
// @Description:
// @receiver bs
// @param contractName
// @param keys
// @return [][]byte
// @return error
func (bs *BlockStoreImpl) ReadObjects(contractName string, keys [][]byte) ([][]byte, error) {
	if len(keys) == 0 {
		return nil, nil
	}

	if bs.SlowLogThreshold > 0 {
		start := time.Now()
		defer func() {
			spend := time.Since(start).Milliseconds()
			if spend > bs.SlowLogThreshold {
				bs.logger.Infof("slow log: ReadObjects(%s,%s) total keys: %d ,spend time %d",
					contractName, string(keys[0]), len(keys), spend)
			}
		}()
	}
	return bs.stateDB.ReadObjects(contractName, keys)
}

// SelectObject returns an iterator that contains all the key-values between given key ranges.
// startKey is included in the results and limit is excluded.
// @Description:
// @receiver bs
// @param contractName
// @param startKey
// @param limit
// @return protocol.StateIterator
// @return error
func (bs *BlockStoreImpl) SelectObject(contractName string, startKey []byte, limit []byte) (
	protocol.StateIterator, error) {
	return bs.stateDB.SelectObject(contractName, startKey, limit)
}

// GetHistoryForKey add next time
// @Description:
// @receiver bs
// @param contractName
// @param key
// @return protocol.KeyHistoryIterator
// @return error
func (bs *BlockStoreImpl) GetHistoryForKey(contractName string, key []byte) (protocol.KeyHistoryIterator, error) {
	txs, err := bs.historyDB.GetHistoryForKey(contractName, key)
	if err != nil {
		return nil, err
	}
	return types.NewHistoryIterator(contractName, key, txs, bs.resultDB, bs.blockDB), nil
}

// GetAccountTxHistory add next time
// @Description:
// @receiver bs
// @param accountId
// @return protocol.TxHistoryIterator
// @return error
func (bs *BlockStoreImpl) GetAccountTxHistory(accountId []byte) (protocol.TxHistoryIterator, error) {
	txs, err := bs.historyDB.GetAccountTxHistory(accountId)
	if err != nil {
		return nil, err
	}
	return types.NewTxHistoryIterator(txs, bs.blockDB), nil
}

// GetContractTxHistory add next time
// @Description:
// @receiver bs
// @param contractName
// @return protocol.TxHistoryIterator
// @return error
func (bs *BlockStoreImpl) GetContractTxHistory(contractName string) (protocol.TxHistoryIterator, error) {
	txs, err := bs.historyDB.GetContractTxHistory(contractName)
	if err != nil {
		return nil, err
	}
	return types.NewTxHistoryIterator(txs, bs.blockDB), nil
}

// GetTxRWSet returns an txRWSet for given txId, or returns nil if none exists.
// @Description:
// @receiver bs
// @param txId
// @return *commonPb.TxRWSet
// @return error
func (bs *BlockStoreImpl) GetTxRWSet(txId string) (*commonPb.TxRWSet, error) {
	var (
		err        error
		isArchived bool
	)
	rwSet := &commonPb.TxRWSet{
		TxId:     "",
		TxReads:  make([]*commonPb.TxRead, 0),
		TxWrites: make([]*commonPb.TxWrite, 0),
	}

	if !bs.storeConfig.DisableResultDB {
		rws, err1 := bs.getRWSetFromFile(txId)
		if err1 != nil {
			return nil, err1
		}
		if rws != nil {
			return rws, nil
		}

		if rwSet, err = bs.resultDB.GetTxRWSet(txId); err != nil {
			return nil, err
		}
	}

	if rwSet == nil || len(rwSet.TxId) == 0 {
		if isArchived, err = bs.blockDB.TxArchived(txId); err != nil {
			return nil, err
		} else if isArchived {
			return nil, archive.ErrArchivedRWSet
		}
	}

	return rwSet, err
}

// GetTxRWSetsByHeight returns all the rwsets corresponding to the block,
// or returns nil if zhe block does not exist
// @Description:
// @receiver bs
// @param height
// @return []*commonPb.TxRWSet
// @return error
func (bs *BlockStoreImpl) GetTxRWSetsByHeight(height uint64) ([]*commonPb.TxRWSet, error) {
	if !bs.storeConfig.DisableBlockFileDb {
		//read block and rwset from file
		index, err := bs.blockDB.GetBlockIndex(height)
		if err != nil {
			return nil, err
		}
		if index == nil {
			return nil, nil
		}
		data, err := bs.blockFileDB.ReadFileSection(index)
		if err != nil {
			return nil, err
		}
		brw, err := serialization.DeserializeBlock(data)
		if err != nil {
			return nil, err
		}
		return brw.GetTxRWSets(), nil
	}
	blockStoreInfo, err := bs.blockDB.GetFilteredBlock(height)
	if err != nil || blockStoreInfo == nil {
		return nil, err
	}
	var txRWSets = make([]*commonPb.TxRWSet, len(blockStoreInfo.TxIds))
	for i, txId := range blockStoreInfo.TxIds {

		txRWSet, err := bs.GetTxRWSet(txId)
		if err != nil {
			return nil, err
		}
		if txRWSet == nil { //数据库未找到记录，这不正常，记录日志，初始化空实例
			bs.logger.Errorf("not found rwset data in database by txid=%d, please check database", txId)
			txRWSet = &commonPb.TxRWSet{}
		}
		txRWSets[i] = txRWSet
		bs.logger.Debugf("getTxRWSetsByHeight, txid:%s", txId)

	}

	return txRWSets, nil
}

// GetBlockWithRWSets returns the block and all the rwsets corresponding to the block,
// or returns nil if zhe block does not exist
// @Description:
// @receiver bs
// @param height
// @return *storePb.BlockWithRWSet
// @return error
func (bs *BlockStoreImpl) GetBlockWithRWSets(height uint64) (*storePb.BlockWithRWSet, error) {
	if !bs.storeConfig.DisableBlockFileDb {
		//read block and rwset from file
		index, err := bs.blockDB.GetBlockIndex(height)
		if err != nil {
			return nil, err
		}
		if index == nil {
			return nil, nil
		}
		data, err := bs.blockFileDB.ReadFileSection(index)
		if err != nil {
			return nil, err
		}
		return serialization.DeserializeBlock(data)
	}
	//read block and rwset from db
	block, err := bs.GetBlock(height)
	if err != nil {
		return nil, err
	} else if block == nil {
		return nil, nil
	}
	blockWithRWSets := &storePb.BlockWithRWSet{}
	blockWithRWSets.Block = block

	blockWithRWSets.TxRWSets = make([]*commonPb.TxRWSet, len(block.Txs))
	for i, tx := range block.Txs {
		txRWSet, err := bs.GetTxRWSet(tx.Payload.TxId)
		if err != nil {
			return nil, err
		}
		if txRWSet == nil { //数据库未找到记录，这不正常，记录日志，初始化空实例
			bs.logger.Errorf("not found rwset data in database by txid=%d, please check database", tx.Payload.TxId)
			txRWSet = &commonPb.TxRWSet{}
		}
		blockWithRWSets.TxRWSets[i] = txRWSet
		//}
	}

	return blockWithRWSets, nil
}

// GetDBHandle returns the database handle for  given dbName(chainId)
// @Description:
// @receiver bs
// @param dbName
// @return protocol.DBHandle
func (bs *BlockStoreImpl) GetDBHandle(dbName string) protocol.DBHandle {
	return bs.commonDB
}

// Close Close is used to close database
// @Description:
// @receiver bs
// @return error
func (bs *BlockStoreImpl) Close() error {
	bs.blockDB.Close()
	bs.stateDB.Close()
	bs.txExistDB.Close()
	if !bs.storeConfig.DisableHistoryDB && bs.historyDB != nil {
		bs.historyDB.Close()
	}
	if !bs.storeConfig.DisableContractEventDB && bs.contractEventDB != nil {
		//if parseEngineType(bs.storeConfig.ContractEventDbConfig.SqlDbConfig.SqlDbType) == types.MySQL &&
		//	bs.storeConfig.ContractEventDbConfig.Provider == localconf.DbconfigProviderSql {
		bs.contractEventDB.Close()
		//} else {
		//	return errors.New("contract event db config err")
		//}
	}
	if !bs.storeConfig.DisableResultDB && bs.resultDB != nil {
		bs.resultDB.Close()
	}
	if bs.blockFileDB != nil {
		bs.blockFileDB.Close()
	}
	bs.commonDB.Close()
	bs.logger.Debug("close all database and bin log")
	return nil
}

// recover checks savepoint and recommit lost block
// @Description:
// @receiver bs
// @return error
func (bs *BlockStoreImpl) recover() error {
	var logSavepoint, blockSavepoint, stateSavepoint, historySavepoint, resultSavepoint,
		txExistSavepoint, contractEventSavepoint, bigFilterSavepoint, rollingWindowCacheSavepoint uint64
	var err error
	if logSavepoint, err = bs.getLastFileSavepoint(); err != nil {
		return err
	}
	if blockSavepoint, err = bs.blockDB.GetLastSavepoint(); err != nil {
		return err
	}
	if stateSavepoint, err = bs.stateDB.GetLastSavepoint(); err != nil {
		return err
	}
	if txExistSavepoint, err = bs.txExistDB.GetLastSavepoint(); err != nil {
		return err
	}
	if !bs.storeConfig.DisableHistoryDB {
		if historySavepoint, err = bs.historyDB.GetLastSavepoint(); err != nil {
			return err
		}
	}
	if !bs.storeConfig.DisableResultDB {
		if resultSavepoint, err = bs.resultDB.GetLastSavepoint(); err != nil {
			return err
		}
	}
	if !bs.storeConfig.DisableContractEventDB {
		//if parseEngineType(bs.storeConfig.ContractEventDbConfig.SqlDbConfig.SqlDbType) == types.MySQL &&
		//	bs.storeConfig.ContractEventDbConfig.Provider == localconf.DbconfigProviderSql {
		if contractEventSavepoint, err = bs.contractEventDB.GetLastSavepoint(); err != nil {
			return err
		}
		//} else {
		//	return errors.New("contract event db config err")
		//}
	}
	if bs.storeConfig.EnableBigFilter {
		if bigFilterSavepoint, err = bs.bigFilterDB.GetLastSavepoint(); err != nil {
			return err
		}
	}
	rollingWindowCacheSavepoint = logSavepoint

	bs.logger.Debugf("recover checking, savepoint: fileblockDB[%d] blockDB[%d] stateDB[%d]"+
		" historyDB[%d] contractEventDB[%d] bigfilterDB[%d]",
		logSavepoint, blockSavepoint, stateSavepoint, historySavepoint, contractEventSavepoint, bigFilterSavepoint)

	return bs.recoverAllDB(logSavepoint, blockSavepoint, stateSavepoint, historySavepoint, resultSavepoint,
		txExistSavepoint, contractEventSavepoint, bigFilterSavepoint, rollingWindowCacheSavepoint)
}

//  recoverAllDB recover all DB data
//  @Description:
//  @receiver bs
//  @param logSavepoint
//  @param blockSavepoint
//  @param stateSavepoint
//  @param historySavepoint
//  @param resultSavepoint
//  @param txExistSavepoint
//  @param contractEventSavepoint
//  @param bigFilterSavepoint
//  @param rollingWindowCacheSavepoint
//  @return error
func (bs *BlockStoreImpl) recoverAllDB(logSavepoint, blockSavepoint, stateSavepoint, historySavepoint, resultSavepoint,
	txExistSavepoint, contractEventSavepoint, bigFilterSavepoint, rollingWindowCacheSavepoint uint64) error {

	bs.logger.Debugf("recoverAllDB checking, savepoint: fileblockDB[%d] blockDB[%d] stateDB[%d]"+
		" historyDB[%d] contractEventDB[%d] bigfilterDB[%d]",
		logSavepoint, blockSavepoint, stateSavepoint, historySavepoint, contractEventSavepoint, bigFilterSavepoint)

	//recommit blockdb
	if err := bs.recoverBlockDB(blockSavepoint, logSavepoint); err != nil {
		return err
	}

	//recommit statedb
	if err := bs.recoverStateDB(stateSavepoint, logSavepoint); err != nil {
		return err
	}

	//recommit rollingWindowCache 只恢复最后写入的block到cache
	if err := bs.recoverRollingWindowCache(rollingWindowCacheSavepoint); err != nil {
		return err
	}

	if !bs.storeConfig.DisableHistoryDB {
		//recommit historydb
		if err := bs.recoverHistoryDB(stateSavepoint, logSavepoint); err != nil {
			return err
		}
	}
	if !bs.storeConfig.DisableResultDB {
		//recommit resultdb
		if err := bs.recoverResultDB(resultSavepoint, logSavepoint); err != nil {
			return err
		}
	}
	if bs.storeConfig.EnableBigFilter {
		//recommit bigfilter
		if err := bs.recoverBigFilterDB(bigFilterSavepoint, logSavepoint); err != nil {
			return err
		}
	}
	//recommit contract event db
	if !bs.storeConfig.DisableContractEventDB {
		return bs.recoverContractEventDB(contractEventSavepoint, logSavepoint)
	}
	//recommit txExistDB
	return bs.recoverTxExistDB(txExistSavepoint, logSavepoint)
}

//  recoverBlockDB 恢复blockdb数据
//  @Description:
//  @receiver bs
//  @param currentHeight
//  @param savePoint
//  @return error
func (bs *BlockStoreImpl) recoverBlockDB(currentHeight uint64, savePoint uint64) error {
	height := bs.calculateRecoverHeight(currentHeight, savePoint)
	for ; height <= savePoint; height++ {
		bs.logger.Infof("[BlockDB] recommitting lost blocks, blockNum=%d, lastBlockNum=%d", height, savePoint)
		blockWithSerializedInfo, err := bs.getBlockFromLog(height)
		if err != nil {
			return err
		}
		if bs.storeConfig.BlockDbConfig.IsKVDB() {
			err = bs.blockDB.CommitBlock(blockWithSerializedInfo, true)
			if err != nil {
				return err
			}
			err = bs.blockDB.CommitBlock(blockWithSerializedInfo, false)
			if err != nil {
				return err
			}
		}
		if bs.storeConfig.BlockDbConfig.IsSqlDB() {
			err = bs.blockDB.CommitBlock(blockWithSerializedInfo, false)
			if err != nil {
				return err
			}
		}

	}
	return nil
}

//  recoverStateDB 恢复statedb数据
//  @Description:
//  @receiver bs
//  @param currentHeight
//  @param savePoint
//  @return error
func (bs *BlockStoreImpl) recoverStateDB(currentHeight uint64, savePoint uint64) error {
	height := bs.calculateRecoverHeight(currentHeight, savePoint)
	for ; height <= savePoint; height++ {
		bs.logger.Infof("[StateDB] recommitting lost blocks, blockNum=%d, lastBlockNum=%d", height, savePoint)
		blockWithSerializedInfo, err := bs.getBlockFromLog(height)
		if err != nil {
			return err
		}
		if bs.storeConfig.StateDbConfig.IsKVDB() {
			err = bs.stateDB.CommitBlock(blockWithSerializedInfo, true)
			if err != nil {
				return err
			}
			err = bs.stateDB.CommitBlock(blockWithSerializedInfo, false)
			if err != nil {
				return err
			}
		}
		if bs.storeConfig.StateDbConfig.IsSqlDB() {
			err = bs.stateDB.CommitBlock(blockWithSerializedInfo, false)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

//  recoverContractEventDB 恢复contractdb数据
//  @Description:
//  @receiver bs
//  @param currentHeight
//  @param savePoint
//  @return error
func (bs *BlockStoreImpl) recoverContractEventDB(currentHeight uint64, savePoint uint64) error {
	height := bs.calculateRecoverHeight(currentHeight, savePoint)
	for ; height <= savePoint; height++ {
		bs.logger.Infof("[ContractEventDB] recommitting lost blocks, blockNum=%d, lastBlockNum=%d", height, savePoint)
		blockWithSerializedInfo, err := bs.getBlockFromLog(height)
		if err != nil {
			return err
		}

		err = bs.contractEventDB.CommitBlock(blockWithSerializedInfo, false)
		if err != nil {
			return err
		}
	}
	return nil
}

//  recoverHistoryDB 恢复historydb数据
//  @Description:
//  @receiver bs
//  @param currentHeight
//  @param savePoint
//  @return error
func (bs *BlockStoreImpl) recoverHistoryDB(currentHeight uint64, savePoint uint64) error {
	height := bs.calculateRecoverHeight(currentHeight, savePoint)
	for ; height <= savePoint; height++ {
		bs.logger.Infof("[HistoryDB] recommitting lost blocks, blockNum=%d, lastBlockNum=%d", height, savePoint)
		blockWithSerializedInfo, err := bs.getBlockFromLog(height)
		if err != nil {
			return err
		}
		if bs.storeConfig.HistoryDbConfig.IsKVDB() {
			err = bs.historyDB.CommitBlock(blockWithSerializedInfo, true)
			if err != nil {
				return err
			}
			err = bs.historyDB.CommitBlock(blockWithSerializedInfo, false)
			if err != nil {
				return err
			}
		}
		if bs.storeConfig.HistoryDbConfig.IsSqlDB() {
			err = bs.historyDB.CommitBlock(blockWithSerializedInfo, false)
			if err != nil {
				return err
			}
		}

		// delete block from wal after recover
		err = bs.deleteBlockFromLog(height)
		if err != nil {
			bs.logger.Warnf("recover, failed to clean wal, block[%d]", height)
		}
	}
	return nil
}

//  recoverResultDB 恢复resultdb数据
//  @Description:
//  @receiver bs
//  @param currentHeight
//  @param savePoint
//  @return error
func (bs *BlockStoreImpl) recoverResultDB(currentHeight uint64, savePoint uint64) error {
	height := bs.calculateRecoverHeight(currentHeight, savePoint)
	for ; height <= savePoint; height++ {
		bs.logger.Infof("[HistoryDB] recommitting lost blocks, blockNum=%d, lastBlockNum=%d", height, savePoint)
		blockWithSerializedInfo, err := bs.getBlockFromLog(height)
		if err != nil {
			return err
		}
		if bs.storeConfig.ResultDbConfig.IsKVDB() {
			err = bs.resultDB.CommitBlock(blockWithSerializedInfo, true)
			if err != nil {
				return err
			}
			err = bs.resultDB.CommitBlock(blockWithSerializedInfo, false)
			if err != nil {
				return err
			}
		}
		if bs.storeConfig.ResultDbConfig.IsSqlDB() {
			err = bs.resultDB.CommitBlock(blockWithSerializedInfo, false)
			if err != nil {
				return err
			}
		}
		// delete block from wal after recover
		err = bs.deleteBlockFromLog(height)
		if err != nil {
			bs.logger.Warnf("recover, failed to clean wal, block[%d]", height)
		}
	}
	return nil
}

//  recoverTxExistDB 恢复txExistdb数据
//  @Description:
//  @receiver bs
//  @param currentHeight
//  @param savePoint
//  @return error
func (bs *BlockStoreImpl) recoverTxExistDB(currentHeight uint64, savePoint uint64) error {
	height := bs.calculateRecoverHeight(currentHeight, savePoint)
	for ; height <= savePoint; height++ {
		bs.logger.Infof("[TxExistDB] recommitting lost blocks, blockNum=%d, lastBlockNum=%d", height, savePoint)
		blockWithSerializedInfo, err := bs.getBlockFromLog(height)
		if err != nil {
			return err
		}

		err = bs.txExistDB.CommitBlock(blockWithSerializedInfo, false)
		if err != nil {
			return err
		}
	}
	return nil
}

//  recoverBigFilterDB 恢复bigfilter数据
//  @Description:
//  @receiver bs
//  @param currentHeight
//  @param savePoint
//  @return error
func (bs *BlockStoreImpl) recoverBigFilterDB(currentHeight uint64, savePoint uint64) error {
	height := bs.calculateRecoverHeight(currentHeight, savePoint)
	for ; height <= savePoint; height++ {
		bs.logger.Infof("[BigFilter] recommitting lost blocks, blockNum=%d, lastBlockNum=%d", height, savePoint)
		blockWithSerializedInfo, err := bs.getBlockFromLog(height)
		if err != nil {
			return err
		}

		err = bs.bigFilterDB.CommitBlock(blockWithSerializedInfo, true)
		if err != nil {
			return err
		}
		err = bs.bigFilterDB.CommitBlock(blockWithSerializedInfo, false)
		if err != nil {
			return err
		}
	}
	return nil
}

//  recoverRollingWindowCache recover RollingWindowCache ,只恢复最后写入的block到cache中
//  @Description:
//  @receiver bs
//  @param savePoint
//  @return error
func (bs *BlockStoreImpl) recoverRollingWindowCache(savePoint uint64) error {

	bs.logger.Infof("[RollingWindowCache] recommitting lost blocks, blockNum=%d, lastBlockNum=%d", savePoint, savePoint)
	blockWithSerializedInfo, err := bs.getBlockFromLog(savePoint)
	if err != nil {
		return err
	}

	//err = bs.rollingWindowCache.CommitBlock(blockWithSerializedInfo, true)
	err = bs.rollingWindowCache.ResetRWCache(blockWithSerializedInfo)
	if err != nil {
		return err
	}

	return nil
}

//  writeBlockToFile 将block写入到filedb中
//  @Description:
//  @receiver bs
//  @param blockHeight
//  @param bytes
//  @return *storePb.StoreInfo
//  @return error
func (bs *BlockStoreImpl) writeBlockToFile(blockHeight uint64, bytes []byte) (*storePb.StoreInfo, error) {
	if bs.storeConfig.DisableBlockFileDb {
		return nil, bs.walLog.Write(blockHeight+1, bytes)
	}

	// wal log, index increase from 1, while blockHeight increase form 0
	fileName, offset, bytesLen, err := bs.blockFileDB.Write(blockHeight+1, bytes)
	if err != nil {
		return nil, err
	}

	return &storePb.StoreInfo{
		FileName: fileName,
		Offset:   offset,
		ByteLen:  bytesLen,
	}, nil
}

//  getLastFileSavepoint 获得last save point
//  @Description:
//  @receiver bs
//  @return uint64
//  @return error
func (bs *BlockStoreImpl) getLastFileSavepoint() (uint64, error) {
	var (
		err       error
		lastIndex uint64
	)

	if bs.storeConfig.DisableBlockFileDb {
		lastIndex, err = bs.walLog.LastIndex()
	} else {
		lastIndex, err = bs.blockFileDB.LastIndex()
	}

	if err != nil {
		return 0, err
	}
	if lastIndex == 0 {
		return 0, nil
	}
	return lastIndex - 1, nil
}

//  getBlockFromLog 按照块高，从wal中读取 block
//  @Description:
//  @receiver bs
//  @param num
//  @return *serialization.BlockWithSerializedInfo
//  @return error
func (bs *BlockStoreImpl) getBlockFromLog(num uint64) (*serialization.BlockWithSerializedInfo, error) {
	var (
		err       error
		data      []byte
		storeInfo *storePb.StoreInfo
	)

	if bs.walLog != nil {
		data, err = bs.walLog.Read(num + 1)
		if err != nil {
			return nil, err
		}
	} else if bs.blockFileDB != nil {
		storeInfo = &storePb.StoreInfo{}
		data, storeInfo.FileName, storeInfo.Offset, storeInfo.ByteLen, err = bs.blockFileDB.ReadLastSegSection(num + 1)
	} else {
		return nil, fmt.Errorf("wal and blockfiledb should not empty both")
	}

	if err != nil {
		bs.logger.Errorf("read log failed, err:%s", err)
		return nil, err
	}
	blockWithRWSet, err := serialization.DeserializeBlock(data)
	if err != nil {
		return nil, err
	}
	buf, ok := bs.protoBufferPool.Get().(*proto.Buffer)
	if !ok {
		bs.logger.Errorf("chain[%s]: get block from log [%d] (txs:%d) when proto buffer pool get",
			blockWithRWSet.Block.Header.ChainId, num, len(blockWithRWSet.Block.Txs))
		return nil, errGetBufPool
	}
	buf.Reset()
	_, s, err := serialization.SerializeBlock(blockWithRWSet)
	if s == nil {
		return nil, err
	}
	s.Index = storeInfo
	bs.protoBufferPool.Put(buf)
	//buf, s, err := serialization.SerializeBlock(blockWithRWSet)
	//serialization.ReturnBuffer(buf)
	return s, err
}

//  deleteBlockFromLog 从wal中，按照块高 删除 block
//  @Description:
//  @receiver bs
//  @param num
//  @return error
func (bs *BlockStoreImpl) deleteBlockFromLog(num uint64) error {
	if bs.walLog == nil {
		return nil
	}

	index := num + 1
	//delete block from log every 100 block
	if (index % 100) != 0 {
		return nil
	}
	lastBlockNum := ((index - 1) / 100) * 100
	if lastBlockNum == 0 {
		return nil
	}
	return bs.walLog.TruncateFront(lastBlockNum)
}

//  deleteLastNumBlockFromLog 删除 index 前 lastN 的 log, 始终保留 部分最近写入的log
//  @Description:
//  @receiver bs
//  @param index
//  @param lastN
//  @return error
func (bs *BlockStoreImpl) deleteLastNumBlockFromLog(index uint64, lastN uint64) error {
	//删除 index 前 lastN 的 log, 始终保留 部分最近写入的log
	if bs.walLog == nil || index <= lastN {
		return nil
	}
	deleteLogIndex := index - lastN
	return bs.walLog.TruncateFront(deleteLogIndex)
}

// QuerySingle 不在事务中，直接查询状态数据库，返回一行结果
//  @Description:
//  @receiver bs
//  @param contractName
//  @param sql
//  @param values
//  @return protocol.SqlRow
//  @return error
func (bs *BlockStoreImpl) QuerySingle(contractName, sql string, values ...interface{}) (protocol.SqlRow, error) {
	return bs.stateDB.QuerySingle(contractName, sql, values...)
}

// QueryMulti 不在事务中，直接查询状态数据库，返回多行结果
//  @Description:
//  @receiver bs
//  @param contractName
//  @param sql
//  @param values
//  @return protocol.SqlRows
//  @return error
func (bs *BlockStoreImpl) QueryMulti(contractName, sql string, values ...interface{}) (protocol.SqlRows, error) {
	return bs.stateDB.QueryMulti(contractName, sql, values...)
}

// ExecDdlSql execute DDL SQL in a contract
//  @Description:
//  @receiver bs
//  @param contractName
//  @param sql
//  @param version
//  @return error
func (bs *BlockStoreImpl) ExecDdlSql(contractName, sql, version string) error {
	return bs.stateDB.ExecDdlSql(contractName, sql, version)
}

// BeginDbTransaction 启用一个事务
//  @Description:
//  @receiver bs
//  @param txName
//  @return protocol.SqlDBTransaction
//  @return error
func (bs *BlockStoreImpl) BeginDbTransaction(txName string) (protocol.SqlDBTransaction, error) {
	return bs.stateDB.BeginDbTransaction(txName)
}

// GetDbTransaction 根据事务名，获得一个已经启用的事务
//  @Description:
//  @receiver bs
//  @param txName
//  @return protocol.SqlDBTransaction
//  @return error
func (bs *BlockStoreImpl) GetDbTransaction(txName string) (protocol.SqlDBTransaction, error) {
	return bs.stateDB.GetDbTransaction(txName)

}

// CommitDbTransaction 提交一个事务
//  @Description:
//  @receiver bs
//  @param txName
//  @return error
func (bs *BlockStoreImpl) CommitDbTransaction(txName string) error {
	return bs.stateDB.CommitDbTransaction(txName)

}

// RollbackDbTransaction 回滚一个事务
//  @Description:
//  @receiver bs
//  @param txName
//  @return error
func (bs *BlockStoreImpl) RollbackDbTransaction(txName string) error {
	return bs.stateDB.RollbackDbTransaction(txName)
}

// CreateDatabase add next time
//  @Description:
//  @receiver bs
//  @param contractName
//  @return error
func (bs *BlockStoreImpl) CreateDatabase(contractName string) error {
	return bs.stateDB.CreateDatabase(contractName)
}

// DropDatabase 删除一个合约对应的数据库
//  @Description:
//  @receiver bs
//  @param contractName
//  @return error
func (bs *BlockStoreImpl) DropDatabase(contractName string) error {
	return bs.stateDB.DropDatabase(contractName)
}

// GetContractDbName 获得一个合约对应的状态数据库名
//  @Description:
//  @receiver bs
//  @param contractName
//  @return string
func (bs *BlockStoreImpl) GetContractDbName(contractName string) string {
	return bs.stateDB.GetContractDbName(contractName)
}

//  calculateRecoverHeight 计算 需要恢复的块的起始点高度
//  @Description:
//  @receiver bs
//  @param currentHeight
//  @param savePoint
//  @return uint64
func (bs *BlockStoreImpl) calculateRecoverHeight(currentHeight uint64, savePoint uint64) uint64 {
	height := currentHeight + 1
	if currentHeight == 0 && savePoint == 0 {
		//check whether has genesis block
		if bs.walLog != nil {
			if data, _ := bs.walLog.Read(1); len(data) > 0 {
				height = height - 1
			}
		} else if bs.blockFileDB != nil {
			if data, _, _, _, _ := bs.blockFileDB.ReadLastSegSection(1); len(data) > 0 {
				height = height - 1
			}
		} else {
			return 0
		}
	}

	return height
}

// InitArchiveMgr 初始化归档管理器
//  @Description:
//  @receiver bs
//  @param chainId
//  @return error
func (bs *BlockStoreImpl) InitArchiveMgr(chainId string) error {
	if bs.isSupportArchive() {
		archiveMgr, err := archive.NewArchiveMgr(chainId, bs.blockDB, bs.resultDB, bs.storeConfig, bs.logger)
		if err != nil {
			return err
		}

		bs.ArchiveMgr = archiveMgr
	}

	return nil
}

//  isSupportArchive 判断是否支持归档
//  @Description:
//  @receiver bs
//  @return bool
func (bs *BlockStoreImpl) isSupportArchive() bool {
	return bs.storeConfig.DisableBlockFileDb && bs.storeConfig.BlockDbConfig.IsKVDB() &&
		(bs.storeConfig.ResultDbConfig != nil && bs.storeConfig.ResultDbConfig.IsKVDB())
}

// GetContractByName 获得合约
//  @Description:
//  @receiver bs
//  @param name
//  @return *commonPb.Contract
//  @return error
func (bs *BlockStoreImpl) GetContractByName(name string) (*commonPb.Contract, error) {
	return utils.GetContractByName(bs.stateDB.ReadObject, name)
}

// GetContractBytecode add next time
//  @Description:
//  @receiver bs
//  @param name
//  @return []byte
//  @return error
func (bs *BlockStoreImpl) GetContractBytecode(name string) ([]byte, error) {
	return utils.GetContractBytecode(bs.stateDB.ReadObject, name)
}

// GetMemberExtraData add next time
//  @Description:
//  @receiver bs
//  @param member
//  @return *accesscontrol.MemberExtraData
//  @return error
func (bs *BlockStoreImpl) GetMemberExtraData(member *accesscontrol.Member) (*accesscontrol.MemberExtraData, error) {
	return bs.stateDB.GetMemberExtraData(member)
}

//并行序列化
//func (bs *BlockStoreImpl) serializeBlockParallel(blockWithRWSet *storePb.BlockWithRWSet, buf *proto.Buffer) ([]byte,
//	*serialization.BlockWithSerializedInfo, error) {
//
//	//buf := bs.protoBufferPool.Get().(*proto.Buffer)
//	//buf.Reset()
//	//buf := proto.NewBuffer(nil)
//
//	block := blockWithRWSet.Block
//	txRWSets := blockWithRWSet.TxRWSets
//	events := blockWithRWSet.ContractEvents
//
//	//从pool 中获得一个 序列化空对象
//	info, ok := bs.blockSerializedInfoPool.Get().(*serialization.BlockWithSerializedInfo)
//	if !ok {
//		return nil, nil, errGetObjPool
//	}
//	info.ReSet()
//
//	//info := &BlockWithSerializedInfo{}
//	info.Block = block
//	//meta := &storePb.SerializedBlock{
//	//	Header:         block.Header,
//	//	Dag:            block.Dag,
//	//	TxIds:          make([]string, 0, len(block.Txs)),
//	//	AdditionalData: block.AdditionalData,
//	//}
//	//info.Meta = meta
//	info.Meta.Header = block.Header
//	info.Meta.Dag = block.Dag
//	info.Meta.TxIds = make([]string, 0, len(block.Txs))
//	info.Meta.AdditionalData = block.AdditionalData
//
//	for _, tx := range block.Txs {
//		//meta.TxIds = append(meta.TxIds, tx.Payload.TxId)
//		info.Meta.TxIds = append(info.Meta.TxIds, tx.Payload.TxId)
//		info.Txs = append(info.Txs, tx)
//	}
//
//	info.TxRWSets = append(info.TxRWSets, txRWSets...)
//	//info.Meta = meta
//	info.ContractEvents = events
//
//	if err := info.SerializeMeta(buf); err != nil {
//		return nil, nil, err
//	}
//
//	if err := info.SerializeTxs(buf); err != nil {
//		return nil, nil, err
//	}
//
//	if err := info.SerializeTxRWSets(buf); err != nil {
//		return nil, nil, err
//	}
//
//	if err := info.SerializeEventTopicTable(buf); err != nil {
//		return nil, nil, err
//	}
//
//	//copyBytes := make([]byte, len(buf.Bytes()))
//	//copy(copyBytes, buf.Bytes())
//
//	return buf.Bytes(), info, nil
//	//return copyBytes, info, nil
//}

//  getBlockMetaFromFile add next time
//  @Description:
//  @receiver bs
//  @param height
//  @return *storePb.SerializedBlock
//  @return error
func (bs *BlockStoreImpl) getBlockMetaFromFile(height uint64) (*storePb.SerializedBlock, error) {
	if bs.blockFileDB != nil && bs.storeConfig.BlockDbConfig.IsKVDB() && !bs.storeConfig.BlockDbConfig.IsSqlDB() {
		// if block file db is enable and db is kvdb, we will get data from block file db
		index, err := bs.blockDB.GetBlockMetaIndex(height)
		if err != nil {
			return nil, err
		}
		if index == nil {
			return nil, nil
		}
		data, err := bs.blockFileDB.ReadFileSection(index)
		if err != nil {
			return nil, err
		}
		var sb storePb.SerializedBlock
		err = proto.Unmarshal(data, &sb)
		if err != nil {
			return nil, err
		}

		return &sb, nil
	}
	return nil, nil
}

//  getRWSetFromFile add next time
//  @Description:
//  @receiver bs
//  @param txId
//  @return *commonPb.TxRWSet
//  @return error
func (bs *BlockStoreImpl) getRWSetFromFile(txId string) (*commonPb.TxRWSet, error) {
	if bs.resultDB != nil && bs.blockFileDB != nil && !bs.storeConfig.BlockDbConfig.IsSqlDB() {
		index, err := bs.resultDB.GetRWSetIndex(txId)
		if err != nil {
			return nil, err
		}
		if index == nil {
			return nil, nil
		}
		data, err := bs.blockFileDB.ReadFileSection(index)
		if err != nil {
			return nil, err
		}

		var txRWSet commonPb.TxRWSet
		err = proto.Unmarshal(data, &txRWSet)
		if err != nil {
			return nil, err
		}
		return &txRWSet, nil
	}
	return nil, nil
}
