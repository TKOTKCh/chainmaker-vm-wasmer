/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package vm

import (
	"errors"

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

func historyIterKeyInSimCtxWSet(iter *SimContextKeyHistoryIterator) (bool, error) {
	ctx, ok := iter.simContext.(*txSimContextImpl)
	if !ok {
		return false, errors.New("simContext interface convert failed")
	}

	//交易中又跨合约调用，需要遍历多层调用,比如A->B->C,就有3层
	for _, rwSets := range ctx.txRWSetWithDepth {
		//一层有多个合约调用的情况下，比如A->B后，在A->C，A是0层，B和C都是1层，所以要遍历同层的多个合约
		for _, rws := range rwSets {
			//遍历合约下的每个读写集
			for key := range rws.txWriteKeyMap {
				if string(iter.key) == key {
					return true, nil
				}
			}

		}
	}
	return false, nil
}

// Value return the value of current iter
func (iter *SimContextKeyHistoryIterator) Value() (*store.KeyModification, error) {
	//if the final value is null, return nil
	if iter.finalValue == nil {
		return nil, nil
	}

	contractName := iter.simContext.GetTx().Payload.ContractName
	//iter.simContext.PutIntoReadSet(contractName, iter.key, iter.finalValue.Value)
	if iter.simContext.GetBlockVersion() < v235 {
		//小于235版本，沿用历史逻辑
		iter.simContext.PutIntoReadSet(contractName, iter.key, iter.finalValue.Value)

	} else {
		//大于235版本，且迭代器key没有出现在simContext的写集中，才将kv放入读集
		in, err := historyIterKeyInSimCtxWSet(iter)
		if err != nil {
			return nil, err
		}
		if !in {
			iter.simContext.PutIntoReadSet(contractName, iter.key, iter.finalValue.Value)
		}
	}
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
