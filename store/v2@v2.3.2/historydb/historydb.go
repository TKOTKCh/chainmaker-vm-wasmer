/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0

*/

package historydb

import (
	"chainmaker.org/chainmaker/store/v2/serialization"
)

// HistoryDB provides handle to rwSets instances
// @Description:
type HistoryDB interface {

	// InitGenesis
	// @Description:
	// @param genesisBlock
	// @return error
	InitGenesis(genesisBlock *serialization.BlockWithSerializedInfo) error

	// CommitBlock
	// @Description: commits the block rwsets in an atomic operation
	// @param blockInfo
	// @param isCache
	// @return error
	CommitBlock(blockInfo *serialization.BlockWithSerializedInfo, isCache bool) error

	// GetHistoryForKey
	// @Description: 获得Key的交易历史
	// @param contractName
	// @param key
	// @return HistoryIterator
	// @return error
	GetHistoryForKey(contractName string, key []byte) (HistoryIterator, error)

	// GetAccountTxHistory
	// @Description:
	// @param account
	// @return HistoryIterator
	// @return error
	GetAccountTxHistory(account []byte) (HistoryIterator, error)

	// GetContractTxHistory
	// @Description:
	// @param contractName
	// @return HistoryIterator
	// @return error
	GetContractTxHistory(contractName string) (HistoryIterator, error)

	// GetLastSavepoint
	// @Description: returns the last block height
	// @return uint64
	// @return error
	GetLastSavepoint() (uint64, error)

	// Close
	// @Description: Close is used to close database
	Close()
}

// BlockHeightTxId struct
// @Description:
type BlockHeightTxId struct {
	BlockHeight uint64
	TxId        string
}

// HistoryIterator interface
// @Description:
type HistoryIterator interface {

	// Next
	// @Description:
	// @return bool
	Next() bool

	// Value
	// @Description:
	// @return *BlockHeightTxId
	// @return error
	Value() (*BlockHeightTxId, error)

	// Release
	// @Description:
	Release()
}
