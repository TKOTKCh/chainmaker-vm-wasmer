/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

// Package protocol is a protocol package, which is base.
package protocol

import (
	"chainmaker.org/chainmaker/pb-go/v2/common"
)

// TxScheduler schedules a transaction batch and returns a block (maybe not complete) with DAG
// TxScheduler also can run VM with a given DAG, and return results.
// It can only be called by BlockProposer
// Should have multiple implementations and adaptive mode
type TxScheduler interface {
	// Schedule schedule a transaction batch into a block with DAG
	// Return result(and read write set) of each transaction, no matter it is executed OK, or fail, or timeout.
	// For cross-contracts invoke, result(and read write set) include all contract relative.
	Schedule(block *common.Block, txBatch []*common.Transaction, snapshot Snapshot) (
		map[string]*common.TxRWSet, map[string][]*common.ContractEvent, error)
	// SimulateWithDag Run VM with a given DAG, and return results.
	SimulateWithDag(block *common.Block, snapshot Snapshot) (
		map[string]*common.TxRWSet, map[string]*common.Result, error)
	// Halt To halt scheduler and release VM resources.
	Halt()
}
