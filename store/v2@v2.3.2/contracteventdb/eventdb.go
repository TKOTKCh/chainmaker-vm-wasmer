/*
 * Copyright (C) BABEC. All rights reserved.
 * Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
 *
 * SPDX-License-Identifier: Apache-2.0
 */

package contracteventdb

import (
	"chainmaker.org/chainmaker/store/v2/serialization"
)

// ContractEventDB provides handle to contract event
//  @Description:
// This implementation provides a mysql based data model
type ContractEventDB interface {

	// CommitBlock
	//  @Description: commits the event in an atomic operation
	//  @param blockInfo
	//  @param isCache
	//  @return error
	CommitBlock(blockInfo *serialization.BlockWithSerializedInfo, isCache bool) error

	// InitGenesis
	//  @Description: init contract event db
	//  @param genesis
	//  @return error
	InitGenesis(genesis *serialization.BlockWithSerializedInfo) error

	// GetLastSavepoint
	//  @Description: returns the last block height
	//  @return uint64
	//  @return error
	GetLastSavepoint() (uint64, error)

	// Close
	//  @Description: Close is used to close database
	Close()
}
