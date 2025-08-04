/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package vm

import (
	"fmt"

	"chainmaker.org/chainmaker/common/v2/sortedmap"
	"chainmaker.org/chainmaker/pb-go/v2/store"
)

// WSetKeyHistoryIterator historyIterator structure of wasi
type WSetKeyHistoryIterator struct {
	stringKeySortedMap *sortedmap.StringKeySortedMap
}

// NewWSetKeyHistoryIterator is used to create a new historyIterator for write set
func NewWSetKeyHistoryIterator(wSets map[string]interface{}) *WSetKeyHistoryIterator {
	return &WSetKeyHistoryIterator{
		stringKeySortedMap: sortedmap.NewStringKeySortedMapWithInterfaceData(wSets),
	}
}

// Next move the iter to next and return is there value in next iter
func (iter *WSetKeyHistoryIterator) Next() bool {
	return iter.stringKeySortedMap.Length() > 0
}

// Value get next element
func (iter *WSetKeyHistoryIterator) Value() (*store.KeyModification, error) {
	var km *store.KeyModification
	var keyStr string
	ok := true
	// get the first row
	iter.stringKeySortedMap.Range(
		func(key string, val interface{}) (isContinue bool) {
			keyStr = key
			km, ok = val.(*store.KeyModification)
			return false
		},
	)
	if !ok {
		return nil, fmt.Errorf("get value from wsetIterator failed, value type error")
	}
	iter.stringKeySortedMap.Remove(keyStr)
	return km, nil
}

// Release release the iterator
func (iter *WSetKeyHistoryIterator) Release() {}
