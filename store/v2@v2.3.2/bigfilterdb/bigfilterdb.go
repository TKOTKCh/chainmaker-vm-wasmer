/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0

*/

// Package bigfilterdb package
package bigfilterdb

import (
	"chainmaker.org/chainmaker/store/v2/serialization"
)

// BigFilterDB provides handle to block and tx instances
type BigFilterDB interface {

	// InitGenesis
	//  @Description: 初始化，写入一个创世区块:0号区块
	//  @param genesisBlock
	//  @return error
	InitGenesis(genesisBlock *serialization.BlockWithSerializedInfo) error

	// CommitBlock
	//  @Description: commits the txId and savepoint in an atomic operation
	//  @param blockWithRWSet
	//  @param isCache
	//  @return error
	CommitBlock(blockWithRWSet *serialization.BlockWithSerializedInfo, isCache bool) error

	// GetLastSavepoint
	//  @Description: returns the last block height
	//  @return uint64
	//  @return error
	GetLastSavepoint() (uint64, error)

	// TxExists
	//  @Description: if not exists  returns (false,false,nil) . if exists return (true,_,nil).
	// If probably exists return (false,true,nil)
	//  @param txId
	//  @return bool
	//  @return bool
	//  @return error
	TxExists(txId string) (bool, bool, error)

	// Close
	//  @Description: is used to close database
	Close()
}
