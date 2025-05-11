/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0

*/

package blocksqldb

import (
	"encoding/json"

	acPb "chainmaker.org/chainmaker/pb-go/v2/accesscontrol"
	commonPb "chainmaker.org/chainmaker/pb-go/v2/common"
	storePb "chainmaker.org/chainmaker/pb-go/v2/store"
	"chainmaker.org/chainmaker/store/v2/conf"
)

// TxInfo defines mysql orm model, used to create mysql table 'tx_infos'
//  @Description:
type TxInfo struct {
	ChainId        string `gorm:"size:128"`
	TxType         int32  `gorm:"default:0"`
	TxId           string `gorm:"primaryKey;size:128"`
	Timestamp      int64  `gorm:"default:0"`
	ExpirationTime int64  `gorm:"default:0"`

	ContractName string `gorm:"size:128"`
	// invoke method
	Method string `gorm:"size:128"`
	// invoke parameters in k-v format
	Parameters []byte `gorm:"type:longblob"` //json
	//sequence number 交易顺序号，以Sender为主键，0表示无顺序要求，>0则必须连续递增。
	Sequence uint64 `gorm:"default:0"`
	// gas price+gas limit; fee; timeout seconds;
	//Limit *commonPb.Limit `gorm:"type:blob;size:65535"`
	Limit []byte `gorm:"type:blob;size:65535"`

	SenderOrgId      string `gorm:"size:128"`
	SenderMemberInfo []byte `gorm:"type:blob;size:65535"`
	SenderMemberType int    `gorm:"default:0"`
	//SenderSA         uint32 `gorm:"default:0"`
	SenderSignature []byte `gorm:"type:blob;size:65535"`
	Endorsers       string `gorm:"type:longtext"` //json

	TxStatusCode       int32
	ContractResultCode uint32
	ResultData         []byte `gorm:"type:longblob"`
	ResultMessage      string `gorm:"size:2000"`
	GasUsed            uint64
	ContractEvents     string `gorm:"type:longtext"` //json
	RwSetHash          []byte `gorm:"size:128"`
	Message            string `gorm:"size:2000"`

	BlockHeight uint64 `gorm:"index:idx_height_offset"`
	BlockHash   []byte `gorm:"size:128"`
	Offset      uint32 `gorm:"index:idx_height_offset"`
}

// ScanObject use scan-func scan data into TxInfo
//  @Description:
//  @receiver t
//  @param scan
//  @return error
func (t *TxInfo) ScanObject(scan func(dest ...interface{}) error) error {
	return scan(&t.ChainId, &t.TxType, &t.TxId, &t.Timestamp, &t.ExpirationTime,
		&t.ContractName, &t.Method, &t.Parameters, &t.Sequence, &t.Limit,
		&t.SenderOrgId, &t.SenderMemberInfo, &t.SenderMemberType, &t.SenderSignature, &t.Endorsers,
		&t.TxStatusCode, &t.ContractResultCode, &t.ResultData, &t.ResultMessage, &t.GasUsed,
		&t.ContractEvents, &t.RwSetHash, &t.Message,
		&t.BlockHeight, &t.BlockHash, &t.Offset)
}

// GetCreateTableSql generate table(tx_infos)-create sentence , according to dbType
//  @Description:
//  @receiver t
//  @param dbType
//  @return string
func (t *TxInfo) GetCreateTableSql(dbType string) string {
	if dbType == conf.SqldbconfigSqldbtypeMysql {
		return `CREATE TABLE tx_infos (chain_id varchar(128),tx_type int,tx_id varchar(128),timestamp bigint DEFAULT 0,
expiration_time bigint DEFAULT 0,
contract_name varchar(128),method varchar(128),parameters longblob,sequence bigint,limits blob,
sender_org_id varchar(128),sender_member_info blob,sender_member_type int,sender_signature blob,
endorsers longtext,
tx_status_code int,contract_result_code int,result_data longblob,result_message varchar(2000),
gas_used bigint,contract_events longtext,rw_set_hash varbinary(128),message varchar(2000),
block_height bigint unsigned,block_hash varbinary(128),` + "`offset`" + ` int unsigned,
PRIMARY KEY (tx_id),
INDEX idx_height_offset (block_height,` +
			"`offset`)) default character set utf8mb4"
	}
	if dbType == conf.SqldbconfigSqldbtypeSqlite {
		return `CREATE TABLE tx_infos (chain_id text,tx_type int,tx_id text,timestamp integer DEFAULT 0,
expiration_time integer DEFAULT 0,
contract_name text,method text,parameters longblob,sequence bigint,limits blob,
sender_org_id text,sender_member_info blob,sender_member_type integer,sender_signature blob,
endorsers longtext,
tx_status_code integer,contract_result_code integer,result_data longblob,result_message text,
gas_used integer,contract_events longtext,rw_set_hash blob,message text,
block_height integer ,block_hash blob,offset integer,
PRIMARY KEY (tx_id))`
	}
	panic("Unsupported db type:" + dbType)
}

// GetTableName 获得表的名字
//  @Description:
//  @receiver t
//  @return string
func (t *TxInfo) GetTableName() string {
	return "tx_infos"
}

// GetInsertSql generate table(tx_infos)-insert sentence , according to dbType
//  @Description:
//  @receiver t
//  @param dbType
//  @return string
//  @return []interface{}
func (t *TxInfo) GetInsertSql(dbType string) (string, []interface{}) {
	return "INSERT INTO tx_infos values(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)", []interface{}{
		t.ChainId, t.TxType, t.TxId, t.Timestamp, t.ExpirationTime,
		t.ContractName, t.Method, t.Parameters, t.Sequence, t.Limit,
		t.SenderOrgId, t.SenderMemberInfo, t.SenderMemberType, t.SenderSignature, t.Endorsers,
		t.TxStatusCode, t.ContractResultCode, t.ResultData, t.ResultMessage, t.GasUsed,
		t.ContractEvents, t.RwSetHash, t.Message, t.BlockHeight, t.BlockHash, t.Offset}
}

// GetUpdateSql generate table(tx_infos)-update sentence , update chain_id according to tx_id
//  @Description:
//  @receiver t
//  @return string
//  @return []interface{}
func (t *TxInfo) GetUpdateSql() (string, []interface{}) {
	return "UPDATE tx_infos SET chain_id=? WHERE tx_id=?", []interface{}{t.ChainId, t.TxId}
}

// GetCountSql generate table(tx_infos)-count sentence , query item-counts according to tx_id
//  @Description:
//  @receiver b
//  @return string
//  @return []interface{}
func (b *TxInfo) GetCountSql() (string, []interface{}) {
	return "SELECT count(*) FROM tx_infos WHERE tx_id=?", []interface{}{b.TxId}
}

// GetSaveSql REPLACE sql
// @param _
// @return string
// @return []interface{}
func (t *TxInfo) GetSaveSql(_ string) (string, []interface{}) {
	return "REPLACE INTO tx_infos values(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)", []interface{}{
		t.ChainId, t.TxType, t.TxId, t.Timestamp, t.ExpirationTime,
		t.ContractName, t.Method, t.Parameters, t.Sequence, t.Limit,
		t.SenderOrgId, t.SenderMemberInfo, t.SenderMemberType, t.SenderSignature, t.Endorsers,
		t.TxStatusCode, t.ContractResultCode, t.ResultData, t.ResultMessage, t.GasUsed,
		t.ContractEvents, t.RwSetHash, t.Message, t.BlockHeight, t.BlockHash, t.Offset}
}

// NewTxInfo construct new `TxInfo`
//  @Description:
//  @param tx
//  @param blockHeight
//  @param blockHash
//  @param offset
//  @return *TxInfo
//  @return error
func NewTxInfo(tx *commonPb.Transaction, blockHeight uint64, blockHash []byte, offset uint32) (*TxInfo, error) {
	par, _ := json.Marshal(tx.Payload.Parameters)
	var endorsers []byte
	if len(tx.Endorsers) > 0 {
		endorsers, _ = json.Marshal(tx.Endorsers)
	}
	var events []byte
	if len(tx.Result.ContractResult.ContractEvent) > 0 {
		events, _ = json.Marshal(tx.Result.ContractResult.ContractEvent)
	}
	var limitBytes []byte
	var err error
	if tx.Payload.Limit != nil {
		limitBytes, err = tx.Payload.Limit.Marshal()
		if err != nil {
			return nil, err
		}
	}
	txInfo := &TxInfo{
		ChainId:            tx.Payload.ChainId,
		TxType:             int32(tx.Payload.TxType),
		TxId:               tx.Payload.TxId,
		Timestamp:          tx.Payload.Timestamp,
		ExpirationTime:     tx.Payload.ExpirationTime,
		ContractName:       tx.Payload.ContractName,
		Method:             tx.Payload.Method,
		Parameters:         par,
		Sequence:           tx.Payload.Sequence,
		Limit:              limitBytes,
		SenderOrgId:        getSender(tx).Signer.OrgId,
		SenderMemberInfo:   getSender(tx).Signer.MemberInfo,
		SenderMemberType:   int(getSender(tx).Signer.MemberType),
		SenderSignature:    getSender(tx).Signature,
		Endorsers:          string(endorsers),
		TxStatusCode:       int32(tx.Result.Code),
		ContractResultCode: tx.Result.ContractResult.Code,
		ResultData:         tx.Result.ContractResult.Result,
		ResultMessage:      tx.Result.ContractResult.Message,
		GasUsed:            tx.Result.ContractResult.GasUsed,
		ContractEvents:     string(events),
		RwSetHash:          tx.Result.RwSetHash,
		Message:            tx.Result.Message,

		BlockHeight: blockHeight,
		BlockHash:   blockHash,
		Offset:      offset,
	}

	return txInfo, nil
}

//  getSender return tx sender
//  @Description:
//  @param tx
//  @return *commonPb.EndorsementEntry
func getSender(tx *commonPb.Transaction) *commonPb.EndorsementEntry {
	if tx.Sender == nil {
		return &commonPb.EndorsementEntry{
			Signer: &acPb.Member{
				OrgId:      "",
				MemberType: 0,
				MemberInfo: nil,
			},
			Signature: nil,
		}
	}
	return tx.Sender
}

//  getParameters retrieve key-value pairs from json
//  @Description:
//  @param par
//  @return []*commonPb.KeyValuePair
func getParameters(par []byte) []*commonPb.KeyValuePair {
	var pairs []*commonPb.KeyValuePair
	_ = json.Unmarshal(par, &pairs)
	return pairs
}

//  getEndorsers retrieve endorsements from json
//  @Description:
//  @param endorsers
//  @return []*commonPb.EndorsementEntry
func getEndorsers(endorsers string) []*commonPb.EndorsementEntry {
	var pairs []*commonPb.EndorsementEntry
	_ = json.Unmarshal([]byte(endorsers), &pairs)
	return pairs
}

//  getContractEvents retrieve contract event from json
//  @Description:
//  @param events
//  @return []*commonPb.ContractEvent
func getContractEvents(events string) []*commonPb.ContractEvent {
	var pairs []*commonPb.ContractEvent
	_ = json.Unmarshal([]byte(events), &pairs)
	return pairs
}

// GetTx transfer TxInfo to commonPb.Transaction
//  @Description:
//  @receiver t
//  @return *commonPb.Transaction
//  @return error
func (t *TxInfo) GetTx() (*commonPb.Transaction, error) {
	tx := &commonPb.Transaction{
		Payload: &commonPb.Payload{
			ChainId:        t.ChainId,
			TxType:         commonPb.TxType(t.TxType),
			TxId:           t.TxId,
			Timestamp:      t.Timestamp,
			ExpirationTime: t.ExpirationTime,
			ContractName:   t.ContractName,
			Method:         t.Method,
			Parameters:     getParameters(t.Parameters),
			Sequence:       t.Sequence,
		},
		Sender: &commonPb.EndorsementEntry{
			Signer: &acPb.Member{
				OrgId:      t.SenderOrgId,
				MemberInfo: t.SenderMemberInfo,
				MemberType: acPb.MemberType(t.SenderMemberType),
				//SignatureAlgorithm: t.SenderSA,
			},
			Signature: t.SenderSignature,
		},
		Endorsers: getEndorsers(t.Endorsers),
		Result: &commonPb.Result{
			Code:      commonPb.TxStatusCode(t.TxStatusCode),
			RwSetHash: t.RwSetHash,
			Message:   t.Message,
			ContractResult: &commonPb.ContractResult{
				Code:          t.ContractResultCode,
				Result:        t.ResultData,
				Message:       t.ResultMessage,
				GasUsed:       t.GasUsed,
				ContractEvent: getContractEvents(t.ContractEvents),
			},
		},
	}
	if len(t.Limit) > 0 {
		playLoadLimit := &commonPb.Limit{}
		err := playLoadLimit.Unmarshal(t.Limit)
		if err != nil {
			return nil, err
		}
		tx.Payload.Limit = playLoadLimit
	}
	return tx, nil
}

// GetTxInfo retrieve TransactionStoreInfo from TxInfo
//  @Description:
//  @receiver t
//  @return *storePb.TransactionStoreInfo
//  @return error
func (t *TxInfo) GetTxInfo() (*storePb.TransactionStoreInfo, error) {
	txInfo := &storePb.TransactionStoreInfo{
		BlockHeight: t.BlockHeight,
		BlockHash:   t.BlockHash,
		TxIndex:     t.Offset,
	}
	var err error
	txInfo.Transaction, err = t.GetTx()
	return txInfo, err
}
