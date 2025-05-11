/*
 * Copyright (C) BABEC. All rights reserved.
 * Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
 *
 * SPDX-License-Identifier: Apache-2.0
 */

package test

import (
	"fmt"
	"log"
	"runtime/debug"
)

//GoLogger is a golang system log implementation of protocol.Logger, it's for unit test
type GoLogger struct{}

// Debug debug log print
// @param args
func (GoLogger) Debug(args ...interface{}) {
	log.Printf("DEBUG: %v", args)
}

// Debugf debug log with format print
// @param format
// @param args
func (GoLogger) Debugf(format string, args ...interface{}) {
	log.Printf("DEBUG: "+format, args...)
}

// Debugw debug log with KV print
// @param msg
// @param keysAndValues
func (GoLogger) Debugw(msg string, keysAndValues ...interface{}) {
	log.Printf("DEBUG: "+msg+" %v", keysAndValues...)
}

// Error error log print
// @param args
func (GoLogger) Error(args ...interface{}) {
	log.Printf("ERROR: %v\n%s", args, debug.Stack())
}

// Errorf  error log print
// @param format
// @param args
func (GoLogger) Errorf(format string, args ...interface{}) {
	str := fmt.Sprintf(format, args...)
	log.Printf("ERROR: "+str+"\n%s", debug.Stack())
}

// Errorw  error log print
// @param msg
// @param keysAndValues
func (GoLogger) Errorw(msg string, keysAndValues ...interface{}) {
	log.Printf("ERROR: "+msg+" %v", keysAndValues...)
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
	log.Fatalf(msg+" %v", keysAndValues...)
}

// Info info log print
// @param args
func (GoLogger) Info(args ...interface{}) {
	log.Printf("INFO: %v", args)
}

// Infof info log print
// @param format
// @param args
func (GoLogger) Infof(format string, args ...interface{}) {
	log.Printf("INFO: "+format, args...)
}

// Infow info log print
// @param msg
// @param keysAndValues
func (GoLogger) Infow(msg string, keysAndValues ...interface{}) {
	log.Printf("INFO: "+msg+" %v", keysAndValues...)
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
	log.Panicf(msg+" %v", keysAndValues...)
}

// Warn warn log print
// @param args
func (GoLogger) Warn(args ...interface{}) {
	log.Printf("WARN: %v\n%s", args, debug.Stack())
}

// Warnf warn log print
// @param format
// @param args
func (GoLogger) Warnf(format string, args ...interface{}) {
	str := fmt.Sprintf(format, args...)
	log.Printf("WARN: "+str+"\n%s", debug.Stack())
}

// Warnw warn log print
// @param msg
// @param keysAndValues
func (GoLogger) Warnw(msg string, keysAndValues ...interface{}) {
	log.Printf("WARN: "+msg+" %v", keysAndValues...)
}

// DebugDynamic debug log print
// @param l
func (GoLogger) DebugDynamic(l func() string) {
	log.Print("DEBUG:", l())
}

// InfoDynamic info log print
// @param l
func (GoLogger) InfoDynamic(l func() string) {
	log.Print("INFO:", l())
}
