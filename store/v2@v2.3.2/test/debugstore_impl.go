/*
 * Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
 *
 * SPDX-License-Identifier: Apache-2.0
 *
 */

package test

import (
	"errors"
	"fmt"
	"time"

	acPb "chainmaker.org/chainmaker/pb-go/v2/accesscontrol"
	commonPb "chainmaker.org/chainmaker/pb-go/v2/common"
	configPb "chainmaker.org/chainmaker/pb-go/v2/config"
	storePb "chainmaker.org/chainmaker/pb-go/v2/store"
	"chainmaker.org/chainmaker/protocol/v2"
	"chainmaker.org/chainmaker/store/v2/conf"
)

var (
	errNotFound = errors.New("not found")
)

// DebugStore 本实现是用于避开存储模块的内存、CPU和IO的影响的情况下，提供最模拟的基本区块写入，读取等。不能安装合约，写入状态。
// @Description:
type DebugStore struct {
	logger      protocol.Logger
	storeConfig *conf.StorageConfig
	dbHandle    protocol.DBHandle
}

// GetTxWithInfo 根据txid得到 交易info
// @Description:
// @receiver d
// @param txId
// @return *commonPb.TransactionInfo
// @return error
func (d *DebugStore) GetTxWithInfo(txId string) (*commonPb.TransactionInfo, error) {
	//TODO implement me
	panic("implement me")
}

// GetTxInfoOnly 根据txid 得到交易info
// @Description:
// @receiver d
// @param txId
// @return *commonPb.TransactionInfo
// @return error
func (d *DebugStore) GetTxInfoOnly(txId string) (*commonPb.TransactionInfo, error) {
	//TODO implement me
	panic("implement me")
}

// GetTxWithRWSet return tx and it's rw set
// @Description:
// @receiver d
// @param txId
// @return *commonPb.TransactionWithRWSet
// @return error
func (d *DebugStore) GetTxWithRWSet(txId string) (*commonPb.TransactionWithRWSet, error) {
	panic("implement me")
}

// GetTxInfoWithRWSet return tx and tx info and rw set
// @Description:
// @receiver d
// @param txId
// @return *commonPb.TransactionInfoWithRWSet
// @return error
func (d *DebugStore) GetTxInfoWithRWSet(txId string) (*commonPb.TransactionInfoWithRWSet, error) {
	panic("implement me")
}

// NewDebugStore add next time
// @Description:
// @param l
// @param config
// @param db
// @return *DebugStore
func NewDebugStore(l protocol.Logger, config *conf.StorageConfig, db protocol.DBHandle) *DebugStore {
	return &DebugStore{
		logger:      l,
		storeConfig: config,
		dbHandle:    db,
	}
}

// QuerySingle 根据合约，sql,value得到对应行数据
// @Description:
// @receiver d
// @param contractName
// @param sql
// @param values
// @return protocol.SqlRow
// @return error
func (d *DebugStore) QuerySingle(contractName, sql string, values ...interface{}) (protocol.SqlRow, error) {
	d.logger.Panic("implement me")
	panic("implement me")
}

// QueryMulti 根据合约，sql,value得到对应 多行数据
// @Description:
// @receiver d
// @param contractName
// @param sql
// @param values
// @return protocol.SqlRows
// @return error
func (d *DebugStore) QueryMulti(contractName, sql string, values ...interface{}) (protocol.SqlRows, error) {
	d.logger.Panic("not implement")
	panic("implement me")
}

// ExecDdlSql 执行合约sql
// @Description:
// @receiver d
// @param contractName
// @param sql
// @param version
// @return error
func (d *DebugStore) ExecDdlSql(contractName, sql, version string) error {
	d.logger.Panic("not implement")
	panic("implement me")
}

// BeginDbTransaction 开启事务
// @Description:
// @receiver d
// @param txName
// @return protocol.SqlDBTransaction
// @return error
func (d *DebugStore) BeginDbTransaction(txName string) (protocol.SqlDBTransaction, error) {
	d.logger.Panic("not implement")
	panic("implement me")
}

// GetDbTransaction 得到当前事务
// @Description:
// @receiver d
// @param txName
// @return protocol.SqlDBTransaction
// @return error
func (d *DebugStore) GetDbTransaction(txName string) (protocol.SqlDBTransaction, error) {
	d.logger.Panic("not implement")
	panic("implement me")
}

// CommitDbTransaction 提交事务
// @Description:
// @receiver d
// @param txName
// @return error
func (d *DebugStore) CommitDbTransaction(txName string) error {
	d.logger.Panic("not implement")
	panic("implement me")
}

// RollbackDbTransaction 回滚事务
// @Description:
// @receiver d
// @param txName
// @return error
func (d *DebugStore) RollbackDbTransaction(txName string) error {
	d.logger.Panic("not implement")
	panic("implement me")
}

// CreateDatabase 创建database
// @Description:
// @receiver d
// @param contractName
// @return error
func (d *DebugStore) CreateDatabase(contractName string) error {
	d.logger.Panic("not implement")
	panic("implement me")
}

// DropDatabase 删除database
// @Description:
// @receiver d
// @param contractName
// @return error
func (d *DebugStore) DropDatabase(contractName string) error {
	d.logger.Panic("not implement")
	panic("implement me")
}

// GetContractDbName 得到合约对应database
// @Description:
// @receiver d
// @param contractName
// @return string
func (d *DebugStore) GetContractDbName(contractName string) string {
	d.logger.Panic("not implement")
	panic("implement me")
}

// TxExistsInFullDB 判断交易是否存在
// @Description:
// @receiver d
// @param txId
// @return bool
// @return uint64
// @return error
func (d *DebugStore) TxExistsInFullDB(txId string) (bool, uint64, error) {
	d.logger.Panic("not implement")
	panic("implement me")
}

// TxExistsInIncrementDB 增量防重
// @Description:
// @receiver d
// @param txId
// @param startHeight
// @return bool
// @return error
func (d *DebugStore) TxExistsInIncrementDB(txId string, startHeight uint64) (bool, error) {
	d.logger.Panic("not implement")
	panic("implement me")
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
func (d *DebugStore) TxExistsInIncrementDBState(txId string, startHeight uint64) (bool, bool, error) {
	d.logger.Panic("not implement")
	panic("implement me")
}

// GetContractByName 得到合约
// @Description:
// @receiver d
// @param name
// @return *commonPb.Contract
// @return error
func (d *DebugStore) GetContractByName(name string) (*commonPb.Contract, error) {
	return nil, errNotFound
}

// GetContractBytecode 得到合约字节码
// @Description:
// @receiver d
// @param name
// @return []byte
// @return error
func (d *DebugStore) GetContractBytecode(name string) ([]byte, error) {
	return nil, errNotFound
}

// GetMemberExtraData 得到member
// @Description:
// @receiver d
// @param member
// @return *acPb.MemberExtraData
// @return error
func (d *DebugStore) GetMemberExtraData(member *acPb.Member) (*acPb.MemberExtraData, error) {
	d.logger.Panic("not implement")
	panic("implement me")
}

// InitGenesis 创世块初始化写入
// @Description:
// @receiver d
// @param genesisBlock
// @return error
func (d *DebugStore) InitGenesis(genesisBlock *storePb.BlockWithRWSet) error {
	//put block header
	block := genesisBlock.Block
	header, _ := block.Header.Marshal()
	err := d.dbHandle.Put(append([]byte("hash_"), block.Header.BlockHash...), header)
	if err != nil {
		return err
	}
	err = d.dbHandle.Put([]byte("lastBlockHeight"), header)
	if err != nil {
		return err
	}
	err = d.dbHandle.Put([]byte(fmt.Sprintf("height_%d", block.Header.BlockHeight)), header)
	if err != nil {
		return err
	}
	for _, rw := range genesisBlock.TxRWSets {
		for _, w := range rw.TxWrites {
			err = d.dbHandle.Put(append([]byte(w.ContractName+"#"), w.Key...), w.Value)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// PutBlock 写区块和读写集
// @Description:
// @receiver d
// @param block
// @param txRWSets
// @return error
func (d *DebugStore) PutBlock(block *commonPb.Block, txRWSets []*commonPb.TxRWSet) error {
	d.logger.Infof("mock put block[%d] tx count:%d", block.Header.BlockHeight, len(block.Txs))
	startTime := time.Now()
	defer func() {
		d.logger.Infof("put block[%d] cost time:%s", block.Header.BlockHeight, time.Since(startTime))
	}()
	//put block header
	header, _ := block.Header.Marshal()
	err := d.dbHandle.Put(append([]byte("hash_"), block.Header.BlockHash...), header)
	if err != nil {
		return err
	}
	err = d.dbHandle.Put([]byte("lastBlockHeight"), header)
	if err != nil {
		return err
	}
	return d.dbHandle.Put([]byte(fmt.Sprintf("height_%d", block.Header.BlockHeight)), header)
}

// GetBlockByHash 根据hash得到block
// @Description:
// @receiver d
// @param blockHash
// @return *commonPb.Block
// @return error
func (d *DebugStore) GetBlockByHash(blockHash []byte) (*commonPb.Block, error) {
	data, err := d.dbHandle.Get(append([]byte("hash_"), blockHash...))
	if err != nil {
		return nil, err
	}
	header := &commonPb.BlockHeader{}
	err = header.Unmarshal(data)
	if err != nil {
		return nil, err
	}
	return &commonPb.Block{Header: header}, nil
}

// BlockExists 根据hash 判断区块是否存在
// @Description:
// @receiver d
// @param blockHash
// @return bool
// @return error
func (d *DebugStore) BlockExists(blockHash []byte) (bool, error) {
	d.logger.Panic("not implement")
	panic("implement me")
}

// GetHeightByHash 根据hash得到块高
// @Description:
// @receiver d
// @param blockHash
// @return uint64
// @return error
func (d *DebugStore) GetHeightByHash(blockHash []byte) (uint64, error) {
	d.logger.Panic("not implement")
	panic("implement me")
}

// GetBlockHeaderByHeight 根据块高得到区块header
// @Description:
// @receiver d
// @param height
// @return *commonPb.BlockHeader
// @return error
func (d *DebugStore) GetBlockHeaderByHeight(height uint64) (*commonPb.BlockHeader, error) {
	d.logger.Panic("not implement")
	panic("implement me")
}

// GetBlock 根据区块块高，得到区块
// @Description:
// @receiver d
// @param height
// @return *commonPb.Block
// @return error
func (d *DebugStore) GetBlock(height uint64) (*commonPb.Block, error) {
	data, err := d.dbHandle.Get([]byte(fmt.Sprintf("height_%d", height)))
	if err != nil {
		return nil, err
	}
	header := &commonPb.BlockHeader{}
	err = header.Unmarshal(data)
	if err != nil {
		return nil, err
	}
	return &commonPb.Block{Header: header}, nil
}

// GetLastConfigBlock 得到最后的配置区块
// @Description:
// @receiver d
// @return *commonPb.Block
// @return error
func (d *DebugStore) GetLastConfigBlock() (*commonPb.Block, error) {
	return d.GetBlock(0)
}

// GetLastChainConfig 得到链配置
// @Description:
// @receiver d
// @return *configPb.ChainConfig
// @return error
func (d *DebugStore) GetLastChainConfig() (*configPb.ChainConfig, error) {
	d.logger.Panic("not implement")
	panic("implement me")
}

// GetBlockByTx 通过txid得到区块
// @Description:
// @receiver d
// @param txId
// @return *commonPb.Block
// @return error
func (d *DebugStore) GetBlockByTx(txId string) (*commonPb.Block, error) {
	d.logger.Panic("not implement")
	panic("implement me")
}

// GetBlockWithRWSets 通过height得到区块和读写集
// @Description:
// @receiver d
// @param height
// @return *storePb.BlockWithRWSet
// @return error
func (d *DebugStore) GetBlockWithRWSets(height uint64) (*storePb.BlockWithRWSet, error) {
	d.logger.Panic("not implement")
	panic("implement me")
}

// GetTx 根据txid得到交易
// @Description:
// @receiver d
// @param txId
// @return *commonPb.Transaction
// @return error
func (d *DebugStore) GetTx(txId string) (*commonPb.Transaction, error) {
	d.logger.Panic("not implement")
	panic("implement me")
}

// TxExists 交易过滤，判断是否存在
// @Description:
// @receiver d
// @param txId
// @return bool
// @return error
func (d *DebugStore) TxExists(txId string) (bool, error) {
	return false, nil
}

// GetTxHeight 根据txid返回块高
// @Description:
// @receiver d
// @param txId
// @return uint64
// @return error
func (d *DebugStore) GetTxHeight(txId string) (uint64, error) {
	d.logger.Panic("not implement")
	panic("implement me")
}

// GetTxConfirmedTime 根据txid返回确认时间
// @Description:
// @receiver d
// @param txId
// @return int64
// @return error
func (d *DebugStore) GetTxConfirmedTime(txId string) (int64, error) {
	d.logger.Panic("not implement")
	panic("implement me")
}

// GetLastBlock 得到最后写入区块
// @Description:
// @receiver d
// @return *commonPb.Block
// @return error
func (d *DebugStore) GetLastBlock() (*commonPb.Block, error) {
	key := []byte("lastBlockHeight")
	data, err := d.dbHandle.Get(key)
	if err != nil || len(data) == 0 {
		return nil, errNotFound
	}
	header := &commonPb.BlockHeader{}
	err = header.Unmarshal(data)
	if err != nil {
		return nil, err
	}
	return &commonPb.Block{Header: header}, nil
}

// ReadObject 读合约key对应value
// @Description:
// @receiver d
// @param contractName
// @param key
// @return []byte
// @return error
func (d *DebugStore) ReadObject(contractName string, key []byte) ([]byte, error) {
	dbKey := append([]byte(contractName+"#"), key...)
	return d.dbHandle.Get(dbKey)
}

// ReadObjects 读合约keys对应values
// @Description:
// @receiver d
// @param contractName
// @param startKey
// @param limit
// @return protocol.StateIterator
// @return error
func (d *DebugStore) ReadObjects(contractName string, keys [][]byte) ([][]byte, error) {
	dbKeys := make([][]byte, len(keys))
	for i := 0; i < len(keys); i++ {
		dbKeys[i] = append([]byte(contractName+"#"), keys[i]...)
	}

	return d.dbHandle.GetKeys(dbKeys)
}

// SelectObject 合约范围查询
// @Description:
// @receiver d
// @param contractName
// @param startKey
// @param limit
// @return protocol.StateIterator
// @return error
func (d *DebugStore) SelectObject(contractName string, startKey []byte, limit []byte) (protocol.StateIterator, error) {
	d.logger.Panic("not implement")
	panic("implement me")
}

// GetTxRWSet 根据txid得到读写集
// @Description:
// @receiver d
// @param txId
// @return *commonPb.TxRWSet
// @return error
func (d *DebugStore) GetTxRWSet(txId string) (*commonPb.TxRWSet, error) {
	d.logger.Panic("not implement")
	panic("implement me")
}

// GetTxRWSetsByHeight 根据块高，返回读写集
// @Description:
// @receiver d
// @param height
// @return []*commonPb.TxRWSet
// @return error
func (d *DebugStore) GetTxRWSetsByHeight(height uint64) ([]*commonPb.TxRWSet, error) {
	d.logger.Panic("not implement")
	panic("implement me")
}

// GetDBHandle add next time
// @Description:
// @receiver d
// @param dbName
// @return protocol.DBHandle
func (d *DebugStore) GetDBHandle(dbName string) protocol.DBHandle {
	return d.dbHandle
}

// GetArchivedPivot add next time
// @Description:
// @receiver d
// @return uint64
func (d *DebugStore) GetArchivedPivot() uint64 {
	return 0
}

// ArchiveBlock 区块归档
// @Description:
// @receiver d
// @param archiveHeight
// @return error
func (d *DebugStore) ArchiveBlock(archiveHeight uint64) error {
	return nil
}

// RestoreBlocks 转储区块
// @Description:
// @receiver d
// @param serializedBlocks
// @return error
func (d *DebugStore) RestoreBlocks(serializedBlocks [][]byte) error {
	d.logger.Panic("not implement")
	panic("implement me")
}

// Close 关闭db
// @Description:
// @receiver d
// @return error
func (d *DebugStore) Close() error {
	return d.dbHandle.Close()
}

// GetHistoryForKey 得到历史数据
// @Description:
// @receiver d
// @param contractName
// @param key
// @return protocol.KeyHistoryIterator
// @return error
func (d *DebugStore) GetHistoryForKey(contractName string, key []byte) (protocol.KeyHistoryIterator, error) {
	d.logger.Panic("not implement")
	panic("implement me")
}

// GetAccountTxHistory 得到账户历史数据
// @Description:
// @receiver d
// @param accountId
// @return protocol.TxHistoryIterator
// @return error
func (d *DebugStore) GetAccountTxHistory(accountId []byte) (protocol.TxHistoryIterator, error) {
	d.logger.Panic("not implement")
	panic("implement me")
}

// GetContractTxHistory 得到合约历史数据
// @Description:
// @receiver d
// @param contractName
// @return protocol.TxHistoryIterator
// @return error
func (d *DebugStore) GetContractTxHistory(contractName string) (protocol.TxHistoryIterator, error) {
	d.logger.Panic("not implement")
	panic("implement me")
}
