/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0

*/

package blockfiledb

import (
	"encoding/binary"
	"testing"
)

func TestCRC_Binary(t *testing.T) {
	buf := []byte("string")
	checksum := NewCRC(buf).Value()

	// write checksum
	data := make([]byte, 0)
	data = append(data, []byte("0000")...)
	binary.LittleEndian.PutUint32(data[len(data)-4:], checksum)
	// write data_size
	data = appendUvarint(data, uint64(len(buf)))
	// write data
	data = append(data, buf...)

	if checksum != binary.LittleEndian.Uint32(data[:4]) {
		t.Fatalf("unequal checksum, expected:%d, actual:%d", checksum, binary.LittleEndian.Uint32(data[:4]))
	}
}

//func TestCRC_Json(t *testing.T) {
//	index := uint64(1)
//	data := []byte("string")
//
//	dst := make([]byte, 0)
//	// write index checksum and data
//	dst = append(dst, `{"index":"`...)
//	dst = strconv.AppendUint(dst, index, 10)
//	dst = append(dst, `","checksum":"`...)
//	dst = strconv.AppendUint(dst, uint64(NewCRC(data).Value()), 10)
//	dst = append(dst, `","data":`...)
//	dst = appendJSONData(dst, data)
//	dst = append(dst, '}', '\n')
//
//	// read line
//	idx := bytes.IndexByte(dst, '\n')
//	if idx == -1 {
//		t.Fatalf("ErrCorrupt")
//	}
//	line := dst[:idx]
//	dres := gjson.Get(*(*string)(unsafe.Pointer(&line)), "data")
//	if dres.Type != gjson.String {
//		t.Fatalf("ErrCorrupt")
//	}
//	// get data and cs
//	cs := gjson.Get(*(*string)(unsafe.Pointer(&line)), "checksum").String()
//	dt := gjson.Get(*(*string)(unsafe.Pointer(&line)), "data").String()
//	if cs != strconv.FormatUint(uint64(NewCRC([]byte(dt[1:])).Value()), 10) {
//		t.Fatalf("ErrCorrupt")
//	}
//}
