/*
 * Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
 *
 * SPDX-License-Identifier: Apache-2.0
 *
 */

package store

import (
	"fmt"
	"runtime/debug"
	"sync"
	"sync/atomic"
	"time"

	commonPb "chainmaker.org/chainmaker/pb-go/v2/common"
	"chainmaker.org/chainmaker/protocol/v2"
)

// AsyncBlockStoreImpl Asynchronous storage of block data.
//  @Description:
type AsyncBlockStoreImpl struct {
	protocol.BlockchainStore
	logger protocol.Logger
	blocks sync.Map

	receive     chan struct{}
	worksChan   chan *commonPb.BlockInfo
	worksResult []chan error
	worksNum    int

	pendingHeight uint64
	doneHeight    uint64
	latestHeight  uint64

	start             int32
	closeC            chan struct{}
	beginReceiveBlock bool
}

// NewAsyncBlockStoreImpl add next time
//  @Description:
//  @param blockStoreImpl
//  @param logger
//  @return protocol.BlockchainStore
func NewAsyncBlockStoreImpl(blockStoreImpl protocol.BlockchainStore, logger protocol.Logger) protocol.BlockchainStore {
	async := &AsyncBlockStoreImpl{
		logger:          logger,
		receive:         make(chan struct{}, 8000),
		BlockchainStore: blockStoreImpl,
		blocks:          sync.Map{},
		closeC:          make(chan struct{}),
		worksNum:        1, // for TAX
	}

	async.logger.Infof("asynchronously store blocks")
	async.begin()
	return async
}

//  begin add next time
//  @Description:
//  @receiver async
func (async *AsyncBlockStoreImpl) begin() {
	async.worksChan = make(chan *commonPb.BlockInfo, async.worksNum)
	worksResult := make([]chan error, async.worksNum)
	for i := 0; i < async.worksNum; i++ {
		go async.storeBlock(i)
		worksResult[i] = make(chan error)
	}
	async.worksResult = worksResult
	async.start = 1
	go async.loop()
}

//  storeBlock Worker, the task of handling block storage
//  @Description:
//  @receiver async
//  @param index
func (async *AsyncBlockStoreImpl) storeBlock(index int) {
	for {
		select {
		case <-async.closeC:
			async.worksResult[index] <- nil
			return
		case blockInfo := <-async.worksChan:
			if blockInfo == nil {
				async.logger.Errorf("the block data is null when store block content in db")
				panic("the block data is null when store block content in db")
			}
			// todo. will process error
			if err := async.BlockchainStore.PutBlock(blockInfo.Block, blockInfo.RwsetList); err != nil {
				panic(err)
			}
			atomic.StoreUint64(&async.doneHeight, blockInfo.Block.Header.BlockHeight)
			async.deleteCacheBlock(blockInfo)
		}
	}
}

//  deleteCacheBlock add next time
//  @Description:
//  @receiver async
//  @param blkInfo
func (async *AsyncBlockStoreImpl) deleteCacheBlock(blkInfo *commonPb.BlockInfo) {
	async.blocks.Delete(blkInfo.Block.Header.BlockHeight)
}

//  loop  Schedules GO routines to assign tasks to idle workers
//  @Description:
//  @receiver async
func (async *AsyncBlockStoreImpl) loop() {
	timeOut := time.NewTicker(10 * time.Microsecond)
	defer timeOut.Stop()
	beginReceived := false
	for {
		select {
		case <-async.receive:
			beginReceived = true
		case <-timeOut.C:
		case <-async.closeC:
			return
		}

		latestHeight := atomic.LoadUint64(&async.latestHeight)
		if !beginReceived || async.pendingHeight > latestHeight {
			continue
		}

		select {
		// don't block when put work to channel if channel's capacity is full
		case async.worksChan <- async.getBlock():
			atomic.AddUint64(&async.pendingHeight, 1)
		default:
		}
	}
}

//  getBlock add next time
//  @Description:
//  @receiver async
//  @return *commonPb.BlockInfo
func (async *AsyncBlockStoreImpl) getBlock() *commonPb.BlockInfo {
	blk, ok := async.blocks.Load(async.pendingHeight)
	if !ok {
		return nil
	}
	return blk.(*commonPb.BlockInfo)
}

// PutBlock Asynchronous storage of block data.
//  @Description:
//  @receiver async
//  @param block
//  @param txRWSets
//  @return error
// The block data will be cached and stored by idle working GO routines later.
// Note: Concurrent calls are not allowed
func (async *AsyncBlockStoreImpl) PutBlock(block *commonPb.Block, txRWSets []*commonPb.TxRWSet) error {
	if !async.beginReceiveBlock {
		async.beginReceiveBlock = true
		async.pendingHeight = block.Header.BlockHeight
	}

	if atomic.LoadInt32(&async.start) != 1 {
		async.logger.Errorf("the store service has been stopped.")
		return fmt.Errorf("the store service has been stopped")
	}

	blockInfo := &commonPb.BlockInfo{
		Block:     block,
		RwsetList: txRWSets,
	}
	async.blocks.Store(block.Header.BlockHeight, blockInfo)
	atomic.StoreUint64(&async.latestHeight, block.Header.BlockHeight)

	if block.Header.BlockHeight%10 == 0 {
		go debug.FreeOSMemory()
	}

	select {
	// don't block when put work to channel if channel's capacity is full
	case async.receive <- struct{}{}:
	default:
	}
	return nil
}

// GetBlock returns a block given its block height, or returns nil if none exists.
//  @Description:
//  @receiver async
//  @param height
//  @return *commonPb.Block
//  @return error
func (async *AsyncBlockStoreImpl) GetBlock(height uint64) (*commonPb.Block, error) {

	blockInfo, ok := async.blocks.Load(height)
	if ok {
		if blk, isOk := blockInfo.(*commonPb.BlockInfo); isOk {
			return blk.Block, nil
		}
	}

	return async.BlockchainStore.GetBlock(height)
}
