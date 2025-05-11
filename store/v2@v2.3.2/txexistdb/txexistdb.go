/*
 * Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
 *
 * SPDX-License-Identifier: Apache-2.0
 *
 */

package txexistdb

import (
	"chainmaker.org/chainmaker/store/v2/serialization"
)

// TxExistDB provides handle to block and tx instances
// @Description:
type TxExistDB interface {

	// InitGenesis 完成创世块写入
	// @Description:
	// @param genesisBlock
	// @return error
	InitGenesis(genesisBlock *serialization.BlockWithSerializedInfo) error

	// CommitBlock commits the txId and savepoint in an atomic operation
	// @Description:
	// @param blockWithRWSet
	// @param isCache
	// @return error
	CommitBlock(blockWithRWSet *serialization.BlockWithSerializedInfo, isCache bool) error

	// GetLastSavepoint returns the last block height
	// @Description:
	// @return uint64
	// @return error
	GetLastSavepoint() (uint64, error)

	// TxExists returns true if the tx exist, or returns false if none exists.
	// @Description:
	// @param txId
	// @return bool
	// @return error
	TxExists(txId string) (bool, error)

	// Close is used to close database
	// @Description:
	Close()
}
