/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
SPDX-License-Identifier: Apache-2.0
*/

package vm

import (
	"chainmaker.org/chainmaker/pb-go/v2/store"
	"chainmaker.org/chainmaker/protocol/v2"
)

// SimContextIterator iterator structure of simContext
type SimContextIterator struct {
	wsetValueCache *store.KV
	dbValueCache   *store.KV
	finalValue     *store.KV
	wsetIter       protocol.StateIterator
	dbIter         protocol.StateIterator
	simContext     protocol.TxSimContext
	released       bool
}

// NewSimContextIterator is used to create a new iterator
func NewSimContextIterator(simContext protocol.TxSimContext, wsetIter,
	dbIter protocol.StateIterator) *SimContextIterator {
	//construct SimContextIterator instance
	return &SimContextIterator{
		wsetValueCache: nil,
		dbValueCache:   nil,
		finalValue:     nil,
		wsetIter:       wsetIter,
		dbIter:         dbIter,
		simContext:     simContext,
		released:       false,
	}
}

// Next move the iter to next and return is there value in next iter
func (sci *SimContextIterator) Next() bool {
	if sci.wsetValueCache == nil {
		//write set iterator check if write set cache is null
		if sci.wsetIter.Next() {
			//if iterator next not empty, get it's value
			value, err := sci.wsetIter.Value()
			if err != nil {
				//sci.log.Error("get value from wsetIter failed, ", err)
				return false
			}
			sci.wsetValueCache = value
		}
	}

	if sci.dbValueCache == nil {
		//db iterator check if db value cahche is null
		if sci.dbIter.Next() {
			//if db iterator next not null, get it's value
			value, err := sci.dbIter.Value()
			if err != nil {
				//sci.log.Error("get value from dbIter failed, ", err)
				return false
			}
			sci.dbValueCache = value
		}
	}

	//write set value cache and db value cache all null, return false
	if sci.wsetValueCache == nil && sci.dbValueCache == nil {
		return false
	}

	var resultCache *store.KV
	//none of them null, then compare their key
	if sci.wsetValueCache != nil && sci.dbValueCache != nil {
		if string(sci.wsetValueCache.Key) == string(sci.dbValueCache.Key) {
			sci.dbValueCache = nil
			resultCache = sci.wsetValueCache
			sci.wsetValueCache = nil
		} else if string(sci.wsetValueCache.Key) < string(sci.dbValueCache.Key) {
			resultCache = sci.wsetValueCache
			sci.wsetValueCache = nil
		} else {
			resultCache = sci.dbValueCache
			sci.dbValueCache = nil
		}
	} else if sci.wsetValueCache != nil {
		resultCache = sci.wsetValueCache
		sci.wsetValueCache = nil
	} else if sci.dbValueCache != nil {
		resultCache = sci.dbValueCache
		sci.dbValueCache = nil
	}
	sci.finalValue = resultCache
	return true
}

// Value return the value of current iter
func (sci *SimContextIterator) Value() (*store.KV, error) {
	if sci.finalValue == nil {
		return nil, nil
	}
	contractName := sci.simContext.GetTx().Payload.ContractName
	sci.simContext.PutIntoReadSet(contractName, sci.finalValue.Key, sci.finalValue.Value)

	return sci.finalValue, nil
}

// Release release the iterator
func (sci *SimContextIterator) Release() {
	if !sci.released {
		sci.wsetIter.Release()
		sci.dbIter.Release()
		sci.released = true
	}
}
