/*
 * Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
 *
 * SPDX-License-Identifier: Apache-2.0
 *
 */

package statekvdb

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

	"chainmaker.org/chainmaker/store/v2/cache"
	"chainmaker.org/chainmaker/store/v2/conf"
	"github.com/allegro/bigcache/v3"

	acPb "chainmaker.org/chainmaker/pb-go/v2/accesscontrol"
	commonPb "chainmaker.org/chainmaker/pb-go/v2/common"
	storePb "chainmaker.org/chainmaker/pb-go/v2/store"
	"chainmaker.org/chainmaker/pb-go/v2/syscontract"
	"chainmaker.org/chainmaker/protocol/v2"
	"chainmaker.org/chainmaker/protocol/v2/test"
	leveldbprovider "chainmaker.org/chainmaker/store-leveldb/v2"
	"chainmaker.org/chainmaker/store/v2/serialization"
	"github.com/stretchr/testify/assert"
)

var log = &test.GoLogger{}
var config1 = getSqlConfig()
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
func initStateKvDB() *StateKvDB {
	db := NewStateKvDB("chain1", initProvider(), log, config1)
	_, blockInfo0, _ := serialization.SerializeBlock(block0)
	_ = db.InitGenesis(blockInfo0)
	return db
}

//func TestMain(m *testing.M) {
//	fmt.Println("begin")
//	db, err := NewStateMysqlDB(testChainId, log)
//	if err != nil {
//		panic("faild to open mysql")
//	}
//	// clear data
//	stateMysqlDB := db.(*StateKvDB)
//	stateMysqlDB.db.Migrator().DropTable(&StateInfo{})
//	m.Run()
//	fmt.Println("end")
//}

func TestStateKvDB_CommitBlock(t *testing.T) {
	db := initStateKvDB()
	//block1.TxRWSets[0].TxWrites[0].ContractName = ""
	//block1.TxRWSets[0].TxWrites[0].Value = nil
	_, blockInfo1, _ := serialization.SerializeBlock(block1)
	err := db.CommitBlock(blockInfo1, true)
	assert.Nil(t, err)
	err = db.CommitBlock(blockInfo1, false)
	assert.Nil(t, err)
}

func TestStateKvDB_ReadObject(t *testing.T) {
	db := initStateKvDB()
	block1.TxRWSets[0].TxWrites[0].ContractName = ""
	_, blockInfo1, _ := serialization.SerializeBlock(block1)
	_ = db.CommitBlock(blockInfo1, true)
	_ = db.CommitBlock(blockInfo1, false)

	value, err := db.ReadObject(block1.TxRWSets[0].TxWrites[0].ContractName, block1.TxRWSets[0].TxWrites[0].Key)
	assert.Nil(t, err)
	assert.Equal(t, block1.TxRWSets[0].TxWrites[0].Value, value)
	t.Logf("%s", string(value))
	value, err = db.ReadObject(block1.TxRWSets[0].TxWrites[0].ContractName, []byte("another key"))
	assert.True(t, err != nil)
	assert.Nil(t, value)
}

//func TestStateKvDB_GetLastSavepoint(t *testing.T) {
//	db:=initStateKvDB()
//	height, err := db.GetLastSavepoint()
//	assert.Nil(t, err)
//	assert.Equal(t, uint64(block1.Block.Header.BlockHeight), height)
//
//	err = db.CommitBlock(block2)
//	assert.Nil(t, err)
//	height, err = db.GetLastSavepoint()
//	assert.Nil(t, err)
//	assert.Equal(t, uint64(block2.Block.Header.BlockHeight), height)
//
//}
//TODO Devin
//func TestStateKvDB_GetMemberExtraData(t *testing.T) {
//	db := initStateKvDB()
//	member := block1.Block.Txs[0].Sender.Signer
//	data, err := db.GetMemberExtraData(member)
//	assert.Equal(t, uint64(0), data.Sequence)
//	_, blockInfo1, _ := serialization.SerializeBlock(block1)
//	err = db.CommitBlock(blockInfo1)
//	assert.Nil(t, err)
//	data, err = db.GetMemberExtraData(member)
//	assert.Nil(t, err)
//	t.Logf("%v", data)
//}

//两个不同的db，返回的迭代器数据校验一致
func TestStateKvDB_SelectObject(t *testing.T) {
	//config := bigcache.Config{
	//	// number of shards (must be a power of 2)
	//	Shards: 1024,
	//	// time after which entry can be evicted
	//	LifeWindow: 10 * time.Minute,
	//	// rps * lifeWindow, used only in initial memory allocation
	//	MaxEntriesInWindow: 1000 * 10 * 60,
	//	// max entry size in bytes, used only in initial memory allocation
	//	MaxEntrySize: 500,
	//	// prints information about additional memory allocation
	//	Verbose: true,
	//	// cache will not allocate more memory than this limit, value in MB
	//	// if value is reached then the oldest entries can be overridden for the new ones
	//	// 0 value means no size limit
	//	HardMaxCacheSize: 8192,
	//	// callback fired when the oldest entry is removed because of its
	//	// expiration time or no space left for the new entry. Default value is nil which
	//	// means no callback and it prevents from unwrapping the oldest entry.
	//	OnRemove: nil,
	//}

	//bigCache, err := bigcache.NewBigCache(config)
	//if err != nil {
	//	fmt.Println("panic,when NewBigCache")
	//	panic(err)
	//}

	db := initStateKvDB()

	var block1 = createBlockAndRWSets(testChainId, 1, 10)
	contractName := testStr
	block1.TxRWSets[0].TxWrites[0].ContractName = contractName

	_, blockInfo1, _ := serialization.SerializeBlock(block1)
	_ = db.CommitBlock(blockInfo1, true)
	_ = db.CommitBlock(blockInfo1, false)

	startKey := fmt.Sprintf("key_%d", 2)
	//startValue := fmt.Sprintf("value_%d", 2)

	endKey := fmt.Sprintf("key_%d", 6)
	//endValue := fmt.Sprintf("value_%d", 6)

	wantIter, err := db.SelectObject(contractName, []byte(startKey), []byte(endKey))
	if err != nil {
		panic("test error,when db.SelectObject(")
	}

	type fields struct {
		dbHandle protocol.DBHandle
		cache    *cache.StoreCacheMgr
		logger   protocol.Logger
		//RWMutex         sync.RWMutex
		//bigCacheRWMutex sync.RWMutex
		//bigCache        *bigcache.BigCache
		storeConf *conf.StorageConfig
	}
	type args struct {
		contractName string
		startKey     []byte
		limit        []byte
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    protocol.StateIterator
		wantErr bool
	}{
		// TODO: Add test cases.
		{
			name: "t1",
			fields: fields{
				dbHandle: initProvider(),
				cache:    cache.NewStoreCacheMgr("test", 3, nil),
				//bigCache:  bigCache,
				logger:    log,
				storeConf: config1,
			},
			args: args{
				contractName: contractName,
				startKey:     []byte(startKey),
				limit:        []byte(endKey),
			},
			want:    wantIter,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &StateKvDB{
				dbHandle: tt.fields.dbHandle,
				cache:    tt.fields.cache,
				logger:   tt.fields.logger,
				RWMutex:  sync.RWMutex{},
				//bigCache:        tt.fields.bigCache,
				bigCacheRWMutex: sync.RWMutex{},
				storeConf:       tt.fields.storeConf,
			}
			var block1 = createBlockAndRWSets(testChainId, 1, 10)
			contractName := testStr
			block1.TxRWSets[0].TxWrites[0].ContractName = contractName

			_, blockInfo1, err := serialization.SerializeBlock(block1)
			if err != nil {
				t.Errorf("got.Next() got = %v, want %v", nil, err)
			}
			err = s.CommitBlock(blockInfo1, true)
			if err != nil {
				t.Errorf("got.Next() got = %v, want %v", nil, err)
			}
			err = s.CommitBlock(blockInfo1, false)
			if err != nil {
				t.Errorf("got.Next() got = %v, want %v", nil, err)
			}
			//fmt.Println(block1.TxRWSets[0].TxId)

			got, err := s.SelectObject(tt.args.contractName, tt.args.startKey, tt.args.limit)
			if (err != nil) != tt.wantErr {
				t.Errorf("SelectObject() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			defer got.Release()
			defer tt.want.Release()
			defer s.Close()

			//两个迭代器对比，如果都有next，并且key,value 一致，则结果正确
			for got.Next() && tt.want.Next() {
				valueGot, _ := got.Value()
				valueWant, _ := tt.want.Value()
				//对比  迭代器返回的 值，是否一一 对应
				fmt.Println("---------------")
				fmt.Println(string(valueGot.GetKey()), string(valueWant.GetKey()))
				fmt.Println(string(valueGot.GetValue()), string(valueWant.GetValue()))
				fmt.Println(valueGot.ContractName, valueWant.ContractName)

				if valueGot.ContractName != valueWant.ContractName ||
					string(valueGot.GetKey()) != string(valueWant.GetKey()) ||
					string(valueGot.GetValue()) != string(valueWant.GetValue()) {
					t.Errorf("SelectObject() got = %v, want %v", got, tt.want)
				}
			}

		})
	}
}

func TestStateKvDB_GetLastSavepoint(t *testing.T) {
	config := bigcache.Config{
		// number of shards (must be a power of 2)
		Shards: 1024,
		// time after which entry can be evicted
		LifeWindow: 10 * time.Minute,
		// rps * lifeWindow, used only in initial memory allocation
		MaxEntriesInWindow: 1000 * 10 * 60,
		// max entry size in bytes, used only in initial memory allocation
		MaxEntrySize: 500,
		// prints information about additional memory allocation
		Verbose: true,
		// cache will not allocate more memory than this limit, value in MB
		// if value is reached then the oldest entries can be overridden for the new ones
		// 0 value means no size limit
		HardMaxCacheSize: 8192,
		// callback fired when the oldest entry is removed because of its
		// expiration time or no space left for the new entry. Default value is nil which
		// means no callback and it prevents from unwrapping the oldest entry.
		OnRemove: nil,
	}

	bigCache, err := bigcache.NewBigCache(config)
	if err != nil {
		fmt.Println("panic,when NewBigCache")
		panic(err)
	}
	type fields struct {
		dbHandle protocol.DBHandle
		cache    *cache.StoreCacheMgr
		logger   protocol.Logger
		//RWMutex  sync.RWMutex
		bigCache *bigcache.BigCache
		//bigCacheRWMutex sync.RWMutex
		storeConf *conf.StorageConfig
	}
	tests := []struct {
		name    string
		fields  fields
		want    uint64
		wantErr bool
	}{
		// TODO: Add test cases.
		{
			name: "GetLastSavepoint1",
			fields: fields{
				dbHandle:  initProvider(),
				cache:     cache.NewStoreCacheMgr(testChainId, 3, log),
				bigCache:  bigCache,
				logger:    log,
				storeConf: config1,
			},
			want:    1,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &StateKvDB{
				dbHandle:        tt.fields.dbHandle,
				cache:           tt.fields.cache,
				logger:          tt.fields.logger,
				RWMutex:         sync.RWMutex{},
				bigCacheRWMutex: sync.RWMutex{},
				storeConf:       tt.fields.storeConf,
			}
			//写入2个块，1个创世块，1个普通块
			_, blockInfo0, _ := serialization.SerializeBlock(block0)
			_ = b.InitGenesis(blockInfo0)

			var block1 = createBlockAndRWSets(testChainId, 1, 10)
			contractName := testStr
			block1.TxRWSets[0].TxWrites[0].ContractName = contractName

			_, blockInfo1, err := serialization.SerializeBlock(block1)
			if err != nil {
				t.Errorf("got.Next() got = %v, want %v", nil, err)
			}
			err = b.CommitBlock(blockInfo1, true)
			if err != nil {
				t.Errorf("got.Next() got = %v, want %v", nil, err)
			}
			err = b.CommitBlock(blockInfo1, false)
			if err != nil {
				t.Errorf("got.Next() got = %v, want %v", nil, err)
			}

			//获得 lastSavepoint
			got, err := b.GetLastSavepoint()
			if (err != nil) != tt.wantErr {
				t.Errorf("GetLastSavepoint() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("GetLastSavepoint() got = %v, want %v", got, tt.want)
			}
		})
	}
}

//测试读缓存的case
func TestStateKvDB_ReadObject1(t *testing.T) {
	config := bigcache.Config{
		Shards:             1024,
		LifeWindow:         3 * time.Second,
		CleanWindow:        1 * time.Second,
		MaxEntriesInWindow: 1,
		MaxEntrySize:       500,
		HardMaxCacheSize:   4,
	}
	configWithCache := getSqlConfig()
	configWithCache.DisableStateCache = false

	bigCache, err := bigcache.NewBigCache(config)
	if err != nil {
		fmt.Println("panic,when NewBigCache")
		panic(err)
	}
	contractName := testStr
	//		key := fmt.Sprintf("key_%d", i)
	//		value := fmt.Sprintf("value_%d", i)
	inputKey := []byte(fmt.Sprintf("key_%d", 3))
	inputValue := fmt.Sprintf("value_%d", 3)

	bigConf := &conf.CacheConfig{
		LifeWindow:       3 * time.Second,
		CleanWindow:      1 * time.Second,
		MaxEntrySize:     500,
		HardMaxCacheSize: 4,
	}
	config2 := bigcache.Config{
		Shards:             1,
		LifeWindow:         3 * time.Second,
		CleanWindow:        1 * time.Second,
		MaxEntriesInWindow: 1,
		MaxEntrySize:       10,
		HardMaxCacheSize:   4,
	}
	configWithCache2 := getSqlConfig()
	configWithCache2.DisableStateCache = false
	configWithCache2.StateCache = bigConf

	bigCache2, err := bigcache.NewBigCache(config2)
	if err != nil {
		fmt.Println("panic,when NewBigCache")
		panic(err)
	}

	type fields struct {
		dbHandle protocol.DBHandle
		cache    *cache.StoreCacheMgr
		logger   protocol.Logger
		//RWMutex  sync.RWMutex
		bigCache *bigcache.BigCache
		//bigCacheRWMutex sync.RWMutex
		storeConf *conf.StorageConfig
	}
	type args struct {
		contractName string
		key          []byte
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    []byte
		wantErr bool
	}{
		// TODO: Add test cases.
		{
			name: "readobject1",
			fields: fields{
				dbHandle:  initProvider(),
				cache:     cache.NewStoreCacheMgr(testChainId, 3, log),
				bigCache:  bigCache,
				logger:    log,
				storeConf: config1,
			},
			args: args{
				contractName: contractName,
				key:          inputKey,
			},
			want:    []byte(inputValue),
			wantErr: false,
		},
		{
			//模拟key写入 cache 后，3秒后过期被删除的情况，这种情况，读cache未命中，然后读db
			name: "key-not-in-bigcache-but-in-db", //key not in cache,but in db
			fields: fields{
				dbHandle:  initProvider(),
				cache:     cache.NewStoreCacheMgr(testChainId, 3, log),
				bigCache:  bigCache,
				logger:    log,
				storeConf: configWithCache,
			},
			args: args{
				contractName: contractName,
				key:          inputKey,
			},
			want:    []byte(inputValue),
			wantErr: false,
		},
		{
			name: "storeConfig.StateCache is not nil",
			fields: fields{
				dbHandle:  initProvider(),
				cache:     cache.NewStoreCacheMgr(testChainId, 3, log),
				bigCache:  bigCache2,
				logger:    log,
				storeConf: configWithCache,
			},
			args: args{
				contractName: contractName,
				key:          inputKey,
			},
			want:    []byte(inputValue),
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &StateKvDB{
				dbHandle:        cache.NewCacheWrapToDBHandle(tt.fields.bigCache, tt.fields.dbHandle, tt.fields.logger),
				cache:           tt.fields.cache,
				logger:          tt.fields.logger,
				RWMutex:         sync.RWMutex{},
				bigCacheRWMutex: sync.RWMutex{},
				storeConf:       tt.fields.storeConf,
			}
			//写入2个块，1个创世块，1个普通块
			_, blockInfo0, _ := serialization.SerializeBlock(block0)
			_ = b.InitGenesis(blockInfo0)

			var block1 = createBlockAndRWSets(testChainId, 1, 10)
			//block1.TxRWSets[0].TxWrites[0].ContractName = contractName

			_, blockInfo1, err := serialization.SerializeBlock(block1)
			if err != nil {
				t.Errorf("got.Next() got = %v, want %v", nil, err)
			}
			err = b.CommitBlock(blockInfo1, true)
			if err != nil {
				t.Errorf("got.Next() got = %v, want %v", nil, err)
			}
			err = b.CommitBlock(blockInfo1, false)
			if err != nil {
				t.Errorf("got.Next() got = %v, want %v", nil, err)
			}

			//让key在cache中ttl过期被删除
			if tt.name == "key-not-in-bigcache-but-in-db" {
				time.Sleep(4 * time.Second)
			}
			got, err := b.ReadObject(tt.args.contractName, tt.args.key)
			if (err != nil) != tt.wantErr {
				t.Errorf("ReadObject() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ReadObject() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewStateKvDB(t *testing.T) {
	bigConf := &conf.CacheConfig{
		LifeWindow:       3 * time.Second,
		CleanWindow:      1 * time.Second,
		MaxEntrySize:     500,
		HardMaxCacheSize: 4,
	}

	configWithCache2 := getSqlConfig()
	configWithCache2.DisableStateCache = false
	configWithCache2.StateCache = bigConf

	db := NewStateKvDB("chain1", initProvider(), log, configWithCache2)
	_, blockInfo0, _ := serialization.SerializeBlock(block0)
	_ = db.InitGenesis(blockInfo0)

	key := fmt.Sprintf("key_%d", 0)
	//value := fmt.Sprintf("value_%d", 0)

	type args struct {
		chainId     string
		handle      protocol.DBHandle
		logger      protocol.Logger
		storeConfig *conf.StorageConfig
	}
	tests := []struct {
		name    string
		args    args
		want    *StateKvDB
		wantErr bool
	}{
		// TODO: Add test cases.
		{
			name: "NewStateKvDB",
			args: args{
				chainId:     "chain1",
				handle:      initProvider(),
				logger:      log,
				storeConfig: configWithCache2,
			},
			want:    db,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewStateKvDB(tt.args.chainId, tt.args.handle, tt.args.logger, tt.args.storeConfig)
			_, blockInfo0, _ := serialization.SerializeBlock(block0)
			_ = got.InitGenesis(blockInfo0)

			getValue, _ := got.get([]byte(key))

			wantValue, _ := tt.want.get([]byte(key))

			if (string(getValue) != string(wantValue)) != tt.wantErr {

				t.Errorf("NewStateKvDB() = %v, want %v", getValue, wantValue)
			}

		})
	}
}

func TestStateKvDB_DirectFlushDB(t *testing.T) {

	config := bigcache.Config{
		Shards:             1024,
		LifeWindow:         3 * time.Second,
		CleanWindow:        1 * time.Second,
		MaxEntriesInWindow: 1,
		MaxEntrySize:       500,
		HardMaxCacheSize:   4,
	}
	configWithCache := getSqlConfig()
	configWithCache.DisableStateCache = false

	bigCache, _ := bigcache.NewBigCache(config)

	mp := make(map[string][]byte)
	for i := 11; i < 20; i++ {
		inputKey := []byte(fmt.Sprintf("key_%d", i))
		inputValue := fmt.Sprintf("value_%d", i)
		mp[string(inputKey)] = []byte(inputValue)
	}

	type fields struct {
		dbHandle protocol.DBHandle
		cache    *cache.StoreCacheMgr
		logger   protocol.Logger
		//RWMutex         sync.RWMutex
		bigCache *bigcache.BigCache
		//bigCacheRWMutex sync.RWMutex
		storeConf *conf.StorageConfig
	}
	type args struct {
		keyMap map[string][]byte
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
		{
			name: "DirectFlushDB",
			fields: fields{
				dbHandle:  initProvider(),
				cache:     cache.NewStoreCacheMgr(testChainId, 3, log),
				bigCache:  bigCache,
				logger:    log,
				storeConf: configWithCache,
			},
			args:    args{keyMap: mp},
			wantErr: false,
		},
		{
			name: "DirectFlushDBErr",
			fields: fields{
				dbHandle:  initProvider(),
				cache:     cache.NewStoreCacheMgr(testChainId, 3, log),
				bigCache:  bigCache,
				logger:    log,
				storeConf: configWithCache,
			},
			args:    args{keyMap: mp},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &StateKvDB{
				dbHandle:        tt.fields.dbHandle,
				cache:           tt.fields.cache,
				logger:          tt.fields.logger,
				RWMutex:         sync.RWMutex{},
				bigCacheRWMutex: sync.RWMutex{},
				storeConf:       tt.fields.storeConf,
			}
			_, blockInfo0, _ := serialization.SerializeBlock(block0)
			_ = s.InitGenesis(blockInfo0)
			//模拟写db错误
			if tt.name == "DirectFlushDBErr" {
				s.Close()
			}
			if err := s.DirectFlushDB(tt.args.keyMap); (err != nil) != tt.wantErr {
				t.Errorf("DirectFlushDB() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func BenchmarkCopyBytes(b *testing.B) {
	input := make([]byte, 1024*1024)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		copyBytes(input)
	}

}

func BenchmarkCopyBytes2(b *testing.B) {
	input := make([]byte, 1024*1024)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		var out []byte
		out = append(out, input...)
	}
}
func BenchmarkCopyBytes3(b *testing.B) {
	input := make([]byte, 1024*1024)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		out := make([]byte, len(input))
		for x := 0; x < len(input); x++ {
			out[x] = input[x]
		}

	}
}
