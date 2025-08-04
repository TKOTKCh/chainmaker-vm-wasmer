/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package protocol

import (
	"chainmaker.org/chainmaker/pb-go/v2/common"
)

// ProposalCache Cache proposed blocks that are not committed yet
type ProposalCache interface {
	// Clear proposed blocks with height.
	ClearProposedBlockAt(height uint64)
	// Get all proposed blocks at a specific height
	GetProposedBlocksAt(height uint64) []*common.Block
	// Get proposed block with specific block hash in current consensus height.
	GetProposedBlock(b *common.Block) (*common.Block, map[string]*common.TxRWSet, map[string][]*common.ContractEvent)
	// Set porposed block in current consensus height, after it's generated or verified.
	SetProposedBlock(b *common.Block, rwSetMap map[string]*common.TxRWSet,
		contractEventMap map[string][]*common.ContractEvent, selfProposed bool) error
	// Get proposed block that is proposed by node itself.
	GetSelfProposedBlockAt(height uint64) *common.Block
	// Get proposed block by block hash and block height
	GetProposedBlockByHashAndHeight(hash []byte, height uint64) (*common.Block, map[string]*common.TxRWSet)
	// Return if a proposed block has cached in current consensus height.
	HasProposedBlockAt(height uint64) bool
	// Return if this node has proposed a block as proposer.
	IsProposedAt(height uint64) bool
	// To mark this node has proposed a block as proposer.
	SetProposedAt(height uint64)
	// Reset propose status of this node.
	ResetProposedAt(height uint64)
	// Remove proposed block in height except the specific block.
	KeepProposedBlock(hash []byte, height uint64) []*common.Block
	// DiscardAboveHeight Delete blocks data greater than the baseHeight
	DiscardBlocks(baseHeight uint64) []*common.Block
	// ClearTheBlock clean the special block in proposerCache
	ClearTheBlock(block *common.Block)
}

// LedgerCache Cache the latest block in ledger(DB).
type LedgerCache interface {
	// Get the latest committed block
	GetLastCommittedBlock() *common.Block
	// Set the latest committed block
	SetLastCommittedBlock(b *common.Block)
	// Return current block height
	CurrentHeight() (uint64, error)
}
