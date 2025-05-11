/*
 * Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
 *
 * SPDX-License-Identifier: Apache-2.0
 *
 */

package rolling_window_cache

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"reflect"
	"testing"
	"time"

	acPb "chainmaker.org/chainmaker/pb-go/v2/accesscontrol"
	commonPb "chainmaker.org/chainmaker/pb-go/v2/common"
	storePb "chainmaker.org/chainmaker/pb-go/v2/store"
	"chainmaker.org/chainmaker/pb-go/v2/syscontract"
	"chainmaker.org/chainmaker/protocol/v2"
	"chainmaker.org/chainmaker/protocol/v2/test"
	"chainmaker.org/chainmaker/store/v2/cache"
	"chainmaker.org/chainmaker/store/v2/serialization"
	"chainmaker.org/chainmaker/store/v2/types"
	"github.com/gogo/protobuf/proto"
)

var (
	logger                               = &test.GoLogger{}
	testStr                              = "contract1"
	testChainId                          = "testchainid_1"
	block0                               = createConfigBlock(testChainId, 0)
	block1                               = createBlockAndRWSets(testChainId, 1, 10)
	txId1                                = "0000000000-abcdfjelf-as88"
	txId2                                = "1111111111-abcdfjelf-as88"
	rolling_window_cache_capacity uint64 = 100
)

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

//创建配置区块
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

//生成读写集
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

func TestNewRollingWindowCacher(t *testing.T) {
	type args struct {
		txIdCount        uint64
		currCount        uint64
		startBlockHeight uint64
		endBlockHeight   uint64
		lastBlockHeight  uint64
		logger           protocol.Logger
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		// TODO: Add test cases.
		{
			name: "new_rolling_window_cacher",
			args: args{
				txIdCount:        rolling_window_cache_capacity,
				currCount:        0,
				startBlockHeight: 0,
				endBlockHeight:   0,
				lastBlockHeight:  0,
				logger:           logger,
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewRollingWindowCacher(tt.args.txIdCount, tt.args.currCount, tt.args.startBlockHeight,
				tt.args.endBlockHeight, tt.args.lastBlockHeight, tt.args.logger)
			if got == nil {
				t.Errorf("NewRollingWindowCacher() = nil, want got !=nil ,%v", got)
			}
		})
	}
}

func TestRollingWindowCacher_InitGenesis(t *testing.T) {
	_, blockInfo0, _ := serialization.SerializeBlock(block0)

	type fields struct {
		Cache            *cache.StoreCacheMgr
		txIdCount        uint64
		currCount        uint64
		startBlockHeight uint64
		endBlockHeight   uint64
		lastBlockHeight  uint64
		logger           protocol.Logger
		blockSerChan     chan *serialization.BlockWithSerializedInfo
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
			name: "test_rollingWindowCacher_initGenesis",
			fields: fields{
				Cache:            cache.NewStoreCacheMgr("", 10, logger),
				txIdCount:        rolling_window_cache_capacity,
				currCount:        0,
				startBlockHeight: 0,
				endBlockHeight:   0,
				lastBlockHeight:  0,
				logger:           logger,
				blockSerChan:     make(chan *serialization.BlockWithSerializedInfo, 10),
			},
			args: args{
				genesisBlock: blockInfo0,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &RollingWindowCacher{
				Cache:            tt.fields.Cache,
				txIdCount:        tt.fields.txIdCount,
				currCount:        tt.fields.currCount,
				startBlockHeight: tt.fields.startBlockHeight,
				endBlockHeight:   tt.fields.endBlockHeight,
				lastBlockHeight:  tt.fields.lastBlockHeight,
				logger:           tt.fields.logger,
				blockSerChan:     tt.fields.blockSerChan,
			}

			r.batchPool.New = func() interface{} {
				return types.NewUpdateBatch()
			}

			if err := r.InitGenesis(tt.args.genesisBlock); (err != nil) != tt.wantErr {
				t.Errorf("InitGenesis() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRollingWindowCacher_CommitBlock(t *testing.T) {
	_, blockInfo0, _ := serialization.SerializeBlock(block0)

	type fields struct {
		Cache            *cache.StoreCacheMgr
		txIdCount        uint64
		currCount        uint64
		startBlockHeight uint64
		endBlockHeight   uint64
		lastBlockHeight  uint64
		logger           protocol.Logger
		blockSerChan     chan *serialization.BlockWithSerializedInfo
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
			name: "test_rwcache_commit",
			fields: fields{
				Cache:            cache.NewStoreCacheMgr("", 10, logger),
				txIdCount:        rolling_window_cache_capacity,
				currCount:        0,
				startBlockHeight: 0,
				endBlockHeight:   0,
				lastBlockHeight:  0,
				logger:           logger,
				blockSerChan:     make(chan *serialization.BlockWithSerializedInfo, 10),
			},
			args: args{
				blockInfo: blockInfo0,
				isCache:   true,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &RollingWindowCacher{
				Cache:            tt.fields.Cache,
				txIdCount:        tt.fields.txIdCount,
				currCount:        tt.fields.currCount,
				startBlockHeight: tt.fields.startBlockHeight,
				endBlockHeight:   tt.fields.endBlockHeight,
				lastBlockHeight:  tt.fields.lastBlockHeight,
				logger:           tt.fields.logger,
				blockSerChan:     tt.fields.blockSerChan,
			}
			r.batchPool.New = func() interface{} {
				return types.NewUpdateBatch()
			}

			if err := r.CommitBlock(tt.args.blockInfo, tt.args.isCache); (err != nil) != tt.wantErr {
				t.Errorf("CommitBlock() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRollingWindowCacher_ResetRWCache(t *testing.T) {
	_, blockInfo0, _ := serialization.SerializeBlock(block0)

	type fields struct {
		Cache            *cache.StoreCacheMgr
		txIdCount        uint64
		currCount        uint64
		startBlockHeight uint64
		endBlockHeight   uint64
		lastBlockHeight  uint64
		logger           protocol.Logger
		blockSerChan     chan *serialization.BlockWithSerializedInfo
	}
	type args struct {
		blockInfo *serialization.BlockWithSerializedInfo
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
		{
			name: "test_reset_rw_cache",
			fields: fields{
				Cache:            cache.NewStoreCacheMgr("", 10, logger),
				txIdCount:        rolling_window_cache_capacity,
				currCount:        0,
				startBlockHeight: 0,
				endBlockHeight:   0,
				lastBlockHeight:  0,
				logger:           logger,
				blockSerChan:     make(chan *serialization.BlockWithSerializedInfo, 10),
			},
			args: args{
				blockInfo: blockInfo0,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &RollingWindowCacher{
				Cache:            tt.fields.Cache,
				txIdCount:        tt.fields.txIdCount,
				currCount:        tt.fields.currCount,
				startBlockHeight: tt.fields.startBlockHeight,
				endBlockHeight:   tt.fields.endBlockHeight,
				lastBlockHeight:  tt.fields.lastBlockHeight,
				logger:           tt.fields.logger,
				blockSerChan:     tt.fields.blockSerChan,
			}
			r.batchPool.New = func() interface{} {
				return types.NewUpdateBatch()
			}

			if err := r.ResetRWCache(tt.args.blockInfo); (err != nil) != tt.wantErr {
				t.Errorf("ResetRWCache() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRollingWindowCacher_Has(t *testing.T) {
	block0.Block.Txs[0].Payload.TxId = txId1
	block1.Block.Txs[0].Payload.TxId = txId2

	_, blockInfo0, _ := serialization.SerializeBlock(block0)
	_, blockInfo1, _ := serialization.SerializeBlock(block1)

	type fields struct {
		Cache            *cache.StoreCacheMgr
		txIdCount        uint64
		currCount        uint64
		startBlockHeight uint64
		endBlockHeight   uint64
		lastBlockHeight  uint64
		logger           protocol.Logger
		blockSerChan     chan *serialization.BlockWithSerializedInfo
	}
	type args struct {
		key   string
		start uint64
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
			name: "rwcache_has",
			fields: fields{
				Cache:            cache.NewStoreCacheMgr("", 10, logger),
				txIdCount:        rolling_window_cache_capacity,
				currCount:        0,
				startBlockHeight: 0,
				endBlockHeight:   0,
				lastBlockHeight:  0,
				logger:           logger,
				blockSerChan:     make(chan *serialization.BlockWithSerializedInfo, 10),
			},
			args: args{
				key:   txId1,
				start: 0,
			},
			want:    true,
			want1:   true,
			wantErr: false,
		},
		{
			name: "rwcache_not_has",
			fields: fields{
				Cache:            cache.NewStoreCacheMgr("", 10, logger),
				txIdCount:        rolling_window_cache_capacity,
				currCount:        0,
				startBlockHeight: 0,
				endBlockHeight:   0,
				lastBlockHeight:  0,
				logger:           logger,
				blockSerChan:     make(chan *serialization.BlockWithSerializedInfo, 10),
			},
			args: args{
				key:   txId1,
				start: 2,
			},
			want:    false,
			want1:   false,
			wantErr: false,
		},
		{
			name: "startHeight_delete_rwcache_has",
			fields: fields{
				Cache:            cache.NewStoreCacheMgr("", 10, logger),
				txIdCount:        10, //窗口内一共保持10个交易，block1包含10个，block2包含10个
				currCount:        0,
				startBlockHeight: 0,
				endBlockHeight:   0,
				lastBlockHeight:  0,
				logger:           logger,
				blockSerChan:     make(chan *serialization.BlockWithSerializedInfo, 10),
			},
			args: args{
				key:   txId1, //txId1 在0号区块内，模拟 第二次查询的情况
				start: 1,
			},
			want:    true,
			want1:   false,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &RollingWindowCacher{
				Cache:            tt.fields.Cache,
				txIdCount:        tt.fields.txIdCount,
				currCount:        tt.fields.currCount,
				startBlockHeight: tt.fields.startBlockHeight,
				endBlockHeight:   tt.fields.endBlockHeight,
				lastBlockHeight:  tt.fields.lastBlockHeight,
				logger:           tt.fields.logger,
				blockSerChan:     tt.fields.blockSerChan,
			}
			r.batchPool.New = func() interface{} {
				return types.NewUpdateBatch()
			}
			go r.Consumer()

			r.InitGenesis(blockInfo0)
			r.CommitBlock(blockInfo1, true)

			//sleep 3秒，保证cache 被异步消费成功
			time.Sleep(100 * time.Millisecond)
			//fmt.Println("r =====",r)

			got, got1, err := r.Has(tt.args.key, tt.args.start)
			if (err != nil) != tt.wantErr {
				t.Errorf("Has() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Has() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("Has() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func TestRollingWindowCacher_Consumer(t *testing.T) {
	block0.Block.Txs[0].Payload.TxId = txId1
	block1.Block.Txs[0].Payload.TxId = txId2

	_, blockInfo0, _ := serialization.SerializeBlock(block0)
	_, blockInfo1, _ := serialization.SerializeBlock(block1)

	type fields struct {
		Cache            *cache.StoreCacheMgr
		txIdCount        uint64
		currCount        uint64
		startBlockHeight uint64
		endBlockHeight   uint64
		lastBlockHeight  uint64
		logger           protocol.Logger
		blockSerChan     chan *serialization.BlockWithSerializedInfo
	}
	tests := []struct {
		name   string
		fields fields
	}{
		// TODO: Add test cases.
		{
			name: "test_consumer",
			fields: fields{
				Cache:            cache.NewStoreCacheMgr("", 10, logger),
				txIdCount:        10, //cache中保留10个交易
				currCount:        0,
				startBlockHeight: 0,
				endBlockHeight:   0,
				lastBlockHeight:  0,
				logger:           logger,
				blockSerChan:     make(chan *serialization.BlockWithSerializedInfo, 10),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &RollingWindowCacher{
				Cache:            tt.fields.Cache,
				txIdCount:        tt.fields.txIdCount,
				currCount:        tt.fields.currCount,
				startBlockHeight: tt.fields.startBlockHeight,
				endBlockHeight:   tt.fields.endBlockHeight,
				lastBlockHeight:  tt.fields.lastBlockHeight,
				logger:           tt.fields.logger,
				blockSerChan:     tt.fields.blockSerChan,
			}
			r.batchPool.New = func() interface{} {
				return types.NewUpdateBatch()
			}

			go r.Consumer()

			//写入2个块，共20个交易，根据窗口大小，会淘汰一部分，完成滑动
			r.InitGenesis(blockInfo0)
			r.CommitBlock(blockInfo1, true)

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
			if got := constructTxIDKey(tt.args.txId); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("constructTxIDKey() = %v, want %v", got, tt.want)
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
