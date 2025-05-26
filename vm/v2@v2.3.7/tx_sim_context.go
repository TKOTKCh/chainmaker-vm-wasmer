/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package vm

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"chainmaker.org/chainmaker/common/v2/bytehelper"
	bcx509 "chainmaker.org/chainmaker/common/v2/crypto/x509"
	acPb "chainmaker.org/chainmaker/pb-go/v2/accesscontrol"
	"chainmaker.org/chainmaker/pb-go/v2/common"
	configPb "chainmaker.org/chainmaker/pb-go/v2/config"
	"chainmaker.org/chainmaker/pb-go/v2/store"
	"chainmaker.org/chainmaker/pb-go/v2/syscontract"
	vmPb "chainmaker.org/chainmaker/pb-go/v2/vm"
	"chainmaker.org/chainmaker/protocol/v2"
	"chainmaker.org/chainmaker/utils/v2"
	gasutils "chainmaker.org/chainmaker/utils/v2/gas"
)

const (
	constructKeySeparator = "#"
	keyHistoryPrefix      = "k"
	splitChar             = "#"
	splitCharPlusOne      = "$"
	maxKeyLength          = 600
	v230                  = 2300    //version 2.3.0
	v233                  = 2030300 //version 2.3.3
	v235                  = 2030500 //version 2.3.5
	v300                  = 3000000 //version 3.0.0
)

var (
	txSimContextPool = sync.Pool{
		New: func() interface{} {
			return NewTxSimContext(nil, nil, nil, 0, nil)
		},
	}
	chainConfigContractName = syscontract.SystemContract_CHAIN_CONFIG.String() //用于GetSenderAddr解析
	keyChainConfig          = chainConfigContractName                          //用于GetSenderAddr解析
)

// NewTxSimContext is used to create a new simContext for a transaction
func NewTxSimContext(vmManager protocol.VmManager, snapshot protocol.Snapshot, tx *common.Transaction,
	blockVersion uint32, logger protocol.Logger) protocol.TxSimContext {
	txRWSet := make([]map[string]*rwSet, protocol.CallContractDepth+1)
	txRWSet[0] = make(map[string]*rwSet)

	//construct  txSimContextImpl instance
	return &txSimContextImpl{
		crossInfo:                        0,
		logger:                           logger,
		txExecSeq:                        snapshot.GetSnapshotSize(),
		tx:                               tx,
		txRWSetWithDepth:                 txRWSet,
		rowCache:                         make([]map[int32]interface{}, protocol.CallContractDepth+1),
		txWriteKeySql:                    make([]*common.TxWrite, 0),
		txWriteKeyDdlSql:                 make([]*common.TxWrite, 0),
		snapshot:                         snapshot,
		vmManager:                        vmManager,
		gasUsed:                          0,
		currentDepth:                     0,
		hisResult:                        make([]*callContractResult, 0),
		blockVersion:                     blockVersion,
		usedSimContextIterator:           make([]*SimContextIterator, 0),
		usedSimContextKeyHistoryIterator: make([]*SimContextKeyHistoryIterator, 0),
		gasRemaining:                     gasutils.GetGasLimit(tx),
	}
}

// GetTxSimContext is used to get simContext from sync.Pool
func GetTxSimContext(vmManager protocol.VmManager, snapshot protocol.Snapshot, tx *common.Transaction,
	blockVersion uint32, logger protocol.Logger) protocol.TxSimContext {
	tsc, ok := txSimContextPool.Get().(*txSimContextImpl)
	if !ok {
		return nil
	}
	tsc.logger = logger
	tsc.txExecSeq = snapshot.GetSnapshotSize()
	tsc.tx = tx
	tsc.snapshot = snapshot
	tsc.vmManager = vmManager
	tsc.blockVersion = blockVersion
	tsc.gasRemaining = gasutils.GetGasLimit(tx)

	return tsc
}

// PutTxSimContext is used to put simContext to sync.Pool
func PutTxSimContext(tsc protocol.TxSimContext) {
	tscImpl, ok := tsc.(*txSimContextImpl)
	if !ok {
		return
	}
	tscImpl.crossInfo = 0
	tscImpl.logger = nil
	tscImpl.txExecSeq = 0
	tscImpl.tx = nil
	for k := range tscImpl.txRWSetWithDepth {
		for i := range tscImpl.txRWSetWithDepth[k] {
			delete(tscImpl.txRWSetWithDepth[k], i)
		}
	}
	for k := range tscImpl.rowCache {
		for i := range tscImpl.rowCache[k] {
			delete(tscImpl.rowCache[k], i)
		}
	}
	tscImpl.txWriteKeySql = tscImpl.txWriteKeySql[:0]
	tscImpl.txWriteKeyDdlSql = tscImpl.txWriteKeyDdlSql[:0]
	tscImpl.snapshot = nil
	tscImpl.vmManager = nil
	tscImpl.gasUsed = 0
	tscImpl.gasRemaining = 0
	tscImpl.currentDepth = 0
	tscImpl.hisResult = tscImpl.hisResult[:0]
	tscImpl.blockVersion = 0
	tscImpl.usedSimContextIterator = tscImpl.usedSimContextIterator[:0]
	tscImpl.usedSimContextKeyHistoryIterator = tscImpl.usedSimContextKeyHistoryIterator[:0]
	txSimContextPool.Put(tscImpl)
}

// Storage interface for smart contracts
type txSimContextImpl struct {
	// 63          59           43                   0
	// +----------+^-----------+-^---------+-^-------
	// |   4bits   |   16bits    |   .....   | 4bits|
	// +----------+^-----------+-^---------+-^-------
	// depth_count | history_flag | vec<runtime_type>
	// the length of vec is controlled by depth_count
	crossInfo                        uint64
	logger                           protocol.Logger
	txExecSeq                        int
	txResult                         *common.Result
	tx                               *common.Transaction
	txRWSetWithDepth                 []map[string]*rwSet // RWSet of a transaction
	txWriteKeySql                    []*common.TxWrite
	txWriteKeyDdlSql                 []*common.TxWrite // record ddl vm run success or failure
	snapshot                         protocol.Snapshot
	vmManager                        protocol.VmManager
	gasUsed                          uint64 // only for callContract
	currentDepth                     int
	currentResult                    []byte
	hisResult                        []*callContractResult
	rowCache                         []map[int32]interface{}
	blockVersion                     uint32
	keyIndex                         int
	usedSimContextIterator           []*SimContextIterator
	usedSimContextKeyHistoryIterator []*SimContextKeyHistoryIterator
	dbSpendTime                      int64  //合约执行过程中，访问DB花费的时间（毫秒）
	gasRemaining                     uint64 // 统一计费使用的字段
}

// call contract result
type callContractResult struct {
	contractName string
	method       string
	param        map[string][]byte
	depth        int
	gasUsed      uint64
	result       []byte
}

// Get key from cache, record this operation to read set
func (s *txSimContextImpl) Get(contractName string, key []byte) ([]byte, error) {
	start := time.Now()
	defer func() {
		s.dbSpendTime += time.Since(start).Milliseconds()
	}()

	// Get from write set
	if value, done := s.getFromWriteSet(contractName, key); done {
		if s.blockVersion < v235 {
			//从写集读取的数据放入读集，如果合约执行失败，那么会造成错误的写数据被放入了读集，所以235后，写集读取的数据不再放入读集
			//下文的代码中snapshot.GetKey也会先从写集缓存读数据，但snapshot的读写集是区块级别的，在交易级别是已经确定的,
			//这个读写集是提交区块的时候apply的,交易从snapshot中写集中读取的数据，是上一笔交易或上一个区块已经执行完确定的数据，所以没问题
			s.PutIntoReadSet(contractName, key, value)
		}
		return value, nil
	}

	// Get from read set
	if value, done := s.getFromReadSet(contractName, key); done {
		return value, nil
	}

	// Get from db
	var value []byte
	var err error
	if value, err = s.snapshot.GetKey(s.txExecSeq, contractName, key); err != nil {
		return nil, err
	}

	// if get from db success, put into read set
	s.PutIntoReadSet(contractName, key, value)
	return value, nil
}

// GetKeys  GetKeys
func (s *txSimContextImpl) GetKeys(keys []*vmPb.BatchKey) ([]*vmPb.BatchKey, error) {
	start := time.Now()
	defer func() {
		s.dbSpendTime += time.Since(start).Milliseconds()
	}()

	var (
		done              bool
		writeSetValues    []*vmPb.BatchKey
		readSetValues     []*vmPb.BatchKey
		emptyWriteSetKeys []*vmPb.BatchKey
		emptyReadSetKeys  []*vmPb.BatchKey
	)
	// batch get from write set
	if writeSetValues, emptyWriteSetKeys, done = s.getBatchFromWriteSet(keys); done {
		return writeSetValues, nil
	}

	// batch get from read set
	if readSetValues, emptyReadSetKeys, done = s.getBatchFromReadSet(emptyWriteSetKeys); done {
		return append(writeSetValues, readSetValues...), nil
	}

	// batch get from db
	value, err := s.snapshot.GetKeys(s.txExecSeq, emptyReadSetKeys)
	if err != nil {
		return nil, err
	}
	// if get from db success, put batch into read set
	s.PutBatchIntoReadSet(value)
	return append(value, append(writeSetValues, readSetValues...)...), nil
}

// GetNoRecord read data from state, but not record into read set, only used for framework
func (s *txSimContextImpl) GetNoRecord(contractName string, key []byte) ([]byte, error) {
	start := time.Now()
	defer func() {
		s.dbSpendTime += time.Since(start).Milliseconds()
	}()
	// Get from write set
	if value, done := s.getFromWriteSet(contractName, key); done {
		return value, nil
	}

	// Get from db
	var value []byte
	var err error
	if value, err = s.snapshot.GetKey(s.txExecSeq, contractName, key); err != nil {
		return nil, err
	}
	// if get from db success, put into read set
	return value, nil
}

// GetSnapshot get snapshot in TxSimContext
func (s *txSimContextImpl) GetSnapshot() protocol.Snapshot {
	return s.snapshot
}

// Put key into cache
func (s *txSimContextImpl) Put(contractName string, key []byte, value []byte) error {
	return s.putIntoWriteSet(contractName, key, value)
}

// PutRecord put sql state into cache
func (s *txSimContextImpl) PutRecord(contractName string, value []byte, sqlType protocol.SqlType) {
	txWrite := &common.TxWrite{
		Key:          []byte(s.getSqlKey()),
		Value:        value,
		ContractName: contractName,
	}
	s.txWriteKeySql = append(s.txWriteKeySql, txWrite)

	//if it is SqlTypeDbl, and it tor txWriteKeyDdlSql
	if sqlType == protocol.SqlTypeDdl {
		s.txWriteKeyDdlSql = append(s.txWriteKeyDdlSql, txWrite)
	}
}

// get sql key
func (s *txSimContextImpl) getSqlKey() string {
	s.keyIndex++
	return "#sql#" + s.tx.Payload.TxId + "#" + strconv.Itoa(s.keyIndex)
}

// Del Delete key from cache
func (s *txSimContextImpl) Del(contractName string, key []byte) error {
	return s.putIntoWriteSet(contractName, key, nil)
}

// Select range query for key [start, limit)
func (s *txSimContextImpl) Select(contractName string, startKey []byte, limit []byte) (protocol.StateIterator, error) {
	start := time.Now()
	defer func() {
		s.dbSpendTime += time.Since(start).Milliseconds()
	}()
	// 1. get wset of the block and filter wsets with startKey, limit
	// 2. get wset of current tx and filter wsets with startKey, limit
	// 3. construct an iterator for wset
	// 4. get store's iterator
	// 5. construct an iterator which includes wset iterator and store's iterator
	wsetsMap := make(map[string]interface{})
	txRWSets := s.snapshot.GetTxRWSetTable()

	//iterate every rw sets
	for _, txRWSet := range txRWSets {
		//iterate every key-value
		for _, wset := range txRWSet.TxWrites {
			//Gets the key-value in the specified range between start and limit
			if string(wset.Key) >= string(startKey) && string(wset.Key) < string(limit) {
				wsetsMap[string(wset.Key)] = &store.KV{
					Key:          wset.Key,
					Value:        wset.Value,
					ContractName: contractName,
				}
			}
		}
	}

	txWrites := s.collectWriteSetByContractName(contractName)

	//add wsetsMap
	for _, txWrite := range txWrites {
		//Gets the key-value in the specified range between start and limit
		if string(txWrite.Key) >= string(startKey) && string(txWrite.Key) < string(limit) {
			wsetsMap[string(txWrite.Key)] = &store.KV{
				Key:          txWrite.Key,
				Value:        txWrite.Value,
				ContractName: contractName,
			}
		}
	}

	wsetIterator := NewWsetIterator(wsetsMap)
	dbIterator, err := s.snapshot.GetBlockchainStore().SelectObject(contractName, startKey, limit)
	if err != nil {
		return nil, err
	}
	iter := NewSimContextIterator(s, wsetIterator, dbIterator)
	s.usedSimContextIterator = append(s.usedSimContextIterator, iter)
	return iter, nil
}

// GetHistoryIterForKey query the change history of a key in a contract
func (s *txSimContextImpl) GetHistoryIterForKey(contractName string, key []byte) (protocol.KeyHistoryIterator, error) {
	start := time.Now()
	defer func() {
		s.dbSpendTime += time.Since(start).Milliseconds()
	}()
	wSetsMap := make(map[string]interface{})
	txRWSets := s.snapshot.GetTxRWSetTable()

	blockHeight := s.snapshot.GetBlockHeight()
	scBlockHeight := blockHeight + 1
	//check every read-write set
	for _, txRwSet := range txRWSets {
		txId := txRwSet.GetTxId()
		//check every write in write set
		for index, wSet := range txRwSet.TxWrites {
			//Builds the historical modification record for the found key
			if bytes.Equal(wSet.Key, key) {
				wSetsMap[string(constructKeyWithDAGIndex(contractName, key, blockHeight,
					txId, uint64(index)))] =
					&store.KeyModification{
						TxId:        txId,
						Value:       wSet.Value,
						IsDelete:    len(wSet.Value) == 0,
						BlockHeight: blockHeight,
						Timestamp:   s.snapshot.GetBlockTimestamp(),
					}
			}
		}
	}

	txWrites := s.collectWriteSetByContractName(contractName)

	txId := s.GetTx().GetPayload().GetTxId()
	//check every write in set
	for _, txWrite := range txWrites {
		//Builds the historical modification record for the found key
		if bytes.Equal(txWrite.Key, key) {
			wSetsMap[string(constructKeyWithDAGIndex(contractName, key, scBlockHeight, txId, 0))] =
				&store.KeyModification{
					TxId:        txId,
					Value:       txWrite.Value,
					IsDelete:    len(txWrite.Value) == 0,
					BlockHeight: scBlockHeight,
					Timestamp:   0,
				}
		}
	}

	wSetIKeyHistoryIter := NewWSetKeyHistoryIterator(wSetsMap)

	dbKeyHistoryIter, err := s.snapshot.GetBlockchainStore().GetHistoryForKey(contractName, key)
	if err != nil {
		return nil, err
	}
	iter := NewSimContextKeyHistoryIterator(s, wSetIKeyHistoryIter, dbKeyHistoryIter, key)
	s.usedSimContextKeyHistoryIterator = append(s.usedSimContextKeyHistoryIterator, iter)
	return iter, nil
}

// k+ContractName+StateKey+BlockHeight+DAGIndex+TxId
func constructKeyWithDAGIndex(contractName string, key []byte, blockHeight uint64, txId string,
	dagIndex uint64) []byte {
	// blockHeight、DAGIndex转bytes表示
	dbkey := make([]byte, 0)
	dbkey = append(dbkey, []byte(keyHistoryPrefix+contractName)...)
	dbkey = appendWithSplitChar(dbkey, key, splitChar)
	blockHeightBytes, err := bytehelper.Uint64ToBytes(blockHeight)
	if err != nil {
		return nil
	}
	dbkey = appendWithSplitChar(dbkey, blockHeightBytes, splitChar)

	dagIndexBytes, err := bytehelper.Uint64ToBytes(dagIndex)
	if err != nil {
		return nil
	}
	dbkey = appendWithSplitChar(dbkey, dagIndexBytes, splitChar)
	dbkey = appendWithSplitChar(dbkey, []byte(txId), splitCharPlusOne)
	return dbkey
}

// append with sqlite char
func appendWithSplitChar(s1 []byte, s2 []byte, splitChar string) []byte {
	var buf bytes.Buffer
	buf.Write(s1)
	if len(s2) > 0 {
		buf.Write([]byte(splitChar))
		buf.Write(s2)
	}
	return buf.Bytes()
}

// GetCreator get contract creator
func (s *txSimContextImpl) GetCreator(contractName string) *acPb.Member {
	//contractName can be name or address if version >= 2300
	contract, err := s.GetContractByName(contractName)
	if err != nil {
		s.logger.Warn(err.Error())
		return nil
	}
	if contract.Creator == nil {
		return nil
	}

	//make creator member
	return &acPb.Member{
		OrgId:      contract.Creator.OrgId,
		MemberType: contract.Creator.MemberType,
		MemberInfo: contract.Creator.MemberInfo,
	}
}

// GetSender get the invoker of the transaction
func (s *txSimContextImpl) GetSender() *acPb.Member {
	return s.tx.Sender.GetSigner()
}

// PutIntoReadSet put into read set
func (s *txSimContextImpl) PutIntoReadSet(contractName string, key []byte, value []byte) {
	//if set for specified depth is null, generate one
	if s.txRWSetWithDepth[s.currentDepth] == nil {
		s.txRWSetWithDepth[s.currentDepth] = make(map[string]*rwSet)
	}

	//if set for specified contract is null, generate one
	if s.txRWSetWithDepth[s.currentDepth][contractName] == nil {
		s.txRWSetWithDepth[s.currentDepth][contractName] = &rwSet{
			txWriteKeyMap: make(map[string]*common.TxWrite),
			txReadKeyMap:  make(map[string]*common.TxRead),
		}
	}

	//just feel it
	s.txRWSetWithDepth[s.currentDepth][contractName].txReadKeyMap[constructKey(contractName, key)] = &common.TxRead{
		Key:          key,
		Value:        value,
		ContractName: contractName,
		Version:      nil,
	}
}

// PutBatchIntoReadSet put txs into read set
func (s *txSimContextImpl) PutBatchIntoReadSet(keys []*vmPb.BatchKey) {
	//put every key of BatchKey into contract
	for _, key := range keys {
		s.PutIntoReadSet(key.ContractName, []byte(key.Key), []byte(key.Field))
	}
}

// putIntoWriteSet put into write set
func (s *txSimContextImpl) putIntoWriteSet(contractName string, key []byte, value []byte) error {
	//exceed max len
	if len(key) > maxKeyLength {
		return fmt.Errorf("key length is too long, max length=%d, current key:%s", maxKeyLength, string(key))
	}

	//if specified read write set is null, make one
	if s.txRWSetWithDepth[s.currentDepth] == nil {
		s.txRWSetWithDepth[s.currentDepth] = make(map[string]*rwSet)
	}

	//if set for specified contract is null, generate one
	if s.txRWSetWithDepth[s.currentDepth][contractName] == nil {
		s.txRWSetWithDepth[s.currentDepth][contractName] = &rwSet{
			txWriteKeyMap: make(map[string]*common.TxWrite),
			txReadKeyMap:  make(map[string]*common.TxRead),
		}
	}

	//just feel it again
	s.txRWSetWithDepth[s.currentDepth][contractName].txWriteKeyMap[constructKey(contractName, key)] = &common.TxWrite{
		Key:          key,
		Value:        value,
		ContractName: contractName,
	}

	return nil
}

// getFromReadSet get value from read set
func (s *txSimContextImpl) getFromReadSet(contractName string, key []byte) ([]byte, bool) {
	//check every depth
	for depth := s.currentDepth; depth >= 0; depth-- {
		currentDepthRWSetMap := s.txRWSetWithDepth[depth]

		rwSetCache, ok := currentDepthRWSetMap[contractName]
		if !ok {
			continue
		}

		if rwSetCache == nil || rwSetCache.txReadKeyMap == nil {
			continue
		}

		txRead, exist := rwSetCache.txReadKeyMap[constructKey(contractName, key)]
		if !exist {
			continue
		}
		return txRead.Value, true
	}

	return nil, false
}

// getBatchFromReadSet  fetch txs from read set
func (s *txSimContextImpl) getBatchFromReadSet(keys []*vmPb.BatchKey) ([]*vmPb.BatchKey,
	[]*vmPb.BatchKey, bool) {
	// todo optimize logic
	txReads := make([]*vmPb.BatchKey, 0, len(keys))
	emptyTxReadsKeys := make([]*vmPb.BatchKey, 0, len(keys))

	//get all key's value in keys
	for _, key := range keys {
		//if has value, add txReads, else add emptyTxReadsKeys
		if value, ok := s.getFromReadSet(key.ContractName, protocol.GetKeyStr(key.Key, key.Field)); ok {
			key.Value = value
			txReads = append(txReads, key)
		} else {
			emptyTxReadsKeys = append(emptyTxReadsKeys, key)
		}
	}

	if len(emptyTxReadsKeys) == 0 {
		return txReads, nil, true
	}

	return txReads, emptyTxReadsKeys, false
}

// getFromWriteSet get from write set
func (s *txSimContextImpl) getFromWriteSet(contractName string, key []byte) ([]byte, bool) {
	//check every depth
	for depth := s.currentDepth; depth >= 0; depth-- {
		currentDepthRWSetMap := s.txRWSetWithDepth[depth]

		rwSetCache, ok := currentDepthRWSetMap[contractName]
		if !ok {
			continue
		}

		if rwSetCache == nil || rwSetCache.txWriteKeyMap == nil {
			continue
		}

		txWrite, exist := rwSetCache.txWriteKeyMap[constructKey(contractName, key)]
		if !exist {
			continue
		}
		return txWrite.Value, true
	}

	return nil, false
}

// getBatchFromWriteSet  getBatchFromWriteSet
func (s *txSimContextImpl) getBatchFromWriteSet(keys []*vmPb.BatchKey) ([]*vmPb.BatchKey,
	[]*vmPb.BatchKey, bool) {
	txWrites := make([]*vmPb.BatchKey, 0, len(keys))
	emptyTxWrite := make([]*vmPb.BatchKey, 0, len(keys))
	//check every key in keys
	for _, key := range keys {
		//add to txWrites if got, or add emptyTxWrite else
		if value, ok := s.getFromWriteSet(key.ContractName, protocol.GetKeyStr(key.Key, key.Field)); ok {
			key.Value = value
			txWrites = append(txWrites, key)
			if s.blockVersion < v235 {
				s.PutIntoReadSet(key.ContractName, protocol.GetKeyStr(key.Key, key.Field), key.Value)
			}
		} else {
			emptyTxWrite = append(emptyTxWrite, key)
		}
	}

	if len(emptyTxWrite) == 0 {
		return txWrites, nil, true
	}

	return txWrites, emptyTxWrite, false
}

// GetTx get the corresponding transaction
func (s *txSimContextImpl) GetTx() *common.Transaction {
	return s.tx
}

// GetBlockchainStore get blockchain storage
func (s *txSimContextImpl) GetBlockchainStore() protocol.BlockchainStore {
	return s.snapshot.GetBlockchainStore()
}

// GetLastChainConfig returns last chain config
func (s *txSimContextImpl) GetLastChainConfig() *configPb.ChainConfig {
	return s.snapshot.GetLastChainConfig()
}

// GetAccessControl get access control service
func (s *txSimContextImpl) GetAccessControl() (protocol.AccessControlProvider, error) {
	//get nil return err
	if s.vmManager.GetAccessControl() == nil {
		return nil, errors.New("access control for tx sim context is nil")
	}
	return s.vmManager.GetAccessControl(), nil
}

// GetChainNodesInfoProvider get organization service
func (s *txSimContextImpl) GetChainNodesInfoProvider() (protocol.ChainNodesInfoProvider, error) {
	//get nil return err
	if s.vmManager.GetChainNodesInfoProvider() == nil {
		return nil, errors.New("chainNodesInfoProvider for tx sim context is nil, may be running in singleton mode")
	}
	return s.vmManager.GetChainNodesInfoProvider(), nil
}

// GetTxRWSet return current transaction read write set
func (s *txSimContextImpl) GetTxRWSet(runVmSuccess bool) *common.TxRWSet {

	txRWSet := &common.TxRWSet{
		TxId:     s.tx.Payload.TxId,
		TxReads:  nil,
		TxWrites: nil,
	}

	var readKeys []string
	var reads map[string]*common.TxRead
	var writeKeys []string
	var writes map[string]*common.TxWrite

	// read set
	reads, readKeys = s.collectReadSetAndKey()
	sort.Strings(readKeys)
	//check every read key
	for _, k := range readKeys {
		txRWSet.TxReads = append(txRWSet.TxReads, reads[k])
	}

	// write set
	if runVmSuccess {
		writes, writeKeys = s.collectWriteSetAndKey()
		sort.Strings(writeKeys)
		//check every write key
		for _, k := range writeKeys {
			txRWSet.TxWrites = append(txRWSet.TxWrites, writes[k])
		}
		// sql nil key tx writes
		txRWSet.TxWrites = append(txRWSet.TxWrites, s.txWriteKeySql...)
	} else {
		// charge gas
		if s.GetBlockVersion() >= 2300 {
			writeKeys := make([]string, 0, 8)

			_, writesOfAccountManager := s.GetTxRWMapByContractName(syscontract.SystemContract_ACCOUNT_MANAGER.String())

			// contractName#key
			for key, txWrite := range writesOfAccountManager {
				if txWrite.ContractName == syscontract.SystemContract_ACCOUNT_MANAGER.String() {
					writeKeys = append(writeKeys, key)
				}
			}

			sort.Strings(writeKeys)
			//check every write key
			for _, key := range writeKeys {
				txRWSet.TxWrites = append(txRWSet.TxWrites, writesOfAccountManager[key])
			}
		}

		// ddl sql tx writes
		txRWSet.TxWrites = append(txRWSet.TxWrites, s.txWriteKeyDdlSql...)
	}

	s.releaseAllUsedIters()
	//for debug
	s.logger.DebugDynamic(func() string {
		return fmt.Sprintf("txSimContext[%s] access db spend time:%d", s.tx.Payload.TxId, s.dbSpendTime)
	})
	return txRWSet
}

// collectReadSetAndKey collect read set and key
func (s *txSimContextImpl) collectReadSetAndKey() (map[string]*common.TxRead, []string) {
	readSet := make(map[string]*common.TxRead, 8)
	readKeys := make([]string, 0, 8)
	//check every depth
	if s.GetBlockVersion() < v235 {
		//如果合并顺序是从高到低，会出现老数据覆盖新数据问题，比如A->B->C, 那么最新的C会首先合入，然后B合入，覆盖了C，A又覆盖B
		//此外，跨合约调用时，每层合约返回后已经将读写集合并到上一层，所以，理论上来讲交易提交时，只有一层，但保险起见,也需要修改
		//所以，235版本开始修复该问题，将合并顺序调整为从低到高合并
		for depth := s.currentDepth; depth >= 0; depth-- {
			//check every contract read-write set
			for _, contractRWSet := range s.txRWSetWithDepth[depth] {
				readKeys = append(readKeys, contractRWSet.collectReadKey()...)
				if s.blockVersion < blockVersion2312 {
					readSet = mergeTxReadSet(contractRWSet.txReadKeyMap, readSet)
				} else {
					mergeTxReadSet(readSet, contractRWSet.txReadKeyMap)
				}
			}
		}
	} else {
		for depth := 0; depth <= s.currentDepth; depth++ {
			for _, contractRWSet := range s.txRWSetWithDepth[depth] {
				readKeys = append(readKeys, contractRWSet.collectReadKey()...)
				mergeTxReadSet(readSet, contractRWSet.txReadKeyMap)
			}
		}

	}

	return readSet, readKeys
}

// collectWriteSetAndKey collect write set and key
func (s *txSimContextImpl) collectWriteSetAndKey() (map[string]*common.TxWrite, []string) {
	writeSet := make(map[string]*common.TxWrite, 8)
	writeKeys := make([]string, 0, 8)
	//如果合并顺序是从高到低，会出现老数据覆盖新数据问题，比如A->B->C, 那么最新的C会首先合入，然后B合入，覆盖了C，A又覆盖B
	//此外，跨合约调用时，每层合约返回后已经将读写集合并到上一层，所以，理论上来讲交易提交时，只有一层，但保险起见,也需要修改
	//所以，235版本开始修复该问题，将合并顺序调整为从低到高合并
	if s.GetBlockVersion() < v235 {
		//check every depth
		for depth := s.currentDepth; depth >= 0; depth-- {
			//check every contract read-write set
			for _, contractRWSet := range s.txRWSetWithDepth[depth] {
				writeKeys = append(writeKeys, contractRWSet.collectWriteKey()...)
				if s.blockVersion < blockVersion2312 {
					writeSet = mergeTxWriteSet(contractRWSet.txWriteKeyMap, writeSet)
				} else {
					mergeTxWriteSet(writeSet, contractRWSet.txWriteKeyMap)
				}
			}
		}
	} else {
		for depth := 0; depth <= s.currentDepth; depth++ {
			for _, contractRWSet := range s.txRWSetWithDepth[depth] {
				writeKeys = append(writeKeys, contractRWSet.collectWriteKey()...)
				mergeTxWriteSet(writeSet, contractRWSet.txWriteKeyMap)
			}
		}

	}

	return writeSet, writeKeys
}

func (s *txSimContextImpl) collectWriteSetByContractName(contractName string) map[string]*common.TxWrite {
	txWrites := make(map[string]*common.TxWrite)

	//如果合并顺序是从高到低，会出现老数据覆盖新数据问题，比如A->B->C, 那么最新的C会首先合入，然后B合入，覆盖了C，A又覆盖B
	//所以，235开始修复该问题，将顺序调整为从低到高合并
	if s.GetBlockVersion() < v235 {
		for current := s.currentDepth; current >= 0; current-- {
			//Merge read and write sets in terms of contract
			if txRWSetWithContract, ok := s.txRWSetWithDepth[current][contractName]; ok {
				txWrites = mergeTxWriteSet(txRWSetWithContract.txWriteKeyMap, txWrites)
			}
		}
	} else {
		for i := 0; i <= s.currentDepth; i++ {
			if txRWSetWithContract, ok := s.txRWSetWithDepth[i][contractName]; ok {
				txWrites = mergeTxWriteSet(txWrites, txRWSetWithContract.txWriteKeyMap)
			}
		}

	}

	return txWrites
}

// GetTxRWMapByContractName get tx read write map by contract name
func (s *txSimContextImpl) GetTxRWMapByContractName(contractName string) (
	map[string]*common.TxRead, map[string]*common.TxWrite) {
	txReads := make(map[string]*common.TxRead)
	txWrites := make(map[string]*common.TxWrite)

	//check every depth
	for current := s.currentDepth; current >= 0; current-- {
		//check every contract read-write set
		if txRWSetWithContract, ok := s.txRWSetWithDepth[current][contractName]; ok {
			txReads = mergeTxReadSet(txRWSetWithContract.txReadKeyMap, txReads)
			txWrites = mergeTxWriteSet(txRWSetWithContract.txWriteKeyMap, txWrites)
		}
	}

	return txReads, txWrites
}

// releaseAllUsedIters At the end of the simulation, close all open iterators
func (s *txSimContextImpl) releaseAllUsedIters() {
	//release all used simContext iter
	for _, iter := range s.usedSimContextIterator {
		iter.Release()
	}
	//release all history iter
	for _, iter := range s.usedSimContextKeyHistoryIterator {
		iter.Release()
	}
}

// GetBlockHeight returns current block height
func (s *txSimContextImpl) GetBlockHeight() uint64 {
	return s.snapshot.GetBlockHeight()
}

// GetBlockFingerprint returns block fingerprint
func (s *txSimContextImpl) GetBlockFingerprint() string {
	return s.snapshot.GetBlockFingerprint()
}

// GetBlockTimestamp returns current block timestamp
func (s *txSimContextImpl) GetBlockTimestamp() int64 {
	return s.snapshot.GetBlockTimestamp()
}

// GetBlockProposer get block proposer
func (s *txSimContextImpl) GetBlockProposer() *acPb.Member {
	return s.snapshot.GetBlockProposer()
}

// GetTxExecSeq obtain the corresponding transaction execution sequence
func (s *txSimContextImpl) GetTxExecSeq() int {
	return s.txExecSeq
}

// SetTxExecSeq set the corresponding transaction execution sequence
func (s *txSimContextImpl) SetTxExecSeq(txExecSeq int) {
	s.txExecSeq = txExecSeq
}

// GetTxResult get the tx result
func (s *txSimContextImpl) GetTxResult() *common.Result {
	return s.txResult
}

// Set the tx result
func (s *txSimContextImpl) SetTxResult(txResult *common.Result) {
	s.txResult = txResult
}

// CallContract Cross contract call before block version 231
func (s *txSimContextImpl) CallContract(caller, contract *common.Contract, method string, byteCode []byte,
	parameter map[string][]byte, gasUsed uint64, refTxType common.TxType) (
	*common.ContractResult, protocol.ExecOrderTxType, common.TxStatusCode) {
	s.gasUsed = gasUsed
	s.currentDepth = s.currentDepth + 1
	defer func() {
		s.currentDepth--
	}()

	// exceed max depth, return err
	if s.currentDepth > protocol.CallContractDepth {
		contractResult := &common.ContractResult{
			Code:    uint32(1),
			Result:  nil,
			Message: fmt.Sprintf("CallContract too depth %d", s.currentDepth),
		}
		return contractResult, protocol.ExecOrderTxTypeNormal, common.TxStatusCode_CONTRACT_TOO_DEEP_FAILED
	}
	s.txRWSetWithDepth[s.currentDepth] = make(map[string]*rwSet)

	//exceed gas limit, return err
	if s.gasUsed > protocol.GasLimit {
		contractResult := &common.ContractResult{
			Code:   uint32(1),
			Result: nil,
			Message: fmt.Sprintf("There is not enough gas, gasUsed %d GasLimit %d ",
				gasUsed, int64(protocol.GasLimit)),
		}
		return contractResult, protocol.ExecOrderTxTypeNormal, common.TxStatusCode_CONTRACT_FAIL
	}

	//if byte code null, get it
	if len(byteCode) == 0 && contract.RuntimeType != common.RuntimeType_DOCKER_GO &&
		contract.RuntimeType != common.RuntimeType_GO {
		dbByteCode, err := s.GetContractBytecode(contract.Name)
		if err != nil {
			return nil, protocol.ExecOrderTxTypeNormal, common.TxStatusCode_CONTRACT_FAIL
		}
		byteCode = dbByteCode
	}

	if s.blockVersion >= blockVersion2300 && caller != nil && caller.Address != "" {
		if caller.RuntimeType == common.RuntimeType_GO || s.blockVersion >= 2030100 {
			parameter[protocol.ContractCrossCallerParam] = []byte(caller.Address)
		}
	}

	if err := s.verifyCallContract(contract, method, s.blockVersion); err != nil {
		resultMsg := fmt.Sprintf("the tx has no right to call contract `%v:%v`", contract.Name, method)
		s.logger.Warnf("tx[%v] call contract failed: %s", s.tx.Payload.TxId, err)
		result := callContractResult{
			depth:        s.currentDepth,
			gasUsed:      s.gasUsed,
			result:       []byte(resultMsg),
			contractName: contract.Name,
			method:       method,
			param:        parameter,
		}
		s.hisResult = append(s.hisResult, &result)
		s.currentResult = []byte(resultMsg)

		return &common.ContractResult{
			Code:    1,
			Message: resultMsg,
		}, protocol.ExecOrderTxTypeNormal, common.TxStatusCode_CONTRACT_FAIL
	}

	// call `vmManager` to run contract
	r, specialTxType, code := s.vmManager.RunContract(contract, method, byteCode, parameter, s, gasUsed, refTxType)

	// set result of the call contract
	result := callContractResult{
		depth:        s.currentDepth,
		gasUsed:      s.gasUsed,
		result:       r.Result,
		contractName: contract.Name,
		method:       method,
		param:        parameter,
	}
	s.hisResult = append(s.hisResult, &result)
	s.currentResult = r.Result

	if r.Code != 0 {
		if s.blockVersion < 2300 {
			s.commitRWSetToPreDepth()
			return r, specialTxType, code
		}

		s.commitRSetAndRollbackWSet()
		return r, specialTxType, code
	}

	s.commitRWSetToPreDepth()
	return r, specialTxType, code
}

// commitRWSetToPreDepth after the cross contract call ends, the read-write set is submitted to the previous layer
func (s *txSimContextImpl) commitRWSetToPreDepth() {
	currentDepth := s.currentDepth
	preDepth := currentDepth - 1

	//check every depth
	for contractName, contractRWSet := range s.txRWSetWithDepth[currentDepth] {
		preDepthRWSet := s.txRWSetWithDepth[preDepth]
		if preDepthRWSet == nil {
			return
		}

		preContractRWSet, ok := preDepthRWSet[contractName]
		if !ok {
			s.txRWSetWithDepth[preDepth][contractName] = contractRWSet
			delete(s.txRWSetWithDepth[currentDepth], contractName)
			continue
		}
		s.txRWSetWithDepth[preDepth][contractName].txReadKeyMap =
			mergeTxReadSet(preContractRWSet.txReadKeyMap, contractRWSet.txReadKeyMap)
		s.txRWSetWithDepth[preDepth][contractName].txWriteKeyMap =
			mergeTxWriteSet(preContractRWSet.txWriteKeyMap, contractRWSet.txWriteKeyMap)
		delete(s.txRWSetWithDepth[currentDepth], contractName)
	}
}

// commitRSetAndRollbackWSet rollback the write set and commit the read set to pre depth
func (s *txSimContextImpl) commitRSetAndRollbackWSet() {
	currentDepth := s.currentDepth
	preDepth := currentDepth - 1

	//check every depth
	for contractName, contractRWSet := range s.txRWSetWithDepth[currentDepth] {
		preDepthRWSet := s.txRWSetWithDepth[preDepth]
		if preDepthRWSet == nil {
			return
		}

		preContractRWSet, ok := preDepthRWSet[contractName]
		if !ok {
			s.txRWSetWithDepth[preDepth][contractName] = contractRWSet
			delete(s.txRWSetWithDepth[currentDepth], contractName)
			continue
		}

		s.txRWSetWithDepth[preDepth][contractName].txReadKeyMap =
			mergeTxReadSet(preContractRWSet.txReadKeyMap, contractRWSet.txReadKeyMap)
		delete(s.txRWSetWithDepth[currentDepth], contractName)
	}
}

// GetCurrentResult obtain the execution result of current contract (cross contract)
func (s *txSimContextImpl) GetCurrentResult() []byte {
	return s.currentResult
}

// GetDepth get contract call depth
func (s *txSimContextImpl) GetDepth() int {
	return s.currentDepth
}

// constructKey contract key
func constructKey(contractName string, key []byte) string {
	sb := new(strings.Builder)
	sb.WriteString(contractName)
	sb.WriteString(constructKeySeparator)
	sb.Write(key)
	return sb.String()
}

// SetIterHandle cache iterator
func (s *txSimContextImpl) SetIterHandle(index int32, rows interface{}) {
	//if rowCache of current depth is null, make one
	if s.rowCache[s.currentDepth] == nil {
		s.rowCache[s.currentDepth] = make(map[int32]interface{})
	}

	s.rowCache[s.currentDepth][index] = rows
}

// GetIterHandle get iterator by ID
func (s *txSimContextImpl) GetIterHandle(index int32) (interface{}, bool) {
	//if rowCache of current depth is null, make one
	if s.rowCache[s.currentDepth] == nil {
		s.rowCache[s.currentDepth] = make(map[int32]interface{})
	}

	data, ok := s.rowCache[s.currentDepth][index]
	return data, ok
}

// GetBlockVersion get block version
func (s *txSimContextImpl) GetBlockVersion() uint32 {
	return s.blockVersion
}

// GetContractByName used to obtain contracts by name
func (s *txSimContextImpl) GetContractByName(name string) (*common.Contract, error) {
	start := time.Now()
	defer func() {
		//for time spend
		s.dbSpendTime += time.Since(start).Milliseconds()
	}()

	//for addr type
	cfg := s.GetLastChainConfig()

	//truncate zx prefix
	if cfg.Vm.AddrType == configPb.AddrType_ZXL && s.blockVersion >= v230 && utils.CheckZxlAddrFormat(name) {
		name = name[2:]
	}

	//Get Contract from write set
	if s.blockVersion >= v233 && s.blockVersion != v300 {
		nameKey := utils.GetContractDbKey(name)
		contractBytes, has := s.getFromWriteSet(syscontract.SystemContract_CONTRACT_MANAGE.String(), nameKey)
		if has {
			contract := &common.Contract{}
			err := contract.Unmarshal(contractBytes)
			if err != nil {
				return nil, err
			}
			return contract, nil
		}
	}

	//Get Contract from store
	return s.snapshot.GetBlockchainStore().GetContractByName(name)
}

// GetContractBytecode get contract bytecode
func (s *txSimContextImpl) GetContractBytecode(name string) ([]byte, error) {
	start := time.Now()
	defer func() {
		//for time spend
		s.dbSpendTime += time.Since(start).Milliseconds()
	}()

	//Get Contract bytecode from write set
	if s.blockVersion >= v233 && s.blockVersion != v300 {
		nameKey := utils.GetContractByteCodeDbKey(name)
		bytecode, has := s.getFromWriteSet(syscontract.SystemContract_CONTRACT_MANAGE.String(), nameKey)
		if has {
			return bytecode, nil
		}
	}

	return s.snapshot.GetBlockchainStore().GetContractBytecode(name)
}

// GetCrossInfo get contract call link information
func (s *txSimContextImpl) GetCrossInfo() uint64 {
	return s.crossInfo
}

// HasUsed judge whether the specified common.RuntimeType has appeared in the previous depth
// in the current cross-link
func (s *txSimContextImpl) HasUsed(runtimeType common.RuntimeType) bool {
	callContractCtx := NewCallContractContext(s.crossInfo)
	return callContractCtx.HasUsed(runtimeType)
}

// RecordRuntimeTypeIntoCrossInfo record the new contract call information to the top of crossInfo
func (s *txSimContextImpl) RecordRuntimeTypeIntoCrossInfo(runtimeType common.RuntimeType) {
	callContractCtx := NewCallContractContext(s.crossInfo)
	callContractCtx.AddLayer(runtimeType)
	s.crossInfo = callContractCtx.ctxBitmap
}

// RemoveRuntimeTypeFromCrossInfo remove the top-level information from the crossInfo
func (s *txSimContextImpl) RemoveRuntimeTypeFromCrossInfo() {
	callContractCtx := NewCallContractContext(s.crossInfo)
	callContractCtx.ClearLatestLayer()
	s.crossInfo = callContractCtx.ctxBitmap
}

// GetStrAddrFromPbMember get string address from pb member
func (s *txSimContextImpl) GetStrAddrFromPbMember(pbMember *acPb.Member) (string, error) {

	var err error
	var pkBytes []byte

	ac, err := s.GetAccessControl()
	if err != nil {
		return "", err
	}

	switch pbMember.MemberType {
	case acPb.MemberType_CERT:
		var certificate *bcx509.Certificate
		var pubKeyStr string
		certificate, err = utils.ParseCert(pbMember.MemberInfo)
		if err != nil {
			return "", err
		}
		pubKeyStr, err = certificate.PublicKey.String()
		if err != nil {
			return "", err
		}
		pkBytes = []byte(pubKeyStr)

	case acPb.MemberType_CERT_HASH,
		acPb.MemberType_ALIAS:
		var pubKeyStr string
		certHashKey := hex.EncodeToString(pbMember.MemberInfo)
		var certBytes []byte
		certBytes, err = s.GetNoRecord(syscontract.SystemContract_CERT_MANAGE.String(), []byte(certHashKey))
		if err != nil {
			s.logger.Errorf("get cert from chain failed, %s", err.Error())
			return "", err
		}
		var cert *bcx509.Certificate
		cert, err = utils.ParseCert(certBytes)
		if err != nil {
			return "", err
		}
		pubKeyStr, err = cert.PublicKey.String()
		if err != nil {
			return "", err
		}
		pkBytes = []byte(pubKeyStr)

	case acPb.MemberType_PUBLIC_KEY:
		pkBytes = pbMember.MemberInfo
	case acPb.MemberType_ADDR:
		return string(pbMember.MemberInfo), nil

	default:
		s.logger.Errorf("getSenderAddress failed, invalid member type")
		return "", err
	}

	addr, _, err := ac.GetAddressFromCache(pkBytes)
	if err != nil {
		return "", err
	}
	return addr, nil
}

func (s *txSimContextImpl) SubtractGas(gasUsed uint64) error {
	oldGas := s.gasRemaining
	if int64(oldGas) < 0 {
		return fmt.Errorf("tx[%s] gas is not enough", s.tx.Payload.TxId)
	}
	s.gasRemaining -= gasUsed
	if s.gasRemaining > oldGas {
		return fmt.Errorf("tx[%s] gas is not enough", s.tx.Payload.TxId)
	}
	return nil
}

func (s *txSimContextImpl) GetGasRemaining() uint64 {
	return s.gasRemaining
}

func (s *txSimContextImpl) verifyCallContract(contract *common.Contract, method string, blockVersion uint32) error {

	if blockVersion < blockVersion2310 {
		return nil
	}

	s.logger.Debugf("origin contract name = %v", s.tx.Payload.ContractName)
	s.logger.Debugf("origin method = %v", s.tx.Payload.Method)
	if s.tx.Payload.ContractName == syscontract.SystemContract_MULTI_SIGN.String() {
		if s.tx.Payload.Method == syscontract.MultiSignFunction_VOTE.String() ||
			s.tx.Payload.Method == syscontract.MultiSignFunction_TRIG.String() {
			s.logger.Debugf("bypass the cross call contract when multisign VOTE | REIG.")
			return nil
		}
	}

	return s.verifyPolicy(contract, method, blockVersion)
}

func (s *txSimContextImpl) verifyPolicy(contract *common.Contract, method string, blockVersion uint32) error {

	var err error

	ac, err := s.GetAccessControl()
	if err != nil {
		return err
	}

	resourceName := contract.Name + "-" + method
	allowed, err := ac.VerifyTxPrincipal(s.tx, resourceName, blockVersion)
	if err != nil {
		return err
	}

	if !allowed {
		return fmt.Errorf("authentication failed")
	}

	return nil
}
