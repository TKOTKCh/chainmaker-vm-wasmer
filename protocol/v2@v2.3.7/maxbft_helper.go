/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

// Package protocol is a protocol package, which is base.
package protocol

// MaxbftHelper MaxBFT共识的接口
type MaxbftHelper interface {
	// DiscardBlocks Delete blocks data greater than the baseHeight
	DiscardBlocks(baseHeight uint64)
}
