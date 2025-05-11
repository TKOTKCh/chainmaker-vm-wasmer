/*
 * Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
 *
 * SPDX-License-Identifier: Apache-2.0
 *
 */

package statesqldb

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestStateInfo_Sql(t *testing.T) {
	info := NewStateInfo("contract1", []byte("key1"), []byte("value1"), 123, time.Now())
	db := initProvider()
	db.CreateTableIfNotExist(info)
	i, err := db.Save(info)
	assert.Nil(t, err)
	assert.EqualValues(t, 1, i)
	info.ObjectValue = []byte("value2")
	i, err = db.Save(info)
	assert.Nil(t, err)
	assert.EqualValues(t, 1, i)
}

func TestStateInfo_Iterator(t *testing.T) {
	info := NewStateInfo("contract1", []byte("key1"), []byte("value1"), 123, time.Now())
	db := initProvider()
	sdb, _ := NewStateSqlDB("org1_", "chain1", db, nil, log, 1)
	db.CreateTableIfNotExist(info)
	i, err := db.Save(info)
	assert.Nil(t, err)
	assert.EqualValues(t, 1, i)
	db.Save(NewStateInfo("contract1", []byte("key2"), []byte("value2"), 123, time.Now()))
	db.Save(NewStateInfo("contract1", []byte("key3"), []byte("value3"), 123, time.Now()))
	db.Save(NewStateInfo("contract1", []byte("key4"), []byte("value4"), 123, time.Now()))
	it, err := sdb.SelectObject("contract1", []byte("key2"), []byte("key4"))
	assert.Nil(t, err)
	defer it.Release()
	count := 0
	for it.Next() {
		kv, _ := it.Value()
		t.Log(kv.String())
		count++
	}
	assert.Equal(t, 2, count)
}
