/*
 * Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
 *
 * SPDX-License-Identifier: Apache-2.0
 *
 */

package resultfiledb

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"chainmaker.org/chainmaker/store/v2/binlog"
	"chainmaker.org/chainmaker/store/v2/conf"

	"chainmaker.org/chainmaker/pb-go/v2/syscontract"
	leveldbprovider "chainmaker.org/chainmaker/store-leveldb/v2"

	acPb "chainmaker.org/chainmaker/pb-go/v2/accesscontrol"
	commonPb "chainmaker.org/chainmaker/pb-go/v2/common"
	storePb "chainmaker.org/chainmaker/pb-go/v2/store"
	"chainmaker.org/chainmaker/protocol/v2"
	"chainmaker.org/chainmaker/protocol/v2/test"
	"chainmaker.org/chainmaker/store/v2/serialization"
	"github.com/stretchr/testify/assert"
)

var log = &test.GoLogger{}
var blog binlog.BinLogger = binlog.NewMemBinlog(&test.GoLogger{})

func generateBlockHash(chainId string, height uint64) []byte {
	blockHash := sha256.Sum256([]byte(fmt.Sprintf("%s-%d", chainId, height)))
	return blockHash[:]
}

func generateTxId(chainId string, height uint64, index int) string {
	txIdBytes := sha256.Sum256([]byte(fmt.Sprintf("%s-%d-%d", chainId, height, index)))
	return hex.EncodeToString(txIdBytes[:32])
}

func createConfigBlock(chainId string, height uint64) *storePb.BlockWithRWSet {
	block := &commonPb.Block{
		Header: &commonPb.BlockHeader{
			ChainId:     chainId,
			BlockHeight: height,
		},
		Txs: []*commonPb.Transaction{
			{
				Payload: &commonPb.Payload{
					ChainId:      chainId,
					TxType:       commonPb.TxType_INVOKE_CONTRACT,
					ContractName: syscontract.SystemContract_CHAIN_CONFIG.String(),
				},
				Sender: &commonPb.EndorsementEntry{Signer: &acPb.Member{OrgId: "org1", MemberInfo: []byte("cert1...")},
					Signature: []byte("sign1"),
				},
				Result: &commonPb.Result{
					Code: commonPb.TxStatusCode_SUCCESS,
					ContractResult: &commonPb.ContractResult{
						Result: []byte("ok"),
					},
				},
			},
		},
	}

	block.Header.BlockHash = generateBlockHash(chainId, height)
	block.Txs[0].Payload.TxId = generateTxId(chainId, height, 0)
	return &storePb.BlockWithRWSet{
		Block:    block,
		TxRWSets: []*commonPb.TxRWSet{},
	}
}

func createBlockAndRWSets(chainId string, height uint64, txNum int) *storePb.BlockWithRWSet {
	block := &commonPb.Block{
		Header: &commonPb.BlockHeader{
			ChainId:     chainId,
			BlockHeight: height,
		},
	}

	for i := 0; i < txNum; i++ {
		tx := &commonPb.Transaction{
			Payload: &commonPb.Payload{
				ChainId: chainId,
				TxId:    generateTxId(chainId, height, i),
			},
			Sender: &commonPb.EndorsementEntry{Signer: &acPb.Member{OrgId: "org1", MemberInfo: []byte("cert1...")},
				Signature: []byte("sign1"),
			},
			Result: &commonPb.Result{
				Code: commonPb.TxStatusCode_SUCCESS,
				ContractResult: &commonPb.ContractResult{
					Result: []byte("ok"),
				},
			},
		}
		block.Txs = append(block.Txs, tx)
	}

	block.Header.BlockHash = generateBlockHash(chainId, height)
	var txRWSets []*commonPb.TxRWSet
	for i := 0; i < txNum; i++ {
		key := fmt.Sprintf("key_%d", i)
		value := fmt.Sprintf("value_%d", i)
		txRWset := &commonPb.TxRWSet{
			TxId: block.Txs[i].Payload.TxId,
			TxWrites: []*commonPb.TxWrite{
				{
					Key:          []byte(key),
					Value:        []byte(value),
					ContractName: "contract1",
				},
			},
		}
		txRWSets = append(txRWSets, txRWset)
	}

	return &storePb.BlockWithRWSet{
		Block:    block,
		TxRWSets: txRWSets,
	}
}

var testChainId = "testchainid_1"
var block0 = createConfigBlock(testChainId, 0)
var block1 = createBlockAndRWSets(testChainId, 1, 10)
var block2 = createBlockAndRWSets(testChainId, 2, 2)

/*var block3, _ = createBlockAndRWSets(testChainId, 3, 2)
var configBlock4 = createConfigBlock(testChainId, 4)
var block5, _ = createBlockAndRWSets(testChainId, 5, 3)*/

func createBlock(chainId string, height uint64) *commonPb.Block {
	block := &commonPb.Block{
		Header: &commonPb.BlockHeader{
			ChainId:     chainId,
			BlockHeight: height,
		},
		Txs: []*commonPb.Transaction{
			{
				Payload: &commonPb.Payload{
					ChainId: chainId,
				},
				Sender: &commonPb.EndorsementEntry{Signer: &acPb.Member{OrgId: "org1", MemberInfo: []byte("cert1...")},
					Signature: []byte("sign1"),
				},
				Result: &commonPb.Result{
					Code: commonPb.TxStatusCode_SUCCESS,
					ContractResult: &commonPb.ContractResult{
						Result: []byte("ok"),
					},
				},
			},
		},
	}

	block.Header.BlockHash = generateBlockHash(chainId, height)
	block.Txs[0].Payload.TxId = generateTxId(chainId, height, 0)
	return block
}

//
//func TestMain(m *testing.M) {
//	fmt.Println("begin")
//	db, err := NewHistoryMysqlDB(testChainId, log)
//	if err != nil {
//		panic("faild to open mysql")
//	}
//	// clear data
//	historyMysqlDB := db.(*ResultFileDB)
//	historyMysqlDB.db.Migrator().DropTable(&HistoryInfo{})
//	m.Run()
//	fmt.Println("end")
//}

func initProvider() protocol.DBHandle {
	return leveldbprovider.NewMemdbHandle()
}

func getSqlConfig() *conf.StorageConfig {
	conf1 := &conf.StorageConfig{}
	conf1.StorePath = filepath.Join(os.TempDir(), fmt.Sprintf("%d", time.Now().Nanosecond()))
	conf1.WriteBlockType = 0
	conf1.DisableStateCache = true

	var sqlconfig = make(map[string]interface{})
	sqlconfig["sqldb_type"] = "sqlite"
	sqlconfig["dsn"] = ":memory:"

	dbConfig := &conf.DbConfig{
		Provider:    "sql",
		SqlDbConfig: sqlconfig,
	}
	statedbConfig := &conf.DbConfig{
		Provider:    "sql",
		SqlDbConfig: sqlconfig,
	}
	conf1.BlockDbConfig = dbConfig
	conf1.StateDbConfig = statedbConfig
	conf1.HistoryDbConfig = conf.NewHistoryDbConfig(dbConfig)
	conf1.ResultDbConfig = dbConfig
	conf1.ContractEventDbConfig = dbConfig
	conf1.DisableContractEventDB = true
	return conf1
}

//初始化DB并同时初始化创世区块
func initDb() *ResultFileDB {
	blog = binlog.NewMemBinlog(&test.GoLogger{})
	db := NewResultFileDB(testChainId, initProvider(), log, getSqlConfig(), blog)
	commitBlock(db, block0)
	return db
}

func TestResultFileDB_CommitBlock(t *testing.T) {
	db := initDb()
	block1.TxRWSets[0].TxWrites[0].Value = nil

	err := commitBlock(db, block1)
	assert.Nil(t, err)
}
func commitBlock(db *ResultFileDB, block *storePb.BlockWithRWSet) error {
	data, bl, _ := serialization.SerializeBlock(block)
	fileName, offset, length, err := blog.Write(block.Block.Header.BlockHeight+1, data)
	if err != nil {
		return err
	}
	bl.Index = &storePb.StoreInfo{
		StoreType: 0,
		FileName:  fileName,
		Offset:    offset,
		ByteLen:   length,
	}
	if block.Block.Header.BlockHeight == 0 {
		return db.InitGenesis(bl)
	}
	err = db.CommitBlock(bl, true)
	if err != nil {
		panic("db.CommitBlock(bl, true) error")
	}
	err = db.CommitBlock(bl, false)
	if err != nil {
		panic("db.CommitBlock(bl, false) error")
	}
	return nil
}

func TestHistorySqlDB_GetLastSavepoint(t *testing.T) {
	db := initDb()
	err := commitBlock(db, block1)
	assert.Nil(t, err)
	height, err := db.GetLastSavepoint()
	assert.Nil(t, err)
	assert.Equal(t, uint64(block1.Block.Header.BlockHeight), height)
	err = commitBlock(db, block2)
	assert.Nil(t, err)
	height, err = db.GetLastSavepoint()
	assert.Nil(t, err)
	assert.Equal(t, uint64(block2.Block.Header.BlockHeight), height)
}
func TestResultFileDB_GetTxRWSet(t *testing.T) {
	db := initDb()
	err := commitBlock(db, block1)
	assert.Nil(t, err)
	txId := block1.Block.Txs[0].Payload.TxId
	rwset, err := db.GetTxRWSet(txId)
	assert.Nil(t, err)
	t.Log(rwset)
}
