/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package vm

import (
	"strings"

	"chainmaker.org/chainmaker/protocol/v2"
)

// Provider provides a function to generate vm instance manager
type Provider func(chainId string, config map[string]interface{}) (protocol.VmInstancesManager, error)

var vmProviders = make(map[string]Provider)

// RegisterVmProvider register vm provider
func RegisterVmProvider(t string, f Provider) {
	vmProviders[strings.ToUpper(t)] = f
}

// GetVmProvider get vm provider
func GetVmProvider(t string) Provider {
	provider, ok := vmProviders[strings.ToUpper(t)]
	if !ok {
		return nil
	}
	return provider
}
