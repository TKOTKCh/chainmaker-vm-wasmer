/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package protocol

import (
	"chainmaker.org/chainmaker/pb-go/v2/common"
	consensuspb "chainmaker.org/chainmaker/pb-go/v2/consensus"
)

//DPoS dpos共识相关接口
type DPoS interface {
	// CreateDPoSRWSet Creates a RwSet for DPoS for the proposed block
	CreateDPoSRWSet(preBlkHash []byte, proposedBlock *consensuspb.ProposalBlock) error
	// VerifyConsensusArgs Verify the contents of the DPoS RwSet contained within the block
	VerifyConsensusArgs(block *common.Block, blockTxRwSet map[string]*common.TxRWSet) error
	// GetValidators Gets the validators for the current epoch
	GetValidators() ([]string, error)
}
