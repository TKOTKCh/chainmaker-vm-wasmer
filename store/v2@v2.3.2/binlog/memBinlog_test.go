/*
 * Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
 *
 * SPDX-License-Identifier: Apache-2.0
 *
 */

package binlog

import (
	"testing"

	storePb "chainmaker.org/chainmaker/pb-go/v2/store"
	"chainmaker.org/chainmaker/protocol/v2/test"
	"github.com/stretchr/testify/assert"
)

func TestMemBinlog_ReadFileSection(t *testing.T) {
	l := NewMemBinlog(&test.GoLogger{})
	data1 := make([]byte, 100)
	data1[0] = 'a'
	fileName, offset, length, err := l.Write(1, data1)
	assert.Nil(t, err)
	t.Log(fileName, offset, length)
	data2 := make([]byte, 200)
	data2[100] = 'b'
	fileName, offset, length, err = l.Write(2, data2)
	assert.Nil(t, err)
	t.Log(fileName, offset, length)
	data3 := make([]byte, 300)
	data3[200] = 'c'
	fileName, offset, length, err = l.Write(3, data3)
	assert.Nil(t, err)
	t.Log(fileName, offset, length)
	data4, err := l.ReadFileSection(&storePb.StoreInfo{
		StoreType: 0,
		FileName:  fileName,
		Offset:    offset,
		ByteLen:   length,
	})
	assert.Nil(t, err)
	t.Log(data4)

	data5, err := l.ReadFileSection(&storePb.StoreInfo{
		StoreType: 0,
		FileName:  fileName,
		Offset:    200,
		ByteLen:   10,
	})
	assert.Nil(t, err)
	t.Log(data5)
}
