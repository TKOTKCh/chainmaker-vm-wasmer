/*
 * Copyright (C) BABEC. All rights reserved.
 * Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
 *
 * SPDX-License-Identifier: Apache-2.0
 */

package binlog

import (
	"errors"
	"fmt"
	"strconv"

	storePb "chainmaker.org/chainmaker/pb-go/v2/store"
	"chainmaker.org/chainmaker/protocol/v2"
)

// MemBinlog add next time
//  @Description:
type MemBinlog struct {
	mem  map[uint64][]byte
	last uint64
	log  protocol.Logger
}

// ReadFileSection 根据索引，读取对应数据
//  @Description:
//  @receiver l
//  @param fiIndex
//  @return []byte
//  @return error
func (l *MemBinlog) ReadFileSection(fiIndex *storePb.StoreInfo) ([]byte, error) {
	i, _ := strconv.Atoi(fiIndex.FileName)
	data, found := l.mem[uint64(i)]
	if !found {
		return nil, errors.New("not found")
	}
	return data[fiIndex.Offset : fiIndex.Offset+fiIndex.ByteLen], nil
}

// NewMemBinlog 创建一个内存binlog
//  @Description:
//  @param log
//  @return *MemBinlog
func NewMemBinlog(log protocol.Logger) *MemBinlog {
	return &MemBinlog{
		mem:  make(map[uint64][]byte),
		last: 0,
		log:  log,
	}
}

// Close 关闭
//  @Description:
//  @receiver l
//  @return error
func (l *MemBinlog) Close() error {
	l.mem = make(map[uint64][]byte)
	l.last = 0
	return nil
}

// TruncateFront 指定index 清空
//  @Description:
//  @receiver l
//  @param index
//  @return error
func (l *MemBinlog) TruncateFront(index uint64) error {
	return nil
}

// ReadLastSegSection 读取最后的segment
//  @Description:
//  @receiver l
//  @param index
//  @return []byte
//  @return string
//  @return uint64
//  @return uint64
//  @return error
func (l *MemBinlog) ReadLastSegSection(index uint64) ([]byte, string, uint64, uint64, error) {
	return l.mem[index], "", 0, 0, nil
}

// LastIndex 返回最后写入的索引
//  @Description:
//  @receiver l
//  @return uint64
//  @return error
func (l *MemBinlog) LastIndex() (uint64, error) {
	l.log.Debugf("get last index %d", l.last)
	return l.last, nil
}

// Write 写一个区块到内存中
//  @Description:
//  @receiver l
//  @param index
//  @param data
//  @return fileName
//  @return offset
//  @return blkLen
//  @return err
func (l *MemBinlog) Write(index uint64, data []byte) (fileName string, offset, blkLen uint64, err error) {
	if index != l.last+1 {
		return "", 0, 0, errors.New("binlog out of order")
	}
	l.mem[index] = data
	l.last = index
	l.log.Debugf("write binlog index=%d,offset=%d,len=%d", index, 0, len(data))
	return fmt.Sprintf("%d", index), 0, uint64(len(data)), nil
}
