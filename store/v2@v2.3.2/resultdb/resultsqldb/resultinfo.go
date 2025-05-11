/*
 * Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
 *
 * SPDX-License-Identifier: Apache-2.0
 *
 */

package resultsqldb

import (
	commonpb "chainmaker.org/chainmaker/pb-go/v2/common"
	"chainmaker.org/chainmaker/store/v2/conf"
	"github.com/gogo/protobuf/proto"
)

// ResultInfo defines mysql orm model, used to create mysql table 'result_infos'
// @Description:
type ResultInfo struct {
	TxId        string `gorm:"size:128;primaryKey"`
	BlockHeight uint64
	TxIndex     uint32
	Rwset       []byte `gorm:"type:longblob"`
	Status      int    `gorm:"default:0"`
	Result      []byte `gorm:"type:blob"`
	Message     string `gorm:"type:longtext"`
}

// ScanObject scan data from db
// @Description:
// @receiver b
// @param scan
// @return error
func (b *ResultInfo) ScanObject(scan func(dest ...interface{}) error) error {
	return scan(&b.TxId, &b.BlockHeight, &b.TxIndex, &b.Rwset, &b.Status, &b.Result, &b.Message)
}

// GetCreateTableSql return create table sql (mysql for production,sqlite for test)
// @Description:
// @receiver b
// @param dbType
// @return string
func (b *ResultInfo) GetCreateTableSql(dbType string) string {
	if dbType == conf.SqldbconfigSqldbtypeMysql {
		return `CREATE TABLE result_infos (
    tx_id varchar(128),block_height bigint,tx_index bigint,
    rwset longblob,status bigint DEFAULT 0,result blob,
    message longtext,PRIMARY KEY (tx_id)
    ) default character set utf8mb4`
	} else if dbType == conf.SqldbconfigSqldbtypeSqlite {
		return `CREATE TABLE result_infos (
    tx_id text,block_height integer,tx_index integer,rwset longblob,
    status integer DEFAULT 0,result blob,message longtext,
    PRIMARY KEY (tx_id)
    )`
	}
	panic("Unsupported db type:" + dbType)
}

// GetTableName 获得表的名字
// @Description:
// @receiver b
// @return string
func (b *ResultInfo) GetTableName() string {
	return "result_infos"
}

// GetInsertSql insert sql sentence
// @Description:
// @receiver b
// @param dbType
// @return string
// @return []interface{}
func (b *ResultInfo) GetInsertSql(dbType string) (string, []interface{}) {
	return "INSERT INTO result_infos values(?,?,?,?,?,?,?)",
		[]interface{}{b.TxId, b.BlockHeight, b.TxIndex, b.Rwset, b.Status, b.Result, b.Message}
}

// GetUpdateSql update sql sentence use primary key (tx_id)
// @Description:
// @receiver b
// @return string
// @return []interface{}
func (b *ResultInfo) GetUpdateSql() (string, []interface{}) {
	return "UPDATE result_infos set block_height=?,tx_index=?,rwset=?,status=?,result=?,message=?" +
			" WHERE tx_id=?",
		[]interface{}{b.BlockHeight, b.TxIndex, b.Rwset, b.Status, b.Result, b.Message, b.TxId}
}

// GetCountSql return count sql sentence use primary key (tx_id)
// @Description:
// @receiver b
// @return string
// @return []interface{}
func (b *ResultInfo) GetCountSql() (string, []interface{}) {
	return "SELECT count(*) FROM result_infos WHERE tx_id=?", []interface{}{b.TxId}
}

// GetSaveSql return replace save sql sentence
// @Description:
// @receiver b
// @param _
// @return string
// @return []interface{}
func (b *ResultInfo) GetSaveSql(_ string) (string, []interface{}) {
	return "REPLACE INTO result_infos values(?,?,?,?,?,?,?)",
		[]interface{}{b.TxId, b.BlockHeight, b.TxIndex, b.Rwset, b.Status, b.Result, b.Message}
}

// NewResultInfo construct a new HistoryInfo
// @Description:
// @param txId
// @param blockHeight
// @param txIndex
// @param result
// @param rw
// @return *ResultInfo
func NewResultInfo(txId string, blockHeight uint64, txIndex uint32, result *commonpb.ContractResult,
	rw *commonpb.TxRWSet) *ResultInfo {
	rwBytes, _ := proto.Marshal(rw)

	return &ResultInfo{
		TxId:        txId,
		BlockHeight: blockHeight,
		TxIndex:     txIndex,
		Status:      int(result.Code),
		Result:      result.Result,
		Message:     result.Message,
		Rwset:       rwBytes,
	}
}
