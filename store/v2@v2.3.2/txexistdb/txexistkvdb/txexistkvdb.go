/*
 * Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
 *
 * SPDX-License-Identifier: Apache-2.0
 *
 */

package txexistkvdb

import (
	"encoding/binary"
	"sync"

	bytesconv "chainmaker.org/chainmaker/common/v2/bytehelper"
	"chainmaker.org/chainmaker/protocol/v2"
	"chainmaker.org/chainmaker/store/v2/serialization"
	"chainmaker.org/chainmaker/store/v2/types"
	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
)

const (
	blockNumIdxKeyPrefix = 'n'
	txIDIdxKeyPrefix     = 't'
	lastBlockNumKeyStr   = "lastBlockNumKey"
)

var (
	errGetBatchPool = errors.New("get updatebatch error")
)

// TxExistKvDB provider a implementation of `txexistdb.TxExistDB`
// @Description:
// This implementation provides a key-value based data model
type TxExistKvDB struct {
	dbHandle protocol.DBHandle
	logger   protocol.Logger
	sync.RWMutex
	batchPool sync.Pool
}

// NewTxExistKvDB 创建一个txExistKvDB
// @Description:
// @param chainId
// @param dbHandle
// @param logger
// @return *TxExistKvDB
func NewTxExistKvDB(chainId string, dbHandle protocol.DBHandle, logger protocol.Logger) *TxExistKvDB {
	//nWorkers := int64(runtime.NumCPU())
	t := &TxExistKvDB{
		dbHandle:  dbHandle,
		logger:    logger,
		batchPool: sync.Pool{},
	}
	t.batchPool.New = func() interface{} {
		return types.NewUpdateBatch()
	}
	return t
}

// InitGenesis 初始化写入创世块:0号区块
// @Description:
// @receiver t
// @param genesisBlock
// @return error
func (t *TxExistKvDB) InitGenesis(genesisBlock *serialization.BlockWithSerializedInfo) error {
	//update TxExistKvDB
	return t.CommitBlock(genesisBlock, false)
}

// CommitBlock commits the txId and savepoint in an atomic operation
// @Description:
// @receiver t
// @param blockInfo
// @param isCache
// @return error
func (t *TxExistKvDB) CommitBlock(blockInfo *serialization.BlockWithSerializedInfo, isCache bool) error {
	//原则上，写db失败，重试一定次数后，仍然失败，panic

	//从对象池中取一个对象,并重置
	//拼接 txExistDB,需要用到的 txExist
	batch, ok := t.batchPool.Get().(*types.UpdateBatch)
	if !ok {
		t.logger.Errorf("chain[%s]: blockInfo[%d] get updatebatch error in txExistKvDB",
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
			//wg.Add(1)
			defer wg.Done()
			tx := blockInfo.Block.Txs[index]
			txIdKey := constructTxIDKey(tx.Payload.TxId)
			batch.Put(txIdKey, heightKey)

			t.logger.Debugf("chain[%s]: blockInfo[%d] batch transaction index[%d] txid[%s]",
				block.Header.ChainId, block.Header.BlockHeight, index, tx.Payload.TxId)
		}(index, txBytes, batch, wg)
	}

	wg.Wait()

	// 2.写 txExistKvDB
	if err := t.dbHandle.WriteBatch(batch, false); err != nil {
		t.logger.Errorf("chain[%s]: commit txExistKvDB blockInfo[%d]", block.Header.ChainId, block.Header.BlockHeight)
		return err
	}

	// 3.单独写 savepoint lastBlock height
	lastBlockNumBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(lastBlockNumBytes, block.Header.BlockHeight)
	if err := t.dbHandle.Put(bytesconv.StringToBytes(lastBlockNumKeyStr), lastBlockNumBytes); err != nil {
		t.logger.Errorf("chain[%s]: commit txExistKvDB blockInfo[%d] in lastsavepoint", block.Header.ChainId,
			block.Header.BlockHeight)
		return err
	}

	batch.ReSet()
	t.batchPool.Put(batch)

	return nil
}

// GetLastSavepoint returns the last block height
// @Description:
// @receiver t
// @return uint64
// @return error
func (t *TxExistKvDB) GetLastSavepoint() (uint64, error) {
	bytes, err := t.dbHandle.Get([]byte(lastBlockNumKeyStr))
	if err != nil {
		return 0, err
	} else if bytes == nil {
		return 0, nil
	}

	num := binary.BigEndian.Uint64(bytes)
	return num, nil
}

// TxExists returns true if the tx exist, or returns false if none exists.
// @Description:
// @receiver t
// @param txId
// @return bool
// @return error
func (t *TxExistKvDB) TxExists(txId string) (bool, error) {
	return t.dbHandle.Has(constructTxIDKey(txId))
}

// Close is used to close database
// @Description:
// @receiver t
func (t *TxExistKvDB) Close() {
	t.logger.Info("close block kv db")
	t.dbHandle.Close()
}

// constructTxIDKey 对txid转换，并添加前缀
// @Description:
// @param txId
// @return []byte
func constructTxIDKey(txId string) []byte {
	return append([]byte{txIDIdxKeyPrefix}, bytesconv.StringToBytes(txId)...)
}

// constructBlockNumKey  块高转换成byte并添加头
// @Description:
// @param blockNum
// @return []byte
func constructBlockNumKey(blockNum uint64) []byte {
	blkNumBytes := encodeBlockNum(blockNum)
	return append([]byte{blockNumIdxKeyPrefix}, blkNumBytes...)
}

// encodeBlockNum 序列化uint64转[]byte
// @Description:
// @param blockNum
// @return []byte
func encodeBlockNum(blockNum uint64) []byte {
	return proto.EncodeVarint(blockNum)
}
