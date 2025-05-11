/*
 * Copyright (C) BABEC. All rights reserved.
 * Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
 *
 * SPDX-License-Identifier: Apache-2.0
 */

package types

import (
	"bytes"

	storePb "chainmaker.org/chainmaker/pb-go/v2/store"
	"chainmaker.org/chainmaker/store/v2/historydb"
	"chainmaker.org/chainmaker/store/v2/resultdb"
)

// HistoryIteratorImpl add next time
// @Description:
type HistoryIteratorImpl struct {
	contractName string
	key          []byte
	dbItr        historydb.HistoryIterator
	resultStore  resultdb.ResultDB
	blockStore   BlockGetter
}

// BlockGetter add next time
// @Description:
type BlockGetter interface {

	// GetTxInfoOnly add next time
	// @Description:
	// @param txId
	// @return *storePb.TransactionStoreInfo
	// @return error
	GetTxInfoOnly(txId string) (*storePb.TransactionStoreInfo, error)
}

// NewHistoryIterator add next time
// @Description:
// @param contractName
// @param key
// @param dbItr
// @param resultStore
// @param blockStore
// @return *HistoryIteratorImpl
func NewHistoryIterator(contractName string, key []byte, dbItr historydb.HistoryIterator,
	resultStore resultdb.ResultDB, blockStore BlockGetter) *HistoryIteratorImpl {
	return &HistoryIteratorImpl{
		contractName: contractName,
		key:          key,
		dbItr:        dbItr,
		resultStore:  resultStore,
		blockStore:   blockStore,
	}
}

// Next add next time
// @Description:
// @receiver hs
// @return bool
func (hs *HistoryIteratorImpl) Next() bool {
	return hs.dbItr.Next()
}

// Value add next time
// @Description:
// @receiver hs
// @return *storePb.KeyModification
// @return error
func (hs *HistoryIteratorImpl) Value() (*storePb.KeyModification, error) {

	txId, _ := hs.dbItr.Value()
	result := storePb.KeyModification{
		TxId:     txId.TxId,
		IsDelete: false,
	}
	rwset, _ := hs.resultStore.GetTxRWSet(txId.TxId)
	for _, wset := range rwset.TxWrites {
		if bytes.Equal(wset.Key, hs.key) && wset.ContractName == hs.contractName {
			result.Value = wset.Value
		}
	}
	if len(result.Value) == 0 {
		result.IsDelete = true
	}
	tx, err := hs.blockStore.GetTxInfoOnly(txId.TxId)
	if err != nil {
		return nil, err
	}
	result.Timestamp = tx.BlockTimestamp
	result.BlockHeight = tx.BlockHeight
	return &result, nil
}

// Release add next time
// @Description:
// @receiver hs
func (hs *HistoryIteratorImpl) Release() {
	hs.dbItr.Release()
}

// TxHistoryIteratorImpl add next time
// @Description:
type TxHistoryIteratorImpl struct {
	dbItr      historydb.HistoryIterator
	blockStore BlockGetter
}

// NewTxHistoryIterator add next time
// @Description:
// @param dbItr
// @param blockStore
// @return *TxHistoryIteratorImpl
func NewTxHistoryIterator(dbItr historydb.HistoryIterator, blockStore BlockGetter) *TxHistoryIteratorImpl {
	return &TxHistoryIteratorImpl{
		dbItr:      dbItr,
		blockStore: blockStore,
	}
}

// Next add next time
// @Description:
// @receiver hs
// @return bool
func (hs *TxHistoryIteratorImpl) Next() bool {
	return hs.dbItr.Next()
}

// Value add next time
// @Description:
// @receiver hs
// @return *storePb.TxHistory
// @return error
func (hs *TxHistoryIteratorImpl) Value() (*storePb.TxHistory, error) {
	txId, _ := hs.dbItr.Value()
	result := storePb.TxHistory{
		TxId:        txId.TxId,
		BlockHeight: txId.BlockHeight,
	}
	tx, _ := hs.blockStore.GetTxInfoOnly(txId.TxId)
	result.Timestamp = tx.BlockTimestamp
	result.BlockHash = tx.BlockHash
	return &result, nil
}

// Release add next time
// @Description:
// @receiver hs
func (hs *TxHistoryIteratorImpl) Release() {
	hs.dbItr.Release()
}
