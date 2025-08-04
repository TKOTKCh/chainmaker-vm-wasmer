/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package protocol

import (
	"chainmaker.org/chainmaker/pb-go/v2/common"
	consensuspb "chainmaker.org/chainmaker/pb-go/v2/consensus"
)

// TBFT chain config keys
const (
	TBFT_propose_timeout_key       = "TBFT_propose_timeout"
	TBFT_propose_delta_timeout_key = "TBFT_propose_delta_timeout"
	TBFT_blocks_per_proposer       = "TBFT_blocks_per_proposer"
)

// ConsensusEngine consensus abstract engine
type ConsensusEngine interface {
	// Start the consensus engine.
	Start() error
	// Stop stops the consensus engine.
	Stop() error
}

// ConsensusState get consensus state
type ConsensusState interface {
	GetValidators() ([]string, error)
	GetLastHeight() uint64
	GetConsensusStateJSON() ([]byte, error)
}

// ConsensusExtendEngine extend engine for consensus
type ConsensusExtendEngine interface {
	ConsensusEngine
	InitExtendHandler(handler ConsensusExtendHandler)
}

//ConsensusExtendHandler extend consensus handler
type ConsensusExtendHandler interface {
	// CreateRWSet Creates a RwSet for the proposed block
	CreateRWSet(preBlkHash []byte, proposedBlock *consensuspb.ProposalBlock) error
	// VerifyConsensusArgs Verify the contents of the DPoS RwSet contained within the block
	VerifyConsensusArgs(block *common.Block, blockTxRwSet map[string]*common.TxRWSet) error
	// GetValidators Gets the validators for the current epoch
	GetValidators() ([]string, error)
}
