/*
 * Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
 *
 * SPDX-License-Identifier: Apache-2.0
 *
 */

package statekvdb

import (
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"sync"
	"time"

	bytesconv "chainmaker.org/chainmaker/common/v2/bytehelper"
	"chainmaker.org/chainmaker/pb-go/v2/accesscontrol"
	commonPb "chainmaker.org/chainmaker/pb-go/v2/common"
	configPb "chainmaker.org/chainmaker/pb-go/v2/config"
	storePb "chainmaker.org/chainmaker/pb-go/v2/store"
	"chainmaker.org/chainmaker/pb-go/v2/syscontract"
	"chainmaker.org/chainmaker/protocol/v2"
	"chainmaker.org/chainmaker/store/v2/cache"
	"chainmaker.org/chainmaker/store/v2/conf"
	"chainmaker.org/chainmaker/store/v2/serialization"
	"chainmaker.org/chainmaker/store/v2/types"
	"chainmaker.org/chainmaker/utils/v2"
)

const (
	contractStoreSeparator = '#'
	stateDBSavepointKey    = "stateDBSavePointKey"
	memberPrefix           = "m:"
)

// StateKvDB provider a implementation of `statedb.StateDB`
// This implementation provides a key-value based data model
// @Description:
type StateKvDB struct {
	dbHandle protocol.DBHandle
	cache    *cache.StoreCacheMgr
	logger   protocol.Logger
	//sync.Mutex
	sync.RWMutex
	bigCacheRWMutex sync.RWMutex
	storeConf       *conf.StorageConfig
}

// NewStateKvDB construct  StateKvDB
// @Description:
// @param chainId
// @param handle
// @param logger
// @param storeConfig
// @return *StateKvDB
func NewStateKvDB(chainId string, handle protocol.DBHandle, logger protocol.Logger,
	storeConfig *conf.StorageConfig) *StateKvDB {

	return &StateKvDB{
		dbHandle:        handle,
		cache:           cache.NewStoreCacheMgr(chainId, 10, logger),
		logger:          logger,
		bigCacheRWMutex: sync.RWMutex{},
		storeConf:       storeConfig,
	}
}

// InitGenesis 创世区块写入
// @Description:
// @receiver s
// @param genesisBlock
// @return error
func (s *StateKvDB) InitGenesis(genesisBlock *serialization.BlockWithSerializedInfo) error {
	s.logger.Debug("initial genesis state data into leveldb")
	err := s.CommitBlock(genesisBlock, true)
	if err != nil {
		return err
	}
	return s.CommitBlock(genesisBlock, false)
}

// CommitBlock commits the state in an atomic operation
// @Description:
// @receiver s
// @param blockWithRWSet
// @param isCache
// @return error
func (s *StateKvDB) CommitBlock(blockWithRWSet *serialization.BlockWithSerializedInfo, isCache bool) error {
	start := time.Now()
	if isCache {
		batch := types.NewUpdateBatch()
		// 1. last block height
		block := blockWithRWSet.Block
		lastBlockNumBytes := make([]byte, 8)
		binary.BigEndian.PutUint64(lastBlockNumBytes, block.Header.BlockHeight)
		batch.Put([]byte(stateDBSavepointKey), lastBlockNumBytes)

		txRWSets := blockWithRWSet.TxRWSets
		for _, txRWSet := range txRWSets {
			for _, txWrite := range txRWSet.TxWrites {
				s.operateDbByWriteSet(batch, txWrite)
			}
		}
		//process consensusArgs
		if len(block.Header.ConsensusArgs) > 0 {
			err := s.updateConsensusArgs(batch, block)
			if err != nil {
				return err
			}
		}
		// update Cache
		s.cache.AddBlock(block.Header.BlockHeight, batch)

		s.logger.Debugf("chain[%s]: commit cache block[%d] statedb, batch[%d], time used: %d",
			block.Header.ChainId, block.Header.BlockHeight, batch.Len(),
			time.Since(start).Milliseconds())
		return nil
	}

	// IsCache == false ,update StateKvDB
	batch := types.NewUpdateBatch()
	// 1. last block height
	block := blockWithRWSet.Block
	lastBlockNumBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(lastBlockNumBytes, block.Header.BlockHeight)
	batch.Put([]byte(stateDBSavepointKey), lastBlockNumBytes)

	txRWSets := blockWithRWSet.TxRWSets
	for _, txRWSet := range txRWSets {
		for _, txWrite := range txRWSet.TxWrites {
			s.operateDbByWriteSet(batch, txWrite)
		}
	}
	//process consensusArgs
	if len(block.Header.ConsensusArgs) > 0 {
		err := s.updateConsensusArgs(batch, block)
		if err != nil {
			return err
		}
	}
	batchDur := time.Since(start)
	err := s.writeBatch(block.Header.BlockHeight, batch)
	if err != nil {
		return err
	}
	writeDur := time.Since(start)
	s.logger.Debugf("chain[%s]: commit block[%d] kv statedb, time used (batch[%d]:%d, "+
		"write:%d, total:%d", block.Header.ChainId, block.Header.BlockHeight, batch.Len(),
		batchDur.Milliseconds(), (writeDur - batchDur).Milliseconds(), time.Since(start).Milliseconds())
	return nil
}

// commit k,v to bigCache
//func (s *StateKvDB) CommitBigCache(contractName string, key []byte, value []byte) {
//	s.bigCacheRWMutex.Lock()
//	defer s.bigCacheRWMutex.Unlock()
//	txWriteKey := constructStateKey(contractName, key)
//	//更新bigCache失败，不影响功能
//	if err := s.bigCache.Set(bytesconv.BytesToString(txWriteKey), value); err != nil {
//		s.logger.Infof("CommitBigCache get an error:%s", err)
//	}
//}

// updateConsensusArgs
// @Description:
// @receiver s
// @param batch
// @param block
// @return error
func (s *StateKvDB) updateConsensusArgs(batch protocol.StoreBatcher, block *commonPb.Block) error {
	//try to add consensusArgs
	consensusArgs, err := utils.GetConsensusArgsFromBlock(block)
	if err != nil {
		s.logger.Errorf("parse header.ConsensusArgs get an error:%s", err)
		return err
	}
	if consensusArgs.ConsensusData != nil {
		s.logger.Debugf("add consensusArgs ConsensusData to statedb")
		for _, write := range consensusArgs.ConsensusData.TxWrites {
			s.operateDbByWriteSet(batch, write)
		}
	}
	return nil
}

// ReadObject returns the state value for given contract name and key, or returns nil if none exists.
// @Description:
// @receiver s
// @param contractName
// @param key
// @return []byte
// @return error
func (s *StateKvDB) ReadObject(contractName string, key []byte) ([]byte, error) {
	objectKey := constructStateKey(contractName, key)

	return s.get(objectKey)
}

// ReadObjects returns the state values for given contract name and keys
func (s *StateKvDB) ReadObjects(contractName string, keys [][]byte) ([][]byte, error) {
	objectKeys := make([][]byte, len(keys))
	for i := 0; i < len(keys); i++ {
		objectKeys[i] = constructStateKey(contractName, keys[i])
	}

	return s.getKeys(objectKeys)
}

// SelectObject returns an iterator that contains all the key-values between given key ranges.
// startKey is included in the results and limit is excluded.
// @Description:
// @receiver s
// @param contractName
// @param startKey
// @param limit
// @return protocol.StateIterator
// @return error
func (s *StateKvDB) SelectObject(contractName string, startKey []byte, limit []byte) (protocol.StateIterator, error) {
	objectStartKey := constructStateKey(contractName, startKey)
	objectLimitKey := constructStateKey(contractName, limit)
	//todo combine cache and database
	//todo 消费channel 和 DirectFlushDB 并发进行，可能会导致 数据不一致，这里需要保证写的串行话，加锁
	//s.cache.LockForFlush()
	//defer s.cache.UnLockFlush()

	//1. read data from cache
	kvData, err := s.cache.KVRange(objectStartKey, objectLimitKey)
	if err != nil {
		return nil, err
	}
	//2. flush cache data to db
	s.RWMutex.Lock()
	defer s.RWMutex.Unlock()
	err = s.DirectFlushDB(kvData)
	if err != nil {
		return nil, err
	}
	//s.logger.Debugf("range query start[%s], limit[%s]", objectStartKey, objectLimitKey)
	//3. return Iterator from db
	iter, err := s.dbHandle.NewIteratorWithRange(objectStartKey, objectLimitKey)
	if err != nil {
		return nil, err
	}
	return &kvi{
		iter:         iter,
		contractName: contractName,
	}, nil
}

// DirectFlushDB  flush keyMap data direct write to db, not write cache
// @Description:
// @receiver s
// @param keyMap
// @return error
func (s *StateKvDB) DirectFlushDB(keyMap map[string][]byte) error {
	for k, v := range keyMap {
		if err := s.dbHandle.Put([]byte(k), v); err != nil {
			s.logger.Errorf("FlushDB error,", err)
			return err
		}
	}
	return nil
}

// kvi encapsulate iterator
// @Description:
type kvi struct {
	iter         protocol.Iterator
	contractName string
}

// Next 迭代器指向下一个元素
// @Description:
// @receiver i
// @return bool
func (i *kvi) Next() bool {
	return i.iter.Next()
}

// Value 获得value
// @Description:
// @receiver i
// @return *storePb.KV
// @return error
func (i *kvi) Value() (*storePb.KV, error) {
	err := i.iter.Error()
	if err != nil {
		return nil, err
	}
	return &storePb.KV{
		ContractName: i.contractName,
		Key:          parseStateKey(copyBytes(i.iter.Key()), i.contractName),
		Value:        copyBytes(i.iter.Value()),
	}, nil
}

// copyBytes 深拷贝input
// @Description:
// @param input
// @return []byte
func copyBytes(input []byte) []byte {
	out := make([]byte, len(input))
	copy(out, input)
	return out
}

// Release  释放迭代器
// @Description:
// @receiver i
func (i *kvi) Release() {
	i.iter.Release()
}

// GetLastSavepoint returns the last block height
// @Description:
// @receiver b
// @return uint64
// @return error
func (b *StateKvDB) GetLastSavepoint() (uint64, error) {
	bytes, err := b.get([]byte(stateDBSavepointKey))
	if err != nil {
		return 0, err
	} else if bytes == nil {
		return 0, nil
	}
	num := binary.BigEndian.Uint64(bytes)
	return num, nil
}

// Close is used to close database
// @Description:
// @receiver s
func (s *StateKvDB) Close() {
	s.logger.Info("close state kv db")
	s.dbHandle.Close()
	s.cache.Clear()
}

// writeBatch 将对应区块的batch写入到db中,删除缓存
// @Description:
// @receiver s
// @param blockHeight
// @param batch
// @return error
func (s *StateKvDB) writeBatch(blockHeight uint64, batch protocol.StoreBatcher) error {
	start := time.Now()
	s.RWMutex.Lock()
	//go func(ch chan struct{}) {
	batches := batch.SplitBatch(s.storeConf.WriteBatchSize)
	batchDur := time.Since(start)

	wg := &sync.WaitGroup{}
	wg.Add(len(batches))
	for i := 0; i < len(batches); i++ {
		go func(index int) {
			defer wg.Done()
			if err := s.dbHandle.WriteBatch(batches[index], false); err != nil {
				panic(fmt.Sprintf("Error writing state db: %s", err))
			}
		}(i)
	}
	wg.Wait()
	s.RWMutex.Unlock()

	//db committed, clean cache
	s.cache.DelBlock(blockHeight)

	writeDur := time.Since(start)

	s.logger.Infof("write state db, block[%d], time used: (batchSplit[%d]:%d, "+
		"write:%d, total:%d)", blockHeight, len(batches), batchDur.Milliseconds(),
		(writeDur - batchDur).Milliseconds(), time.Since(start).Milliseconds())

	//s.RWMutex.Lock()
	////go func(ch chan struct{}) {
	//err := s.dbHandle.WriteBatch(batch, false)
	//s.RWMutex.Unlock()
	//if err != nil {
	//	panic(fmt.Sprintf("Error writing leveldb: %s", err))
	//}
	////db committed, clean cache
	//s.cache.DelBlock(blockHeight)

	//}()
	return nil
}

// get 获得key 对应的value，优先从cache，没有查db
// @Description:
// @receiver s
// @param key
// @return []byte
// @return error
func (s *StateKvDB) get(key []byte) ([]byte, error) {
	//get from cache
	value, exist := s.cache.Get(string(key))
	if exist {
		return value, nil
	}
	//get from database
	return s.dbHandle.Get(key)
}

// getKeys 获得keys 对应的values,优先从cache中取，取不到查db
// @Description:
// @receiver s
// @param keys [][]byte
// @return [][]byte
// @return error
func (s *StateKvDB) getKeys(keys [][]byte) ([][]byte, error) {
	//get from cache

	if len(keys) == 0 {
		return nil, nil
	}

	values := make([][]byte, len(keys))
	nfIndexs := make([]int, 0, len(keys))
	for index, key := range keys {
		if len(key) == 0 {
			values[index] = nil
			continue
		}

		value, exist := s.cache.Get(string(key))
		if exist {
			values[index] = value
		} else {
			nfIndexs = append(nfIndexs, index)
		}
	}

	//get from database
	nfKeys := make([][]byte, len(nfIndexs))
	for i := 0; i < len(nfIndexs); i++ {
		nfKeys[i] = keys[nfIndexs[i]]
	}
	nfValues, err := s.dbHandle.GetKeys(nfKeys)
	if err != nil {
		return nil, err
	}

	for i := 0; i < len(nfIndexs); i++ {
		values[nfIndexs[i]] = nfValues[i]
	}
	return values, nil
}

//func (s *StateKvDB) has(key []byte) (bool, error) {
//	//check has from cache
//	isDelete, exist := s.cache.Has(string(key))
//	if exist {
//		return !isDelete, nil
//	}
//	return s.dbHandle.Has(key)
//}

// constructStateKey 拼接合约和对应的key，并添加分隔符
// format : {contractName}#{key}
// @Description:
// @param contractName
// @param key
// @return []byte
func constructStateKey(contractName string, key []byte) []byte {
	//return append(append([]byte(contractName), contractStoreSeparator), key...)
	return append(append(bytesconv.StringToBytes(contractName), contractStoreSeparator), key...)
}

// parseStateKey corresponding to the constructStateKey(),  delete contract name from leveldb key
// format : {contractName}#{key}
// @Description:
// @param key
// @param contractName
// @return []byte
func parseStateKey(key []byte, contractName string) []byte {
	return key[len(contractName)+1:]
}

var errorSqldbOnly = errors.New("leveldb don't support this operation, please change to sql db")

// QuerySingle not implement
// @Description:
// @receiver s
// @param contractName
// @param sql
// @param values
// @return protocol.SqlRow
// @return error
func (s *StateKvDB) QuerySingle(contractName, sql string, values ...interface{}) (protocol.SqlRow, error) {
	s.logger.Warn("call QuerySingle")
	return nil, errorSqldbOnly
}

// QueryMulti not implement
// @Description:
// @receiver s
// @param contractName
// @param sql
// @param values
// @return protocol.SqlRows
// @return error
func (s *StateKvDB) QueryMulti(contractName, sql string, values ...interface{}) (protocol.SqlRows, error) {
	s.logger.Warn("call QueryMulti")
	return nil, errorSqldbOnly
}

// BeginDbTransaction not implement
// @Description:
// @receiver s
// @param txName
// @return protocol.SqlDBTransaction
// @return error
func (s *StateKvDB) BeginDbTransaction(txName string) (protocol.SqlDBTransaction, error) {
	s.logger.Warn("call BeginDbTransaction")
	return nil, errorSqldbOnly
}

// GetDbTransaction not implement
// @Description:
// @receiver s
// @param txName
// @return protocol.SqlDBTransaction
// @return error
func (s *StateKvDB) GetDbTransaction(txName string) (protocol.SqlDBTransaction, error) {
	s.logger.Warn("call GetDbTransaction")
	return nil, errorSqldbOnly
}

// CommitDbTransaction not implement
// @Description:
// @receiver s
// @param txName
// @return error
func (s *StateKvDB) CommitDbTransaction(txName string) error {
	s.logger.Warn("call CommitDbTransaction")
	return errorSqldbOnly
}

// RollbackDbTransaction not implement
// @Description:
// @receiver s
// @param txName
// @return error
func (s *StateKvDB) RollbackDbTransaction(txName string) error {
	s.logger.Warn("call RollbackDbTransaction")
	return errorSqldbOnly

}

// ExecDdlSql not implement
// @Description:
// @receiver s
// @param contractName
// @param sql
// @param version
// @return error
func (s *StateKvDB) ExecDdlSql(contractName, sql, version string) error {
	s.logger.Warn("call ExecDdlSql")
	return errorSqldbOnly
}

// CreateDatabase not implement
// @Description:
// @receiver s
// @param contractName
// @return error
func (s *StateKvDB) CreateDatabase(contractName string) error {
	s.logger.Warn("call CreateDatabase")
	return errorSqldbOnly
}

// DropDatabase 删除一个合约对应的数据库
// @Description:
// @receiver s
// @param contractName
// @return error
func (s *StateKvDB) DropDatabase(contractName string) error {
	s.logger.Warn("call DropDatabase")
	return errorSqldbOnly
}

// GetContractDbName 获得一个合约对应的状态数据库名
// @Description:
// @receiver s
// @param contractName
// @return string
func (s *StateKvDB) GetContractDbName(contractName string) string {
	return ""
}

// operateDbByWriteSet add next time
// @Description:
// @receiver s
// @param batch
// @param txWrite
func (s *StateKvDB) operateDbByWriteSet(batch protocol.StoreBatcher, txWrite *commonPb.TxWrite) {
	// 5. state: contractID + stateKey
	txWriteKey := constructStateKey(txWrite.ContractName, txWrite.Key)
	if txWrite.Value == nil {
		batch.Delete(txWriteKey)
		//s.logger.Debugf("delete state key[%s]", txWriteKey)
	} else {
		batch.Put(txWriteKey, txWrite.Value)
		//s.logger.Debugf("put state key[%s] value[%x]", txWriteKey, txWrite.Value)
	}
}

// GetChainConfig return chain config
// @Description:
// @receiver s
// @return *configPb.ChainConfig
// @return error
func (s *StateKvDB) GetChainConfig() (*configPb.ChainConfig, error) {
	val, err := s.ReadObject(syscontract.SystemContract_CHAIN_CONFIG.String(),
		[]byte(syscontract.SystemContract_CHAIN_CONFIG.String()))
	if err != nil {
		return nil, err
	}
	conf := &configPb.ChainConfig{}
	err = conf.Unmarshal(val)
	if err != nil {
		return nil, err
	}
	return conf, nil
}

// GetMemberExtraData retrieve memberinfo from kv db
// @Description:
// @receiver s
// @param member
// @return *accesscontrol.MemberExtraData
// @return error
func (s *StateKvDB) GetMemberExtraData(member *accesscontrol.Member) (*accesscontrol.MemberExtraData, error) {
	key := append([]byte(memberPrefix), getMemberHash(member)...)
	value, err := s.get(key)
	if err != nil {
		return nil, err
	}
	mei := &accesscontrol.MemberAndExtraData{}
	err = mei.Unmarshal(value)
	if err != nil {
		return nil, err
	}
	return mei.ExtraData, nil
}

//func (s *StateKvDB) saveMemberExtraData(batch protocol.StoreBatcher, member *accesscontrol.Member,
//	extra *accesscontrol.MemberExtraData) error {
//	key := append([]byte(memberPrefix), getMemberHash(member)...)
//	mei := &accesscontrol.MemberAndExtraData{Member: member, ExtraData: extra}
//	value, err := mei.Marshal()
//	if err != nil {
//		return err
//	}
//	batch.Put(key, value)
//	return nil
//}

// getMemberHash return member hash
// @Description:
// @param member
// @return []byte
func getMemberHash(member *accesscontrol.Member) []byte {
	data, _ := member.Marshal()
	hash := sha256.Sum256(data)
	return hash[:]
}
