/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0

*/

package historysqldb

import "chainmaker.org/chainmaker/store/v2/conf"

// StateHistoryInfo defines mysql orm model, used to create mysql table 'state_history_infos'
// @Description:
type StateHistoryInfo struct {
	ContractName string `gorm:"size:128;primaryKey"`
	StateKey     []byte `gorm:"size:128;primaryKey"`
	TxId         string `gorm:"size:128;primaryKey"`
	BlockHeight  uint64 `gorm:"primaryKey"`
}

// GetCreateTableSql 构造返回建表sql语句
// @Description:
// @receiver b
// @param dbType
// @return string
func (b *StateHistoryInfo) GetCreateTableSql(dbType string) string {
	if dbType == conf.SqldbconfigSqldbtypeMysql {
		return `CREATE TABLE state_history_infos (
    contract_name varchar(128),state_key varbinary(256),tx_id varchar(128),block_height bigint unsigned,
    PRIMARY KEY (contract_name,state_key,tx_id,block_height)
    ) default character set utf8mb4`
	} else if dbType == conf.SqldbconfigSqldbtypeSqlite {
		return `CREATE TABLE state_history_infos (
    contract_name text,state_key blob,tx_id text,block_height integer,
    PRIMARY KEY (contract_name,state_key,tx_id,block_height))`
	}
	panic("Unsupported db type:" + dbType)
}

// GetTableName 获得表的名字
// @Description: 返回表名字
// @receiver b
// @return string
func (b *StateHistoryInfo) GetTableName() string {
	return "state_history_infos"
}

// GetInsertSql 返回insert sql
// @Description:
// @receiver b
// @param _
// @return string
// @return []interface{}
func (b *StateHistoryInfo) GetInsertSql(_ string) (string, []interface{}) {
	return "INSERT INTO state_history_infos values(?,?,?,?)",
		[]interface{}{b.ContractName, b.StateKey, b.TxId, b.BlockHeight}
}

// GetUpdateSql 返回 update sql
// @Description:
// @receiver b
// @return string
// @return []interface{}
func (b *StateHistoryInfo) GetUpdateSql() (string, []interface{}) {
	return `UPDATE state_history_infos set contract_name=?
			 WHERE contract_name=? and state_key=? and tx_id=? and block_height=?`,
		[]interface{}{b.ContractName, b.ContractName, b.StateKey, b.TxId, b.BlockHeight}
}

// GetCountSql 返回 总行数
// @Description:
// @receiver b
// @return string
// @return []interface{}
func (b *StateHistoryInfo) GetCountSql() (string, []interface{}) {
	return "SELECT count(*) FROM state_history_infos" +
			" WHERE contract_name=? and state_key=? and tx_id=? and block_height=?",
		[]interface{}{b.ContractName, b.StateKey, b.TxId, b.BlockHeight}
}

// GetSaveSql 返回 replace into 语句
// @Description:
// @receiver b
// @param _
// @return string
// @return []interface{}
func (b *StateHistoryInfo) GetSaveSql(_ string) (string, []interface{}) {
	return "REPLACE INTO state_history_infos values(?,?,?,?)",
		[]interface{}{b.ContractName, b.StateKey, b.TxId, b.BlockHeight}
}

// NewStateHistoryInfo construct a new HistoryInfo
// @Description:
// @param contractName
// @param txid
// @param stateKey
// @param blockHeight
// @return *StateHistoryInfo
func NewStateHistoryInfo(contractName, txid string, stateKey []byte, blockHeight uint64) *StateHistoryInfo {
	return &StateHistoryInfo{
		TxId:         txid,
		ContractName: contractName,
		StateKey:     stateKey,
		BlockHeight:  blockHeight,
	}
}

// AccountTxHistoryInfo defines mysql orm model, used to create mysql table 'account_tx_history_infos'
// @Description:
type AccountTxHistoryInfo struct {
	AccountId   []byte `gorm:"size:2048;primaryKey"` //primary key size max=3072
	BlockHeight uint64 `gorm:"primaryKey"`
	TxId        string `gorm:"size:128;primaryKey"`
}

// GetCreateTableSql  获得 建表ddl语句
// @Description:
// @receiver b
// @param dbType
// @return string
func (b *AccountTxHistoryInfo) GetCreateTableSql(dbType string) string {
	if dbType == conf.SqldbconfigSqldbtypeMysql {
		return `CREATE TABLE account_tx_history_infos (
    account_id varbinary(2048),block_height bigint unsigned,tx_id varchar(128),
    PRIMARY KEY (account_id,block_height,tx_id)
    ) default character set utf8mb4`
	} else if dbType == conf.SqldbconfigSqldbtypeSqlite {
		return `CREATE TABLE account_tx_history_infos (
account_id blob,block_height integer,tx_id text,
PRIMARY KEY (account_id,block_height,tx_id))`
	}
	panic("Unsupported db type:" + dbType)
}

// GetTableName 获得表的名字
// @Description: 获得表名
// @receiver b
// @return string
func (b *AccountTxHistoryInfo) GetTableName() string {
	return "account_tx_history_infos"
}

// GetInsertSql 获得insert sql
// @Description:
// @receiver b
// @param dbType
// @return string
// @return []interface{}
func (b *AccountTxHistoryInfo) GetInsertSql(dbType string) (string, []interface{}) {
	return "INSERT INTO account_tx_history_infos values(?,?,?)", []interface{}{b.AccountId, b.BlockHeight, b.TxId}
}

// GetUpdateSql 获得 update sql
// @Description:
// @receiver b
// @return string
// @return []interface{}
func (b *AccountTxHistoryInfo) GetUpdateSql() (string, []interface{}) {
	return "UPDATE account_tx_history_infos set account_id=?" +
			" WHERE account_id=? and block_height=? and tx_id=?",
		[]interface{}{b.AccountId, b.AccountId, b.BlockHeight, b.TxId}
}

// GetCountSql  获得 总行数
// @Description:
// @receiver b
// @return string
// @return []interface{}
func (b *AccountTxHistoryInfo) GetCountSql() (string, []interface{}) {
	return "SELECT count(*) FROM account_tx_history_infos" +
			" WHERE account_id=? and block_height=? and tx_id=?",
		[]interface{}{b.AccountId, b.BlockHeight, b.TxId}
}

// GetSaveSql  获得 replace into sql
// @Description:
// @receiver b
// @param _
// @return string
// @return []interface{}
func (b *AccountTxHistoryInfo) GetSaveSql(_ string) (string, []interface{}) {
	return "REPLACE INTO account_tx_history_infos values(?,?,?)", []interface{}{b.AccountId, b.BlockHeight, b.TxId}
}

// ContractTxHistoryInfo struct
// @Description:
type ContractTxHistoryInfo struct {
	ContractName string `gorm:"size:128;primaryKey"`
	BlockHeight  uint64 `gorm:"primaryKey"`
	TxId         string `gorm:"size:128;primaryKey"`
	AccountId    []byte `gorm:"size:2048"`
}

// GetCreateTableSql 获得建表ddl语句
// @Description:
// @receiver b
// @param dbType
// @return string
func (b *ContractTxHistoryInfo) GetCreateTableSql(dbType string) string {
	if dbType == conf.SqldbconfigSqldbtypeMysql {
		return `CREATE TABLE contract_tx_history_infos (
    contract_name varchar(128),block_height bigint unsigned,tx_id varchar(128),
    account_id varbinary(2048),PRIMARY KEY (contract_name,block_height,tx_id)
    ) default character set utf8mb4`
	} else if dbType == conf.SqldbconfigSqldbtypeSqlite {
		return `CREATE TABLE contract_tx_history_infos (
    contract_name text,block_height integer,tx_id text,account_id blob,
    PRIMARY KEY (contract_name,block_height,tx_id)
    )`
	}
	panic("Unsupported db type:" + dbType)
}

// GetTableName 获得表的名字
// @Description: 获得 表名
// @receiver b
// @return string
func (b *ContractTxHistoryInfo) GetTableName() string {
	return "contract_tx_history_infos"
}

// GetInsertSql 获得 insert 语句
// @Description:
// @receiver b
// @param dbType
// @return string
// @return []interface{}
func (b *ContractTxHistoryInfo) GetInsertSql(dbType string) (string, []interface{}) {
	return "INSERT INTO contract_tx_history_infos values(?,?,?,?)",
		[]interface{}{b.ContractName, b.BlockHeight, b.TxId, b.AccountId}
}

// GetUpdateSql 获得更新sql语句
// @Description:
// @receiver b
// @return string
// @return []interface{}
func (b *ContractTxHistoryInfo) GetUpdateSql() (string, []interface{}) {
	return `UPDATE contract_tx_history_infos 
set account_id=?
WHERE contract_name=? and block_height=? and tx_id=?`,
		[]interface{}{b.AccountId, b.ContractName, b.BlockHeight, b.TxId}
}

// GetCountSql 获得 count语句
// @Description:
// @receiver b
// @return string
// @return []interface{}
func (b *ContractTxHistoryInfo) GetCountSql() (string, []interface{}) {
	return "SELECT count(*) FROM contract_tx_history_infos" +
			" WHERE contract_name=? and block_height=? and tx_id=?",
		[]interface{}{b.ContractName, b.BlockHeight, b.TxId}
}

// GetSaveSql 返回 replace into 语句
// @Description:
// @receiver b
// @param _
// @return string
// @return []interface{}
func (b *ContractTxHistoryInfo) GetSaveSql(_ string) (string, []interface{}) {
	return "REPLACE INTO contract_tx_history_infos values(?,?,?,?)",
		[]interface{}{b.ContractName, b.BlockHeight, b.TxId, b.AccountId}
}
