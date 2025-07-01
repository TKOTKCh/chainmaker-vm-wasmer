/*
 * Copyright (C) BABEC. All rights reserved.
 * Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
 *
 * SPDX-License-Identifier: Apache-2.0
 */

// Package test is a test package.
package test

import (
	"fmt"
	"log"
	"runtime/debug"
)

const (
	// DEBUG debug string
	DEBUG = "DEBUG: "
	// ERROR error string
	ERROR = "ERROR: "
	// INFO info string
	INFO = "INFO: "
	// WARN warn string
	WARN = "WARN: "
	// ValSpace val string with space
	ValSpace = " %v"
	// ValNoSpace val string without space
	ValNoSpace = "%v"
	// NextLineString string with next line
	NextLineString = "\n%s"
)

// GoLogger is a golang system log implementation of protocol.Logger, it's for unit test
type GoLogger struct{}

// Debug debug log print
// @param args
func (GoLogger) Debug(args ...interface{}) {
	log.Printf(DEBUG+ValNoSpace, args)
}

// Debugf debug log with format print
// @param format
// @param args
func (GoLogger) Debugf(format string, args ...interface{}) {
	log.Printf(DEBUG+format, args...)
}

// Debugw debug log with KV print
// @param msg
// @param keysAndValues
func (GoLogger) Debugw(msg string, keysAndValues ...interface{}) {
	log.Printf(DEBUG+msg+ValSpace, keysAndValues...)
}

// Error error log print
// @param args
func (GoLogger) Error(args ...interface{}) {
	log.Printf(ERROR+ValNoSpace+NextLineString, args, debug.Stack())
}

// Errorf  error log print
// @param format
// @param args
func (GoLogger) Errorf(format string, args ...interface{}) {
	str := fmt.Sprintf(format, args...)
	log.Printf(ERROR+str+NextLineString, debug.Stack())
}

// Errorw  error log print
// @param msg
// @param keysAndValues
func (GoLogger) Errorw(msg string, keysAndValues ...interface{}) {
	log.Printf(ERROR+msg+ValSpace, keysAndValues...)
}

// Fatal log.Fatal
// @param args
func (GoLogger) Fatal(args ...interface{}) {
	log.Fatal(args...)
}

// Fatalf log.Fatalf
// @param format
// @param args
func (GoLogger) Fatalf(format string, args ...interface{}) {
	log.Fatalf(format, args...)
}

// Fatalw log.Fatalf
// @param msg
// @param keysAndValues
func (GoLogger) Fatalw(msg string, keysAndValues ...interface{}) {
	log.Fatalf(msg+ValSpace, keysAndValues...)
}

// Info info log print
// @param args
func (GoLogger) Info(args ...interface{}) {
	log.Printf(INFO+ValNoSpace, args)
}

// Infof info log print
// @param format
// @param args
func (GoLogger) Infof(format string, args ...interface{}) {
	log.Printf(INFO+format, args...)
}

// Infow info log print
// @param msg
// @param keysAndValues
func (GoLogger) Infow(msg string, keysAndValues ...interface{}) {
	log.Printf(INFO+msg+ValSpace, keysAndValues...)
}

// Panic log.Panic
// @param args
func (GoLogger) Panic(args ...interface{}) {
	log.Panic(args...)
}

// Panicf log.Panicf
// @param format
// @param args
func (GoLogger) Panicf(format string, args ...interface{}) {
	log.Panicf(format, args...)
}

// Panicw log.Panicf
// @param msg
// @param keysAndValues
func (GoLogger) Panicw(msg string, keysAndValues ...interface{}) {
	log.Panicf(msg+ValSpace, keysAndValues...)
}

// Warn warn log print
// @param args
func (GoLogger) Warn(args ...interface{}) {
	log.Printf(WARN+ValNoSpace+NextLineString, args, debug.Stack())
}

// Warnf warn log print
// @param format
// @param args
func (GoLogger) Warnf(format string, args ...interface{}) {
	str := fmt.Sprintf(format, args...)
	log.Printf(WARN+str+NextLineString, debug.Stack())
}

// Warnw warn log print
// @param msg
// @param keysAndValues
func (GoLogger) Warnw(msg string, keysAndValues ...interface{}) {
	log.Printf(WARN+msg+ValSpace, keysAndValues...)
}

// DebugDynamic debug log print
// @param l
func (GoLogger) DebugDynamic(l func() string) {
	log.Print(DEBUG, l())
}

// InfoDynamic info log print
// @param l
func (GoLogger) InfoDynamic(l func() string) {
	log.Print(INFO, l())
}
