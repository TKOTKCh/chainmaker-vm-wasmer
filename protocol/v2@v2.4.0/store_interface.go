/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

// Package protocol is a protocol package, which is base.
package protocol

import (
	"chainmaker.org/chainmaker/pb-go/v2/accesscontrol"
	"chainmaker.org/chainmaker/pb-go/v2/common"
	commonPb "chainmaker.org/chainmaker/pb-go/v2/common"
	configPb "chainmaker.org/chainmaker/pb-go/v2/config"
	"chainmaker.org/chainmaker/pb-go/v2/store"
)

var (
	// ConsensusDBName is used to store consensus data
	ConsensusDBName = "consensus"
)

// Iterator allows a chaincode to iterator over a set of
// kev/value pairs returned by range query.
type Iterator interface {
	Next() bool
	First() bool
	Error() error
	Key() []byte
	Value() []byte
	Release()
}

// StateIterator 状态数据查询迭代器结果，在对状态数据进行前缀查询或者范围查询时返回
type StateIterator interface {
	// Next 是否有下一个值
	// @return bool
	Next() bool
	// Value 当前值的KV
	// @return *store.KV
	// @return error
	Value() (*store.KV, error)
	// Release 释放迭代器的句柄
	Release()
}

// KeyHistoryIterator 状态数据Key的历史记录查询返回的迭代器接口
type KeyHistoryIterator interface {
	// Next 是否有下一个值
	// @return bool
	Next() bool
	// Value Key的变更记录
	// @return *store.KeyModification
	// @return error
	Value() (*store.KeyModification, error)
	// Release 释放迭代器的句柄
	Release()
}

// TxHistoryIterator 交易的历史迭代器
type TxHistoryIterator interface {
	// Next 是否有下一个值
	// @return bool
	Next() bool
	// Value 交易的历史记录
	// @return *store.TxHistory
	// @return error
	Value() (*store.TxHistory, error)
	// Release 释放迭代器的句柄
	Release()
}

// BlockchainStore provides handle to store instances
type BlockchainStore interface {
	StateSqlOperation
	SysContractQuery
	MemberQuery
	//InitGenesis 初始化创世单元到数据库
	InitGenesis(genesisBlock *store.BlockWithRWSet) error
	// PutBlock commits the block and the corresponding rwsets in an atomic operation
	PutBlock(block *common.Block, txRWSets []*common.TxRWSet) error

	// GetBlockByHash returns a block given it's hash, or returns nil if none exists.
	GetBlockByHash(blockHash []byte) (*common.Block, error)

	// BlockExists returns true if the black hash exist, or returns false if none exists.
	BlockExists(blockHash []byte) (bool, error)

	// GetHeightByHash returns a block height given it's hash, or returns nil if none exists.
	GetHeightByHash(blockHash []byte) (uint64, error)

	// GetBlockHeaderByHeight returns a block header by given it's height, or returns nil if none exists.
	GetBlockHeaderByHeight(height uint64) (*common.BlockHeader, error)

	// GetBlock returns a block given it's block height, or returns nil if none exists.
	GetBlock(height uint64) (*common.Block, error)

	// GetLastConfigBlock returns the last config block.
	GetLastConfigBlock() (*common.Block, error)
	//GetLastChainConfig return the last chain config
	GetLastChainConfig() (*configPb.ChainConfig, error)
	// GetBlockByTx returns a block which contains a tx.
	GetBlockByTx(txId string) (*common.Block, error)

	// GetBlockWithRWSets returns a block and the corresponding rwsets given
	// its block height, or returns nil if none exists.
	GetBlockWithRWSets(height uint64) (*store.BlockWithRWSet, error)

	// GetTx retrieves a transaction by txid, or returns nil if none exists.
	GetTx(txId string) (*common.Transaction, error)
	//GetTxWithRWSet return tx and it's rw set
	GetTxWithRWSet(txId string) (*common.TransactionWithRWSet, error)
	//GetTxInfoWithRWSet return tx and tx info and rw set
	GetTxInfoWithRWSet(txId string) (*common.TransactionInfoWithRWSet, error)
	//GetTxWithInfo get tx and tx block information
	GetTxWithInfo(txId string) (*common.TransactionInfo, error)
	// TxExists returns true if the tx exist, or returns false if none exists.
	TxExists(txId string) (bool, error)
	// TxExistsInFullDB returns true and the latest committed block height in db if the tx exist,
	// or returns false and math.MaxUint64 if none exists.
	TxExistsInFullDB(txId string) (bool, uint64, error)
	// TxExistsInIncrementDB returns true if the tx exist from starHeight to the latest committed block,
	// or returns false if none exists.
	TxExistsInIncrementDB(txId string, startHeight uint64) (bool, error)
	// TxExistsInIncrementDBState returns
	// first return value is true if the tx exist from starHeight to the
	// latest committed block,or returns false if none exists.
	// second return value is true if inside the window or false if outside the window.
	TxExistsInIncrementDBState(txId string, startHeight uint64) (bool, bool, error)
	//GetTxInfoOnly get tx block height,timestamp, txindex
	GetTxInfoOnly(txId string) (*commonPb.TransactionInfo, error)
	//Deprecated, please use GetTxInfoOnly, retrieves a transaction height by txid, or returns nil if none exists.
	GetTxHeight(txId string) (uint64, error)

	//Deprecated, please use GetTxInfoOnly, returns the confirmed time for given tx
	GetTxConfirmedTime(txId string) (int64, error)

	// GetLastBlock returns the last block.
	GetLastBlock() (*common.Block, error)

	// GetLastHeight returns the last block height.
	GetLastHeight() (uint64, error)

	// ReadObject returns the state value for given contract name and key, or returns nil if none exists.
	ReadObject(contractName string, key []byte) ([]byte, error)

	// ReadObjects returns the state values for given contract name and keys
	ReadObjects(contractName string, keys [][]byte) ([][]byte, error)

	// SelectObject returns an iterator that contains all the key-values between given key ranges.
	// startKey is included in the results and limit is excluded.
	SelectObject(contractName string, startKey []byte, limit []byte) (StateIterator, error)

	// GetTxRWSet returns an txRWSet for given txId, or returns nil if none exists.
	GetTxRWSet(txId string) (*common.TxRWSet, error)

	// GetTxRWSetsByHeight returns all the rwsets corresponding to the block,
	// or returns nil if zhe block does not exist
	GetTxRWSetsByHeight(height uint64) ([]*common.TxRWSet, error)

	// GetDBHandle returns the database handle for given dbName
	GetDBHandle(dbName string) DBHandle

	//GetArchivedPivot returns the archived pivot (include this pivot height)
	//Deprecated
	GetArchivedPivot() uint64

	// GetArchiveStatus returns archive status
	GetArchiveStatus() (*store.ArchiveStatus, error)

	// ArchiveBlock the block after backup
	ArchiveBlock(archiveHeight uint64) error

	//RestoreBlocks restore blocks from outside block data
	RestoreBlocks(serializedBlocks [][]byte) error

	// Close closes all the store db instances and releases any resources held by BlockchainStore
	Close() error
	//GetHistoryForKey 查询某合约中某个Key的变更历史
	GetHistoryForKey(contractName string, key []byte) (KeyHistoryIterator, error)
	//GetAccountTxHistory 查询一个账户的交易历史记录
	GetAccountTxHistory(accountId []byte) (TxHistoryIterator, error)
	//GetContractTxHistory 查询一个合约的调用交易历史记录
	GetContractTxHistory(contractName string) (TxHistoryIterator, error)
}

// StateSqlOperation 状态数据库的SQL操作
type StateSqlOperation interface {
	// QuerySingle 不在事务中，直接查询状态数据库，返回一行结果
	QuerySingle(contractName, sql string, values ...interface{}) (SqlRow, error)
	// QueryMulti 不在事务中，直接查询状态数据库，返回多行结果
	QueryMulti(contractName, sql string, values ...interface{}) (SqlRows, error)
	// ExecDdlSql 执行建表、修改表等DDL语句，不得在事务中运行
	ExecDdlSql(contractName, sql, version string) error
	// BeginDbTransaction 启用一个事务
	BeginDbTransaction(txName string) (SqlDBTransaction, error)
	// GetDbTransaction 根据事务名，获得一个已经启用的事务
	GetDbTransaction(txName string) (SqlDBTransaction, error)
	// CommitDbTransaction 提交一个事务
	CommitDbTransaction(txName string) error
	// RollbackDbTransaction 回滚一个事务
	RollbackDbTransaction(txName string) error
	// CreateDatabase 为新合约创建数据库
	CreateDatabase(contractName string) error
	// DropDatabase 删除一个合约对应的数据库
	DropDatabase(contractName string) error
	// GetContractDbName 获得一个合约对应的状态数据库名
	GetContractDbName(contractName string) string
}

// SqlDBHandle 对SQL数据库的操作方法
type SqlDBHandle interface {
	DBHandle
	// CreateDatabaseIfNotExist 如果数据库不存在则创建对应的数据库，创建后将当前数据库设置为新数据库，返回是否已存在
	CreateDatabaseIfNotExist(dbName string) (bool, error)
	// CreateTableIfNotExist 根据一个对象struct，自动构建对应的sql数据库表
	CreateTableIfNotExist(obj interface{}) error
	// Save 直接保存一个对象到SQL数据库中
	Save(value interface{}) (int64, error)
	// ExecSql 执行指定的SQL语句，返回受影响的行数
	ExecSql(sql string, values ...interface{}) (int64, error)
	// QuerySingle 执行指定的SQL语句，查询单条数据记录，如果查询到0条，则返回nil,nil，如果查询到多条，则返回第一条
	QuerySingle(sql string, values ...interface{}) (SqlRow, error)
	// QueryMulti 执行指定的SQL语句，查询多条数据记录，如果查询到0条，则SqlRows.Next()直接返回false
	QueryMulti(sql string, values ...interface{}) (SqlRows, error)
	// BeginDbTransaction 开启一个数据库事务，并指定该事务的名字，并缓存其句柄，如果之前已经开启了同名的事务，则返回错误
	BeginDbTransaction(txName string) (SqlDBTransaction, error)
	// GetDbTransaction 根据事务的名字，获得事务的句柄,如果事务不存在，则返回错误
	GetDbTransaction(txName string) (SqlDBTransaction, error)
	// CommitDbTransaction 提交一个事务，并从缓存中清除该事务，如果找不到对应的事务，则返回错误
	CommitDbTransaction(txName string) error
	// RollbackDbTransaction 回滚一个事务，并从缓存中清除该事务，如果找不到对应的事务，则返回错误
	RollbackDbTransaction(txName string) error
	// GetSqlDbType 获得SqlDBType字段的值
	GetSqlDbType() string
}

// SqlDBTransaction 开启一个事务后，能在这个事务中进行的操作
type SqlDBTransaction interface {
	// ChangeContextDb 改变当前上下文所使用的数据库
	ChangeContextDb(dbName string) error
	// SaveBatch 直接保存一批对象到SQL数据库中
	SaveBatch(values []interface{}) (int64, error)
	// Save 直接保存一个对象到SQL数据库中
	Save(value interface{}) (int64, error)
	// ExecSql 执行指定的SQL语句，返回受影响的行数
	ExecSql(sql string, values ...interface{}) (int64, error)
	// QuerySingle 执行指定的SQL语句，查询单条数据记录，如果查询到0条，则返回nil,nil，如果查询到多条，则返回第一条
	QuerySingle(sql string, values ...interface{}) (SqlRow, error)
	// QueryMulti 执行指定的SQL语句，查询多条数据记录，如果查询到0条，则SqlRows.Next()直接返回false
	QueryMulti(sql string, values ...interface{}) (SqlRows, error)
	// BeginDbSavePoint 创建一个新的保存点
	BeginDbSavePoint(savePointName string) error
	// RollbackDbSavePoint 回滚事务到指定的保存点
	RollbackDbSavePoint(savePointName string) error
}

// SqlRow 运行SQL查询后返回的一行数据，在获取这行数据时提供了ScanColumns，ScanObject和Data三种方法，但是三选一，调用其中一个就别再调另外一个。
type SqlRow interface {
	// ScanColumns 将这个数据的每个列赋值到dest指针对应的对象中
	ScanColumns(dest ...interface{}) error

	// Data 将这个数据转换为ColumnName为Key，Data为Value的Map中
	Data() (map[string][]byte, error)
	// IsEmpty 判断返回的SqlRow是否为空
	IsEmpty() bool
}

// SqlRows 运行SQL查询后返回的多行数据
type SqlRows interface {
	// Next 还有下一行
	Next() bool
	// ScanColumns 将当前行这个数据的每个列赋值到dest指针对应的对象中
	ScanColumns(dest ...interface{}) error
	// Data 将当前行这个数据转换为ColumnName为Key，Data为Value的Map中
	Data() (map[string][]byte, error)
	// Close 关闭sql.Rows连接
	// @return error
	Close() error
}

// DBHandle is an handle to a db
type DBHandle interface {

	// GetDbType returns db type
	GetDbType() string

	// Get returns the value for the given key, or returns nil if none exists
	Get(key []byte) ([]byte, error)

	// GetKeys returns the values for the given keys concurrent
	GetKeys(keys [][]byte) ([][]byte, error)

	// Put saves the key-values
	Put(key []byte, value []byte) error

	// Has return true if the given key exist, or return false if none exists
	Has(key []byte) (bool, error)

	// Delete deletes the given key
	Delete(key []byte) error

	// WriteBatch writes a batch in an atomic operation
	WriteBatch(batch StoreBatcher, sync bool) error

	// CompactRange compacts the underlying DB for the given key range.
	CompactRange(start, limit []byte) error

	// NewIteratorWithRange returns an iterator that contains all the key-values between given key ranges
	// start is included in the results and limit is excluded.
	NewIteratorWithRange(start []byte, limit []byte) (Iterator, error)

	// NewIteratorWithPrefix returns an iterator that contains all the key-values with given prefix
	NewIteratorWithPrefix(prefix []byte) (Iterator, error)

	// GetWriteBatchSize get each write batch numbers
	GetWriteBatchSize() uint64

	Close() error
}

// StoreBatcher used to cache key-values that commit in a atomic operation
type StoreBatcher interface {
	// Put adds a key-value
	Put(key []byte, value []byte)

	// Delete deletes a key and associated value,value is be set to nil
	Delete(key []byte)

	// Remove key and value be removed
	Remove(key []byte)

	// Len retrun the number of key-values
	Len() int

	// Merge used to merge two StoreBatcher
	Merge(batcher StoreBatcher)

	// KVs return the map of key-values
	KVs() map[string][]byte

	// SplitBatch split other kvs to more updateBatchs division by batchCnt
	SplitBatch(batchCnt uint64) []StoreBatcher

	// Get value by key
	Get(key []byte) ([]byte, error)

	// Check key-value wether exist or not
	Has(key []byte) bool
}

// SqlVerifier 在支持SQL语句操作状态数据库模式下，对合约中输入的SQL语句进行规则校验
type SqlVerifier interface {
	// VerifyDDLSql 验证输入语句是不是DDL语句，是DDL则返回nil，不是则返回error
	VerifyDDLSql(sql string) error
	// VerifyDMLSql 验证输入的SQL语句是不是更新语句（insert、update、delete），是则返回nil，不是则返回error
	VerifyDMLSql(sql string) error
	// VerifyDQLSql 验证输入的语句是不是查询语句，是则返回nil，不是则返回error
	VerifyDQLSql(sql string) error
}

// SysContractQuery query system contract data
type SysContractQuery interface {
	GetContractByName(name string) (*commonPb.Contract, error)
	GetContractBytecode(name string) ([]byte, error)
}

// MemberQuery query member information
type MemberQuery interface {
	// GetMemberExtraData get member extra data by member
	GetMemberExtraData(member *accesscontrol.Member) (*accesscontrol.MemberExtraData, error)
}
