/*
 * Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
 *
 * SPDX-License-Identifier: Apache-2.0
 *
 */

package conf

import (
	"testing"

	"github.com/mitchellh/mapstructure"
	"github.com/stretchr/testify/assert"
)

func TestNewHistoryDbConfig(t *testing.T) {
	storageConfig := make(map[string]interface{})
	storageConfig["store_path"] = "dbPath1"
	storageConfig["blockdb_config"] = map[string]interface{}{
		"provider": "badgerdb",
		"badgerdb_config": map[string]interface{}{
			"store_path": "badgerPath1",
		},
	}
	storageConfig["statedb_config"] = map[string]interface{}{
		"provider": "badgerdb",
		"badgerdb_config": map[string]interface{}{
			"store_path": "badgerPath2",
		},
	}
	storageConfig["historydb_config"] = map[string]interface{}{
		"provider":                 "badgerdb",
		"disable_contract_history": true,
		"badgerdb_config": map[string]interface{}{
			"store_path": "badgerPath3",
		},
	}
	config := StorageConfig{}
	err := mapstructure.Decode(storageConfig, &config)
	assert.Nil(t, err)
	t.Logf("blockdb:%#v", config.BlockDbConfig)
	t.Logf("statedb:%#v", config.StateDbConfig)
	t.Logf("historydb:%#v", config.HistoryDbConfig)
}
func TestNewStorageConfig(t *testing.T) {
	storageConfig := make(map[string]interface{})
	storageConfig["store_path"] = "./mypath1"
	storageConfig["blockdb_config"] = map[string]interface{}{
		"provider": "badgerdb",
		"badgerdb_config": map[string]interface{}{
			"store_path": "badgerPath1",
		},
	}
	config, err := NewStorageConfig(storageConfig)
	assert.Nil(t, err)
	t.Log(config.WriteBatchSize)
}
