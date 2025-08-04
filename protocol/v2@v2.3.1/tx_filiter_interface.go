/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package protocol

import (
	"chainmaker.org/chainmaker/common/v2/birdsnest"
	"chainmaker.org/chainmaker/pb-go/v2/txfilter"
)

//TxFilter 交易过滤接口
type TxFilter interface {
	GetHeight() uint64

	SetHeight(height uint64)

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
