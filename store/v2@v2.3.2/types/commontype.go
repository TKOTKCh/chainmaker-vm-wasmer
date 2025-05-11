/*
 * Copyright (C) BABEC. All rights reserved.
 * Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
 *
 * SPDX-License-Identifier: Apache-2.0
 */

// Package types package
// @Description:
package types

// SavePoint add next time
// @Description:
type SavePoint struct {
	BlockHeight uint64 `gorm:"primarykey"`
}

// GetCreateTableSql 获得创建table的ddl语句
// @Description:
// @receiver b
// @param dbType
// @return string
func (b *SavePoint) GetCreateTableSql(dbType string) string {
	if dbType == "mysql" {
		return "CREATE TABLE `save_points` (`block_height` bigint unsigned AUTO_INCREMENT,PRIMARY KEY (`block_height`))"
	} else if dbType == "sqlite" {
		return "CREATE TABLE `save_points` (`block_height` integer,PRIMARY KEY (`block_height`))"
	}
	panic("Unsupported db type:" + dbType)
}

// GetTableName 获得table name
// @Description:
// @receiver b
// @return string
func (b *SavePoint) GetTableName() string {
	return "save_points"
}

// GetInsertSql 获得 insert sql 语句
// @Description:
// @receiver b
// @param dbType
// @return string
// @return []interface{}
func (b *SavePoint) GetInsertSql(dbType string) (string, []interface{}) {
	return "INSERT INTO save_points values(?)", []interface{}{b.BlockHeight}
}

// GetUpdateSql 获得 update sql 语句
// @Description:
// @receiver b
// @return string
// @return []interface{}
func (b *SavePoint) GetUpdateSql() (string, []interface{}) {
	return "UPDATE save_points set block_height=?", []interface{}{b.BlockHeight}
}

// GetCountSql 获得 得到数据总行数 sql 语句
// @Description:
// @receiver b
// @return string
// @return []interface{}
func (b *SavePoint) GetCountSql() (string, []interface{}) {
	return "SELECT count(*) FROM save_points", []interface{}{}
}

// GetSaveSql 获得Replace into SQL
// @param _
// @return string
// @return []interface{}
func (b *SavePoint) GetSaveSql(_ string) (string, []interface{}) {
	return "REPLACE INTO save_points (block_height) values(?)",
		[]interface{}{b.BlockHeight}
}
