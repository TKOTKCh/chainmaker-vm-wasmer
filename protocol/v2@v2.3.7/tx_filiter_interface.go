/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

// Package protocol is a protocol package, which is base.
package protocol

import (
	"chainmaker.org/chainmaker/common/v2/birdsnest"
	"chainmaker.org/chainmaker/pb-go/v2/txfilter"
)

// TxFilter 交易过滤接口
type TxFilter interface {

	// GetHeight return the filtered non-existent block height for this transaction
	GetHeight() uint64

	// SetHeight set the confirmed non-existent block height
	SetHeight(height uint64)

	// Add add txId to filter
	Add(txId string) error

	// Adds add transactions to the filter in batches,
	//and log and return an array of abnormal transactions if an exception occurs
	Adds(txIds []string) error

	// IsExists ruleType see chainmaker.org/chainmaker/protocol/v2/birdsnest.RulesType
	IsExists(txId string, ruleType ...birdsnest.RuleType) (bool, *txfilter.Stat, error)

	// ValidateRule validate rules
	ValidateRule(txId string, ruleType ...birdsnest.RuleType) error

	IsExistsAndReturnHeight(txId string, ruleType ...birdsnest.RuleType) (bool, uint64, *txfilter.Stat, error)

	AddsAndSetHeight(txId []string, height uint64) (result error)

	Close()
}
