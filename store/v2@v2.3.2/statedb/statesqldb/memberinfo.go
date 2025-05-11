/*
 * Copyright (C) BABEC. All rights reserved.
 * Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
 *
 * SPDX-License-Identifier: Apache-2.0
 */

package statesqldb

import (
	"crypto/sha256"

	"chainmaker.org/chainmaker/pb-go/v2/accesscontrol"
	"chainmaker.org/chainmaker/store/v2/conf"
)

// MemberExtraInfo defines mysql orm model, used to create mysql table 'member_extra_infos'
// @Description:
type MemberExtraInfo struct {
	MemberHash []byte `gorm:"size:32;primaryKey"`
	MemberType int    `gorm:""`
	MemberInfo []byte `gorm:"size:2000"`
	OrgId      string `gorm:"size:200"`
	Seq        uint64 //防止Sequence是数据库关键字
}

// ScanObject scan data from db
// @Description:
// @receiver b
// @param scan
// @return error
func (b *MemberExtraInfo) ScanObject(scan func(dest ...interface{}) error) error {
	return scan(&b.MemberHash, &b.MemberType, &b.MemberInfo, &b.OrgId, &b.Seq)
}

// GetCreateTableSql return member_extra_infos table ddl
// @Description:
// @receiver b
// @param dbType
// @return string
func (b *MemberExtraInfo) GetCreateTableSql(dbType string) string {
	if dbType == conf.SqldbconfigSqldbtypeMysql {
		return `CREATE TABLE member_extra_infos (
    member_hash binary(32) primary key,
    member_type int,
	member_info blob(2000),
    org_id varchar(200),
    seq bigint default 0
    ) default character set utf8mb4`
	} else if dbType == conf.SqldbconfigSqldbtypeSqlite {
		return `CREATE TABLE member_extra_infos (
	member_hash blob primary key,
    member_type integer,
	member_info blob,
    org_id text,
    seq integer
    )`
	}
	panic("Unsupported db type:" + dbType)
}

// GetTableName 获得表的名字
// @Description:
// @receiver b
// @return string
func (b *MemberExtraInfo) GetTableName() string {
	return "member_extra_infos"
}

// GetInsertSql return insert sql
// @Description:
// @receiver b
// @param dbType
// @return string
// @return []interface{}
func (b *MemberExtraInfo) GetInsertSql(dbType string) (string, []interface{}) {
	return "INSERT INTO member_extra_infos values(?,?,?,?,?)",
		[]interface{}{b.MemberHash, b.MemberType, b.MemberInfo, b.OrgId, b.Seq}
}

// GetUpdateSql return update sql
// @Description:
// @receiver b
// @return string
// @return []interface{}
func (b *MemberExtraInfo) GetUpdateSql() (string, []interface{}) {
	return "UPDATE member_extra_infos set member_type=?,member_info=?,org_id=?,seq=?" +
			" WHERE member_hash=?",
		[]interface{}{b.MemberType, b.MemberInfo, b.OrgId, b.Seq, b.MemberHash}
}

// GetCountSql return count sql
// @Description:
// @receiver b
// @return string
// @return []interface{}
func (b *MemberExtraInfo) GetCountSql() (string, []interface{}) {
	return "select count(*) FROM member_extra_infos WHERE member_hash=?",
		[]interface{}{b.MemberHash}
}

// GetSaveSql return update or insert sql
// @Description:
// @receiver b
// @return string
// @return []interface{}
func (b *MemberExtraInfo) GetSaveSql() (string, []interface{}) {
	if b.Seq > 1 { //update
		return "UPDATE member_extra_infos set seq=? WHERE member_hash=?",
			[]interface{}{b.Seq, b.MemberHash}
	}
	return b.GetInsertSql("")
}

// NewMemberExtraInfo construct MemberExtraInfo
// @Description:
// @param member
// @param extra
// @return *MemberExtraInfo
func NewMemberExtraInfo(member *accesscontrol.Member, extra *accesscontrol.MemberExtraData) *MemberExtraInfo {
	hash := getMemberHash(member)
	return &MemberExtraInfo{
		MemberHash: hash,
		MemberType: int(member.MemberType),
		MemberInfo: member.MemberInfo,
		OrgId:      member.OrgId,
		Seq:        extra.Sequence,
	}
}

// getMemberHash compute member hash
// @Description:
// @param member
// @return []byte
func getMemberHash(member *accesscontrol.Member) []byte {
	data, _ := member.Marshal()
	hash := sha256.Sum256(data)
	return hash[:]
}

// GetExtraData return MemberExtraInfo Seq
// @Description:
// @receiver b
// @return *accesscontrol.MemberExtraData
func (b *MemberExtraInfo) GetExtraData() *accesscontrol.MemberExtraData {
	return &accesscontrol.MemberExtraData{Sequence: b.Seq}
}
