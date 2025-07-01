/*
 * Copyright (C) BABEC. All rights reserved.
 * Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
 *
 * SPDX-License-Identifier: Apache-2.0
 */

package test

import (
	"fmt"
	"os"
)

//HoleLogger do record nothing
//不做任何日志记录的Logger，主要用于BenchmarkTest性能测试的场景
type HoleLogger struct {
}

// Debug nothing
// @param args
func (h HoleLogger) Debug(args ...interface{}) {

}

// Debugf nothing
// @param format
// @param args
func (h HoleLogger) Debugf(format string, args ...interface{}) {

}

// Debugw nothing
// @param msg
// @param keysAndValues
func (h HoleLogger) Debugw(msg string, keysAndValues ...interface{}) {

}

// Error nothing
// @param args
func (h HoleLogger) Error(args ...interface{}) {

}

// Errorf nothing
// @param format
// @param args
func (h HoleLogger) Errorf(format string, args ...interface{}) {

}

// Errorw nothing
// @param msg
// @param keysAndValues
func (h HoleLogger) Errorw(msg string, keysAndValues ...interface{}) {

}

// Fatal os.Exit(1)
// @param args
func (h HoleLogger) Fatal(args ...interface{}) {
	os.Exit(1)
}

// Fatalf os.Exit(1)
// @param format
// @param args
func (h HoleLogger) Fatalf(format string, args ...interface{}) {
	os.Exit(1)
}

// Fatalw os.Exit(1)
// @param msg
// @param keysAndValues
func (h HoleLogger) Fatalw(msg string, keysAndValues ...interface{}) {
	os.Exit(1)
}

// Info nothing
// @param args
func (h HoleLogger) Info(args ...interface{}) {

}

// Infof nothing
// @param format
// @param args
func (h HoleLogger) Infof(format string, args ...interface{}) {

}

// Infow nothing
// @param msg
// @param keysAndValues
func (h HoleLogger) Infow(msg string, keysAndValues ...interface{}) {

}

// Panic nothing
// @param args
func (h HoleLogger) Panic(args ...interface{}) {
	panic(args)
}

// Panicf panic
// @param format
// @param args
func (h HoleLogger) Panicf(format string, args ...interface{}) {
	panic(fmt.Sprintf(format, args...))
}

// Panicw panic
// @param msg
// @param keysAndValues
func (h HoleLogger) Panicw(msg string, keysAndValues ...interface{}) {

	panic(fmt.Sprintf(msg+" %v", keysAndValues...))
}

// Warn nothing
// @param args
func (h HoleLogger) Warn(args ...interface{}) {

}

// Warnf nothing
// @param format
// @param args
func (h HoleLogger) Warnf(format string, args ...interface{}) {

}

// Warnw nothing
// @param msg
// @param keysAndValues
func (h HoleLogger) Warnw(msg string, keysAndValues ...interface{}) {

}

// DebugDynamic nothing
// @param getStr
func (h HoleLogger) DebugDynamic(getStr func() string) {
	getStr()
}

// InfoDynamic nothing
// @param getStr
func (h HoleLogger) InfoDynamic(getStr func() string) {
	getStr()
}
