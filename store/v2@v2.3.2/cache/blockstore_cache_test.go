/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0

*/

package cache

import (
	"reflect"
	"sync"
	"testing"

	"chainmaker.org/chainmaker/protocol/v2"
	"chainmaker.org/chainmaker/protocol/v2/test"
	"chainmaker.org/chainmaker/store/v2/types"
)

var (
	cacheMap  = make(map[uint64]protocol.StoreBatcher)
	wantBatch = types.NewUpdateBatch()
)

func TestNewStoreCacheMgr(t *testing.T) {
	type args struct {
		chainId              string
		blockWriteBufferSize int
		logger               protocol.Logger
	}
	tests := []struct {
		name string
		args args
		want *StoreCacheMgr
	}{
		// TODO: Add test cases.
		{
			name: "key1",
			args: args{
				chainId:              "chain1",
				blockWriteBufferSize: 1,
				logger:               nil,
			},
			want: &StoreCacheMgr{
				pendingBlockUpdates: cacheMap,
				//blockSizeSem:        semaphore.NewWeighted(int64(1)),
				//blockSizeSem:        nil,
				//cache:               nil,
				cacheSize: 1,
				logger:    nil,
			},
		},
		{
			name: "key2",
			args: args{
				chainId:              "chain2",
				blockWriteBufferSize: 0,
				logger:               nil,
			},
			want: &StoreCacheMgr{
				pendingBlockUpdates: cacheMap,
				//blockSizeSem:        semaphore.NewWeighted(int64(1)),
				//blockSizeSem:        nil,
				//cache:               nil,
				cacheSize: 10,
				logger:    nil,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewStoreCacheMgr(tt.args.chainId, tt.args.blockWriteBufferSize, tt.args.logger); !reflect.DeepEqual(got.pendingBlockUpdates, tt.want.pendingBlockUpdates) {
				//if got.pendingBlockUpdates != tt.want.pendingBlockUpdates {
				t.Errorf("NewStoreCacheMgr() = %v, want %v", got, tt.want)
				//}
			}
		})
	}
}

func TestQuickSort(t *testing.T) {
	type args struct {
		arr []uint64
	}
	tests := []struct {
		name string
		args args
		want []uint64
	}{
		// TODO: Add test cases.
		{
			name: "case1",
			args: args{arr: []uint64{1, 5, 3, 7, 9}},
			want: []uint64{9, 7, 5, 3, 1},
		},
		{
			name: "case2",
			args: args{arr: []uint64{1}},
			want: []uint64{1},
		},
		{
			name: "case3",
			args: args{arr: []uint64{6, 5, 4, 3, 2, 1}},
			want: []uint64{6, 5, 4, 3, 2, 1},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := QuickSort(tt.args.arr); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("QuickSort() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSystemQuickSort(t *testing.T) {
	type args struct {
		arr []uint64
	}
	tests := []struct {
		name string
		args args
		want []uint64
	}{
		// TODO: Add test cases.
		{
			name: "case1",
			args: args{arr: []uint64{1, 5, 3, 7, 9}},
			want: []uint64{9, 7, 5, 3, 1},
		},
		{
			name: "case2",
			args: args{arr: []uint64{1}},
			want: []uint64{1},
		},
		{
			name: "case3",
			args: args{arr: []uint64{6, 5, 4, 3, 2, 1}},
			want: []uint64{6, 5, 4, 3, 2, 1},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			SystemQuickSort(tt.args.arr)
			got := make([]uint64, len(tt.args.arr))
			copy(got, tt.args.arr)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("SystemQuickSort() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestStoreCacheMgr_AddBlock(t *testing.T) {
	type fields struct {
		RWMutex             *sync.RWMutex
		pendingBlockUpdates map[uint64]protocol.StoreBatcher
		//blockSizeSem        *semaphore.Weighted
		//cache               *storeCache
		cacheSize int
		logger    protocol.Logger
	}
	type args struct {
		blockHeight uint64
		updateBatch protocol.StoreBatcher
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		// TODO: Add test cases.
		{
			name: "test1",
			fields: fields{
				RWMutex:             &sync.RWMutex{},
				pendingBlockUpdates: make(map[uint64]protocol.StoreBatcher),
				logger:              test.GoLogger{},
			},
			args: args{
				blockHeight: 1,
				updateBatch: types.NewUpdateBatch(),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mgr := &StoreCacheMgr{
				//RWMutex:             *tt.fields.RWMutex,
				pendingBlockUpdates: tt.fields.pendingBlockUpdates,
				//blockSizeSem:        tt.fields.blockSizeSem,
				//cache:               tt.fields.cache,
				cacheSize: tt.fields.cacheSize,
				logger:    tt.fields.logger,
			}
			mgr.RWMutex = sync.RWMutex{}
			tt.args.updateBatch.Put([]byte(tt.name), []byte(tt.name))
			mgr.AddBlock(0, tt.args.updateBatch)
		})
	}
}

func TestStoreCacheMgr_Clear(t *testing.T) {
	type fields struct {
		RWMutex             *sync.RWMutex
		pendingBlockUpdates map[uint64]protocol.StoreBatcher
		//blockSizeSem        *semaphore.Weighted
		//cache               *storeCache
		cacheSize int
		logger    protocol.Logger
	}
	tests := []struct {
		name   string
		fields fields
	}{
		// TODO: Add test cases.
		{
			name: "clear1",
			fields: fields{
				RWMutex:             &sync.RWMutex{},
				pendingBlockUpdates: make(map[uint64]protocol.StoreBatcher),
				//logger: nil,
				logger: test.GoLogger{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mgr := &StoreCacheMgr{
				//RWMutex:             *tt.fields.RWMutex,
				pendingBlockUpdates: tt.fields.pendingBlockUpdates,
				//blockSizeSem:        tt.fields.blockSizeSem,
				//cache:               tt.fields.cache,
				cacheSize: tt.fields.cacheSize,
				logger:    tt.fields.logger,
			}
			mgr.RWMutex = sync.RWMutex{}
			mgr.Clear()
		})
	}
}

func TestStoreCacheMgr_DelBlock(t *testing.T) {
	type fields struct {
		RWMutex             *sync.RWMutex
		pendingBlockUpdates map[uint64]protocol.StoreBatcher
		//blockSizeSem        *semaphore.Weighted
		//cache               *storeCache
		cacheSize int
		logger    protocol.Logger
	}
	type args struct {
		blockHeight uint64
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		// TODO: Add test cases.
		{
			name: "del1",
			fields: fields{
				RWMutex:             &sync.RWMutex{},
				pendingBlockUpdates: make(map[uint64]protocol.StoreBatcher),
				logger:              test.GoLogger{},
			},
			args: args{blockHeight: 1},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mgr := &StoreCacheMgr{
				//RWMutex:             *tt.fields.RWMutex,
				pendingBlockUpdates: tt.fields.pendingBlockUpdates,
				//blockSizeSem:        tt.fields.blockSizeSem,
				//cache:               tt.fields.cache,
				cacheSize: tt.fields.cacheSize,
				logger:    tt.fields.logger,
			}
			mgr.RWMutex = sync.RWMutex{}
			mgr.AddBlock(0, types.NewUpdateBatch())
			mgr.DelBlock(0)
			mgr.AddBlock(1, types.NewUpdateBatch())
			mgr.DelBlock(2)
		})
	}
}

func TestStoreCacheMgr_Get(t *testing.T) {
	type fields struct {
		RWMutex             *sync.RWMutex
		pendingBlockUpdates map[uint64]protocol.StoreBatcher
		//blockSizeSem        *semaphore.Weighted
		//cache               *storeCache
		cacheSize int
		logger    protocol.Logger
	}
	type args struct {
		key string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   []byte
		want1  bool
	}{
		// TODO: Add test cases.
		{
			name: "get1",
			fields: fields{
				RWMutex:             &sync.RWMutex{},
				pendingBlockUpdates: make(map[uint64]protocol.StoreBatcher),
				logger:              nil,
			},
			args:  args{key: "get1"},
			want:  []byte("get1"),
			want1: true,
		},
		{
			name: "get3",
			fields: fields{
				RWMutex:             &sync.RWMutex{},
				pendingBlockUpdates: make(map[uint64]protocol.StoreBatcher),
				logger:              nil,
			},
			args:  args{key: "get3"},
			want:  nil,
			want1: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mgr := &StoreCacheMgr{
				//RWMutex:             *tt.fields.RWMutex,
				pendingBlockUpdates: tt.fields.pendingBlockUpdates,
				//blockSizeSem:        tt.fields.blockSizeSem,
				//cache:               tt.fields.cache,
				cacheSize: tt.fields.cacheSize,
				logger:    tt.fields.logger,
			}
			mgr.RWMutex = sync.RWMutex{}
			//模拟 块高 大的map没找到，从块高 小的map中找到了key，value
			if tt.name == "get1" {
				batch1 := types.NewUpdateBatch()
				batch1.Put([]byte(tt.args.key), []byte(tt.args.key))
				mgr.AddBlock(1, batch1)
				batch2 := types.NewUpdateBatch()
				batch2.Put([]byte("get2"), []byte(tt.args.key))
				mgr.AddBlock(2, batch2)
			}

			//模拟 没找到的情况
			if tt.name == "get3" {
				batch1 := types.NewUpdateBatch()
				batch1.Put([]byte("get-not-exists1"), []byte(tt.args.key))
				mgr.AddBlock(1, batch1)
				batch2 := types.NewUpdateBatch()
				batch2.Put([]byte("get-not-exists2"), []byte(tt.args.key))
				mgr.AddBlock(2, batch2)
			}

			got, got1 := mgr.Get(tt.args.key)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Get() got = %v, want %v", string(got), string(tt.want))
			}
			if got1 != tt.want1 {
				t.Errorf("Get() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func TestStoreCacheMgr_GetBatch(t *testing.T) {
	type fields struct {
		RWMutex             *sync.RWMutex
		pendingBlockUpdates map[uint64]protocol.StoreBatcher
		//blockSizeSem        *semaphore.Weighted
		//cache               *storeCache
		cacheSize int
		logger    protocol.Logger
	}
	type args struct {
		height uint64
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    protocol.StoreBatcher
		wantErr bool
	}{
		// TODO: Add test cases.
		{
			name: "getBatch1",
			fields: fields{
				RWMutex:             &sync.RWMutex{},
				pendingBlockUpdates: make(map[uint64]protocol.StoreBatcher),
				logger:              nil,
			},
			args:    args{height: 1},
			want:    wantBatch,
			wantErr: true,
		},
		{
			name: "getBatch2",
			fields: fields{
				RWMutex:             &sync.RWMutex{},
				pendingBlockUpdates: make(map[uint64]protocol.StoreBatcher),
				logger:              nil,
			},
			args:    args{height: 2},
			want:    nil,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mgr := &StoreCacheMgr{
				//RWMutex:             *tt.fields.RWMutex,
				pendingBlockUpdates: tt.fields.pendingBlockUpdates,
				//blockSizeSem:        tt.fields.blockSizeSem,
				//cache:               tt.fields.cache,
				cacheSize: tt.fields.cacheSize,
				logger:    tt.fields.logger,
			}
			mgr.RWMutex = sync.RWMutex{}
			if tt.name == "getBatch1" {
				mgr.AddBlock(1, wantBatch)
				got, err := mgr.GetBatch(tt.args.height)
				if (err == nil) != tt.wantErr {
					t.Errorf("GetBatch() error = %v, wantErr %v", err, tt.wantErr)
					return
				}
				if !reflect.DeepEqual(got, tt.want) {
					t.Errorf("GetBatch() got = %v, want %v", got, tt.want)
				}
			}
			//测试获取不到的情况
			if tt.name == "getBatch2" {
				mgr.AddBlock(3, wantBatch)
				got, err := mgr.GetBatch(tt.args.height)
				if (err == nil) != tt.wantErr {
					t.Errorf("GetBatch() error = %v, wantErr %v", err, tt.wantErr)
					return
				}
				if !reflect.DeepEqual(got, tt.want) {
					t.Errorf("GetBatch() got = %v, want %v", got, tt.want)
				}

			}

		})
	}
}

func TestStoreCacheMgr_Has(t *testing.T) {
	type fields struct {
		RWMutex             *sync.RWMutex
		pendingBlockUpdates map[uint64]protocol.StoreBatcher
		//blockSizeSem        *semaphore.Weighted
		//cache               *storeCache
		cacheSize int
		logger    protocol.Logger
	}
	type args struct {
		key string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
		want1  bool
	}{
		// TODO: Add test cases.
		{
			name: "has1",
			fields: fields{
				RWMutex:             &sync.RWMutex{},
				pendingBlockUpdates: make(map[uint64]protocol.StoreBatcher),
				logger:              nil,
			},
			args:  args{key: "has1"},
			want:  false,
			want1: true,
		},
		{
			name: "has3",
			fields: fields{
				RWMutex:             &sync.RWMutex{},
				pendingBlockUpdates: make(map[uint64]protocol.StoreBatcher),
				logger:              nil,
			},
			args:  args{key: "has3"},
			want:  true,
			want1: true,
		},
		{
			name: "has4",
			fields: fields{
				RWMutex:             &sync.RWMutex{},
				pendingBlockUpdates: make(map[uint64]protocol.StoreBatcher),
				logger:              nil,
			},
			args:  args{key: "has4"},
			want:  false,
			want1: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mgr := &StoreCacheMgr{
				//RWMutex:             *tt.fields.RWMutex,
				pendingBlockUpdates: tt.fields.pendingBlockUpdates,
				//blockSizeSem:        tt.fields.blockSizeSem,
				//cache:               tt.fields.cache,
				cacheSize: tt.fields.cacheSize,
				logger:    tt.fields.logger,
			}
			mgr.RWMutex = sync.RWMutex{}

			//测试 未删除，存在 的情况
			if tt.name == "has1" {
				batch1 := types.NewUpdateBatch()
				batch1.Put([]byte(tt.args.key), []byte(tt.args.key))
				mgr.AddBlock(1, batch1)
				batch2 := types.NewUpdateBatch()
				batch2.Put([]byte("exists-key"), []byte(tt.args.key))
				mgr.AddBlock(2, batch2)
				got, got1 := mgr.Has(tt.args.key)
				if got != tt.want {
					t.Errorf("Has() got = %v, want %v", got, tt.want)
				}
				if got1 != tt.want1 {
					t.Errorf("Has() got1 = %v, want %v", got1, tt.want1)
				}
			}
			//测试，已删除，存在
			if tt.name == "has3" {
				batch1 := types.NewUpdateBatch()
				batch1.Put([]byte(tt.args.key), []byte(tt.args.key))
				mgr.AddBlock(1, batch1)

				batch2 := types.NewUpdateBatch()
				batch2.Delete([]byte(tt.args.key))
				mgr.AddBlock(2, batch2)
				got, got1 := mgr.Has(tt.args.key)
				if got != tt.want {
					t.Errorf("Has() got = %v, want %v", got, tt.want)
				}
				if got1 != tt.want1 {
					t.Errorf("Has() got1 = %v, want %v", got1, tt.want1)
				}
			}
			//测试，未删除，不存在
			if tt.name == "has4" {
				batch1 := types.NewUpdateBatch()
				batch1.Put([]byte(tt.args.key), []byte(tt.args.key))
				mgr.AddBlock(1, batch1)

				batch2 := types.NewUpdateBatch()
				batch2.Delete([]byte(tt.args.key))
				mgr.AddBlock(2, batch2)
				got, got1 := mgr.Has("key-not-exists")
				if got != tt.want {
					t.Errorf("Has() got = %v, want %v", got, tt.want)
				}
				if got1 != tt.want1 {
					t.Errorf("Has() got1 = %v, want %v", got1, tt.want1)
				}
			}

		})
	}
}

func TestStoreCacheMgr_HasFromHeight(t *testing.T) {
	type fields struct {
		RWMutex             *sync.RWMutex
		pendingBlockUpdates map[uint64]protocol.StoreBatcher
		//blockSizeSem        *semaphore.Weighted
		//cache               *storeCache
		cacheSize int
		logger    protocol.Logger
	}
	type args struct {
		key string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
		want1  bool
	}{
		// TODO: Add test cases.
		{
			name: "has11",
			fields: fields{
				RWMutex:             &sync.RWMutex{},
				pendingBlockUpdates: make(map[uint64]protocol.StoreBatcher),
				logger:              nil,
			},
			args:  args{key: "has11"},
			want:  false,
			want1: true,
		},
		{
			name: "has33",
			fields: fields{
				RWMutex:             &sync.RWMutex{},
				pendingBlockUpdates: make(map[uint64]protocol.StoreBatcher),
				logger:              nil,
			},
			args:  args{key: "has33"},
			want:  true,
			want1: true,
		},
		{
			name: "has44",
			fields: fields{
				RWMutex:             &sync.RWMutex{},
				pendingBlockUpdates: make(map[uint64]protocol.StoreBatcher),
				logger:              nil,
			},
			args:  args{key: "has44"},
			want:  false,
			want1: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mgr := &StoreCacheMgr{
				//RWMutex:             *tt.fields.RWMutex,
				pendingBlockUpdates: tt.fields.pendingBlockUpdates,
				//blockSizeSem:        tt.fields.blockSizeSem,
				//cache:               tt.fields.cache,
				cacheSize: tt.fields.cacheSize,
				logger:    tt.fields.logger,
			}
			mgr.RWMutex = sync.RWMutex{}

			//测试 未删除，存在 的情况
			if tt.name == "has11" {
				batch1 := types.NewUpdateBatch()
				batch1.Put([]byte(tt.args.key), []byte(tt.args.key))
				mgr.AddBlock(1, batch1)
				batch2 := types.NewUpdateBatch()
				batch2.Put([]byte("exists-key"), []byte(tt.args.key))
				mgr.AddBlock(2, batch2)
				got, got1 := mgr.HasFromHeight(tt.args.key, 0, 100)
				if got != tt.want {
					t.Errorf("Has() got = %v, want %v", got, tt.want)
				}
				if got1 != tt.want1 {
					t.Errorf("Has() got1 = %v, want %v", got1, tt.want1)
				}
			}
			//测试，已删除，存在
			if tt.name == "has33" {
				batch1 := types.NewUpdateBatch()
				batch1.Put([]byte(tt.args.key), []byte(tt.args.key))
				mgr.AddBlock(1, batch1)

				batch2 := types.NewUpdateBatch()
				batch2.Delete([]byte(tt.args.key))
				mgr.AddBlock(2, batch2)
				got, got1 := mgr.HasFromHeight(tt.args.key, 0, 100)
				if got != tt.want {
					t.Errorf("Has() got = %v, want %v", got, tt.want)
				}
				if got1 != tt.want1 {
					t.Errorf("Has() got1 = %v, want %v", got1, tt.want1)
				}
			}
			//测试，未删除，不存在
			if tt.name == "has44" {
				batch1 := types.NewUpdateBatch()
				batch1.Put([]byte(tt.args.key), []byte(tt.args.key))
				mgr.AddBlock(1, batch1)

				batch2 := types.NewUpdateBatch()
				batch2.Delete([]byte(tt.args.key))
				mgr.AddBlock(2, batch2)
				got, got1 := mgr.HasFromHeight("key-not-exists", 0, 100)
				if got != tt.want {
					t.Errorf("Has() got = %v, want %v", got, tt.want)
				}
				if got1 != tt.want1 {
					t.Errorf("Has() got1 = %v, want %v", got1, tt.want1)
				}
			}

		})
	}
}

func TestStoreCacheMgr_LockForFlush(t *testing.T) {
	type fields struct {
		RWMutex             *sync.RWMutex
		pendingBlockUpdates map[uint64]protocol.StoreBatcher
		//blockSizeSem        *semaphore.Weighted
		//cache               *storeCache
		cacheSize int
		logger    protocol.Logger
	}
	tests := []struct {
		name   string
		fields fields
	}{
		// TODO: Add test cases.
		{
			name: "lockForFlush",
			fields: fields{
				RWMutex:             &sync.RWMutex{},
				pendingBlockUpdates: make(map[uint64]protocol.StoreBatcher),
				logger:              nil,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mgr := &StoreCacheMgr{
				//RWMutex:             *tt.fields.RWMutex,
				pendingBlockUpdates: tt.fields.pendingBlockUpdates,
				//blockSizeSem:        tt.fields.blockSizeSem,
				//cache:               tt.fields.cache,
				cacheSize: tt.fields.cacheSize,
				logger:    tt.fields.logger,
			}
			mgr.RWMutex = sync.RWMutex{}
			mgr.LockForFlush()
			defer mgr.UnLockFlush()
		})
	}
}

func TestStoreCacheMgr_UnLockFlush(t *testing.T) {
	type fields struct {
		RWMutex             *sync.RWMutex
		pendingBlockUpdates map[uint64]protocol.StoreBatcher
		//blockSizeSem        *semaphore.Weighted
		//cache               *storeCache
		cacheSize int
		logger    protocol.Logger
	}
	tests := []struct {
		name   string
		fields fields
	}{
		// TODO: Add test cases.
		{
			name: "unLockForFlush",
			fields: fields{
				RWMutex:             &sync.RWMutex{},
				pendingBlockUpdates: make(map[uint64]protocol.StoreBatcher),
				logger:              nil,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mgr := &StoreCacheMgr{
				//RWMutex:             *tt.fields.RWMutex,
				pendingBlockUpdates: tt.fields.pendingBlockUpdates,
				//blockSizeSem:        tt.fields.blockSizeSem,
				//cache:               tt.fields.cache,
				cacheSize: tt.fields.cacheSize,
				logger:    tt.fields.logger,
			}
			mgr.RWMutex = sync.RWMutex{}
			mgr.LockForFlush()
			defer mgr.UnLockFlush()
		})
	}
}

func TestStoreCacheMgr_getPendingBlockSize(t *testing.T) {
	type fields struct {
		RWMutex             *sync.RWMutex
		pendingBlockUpdates map[uint64]protocol.StoreBatcher
		//blockSizeSem        *semaphore.Weighted
		//cache               *storeCache
		cacheSize int
		logger    protocol.Logger
	}
	tests := []struct {
		name   string
		fields fields
		want   int
	}{
		// TODO: Add test cases.
		{
			name: "pedning",
			fields: fields{
				RWMutex:             &sync.RWMutex{},
				pendingBlockUpdates: make(map[uint64]protocol.StoreBatcher),
				logger:              nil,
			},
			want: 10,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mgr := &StoreCacheMgr{
				//RWMutex:             *tt.fields.RWMutex,
				pendingBlockUpdates: tt.fields.pendingBlockUpdates,
				//blockSizeSem:        tt.fields.blockSizeSem,
				//cache:               tt.fields.cache,
				cacheSize: tt.fields.cacheSize,
				logger:    tt.fields.logger,
			}
			mgr.RWMutex = sync.RWMutex{}
			for i := 0; i < 10; i++ {
				mgr.AddBlock(uint64(i), wantBatch)
			}
			if got := mgr.getPendingBlockSize(); got != tt.want {
				t.Errorf("getPendingBlockSize() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestStoreCacheMgr_KVRange(t *testing.T) {
	type fields struct {
		RWMutex             *sync.RWMutex
		pendingBlockUpdates map[uint64]protocol.StoreBatcher
		cacheSize           int
		logger              protocol.Logger
	}
	type args struct {
		startKey []byte
		endKey   []byte
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    map[string][]byte
		wantErr bool
	}{
		// TODO: Add test cases.
		{
			name: "kvrange1",
			fields: fields{
				RWMutex:             &sync.RWMutex{},
				pendingBlockUpdates: make(map[uint64]protocol.StoreBatcher),
				logger:              nil,
			},
			args: args{
				startKey: []byte("a"),
				endKey:   []byte("d"),
			},
			want: map[string][]byte{
				"a": []byte("a"),
				"b": []byte("b"),
				"c": []byte("c"),
				//"d":[]byte("d"),
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mgr := &StoreCacheMgr{
				//RWMutex:             *tt.fields.RWMutex,
				pendingBlockUpdates: tt.fields.pendingBlockUpdates,
				cacheSize:           tt.fields.cacheSize,
				logger:              tt.fields.logger,
			}
			mgr.RWMutex = sync.RWMutex{}
			batch := types.NewUpdateBatch()
			for k, v := range tt.want {
				batch.Put([]byte(k), v)
			}
			mgr.AddBlock(1, batch)
			got, err := mgr.KVRange(tt.args.startKey, tt.args.endKey)
			if (err != nil) != tt.wantErr {
				t.Errorf("KVRange() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("KVRange() got = %v, want %v", got, tt.want)
			}
		})
	}
}
