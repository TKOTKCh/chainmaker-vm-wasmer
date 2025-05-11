/*
 * Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
 *
 * SPDX-License-Identifier: Apache-2.0
 *
 */

package types

import (
	"reflect"
	"strconv"
	"sync"
	"testing"

	"chainmaker.org/chainmaker/protocol/v2"
)

var (
	keyStr1 = "key1"
	keyStr2 = "key2"
)

func TestNewUpdateBatch(t *testing.T) {
	tests := []struct {
		name string
		want protocol.StoreBatcher
	}{
		// TODO: Add test cases.
		{
			name: "newUpdateBatch",
			want: &UpdateBatch{
				kvs:     make(map[string][]byte),
				rwLock:  &sync.RWMutex{},
				currMap: New(),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewUpdateBatch(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewUpdateBatch() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUpdateBatch_Delete(t *testing.T) {
	type fields struct {
		kvs     map[string][]byte
		rwLock  *sync.RWMutex
		currMap ConcurrentMap
	}
	type args struct {
		key []byte
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		// TODO: Add test cases.
		{
			name: keyStr1,
			fields: fields{
				kvs:     make(map[string][]byte),
				rwLock:  &sync.RWMutex{},
				currMap: New(),
			},
			args: args{key: []byte(keyStr1)},
		},
		{
			name: keyStr2,
			fields: fields{
				kvs:     make(map[string][]byte),
				rwLock:  &sync.RWMutex{},
				currMap: New(),
			},
			args: args{key: []byte(keyStr2)},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			batch := &UpdateBatch{
				kvs:     tt.fields.kvs,
				rwLock:  tt.fields.rwLock,
				currMap: tt.fields.currMap,
			}
			batch.Put(tt.args.key, tt.args.key)
			batch.Delete(tt.args.key)
		})

	}

}

func TestUpdateBatch_Get(t *testing.T) {
	type fields struct {
		kvs     map[string][]byte
		rwLock  *sync.RWMutex
		currMap ConcurrentMap
	}
	type args struct {
		key []byte
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
			name: keyStr1,
			fields: fields{
				kvs:     make(map[string][]byte),
				rwLock:  &sync.RWMutex{},
				currMap: New(),
			},
			args:    args{key: []byte(keyStr1)},
			want:    []byte(keyStr1),
			wantErr: false,
		},
		{
			name: keyStr2,
			fields: fields{
				kvs:     make(map[string][]byte),
				rwLock:  &sync.RWMutex{},
				currMap: New(),
			},
			args:    args{key: []byte(keyStr2)},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			batch := &UpdateBatch{
				kvs:     tt.fields.kvs,
				rwLock:  tt.fields.rwLock,
				currMap: tt.fields.currMap,
			}
			key2 := keyStr2
			if string(tt.args.key) == key2 {
				batch.Put([]byte("key3"), []byte(key2))

			} else {
				batch.Put(tt.args.key, tt.args.key)
			}

			got, err := batch.Get(tt.args.key)
			if (err != nil) != tt.wantErr {
				t.Errorf("Get() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Get() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUpdateBatch_Has(t *testing.T) {
	type fields struct {
		kvs     map[string][]byte
		rwLock  *sync.RWMutex
		currMap ConcurrentMap
	}
	type args struct {
		key []byte
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		// TODO: Add test cases.
		{
			name: keyStr1,
			fields: fields{
				kvs:     make(map[string][]byte),
				rwLock:  &sync.RWMutex{},
				currMap: New(),
			},
			args: args{key: []byte(keyStr1)},
			want: true,
		},
		{
			name: keyStr2,
			fields: fields{
				kvs:     make(map[string][]byte),
				rwLock:  &sync.RWMutex{},
				currMap: New(),
			},
			args: args{key: []byte("key3")},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			batch := &UpdateBatch{
				kvs:     tt.fields.kvs,
				rwLock:  tt.fields.rwLock,
				currMap: tt.fields.currMap,
			}
			if string(tt.args.key) == keyStr2 {
				batch.Put(tt.args.key, tt.args.key)
			} else {
				batch.Put(tt.args.key, tt.args.key)
			}

			//}
			if got := batch.Has([]byte(tt.name)); got != tt.want {
				t.Errorf("Has() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUpdateBatch_KVs(t *testing.T) {
	type fields struct {
		kvs     map[string][]byte
		rwLock  *sync.RWMutex
		currMap ConcurrentMap
	}
	tests := []struct {
		name   string
		fields fields
		want   map[string][]byte
	}{
		// TODO: Add test cases.
		{
			name: keyStr1,
			fields: fields{
				kvs:     make(map[string][]byte),
				rwLock:  &sync.RWMutex{},
				currMap: New(),
			},
			want: map[string][]byte{
				keyStr1: []byte(keyStr1),
			},
		},
		{
			name: keyStr2,
			fields: fields{
				kvs:     make(map[string][]byte),
				rwLock:  &sync.RWMutex{},
				currMap: New(),
			},
			want: map[string][]byte{
				keyStr2: []byte(keyStr2),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			batch := &UpdateBatch{
				kvs:     tt.fields.kvs,
				rwLock:  tt.fields.rwLock,
				currMap: tt.fields.currMap,
			}
			if tt.name == keyStr1 {
				batch.Put([]byte(tt.name), []byte(tt.name))
			}
			if tt.name == keyStr2 {
				batch.Put([]byte(tt.name), []byte(tt.name))
			}

			if got := batch.KVs(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("KVs() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUpdateBatch_Keys(t *testing.T) {
	type fields struct {
		kvs     map[string][]byte
		rwLock  *sync.RWMutex
		currMap ConcurrentMap
	}
	tests := []struct {
		name   string
		fields fields
		want   []string
	}{
		// TODO: Add test cases.
		{
			name: keyStr1,
			fields: fields{
				kvs:     make(map[string][]byte),
				rwLock:  &sync.RWMutex{},
				currMap: New(),
			},
			want: []string{keyStr1},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			batch := &UpdateBatch{
				kvs:     tt.fields.kvs,
				rwLock:  tt.fields.rwLock,
				currMap: tt.fields.currMap,
			}
			batch.Put([]byte(tt.name), []byte(tt.name))
			if got := batch.Keys(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Keys() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUpdateBatch_Len(t *testing.T) {
	type fields struct {
		kvs     map[string][]byte
		rwLock  *sync.RWMutex
		currMap ConcurrentMap
	}
	tests := []struct {
		name   string
		fields fields
		want   int
	}{
		// TODO: Add test cases.
		{
			name: keyStr1,
			fields: fields{
				kvs:     make(map[string][]byte),
				rwLock:  &sync.RWMutex{},
				currMap: New(),
			},
			want: 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			batch := &UpdateBatch{
				kvs:     tt.fields.kvs,
				rwLock:  tt.fields.rwLock,
				currMap: tt.fields.currMap,
			}
			batch.Put([]byte(tt.name), []byte(tt.name))
			if got := batch.Len(); got != tt.want {
				t.Errorf("Len() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUpdateBatch_Merge(t *testing.T) {
	type fields struct {
		kvs     map[string][]byte
		rwLock  *sync.RWMutex
		currMap ConcurrentMap
	}
	//type args struct {
	//	u protocol.StoreBatcher
	//}
	tests := []struct {
		name   string
		fields fields
		//args   args
	}{
		// TODO: Add test cases.
		{
			name: keyStr1,
			fields: fields{
				kvs:     make(map[string][]byte),
				rwLock:  &sync.RWMutex{},
				currMap: New(),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			batch := &UpdateBatch{
				kvs:     tt.fields.kvs,
				rwLock:  tt.fields.rwLock,
				currMap: tt.fields.currMap,
			}
			batch2 := &UpdateBatch{
				kvs:     tt.fields.kvs,
				rwLock:  tt.fields.rwLock,
				currMap: tt.fields.currMap,
			}
			batch.Merge(batch2)
		})
	}
}

func TestUpdateBatch_Put(t *testing.T) {
	type fields struct {
		kvs     map[string][]byte
		rwLock  *sync.RWMutex
		currMap ConcurrentMap
	}
	type args struct {
		key   []byte
		value []byte
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		// TODO: Add test cases.
		{
			name: keyStr1,
			fields: fields{
				kvs:     make(map[string][]byte),
				rwLock:  &sync.RWMutex{},
				currMap: New(),
			},
			args: args{
				key:   []byte(keyStr1),
				value: []byte("value1"),
			},
		},
		{
			name: keyStr2,
			fields: fields{
				kvs:     make(map[string][]byte),
				rwLock:  &sync.RWMutex{},
				currMap: New(),
			},
			args: args{
				key:   []byte(keyStr2),
				value: nil,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if err := recover(); err != nil {
					if tt.name != keyStr2 {
						t.Errorf("Put() error ,want panic where key=key2,got %v", tt.name)
					}
				}
			}()
			batch := &UpdateBatch{
				kvs:     tt.fields.kvs,
				rwLock:  tt.fields.rwLock,
				currMap: tt.fields.currMap,
			}
			batch.Put(tt.args.key, tt.args.value)
		})
	}
}

func TestUpdateBatch_ReSet(t *testing.T) {
	type fields struct {
		kvs     map[string][]byte
		rwLock  *sync.RWMutex
		currMap ConcurrentMap
	}
	tests := []struct {
		name   string
		fields fields
	}{
		// TODO: Add test cases.
		{
			name: keyStr1,
			fields: fields{
				kvs:     make(map[string][]byte),
				rwLock:  &sync.RWMutex{},
				currMap: New(),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			batch := &UpdateBatch{
				kvs:     tt.fields.kvs,
				rwLock:  tt.fields.rwLock,
				currMap: tt.fields.currMap,
			}
			batch.ReSet()
		})
	}
}

//拆分之后，判断拆分前后的key,val是否一致，个数一致
func TestUpdateBatch_SplitBatch(t *testing.T) {
	type fields struct {
		kvs     map[string][]byte
		rwLock  *sync.RWMutex
		currMap ConcurrentMap
	}
	type args struct {
		batchCnt uint64
	}
	//wantSlice := make([]UpdateBatch,0)
	//wantSlice = append(wantSlice,UpdateBatch{})
	//构建case1
	updateBatch1 := UpdateBatch{
		kvs:     make(map[string][]byte),
		rwLock:  &sync.RWMutex{},
		currMap: New(),
	}
	updateBatch2 := UpdateBatch{
		kvs:     make(map[string][]byte),
		rwLock:  &sync.RWMutex{},
		currMap: New(),
	}

	inputArr1 := make(map[string][]byte)
	for i := 0; i < 2; i++ {
		key := strconv.Itoa(i)
		value := []byte(key)
		inputArr1[key] = value
		updateBatch1.Put([]byte(key), value)
	}

	inputArr2 := make(map[string][]byte)
	for i := 2; i < 4; i++ {
		key := strconv.Itoa(i)
		value := []byte(key)
		inputArr2[key] = value
		updateBatch2.Put([]byte(key), value)
	}

	//构建case2
	updateBatch3 := UpdateBatch{
		kvs:     make(map[string][]byte),
		rwLock:  &sync.RWMutex{},
		currMap: New(),
	}
	updateBatch4 := UpdateBatch{
		kvs:     make(map[string][]byte),
		rwLock:  &sync.RWMutex{},
		currMap: New(),
	}

	inputArr3 := make(map[string][]byte)
	for i := 0; i < 123; i++ {
		key := strconv.Itoa(i)
		value := []byte(key)
		inputArr3[key] = value
		updateBatch3.Put([]byte(key), value)
	}

	inputArr4 := make(map[string][]byte)
	for i := 124; i < 996; i++ {
		key := strconv.Itoa(i)
		value := []byte(key)
		inputArr4[key] = value
		updateBatch4.Put([]byte(key), value)
	}

	tests := []struct {
		name   string
		fields fields
		args   args
		//want   []protocol.StoreBatcher
		want []map[string][]byte
	}{
		// TODO: Add test cases.
		{
			name: keyStr1,
			fields: fields{
				kvs:     make(map[string][]byte),
				rwLock:  &sync.RWMutex{},
				currMap: New(),
			},
			args: args{
				batchCnt: 2,
			},
			want: []map[string][]byte{
				updateBatch1.KVs(),
				updateBatch2.KVs(),
			},
		},
		{
			name: keyStr2,
			fields: fields{
				kvs:     make(map[string][]byte),
				rwLock:  &sync.RWMutex{},
				currMap: New(),
			},
			args: args{
				batchCnt: 5,
			},
			want: []map[string][]byte{
				updateBatch3.KVs(),
				updateBatch4.KVs(),
			},
		},
		{
			name: "key3",
			fields: fields{
				kvs:     make(map[string][]byte),
				rwLock:  &sync.RWMutex{},
				currMap: New(),
			},
			args: args{
				batchCnt: 0,
			},
			want: []map[string][]byte{},
		},
		{
			name: "key4",
			fields: fields{
				kvs:     make(map[string][]byte),
				rwLock:  &sync.RWMutex{},
				currMap: New(),
			},
			args: args{
				batchCnt: 2,
			},
			want: []map[string][]byte{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			batch := &UpdateBatch{
				kvs:     tt.fields.kvs,
				rwLock:  tt.fields.rwLock,
				currMap: tt.fields.currMap,
			}
			//batch.Put([]byte("key1"),[]byte(tt.name))
			//batch.Put([]byte("key2"),[]byte(tt.name))
			if tt.name == keyStr1 {
				batch.Merge(&updateBatch1)
				batch.Merge(&updateBatch2)
			}
			if tt.name == keyStr2 {
				batch.Merge(&updateBatch3)
				batch.Merge(&updateBatch4)
			}
			//batchCnt: 0
			if tt.name == "key3" {
				gotS := batch.SplitBatch(tt.args.batchCnt)
				if len(gotS) != 1 && !reflect.DeepEqual(gotS[0], batch) {
					t.Errorf("SplitBatch() = %v, want %v", batch, gotS[0])
				}
				return
			}
			if tt.name == "key4" {
				batch.Put([]byte("key4"), []byte("key4"))
				//gotS := batch.SplitBatch(tt.args.batchCnt)

			}

			//fmt.Println("batch.Len() =",batch.Len())
			//fmt.Println("len(batch.KVs()) =",len(batch.KVs()))
			//if got := batch.SplitBatch(tt.args.batchCnt); !reflect.DeepEqual(got, tt.want) {
			//	t.Errorf("SplitBatch() = %v, want %v", got, tt.want)
			//}
			//if tt.args.batchCnt == 0 {
			//	gotS := batch.SplitBatch(tt.args.batchCnt)
			//	if len(gotS) != 1 && !reflect.DeepEqual(gotS[0],batch) {
			//		t.Errorf("SplitBatch() = %v, want %v", batch, gotS[0])
			//	}
			//	return
			//}

			got := batch.SplitBatch(tt.args.batchCnt)
			//fmt.Println("len(got) =",len(got))
			//fmt.Println("------------------------------------------------------------------------------")
			//fmt.Println("got[0].KVs() = ",got[0].KVs())
			//fmt.Println("-------------------========================-----------------------------------")
			//fmt.Println("got[1].KVs() = ",got[1].KVs())
			//fmt.Println("------------------------------------------------------------------------------")
			//fmt.Println("got[2].KVs() = ",got[2].KVs())
			//wBatch := &UpdateBatch{}
			//拆分之后，拆分的子batch 与原batch的 key,value 值一致，   k/v 个数一致，则说明拆分正常
			wBatch := NewUpdateBatch()
			for i := 0; i < len(got); i++ {
				for k, v := range got[i].KVs() {
					wBatch.Put([]byte(k), v)
				}
			}
			for k, wantV := range batch.KVs() {
				realV, err := wBatch.Get([]byte(k))
				if err != nil {
					t.Errorf("SplitBatch() error ,want %v,got %v", nil, err)
					return
				}
				if string(wantV) != string(realV) {
					t.Errorf("SplitBatch() error ,want %v,got %v", wantV, realV)
					return
				}

			}
			if wBatch.Len() != batch.Len() {
				t.Errorf("SplitBatch() error ,want %v,got %v", batch.Len(), wBatch.Len())
				return
			}

			//for i:=0 ;i<len(got);i++{
			//	if i == 0 {
			//		if got[0].Len() != 1  {
			//			t.Errorf("SplitBatch() error ,want %v,got %v",1,got[0].Len())
			//		}
			//		get, err := got[0].Get([]byte("key1"));
			//		if err != nil {
			//			t.Errorf("SplitBatch() error ,want %v,got %v",nil,err)
			//		}
			//		if string(get) != "key1" {
			//			t.Errorf("SplitBatch() error ,want %v,got %v", "key1", string(get))
			//		}
			//	}
			//	if i == 1 {
			//		if got[1].Len() != 1  {
			//			t.Errorf("SplitBatch() error ,want %v,got %v",1,got[1].Len())
			//		}
			//		get, err := got[1].Get([]byte("key2"));
			//		if err != nil {
			//			t.Errorf("SplitBatch() error ,want %v,got %v",nil,err)
			//		}
			//		if string(get) != "key2" {
			//			t.Errorf("SplitBatch() error ,want %v,got %v", "key2", string(get))
			//		}
			//	}

			//}
		})
	}
}
