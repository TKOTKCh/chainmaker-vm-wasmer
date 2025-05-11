/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package protocol

import (
	"chainmaker.org/chainmaker/pb-go/v2/accesscontrol"
	"chainmaker.org/chainmaker/pb-go/v2/common"
	vmPb "chainmaker.org/chainmaker/pb-go/v2/vm"
)

// SnapshotManager Snapshot management container to manage chained snapshots
type SnapshotManager interface {
	// Create ContractStore at the current block height
	NewSnapshot(prevBlock *common.Block, block *common.Block) Snapshot

	//Once the block is submitted, notify the snapshot to clean up
	NotifyBlockCommitted(block *common.Block) error

	//clean the snapshot when verify fail in tbft
	ClearSnapshot(block *common.Block) error

	// Get Snapshot by block fingerPrint
	GetSnapshot(prevBlock *common.Block, block *common.Block) Snapshot
}

//Snapshot is a chain structure that saves the read and write cache information of the blocks
//that are not in the library
type Snapshot interface {

	// Get database for virtual machine access
	GetBlockchainStore() BlockchainStore

	//Read the key from the current snapshot and the previous snapshot
	GetKey(txExecSeq int, contractName string, key []byte) ([]byte, error)

	// GetKeys Read the key from the current snapshot and the previous snapshot
	GetKeys(txExecSeq int, keys []*vmPb.BatchKey) ([]*vmPb.BatchKey, error)

	// After the scheduling is completed, get the read and write set from the current snapshot
	GetTxRWSetTable() []*common.TxRWSet

	// After the scheduling is completed, get the result from the current snapshot
	GetTxResultMap() map[string]*common.Result

	// Get exec seq for snapshot
	GetSnapshotSize() int

	// After the scheduling is completed, obtain the transaction sequence table from the current snapshot
	GetTxTable() []*common.Transaction

	// GetSpecialTxTable return specialTxTable which will be exec sequencially
	GetSpecialTxTable() []*common.Transaction

	// Get previous snapshot
	GetPreSnapshot() Snapshot

	// Set previous snapshot
	SetPreSnapshot(Snapshot)

	// GetBlockHeight returns current block height
	GetBlockHeight() uint64

	// GetBlockFingerprint returns current block fingerprint
	GetBlockFingerprint() string

	// GetBlockTimestamp returns current block timestamp
	GetBlockTimestamp() int64

	// GetBlockProposer returns Block Proposer for current snapshot
	GetBlockProposer() *accesscontrol.Member

	// ApplyTxSimContext If the transaction can be added to the snapshot after the conflict dependency is established
	// Even if an exception occurs when the transaction is handed over to the virtual machine module,
	// the transaction is still packed into a block, but the read-write set of the transaction is left empty.
	// This situation includes:
	// 1 wrong txtype is used,
	// 2 parameter error occurs when parsing querypayload and transactpayload,
	// 3 virtual machine runtime throws panic,
	// 4 smart contract byte code actively throws panic
	// The second bool parameter here indicates whether the above exception has occurred
	ApplyTxSimContext(TxSimContext, ExecOrderTxType, bool, bool) (bool, int)

	// Build a dag for all transactions that have resolved the read-write conflict dependencies
	// If txRWSetTable is nil, it uses snapshot.txRWSetTable. Otherwise use txRWSetTable in argument.
	BuildDAG(isSql bool, txRWSetTable []*common.TxRWSet) *common.DAG

	// If snapshot is sealed, no more transaction will be added into snapshot
	IsSealed() bool
	Seal()

	// ApplyBlock In the fast synchronization mode, the results pulled from other
	// nodes will be written to snapshot after the block verification passes
	ApplyBlock(block *common.Block, txRWSetMap map[string]*common.TxRWSet)
}
