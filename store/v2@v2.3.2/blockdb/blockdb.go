/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0

*/

package blockdb

import (
	commonPb "chainmaker.org/chainmaker/pb-go/v2/common"
	storePb "chainmaker.org/chainmaker/pb-go/v2/store"
	"chainmaker.org/chainmaker/store/v2/serialization"
)

// BlockDB provides handle to block and tx instances
type BlockDB interface {

	// InitGenesis 完成创世区块的写入
	//  @Description:
	//  @param genesisBlock
	//  @return error
	InitGenesis(genesisBlock *serialization.BlockWithSerializedInfo) error

	// CommitBlock commits the block and the corresponding rwsets in an atomic operation
	//  @Description:
	//  @param blockInfo
	//  @param isCache
	//  @return error
	CommitBlock(blockInfo *serialization.BlockWithSerializedInfo, isCache bool) error

	// BlockExists returns true if the block hash exist, or returns false if none exists.
	//  @Description:
	//  @param blockHash
	//  @return bool
	//  @return error
	BlockExists(blockHash []byte) (bool, error)

	// GetBlockByHash returns a block given it's hash, or returns nil if none exists.
	//  @Description:
	//  @param blockHash
	//  @return *commonPb.Block
	//  @return error
	GetBlockByHash(blockHash []byte) (*commonPb.Block, error)

	// GetHeightByHash returns a block height given it's hash, or returns nil if none exists.
	//  @Description:
	//  @param blockHash
	//  @return uint64
	//  @return error
	GetHeightByHash(blockHash []byte) (uint64, error)

	// GetBlockHeaderByHeight returns a block header by given it's height, or returns nil if none exists.
	//  @Description:
	//  @param height
	//  @return *commonPb.BlockHeader
	//  @return error
	GetBlockHeaderByHeight(height uint64) (*commonPb.BlockHeader, error)

	// GetBlock returns a block given it's block height, or returns nil if none exists.
	//  @Description:
	//  @param height
	//  @return *commonPb.Block
	//  @return error
	GetBlock(height uint64) (*commonPb.Block, error)

	// GetTx retrieves a transaction by txid, or returns nil if none exists.
	//  @Description:
	//  @param txId
	//  @return *commonPb.Transaction
	//  @return error
	GetTx(txId string) (*commonPb.Transaction, error)

	// GetTxWithBlockInfo retrieves a transaction info by txid, or returns nil if none exists.
	//  @Description:
	//  @param txId
	//  @return *storePb.TransactionStoreInfo
	//  @return error
	GetTxWithBlockInfo(txId string) (*storePb.TransactionStoreInfo, error)

	// GetTxInfoOnly 获得除Tx之外的其他TxInfo信息
	//  @Description:
	//  @param txId
	//  @return *storePb.TransactionStoreInfo
	//  @return error
	GetTxInfoOnly(txId string) (*storePb.TransactionStoreInfo, error)

	// GetTxHeight retrieves a transaction height by txid, or returns nil if none exists.
	//  @Description:
	//  @param txId
	//  @return uint64
	//  @return error
	GetTxHeight(txId string) (uint64, error)

	// TxExists returns true if the tx exist, or returns false if none exists.
	//  @Description:
	//  @param txId
	//  @return bool
	//  @return error
	TxExists(txId string) (bool, error)

	// TxArchived returns true if the tx archived, or returns false.
	//  @Description:
	//  @param txId
	//  @return bool
	//  @return error
	TxArchived(txId string) (bool, error)

	// GetTxConfirmedTime retrieves time of the tx confirmed in the blockChain
	//  @Description:
	//  @param txId
	//  @return int64
	//  @return error
	GetTxConfirmedTime(txId string) (int64, error)

	// GetLastBlock returns the last block.
	//  @Description:
	//  @return *commonPb.Block
	//  @return error
	GetLastBlock() (*commonPb.Block, error)

	// GetFilteredBlock returns a filtered block given it's block height, or return nil if none exists.
	//  @Description:
	//  @param height
	//  @return *storePb.SerializedBlock
	//  @return error
	GetFilteredBlock(height uint64) (*storePb.SerializedBlock, error)

	// GetLastSavepoint returns the last block height
	//  @Description:
	//  @return uint64
	//  @return error
	GetLastSavepoint() (uint64, error)

	// GetLastConfigBlock returns the last config block.
	//  @Description:
	//  @return *commonPb.Block
	//  @return error
	GetLastConfigBlock() (*commonPb.Block, error)

	// GetLastConfigBlockHeight returns the last config block height.
	//  @Description:
	//  @return uint64
	//  @return error
	GetLastConfigBlockHeight() (uint64, error)

	// GetBlockByTx returns a block which contains a tx.如果查询不到，则返回nil,nil
	//  @Description:
	//  @param txId
	//  @return *commonPb.Block
	//  @return error
	GetBlockByTx(txId string) (*commonPb.Block, error)

	// GetArchivedPivot get archived pivot
	//  @Description:
	//  @return uint64
	//  @return error
	GetArchivedPivot() (uint64, error)

	// ShrinkBlocks
	//  @Description: archive old blocks in an atomic operation
	//  @param startHeight
	//  @param endHeight
	//  @return map[uint64][]string
	//  @return error
	ShrinkBlocks(startHeight uint64, endHeight uint64) (map[uint64][]string, error)

	// RestoreBlocks
	//  @Description: restore blocks from outside block data in an atomic operation
	//  @param blockInfos
	//  @return error
	RestoreBlocks(blockInfos []*serialization.BlockWithSerializedInfo) error

	// GetBlockMetaIndex
	//  @Description: returns the offset of the block metadata in the file
	//  @param height
	//  @return *storePb.StoreInfo
	//  @return error
	GetBlockMetaIndex(height uint64) (*storePb.StoreInfo, error)

	// GetTxIndex
	//  @Description: returns the offset of the transaction in the file
	//  @param txId
	//  @return *storePb.StoreInfo
	//  @return error
	GetTxIndex(txId string) (*storePb.StoreInfo, error)

	// GetBlockIndex
	//  @Description: returns the offset of the transaction in the file
	//  @param height
	//  @return *storePb.StoreInfo
	//  @return error
	GetBlockIndex(height uint64) (*storePb.StoreInfo, error)

	// Close
	//  @Description: Close is used to close database
	Close()
}
