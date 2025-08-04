/*
 Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
   SPDX-License-Identifier: Apache-2.0
*/

package vm

import (
	"testing"

	commonPb "chainmaker.org/chainmaker/pb-go/v2/common"
	"chainmaker.org/chainmaker/protocol/v2"
	"github.com/stretchr/testify/require"
)

// TestSimContextIteratorNextValue comment at next version
func TestSimContextIteratorNextValue(t *testing.T) {
	transactPayload := &commonPb.Payload{
		ContractName: "test",
		Method:       "test",
		Parameters:   nil,
	}
	_, err := transactPayload.Marshal()
	require.Nil(t, err)
	simContext := &txSimContextImpl{
		txRWSetWithDepth: make([]map[string]*rwSet, 5),
		rowCache:         make([]map[int32]interface{}, 5),
		txWriteKeySql:    make([]*commonPb.TxWrite, 0),
		txWriteKeyDdlSql: make([]*commonPb.TxWrite, 0),
		tx: &commonPb.Transaction{
			Payload: &commonPb.Payload{
				ChainId:        "chain1",
				TxId:           "12345678",
				TxType:         commonPb.TxType_INVOKE_CONTRACT,
				Timestamp:      0,
				ExpirationTime: 0,
				ContractName:   "test_contract",
			},
			Sender: nil,
			Result: nil,
		},
		gasUsed:      0,
		currentDepth: 0,
		hisResult:    make([]*callContractResult, 0),
	}
	simContextEmptyIterator := NewSimContextIterator(simContext, makeEmptyWSetIterator(), makeEmptyWSetIterator())
	require.False(t, simContextEmptyIterator.Next())
	val, err := simContextEmptyIterator.Value()
	require.Nil(t, err)
	require.Nil(t, val)

	i := 0
	_, vals := makeStringKeyMap()
	simContextMockWSetIterator := NewSimContextIterator(simContext, makeMockWSetIterator(), makeEmptyWSetIterator())
	for {
		if !simContextMockWSetIterator.Next() {
			break
		}
		val, err := simContextMockWSetIterator.Value()
		require.Nil(t, err)
		require.Equal(t, vals[i], val)
		i++
	}

	i = 0
	simContextIterator := NewSimContextIterator(simContext, makeMockWSetIterator(), makeMockDbIterator())
	for {
		if !simContextIterator.Next() {
			break
		}
		val, err := simContextIterator.Value()
		require.Nil(t, err)
		require.Equal(t, vals[i], val)
		i++
	}
}

func makeEmptyWSetIterator() protocol.StateIterator {
	return NewWsetIterator(make(map[string]interface{}))
}

func makeMockWSetIterator() protocol.StateIterator {
	stringKeyMap, _ := makeStringKeyMap()
	return NewWsetIterator(stringKeyMap)
}

func makeMockDbIterator() protocol.StateIterator {
	stringKeyMap, _ := makeStringKeyMap()
	return NewWsetIterator(stringKeyMap)
}
