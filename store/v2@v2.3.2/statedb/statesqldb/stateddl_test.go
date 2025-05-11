/*
 * Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
 *
 * SPDX-License-Identifier: Apache-2.0
 *
 */

package statesqldb

import (
	"testing"

	"chainmaker.org/chainmaker/protocol/v2"
	"github.com/stretchr/testify/assert"
)

func TestStateRecordSql_Sql(t *testing.T) {
	r := NewStateRecordSql("contract1", "create table t1(id int)", protocol.SqlTypeDdl, "v1", 0)
	db := initProvider()
	db.CreateTableIfNotExist(r)
	i, err := db.Save(r)
	assert.Nil(t, err)
	assert.EqualValues(t, 1, i)
	r.Status = 1
	i, err = db.Save(r)
	assert.Nil(t, err)
	assert.EqualValues(t, 1, i)
}
