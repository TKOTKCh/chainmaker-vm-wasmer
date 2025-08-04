/*
 Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
   SPDX-License-Identifier: Apache-2.0
*/

package vm

import (
	"fmt"

	"chainmaker.org/chainmaker/common/v2/sortedmap"
	"chainmaker.org/chainmaker/pb-go/v2/store"
)

// WsetIterator historyIterator structure of wasi
type WsetIterator struct {
	stringKeySortedMap *sortedmap.StringKeySortedMap
	//log protocol.Logger
}

// NewWsetIterator is used to create a new iterator for write set
func NewWsetIterator(wsets map[string]interface{}) *WsetIterator {
	return &WsetIterator{
		stringKeySortedMap: sortedmap.NewStringKeySortedMapWithInterfaceData(wsets),
		//log: logger.GetLoggerByChain(logger.MODULE_CORE, chainConf.ChainConfig().ChainId),
	}
}

// Next move the iter to next and return is there value in next iter
func (wi *WsetIterator) Next() bool {
	return wi.stringKeySortedMap.Length() > 0
}

// Value get next element
func (wi *WsetIterator) Value() (*store.KV, error) {
	var kv *store.KV
	var keyStr string
	ok := true
	// get the first row
	wi.stringKeySortedMap.Range(func(key string, val interface{}) (isContinue bool) {
		keyStr = key
		kv, ok = val.(*store.KV)
		return false
	})
	if !ok {
		return nil, fmt.Errorf("get value from wsetIterator failed, value type error")
	}
	wi.stringKeySortedMap.Remove(keyStr)
	return kv, nil
}

// Release release the iterator
func (wi *WsetIterator) Release() {
	// do nothing
}
