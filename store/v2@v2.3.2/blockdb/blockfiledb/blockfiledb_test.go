/*
 * Copyright (C) BABEC. All rights reserved.
 * Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
 *
 * SPDX-License-Identifier: Apache-2.0
 */

package blockfiledb

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"chainmaker.org/chainmaker/store/v2/binlog"
	"chainmaker.org/chainmaker/store/v2/conf"

	acPb "chainmaker.org/chainmaker/pb-go/v2/accesscontrol"
	commonPb "chainmaker.org/chainmaker/pb-go/v2/common"
	storePb "chainmaker.org/chainmaker/pb-go/v2/store"
	"chainmaker.org/chainmaker/pb-go/v2/syscontract"
	"chainmaker.org/chainmaker/protocol/v2/test"
	leveldbprovider "chainmaker.org/chainmaker/store-leveldb/v2"
	"chainmaker.org/chainmaker/store/v2/serialization"
	"github.com/stretchr/testify/assert"
)

var (
	logger = &test.GoLogger{}
	//chainId = "chain1"
	config1 = getSqlConfig()
)

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
	conf1.DisableBlockFileDb = true
	return conf1
}

func createConfigBlock(chainId string, height uint64, preConfHeight uint64) *commonPb.Block {
	block := &commonPb.Block{
		Header: &commonPb.BlockHeader{
			ChainId:     chainId,
			BlockHeight: height,
			Proposer:    &acPb.Member{MemberInfo: []byte("User1")},
			BlockType:   0,
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
	block.Header.PreConfHeight = preConfHeight
	return block
}

func createBlockAndRWSets(chainId string, height uint64, txNum int, preConfHeight uint64) (*commonPb.Block, []*commonPb.TxRWSet) {
	block := &commonPb.Block{
		Header: &commonPb.BlockHeader{
			ChainId:        chainId,
			BlockHeight:    height,
			Proposer:       &acPb.Member{MemberInfo: []byte("User1")},
			BlockTimestamp: time.Now().UnixNano() / 1e6,
		},
		Dag:            &commonPb.DAG{},
		AdditionalData: &commonPb.AdditionalData{},
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
	block.Header.PreConfHeight = preConfHeight
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

	return block, txRWSets
}

var testChainId = "testchainid_1"
var block0 = createConfigBlock(testChainId, 0, 0)
var block1, _ = createBlockAndRWSets(testChainId, 1, 10, 0)
var block2, _ = createBlockAndRWSets(testChainId, 2, 2, 0)
var block3, _ = createBlockAndRWSets(testChainId, 3, 2, 0)
var configBlock4 = createConfigBlock(testChainId, 4, 4)
var block5, _ = createBlockAndRWSets(testChainId, 5, 3, 4)
var blog binlog.BinLogger = binlog.NewMemBinlog(&test.GoLogger{})

func init5Blocks(db *BlockFileDB) {
	_ = commitBlock(db, block0)
	_ = commitBlock(db, block1)
	_ = commitBlock(db, block2)
	_ = commitBlock(db, block3)
	_ = commitBlock(db, configBlock4)
	_ = commitBlock(db, block5)
}
func commitBlock(db *BlockFileDB, block *commonPb.Block) error {
	data, bl, _ := serialization.SerializeBlock(&storePb.BlockWithRWSet{Block: block})
	fileName, offset, length, err := blog.Write(block.Header.BlockHeight+1, data)
	if err != nil {
		return err
	}
	bl.Index = &storePb.StoreInfo{
		StoreType: 0,
		FileName:  fileName,
		Offset:    offset,
		ByteLen:   length,
	}
	if block.Header.BlockHeight == 0 {
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

func initDb() *BlockFileDB {
	blog = binlog.NewMemBinlog(&test.GoLogger{})
	blockDB := NewBlockFileDB("test-chain", leveldbprovider.NewMemdbHandle(), logger, config1, blog)
	return blockDB
}

func TestBlockFileDB_GetTxWithBlockInfo(t *testing.T) {
	block := block1
	blockDB := NewBlockFileDB("test-chain", leveldbprovider.NewMemdbHandle(), logger, config1, blog)
	//_, sb, _ := serialization.SerializeBlock(&storePb.BlockWithRWSet{Block: block})
	_ = commitBlock(blockDB, block0)
	_ = commitBlock(blockDB, block)
	tx, err := blockDB.GetTxWithBlockInfo(block.Txs[1].Payload.TxId)
	assert.Nil(t, err)
	t.Logf("%+v", tx)
	assert.EqualValues(t, 1, tx.BlockHeight)
	assert.EqualValues(t, 1, tx.TxIndex)

	tx, err = blockDB.GetTxWithBlockInfo("i am test")
	assert.Nil(t, tx)
	assert.Equal(t, strings.Contains(err.Error(), "leveldb: not found"), true)
}

func generateBlockHash(chainId string, height uint64) []byte {
	blockHash := sha256.Sum256([]byte(fmt.Sprintf("%s-%d", chainId, height)))
	return blockHash[:]
}

func generateTxId(chainId string, height uint64, index int) string {
	txIdBytes := sha256.Sum256([]byte(fmt.Sprintf("%s-%d-%d", chainId, height, index)))
	return hex.EncodeToString(txIdBytes[:32])
}

//func TestBlockFileDB_parseTxIdBlockInfo(t *testing.T) {
//	value := constructTxIDBlockInfo(1, []byte("hash1"), 2, 0)
//	a, b, c, e := parseTxIdBlockInfo(value)
//	assert.Nil(t, e)
//	t.Log(a, b, c)
//	a, b, c, e = parseTxIdBlockInfo([]byte("bad data"))
//	assert.NotNil(t, e)
//	t.Log(a, b, c, e)
//}

//func TestBlockFileDB_GetArchivedPivot(t *testing.T) {
//	db := initDb()
//	defer db.Close()
//	init5Blocks(db)
//
//
//	archivedPivot, err := db.GetArchivedPivot()
//	assert.Equal(t, uint64(0), archivedPivot)
//	assert.Equal(t, strings.Contains(err.Error(), "leveldb: not found"), true)
//
//	err = db.dbHandle.Put([]byte(archivedPivotKey), constructBlockNumKey(db.dbHandle.GetDbType(), 10))
//	assert.Nil(t, err)
//	archivedPivot, err = db.GetArchivedPivot()
//	assert.Equal(t, uint64(10), archivedPivot)
//	assert.Nil(t, err)
//}
//
//func TestBlockFileDB_ShrinkBlocks(t *testing.T) {
//	db := initDb()
//	defer db.Close()
//	init5Blocks(db)
//
//
//	exist, err := db.BlockExists(block1.Header.BlockHash)
//	assert.Nil(t, err)
//	assert.True(t, exist)
//
//	txIdsMap, err := db.ShrinkBlocks(1, 5)
//	assert.Nil(t, err)
//	assert.Equal(t, len(txIdsMap[1]), 10)
//
//	txIdsMap, err = db.ShrinkBlocks(1, 10)
//	assert.Nil(t, txIdsMap)
//	assert.Equal(t, strings.Contains(err.Error(), "leveldb: not found"), true)
//
//	_, bl, _ := serialization.SerializeBlock(&storePb.BlockWithRWSet{Block: block1})
//	var arr []*serialization.BlockWithSerializedInfo
//	arr = append(arr, bl)
//	err = db.RestoreBlocks(arr)
//	assert.Nil(t, err)
//}

func TestBlockFileDB_GetBlockByHash(t *testing.T) {
	db := initDb()
	defer db.Close()
	init5Blocks(db)

	block, err := db.GetBlockByHash(block1.Header.BlockHash)
	assert.Nil(t, err)
	assert.Equal(t, block1.String(), block.String())

	block, err = db.GetBlockByHash([]byte("i am test"))
	assert.Nil(t, block)
	assert.Equal(t, strings.Contains(err.Error(), "leveldb: not found"), true)
}

func TestBlockFileDB_GetHeightByHash(t *testing.T) {
	db := initDb()
	defer db.Close()
	init5Blocks(db)

	height, err := db.GetHeightByHash(block1.Header.BlockHash)
	assert.Nil(t, err)
	assert.Equal(t, height, uint64(1))

	height, err = db.GetHeightByHash([]byte("i am test"))
	assert.Equal(t, height, uint64(0))
	assert.Equal(t, strings.Contains(err.Error(), "leveldb: not found"), true)
}

func TestBlockFileDB_GetBlockHeaderByHeight(t *testing.T) {
	db := initDb()
	defer db.Close()
	init5Blocks(db)

	header, err := db.GetBlockHeaderByHeight(3)
	assert.Nil(t, err)
	assert.Equal(t, header.String(), block3.Header.String())

	header, err = db.GetBlockHeaderByHeight(10)
	assert.Nil(t, header)
	assert.Equal(t, strings.Contains(err.Error(), "leveldb: not found"), true)
}

func TestBlockFileDB_GetBlock(t *testing.T) {
	db := initDb()
	defer db.Close()
	init5Blocks(db)

	block, err := db.GetBlock(3)
	assert.Nil(t, err)
	assert.Equal(t, block.String(), block3.String())
}

func TestBlockFileDB_GetLastBlock(t *testing.T) {
	db := initDb()
	//defer db.Close()
	init5Blocks(db)

	block, err := db.GetLastBlock()
	assert.Nil(t, err)
	assert.Equal(t, block.String(), block5.String())

	db.Close()
	block, err = db.GetLastBlock()
	assert.Nil(t, block)
	assert.Equal(t, strings.Contains(err.Error(), "closed"), true)
}

func TestBlockFileDB_GetLastConfigBlock(t *testing.T) {
	db := initDb()
	defer db.Close()
	//init5Blocks(db)

	_ = commitBlock(db, block0)
	_ = commitBlock(db, block1)
	_ = commitBlock(db, block2)
	_ = commitBlock(db, block3)

	block, err := db.GetLastConfigBlock()
	assert.Nil(t, err)
	assert.Equal(t, block.String(), block0.String())

	_ = commitBlock(db, configBlock4)
	_ = commitBlock(db, block5)

	block, err = db.GetLastConfigBlock()
	assert.Nil(t, err)
	assert.Equal(t, block.String(), configBlock4.String())
}

func TestBlockFileDB_GetBlockByTx(t *testing.T) {
	db := initDb()
	defer db.Close()
	init5Blocks(db)

	block, err := db.GetBlockByTx(block5.Txs[0].Payload.TxId)
	assert.Nil(t, err)
	assert.Equal(t, block5.String(), block.String())

	block, err = db.GetBlockByTx("i am test")
	assert.Nil(t, block)
	assert.Equal(t, strings.Contains(err.Error(), "leveldb: not found"), true)
}

func TestBlockFileDB_GetTxHeight(t *testing.T) {
	db := initDb()
	defer db.Close()
	init5Blocks(db)

	txHeight, err := db.GetTxHeight(block5.Txs[0].Payload.TxId)
	assert.Nil(t, err)
	assert.Equal(t, txHeight, uint64(5))

	txHeight, err = db.GetTxHeight("i am test")
	assert.Equal(t, txHeight, uint64(0))
	assert.Equal(t, strings.Contains(err.Error(), "leveldb: not found"), true)
}

func TestBlockFileDB_GetTx(t *testing.T) {
	db := initDb()
	defer db.Close()
	init5Blocks(db)

	tx, err := db.GetTx(block5.Txs[0].Payload.TxId)
	assert.Nil(t, err)
	assert.Equal(t, tx.String(), block5.Txs[0].String())

	tx, err = db.GetTx("i am test")
	assert.Nil(t, tx)
	assert.Equal(t, strings.Contains(err.Error(), "leveldb: not found"), true)
}

func TestBlockFileDB_TxExists(t *testing.T) {
	db := initDb()
	defer db.Close()
	init5Blocks(db)

	exist, err := db.TxExists(block5.Txs[0].Payload.TxId)
	assert.True(t, exist)
	assert.Nil(t, err)

	exist, err = db.TxExists("i am test")
	assert.False(t, exist)
	assert.Nil(t, err)
}

func TestBlockFileDB_TxArchived(t *testing.T) {
	//db := initDb()
	//defer db.Close()
	//init5Blocks(db)
	//
	//
	//exist, err := db.TxExists(block1.Txs[0].Payload.TxId)
	//assert.True(t, exist)
	//assert.Nil(t, err)
	//
	//_, _ = db.TxArchived(block1.Txs[0].Payload.TxId)
	//// todo houfa
	////assert.False(t, archived)
	////assert.Nil(t, err)
	//
	//txIdsMap, err := db.ShrinkBlocks(1, 5)
	//assert.Nil(t, err)
	//assert.Equal(t, len(txIdsMap[1]), 10)
	//
	//_, _ = db.TxArchived(block1.Txs[0].Payload.TxId)
	//assert.True(t, archived)
	//assert.Nil(t, err)
}

func TestBlockFileDB_GetTxConfirmedTime(t *testing.T) {
	db := initDb()
	defer db.Close()
	init5Blocks(db)

	time, err := db.GetTxConfirmedTime(block5.Txs[0].Payload.TxId)
	assert.Nil(t, err)
	assert.Equal(t, time, block5.Header.BlockTimestamp)

	time, err = db.GetTxConfirmedTime("i am test")
	assert.Equal(t, time, int64(0))
	assert.Equal(t, strings.Contains(err.Error(), "leveldb: not found"), true)
}

//func constructTxIDBlockInfo(height uint64, blockHash []byte, txIndex uint32, timestamp int64) []byte {
//	value := fmt.Sprintf("%d,%x,%d,%d", height, blockHash, txIndex, timestamp)
//	return []byte(value)
//}
//func BenchmarkBlockFileDB_constructTxIDBlockInfo(b *testing.B) {
//	hash := sha256.Sum256([]byte("hello"))
//	for i := 0; i < b.N; i++ {
//		constructTxIDBlockInfo(uint64(i), hash[:], uint32(i), int64(i))
//	}
//}
func BenchmarkBlockFileDB_TxInfo(b *testing.B) {
	hash := sha256.Sum256([]byte("hello"))
	for i := 0; i < b.N; i++ {
		txInfo := &storePb.TransactionStoreInfo{
			BlockHeight:    uint64(i),
			BlockHash:      hash[:],
			TxIndex:        uint32(i),
			BlockTimestamp: int64(i),
		}
		_, _ = txInfo.Marshal()
	}
}

func TestBlockFileDB_GetLastConfigBlockHeight(t *testing.T) {
	db := initDb()
	defer db.Close()
	init5Blocks(db)

	height, err := db.GetLastConfigBlockHeight()
	assert.Nil(t, err)
	assert.Equal(t, uint64(4), height)
}

func TestBlockFileDB_GetBlockIndex(t *testing.T) {
	db := initDb()
	defer db.Close()
	init5Blocks(db)

	blockIndex, err := db.GetBlockIndex(1)
	assert.Nil(t, err)
	t.Log(blockIndex)
}

func TestBlockFileDB_GetBlockMetaIndex(t *testing.T) {
	db := initDb()
	defer db.Close()
	init5Blocks(db)

	// 写入db时没有索引，所以查不到
	metaIndex, err := db.GetBlockMetaIndex(1)
	assert.Nil(t, err)
	t.Log(metaIndex)
}

func TestBlockFileDB_GetTxIndex(t *testing.T) {
	db := initDb()
	defer db.Close()
	init5Blocks(db)

	// 写入db时没有索引，所以查不到
	txIndex, err := db.GetBlockMetaIndex(1)
	assert.Nil(t, err)
	t.Log(txIndex)
}
