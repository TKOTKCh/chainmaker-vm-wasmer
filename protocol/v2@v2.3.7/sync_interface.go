/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

// Package protocol is a protocol package, which is base.
package protocol

import (
	syncPb "chainmaker.org/chainmaker/pb-go/v2/sync"
)

// SYNC_txHash_set_key additional data key
const (
	SYNC_txHash_set_key = "SYNC_txHash_set_key"
)

// SyncService is the server to sync the blockchain
type SyncService interface {
	// Start Init the sync server, and the sync server broadcast the current block height every broadcastTime
	Start() error

	// Stop the sync server
	Stop()

	// ListenSyncToIdealHeight listen local block height has synced to ideal height
	ListenSyncToIdealHeight() <-chan struct{}

	// StopBlockSync syncing blocks from other nodes, but still process other nodes synchronizing blocks from itself
	StopBlockSync()

	// StartBlockSync start request service
	StartBlockSync()

	// GetState get sync state, with_others indicates whether to obtain the height state of other nodes known.
	// Return an error if failed.
	GetState(with_others bool) (*syncPb.SyncState, error)
}
