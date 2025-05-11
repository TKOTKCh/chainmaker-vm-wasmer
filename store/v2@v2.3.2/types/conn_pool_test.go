/*
 * Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
 *
 * SPDX-License-Identifier: Apache-2.0
 *
 */

package types

import (
	"fmt"
	"testing"

	"chainmaker.org/chainmaker/protocol/v2/mock"
	"chainmaker.org/chainmaker/protocol/v2/test"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func TestConnectionPoolDBHandle_Clear(t *testing.T) {
	ctl := gomock.NewController(t)
	defer ctl.Finish()
	pool := NewConnectionPoolDBHandle(10, &test.GoLogger{})
	closeCount := 0
	for i := 0; i < 12; i++ {
		dbHandle := mock.NewMockSqlDBHandle(ctl)
		dbHandle.EXPECT().Close().Do(func() { closeCount++ }).AnyTimes()
		pool.SetDBHandle(fmt.Sprintf("contract%d", i), dbHandle)
	}
	assert.Equal(t, 2, closeCount)
	_, ok := pool.GetDBHandle("contract3")
	assert.True(t, ok)
	pool.Clear()
	assert.Equal(t, 12, closeCount)
	_, ok = pool.GetDBHandle("contract3")
	assert.False(t, ok)
}
