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

// SimContextKeyHistoryIterator historyIterator structure of simContext
type SimContextKeyHistoryIterator struct {
	key                []byte
	finalValue         *store.KeyModification
	wSetKeyHistoryIter protocol.KeyHistoryIterator
	dbKeyHistoryIter   protocol.KeyHistoryIterator
	simContext         protocol.TxSimContext
	released           bool
}

// NewSimContextKeyHistoryIterator is used to create a new historyIterator
func NewSimContextKeyHistoryIterator(simContext protocol.TxSimContext, wSetIter,
	dbIter protocol.KeyHistoryIterator,
	key []byte) *SimContextKeyHistoryIterator {

	//construct SimContextKeyHistoryIterator instance
	return &SimContextKeyHistoryIterator{
		key:                key,
		finalValue:         nil,
		wSetKeyHistoryIter: wSetIter,
		dbKeyHistoryIter:   dbIter,
		simContext:         simContext,
		released:           false,
	}
}

// Next move the iter to next and return is there value in next iter
func (iter *SimContextKeyHistoryIterator) Next() bool {
	iter.finalValue = nil
	if iter.dbKeyHistoryIter.Next() {
		//if db key history iterator next not null, get its value
		value, err := iter.dbKeyHistoryIter.Value()
		if err != nil {
			return false
		}
		iter.finalValue = value
		return true
	}

	if iter.wSetKeyHistoryIter.Next() {
		//if write set history iterator next not null, get its value
		value, err := iter.wSetKeyHistoryIter.Value()
		if err != nil {
			return false
		}
		iter.finalValue = value
		return true
	}

	return false
}

// Value return the value of current iter
func (iter *SimContextKeyHistoryIterator) Value() (*store.KeyModification, error) {
	//if the final value is null, return nil
	if iter.finalValue == nil {
		return nil, nil
	}

	contractName := iter.simContext.GetTx().Payload.ContractName
	iter.simContext.PutIntoReadSet(contractName, iter.key, iter.finalValue.Value)
	return iter.finalValue, nil
}

// Release release the iterator
func (iter *SimContextKeyHistoryIterator) Release() {
	//if iterator not released, then release it
	if !iter.released {
		iter.wSetKeyHistoryIter.Release()
		iter.dbKeyHistoryIter.Release()
		iter.released = true
	}
}
