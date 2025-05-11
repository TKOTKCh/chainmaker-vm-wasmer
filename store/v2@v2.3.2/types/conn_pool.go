/*
 * Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
 *
 * SPDX-License-Identifier: Apache-2.0
 *
 */

package types

import (
	"chainmaker.org/chainmaker/protocol/v2"
	"github.com/golang/groupcache/lru"
)

// ConnectionPoolDBHandle encapsulate lru cache,and logger
// @Description:
type ConnectionPoolDBHandle struct {
	c   *lru.Cache
	log protocol.Logger
}

// NewConnectionPoolDBHandle construct ConnectionPoolDBHandle
// @Description:
// @param size
// @param log
// @return *ConnectionPoolDBHandle
func NewConnectionPoolDBHandle(size int, log protocol.Logger) *ConnectionPoolDBHandle {
	cache := lru.New(size)
	cache.OnEvicted = func(key lru.Key, value interface{}) {
		handle, _ := value.(protocol.SqlDBHandle)
		log.Infof("close state sql db for contract:%s", key)
		handle.Close()
	}
	return &ConnectionPoolDBHandle{c: cache, log: log}
}

// GetDBHandle retrieve db handler from cache
// @Description:
// @receiver c
// @param contractName
// @return protocol.SqlDBHandle
// @return bool
func (c *ConnectionPoolDBHandle) GetDBHandle(contractName string) (protocol.SqlDBHandle, bool) {
	handle, ok := c.c.Get(contractName)
	if ok {
		return handle.(protocol.SqlDBHandle), true
	}
	return nil, false
}

// SetDBHandle add db handler to cache
// @Description:
// @receiver c
// @param contractName
// @param handle
func (c *ConnectionPoolDBHandle) SetDBHandle(contractName string, handle protocol.SqlDBHandle) {
	c.c.Add(contractName, handle)
}

// Clear clear the cache
// @Description:
// @receiver c
func (c *ConnectionPoolDBHandle) Clear() {
	c.c.Clear()
}
