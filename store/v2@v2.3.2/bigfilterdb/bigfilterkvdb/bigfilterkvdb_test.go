/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0

*/
package bigfilterkvdb

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strconv"
	"sync"
	"testing"
	"time"

	redisbloom "github.com/RedisBloom/redisbloom-go"

	acPb "chainmaker.org/chainmaker/pb-go/v2/accesscontrol"
	commonPb "chainmaker.org/chainmaker/pb-go/v2/common"
	storePb "chainmaker.org/chainmaker/pb-go/v2/store"
	"chainmaker.org/chainmaker/pb-go/v2/syscontract"
	"chainmaker.org/chainmaker/protocol/v2"
	"chainmaker.org/chainmaker/protocol/v2/test"
	"chainmaker.org/chainmaker/store/v2/bigfilterdb/filter"
	"chainmaker.org/chainmaker/store/v2/conf"
	"chainmaker.org/chainmaker/store/v2/serialization"
	"chainmaker.org/chainmaker/store/v2/types"
)

const (
	redisBloomName = "bigfilter_"
)

var (
	//redis端口和个数 要和 start_redis.sh stop_redis.sh 保持一致
	redisServerCluster = []string{"127.0.0.1:7500", "127.0.0.1:7501", "127.0.0.1:7502"}
	redisPassword      *string
	logger             = &test.GoLogger{}
	testStr            = "contract1"
	testChainId        = "testchainid_1"
	block0             = createConfigBlock(testChainId, 0)
	block1             = createBlockAndRWSets(testChainId, 1, 10)
	txId1              = "0000000000-abcdfjelf-as88"
	txId2              = "1111111111-abcdfjelf-as88"
)

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

func initRedisHosts() {
	cmd := exec.Command("/bin/bash", "./start_redis.sh")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		fmt.Println("start redis error :", err)
	}
	//outStr, errStr := string(stdout.Bytes()), string(stderr.Bytes())
	outStr, errStr := stdout.String(), stderr.String()
	fmt.Printf("out:%s\n", outStr)
	if errStr != "" {
		fmt.Printf("err:%s\n", errStr)
	}
}

func destroyRedisHosts() {
	cmd := exec.Command("/bin/bash", "./stop_redis.sh")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		fmt.Println("stop redis error: ", err)
	}
	//outStr, errStr := string(stdout.Bytes()), string(stderr.Bytes())
	outStr, errStr := stdout.String(), stderr.String()
	fmt.Printf("out:%s\n", outStr)
	if errStr != "" {
		fmt.Printf("err:%s\n", errStr)
	}

}

func createRedisClient() []*redisbloom.Client {
	filters := []*redisbloom.Client{}

	for i := 0; i < len(redisServerCluster); i++ {
		//把过滤器，映射到对应的redisServerCluster 中，一共n个redis
		redisNum := i % (len(redisServerCluster))
		bloomFilter := redisbloom.NewClient(redisServerCluster[redisNum], "nohelp", redisPassword)
		//创建redisBloom 过滤器
		idxStr := strconv.Itoa(i)
		bloomName := redisBloomName + idxStr

		//filter已存在则创建失败，不会重新创建
		err := bloomFilter.Reserve(bloomName, 0.1, 100*100*100)
		if err != nil {
			logger.Infof("create filter [%s] in redis server [%s] err:", bloomName,
				redisServerCluster[redisNum], err)
		}
		filters = append(filters, bloomFilter)
	}
	return filters

}
func createFilter() filter.Filter {
	f, _ := filter.NewBigFilter(3, 100*1000, 0.00001, redisServerCluster, redisPassword, "chainID2", logger)
	return f
}

func TestNewBigFilterKvDB(t *testing.T) {
	initRedisHosts()
	defer destroyRedisHosts()

	type args struct {
		n                  int
		m                  uint
		fpRate             float64
		logger             protocol.Logger
		redisServerCluster []string
		pass               *string
	}
	tests := []struct {
		name    string
		args    args
		want    *BigFilterKvDB
		wantErr bool
	}{
		// TODO: Add test cases.
		{
			name: "new_big_filter",
			args: args{
				n:                  3,
				m:                  10 * 100 * 100,
				fpRate:             0.00001,
				logger:             logger,
				redisServerCluster: redisServerCluster,
				pass:               redisPassword,
			},
			want:    nil,
			wantErr: false,
		},
		{
			name: "new_big_filter_err",
			args: args{
				n:                  3,
				m:                  10 * 100 * 100,
				fpRate:             0.00001,
				logger:             logger,
				redisServerCluster: redisServerCluster,
				pass:               redisPassword,
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.name == "new_big_filter_err" { //关闭docker redis
				destroyRedisHosts()
			}
			_, err := NewBigFilterKvDB(tt.args.n, tt.args.m, tt.args.fpRate, tt.args.logger, tt.args.redisServerCluster, tt.args.pass, "chainID1")
			if (err != nil) != tt.wantErr {
				t.Errorf("NewBigFilterKvDB() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

		})
	}

}

func TestBigFilterKvDB_InitGenesis(t1 *testing.T) {
	initRedisHosts()
	_, blockInfo0, _ := serialization.SerializeBlock(block0)

	type fields struct {
		filter filter.Filter
		logger protocol.Logger
		Cache  types.ConcurrentMap
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
			name: "initGenesis_test",
			fields: fields{
				filter: createFilter(),
				logger: logger,
				Cache:  types.New(),
			},
			args: args{
				genesisBlock: blockInfo0,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t1.Run(tt.name, func(t1 *testing.T) {
			t := &BigFilterKvDB{
				filter: tt.fields.filter,
				logger: tt.fields.logger,
				Cache:  tt.fields.Cache,
			}
			t.RWMutex = sync.RWMutex{}
			if err := t.InitGenesis(tt.args.genesisBlock); (err != nil) != tt.wantErr {
				t1.Errorf("InitGenesis() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
	destroyRedisHosts()
}

func TestBigFilterKvDB_CommitBlock(t1 *testing.T) {
	initRedisHosts()
	_, blockInfo0, _ := serialization.SerializeBlock(block0)

	type fields struct {
		filter filter.Filter
		logger protocol.Logger
		Cache  types.ConcurrentMap
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
			name: "commitblock_cache_test",
			fields: fields{
				filter: createFilter(),
				logger: logger,
				Cache:  types.New(),
			},
			args: args{
				blockInfo: blockInfo0,
				isCache:   true,
			},
			wantErr: false,
		},
		{
			name: "commitblock_db_test",
			fields: fields{
				filter: createFilter(),
				logger: logger,
				Cache:  types.New(),
			},
			args: args{
				blockInfo: blockInfo0,
				isCache:   true,
			},
			wantErr: false,
		},
		//{
		//	name: "commitblock_db_error",
		//	fields: fields{
		//		filter: createFilter(),
		//		logger: logger,
		//		Cache:  types.New(),
		//	},
		//	args: args{
		//		blockInfo: blockInfo0,
		//		isCache:   false,
		//	},
		//	wantErr: true,
		//},
	}
	for _, tt := range tests {
		t1.Run(tt.name, func(t1 *testing.T) {
			t := &BigFilterKvDB{
				filter: tt.fields.filter,
				logger: tt.fields.logger,
				Cache:  tt.fields.Cache,
			}
			t.RWMutex = sync.RWMutex{}

			//模拟redis 故障的case
			if tt.name == "commitblock_db_error" {
				destroyRedisHosts()
			}

			if err := t.CommitBlock(tt.args.blockInfo, tt.args.isCache); (err != nil) != tt.wantErr {
				t1.Errorf("CommitBlock() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
	destroyRedisHosts()
}

func TestBigFilterKvDB_GetLastSavepoint(t1 *testing.T) {
	initRedisHosts()
	_, blockInfo0, _ := serialization.SerializeBlock(block0)
	_, blockInfo1, _ := serialization.SerializeBlock(block1)

	type fields struct {
		filter filter.Filter
		logger protocol.Logger
		Cache  types.ConcurrentMap
	}
	tests := []struct {
		name    string
		fields  fields
		want    uint64
		wantErr bool
	}{
		// TODO: Add test cases.
		{
			name: "get_last_savepoint",
			fields: fields{
				filter: createFilter(),
				logger: logger,
				Cache:  types.New(),
			},
			want:    1,
			wantErr: false,
		},
		{
			name: "can_not_get_last_savepoint_from_cache",
			fields: fields{
				filter: createFilter(),
				logger: logger,
				Cache:  types.New(),
			},
			want:    1,
			wantErr: false,
		},
		{
			name: "get_last_savepoint_from_db_err",
			fields: fields{
				filter: createFilter(),
				logger: logger,
				Cache:  types.New(),
			},
			want:    0,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t1.Run(tt.name, func(t1 *testing.T) {
			t := &BigFilterKvDB{
				filter: tt.fields.filter,
				logger: tt.fields.logger,
				Cache:  tt.fields.Cache,
			}
			t.RWMutex = sync.RWMutex{}

			//写两个快,块高分别为 0，1
			t.CommitBlock(blockInfo0, true)
			t.CommitBlock(blockInfo0, false)
			t.CommitBlock(blockInfo1, true)
			t.CommitBlock(blockInfo1, false)

			//模拟从cache中未找到lastSavePoint的情况，重新创建一个Cache
			if tt.name == "can_not_get_last_savepoint_from_cache" {
				t.Cache = types.New()
			}
			//模拟未命中cache，同时查db也报错的情况
			if tt.name == "get_last_savepoint_from_db_err" {
				t.Cache = types.New()
				destroyRedisHosts()
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
	destroyRedisHosts()
}

func TestBigFilterKvDB_TxExists(t1 *testing.T) {
	initRedisHosts()

	b0 := block0
	b0.Block.Txs[0].Payload.TxId = txId1

	b1 := block1
	b1.Block.Txs[0].Payload.TxId = txId2

	_, blockInfo0, _ := serialization.SerializeBlock(block0)
	//_, blockInfo1, _ := serialization.SerializeBlock(block1)

	type fields struct {
		filter filter.Filter
		logger protocol.Logger
		Cache  types.ConcurrentMap
	}
	type args struct {
		txId string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    bool
		want1   bool
		wantErr bool
	}{
		// TODO: Add test cases.
		{
			name: "txExist",
			fields: fields{
				filter: createFilter(),
				logger: logger,
				Cache:  types.New(),
			},
			args: args{
				txId: txId1,
			},
			want:    true,
			want1:   true,
			wantErr: false,
		},
		{
			name: "tx_not_Exist",
			fields: fields{
				filter: createFilter(),
				logger: logger,
				Cache:  types.New(),
			},
			args: args{
				txId: txId2,
			},
			want:    false,
			want1:   false,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t1.Run(tt.name, func(t1 *testing.T) {
			t := &BigFilterKvDB{
				filter: tt.fields.filter,
				logger: tt.fields.logger,
				Cache:  tt.fields.Cache,
			}
			t.RWMutex = sync.RWMutex{}

			//写两个快,块高分别为 0，1
			t.CommitBlock(blockInfo0, true)
			t.CommitBlock(blockInfo0, false)

			got, got1, err := t.TxExists(tt.args.txId)
			if (err != nil) != tt.wantErr {
				t1.Errorf("TxExists() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t1.Errorf("TxExists() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t1.Errorf("TxExists() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
	destroyRedisHosts()
}

func TestBigFilterKvDB_Close(t1 *testing.T) {
	type fields struct {
		filter filter.Filter
		logger protocol.Logger
		Cache  types.ConcurrentMap
	}
	tests := []struct {
		name   string
		fields fields
	}{
		// TODO: Add test cases.
		{
			name: "bigfilter_close",
			fields: fields{
				filter: createFilter(),
				logger: logger,
				Cache:  types.New(),
			},
		},
	}
	for _, tt := range tests {
		t1.Run(tt.name, func(t1 *testing.T) {
			t := &BigFilterKvDB{
				filter: tt.fields.filter,
				logger: tt.fields.logger,
				Cache:  tt.fields.Cache,
			}
			t.RWMutex = sync.RWMutex{}
			t.Close()
		})
	}
}

func Test_constructTxIDKey(t *testing.T) {
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
			name: "constructTx",
			args: args{
				txId: txId1,
			},
			want: []byte("t" + txId1),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			//fmt.Println(string(constructTxIDKey(tt.args.txId)))
			if got := constructTxIDKey(tt.args.txId); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("constructTxIDKey() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMain(m *testing.M) {
	if len(os.Getenv("BUILD_ID")) == 0 { //jenkins has it
		return
	}
	run := m.Run()
	os.Exit(run)
}
