/*
 * Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
 *
 * SPDX-License-Identifier: Apache-2.0
 *
 */

package types

import (
	"errors"
	"sync"

	bytesconv "chainmaker.org/chainmaker/common/v2/bytehelper"
	"chainmaker.org/chainmaker/protocol/v2"
)

var (
	errValueNotFound = errors.New("not found")
)

// UpdateBatch encloses the details of multiple `updates`
// @Description:
type UpdateBatch struct {
	kvs     map[string][]byte
	rwLock  *sync.RWMutex
	currMap ConcurrentMap
}

// NewUpdateBatch constructs an instance of a Batch
// @Description:
// @return protocol.StoreBatcher
func NewUpdateBatch() protocol.StoreBatcher {
	return &UpdateBatch{
		kvs:     make(map[string][]byte),
		rwLock:  &sync.RWMutex{},
		currMap: New(),
	}
}

// KVs return map, only read
// @Description:
// @receiver batch
// @return map[string][]byte
func (batch *UpdateBatch) KVs() map[string][]byte {
	kvMap := make(map[string][]byte)
	keysArr := batch.Keys()
	for _, key := range keysArr {
		value, err := batch.Get(bytesconv.StringToBytes(key))
		if err == nil {
			kvMap[key] = value
		}
	}
	return kvMap
	//return batch.kvs
	//return batch.currMap.kvs
}

// Keys keys in map
// @Description:
// @receiver batch
// @return []string
func (batch *UpdateBatch) Keys() []string {
	return batch.currMap.Keys()
	//return batch.currMap.kvs
}

// Put adds a KV
// @Description:
// @receiver batch
// @param key
// @param value
func (batch *UpdateBatch) Put(key []byte, value []byte) {
	//batch.rwLock.Lock()
	//defer batch.rwLock.Unlock()
	if value == nil {
		panic("Nil value not allowed")
	}
	//batch.kvs[bytesconv.BytesToString(key)] = value
	//batch.kvs[string(key)] = value
	batch.currMap.Set(bytesconv.BytesToString(key), value)
}

// Delete deletes a Key and associated value
// @Description:
// @receiver batch
// @param key
func (batch *UpdateBatch) Delete(key []byte) {
	//batch.rwLock.Lock()
	//defer batch.rwLock.Unlock()
	//batch.kvs[string(key)] = nil
	//不做物理删除，直接把value 改成 nil
	batch.currMap.Set(bytesconv.BytesToString(key), nil)
}

// Len returns the number of entries in the batch
// @Description:
// @receiver batch
// @return int
func (batch *UpdateBatch) Len() int {
	//return len(batch.kvs)
	return batch.currMap.Count()
}

// Merge merges other kvs to this updateBatch
// @Description:
// @receiver batch
// @param u
func (batch *UpdateBatch) Merge(u protocol.StoreBatcher) {
	batch.rwLock.Lock()
	defer batch.rwLock.Unlock()
	for key, value := range u.KVs() {
		//batch.kvs[key] = value
		batch.currMap.Set(key, value)
	}
}

// Get 根据key，返回value
// @Description:
// @receiver batch
// @param key
// @return []byte
// @return error
func (batch *UpdateBatch) Get(key []byte) ([]byte, error) {
	//batch.rwLock.RLock()
	//defer batch.rwLock.RUnlock()
	//if _, exists := batch.kvs[string(key)]; exists {
	//	return batch.kvs[string(key)], nil
	//}
	if value, exists := batch.currMap.Get(bytesconv.BytesToString(key)); exists {
		v, ok := value.([]byte)
		if ok {
			return v, nil
		}
		//非[]byte类型，返回nil,这种情况key是存在的,value为nil
		return nil, nil
		//return nil, errors.New("value is not []byte")
	}

	return nil, errValueNotFound

}

// Has 判断key是否存在
// @Description:
// @receiver batch
// @param key
// @return bool
func (batch *UpdateBatch) Has(key []byte) bool {
	//batch.rwLock.RLock()
	//defer batch.rwLock.RUnlock()
	if exists := batch.currMap.Has(bytesconv.BytesToString(key)); exists {
		return true
	}
	return false
}

// SplitBatch split other kvs to more updateBatchs division by batchCnt
// @Description:
// batchCnt: 拆分后，每个子batch，最多包含多少个k/v
// @receiver batch
// @param batchCnt
// @return []protocol.StoreBatcher
func (batch *UpdateBatch) SplitBatch(batchCnt uint64) []protocol.StoreBatcher {
	batch.rwLock.Lock()
	defer batch.rwLock.Unlock()

	if batchCnt == 0 {
		return []protocol.StoreBatcher{batch}
	}

	times := uint64(batch.Len()) / batchCnt
	batches := make([]protocol.StoreBatcher, 0, times+2) //time + 2 = (time + 1) + 1(for blkbatch)

	index := uint64(1)
	lastBatchStart := times * batchCnt
	lastBth := NewUpdateBatch()
	bth := NewUpdateBatch()
	for key, value := range batch.KVs() {
		if len(key) == 0 {
			continue
		}
		if len(value) > 5<<20 {
			// if len(value) > 5 MB, will use one single batch for it
			sBth := NewUpdateBatch()
			sBth.Put([]byte(key), value)
			batches = append(batches, sBth)
			continue
		}
		if index <= lastBatchStart {
			if len(value) > 0 {
				bth.Put([]byte(key), value)
			} else {
				bth.Delete([]byte(key))
			}
			if index%batchCnt == 0 {
				batches = append(batches, bth)
				bth = NewUpdateBatch()
			}
		} else if index > lastBatchStart {
			if len(value) > 0 {
				lastBth.Put([]byte(key), value)
			} else {
				lastBth.Delete([]byte(key))
			}
		}
		index++
	}
	batches = append(batches, lastBth)

	return batches
}

// ReSet return map, only read
// @Description:
// @receiver batch
func (batch *UpdateBatch) ReSet() {
	batch.currMap = New()
	batch.kvs = make(map[string][]byte)
}
