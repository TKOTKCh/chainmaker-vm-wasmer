/*
 * Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
 *
 * SPDX-License-Identifier: Apache-2.0
 *
 */

package statedb

import (
	"chainmaker.org/chainmaker/pb-go/v2/accesscontrol"
	configPb "chainmaker.org/chainmaker/pb-go/v2/config"
	"chainmaker.org/chainmaker/protocol/v2"
	"chainmaker.org/chainmaker/store/v2/serialization"
)

// StateDB provides handle to world state instances
// @Description:
type StateDB interface {

	// InitGenesis 创世块写入
	// @Description:
	// @param genesisBlock
	// @return error
	InitGenesis(genesisBlock *serialization.BlockWithSerializedInfo) error

	// CommitBlock commits the state in an atomic operation
	// @Description:
	// @param blockWithRWSet
	// @param isCache
	// @return error
	CommitBlock(blockWithRWSet *serialization.BlockWithSerializedInfo, isCache bool) error

	// GetChainConfig  get last chain config
	// @Description:
	// @return *configPb.ChainConfig
	// @return error
	GetChainConfig() (*configPb.ChainConfig, error)

	// ReadObject returns the state value for given contract name and key, or returns nil if none exists.
	// @Description:
	// @param contractName
	// @param key
	// @return []byte
	// @return error
	ReadObject(contractName string, key []byte) ([]byte, error)

	// ReadObjects returns the state values for given contract name and keys
	ReadObjects(contractName string, keys [][]byte) ([][]byte, error)

	// SelectObject returns an iterator that contains all the key-values between given key ranges.
	// @Description:
	// startKey is included in the results and limit is excluded.
	// @param contractName
	// @param startKey
	// @param limit
	// @return protocol.StateIterator
	// @return error
	SelectObject(contractName string, startKey []byte, limit []byte) (protocol.StateIterator, error)

	// GetLastSavepoint returns the last block height
	// @Description:
	// @return uint64
	// @return error
	GetLastSavepoint() (uint64, error)

	// Close Close is used to close database
	// @Description:
	Close()

	// QuerySingle 不在事务中，直接查询状态数据库，返回一行结果
	// @Description:
	// @param contractName
	// @param sql
	// @param values
	// @return protocol.SqlRow
	// @return error
	QuerySingle(contractName, sql string, values ...interface{}) (protocol.SqlRow, error)

	// QueryMulti  不在事务中，直接查询状态数据库，返回多行结果
	// @Description:
	// @param contractName
	// @param sql
	// @param values
	// @return protocol.SqlRows
	// @return error
	QueryMulti(contractName, sql string, values ...interface{}) (protocol.SqlRows, error)

	// ExecDdlSql
	// @Description: 执行DDL语句
	// @param contractName
	// @param sql
	// @param version
	// @return error
	ExecDdlSql(contractName, sql, version string) error

	// BeginDbTransaction
	// @Description: 启用一个事务
	// @param txName
	// @return protocol.SqlDBTransaction
	// @return error
	BeginDbTransaction(txName string) (protocol.SqlDBTransaction, error)

	// GetDbTransaction
	// @Description: 根据事务名，获得一个已经启用的事务
	// @param txName
	// @return protocol.SqlDBTransaction
	// @return error
	GetDbTransaction(txName string) (protocol.SqlDBTransaction, error)

	// CommitDbTransaction
	// @Description: 提交一个事务
	// @param txName
	// @return error
	CommitDbTransaction(txName string) error

	// RollbackDbTransaction
	// @Description: 回滚一个事务
	// @param txName
	// @return error
	RollbackDbTransaction(txName string) error

	// CreateDatabase
	// @Description: 创建database
	// @param contractName
	// @return error
	CreateDatabase(contractName string) error

	// DropDatabase
	// @Description: 删除一个合约对应的数据库
	// @param contractName
	// @return error
	DropDatabase(contractName string) error

	// GetContractDbName
	// @Description:  获得一个合约对应的状态数据库名
	// @param contractName
	// @return string
	GetContractDbName(contractName string) string

	// GetMemberExtraData
	// @Description:
	// @param member
	// @return *accesscontrol.MemberExtraData
	// @return error
	GetMemberExtraData(member *accesscontrol.Member) (*accesscontrol.MemberExtraData, error)
}
