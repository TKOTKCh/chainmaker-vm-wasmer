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
)

func TestNewWSetKeyHistoryIterator(t *testing.T) {
	wSets := map[string]interface{}{key: value}
	type args struct {
		wSets map[string]interface{}
	}
	tests := []struct {
		name string
		args args
		want *WSetKeyHistoryIterator
	}{
		{
			name: "t1",
			args: args{wSets: wSets},
			want: &WSetKeyHistoryIterator{
				stringKeySortedMap: sortedmap.NewStringKeySortedMapWithInterfaceData(wSets),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewWSetKeyHistoryIterator(tt.args.wSets); got == nil {
				t.Errorf("NewWSetKeyHistoryIterator() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestWSetKeyHistoryIterator_Next(t *testing.T) {
	type fields struct {
		stringKeySortedMap *sortedmap.StringKeySortedMap
	}
	tests := []struct {
		name   string
		fields fields
		want   bool
	}{
		{
			name: "t1",
			fields: fields{
				stringKeySortedMap: sortedmap.NewStringKeySortedMapWithInterfaceData(map[string]interface{}{key: value}),
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			iter := &WSetKeyHistoryIterator{
				stringKeySortedMap: tt.fields.stringKeySortedMap,
			}
			if got := iter.Next(); got != tt.want {
				t.Errorf("Next() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestWSetKeyHistoryIterator_Release(t *testing.T) {
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
			iter := &WSetKeyHistoryIterator{
				stringKeySortedMap: tt.fields.stringKeySortedMap,
			}
			iter.Release()
		})
	}
}

func TestWSetKeyHistoryIterator_Value(t *testing.T) {
	type fields struct {
		stringKeySortedMap *sortedmap.StringKeySortedMap
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
				stringKeySortedMap: sortedmap.NewStringKeySortedMapWithInterfaceData(map[string]interface{}{key: value}),
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			iter := &WSetKeyHistoryIterator{
				stringKeySortedMap: tt.fields.stringKeySortedMap,
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
