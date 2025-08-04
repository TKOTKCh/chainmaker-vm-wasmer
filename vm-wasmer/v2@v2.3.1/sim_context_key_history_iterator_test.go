/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package vm

import (
	"reflect"
	"testing"

	"chainmaker.org/chainmaker/pb-go/v2/common"
	"chainmaker.org/chainmaker/pb-go/v2/store"
	"chainmaker.org/chainmaker/protocol/v2"
	"chainmaker.org/chainmaker/protocol/v2/mock"
	"github.com/golang/mock/gomock"
)

const (
	key   = "key1"
	value = "value1"
)

func TestNewSimContextKeyHistoryIterator(t *testing.T) {
	c := gomock.NewController(t)
	defer c.Finish()
	context := mock.NewMockTxSimContext(c)
	iterator := mock.NewMockKeyHistoryIterator(c)
	dbInter := mock.NewMockKeyHistoryIterator(c)

	type args struct {
		simContext protocol.TxSimContext
		wSetIter   protocol.KeyHistoryIterator
		dbIter     protocol.KeyHistoryIterator
		key        []byte
	}
	tests := []struct {
		name string
		args args
		want *SimContextKeyHistoryIterator
	}{
		{
			name: "t1",
			args: args{
				simContext: context,
				wSetIter:   iterator,
				dbIter:     dbInter,
				key:        []byte(key),
			},
			want: NewSimContextKeyHistoryIterator(context, iterator, dbInter, []byte(key)),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewSimContextKeyHistoryIterator(tt.args.simContext, tt.args.wSetIter, tt.args.dbIter, tt.args.key); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewSimContextKeyHistoryIterator() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSimContextKeyHistoryIterator_Next(t *testing.T) {
	c := gomock.NewController(t)
	defer c.Finish()
	context := mock.NewMockTxSimContext(c)
	dbKeyHistoryIter := mock.NewMockKeyHistoryIterator(c)
	gomock.InOrder(
		dbKeyHistoryIter.EXPECT().Next().Return(true).AnyTimes(),
		dbKeyHistoryIter.EXPECT().Value().Return(nil, nil).AnyTimes(),
	)

	wSetKeyHistoryIter := mock.NewMockKeyHistoryIterator(c)
	gomock.InOrder(
		wSetKeyHistoryIter.EXPECT().Next().Return(true).AnyTimes(),
		wSetKeyHistoryIter.EXPECT().Value().Return(nil, nil).AnyTimes(),
	)

	dbKeyHistoryIter1 := mock.NewMockKeyHistoryIterator(c)
	gomock.InOrder(
		dbKeyHistoryIter1.EXPECT().Next().Return(false).AnyTimes(),
		dbKeyHistoryIter1.EXPECT().Value().Return(nil, nil).AnyTimes(),
	)

	type fields struct {
		key                []byte
		finalValue         *store.KeyModification
		wSetKeyHistoryIter protocol.KeyHistoryIterator
		dbKeyHistoryIter   protocol.KeyHistoryIterator
		simContext         protocol.TxSimContext
	}
	tests := []struct {
		name   string
		fields fields
		want   bool
	}{
		{
			name: "t1",
			fields: fields{
				key: []byte("key1"),
				finalValue: &store.KeyModification{
					TxId:        txId,
					Value:       []byte(value),
					Timestamp:   0,
					IsDelete:    false,
					BlockHeight: 0,
				},
				wSetKeyHistoryIter: nil,
				dbKeyHistoryIter:   dbKeyHistoryIter,
				simContext:         context,
			},
			want: true,
		},

		{
			name: "t1",
			fields: fields{
				key: []byte("key1"),
				finalValue: &store.KeyModification{
					TxId:        txId,
					Value:       []byte(value),
					Timestamp:   0,
					IsDelete:    false,
					BlockHeight: 0,
				},
				wSetKeyHistoryIter: wSetKeyHistoryIter,
				dbKeyHistoryIter:   dbKeyHistoryIter1,
				simContext:         context,
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			iter := &SimContextKeyHistoryIterator{
				key:                tt.fields.key,
				finalValue:         tt.fields.finalValue,
				wSetKeyHistoryIter: tt.fields.wSetKeyHistoryIter,
				dbKeyHistoryIter:   tt.fields.dbKeyHistoryIter,
				simContext:         tt.fields.simContext,
			}
			if got := iter.Next(); got != tt.want {
				t.Errorf("Next() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSimContextKeyHistoryIterator_Value(t *testing.T) {
	c := gomock.NewController(t)
	defer c.Finish()
	context := mock.NewMockTxSimContext(c)
	transaction := &common.Transaction{
		Payload: &common.Payload{
			ChainId:      chainId,
			TxId:         txId,
			ContractName: contractName,
		},
	}
	context.EXPECT().GetTx().Return(transaction).AnyTimes()
	context.EXPECT().PutIntoReadSet(gomock.Any(), gomock.Any(), gomock.Any()).Return().AnyTimes()

	type fields struct {
		key                []byte
		finalValue         *store.KeyModification
		wSetKeyHistoryIter protocol.KeyHistoryIterator
		dbKeyHistoryIter   protocol.KeyHistoryIterator
		simContext         protocol.TxSimContext
	}
	tests := []struct {
		name    string
		fields  fields
		want    *store.KeyModification
		wantErr bool
	}{
		{
			name: "t1",
			fields: fields{
				key:        []byte(key),
				simContext: context,
			},
			want:    nil,
			wantErr: false,
		},

		{
			name: "t2",
			fields: fields{
				key: []byte(key),
				finalValue: &store.KeyModification{
					TxId:  txId,
					Value: []byte(value),
				},
				simContext: context,
			},
			want: &store.KeyModification{
				TxId:  txId,
				Value: []byte(value),
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			iter := &SimContextKeyHistoryIterator{
				key:                tt.fields.key,
				finalValue:         tt.fields.finalValue,
				wSetKeyHistoryIter: tt.fields.wSetKeyHistoryIter,
				dbKeyHistoryIter:   tt.fields.dbKeyHistoryIter,
				simContext:         tt.fields.simContext,
			}
			got, err := iter.Value()
			if (err != nil) != tt.wantErr {
				t.Errorf("Value() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Value() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSimContextKeyHistoryIterator_Release(t *testing.T) {
	c := gomock.NewController(t)
	defer c.Finish()
	context := mock.NewMockTxSimContext(c)
	iterator := mock.NewMockKeyHistoryIterator(c)
	iterator.EXPECT().Release().Return().AnyTimes()
	dbInter := mock.NewMockKeyHistoryIterator(c)
	dbInter.EXPECT().Release().Return().AnyTimes()

	type fields struct {
		key                []byte
		finalValue         *store.KeyModification
		wSetKeyHistoryIter protocol.KeyHistoryIterator
		dbKeyHistoryIter   protocol.KeyHistoryIterator
		simContext         protocol.TxSimContext
	}
	tests := []struct {
		name   string
		fields fields
	}{
		{
			name: "t1",
			fields: fields{
				key:                []byte(key),
				wSetKeyHistoryIter: iterator,
				dbKeyHistoryIter:   dbInter,
				simContext:         context,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			iter := &SimContextKeyHistoryIterator{
				key:                tt.fields.key,
				finalValue:         tt.fields.finalValue,
				wSetKeyHistoryIter: tt.fields.wSetKeyHistoryIter,
				dbKeyHistoryIter:   tt.fields.dbKeyHistoryIter,
				simContext:         tt.fields.simContext,
			}
			iter.Release()
		})
	}
}
