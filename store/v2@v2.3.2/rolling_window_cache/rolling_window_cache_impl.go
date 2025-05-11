/*
 * Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
 *
 * SPDX-License-Identifier: Apache-2.0
 *
 */

package rolling_window_cache

import (
	"errors"
	"sync"

	bytesconv "chainmaker.org/chainmaker/common/v2/bytehelper"
	"chainmaker.org/chainmaker/protocol/v2"
	"chainmaker.org/chainmaker/store/v2/cache"
	"chainmaker.org/chainmaker/store/v2/serialization"
	"chainmaker.org/chainmaker/store/v2/types"
	"github.com/gogo/protobuf/proto"
)

const (
	blockNumIdxKeyPrefix = 'n'
	txIDIdxKeyPrefix     = 't'
)

var (
	errGetBatchPool   = errors.New("get updatebatch error")
	errBlockInfoNil   = errors.New("blockInfo is nil")
	errBlockNil       = errors.New("block is nil")
	errBlockHeaderNil = errors.New("block header is nil")
	//errBlockTxsNil    = errors.New("block txs is nil")
)

/*
| 1                         10|               | 0  1  2                      10|
 —— —— —— —— —— —— —— —— —— ——                 —— —— —— —— —— —— —— —— —— —— ——
 Transaction Pool (cap 10 )                    Rolling Window Cache (cap 11  ) 1.1*tx_pool_cap

      ---->                                   ^ ---> rolling window a step     ^ --->

| 2                         11|                  | 1  2  3                      11|
 —— —— —— —— —— —— —— —— —— ——                    —— —— —— —— —— —— —— —— —— —— ——
 Transaction Pool (cap 10 )                       Rolling Window Cache (cap 11  ) 1.1*tx_pool_cap

         RollingWindowCache always covers TransactionPool
*/

// RollingWindowCacher RollingWindowCacher 中 cache 1.1倍交易池大小 的txid
// @Description:
// 保证 当前窗口，可以覆盖 交易池大小的txid
// 保证 交易在做范围查重时的命中率达到100%
type RollingWindowCacher struct {
	Cache            *cache.StoreCacheMgr
	txIdCount        uint64
	currCount        uint64
	startBlockHeight uint64
	endBlockHeight   uint64
	lastBlockHeight  uint64
	sync.RWMutex
	batchPool    sync.Pool
	logger       protocol.Logger
	blockSerChan chan *serialization.BlockWithSerializedInfo
}

// NewRollingWindowCacher 创建一个 滑动窗口 cache
// @Description:
// @param txIdCount
// @param currCount
// @param startBlockHeight
// @param endBlockHeight
// @param lastBlockHeight
// @param logger
// @return RollingWindowCache
func NewRollingWindowCacher(txIdCount, currCount, startBlockHeight, endBlockHeight, lastBlockHeight uint64,
	logger protocol.Logger) RollingWindowCache {
	r := &RollingWindowCacher{
		Cache:            cache.NewStoreCacheMgr("", 10, logger),
		txIdCount:        txIdCount,
		currCount:        currCount,
		startBlockHeight: startBlockHeight,
		endBlockHeight:   endBlockHeight,
		lastBlockHeight:  lastBlockHeight,
		blockSerChan:     make(chan *serialization.BlockWithSerializedInfo, 10),
		logger:           logger,
	}
	r.batchPool.New = func() interface{} {
		return types.NewUpdateBatch()
	}

	//异步消费，更新cache
	go r.Consumer()
	return r

}

// InitGenesis commit genesis block
// @Description:
// @receiver r
// @param genesisBlock
// @return error
func (r *RollingWindowCacher) InitGenesis(genesisBlock *serialization.BlockWithSerializedInfo) error {
	return r.ResetRWCache(genesisBlock)
}

// CommitBlock  commits the txId to chan
// @Description:
// @receiver r
// @param blockInfo
// @param isCache
// @return error
func (r *RollingWindowCacher) CommitBlock(blockInfo *serialization.BlockWithSerializedInfo, isCache bool) error {
	if isCache {
		if blockInfo == nil {
			r.logger.Errorf("blockInfo is nil,when commitblock")
			return errBlockInfoNil
		}
		if blockInfo.Block == nil {
			r.logger.Errorf("block is nil,when commitblock")
			return errBlockNil
		}
		if blockInfo.Block.Header == nil {
			r.logger.Errorf("block header is nil,when commitblock")
			return errBlockHeaderNil
		}

		r.blockSerChan <- blockInfo
		return nil
	}
	return nil
}

// ResetRWCache  use the last  block to reset RWCache ,when blockstore is restarting
// @Description:
// @receiver r
// @param blockInfo
// @return error
func (r *RollingWindowCacher) ResetRWCache(blockInfo *serialization.BlockWithSerializedInfo) error {
	r.Lock()
	defer r.Unlock()

	batch, ok := r.batchPool.Get().(*types.UpdateBatch)
	if !ok {
		r.logger.Errorf("chain[%s]: blockInfo[%d] get updatebatch error in rollingWindowCache",
			blockInfo.Block.Header.ChainId, blockInfo.Block.Header.BlockHeight)
		return errGetBatchPool
	}
	batch.ReSet()

	//batch := types.NewUpdateBatch()
	block := blockInfo.Block

	// 1. Concurrency batch Put
	//并发更新cache,ConcurrencyMap,提升并发更新能力
	//groups := len(blockInfo.SerializedContractEvents)
	wg := &sync.WaitGroup{}
	wg.Add(len(blockInfo.SerializedTxs))
	heightKey := constructBlockNumKey(block.Header.BlockHeight)

	for index, txBytes := range blockInfo.SerializedTxs {
		go func(index int, txBytes []byte, batch protocol.StoreBatcher, wg *sync.WaitGroup) {
			defer wg.Done()
			tx := blockInfo.Block.Txs[index]
			txIdKey := constructTxIDKey(tx.Payload.TxId)
			batch.Put(txIdKey, heightKey)

			r.logger.Debugf("chain[%s]: blockInfo[%d] batch transaction index[%d] txid[%s]",
				block.Header.ChainId, block.Header.BlockHeight, index, tx.Payload.TxId)
		}(index, txBytes, batch, wg)
	}

	wg.Wait()
	r.startBlockHeight = blockInfo.Block.Header.BlockHeight
	r.endBlockHeight = r.startBlockHeight
	r.currCount = uint64(len(blockInfo.Txs))
	r.Cache.AddBlock(block.Header.BlockHeight, batch)
	r.lastBlockHeight = blockInfo.Block.Header.BlockHeight

	return nil
}

// Has 判断 key 在 start >r.startBlockHeight条件下，在 [start,r.endBlockHeight]区间内 是否存在
// @Description:
// @receiver r
// @param key
// @param start
// @return bool start是否在滑动窗口内
// @return bool key是否存在
// @return error
func (r *RollingWindowCacher) Has(key string, start uint64) (bool, bool, error) {
	r.RLock()
	defer r.RUnlock()

	txKey := bytesconv.BytesToString(constructTxIDKey(key))
	//fmt.Println("start ,r.startBlockHeight=",start," ",r.startBlockHeight)
	if start < r.startBlockHeight || start > r.endBlockHeight {
		return false, false, nil
	}

	//_, b := r.Cache.Get(txKey)
	_, b := r.Cache.HasFromHeight(txKey, start, r.endBlockHeight)
	return true, b, nil
}

// Consumer  异步消费 管道数据，完成对滑动窗口缓存的更新
// @Description:
// @receiver r
func (r *RollingWindowCacher) Consumer() {
	defer func() {
		if err := recover(); err != nil {
			r.logger.Errorf("consumer has a error in rollingWindowCache, errinfo:[%s]", err)
		}
	}()

	for {
		blockInfo, chanOk := <-r.blockSerChan
		if !chanOk {
			r.logger.Infof("blockSerChan close in rollingWindowCache")
			return
		}
		if blockInfo == nil {
			r.logger.Errorf("blockInfo is nil")
			panic(errBlockNil)
		}
		if blockInfo.Block == nil {
			r.logger.Errorf("Block is nil")
			panic(errBlockInfoNil)
		}
		if blockInfo.Block.Header == nil {
			r.logger.Errorf("Block header is nil")
			panic(errBlockHeaderNil)
		}

		//}
		//for blockInfo := range r.blockSerChan {

		batch, ok := r.batchPool.Get().(*types.UpdateBatch)
		if !ok {
			r.logger.Errorf("chain[%s]: blockInfo[%d] get updatebatch error in rollingWindowCache",
				blockInfo.Block.Header.ChainId, blockInfo.Block.Header.BlockHeight)
		}
		batch.ReSet()

		//batch := types.NewUpdateBatch()
		block := blockInfo.Block

		if block.Header.BlockHeight <= r.lastBlockHeight {
			continue
		}
		// 1. Concurrency batch Put
		//并发更新cache,ConcurrencyMap,提升并发更新能力
		//groups := len(blockInfo.SerializedContractEvents)
		wg := &sync.WaitGroup{}
		wg.Add(len(blockInfo.SerializedTxs))
		heightKey := constructBlockNumKey(block.Header.BlockHeight)

		for index, txBytes := range blockInfo.SerializedTxs {
			go func(index int, txBytes []byte, batch protocol.StoreBatcher, wg *sync.WaitGroup) {
				defer wg.Done()
				defer func() {
					if err := recover(); err != nil {
						r.logger.Errorf("SerializedTxs error in rollingWindowCache, errinfo:[%s]", err)
					}
				}()
				tx := blockInfo.Block.Txs[index]
				txIdKey := constructTxIDKey(tx.Payload.TxId)
				batch.Put(txIdKey, heightKey)

				r.logger.Debugf("chain[%s]: blockInfo[%d] batch transaction index[%d] txid[%s]",
					block.Header.ChainId, block.Header.BlockHeight, index, tx.Payload.TxId)
			}(index, txBytes, batch, wg)
		}

		wg.Wait()

		// 2. 获得窗口左边界数据，交易数量
		startBatch, err := r.Cache.GetBatch(r.startBlockHeight)
		startTxIdCount := 0
		if err != nil {
			r.logger.Infof("chain[%s]: blockInfo[%d] can not get GetBatch  in rollingWindowCache, info:[%s]",
				blockInfo.Block.Header.ChainId, blockInfo.Block.Header.BlockHeight, err)
		} else {
			startTxIdCount = startBatch.Len()
		}

		r.Lock()

		// 3. 滑动一个窗口 右滑动一个单位
		r.Cache.AddBlock(block.Header.BlockHeight, batch)
		r.lastBlockHeight = block.Header.BlockHeight
		r.endBlockHeight = block.Header.BlockHeight
		//如果当前窗口没有元素，更新窗口左边界
		if r.currCount == 0 {
			r.startBlockHeight = block.Header.BlockHeight
		}
		// cache中当前总交易量增加
		r.currCount = r.currCount + uint64(len(blockInfo.Txs))

		// 4. 滑动一个窗口 左滑动一个单位
		//如果cache中数据大于等于 缓存配置大小，则，删除最旧的块
		if r.currCount > r.txIdCount {
			r.Cache.DelBlock(r.startBlockHeight)
			r.startBlockHeight++
			r.currCount = r.currCount - uint64(startTxIdCount)
			//放回对象池
			r.batchPool.Put(startBatch)
		}
		r.Unlock()

	}

}

// constructTxIDKey 给交易添加交易前缀
// @Description:
// @param txId
// @return []byte
func constructTxIDKey(txId string) []byte {
	return append([]byte{txIDIdxKeyPrefix}, bytesconv.StringToBytes(txId)...)
}

// constructBlockNumKey 给区块高度添加一个头
// @Description:
// @param blockNum
// @return []byte
func constructBlockNumKey(blockNum uint64) []byte {
	blkNumBytes := encodeBlockNum(blockNum)
	return append([]byte{blockNumIdxKeyPrefix}, blkNumBytes...)
}

// encodeBlockNum 序列化 整型数据
// @Description:
// @param blockNum
// @return []byte
func encodeBlockNum(blockNum uint64) []byte {
	return proto.EncodeVarint(blockNum)
}
