/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

// Package protocol is a protocol package, which is base.
package protocol

// Spv 简单支付验证节点的接口
type Spv interface {
	Start() error
	Stop()
}
