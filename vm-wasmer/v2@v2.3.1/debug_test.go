/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package vm

import (
	"testing"

	"chainmaker.org/chainmaker/pb-go/v2/common"
	commonPb "chainmaker.org/chainmaker/pb-go/v2/common"
	"chainmaker.org/chainmaker/protocol/v2"
	"chainmaker.org/chainmaker/protocol/v2/mock"
	"github.com/golang/mock/gomock"
)

func TestPrintTxReadSet(t *testing.T) {
	c := gomock.NewController(t)
	defer c.Finish()

	context := mock.NewMockTxSimContext(c)
	simContext := &txSimContextImpl{
		// txReadKeyMap:     make(map[string]*commonPb.TxRead, 8),
		// txWriteKeyMap:    make(map[string]*commonPb.TxWrite, 8),
		txRWSetWithDepth: []map[string]*rwSet{
			{
				contractName: &rwSet{
					txWriteKeyMap: make(map[string]*common.TxWrite, 8),
					txReadKeyMap:  make(map[string]*common.TxRead, 8),
				},
			},
		},
		// rowCache:         make(map[int32]interface{}),
		rowCache: []map[int32]interface{}{
			make(map[int32]interface{}),
		},

		txWriteKeySql:    make([]*commonPb.TxWrite, 0),
		txWriteKeyDdlSql: make([]*commonPb.TxWrite, 0),
		tx: &commonPb.Transaction{
			Payload: &commonPb.Payload{
				ChainId:        "chain1",
				TxId:           "12345678",
				TxType:         commonPb.TxType_INVOKE_CONTRACT,
				Timestamp:      0,
				ExpirationTime: 0,
			},
			Sender: nil,
			Result: nil,
		},
		gasUsed:      0,
		currentDepth: 0,
		hisResult:    make([]*callContractResult, 0),
	}
	// simContext.txReadKeyMap["Contract:"] = &commonPb.TxRead{}
	simContext.txRWSetWithDepth[0][contractName].txReadKeyMap["Contract:"] = &commonPb.TxRead{}

	simContext2 := &txSimContextImpl{
		// txReadKeyMap:     make(map[string]*commonPb.TxRead, 8),
		// txWriteKeyMap:    make(map[string]*commonPb.TxWrite, 8),

		txRWSetWithDepth: []map[string]*rwSet{
			{
				contractName: &rwSet{
					txWriteKeyMap: make(map[string]*common.TxWrite, 8),
					txReadKeyMap:  make(map[string]*common.TxRead, 8),
				},
			},
		},
		// rowCache:         make(map[int32]interface{}),
		rowCache: []map[int32]interface{}{
			make(map[int32]interface{}),
		},

		txWriteKeySql:    make([]*commonPb.TxWrite, 0),
		txWriteKeyDdlSql: make([]*commonPb.TxWrite, 0),
		tx: &commonPb.Transaction{
			Payload: &commonPb.Payload{
				ChainId:        "chain1",
				TxId:           "12345678",
				TxType:         commonPb.TxType_INVOKE_CONTRACT,
				Timestamp:      0,
				ExpirationTime: 0,
			},
			Sender: nil,
			Result: nil,
		},
		gasUsed:      0,
		currentDepth: 0,
		hisResult:    make([]*callContractResult, 0),
	}
	// simContext2.txReadKeyMap["unKnow"] = &commonPb.TxRead{}
	simContext.txRWSetWithDepth[0][contractName].txReadKeyMap["unKnow"] = &commonPb.TxRead{}

	type args struct {
		txSimContext protocol.TxSimContext
	}

	tests := []struct {
		name string
		args args
	}{
		{
			name: "t1",
			args: args{
				txSimContext: context,
			},
		},
		{
			name: "t2",
			args: args{
				txSimContext: simContext,
			},
		},

		{
			name: "t3",
			args: args{
				txSimContext: simContext2,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			PrintTxReadSet(tt.args.txSimContext)
		})
	}
}

func TestPrintTxWriteSet(t *testing.T) {
	c := gomock.NewController(t)
	defer c.Finish()

	context := mock.NewMockTxSimContext(c)

	simContext := &txSimContextImpl{
		// txReadKeyMap:     make(map[string]*commonPb.TxRead, 8),
		// txWriteKeyMap:    make(map[string]*commonPb.TxWrite, 8),
		txRWSetWithDepth: []map[string]*rwSet{
			{
				contractName: &rwSet{
					txWriteKeyMap: make(map[string]*common.TxWrite, 8),
					txReadKeyMap:  make(map[string]*common.TxRead, 8),
				},
			},
		},
		// rowCache:         make(map[int32]interface{}),
		rowCache: []map[int32]interface{}{
			make(map[int32]interface{}),
		},
		txWriteKeySql:    make([]*commonPb.TxWrite, 0),
		txWriteKeyDdlSql: make([]*commonPb.TxWrite, 0),
		tx: &commonPb.Transaction{
			Payload: &commonPb.Payload{
				ChainId:        "chain1",
				TxId:           "12345678",
				TxType:         commonPb.TxType_INVOKE_CONTRACT,
				Timestamp:      0,
				ExpirationTime: 0,
			},
			Sender: nil,
			Result: nil,
		},
		gasUsed:      0,
		currentDepth: 0,
		hisResult:    make([]*callContractResult, 0),
	}
	// simContext.txWriteKeyMap["Contract:"] = &commonPb.TxWrite{}
	simContext.txRWSetWithDepth[0][contractName].txWriteKeyMap["Contract:"] = &commonPb.TxWrite{}

	simContext2 := &txSimContextImpl{
		// txReadKeyMap:     make(map[string]*commonPb.TxRead, 8),
		// txWriteKeyMap:    make(map[string]*commonPb.TxWrite, 8),
		txRWSetWithDepth: []map[string]*rwSet{
			{
				contractName: &rwSet{
					txWriteKeyMap: make(map[string]*common.TxWrite, 8),
					txReadKeyMap:  make(map[string]*common.TxRead, 8),
				},
			},
		},

		rowCache: []map[int32]interface{}{
			make(map[int32]interface{}),
		},

		// rowCache:         make(map[int32]interface{}),
		txWriteKeySql:    make([]*commonPb.TxWrite, 0),
		txWriteKeyDdlSql: make([]*commonPb.TxWrite, 0),
		tx: &commonPb.Transaction{
			Payload: &commonPb.Payload{
				ChainId:        "chain1",
				TxId:           "12345678",
				TxType:         commonPb.TxType_INVOKE_CONTRACT,
				Timestamp:      0,
				ExpirationTime: 0,
			},
			Sender: nil,
			Result: nil,
		},
		gasUsed:      0,
		currentDepth: 0,
		hisResult:    make([]*callContractResult, 0),
	}
	// simContext2.txWriteKeyMap["unKnow"] = &commonPb.TxWrite{}
	simContext.txRWSetWithDepth[0][contractName].txWriteKeyMap["unKnow"] = &commonPb.TxWrite{}

	type args struct {
		txSimContext protocol.TxSimContext
	}

	tests := []struct {
		name string
		args args
	}{
		{
			name: "t1",
			args: args{
				txSimContext: context,
			},
		},
		{
			name: "t2",
			args: args{
				txSimContext: simContext,
			},
		},
		{
			name: "t3",
			args: args{
				txSimContext: simContext2,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			PrintTxWriteSet(tt.args.txSimContext)
		})
	}
}
