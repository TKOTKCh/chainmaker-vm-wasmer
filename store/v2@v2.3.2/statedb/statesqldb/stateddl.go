/*
 * Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
 *
 * SPDX-License-Identifier: Apache-2.0
 *
 */

package statesqldb

import (
	"encoding/hex"
	"time"

	"chainmaker.org/chainmaker/common/v2/crypto"
	"chainmaker.org/chainmaker/common/v2/crypto/hash"
	"chainmaker.org/chainmaker/protocol/v2"
	"chainmaker.org/chainmaker/store/v2/conf"
)

const (
	contractStoreSeparator = "#"
)

// StateRecordSql defines mysql orm model, used to create mysql table 'state_record_sql'
// @Description:
type StateRecordSql struct {
	// id is sql hash
	Id           string    `gorm:"size:64;primaryKey"`
	ContractName string    `gorm:"size:100"`
	SqlString    string    `gorm:"size:4000"`
	SqlType      int       `gorm:"size:1;default:1"`
	Version      string    `gorm:"size:20"`
	Status       int       `gorm:"default:0"` //0: start process, 1:success 2:fail
	UpdatedAt    time.Time `gorm:"default:null"`
}

// GetCreateTableSql return create table sentence
// @Description:
// @receiver b
// @param dbType
// @return string
func (b *StateRecordSql) GetCreateTableSql(dbType string) string {
	if dbType == conf.SqldbconfigSqldbtypeMysql {
		return `CREATE TABLE state_record_sql (
					id varchar(64),
					contract_name varchar(100),
					sql_string varchar(4000),
					sql_type int,
					version varchar(20),
					status int,
					updated_at datetime(3) NULL DEFAULT null,
					PRIMARY KEY (id)
				) default character set utf8mb4`
	} else if dbType == conf.SqldbconfigSqldbtypeSqlite {
		return `CREATE TABLE state_record_sql (
					id text,
					contract_name text,
					sql_string text,
					sql_type integer,
					version text,
					status integer,
					updated_at datetime DEFAULT null,
					PRIMARY KEY (id)
				)`
	}
	panic("Unsupported db type:" + dbType)
}

// GetTableName 获得表的名字
// @Description:
// @receiver b
// @return string
func (b *StateRecordSql) GetTableName() string {
	return "state_record_sql"
}

// GetInsertSql return insert sentence
// @Description:
// @receiver b
// @param dbType
// @return string
// @return []interface{}
func (b *StateRecordSql) GetInsertSql(dbType string) (string, []interface{}) {
	return "INSERT INTO state_record_sql values(?,?,?,?,?,?,?)",
		[]interface{}{b.Id, b.ContractName, b.SqlString, b.SqlType, b.Version, b.Status, b.UpdatedAt}
}

// GetUpdateSql return update sentence
// @Description:
// @receiver b
// @return string
// @return []interface{}
func (b *StateRecordSql) GetUpdateSql() (string, []interface{}) {
	return "UPDATE state_record_sql set updated_at=?,status=?" +
			" WHERE id=?",
		[]interface{}{b.UpdatedAt, b.Status, b.Id}
}

// GetQueryStatusSql return query status sentence
// @Description:
// @receiver b
// @return string
// @return interface{}
func (b *StateRecordSql) GetQueryStatusSql() (string, interface{}) {
	return "select status FROM state_record_sql WHERE id=?",
		b.Id
}

// GetCountSql return count sentence
// @Description:
// @receiver b
// @return string
// @return []interface{}
func (b *StateRecordSql) GetCountSql() (string, []interface{}) {
	return "select count(*) FROM state_record_sql WHERE id=?",
		[]interface{}{b.Id}
}

// NewStateRecordSql construct a new StateRecordSql
// @Description:
// @param contractName
// @param sql
// @param sqlType
// @param version
// @param status
// @return *StateRecordSql
func NewStateRecordSql(contractName string, sql string, sqlType protocol.SqlType,
	version string, status int) *StateRecordSql {
	rawId := []byte(contractName + contractStoreSeparator + version + contractStoreSeparator + sql)
	bytes, _ := hash.Get(crypto.HASH_TYPE_SHA256, rawId)
	id := hex.EncodeToString(bytes)
	return &StateRecordSql{
		Id:           id,
		ContractName: contractName,
		SqlString:    sql,
		SqlType:      int(sqlType),
		Version:      version,
		Status:       status,
		UpdatedAt:    time.Now(),
	}
}
