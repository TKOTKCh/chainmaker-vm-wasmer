/*
 * Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
 *
 * SPDX-License-Identifier: Apache-2.0
 *
 */

package types

import "strings"

//EngineType database type
type EngineType int32

//nolint
const (
	UnknownDb EngineType = 0
	LevelDb   EngineType = 1
	TikvDb    EngineType = 2
	MySQL     EngineType = 3
	Sqlite    EngineType = 4
	BadgerDb  EngineType = 5
)

// String return db type
// @Description:
// @receiver t
// @return string
func (t EngineType) String() string {
	switch t {
	case UnknownDb:
		return "UnknownDb"
	case LevelDb:
		return "LevelDb"
	case TikvDb:
		return "TikvDb"
	case BadgerDb:
		return "BadgerDb"
	case MySQL:
		return "MySQL"
	case Sqlite:
		return "Sqlite"
	default:
		return ""
	}
}

// LowerString lower sql type
// @Description:
// @receiver t
// @return string
func (t EngineType) LowerString() string {
	return strings.ToLower(t.String())
}

//var CommonDBDir = "common" // used to define database dir for other module (for instance consensus) to use kv database
