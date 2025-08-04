/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

// Package protocol is a protocol package, which is base.
package protocol

import (
	"chainmaker.org/chainmaker/pb-go/v2/config"
	consensuspb "chainmaker.org/chainmaker/pb-go/v2/consensus"
)

// Government 治理接口
type Government interface {
	// Verify used to verify consensus data
	Verify(consensusType consensuspb.ConsensusType, chainConfig *config.ChainConfig) error
	// GetGovernanceContract get GovernanceContract
	GetGovernanceContract() (*consensuspb.GovernanceContract, error)
}
