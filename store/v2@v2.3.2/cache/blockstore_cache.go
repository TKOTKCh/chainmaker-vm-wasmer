/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0

*/

package cache

import (
	"errors"
	"sync"

	"github.com/gogo/protobuf/sortkeys"

	"chainmaker.org/chainmaker/protocol/v2"
)

const defaultMaxBlockSize = 10

var (
	errValueNotFound = errors.New("not found")
)

// StoreCacheMgr provide handle to cache instances
//  @Description:
type StoreCacheMgr struct {
	sync.RWMutex
	pendingBlockUpdates map[uint64]protocol.StoreBatcher
	//blockSizeSem        *semaphore.Weighted
	//cache               *storeCache
	cacheSize int //block size in cache, if cache size <= 0, use defalut size = 10

	logger protocol.Logger
}

// NewStoreCacheMgr construct a new `StoreCacheMgr` with given chainId
//  @Description:
//  @param chainId
//  @param blockWriteBufferSize
//  @param logger
//  @return *StoreCacheMgr
func NewStoreCacheMgr(chainId string, blockWriteBufferSize int, logger protocol.Logger) *StoreCacheMgr {
	if blockWriteBufferSize <= 0 {
		blockWriteBufferSize = defaultMaxBlockSize
	}
	storeCacheMgr := &StoreCacheMgr{
		pendingBlockUpdates: make(map[uint64]protocol.StoreBatcher),
		//blockSizeSem:        semaphore.NewWeighted(int64(blockWriteBufferSize)),
		//cache:               newStoreCache(),
		cacheSize: blockWriteBufferSize,
		logger:    logger,
	}
	return storeCacheMgr
}

// AddBlock cache a block with given block height and update batch
//  @Description:
//  @receiver mgr
//  @param blockHeight
//  @param updateBatch
func (mgr *StoreCacheMgr) AddBlock(blockHeight uint64, updateBatch protocol.StoreBatcher) {
	//wait for semaphore
	//err := mgr.blockSizeSem.Acquire(context.Background(), 1)
	//if err != nil {
	//	mgr.logger.Error(err.Error())
	//}
	mgr.Lock()
	defer mgr.Unlock()
	mgr.pendingBlockUpdates[blockHeight] = updateBatch

	//不需要更新 treeMap了
	//update cache
	//mgr.cache.addBatch(updateBatch)
	if mgr.logger != nil {
		mgr.logger.Debugf("add block[%d] to cache, block size:%d", blockHeight, mgr.getPendingBlockSize())
	}
	//更新 最后一次写的block 的高度，到 StoreCacheMgr 中
	//mgr.lastUpdateKey = blockHeight
}

// DelBlock delete block for the given block height
//  @Description:
//  @receiver mgr
//  @param blockHeight
func (mgr *StoreCacheMgr) DelBlock(blockHeight uint64) {
	//release semaphore
	//mgr.blockSizeSem.Release(1)
	mgr.Lock()
	defer mgr.Unlock()
	//batch, exist := mgr.pendingBlockUpdates[blockHeight]
	//if !exist {
	//	return
	//}
	//不需要删除 treeMap 了
	//mgr.cache.delBatch(batch)
	//直接删除 map
	delete(mgr.pendingBlockUpdates, blockHeight)
	if mgr.logger != nil {
		mgr.logger.Debugf("del block[%d] from cache, block size:%d", blockHeight, mgr.getPendingBlockSize())
	}
}

// Get returns value if the key in cache, or returns nil if none exists.
//  @Description:
//  @receiver mgr
//  @param key
//  @return []byte
//  @return bool
func (mgr *StoreCacheMgr) Get(key string) ([]byte, bool) {
	mgr.RLock()
	defer mgr.RUnlock()
	//pendingBlockUpdates，按照块高，从大到小，获得对应map，在map中查找key，如果key存在，则返回
	keysArr := []uint64{}
	for key := range mgr.pendingBlockUpdates {
		keysArr = append(keysArr, key)
	}
	//快高从大到小排序
	//keysArr = QuickSort(keysArr)
	SystemQuickSort(keysArr)
	for i := 0; i < len(keysArr); i++ {
		if storageCache, exists := mgr.pendingBlockUpdates[keysArr[i]]; exists {
			v, err := storageCache.Get([]byte(key))
			//如果没找到，从cache中找下一个块高对应的key/value
			if err != nil {
				if err.Error() == errValueNotFound.Error() {
					continue
				}
			}
			if err != nil {
				return nil, false
			}
			return v, true
		}
	}
	//如果每个块高，对应的map,中都不存在对应的key，则返回nil,false
	return nil, false
	//if _, exists := mgr.pendingBlockUpdates[mgr.lastUpdateKey]; exists {
	//	v, err := mgr.pendingBlockUpdates[mgr.lastUpdateKey].Get([]byte(key))
	//	if err != nil {
	//		return nil, false
	//	}
	//	return v, true
	//}
	//return nil, false
	//keysArr := []uint64{}
	//for key,_ := range(mgr.pendingBlockUpdates){
	//	keysArr = append(keysArr,key)
	//}

	//return mgr.cache.get(key)

}

// Has returns true if the key in cache, or returns false if none exists.
// 如果这个key 对应value 是 nil，说明这个key被删除了
// 所以查找到第一个key，要判断 这个key 是否是被删除的
//  @Description:
//  @receiver mgr
//  @param key
//  @return bool isDelete
//  @return bool isExist
func (mgr *StoreCacheMgr) Has(key string) (bool, bool) {
	mgr.RLock()
	defer mgr.RUnlock()
	//pendingBlockUpdates，按照块高，从大到小，获得对应map，在map中查找key，如果key存在，则返回
	keysArr := []uint64{}
	for key := range mgr.pendingBlockUpdates {
		keysArr = append(keysArr, key)
	}
	//快高从大到小排序
	//keysArr = QuickSort(keysArr)
	SystemQuickSort(keysArr)
	for i := 0; i < len(keysArr); i++ {
		if _, exists := mgr.pendingBlockUpdates[keysArr[i]]; exists {
			v, err := mgr.pendingBlockUpdates[keysArr[i]].Get([]byte(key))
			//如果没找到，从cache中找下一个块高对应的key/value
			if err != nil {
				if err.Error() == errValueNotFound.Error() {
					continue
				}
			}
			//如果出现其他错误，直接报错
			if err != nil {
				panic(err)
			}
			//未删除，存在
			if v != nil {
				return false, true

			}
			//已删除，存在
			return true, true
		}
	}
	//未删除，不存在
	return false, false
	//
	//if _, exists := mgr.pendingBlockUpdates[mgr.lastUpdateKey]; exists {
	//	has := mgr.pendingBlockUpdates[mgr.lastUpdateKey].Has([]byte(key))
	//	if has {
	//		//未删除，存在
	//		return false, true
	//	}
	//	return true, false
	//
	//}
	//return true, false

	//需要修改，直接读取map，而不用读treemap
	//return mgr.cache.has(key)
}

// HasFromHeight returns true if the key in cache, or returns false if none exists by given startHeight, endHeight.
// 如果这个key 对应value 是 nil，说明这个key被删除了
// 所以查找到第一个key，要判断 这个key 是否是被删除的
//  @Description:
//  @receiver mgr
//  @param key, startHeight, endHeight
//  @return bool isDelete
//  @return bool isExist
func (mgr *StoreCacheMgr) HasFromHeight(key string, startHeight uint64, endHeight uint64) (bool, bool) {
	mgr.RLock()
	defer mgr.RUnlock()
	//pendingBlockUpdates，按照块高，从大到小，获得对应map，在map中查找key，如果key存在，则返回
	for i := endHeight + 1; i >= startHeight+1; i-- {
		if _, exists := mgr.pendingBlockUpdates[i-1]; exists {
			v, err := mgr.pendingBlockUpdates[i-1].Get([]byte(key))
			//如果没找到，从cache中找下一个块高对应的key/value
			if err != nil {
				if err.Error() == errValueNotFound.Error() {
					continue
				}
			}
			//如果出现其他错误，直接报错
			if err != nil {
				panic(err)
			}
			//未删除，存在
			if v != nil {
				return false, true

			}
			//已删除，存在
			return true, true
		}
	}
	//未删除，不存在
	return false, false
}

// Clear  清除缓存，目前未做任何清除操作
//  @Description:
//  @receiver mgr
func (mgr *StoreCacheMgr) Clear() {
	if mgr.logger != nil {
		mgr.logger.Infof("Clear")
	}
	//return
	//mgr.cache.clear()
}

// LockForFlush used to lock cache until all cache item be flushed to db
//  @Description:
//  @receiver mgr
func (mgr *StoreCacheMgr) LockForFlush() {
	if mgr.logger != nil {
		mgr.logger.Infof("LockForFlush")
	}
	//return
	//if mgr.blockSizeSem != nil {
	//	err := mgr.blockSizeSem.Acquire(context.Background(), defaultMaxBlockSize)
	//	if err != nil {
	//		mgr.logger.Error(err.Error())
	//	}
	//}
}

// UnLockFlush  used to unlock cache by release all semaphore
//  @Description:
//  @receiver mgr
func (mgr *StoreCacheMgr) UnLockFlush() {
	if mgr.logger != nil {
		mgr.logger.Infof("UnLockFlush")
	}
	//return
	//if mgr.blockSizeSem != nil {
	//	mgr.blockSizeSem.Release(defaultMaxBlockSize)
	//}
}

//  getPendingBlockSize add next time
//  @Description:
//  @receiver mgr
//  @return int
func (mgr *StoreCacheMgr) getPendingBlockSize() int {
	return len(mgr.pendingBlockUpdates)
}

// KVRange  get data from mgr , [startKey,endKey)
//  @Description:
//  @receiver mgr
//  @param startKey
//  @param endKey
//  @return map[string][]byte
//  @return error
func (mgr *StoreCacheMgr) KVRange(startKey []byte, endKey []byte) (map[string][]byte, error) {
	keyMap := make(map[string][]byte)
	//得到[startKey,endKey) 的 keys
	for _, batch := range mgr.pendingBlockUpdates {
		//keys := make(map[string][]byte)
		batchKV := batch.KVs()
		for k := range batchKV {
			if k >= string(startKey) && k < string(endKey) {
				keyMap[k] = nil
			}
		}
	}
	//得到对应 value
	for k := range keyMap {
		if getV, exist := mgr.Get(k); exist {
			keyMap[k] = getV
		} else {
			delete(keyMap, k)
		}

	}

	return keyMap, nil
}

// GetBatch 根据块高，返回 块对应的cache
//  @Description:
//  @receiver mgr
//  @param height
//  @return protocol.StoreBatcher
//  @return error
func (mgr *StoreCacheMgr) GetBatch(height uint64) (protocol.StoreBatcher, error) {
	mgr.RLock()
	defer mgr.RUnlock()
	if v, exists := mgr.pendingBlockUpdates[height]; exists {
		return v, nil
	}

	return nil, errors.New("not found")
}

//type storeCache struct {
//	table *treemap.Map
//}

//todo:这个可以删除了，从创建入口删除
//func newStoreCache() *storeCache {
//	storeCache := &storeCache{
//		table: treemap.NewWithStringComparator(),
//	}
//	return storeCache
//}

/*
func (c *storeCache) addBatch(batch protocol.StoreBatcher) {
	for key, value := range batch.KVs() {
		c.table.Put(key, value)
	}
}

func (c *storeCache) delBatch(batch protocol.StoreBatcher) {
	for key := range batch.KVs() {
		c.table.Remove(key)
	}
}

func (c *storeCache) get(key string) ([]byte, bool) {
	if value, exist := c.table.Get(key); exist {
		result, ok := value.([]byte)
		if !ok {
			panic("type err: value is not []byte")
		}
		return result, true
	}
	return nil, false
}

// Has returns (isDelete, exist)
// if key exist in cache, exist = true
// if key exist in cache and value == nil, isDelete = true
func (c *storeCache) has(key string) (bool, bool) {
	value, exist := c.get(key)
	if exist {
		return value == nil, true
	}
	return false, false
}

*/
//todo: 删除的 treeMap，可以废弃了
//func (c *storeCache) clear() {
//	if c == nil {
//		return
//	}
//	if c.table != nil {
//		c.table.Clear()
//	}
//}

//func (c *storeCache) len() int {
//	return c.table.Size()
//}

// QuickSort 快排，获得从小到大排序后的结果，返回
//  @Description:
//  @param arr
//  @return []uint64
func QuickSort(arr []uint64) []uint64 {
	if len(arr) <= 1 {
		return arr
	}
	if len(arr) == 2 {
		if arr[0] < arr[1] {
			return append([]uint64{}, arr[1], arr[0])
		}
	}
	var (
		left, middle, right []uint64
	)
	left, middle, right = []uint64{}, []uint64{}, []uint64{}
	mid := arr[len(arr)/2]
	for i := 0; i < len(arr); i++ {
		if arr[i] > mid {
			left = append(left, arr[i])
		}
		if arr[i] == mid {
			middle = append(middle, arr[i])

		}
		if arr[i] < mid {
			right = append(right, arr[i])
		}
	}
	return append(append(QuickSort(left), middle...), QuickSort(right)...)
}

// SystemQuickSort 快排，获得从大到小排序后的结果，返回
//  @Description:
//  @param arr
//  @return []uint64
func SystemQuickSort(arr []uint64) {
	sortkeys.Uint64s(arr)

	//reverse arr
	for i := 0; i < len(arr)/2; i++ {
		tmp := arr[i]
		arr[i] = arr[len(arr)-1-i]
		arr[len(arr)-1-i] = tmp
	}
}
