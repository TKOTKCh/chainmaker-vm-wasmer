/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package protocol

import (
	"chainmaker.org/chainmaker/pb-go/v2/common"
	"chainmaker.org/chainmaker/pb-go/v2/txpool"
)

// TxPool Manage pending transactions and update the current status of
// transactions (pending packages, pending warehousing, pending retries, etc.)
type TxPool interface {
	// Start start the txPool service
	Start() error
	// Stop stop the txPool service
	Stop() error

	// AddTx Add a transaction to the txPool.
	// There are three types of Source (RPC/P2P/INTERNAL), which different checks
	// are performed for different types of cases.
	AddTx(tx *common.Transaction, source TxSource) error
	// FetchTxs Get some transactions from single or normal txPool by block height to generate new block.
	// return transactions.
	FetchTxs(blockHeight uint64) (txs []*common.Transaction)
	// FetchTxBatches Get some transactions from batch txPool by block height to generate new block.
	// return transactions table and batchId list.
	FetchTxBatches(blockHeight uint64) (batchIds []string, txsTable [][]*common.Transaction)
	// ReGenTxBatchesWithRetryTxs Generate new batches by retryTxs
	// and return new txsTable and batchIds of new batches for batch txPool,
	// then, put new batches into the pendingCache of pool
	// and retry old batches retrieved by the batchIds into the queue of pool.
	ReGenTxBatchesWithRetryTxs(blockHeight uint64, batchIds []string, retryTxs []*common.Transaction) (
		newBatchIds []string, newTxsTable [][]*common.Transaction)
	// ReGenTxBatchesWithRemoveTxs Remove removeTxs in batches that retrieved by the batchIds
	// to create new batches for batch txPool and return new txsTable and batchIds of new batches
	// then put new batches into the pendingCache of pool
	// and delete old batches retrieved by the batchIds in pool.
	ReGenTxBatchesWithRemoveTxs(blockHeight uint64, batchIds []string, removeTxs []*common.Transaction) (
		newBatchIds []string, newTxsTable [][]*common.Transaction)
	// RemoveTxsInTxBatches Remove removeTxs in batches that retrieved by the batchIds
	// to create new batches for batch txPool.
	// then, put new batches into the queue of pool
	// and delete old batches retrieved by the batchIds in pool.
	RemoveTxsInTxBatches(batchIds []string, removeTxs []*common.Transaction)
	// GetTxsByTxIds Retrieve transactions by the txIds from single or normal txPool,
	// and only return transactions it has.
	// txsRet is the transaction in the txPool, txsMis is the transaction not in the txPool.
	GetTxsByTxIds(txIds []string) (txsRet map[string]*common.Transaction, txsMis map[string]struct{})
	// GetAllTxsByTxIds Retrieve all transactions by the txIds from single or normal txPool synchronously.
	// if there are some transactions lacked, it need to obtain them by height from the proposer.
	// if txPool get all transactions before timeout return txsRet, otherwise, return error.
	GetAllTxsByTxIds(txIds []string, proposerId string, height uint64, timeoutMs int) (
		txsRet map[string]*common.Transaction, err error)
	// GetAllTxsByBatchIds Retrieve all transactions by the batchIds from batch txPool synchronously.
	// if there are some batches lacked, it need to obtain them by height from the proposer.
	// if txPool get all batches before timeout return txsRet, otherwise, return error.
	GetAllTxsByBatchIds(batchIds []string, proposerId string, height uint64, timeoutMs int) (
		txsTable [][]*common.Transaction, err error)
	// AddTxsToPendingCache These transactions will be added to single or normal txPool to avoid the transactions
	// are fetched again and re-filled into the new block. Because of the chain confirmation
	// rule in the HotStuff consensus algorithm.
	AddTxsToPendingCache(txs []*common.Transaction, blockHeight uint64)
	// AddTxBatchesToPendingCache These transactions will be added to batch txPool to avoid the transactions
	// are fetched again and re-filled into the new block. Because of the chain confirmation
	// rule in the HotStuff consensus algorithm.
	AddTxBatchesToPendingCache(batchIds []string, blockHeight uint64)
	// RetryAndRemoveTxs Process transactions within multiple proposed blocks at the same height to
	// ensure that these transactions are not lost for single or normal txPool
	// re-add valid transactions which that are not on local node.
	// remove transactions in the commit block.
	RetryAndRemoveTxs(retryTxs []*common.Transaction, removeTxs []*common.Transaction)
	// RetryAndRemoveTxBatches Process batches within multiple proposed blocks at the same height to
	// ensure that these batches are not lost for batch txPool.
	// re-add valid batches to the queue of pool.
	// remove batches in the commit block.
	RetryAndRemoveTxBatches(retryBatchIds []string, removeBatchIds []string)
	// TxExists verifies whether the transaction exists in the txPool.
	TxExists(tx *common.Transaction) bool
	// GetPoolStatus Returns the max size of config transaction pool and common transaction pool,
	// the num of config transaction in queue and pendingCache,
	// and the the num of common transaction in queue and pendingCache.
	GetPoolStatus() (txPoolStatus *txpool.TxPoolStatus)
	// GetTxIdsByTypeAndStage Returns config or common txIds in different stage.
	// TxType may be TxType_CONFIG_TX, TxType_COMMON_TX, (TxType_CONFIG_TX|TxType_COMMON_TX)
	// TxStage may be TxStage_IN_QUEUE, TxStage_IN_PENDING, (TxStage_IN_QUEUE|TxStage_IN_PENDING)
	GetTxIdsByTypeAndStage(txType, txStage int32) (txIds []string)
	// GetTxsInPoolByTxIds Retrieve the transactions by the txIds from the txPool,
	// return transactions in the txPool and txIds not in txPool.
	// default query upper limit is 1w transaction, and error is returned if the limit is exceeded.
	GetTxsInPoolByTxIds(txIds []string) (txsRet []*common.Transaction, txsMis []string, err error)
}

// TxSource tx come from
type TxSource int

const (
	//RPC add tx by rpc
	RPC TxSource = iota
	// P2P add tx by p2p
	P2P
	//INTERNAL special internal tx
	INTERNAL
)
