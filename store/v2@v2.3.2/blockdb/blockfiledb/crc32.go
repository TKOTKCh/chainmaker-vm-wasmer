/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0

*/

package blockfiledb

import (
	"hash/crc32"
)

var table = crc32.MakeTable(crc32.Castagnoli)

// CRC is a CRC-32 checksum computed using Castagnoli's polynomial.
type CRC uint32

// NewCRC creates a new crc based on the given bytes.
//  @Description:
//  @param b
//  @return CRC
func NewCRC(b []byte) CRC {
	return CRC(0).Update(b)
}

// Update updates the crc with the given bytes.
//  @Description:
//  @receiver c
//  @param b
//  @return CRC
func (c CRC) Update(b []byte) CRC {
	return CRC(crc32.Update(uint32(c), table, b))
}

// Value returns a masked crc.
//  @Description:
//  @receiver c
//  @return uint32
func (c CRC) Value() uint32 {
	return uint32(c>>15|c<<17) + 0xa282ead8
}
