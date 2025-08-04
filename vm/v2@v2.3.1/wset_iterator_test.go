/*
 Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
   SPDX-License-Identifier: Apache-2.0
*/

package vm

import (
	"reflect"
	"testing"

	"chainmaker.org/chainmaker/common/v2/sortedmap"
	"chainmaker.org/chainmaker/pb-go/v2/store"
	"github.com/stretchr/testify/require"
)

// TestWsetIteratorNextValue comment at next version
func TestWsetIteratorNextValue(t *testing.T) {
	type testData struct {
		data      map[string]interface{}
		wantNext  bool
		wantValue *store.KV
	}
	stringKeyMap, _ := makeStringKeyMap()
	tests := []*testData{
		{
			data:      make(map[string]interface{}),
			wantNext:  false,
			wantValue: nil,
		},
		{
			data:     stringKeyMap,
			wantNext: true,
			wantValue: &store.KV{
				ContractName: "a",
				Key:          []byte("a"),
				Value:        []byte("a"),
			},
		},
	}
	for _, test := range tests {
		wi := NewWsetIterator(test.data)
		require.Equal(t, test.wantNext, wi.Next())
		val, err := wi.Value()
		require.Nil(t, err)
		require.Equal(t, test.wantValue, val)
	}
}

func makeStringKeyMap() (map[string]interface{}, []*store.KV) {
	stringKeyMap := make(map[string]interface{})
	kvs := make([]*store.KV, 0)
	kvA := &store.KV{
		ContractName: "a",
		Key:          []byte("a"),
		Value:        []byte("a"),
	}
	kvAb := &store.KV{
		ContractName: "ab",
		Key:          []byte("ab"),
		Value:        []byte("ab"),
	}
	kvB := &store.KV{
		ContractName: "b",
		Key:          []byte("b"),
		Value:        []byte("b"),
	}
	kvBa := &store.KV{
		ContractName: "ba",
		Key:          []byte("ba"),
		Value:        []byte("ba"),
	}
	kvBb := &store.KV{
		ContractName: "bb",
		Key:          []byte("bb"),
		Value:        []byte("bb"),
	}
	stringKeyMap["a"] = kvA
	stringKeyMap["ab"] = kvAb
	stringKeyMap["b"] = kvB
	stringKeyMap["ba"] = kvBa
	stringKeyMap["bb"] = kvBb
	kvs = append(kvs, kvA)
	kvs = append(kvs, kvAb)
	kvs = append(kvs, kvB)
	kvs = append(kvs, kvBa)
	kvs = append(kvs, kvBb)
	return stringKeyMap, kvs
}

func TestWsetIterator_Release(t *testing.T) {
	type fields struct {
		stringKeySortedMap *sortedmap.StringKeySortedMap
	}
	tests := []struct {
		name   string
		fields fields
	}{
		{
			name:   "t1",
			fields: fields{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wi := &WsetIterator{
				stringKeySortedMap: tt.fields.stringKeySortedMap,
			}
			wi.Release()
		})
	}
}

func TestWsetIterator_Value(t *testing.T) {
	type fields struct {
		stringKeySortedMap *sortedmap.StringKeySortedMap
	}
	tests := []struct {
		name    string
		fields  fields
		want    *store.KV
		wantErr bool
	}{
		{
			name: "t1",
			fields: fields{
				stringKeySortedMap: NewWsetIterator(map[string]interface{}{
					"a": store.KV{},
				}).stringKeySortedMap,
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wi := &WsetIterator{
				stringKeySortedMap: tt.fields.stringKeySortedMap,
			}
			got, err := wi.Value()
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
