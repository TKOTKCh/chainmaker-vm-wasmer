/*
 * Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
 *
 * SPDX-License-Identifier: Apache-2.0
 *
 */

package txexistkvdb

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sync"
	"testing"
	"time"

	"chainmaker.org/chainmaker/protocol/v2"
	leveldbprovider "chainmaker.org/chainmaker/store-leveldb/v2"
	"chainmaker.org/chainmaker/store/v2/serialization"
	"github.com/gogo/protobuf/proto"

	"chainmaker.org/chainmaker/store/v2/conf"

	acPb "chainmaker.org/chainmaker/pb-go/v2/accesscontrol"
	commonPb "chainmaker.org/chainmaker/pb-go/v2/common"
	storePb "chainmaker.org/chainmaker/pb-go/v2/store"
	"chainmaker.org/chainmaker/pb-go/v2/syscontract"
	"chainmaker.org/chainmaker/protocol/v2/test"
)

var log = &test.GoLogger{}
var testStr = "contract1"

//var configBadger = getBadgerConfig("")

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

//生成测试用的blockHash
func generateBlockHash(chainId string, height uint64) []byte {
	blockHash := sha256.Sum256([]byte(fmt.Sprintf("%s-%d", chainId, height)))
	return blockHash[:]
}

//生成测试用的txid
func generateTxId(chainId string, height uint64, index int) string {
	txIdBytes := sha256.Sum256([]byte(fmt.Sprintf("%s-%d-%d", chainId, height, index)))
	return hex.EncodeToString(txIdBytes[:32])
}

func createConfigBlock(chainId string, height uint64) *storePb.BlockWithRWSet {
	block := &commonPb.Block{
		Header: &commonPb.BlockHeader{
			ChainId:     chainId,
			BlockHeight: height,
			Proposer: &acPb.Member{
				OrgId:      "org1",
				MemberInfo: []byte("User1"),
				MemberType: 0,
			},
		},
		Txs: []*commonPb.Transaction{
			{
				Payload: &commonPb.Payload{
					ChainId:      chainId,
					TxType:       commonPb.TxType_INVOKE_CONTRACT,
					ContractName: syscontract.SystemContract_CHAIN_CONFIG.String(),
				},
				Sender: &commonPb.EndorsementEntry{
					Signer: &acPb.Member{
						OrgId:      "org1",
						MemberInfo: []byte("User1"),
					},
					Signature: []byte("signature1"),
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
	var txRWSets []*commonPb.TxRWSet
	for i := 0; i < 1; i++ {
		key := fmt.Sprintf("key_%d", i)
		value := fmt.Sprintf("value_%d", i)
		txRWset := &commonPb.TxRWSet{
			TxId: block.Txs[i].Payload.TxId,
			TxWrites: []*commonPb.TxWrite{
				{
					Key:          []byte(key),
					Value:        []byte(value),
					ContractName: testStr,
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

func createBlockAndRWSets(chainId string, height uint64, txNum int) *storePb.BlockWithRWSet {
	block := &commonPb.Block{
		Header: &commonPb.BlockHeader{
			ChainId:     chainId,
			BlockHeight: height,
			Proposer: &acPb.Member{
				OrgId:      "org1",
				MemberInfo: []byte("User1"),
				MemberType: 0,
			},
		},
	}

	for i := 0; i < txNum; i++ {
		tx := &commonPb.Transaction{
			Payload: &commonPb.Payload{
				ChainId:  chainId,
				TxId:     generateTxId(chainId, height, i),
				Sequence: uint64(i),
			},
			Sender: &commonPb.EndorsementEntry{
				Signer: &acPb.Member{
					OrgId:      "org1",
					MemberInfo: []byte("User1"),
				},
				Signature: []byte("signature1"),
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
					ContractName: testStr,
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

//var block2 = createBlockAndRWSets(testChainId, 2, 2)

/*var block3, _ = createBlockAndRWSets(testChainId, 3, 2)
var configBlock4 = createConfigBlock(testChainId, 4)
var block5, _ = createBlockAndRWSets(testChainId, 5, 3)*/

//func createBlock(chainId string, height uint64) *commonPb.Block {
//	block := &commonPb.Block{
//		Header: &commonPb.BlockHeader{
//			ChainId:     chainId,
//			BlockHeight: height,
//		},
//		Txs: []*commonPb.Transaction{
//			{
//				Payload: &commonPb.Payload{
//					ChainId: chainId,
//				},
//				Sender: &commonPb.EndorsementEntry{
//					Signer: &acPb.Member{
//						OrgId:      "org1",
//						MemberInfo: []byte("User1"),
//					},
//					Signature: []byte("signature1"),
//				},
//				Result: &commonPb.Result{
//					Code: commonPb.TxStatusCode_SUCCESS,
//					ContractResult: &commonPb.ContractResult{
//						Result: []byte("ok"),
//					},
//				},
//			},
//		},
//	}
//
//	block.Header.BlockHash = generateBlockHash(chainId, height)
//	block.Txs[0].Payload.TxId = generateTxId(chainId, height, 0)
//	return block
//}

func initProvider() protocol.DBHandle {
	p := leveldbprovider.NewMemdbHandle()
	return p
}
func initTxExistKvDB() *TxExistKvDB {
	db := NewTxExistKvDB("chain1", initProvider(), log)
	_, blockInfo0, _ := serialization.SerializeBlock(block0)
	_ = db.InitGenesis(blockInfo0)
	return db
}

func TestNewTxExistKvDB(t *testing.T) {
	type args struct {
		chainId  string
		dbHandle protocol.DBHandle
		logger   protocol.Logger
	}
	tests := []struct {
		name string
		args args
		want *TxExistKvDB
	}{
		// TODO: Add test cases.
		{
			name: "new-tx-exist-kvdb",
			args: args{
				chainId:  "chain1",
				dbHandle: initProvider(),
				logger:   log,
			},
			want: initTxExistKvDB(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewTxExistKvDB(tt.args.chainId, tt.args.dbHandle, tt.args.logger)
			_, blockInfo0, _ := serialization.SerializeBlock(block0)
			_ = got.InitGenesis(blockInfo0)
			gotLast, _ := got.GetLastSavepoint()
			wantLast, _ := tt.want.GetLastSavepoint()

			if !reflect.DeepEqual(gotLast, wantLast) {
				t.Errorf("NewTxExistKvDB() = %v, want %v", gotLast, wantLast)
			}
		})
	}
}

func TestTxExistKvDB_Close(t1 *testing.T) {
	type fields struct {
		dbHandle protocol.DBHandle
		logger   protocol.Logger
		//batchPool sync.Pool
	}
	tests := []struct {
		name   string
		fields fields
	}{
		// TODO: Add test cases.
		{
			name: "dbclose",
			fields: fields{
				dbHandle: initProvider(),
				logger:   log,
				//batchPool: sync.Pool{},
			},
		},
	}
	for _, tt := range tests {
		t1.Run(tt.name, func(t1 *testing.T) {
			//t := &TxExistKvDB{
			//	dbHandle:  tt.fields.dbHandle,
			//	logger:    tt.fields.logger,
			//	RWMutex:     sync.RWMutex{},
			//	batchPool: tt.fields.batchPool,
			//}
			t := NewTxExistKvDB(tt.name, tt.fields.dbHandle, tt.fields.logger)
			_, blockInfo0, _ := serialization.SerializeBlock(block0)
			_ = t.InitGenesis(blockInfo0)
			t.Close()

		})
	}
}

func TestTxExistKvDB_CommitBlock(t1 *testing.T) {
	_, blockInfo0, _ := serialization.SerializeBlock(block0)
	testCaseBatchPoolGetFail := "commit-block-fail-t.batchPool.Get()"
	writeBatchFail := "WriteBatchFail"
	putDbFail := "putDbFail"

	type fields struct {
		dbHandle protocol.DBHandle
		logger   protocol.Logger
		//batchPool sync.Pool
	}
	type args struct {
		blockInfo *serialization.BlockWithSerializedInfo
		isCache   bool
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
		{
			name: "commitblock",
			fields: fields{
				dbHandle: initProvider(),
				logger:   log,
				//batchPool: sync.Pool{},
			},
			args: args{
				blockInfo: blockInfo0,
				isCache:   true,
			},
			wantErr: false,
		},
		{
			name: testCaseBatchPoolGetFail,
			fields: fields{
				dbHandle: initProvider(),
				logger:   log,
				//batchPool: sync.Pool{},
			},
			args: args{
				blockInfo: blockInfo0,
				isCache:   true,
			},
			wantErr: true,
		},
		{
			name: writeBatchFail,
			fields: fields{
				dbHandle: initProvider(),
				logger:   log,
				//batchPool: sync.Pool{},
			},
			args: args{
				blockInfo: blockInfo0,
				isCache:   true,
			},
			wantErr: true,
		},
		{
			name: putDbFail,
			fields: fields{
				dbHandle: initProvider(),
				logger:   log,
				//batchPool: sync.Pool{},
			},
			args: args{
				blockInfo: blockInfo0,
				isCache:   true,
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t1.Run(tt.name, func(t1 *testing.T) {
			//t := &TxExistKvDB{
			//	dbHandle:  tt.fields.dbHandle,
			//	logger:    tt.fields.logger,
			//	RWMutex:   sync.RWMutex{},
			//	batchPool: tt.fields.batchPool,
			//}
			t := NewTxExistKvDB(tt.name, tt.fields.dbHandle, tt.fields.logger)

			//模拟提交失败 t.batchPool.Get() Error
			if tt.name == testCaseBatchPoolGetFail {
				t = &TxExistKvDB{
					dbHandle:  tt.fields.dbHandle,
					logger:    tt.fields.logger,
					RWMutex:   sync.RWMutex{},
					batchPool: sync.Pool{},
				}
			}
			//模拟提交失败 writeBatch Error
			if tt.name == writeBatchFail {
				tt.fields.dbHandle.Close()
			}
			//模拟提交失败 writeBatch Error
			if tt.name == putDbFail {
				tt.fields.dbHandle.Close()
			}

			if err := t.CommitBlock(tt.args.blockInfo, tt.args.isCache); (err != nil) != tt.wantErr {
				t1.Errorf("CommitBlock() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestTxExistKvDB_GetLastSavepoint(t1 *testing.T) {
	getLastSavepoint := "get-last-savepoint"
	getLastSavepointErr := "get-last-savepoint-err"

	type fields struct {
		dbHandle protocol.DBHandle
		logger   protocol.Logger
		//batchPool sync.Pool
	}
	tests := []struct {
		name    string
		fields  fields
		want    uint64
		wantErr bool
	}{
		// TODO: Add test cases.
		{
			name: getLastSavepoint,
			fields: fields{
				dbHandle: initProvider(),
				logger:   log,
				//batchPool: sync.Pool{},
			},
			want:    1,
			wantErr: false,
		},
		{
			name: getLastSavepointErr,
			fields: fields{
				dbHandle: initProvider(),
				logger:   log,
				//batchPool: sync.Pool{},
			},
			want:    0,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t1.Run(tt.name, func(t1 *testing.T) {
			//t := &TxExistKvDB{
			//	dbHandle:  tt.fields.dbHandle,
			//	logger:    tt.fields.logger,
			//	RWMutex:   sync.RWMutex{},
			//	batchPool: tt.fields.batchPool,
			//}
			t := NewTxExistKvDB(tt.name, tt.fields.dbHandle, tt.fields.logger)
			_, blockInfo0, _ := serialization.SerializeBlock(block0)
			_ = t.InitGenesis(blockInfo0)

			_, blockInfo1, _ := serialization.SerializeBlock(block1)
			t.CommitBlock(blockInfo1, true)

			//模拟dbHandle.Get 返回err的情况
			if tt.name == getLastSavepointErr {
				t.dbHandle.Close()
			}
			got, err := t.GetLastSavepoint()
			if (err != nil) != tt.wantErr {
				t1.Errorf("GetLastSavepoint() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t1.Errorf("GetLastSavepoint() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTxExistKvDB_InitGenesis(t1 *testing.T) {
	_, blockInfo0, _ := serialization.SerializeBlock(block0)
	type fields struct {
		dbHandle protocol.DBHandle
		logger   protocol.Logger
		//batchPool sync.Pool
	}
	type args struct {
		genesisBlock *serialization.BlockWithSerializedInfo
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
		{
			name: "initGenesis",
			fields: fields{
				dbHandle: initProvider(),
				logger:   log,
				//batchPool: sync.Pool{},
			},
			args: args{
				genesisBlock: blockInfo0,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t1.Run(tt.name, func(t1 *testing.T) {
			//t := &TxExistKvDB{
			//	dbHandle:  tt.fields.dbHandle,
			//	logger:    tt.fields.logger,
			//	RWMutex:   tt.fields.RWMutex,
			//	batchPool: tt.fields.batchPool,
			//}
			t := NewTxExistKvDB(tt.name, tt.fields.dbHandle, tt.fields.logger)
			if err := t.InitGenesis(tt.args.genesisBlock); (err != nil) != tt.wantErr {
				t1.Errorf("InitGenesis() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestTxExistKvDB_TxExist(t1 *testing.T) {
	inputTxID := block1.Block.Txs[0].Payload.TxId
	type fields struct {
		dbHandle protocol.DBHandle
		logger   protocol.Logger
		//batchPool sync.Pool
	}
	type args struct {
		txId string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    bool
		wantErr bool
	}{
		// TODO: Add test cases.
		{
			name: "txExist",
			fields: fields{
				dbHandle: initProvider(),
				logger:   log,
				//batchPool: sync.Pool{},
			},
			args: args{
				txId: inputTxID,
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "tx-not-exist",
			fields: fields{
				dbHandle: initProvider(),
				logger:   log,
				//batchPool: sync.Pool{},
			},
			args: args{
				txId: "not-exist-txid",
			},
			want:    false,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t1.Run(tt.name, func(t1 *testing.T) {
			//t := &TxExistKvDB{
			//	dbHandle:  tt.fields.dbHandle,
			//	logger:    tt.fields.logger,
			//	RWMutex:   sync.RWMutex{},
			//	batchPool: tt.fields.batchPool,
			//}
			t := NewTxExistKvDB(tt.name, tt.fields.dbHandle, tt.fields.logger)
			_, blockInfo0, _ := serialization.SerializeBlock(block0)
			_ = t.InitGenesis(blockInfo0)

			_, blockInfo1, _ := serialization.SerializeBlock(block1)
			t.CommitBlock(blockInfo1, false)

			got, err := t.TxExists(tt.args.txId)
			if (err != nil) != tt.wantErr {
				t1.Errorf("TxExist() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t1.Errorf("TxExist() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_constructBlockNumKey(t *testing.T) {
	c1 := rune(1)
	//fmt.Println( []byte(string(c1)))
	type args struct {
		blockNum uint64
	}
	tests := []struct {
		name string
		args args
		want []byte
	}{
		// TODO: Add test cases.
		{
			name: "constructBlockNumKeyTest",
			args: args{
				blockNum: 1,
			},
			want: []byte("n" + string(c1)),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := constructBlockNumKey(tt.args.blockNum); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("constructBlockNumKey() = %v, want %v", got, tt.want)
				t.Errorf("constructBlockNumKey() = %s, want %s", got, tt.want)
			}
		})
	}
}

func Test_constructTxIDKey(t *testing.T) {
	inputTxID := generateTxId("newTxidtest", 6, 8)
	type args struct {
		txId string
	}
	tests := []struct {
		name string
		args args
		want []byte
	}{
		// TODO: Add test cases.
		{
			name: "constructTxIDKeyTest",
			args: args{
				txId: inputTxID,
			},
			want: []byte("t72af69ad8195f1e96758b85009ebd6018a4d31fee68fd53c2195b3c7f265f253"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := constructTxIDKey(tt.args.txId); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("constructTxIDKey() = %v, want %v", got, tt.want)
				t.Errorf("constructTxIDKey() = %s, want %s", got, tt.want)
			}
		})
	}
}

func Test_encodeBlockNum(t *testing.T) {
	var inputInt uint64 = 10
	inputByte := proto.EncodeVarint(inputInt)

	type args struct {
		blockNum uint64
	}
	tests := []struct {
		name string
		args args
		want []byte
	}{
		// TODO: Add test cases.
		{
			name: "encodeBlockNum",
			args: args{
				blockNum: inputInt,
			},
			want: inputByte,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := encodeBlockNum(tt.args.blockNum); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("encodeBlockNum() = %v, want %v", got, tt.want)
			}
		})
	}
}
