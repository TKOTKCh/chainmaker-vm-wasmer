/*
 * Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
 *
 * SPDX-License-Identifier: Apache-2.0
 *
 */

package resultfiledb

import (
	"encoding/binary"
	"fmt"
	"sync"
	"time"

	storePb "chainmaker.org/chainmaker/pb-go/v2/store"
	"chainmaker.org/chainmaker/store/v2/binlog"
	"chainmaker.org/chainmaker/store/v2/conf"

	commonPb "chainmaker.org/chainmaker/pb-go/v2/common"
	"chainmaker.org/chainmaker/protocol/v2"
	"chainmaker.org/chainmaker/store/v2/cache"
	"chainmaker.org/chainmaker/store/v2/serialization"
	"chainmaker.org/chainmaker/store/v2/types"
	su "chainmaker.org/chainmaker/store/v2/utils"
)

const (
	txRWSetIndexKeyPrefix = "ri"
	resultDBSavepointKey  = "resultSavepointKey"
)

// ResultFileDB provider a implementation of `historydb.HistoryDB`
// @Description:
// This implementation provides a key-value based data model
type ResultFileDB struct {
	dbHandle    protocol.DBHandle
	cache       *cache.StoreCacheMgr
	logger      protocol.Logger
	storeConfig *conf.StorageConfig
	filedb      binlog.BinLogger
	sync.Mutex
}

// NewResultFileDB construct ResultFileDB
// @Description:
// @param chainId
// @param handle
// @param logger
// @param storeConfig
// @param fileDB
// @return *ResultFileDB
func NewResultFileDB(chainId string, handle protocol.DBHandle, logger protocol.Logger,
	storeConfig *conf.StorageConfig, fileDB binlog.BinLogger) *ResultFileDB {
	return &ResultFileDB{
		dbHandle:    handle,
		cache:       cache.NewStoreCacheMgr(chainId, 10, logger),
		logger:      logger,
		storeConfig: storeConfig,
		filedb:      fileDB,
	}
}

// InitGenesis init genesis block
// @Description:
// @receiver h
// @param genesisBlock
// @return error
func (h *ResultFileDB) InitGenesis(genesisBlock *serialization.BlockWithSerializedInfo) error {
	err := h.CommitBlock(genesisBlock, true)
	if err != nil {
		return err
	}
	return h.CommitBlock(genesisBlock, false)
}

// CommitBlock commits the block rwsets in an atomic operation
// @Description:
// @receiver h
// @param blockInfo
// @param isCache
// @return error
func (h *ResultFileDB) CommitBlock(blockInfo *serialization.BlockWithSerializedInfo, isCache bool) error {
	start := time.Now()
	if isCache {
		batch := types.NewUpdateBatch()
		// 1. last block height
		block := blockInfo.Block
		lastBlockNumBytes := make([]byte, 8)
		binary.BigEndian.PutUint64(lastBlockNumBytes, block.Header.BlockHeight)
		batch.Put([]byte(resultDBSavepointKey), lastBlockNumBytes)

		txRWSets := blockInfo.TxRWSets
		for index, txRWSet := range txRWSets {
			rwSetIndexKey := constructTxRWSetIndexKey(txRWSet.TxId)
			rwIndex := blockInfo.RWSetsIndex[index]
			rwIndexInfo := su.ConstructDBIndexInfo(blockInfo.Index, rwIndex.Offset, rwIndex.ByteLen)
			batch.Put(rwSetIndexKey, rwIndexInfo)
		}

		// update Cache
		h.cache.AddBlock(block.Header.BlockHeight, batch)

		h.logger.Infof("chain[%s]: commit result file cache block[%d], batch[%d], time used: %d",
			block.Header.ChainId, block.Header.BlockHeight, batch.Len(),
			time.Since(start).Milliseconds())
		return nil

	}
	// if IsCache == false ,update ResultFileDB
	batch := types.NewUpdateBatch()
	// 1. last block height
	block := blockInfo.Block
	lastBlockNumBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(lastBlockNumBytes, block.Header.BlockHeight)
	batch.Put([]byte(resultDBSavepointKey), lastBlockNumBytes)

	txRWSets := blockInfo.TxRWSets
	//rwsetData := blockInfo.SerializedTxRWSets

	for index, txRWSet := range txRWSets {
		rwSetIndexKey := constructTxRWSetIndexKey(txRWSet.TxId)
		rwIndex := blockInfo.RWSetsIndex[index]
		rwIndexInfo := su.ConstructDBIndexInfo(blockInfo.Index, rwIndex.Offset, rwIndex.ByteLen)
		batch.Put(rwSetIndexKey, rwIndexInfo)
	}

	err := h.writeBatch(block.Header.BlockHeight, batch)
	if err != nil {
		return err
	}
	return nil
}

// ShrinkBlocks archive old blocks rwsets in an atomic operation
// @Description:
// @receiver h
// @param txIdsMap
// @return error
func (h *ResultFileDB) ShrinkBlocks(txIdsMap map[uint64][]string) error {
	panic("not implement")
}

// RestoreBlocks not implement
// @Description:
// @receiver h
// @param blockInfos
// @return error
func (h *ResultFileDB) RestoreBlocks(blockInfos []*serialization.BlockWithSerializedInfo) error {
	panic("not implement")
}

// GetTxRWSet returns an txRWSet for given txId, or returns nil if none exists.
// @Description:
// @receiver h
// @param txId
// @return *commonPb.TxRWSet
// @return error
func (h *ResultFileDB) GetTxRWSet(txId string) (*commonPb.TxRWSet, error) {
	sinfo, err := h.GetRWSetIndex(txId)
	if err != nil {
		return nil, err
	}
	if sinfo == nil {
		return nil, nil
	}
	data, err := h.filedb.ReadFileSection(sinfo)
	if err != nil {
		return nil, err
	}
	rwset := &commonPb.TxRWSet{}
	err = rwset.Unmarshal(data)
	if err != nil {
		return nil, err
	}
	return rwset, nil
}

// GetRWSetIndex returns the offset of the block in the file
// @Description:
// @receiver h
// @param txId
// @return *storePb.StoreInfo
// @return error
func (h *ResultFileDB) GetRWSetIndex(txId string) (*storePb.StoreInfo, error) {
	// GetRWSetIndex returns the offset of the block in the file
	index, err := h.get(constructTxRWSetIndexKey(txId))
	if err != nil {
		return nil, err
	}

	return su.DecodeValueToIndex(index)
}

// GetLastSavepoint returns the last block height
// @Description:
// @receiver h
// @return uint64
// @return error
func (h *ResultFileDB) GetLastSavepoint() (uint64, error) {
	bytes, err := h.get([]byte(resultDBSavepointKey))
	if err != nil {
		return 0, err
	} else if bytes == nil {
		return 0, nil
	}
	num := binary.BigEndian.Uint64(bytes)
	return num, nil
}

// Close is used to close database
// @Description:
// @receiver h
func (h *ResultFileDB) Close() {
	h.logger.Info("close result file db")
	h.dbHandle.Close()
	h.cache.Clear()
}

// writeBatch save data,delete cache
// @Description:
// @receiver h
// @param blockHeight
// @param batch
// @return error
func (h *ResultFileDB) writeBatch(blockHeight uint64, batch protocol.StoreBatcher) error {
	//start := time.Now()
	//batches := batch.SplitBatch(102400)
	//batchDur := time.Since(start)
	//
	//wg := &sync.WaitGroup{}
	//wg.Add(len(batches))
	//for i := 0; i < len(batches); i++ {
	//	go func(index int) {
	//		defer wg.Done()
	//		if err := h.dbHandle.WriteBatch(batches[index], false); err != nil {
	//			panic(fmt.Sprintf("Error writing block db: %s", err))
	//		}
	//	}(i)
	//}
	//wg.Wait()
	//writeDur := time.Since(start)
	//h.cache.DelBlock(blockHeight)
	//
	//h.logger.Infof("write result file db, block[%d], time used: (batchSplit[%d]:%d, "+
	//	"write:%d, total:%d)", blockHeight, len(batches), batchDur.Milliseconds(),
	//	(writeDur - batchDur).Milliseconds(), time.Since(start).Milliseconds())

	err := h.dbHandle.WriteBatch(batch, false)
	if err != nil {
		panic(fmt.Sprintf("Error writing db: %s", err))
	}
	//db committed, clean cache
	h.cache.DelBlock(blockHeight)

	return nil
}

// get get value from cache,not found,get from db
// @Description:
// @receiver h
// @param key
// @return []byte
// @return error
func (h *ResultFileDB) get(key []byte) ([]byte, error) {
	//get from cache
	value, exist := h.cache.Get(string(key))
	if exist {
		return value, nil
	}
	//get from database
	return h.dbHandle.Get(key)
}

// constructTxRWSetIndexKey construct key
// format []byte{'r','i',txId}
// @Description:
// @param txId
// @return []byte
func constructTxRWSetIndexKey(txId string) []byte {
	key := append([]byte{}, txRWSetIndexKeyPrefix...)
	return append(key, txId...)
}
