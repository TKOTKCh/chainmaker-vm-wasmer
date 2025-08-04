/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package protocol

import (
	"chainmaker.org/chainmaker/pb-go/v2/common"
	"chainmaker.org/chainmaker/pb-go/v2/config"
	"chainmaker.org/chainmaker/pb-go/v2/consensus"
)

// ChainConf chainconf interface
type ChainConf interface {
	Init() error                                                              // init
	ChainConfig() *config.ChainConfig                                         // get the latest chainconfig
	GetChainConfigFromFuture(blockHeight uint64) (*config.ChainConfig, error) // get chainconfig by (blockHeight-1)
	GetChainConfigAt(blockHeight uint64) (*config.ChainConfig, error)         // get chainconfig by blockHeight
	GetConsensusNodeIdList() ([]string, error)                                // get node list
	// Deprecated: Use msgbus.PublishSync instead since version 2.3.0.
	CompleteBlock(block *common.Block) error // callback after insert block to db success
	// Deprecated: Use msgbus.Register instead since version 2.3.0.
	AddWatch(w Watcher) // add watcher
	// Deprecated: Use msgbus.Register instead since version 2.3.0.
	AddVmWatch(w VmWatcher) // add vm watcher
}

// Watcher chainconfig watcher
// Deprecated: Since version 2.3.0, it has been replaced by implementing msgBus.Subscriber interface.
type Watcher interface {
	Module() string                              // module
	Watch(chainConfig *config.ChainConfig) error // callback the chainconfig
}

// Verifier verify consensus data
type Verifier interface {
	Verify(consensusType consensus.ConsensusType, chainConfig *config.ChainConfig) error
}

// VmWatcher native vm watcher
// Deprecated: Since version 2.3.0, it has been replaced by implementing msgBus.Subscriber interface.
type VmWatcher interface {
	Module() string                                          // module
	ContractNames() []string                                 // watch the contract
	Callback(contractName string, payloadBytes []byte) error // callback
}
