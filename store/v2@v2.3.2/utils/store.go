/*
 * Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
 *
 * SPDX-License-Identifier: Apache-2.0
 *
 */

package utils

import (
	"os"
	"sync"
	"time"

	storePb "chainmaker.org/chainmaker/pb-go/v2/store"
	"chainmaker.org/chainmaker/protocol/v2"
	"github.com/gogo/protobuf/proto"
)

type conCurfunc func(blockHeight uint64, batch protocol.StoreBatcher) error

// ConcurrentWriteBatch  Concurrent write batch to db
// @Description:
// @param batches
// @param writeFunc
// @param blockHeight
// @return error
func ConcurrentWriteBatch(batches []protocol.StoreBatcher, writeFunc conCurfunc,
	blockHeight uint64) error {
	var batchWG sync.WaitGroup
	batchWG.Add(len(batches))
	errsChan := make(chan error, len(batches))
	for _, bth := range batches {
		go func(batch protocol.StoreBatcher) {
			defer batchWG.Done()
			err := writeFunc(blockHeight, batch)
			if err != nil {
				errsChan <- err
			}
		}(bth)
	}

	batchWG.Wait()
	if len(errsChan) > 0 {
		return <-errsChan
	}

	return nil
}

// CompactRange 做一次compact
// @Description:
// @param dbHandle
// @param logger
func CompactRange(dbHandle protocol.DBHandle, logger protocol.Logger) {
	//trigger level compact
	for i := 1; i <= 1; i++ {
		logger.Infof("Do %dst time CompactRange", i)
		if err := dbHandle.CompactRange(nil, nil); err != nil {
			logger.Warnf("blockdb level compact failed: %v", err)
		}
		//time.Sleep(2 * time.Second)
	}
}

// ConstructDBIndexInfo 索引序列化
// @Description:
// @param blkIndex
// @param offset
// @param byteLen
// @return []byte
func ConstructDBIndexInfo(blkIndex *storePb.StoreInfo, offset, byteLen uint64) []byte {
	//return []byte(fmt.Sprintf("%s:%d:%d", blkIndex.FileName, blkIndex.Offset+offset, byteLen))
	index := &storePb.StoreInfo{
		FileName: blkIndex.FileName,
		Offset:   blkIndex.Offset + offset,
		ByteLen:  byteLen,
	}
	indexByte, err := proto.Marshal(index)
	if err != nil {
		return nil
	}
	return indexByte
}

// DecodeValueToIndex 索引反序列化
// @Description:
// @param blockIndexByte
// @return *storePb.StoreInfo
// @return error
func DecodeValueToIndex(blockIndexByte []byte) (*storePb.StoreInfo, error) {
	if len(blockIndexByte) == 0 {
		return nil, nil
	}
	var blockIndex storePb.StoreInfo
	err := proto.Unmarshal(blockIndexByte, &blockIndex)
	if err != nil {
		return nil, err
	}
	return &blockIndex, nil
}

// PathExists  判断path是否存在
// @Description:
// @param path
// @return bool
// @return error
func PathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

// ElapsedMillisSeconds 计算时间差
// @Description:
// @param start
// @return int64
func ElapsedMillisSeconds(start time.Time) int64 {
	return (time.Now().UnixNano() - start.UnixNano()) / 1e6
}
