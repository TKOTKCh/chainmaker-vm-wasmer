/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0

*/

// Package bigfilterkvdb package
// @Description:
package bigfilterkvdb

import (
	"encoding/binary"

	bytesconv "chainmaker.org/chainmaker/common/v2/bytehelper"
	"chainmaker.org/chainmaker/store/v2/bigfilterdb/filter"

	"sync"

	"chainmaker.org/chainmaker/protocol/v2"
	"chainmaker.org/chainmaker/store/v2/serialization"
	"chainmaker.org/chainmaker/store/v2/types"
)

const (
	txIDIdxKeyPrefix       = 't'
	lastBigFilterNumKeyStr = "lastBigFilterNumKey"
)

// BigFilterKvDB provider a implementation of `bigfilterdb.BigFilterDB`
// This implementation provides a key-value based data model
type BigFilterKvDB struct {
	filter filter.Filter
	logger protocol.Logger
	sync.RWMutex
	Cache types.ConcurrentMap
}

// NewBigFilterKvDB construct BigFilterKvDB
//  @Description:
//  @param n
//  @param m
//  @param fpRate
//  @param logger
//  @param redisServerCluster
//  @param pass
//  @param name
//  @return *BigFilterKvDB
//  @return error
func NewBigFilterKvDB(n int, m uint, fpRate float64, logger protocol.Logger, redisServerCluster []string,
	pass *string, name string) (*BigFilterKvDB, error) {
	bigFilter, err := filter.NewBigFilter(n, m, fpRate, redisServerCluster, pass, name, logger)
	if err != nil {
		return nil, err
	}
	t := &BigFilterKvDB{
		filter: bigFilter,
		logger: logger,
		Cache:  types.New(),
	}
	return t, nil
}

// InitGenesis 初始化创世块,写入创世块
//  @Description:
//  @receiver t
//  @param genesisBlock
//  @return error
func (t *BigFilterKvDB) InitGenesis(genesisBlock *serialization.BlockWithSerializedInfo) error {
	//update BigFilterKvDB
	if err := t.CommitBlock(genesisBlock, true); err != nil {
		return err
	}
	return t.CommitBlock(genesisBlock, false)
}

// CommitBlock commits the txId and savepoint in an atomic operation
//  @Description:
//  @receiver t
//  @param blockInfo
//  @param isCache
//  @return error
func (t *BigFilterKvDB) CommitBlock(blockInfo *serialization.BlockWithSerializedInfo, isCache bool) error {
	//原则上，写db失败，重试一定次数后，仍然失败，panic
	if isCache {
		block := blockInfo.Block

		lastBlockHeightBytes := make([]byte, 8)
		binary.BigEndian.PutUint64(lastBlockHeightBytes, block.Header.BlockHeight)
		heightStr := bytesconv.BytesToString(lastBlockHeightBytes)
		//将lastSavePoint保存到cache中
		t.Cache.Set(lastBigFilterNumKeyStr, heightStr)

		//将txid保存到cache中
		for index := range blockInfo.SerializedTxs {
			tx := blockInfo.Block.Txs[index]
			txIdKey := constructTxIDKey(tx.Payload.TxId)
			t.Cache.Set(bytesconv.BytesToString(txIdKey), heightStr)
		}
		return nil

	}

	block := blockInfo.Block

	// 1.获取块内全部txid
	txIds := [][]byte{}
	for index := range blockInfo.SerializedTxs {
		tx := blockInfo.Block.Txs[index]
		txIdKey := constructTxIDKey(tx.Payload.TxId)
		txIds = append(txIds, txIdKey)

		t.logger.Debugf("chain[%s]: blockInfo[%d] batch transaction index[%d] txid[%s]",
			block.Header.ChainId, block.Header.BlockHeight, index, tx.Payload.TxId)
	}
	// 2.更新bigfilter
	err := t.filter.AddMult(txIds)
	if err != nil {
		t.logger.Errorf("chain[%s]: blockInfo[%d] filter AddMult error in BigFilterKvDB,err:[%s]",
			blockInfo.Block.Header.ChainId, blockInfo.Block.Header.BlockHeight, err)
		panic(err)
		//return err
	}

	// 3.更新lastSavePoint
	lastBlockHeightBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(lastBlockHeightBytes, block.Header.BlockHeight)

	heightStr := bytesconv.BytesToString(lastBlockHeightBytes)
	if err := t.filter.SaveLastPoint(lastBigFilterNumKeyStr, heightStr); err != nil {
		t.logger.Errorf("chain[%s]: blockInfo[%d] filter SaveLastPoint error in BigFilterKvDB,err:[%s]",
			blockInfo.Block.Header.ChainId, blockInfo.Block.Header.BlockHeight, err)
		return err
	}

	return nil
}

// GetLastSavepoint returns the last block height
//  @Description:
//  @receiver t
//  @return uint64
//  @return error
func (t *BigFilterKvDB) GetLastSavepoint() (uint64, error) {
	//先从cache中查，如果cache中未找到，从db中查
	get, b := t.Cache.Get(lastBigFilterNumKeyStr)
	if b {
		last, ok := get.(string)
		if !ok {
			return 0, nil
		}
		minPoint := binary.BigEndian.Uint64(bytesconv.StringToBytes(last))
		return minPoint, nil

	}
	//未从cache中找到，从db中查找
	pointArr, err := t.filter.GetLastPoint(lastBigFilterNumKeyStr)
	//bytes, err := t.dbHandle.Get([]byte(lastBlockNumKeyStr))
	if err != nil {
		return 0, err
	}

	// 取出最小值
	minPoint := binary.BigEndian.Uint64(bytesconv.StringToBytes(pointArr[0]))

	for _, curr := range pointArr {
		c := binary.BigEndian.Uint64(bytesconv.StringToBytes(curr))
		if c < minPoint {
			minPoint = c
		}
	}
	return minPoint, nil

}

// TxExists add next time
//  @Description:
//  @receiver t
//  @param txId
//  @return bool
//  @return bool
//  @return error
// BigFilters has a non-zero probability of false positives
// BigFilters returns true if the tx exist, or returns false if none exists.
// Returns: (bool: true real exist, bool: true maybe exist  , error: errinfo)
// 如果从cache中找到了，那么key 真存在，第一个bool 返回true
// 如果cache中未找到，从t.filter中 也未找到，那么 第一个bool,第二个bool，返回false
// 如果cache中未找到，t.filter中找到，返回 false,true, 这时可能是假阳性的
func (t *BigFilterKvDB) TxExists(txId string) (bool, bool, error) {
	// 先查cache，存在直接返回，不存在，则再查过滤器，这里要区分真存在，还是不确定真存在 两种情况，否则击穿db
	tid := constructTxIDKey(txId)
	_, exist := t.Cache.Get(bytesconv.BytesToString(tid))
	if exist {
		t.logger.Infof("Tx[%s] already exist", txId)
		return true, true, nil
	}
	b, err := t.filter.Exist(constructTxIDKey(txId))
	if b {
		t.logger.Infof("Tx[%s] maybe exist", txId)
	}
	return false, b, err

}

// Close is used to close database
//  @Description:
//  @receiver t
func (t *BigFilterKvDB) Close() {
	t.logger.Info("close big filter kv db")
}

//  constructTxIDKey
//  @Description:
//  @param txId
//  @return []byte
func constructTxIDKey(txId string) []byte {
	return append([]byte{txIDIdxKeyPrefix}, bytesconv.StringToBytes(txId)...)
}
