/*
 * Copyright (C) BABEC. All rights reserved.
 * Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
 *
 * SPDX-License-Identifier: Apache-2.0
 */

package test

import (
	"fmt"
	"runtime/debug"
	"testing"
)

//TestLogger is a golang system log implementation of protocol.Logger, it's for unit test
type TestLogger struct {
	t testing.TB
}

// NewTestLogger TestLogger constructor
// @param t
// @return *TestLogger
func NewTestLogger(t testing.TB) *TestLogger {
	return &TestLogger{t: t}
}

// Debug debug log print
// @param args
func (l TestLogger) Debug(args ...interface{}) {
	l.t.Logf("DEBUG: %v", args)
}

// Debugf debug log print
// @param format
// @param args
func (l TestLogger) Debugf(format string, args ...interface{}) {
	l.t.Logf("DEBUG: "+format, args...)
}

// Debugw debug log print
// @param msg
// @param keysAndValues
func (l TestLogger) Debugw(msg string, keysAndValues ...interface{}) {
	l.t.Logf("DEBUG: "+msg+" %v", keysAndValues...)
}

// Error error log print
// @param args
func (l TestLogger) Error(args ...interface{}) {
	l.t.Logf("ERROR: %v\n%s", args, debug.Stack())
}

// Errorf error log print
// @param format
// @param args
func (l TestLogger) Errorf(format string, args ...interface{}) {
	str := fmt.Sprintf(format, args...)
	l.t.Logf("ERROR: "+str+"\n%s", debug.Stack())
}

// Errorw error log print
// @param msg
// @param keysAndValues
func (l TestLogger) Errorw(msg string, keysAndValues ...interface{}) {
	l.t.Logf("ERROR: "+msg+" %v", keysAndValues...)
}

// Fatal t.Fatal
// @param args
func (l TestLogger) Fatal(args ...interface{}) {
	l.t.Fatal(args...)
}

// Fatalf t.Fatalf
// @param format
// @param args
func (l TestLogger) Fatalf(format string, args ...interface{}) {
	l.t.Fatalf(format, args...)
}

// Fatalw t.Fatalf
// @param msg
// @param keysAndValues
func (l TestLogger) Fatalw(msg string, keysAndValues ...interface{}) {
	l.t.Fatalf(msg+" %v", keysAndValues...)
}

// Info info log print
// @param args
func (l TestLogger) Info(args ...interface{}) {
	l.t.Logf("INFO: %v", args)
}

// Infof info log print
// @param format
// @param args
func (l TestLogger) Infof(format string, args ...interface{}) {
	l.t.Logf("INFO: "+format, args...)
}

// Infow info log print
// @param msg
// @param keysAndValues
func (l TestLogger) Infow(msg string, keysAndValues ...interface{}) {
	l.t.Logf("INFO: "+msg+" %v", keysAndValues...)
}

// Panic log print and panic
// @param args
func (l TestLogger) Panic(args ...interface{}) {
	l.t.Log(args...)
	panic(args)
}

// Panicf  log print and panic
// @param format
// @param args
func (l TestLogger) Panicf(format string, args ...interface{}) {
	l.t.Logf(format, args...)
	panic(fmt.Sprintf(format, args...))
}

// Panicw  log print and panic
// @param m
// @param keysAndValues
func (l TestLogger) Panicw(m string, keysAndValues ...interface{}) {
	msg := fmt.Sprintf(m+" %v", keysAndValues...)
	l.t.Log(msg)
	panic(msg)
}

// Warn warn log print
// @param args
func (l TestLogger) Warn(args ...interface{}) {
	l.t.Logf("WARN: %v\n%s", args, debug.Stack())
}

// Warnf warn log print
// @param format
// @param args
func (l TestLogger) Warnf(format string, args ...interface{}) {
	str := fmt.Sprintf(format, args...)
	l.t.Logf("WARN: "+str+"\n%s", debug.Stack())
}

// Warnw warn log print
// @param msg
// @param keysAndValues
func (l TestLogger) Warnw(msg string, keysAndValues ...interface{}) {
	l.t.Logf("WARN: "+msg+" %v", keysAndValues...)
}

// DebugDynamic debug log print
// @param lf
func (l TestLogger) DebugDynamic(lf func() string) {
	l.t.Log("DEBUG:", lf())
}

// InfoDynamic info log print
// @param lf
func (l TestLogger) InfoDynamic(lf func() string) {
	l.t.Log("INFO:", lf())
}
