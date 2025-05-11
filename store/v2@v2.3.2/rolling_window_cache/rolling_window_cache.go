/*
 * Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
 *
 * SPDX-License-Identifier: Apache-2.0
 *
 */

package rolling_window_cache

import "chainmaker.org/chainmaker/store/v2/serialization"

// RollingWindowCache 一个滑动窗口Cache
// @Description:
type RollingWindowCache interface {

	// Has
	// @Description: 判断 key 在 start >r.startBlockHeight条件下，在 [start,r.endBlockHeight]区间内 是否存在
	// @param key
	// @param start
	// @return bool start是否在滑动窗口内
	// @return bool key是否存在
	// @return error
	Has(key string, start uint64) (bool, bool, error)

	// Consumer 消费 chan 中 数据到 RollingWindowCache
	// @Description:
	Consumer()

	// InitGenesis commit genesis block
	// @Description:
	// @param genesisBlock
	// @return error
	InitGenesis(genesisBlock *serialization.BlockWithSerializedInfo) error

	// CommitBlock  commits the txId and savepoint in an atomic operation
	// @Description:
	// @param blockWithRWSet
	// @param isCache
	// @return error
	CommitBlock(blockWithRWSet *serialization.BlockWithSerializedInfo, isCache bool) error

	// ResetRWCache set RollingWindowCache use blockInfo
	// @Description:
	// @param blockInfo
	// @return error
	ResetRWCache(blockInfo *serialization.BlockWithSerializedInfo) error
}
