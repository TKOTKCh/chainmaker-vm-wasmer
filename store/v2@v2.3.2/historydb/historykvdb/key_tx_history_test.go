/*
 * Copyright (C) BABEC. All rights reserved.
 * Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
 *
 * SPDX-License-Identifier: Apache-2.0
 */

package historykvdb

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSplitKey(t *testing.T) {
	key1 := "kContract1#key#123#txId1"
	_, _, blockHeight, _, err := splitKey([]byte(key1))
	assert.Nil(t, err)
	assert.Equal(t, uint64(123), blockHeight)

	key2 := "kContract1#key#field1#1234#txId1"
	_, _, blockHeight, _, err = splitKey([]byte(key2))
	assert.Nil(t, err)
	assert.Equal(t, uint64(1234), blockHeight)

	key3 := "kContract1#key#field1#12345#9$txId1"
	_, _, blockHeight, _, err = splitKey([]byte(key3))
	assert.Nil(t, err)
	assert.Equal(t, uint64(12345), blockHeight)

	key4 := constructKey(
		"Contract1", []byte("key"), 100, "txId1", 222,
	)
	_, _, blockHeight, _, err = splitKey(key4)
	assert.Nil(t, err)
	assert.Equal(t, uint64(100), blockHeight)
}
