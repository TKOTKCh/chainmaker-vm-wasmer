/*
 * Copyright (C) BABEC. All rights reserved.
 * Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
 *
 * SPDX-License-Identifier: Apache-2.0
 */

package test

import (
	"testing"
)

func TestTestLogger_Debug(t *testing.T) {
	l := NewTestLogger(t)
	l.Debug("message", 1)
	l.Debugf("%s-%d", arg0.FilePath, arg0.MaxAge)
	l.Debugw("config", arg0)
}
func TestTestLogger_Info(t *testing.T) {
	l := NewTestLogger(t)
	l.Info("message", 1)
	l.Infof("%s-%d", arg0.FilePath, arg0.MaxAge)
	l.Infow("config", arg0)
}
func TestTestLogger_Warn(t *testing.T) {
	l := NewTestLogger(t)
	l.Warn("message", 1)
	l.Warnf("%s-%d", arg0.FilePath, arg0.MaxAge)
	l.Warnw("config", arg0)
}
func TestTestLogger_Error(t *testing.T) {
	l := NewTestLogger(t)
	l.Error("message", 1)
	l.Errorf("%s-%d", arg0.FilePath, arg0.MaxAge)
	l.Errorw("config", arg0)
}
