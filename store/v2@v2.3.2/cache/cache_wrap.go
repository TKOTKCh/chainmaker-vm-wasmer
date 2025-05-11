/*
 * Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
 *
 * SPDX-License-Identifier: Apache-2.0
 *
 */

package cache

import (
	"chainmaker.org/chainmaker/protocol/v2"
	"github.com/allegro/bigcache/v3"
	"strings"
)

// Cache interface
//
//	@Description:
type Cache interface {

	// Get get key from cache or db
	//  @Description:
	//  @param key
	//  @return []byte
	//  @return error
	Get(key string) ([]byte, error)

	// Set set k/v
	//  @Description:
	//  @param key
	//  @param entry
	//  @return error
	Set(key string, entry []byte) error

	// Delete del k
	//  @Description:
	//  @param key
	//  @return error
	Delete(key string) error

	// Reset set cache to initial state
	//  @Description:
	//  @return error
	Reset() error

	// Close  close cache
	//  @Description:
	//  @return error
	Close() error
}

// CacheWrapToDBHandle struct
//
//	@Description:
type CacheWrapToDBHandle struct {
	c       Cache
	innerDb protocol.DBHandle
	logger  protocol.Logger
}

// GetDbType  get db type from db
//
//	@Description:
//	@receiver c
//	@return string
func (c *CacheWrapToDBHandle) GetDbType() string {
	return c.innerDb.GetDbType()
}

// Get get value by key from cache and db
//
//	@Description:
//	@receiver c
//	@param key
//	@return []byte
//	@return error
func (c *CacheWrapToDBHandle) Get(key []byte) ([]byte, error) {
	skey := string(key)
	//1.get from cache
	v, err := c.c.Get(skey)
	if err == nil && len(v) > 0 {
		return v, nil
	}
	//2.not found in cache, get from inner db
	v, err = c.innerDb.Get(key)
	if err != nil {
		return nil, err
	}
	if strings.Contains(skey, "ContractByteCode") {
		v = []byte(" ")
	}
	//3. set value to cache
	err = c.c.Set(skey, v)
	if err != nil {
		c.logger.Warnf("set cache key[%s] get error:%s", skey, err.Error())
	}
	return v, nil
}

// GetKeys batch get key's value;
// if not found , batch get from db then refill them in cache
// @Description:
// @receiver c
// @param keys
// @return [][]byte
// @return error
func (c *CacheWrapToDBHandle) GetKeys(keys [][]byte) ([][]byte, error) {
	if len(keys) == 0 {
		return nil, nil
	}

	values := make([][]byte, len(keys))
	nfIndexs := make([]int, 0, len(keys))
	for index, key := range keys {
		if len(key) == 0 {
			values[index] = nil
			continue
		}

		value, err := c.c.Get(string(key))
		values[index] = value
		if err != nil {
			c.logger.Warnf("getting cache wrap key [%s], err:%s", key, err.Error())
		}
		if len(value) == 0 {
			nfIndexs = append(nfIndexs, index)
		}
	}

	//2.not found in cache, get from inner db
	nfKeys := make([][]byte, len(nfIndexs))
	for i := 0; i < len(nfIndexs); i++ {
		nfKeys[i] = keys[nfIndexs[i]]
	}
	nfValues, err := c.innerDb.GetKeys(nfKeys)
	if err != nil {
		return nil, err
	}
	//3. set value to cache
	for i := 0; i < len(nfKeys); i++ {
		err = c.c.Set(string(nfKeys[i]), nfValues[i])
		if err != nil {
			c.logger.Warnf("set cache key[%s] get error:%s", nfKeys[i], err.Error())
		}
	}

	for i := 0; i < len(nfIndexs); i++ {
		values[nfIndexs[i]] = nfValues[i]
	}

	return values, nil
}

// Put set key/value include cache and db
// @Description:
// @receiver c
// @param key
// @param value
// @return error
func (c *CacheWrapToDBHandle) Put(key []byte, value []byte) error {
	skey := string(key)
	c.put(skey, value)
	return c.innerDb.Put(key, value)
}

// put set key/value only include cache
// @Description:
// @receiver c
// @param skey
// @param value
func (c *CacheWrapToDBHandle) put(skey string, value []byte) {
	err := c.c.Set(skey, value)
	if err != nil {
		c.logger.Warnf("set cache key[%s] get error:%s", skey, err.Error())
	}
}

// Has check key whether exist in cache and db
//
//	@Description:
//	@receiver c
//	@param key
//	@return bool
//	@return error
func (c *CacheWrapToDBHandle) Has(key []byte) (bool, error) {
	skey := string(key)
	//1.get from cache
	v, err := c.c.Get(skey)
	if err == nil && len(v) > 0 { //has value in cache
		return true, nil
	}
	//query from cache has
	v, err = c.c.Get(cacheHasKey(skey))
	if err == nil && len(v) > 0 { //has value in cache
		return v[0] != 0, nil
	}
	//query from inner db
	has, err := c.innerDb.Has(key)
	if err != nil {
		return has, err
	}
	//set result into cache
	var hasValue byte
	if has {
		hasValue = 1
	}
	err = c.c.Set(cacheHasKey(skey), []byte{hasValue})
	if err != nil {
		c.logger.Warnf("set cache key[%s] get error:%s", skey, err.Error())
	}
	return has, nil
}

// cacheHasKey generate cacheHasKey by key
// @Description:
// @param key
// @return string
func cacheHasKey(key string) string {
	return "#Has#" + key
}

// Delete delete key/value from cache and db
//
//	@Description:
//	@receiver c
//	@param key
//	@return error
func (c *CacheWrapToDBHandle) Delete(key []byte) error {
	skey := string(key)
	c.delete(skey)
	return c.innerDb.Delete(key)
}

// delete  delete key/value from cache
// @Description:
// @receiver c
// @param skey
func (c *CacheWrapToDBHandle) delete(skey string) {
	err := c.c.Delete(skey)
	if err != nil && err != bigcache.ErrEntryNotFound {
		c.logger.Warnf("delete cache get error:%s", err)
	}
	err = c.c.Delete(cacheHasKey(skey))
	if err != nil && err != bigcache.ErrEntryNotFound {
		c.logger.Warnf("delete cache get error:%s", err)
	}
}

// WriteBatch write a batch to cache and db
//
//	@Description:
//	@receiver c
//	@param batch
//	@param sync
//	@return error
func (c *CacheWrapToDBHandle) WriteBatch(batch protocol.StoreBatcher, sync bool) error {
	//process cache
	for k, v := range batch.KVs() {
		if len(v) == 0 {
			c.delete(k)
		} else {
			c.put(k, v)
		}
	}
	return c.innerDb.WriteBatch(batch, sync)
}

// CompactRange compacts the underlying DB for the given key range
//
//	@Description:
//	@receiver c
//	@param start
//	@param limit
//	@return error
func (c *CacheWrapToDBHandle) CompactRange(start, limit []byte) error {
	return c.innerDb.CompactRange(start, limit)
}

// NewIteratorWithRange return iterator from db that contains all the key-values between given key ranges
// start is included in the results and limit is excluded.
//
//	@Description:
//	@receiver c
//	@param start
//	@param limit
//	@return protocol.Iterator
//	@return error
func (c *CacheWrapToDBHandle) NewIteratorWithRange(start []byte, limit []byte) (protocol.Iterator, error) {
	//Cache一般不支持范围查询和模糊查询，直接从数据库中查
	return c.innerDb.NewIteratorWithRange(start, limit)
}

// NewIteratorWithPrefix returns an iterator that contains all the key-values with given prefix
//
//	@Description:
//	@receiver c
//	@param prefix
//	@return protocol.Iterator
//	@return error
func (c *CacheWrapToDBHandle) NewIteratorWithPrefix(prefix []byte) (protocol.Iterator, error) {
	//Cache一般不支持范围查询和模糊查询，直接从数据库中查
	return c.innerDb.NewIteratorWithPrefix(prefix)
}

// GetWriteBatchSize not implement , will panic
//
//	@Description:
//	@receiver c
//	@return uint64
func (c *CacheWrapToDBHandle) GetWriteBatchSize() uint64 {
	//TODO implement me
	panic("implement me GetWriteBatchSize")
}

// Close close cache and db
//
//	@Description:
//	@receiver c
//	@return error
func (c *CacheWrapToDBHandle) Close() error {
	err := c.c.Close()
	if err != nil {
		c.logger.Warn(err)
	}
	return c.innerDb.Close()
}

// NewCacheWrapToDBHandle return a cacheWrapToDB which cache , db and log
//
//	@Description:
//	@param c
//	@param innerDb
//	@param l
//	@return protocol.DBHandle
func NewCacheWrapToDBHandle(c Cache, innerDb protocol.DBHandle, l protocol.Logger) protocol.DBHandle {
	l.Debug("wrap cache to a db handle")
	return &CacheWrapToDBHandle{
		c:       c,
		innerDb: innerDb,
		logger:  l,
	}
}
