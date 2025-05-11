/*
 * Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
 *
 * SPDX-License-Identifier: Apache-2.0
 *
 */

package statesqldb

import (
	"time"

	"chainmaker.org/chainmaker/pb-go/v2/store"
	"chainmaker.org/chainmaker/protocol/v2"
	"chainmaker.org/chainmaker/store/v2/conf"
)

// nolint
var (
	DbType_Mysql  = "mysql"
	DbType_Sqlite = "sqlite"
)

// StateInfo defines mysql orm model, used to create mysql table 'state_infos'
// @Description:
type StateInfo struct {
	//ID           uint   `gorm:"primarykey"`
	ContractName string `gorm:"size:128;primaryKey"`
	ObjectKey    []byte `gorm:"size:128;primaryKey;default:''"`
	ObjectValue  []byte `gorm:"type:longblob"`
	BlockHeight  uint64 `gorm:"index:idx_height"`
	//CreatedAt    time.Time `gorm:"default:null"`
	UpdatedAt time.Time `gorm:"default:null"`
}

// ScanObject scan data from db
// @Description:
// @receiver b
// @param scan
// @return error
func (b *StateInfo) ScanObject(scan func(dest ...interface{}) error) error {
	return scan(&b.ContractName, &b.ObjectKey, &b.ObjectValue, &b.BlockHeight, &b.UpdatedAt)
}

// GetCreateTableSql return create table sentence
// @Description:
// @receiver b
// @param dbType
// @return string
func (b *StateInfo) GetCreateTableSql(dbType string) string {
	if dbType == conf.SqldbconfigSqldbtypeMysql {
		return `CREATE TABLE state_infos (
    contract_name varchar(128),object_key varbinary(256) DEFAULT '',
    object_value longblob,block_height bigint unsigned,
    updated_at datetime(3) NULL DEFAULT null,
    PRIMARY KEY (contract_name,object_key),
    INDEX idx_height (block_height)
    ) default character set utf8mb4`
	} else if dbType == conf.SqldbconfigSqldbtypeSqlite {
		return `CREATE TABLE state_infos (
    contract_name text,object_key blob DEFAULT '',
    object_value longblob,block_height integer,updated_at datetime DEFAULT null,
    PRIMARY KEY (contract_name,object_key)
    )`
	}
	panic("Unsupported db type:" + dbType)
}

// GetTableName 获得表的名字
// @Description:
// @receiver b
// @return string
func (b *StateInfo) GetTableName() string {
	return "state_infos"
}

// GetInsertSql return insert sql sentence
// @Description:
// @receiver b
// @param dbType
// @return string
// @return []interface{}
func (b *StateInfo) GetInsertSql(dbType string) (string, []interface{}) {
	if dbType == DbType_Sqlite {
		return "INSERT INTO state_infos values(?,?,?,?,?)",
			[]interface{}{b.ContractName, b.ObjectKey, b.ObjectValue,
				b.BlockHeight, b.UpdatedAt}
	}
	return "INSERT INTO state_infos" +
			"(contract_name,object_key,object_value,block_height,updated_at)" +
			" values(?,?,?,?,?) " +
			"on duplicate key update object_value=?,block_height=?,updated_at=?",
		[]interface{}{b.ContractName, b.ObjectKey, b.ObjectValue,
			b.BlockHeight, b.UpdatedAt, b.ObjectValue, b.BlockHeight, b.UpdatedAt}
}

// GetUpdateSql return update sql sentence
// @Description:
// @receiver b
// @return string
// @return []interface{}
func (b *StateInfo) GetUpdateSql() (string, []interface{}) {
	return "UPDATE state_infos set object_value=?,block_height=?,updated_at=?" +
			" WHERE contract_name=? and object_key=?",
		[]interface{}{b.ObjectValue, b.BlockHeight, b.UpdatedAt, b.ContractName, b.ObjectKey}
}

// GetCountSql return update sql sentence
// @Description:
// @receiver b
// @return string
// @return []interface{}
func (b *StateInfo) GetCountSql() (string, []interface{}) {
	return "select count(*) FROM state_infos WHERE contract_name=? and object_key=?",
		[]interface{}{b.ContractName, b.ObjectKey}
}

// GetSaveSql return REPLACE into SQL
// @param _
// @return string
// @return []interface{}
func (b *StateInfo) GetSaveSql(_ string) (string, []interface{}) {
	return "REPLACE INTO state_infos" +
			"(contract_name,object_key,object_value,block_height,updated_at)" +
			" values(?,?,?,?,?) ",
		[]interface{}{b.ContractName, b.ObjectKey, b.ObjectValue, b.BlockHeight, b.UpdatedAt}
}

// NewStateInfo construct a new StateInfo
// @Description:
// @param contractName
// @param objectKey
// @param objectValue
// @param blockHeight
// @param t
// @return *StateInfo
func NewStateInfo(contractName string, objectKey []byte, objectValue []byte, blockHeight uint64,
	t time.Time) *StateInfo {
	return &StateInfo{
		ContractName: contractName,
		ObjectKey:    objectKey,
		ObjectValue:  objectValue,
		BlockHeight:  blockHeight,
		UpdatedAt:    t,
	}
}

// kvIterator iterator
// @Description:
type kvIterator struct {
	rows protocol.SqlRows
}

// newKVIterator construct kvIterator
// @Description:
// @param rows
// @return *kvIterator
func newKVIterator(rows protocol.SqlRows) *kvIterator {
	return &kvIterator{
		rows: rows,
	}
}

// Next check has value
// @Description:
// @receiver kvi
// @return bool
func (kvi *kvIterator) Next() bool {
	return kvi.rows.Next()
}

// Value scan current key value pair
// @Description:
// @receiver kvi
// @return *store.KV
// @return error
func (kvi *kvIterator) Value() (*store.KV, error) {
	var kv StateInfo
	err := kv.ScanObject(kvi.rows.ScanColumns)
	if err != nil {
		return nil, err
	}
	return &store.KV{
		ContractName: kv.ContractName,
		Key:          kv.ObjectKey,
		Value:        kv.ObjectValue,
	}, nil
}

// Release release the iterator
// @Description:
// @receiver kvi
func (kvi *kvIterator) Release() {
	kvi.rows.Close()
}
