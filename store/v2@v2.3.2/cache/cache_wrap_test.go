/*
 * Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
 *
 * SPDX-License-Identifier: Apache-2.0
 *
 */

package cache

import (
	"fmt"
	"testing"
	"time"

	"chainmaker.org/chainmaker/protocol/v2/test"
	leveldbprovider "chainmaker.org/chainmaker/store-leveldb/v2"
	"chainmaker.org/chainmaker/store/v2/types"
	"github.com/allegro/bigcache/v3"
	"github.com/stretchr/testify/assert"
)

func TestCacheWrapToDBHandle_Get(t *testing.T) {
	db := leveldbprovider.NewMemdbHandle()
	cache, _ := bigcache.NewBigCache(bigcache.DefaultConfig(time.Second * 4))
	db2 := NewCacheWrapToDBHandle(cache, db, &test.GoLogger{})
	defer db2.Close()
	key := []byte("k1")
	value := []byte("v1")
	has, err := db2.Has(key)
	assert.False(t, has)
	t.Log(err)
	err = db2.Put(key, value)
	assert.Nil(t, err)
	has, err = db2.Has(key)
	assert.Nil(t, err)
	assert.True(t, has)
	v2, err := db2.Get(key)
	assert.Nil(t, err)
	assert.EqualValues(t, value, v2)
	err = db2.Delete(key)
	assert.Nil(t, err)
	has, _ = db2.Has(key)
	assert.False(t, has)
}
func TestCacheWrapToDBHandle_Batch(t *testing.T) {
	db := leveldbprovider.NewMemdbHandle()
	cache, _ := bigcache.NewBigCache(bigcache.DefaultConfig(time.Second * 4))
	db2 := NewCacheWrapToDBHandle(cache, db, &test.GoLogger{})
	defer db2.Close()
	batch := types.NewUpdateBatch()
	batch.Put([]byte("k1"), []byte("v1"))
	batch.Put([]byte("k2"), []byte("v1"))
	batch.Put([]byte("k3"), []byte("v1"))
	batch.Put([]byte("k4"), []byte("v1"))
	err := db2.WriteBatch(batch, true)
	assert.Nil(t, err)
	key := []byte("k5")
	has, err := db2.Has(key)
	assert.False(t, has)
	assert.Nil(t, err)
	has, err = db2.Has([]byte("k3"))
	assert.True(t, has)
	iter, err := db2.NewIteratorWithRange([]byte("k2"), []byte("k4"))
	defer iter.Release()
	assert.Nil(t, err)
	count := 0
	for iter.Next() {
		count++
		t.Logf("key:%s,value:%s", iter.Key(), iter.Value())
	}
	assert.Equal(t, 2, count)
}

func TestCacheWrapToDBHandle_Gets(t *testing.T) {
	db := leveldbprovider.NewMemdbHandle()
	cache, _ := bigcache.NewBigCache(bigcache.DefaultConfig(time.Second * 4))
	db2 := NewCacheWrapToDBHandle(cache, db, &test.GoLogger{})
	defer db2.Close()

	n := 30
	keys := make([][]byte, 0, n+1)
	values := make([][]byte, 0, n+1)
	batch := types.NewUpdateBatch()
	for i := 0; i < n; i++ {
		keyi := []byte(fmt.Sprintf("key%d", i))
		valuei := []byte(fmt.Sprintf("value%d", i))
		keys = append(keys, keyi)
		values = append(values, valuei)

		batch.Put(keyi, valuei)
	}

	err := db2.WriteBatch(batch, true)
	assert.Nil(t, err)

	keys = append(keys, nil)
	values = append(values, nil)
	valuesR, err1 := db2.GetKeys(keys)
	assert.Nil(t, err1)
	for i := 0; i < len(keys); i++ {
		assert.Equal(t, values[i], valuesR[i])
	}
}
