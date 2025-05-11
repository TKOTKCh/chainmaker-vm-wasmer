/*
 * Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
 *
 * SPDX-License-Identifier: Apache-2.0
 *
 */

package serialization

import (
	"sync"

	commonPb "chainmaker.org/chainmaker/pb-go/v2/common"
	storePb "chainmaker.org/chainmaker/pb-go/v2/store"

	"github.com/gogo/protobuf/proto"
)

// BlockWithSerializedInfo contains block,txs and corresponding serialized data
// @Description:
type BlockWithSerializedInfo struct {
	Block                    *commonPb.Block
	Meta                     *storePb.SerializedBlock //Block without Txs
	SerializedMeta           []byte
	Txs                      []*commonPb.Transaction
	SerializedTxs            [][]byte
	TxRWSets                 []*commonPb.TxRWSet
	SerializedTxRWSets       [][]byte
	ContractEvents           []*commonPb.ContractEvent
	SerializedContractEvents [][]byte
	//整个Block+RWSet序列化后存储的位置
	Index *storePb.StoreInfo
	//Block基本信息（storePb.SerializedBlock）序列化后存储的位置
	MetaIndex *storePb.StoreInfo
	//交易列表序列化后的存储位置
	TxsIndex []*storePb.StoreInfo
	//读写集序列化后存储的位置
	RWSetsIndex []*storePb.StoreInfo
}

// NewBlockSerializedInfo 创建一个序列化对象
// @Description:
// @return *BlockWithSerializedInfo
func NewBlockSerializedInfo() *BlockWithSerializedInfo {
	meta := &storePb.SerializedBlock{}
	ser := &BlockWithSerializedInfo{
		Meta: meta,
	}
	return ser
}

// SerializeBlock serialized a BlockWithRWSet and return serialized data
// which combined as a BlockWithSerializedInfo
// @Description:
// @param blockWithRWSet
// @return []byte
// @return *BlockWithSerializedInfo
// @return error
func SerializeBlock(blockWithRWSet *storePb.BlockWithRWSet) ([]byte, *BlockWithSerializedInfo, error) {
	data, info, err := serializeBlockParallel(blockWithRWSet)
	//if err!=nil{
	//	data,info,err=serializeBlockSequence(blockWithRWSet)
	//}
	return data, info, err
}

// serializeBlockParallel 并行序列化
// @Description:
// 将block 和 rwset 序列化到一个 []byte
// @param blockWithRWSet
// @return []byte
// @return *BlockWithSerializedInfo
// @return error
func serializeBlockParallel(blockWithRWSet *storePb.BlockWithRWSet) ([]byte, *BlockWithSerializedInfo, error) {

	buf := proto.NewBuffer(nil)
	block := blockWithRWSet.Block
	txRWSets := blockWithRWSet.TxRWSets
	events := blockWithRWSet.ContractEvents
	info := &BlockWithSerializedInfo{}
	info.Block = block
	meta := &storePb.SerializedBlock{
		Header:         block.Header,
		Dag:            block.Dag,
		TxIds:          make([]string, 0, len(block.Txs)),
		AdditionalData: block.AdditionalData,
	}

	for _, tx := range block.Txs {
		meta.TxIds = append(meta.TxIds, tx.Payload.TxId)
		info.Txs = append(info.Txs, tx)
	}

	info.TxRWSets = append(info.TxRWSets, txRWSets...)
	info.Meta = meta
	info.ContractEvents = events
	//序列化 meta
	if err := info.serializeMeta(buf); err != nil {
		return nil, nil, err
	}
	//序列化 txs 交易
	if err := info.serializeTxs(buf); err != nil {
		return nil, nil, err
	}
	//序列化 交易读写集
	if err := info.serializeTxRWSets(buf); err != nil {
		return nil, nil, err
	}
	//序列化 ContractEvents
	if err := info.serializeEventTopicTable(buf); err != nil {
		return nil, nil, err
	}

	return buf.Bytes(), info, nil
}

// DeserializeBlock returns a deserialized block for given serialized bytes
// @Description:
// @param serializedBlock
// @return *storePb.BlockWithRWSet
// @return error
func DeserializeBlock(serializedBlock []byte) (*storePb.BlockWithRWSet, error) {
	b, err := deserializeBlockParallel(serializedBlock)
	if err != nil {
		b, err = deserializeBlockSequence(serializedBlock)
	}
	return b, err
}

// deserializeBlockParallel 并行反序列化
// @Description:
// @param serializedBlock
// @return *storePb.BlockWithRWSet
// @return error
func deserializeBlockParallel(serializedBlock []byte) (*storePb.BlockWithRWSet, error) {

	info := &BlockWithSerializedInfo{}
	buf := proto.NewBuffer(serializedBlock)
	var err error
	if info.Meta, err = info.deserializeMeta(buf); err != nil {
		return nil, err
	}
	if info.Txs, err = info.deserializeTxs(buf); err != nil {
		return nil, err
	}
	if info.TxRWSets, err = info.deserializeRWSets(buf); err != nil {
		return nil, err
	}
	if info.ContractEvents, err = info.deserializeEventTopicTable(buf); err != nil {
		return nil, err
	}
	block := &commonPb.Block{
		Header:         info.Meta.Header,
		Dag:            info.Meta.Dag,
		Txs:            info.Txs,
		AdditionalData: info.Meta.AdditionalData,
	}
	blockWithRWSet := &storePb.BlockWithRWSet{
		Block:          block,
		TxRWSets:       info.TxRWSets,
		ContractEvents: info.ContractEvents,
	}
	return blockWithRWSet, nil
}

// deserializeBlockSequence add next time
// @Description:
// @param serializedBlock
// @return *storePb.BlockWithRWSet
// @return error
func deserializeBlockSequence(serializedBlock []byte) (*storePb.BlockWithRWSet, error) {
	blockWithRWSet := &storePb.BlockWithRWSet{}
	err := blockWithRWSet.Unmarshal(serializedBlock)
	if err != nil {
		return nil, err
	}
	//info := &BlockWithSerializedInfo{}
	//info.Block = blockWithRWSet.Block
	//info.TxRWSets = blockWithRWSet.TxRWSets
	return blockWithRWSet, nil
}

// serializeMeta  序列化meta信息，主要包括 header,dag,txids,additionalData数据
// @Description:
// @receiver b
// @param buf
// @return error
func (b *BlockWithSerializedInfo) serializeMeta(buf *proto.Buffer) error {
	metaBytes, err := proto.Marshal(b.Meta)
	if err != nil {
		return err
	}
	b.MetaIndex = &storePb.StoreInfo{
		Offset:  uint64(len(buf.Bytes())) + getLenLeng(uint64(len(metaBytes))),
		ByteLen: uint64(len(metaBytes)),
	}
	if err := buf.EncodeRawBytes(metaBytes); err != nil {
		return err
	}
	b.SerializedMeta = metaBytes
	return nil
}

// SerializeMeta 序列化meta信息，主要包括 header,dag,txids,additionalData数据
// @Description:
// @receiver b
// @param buf
// @return error
func (b *BlockWithSerializedInfo) SerializeMeta(buf *proto.Buffer) error {
	metaBytes, err := proto.Marshal(b.Meta)
	if err != nil {
		return err
	}
	b.MetaIndex = &storePb.StoreInfo{
		Offset:  uint64(len(buf.Bytes())) + getLenLeng(uint64(len(metaBytes))),
		ByteLen: uint64(len(metaBytes)),
	}
	if err := buf.EncodeRawBytes(metaBytes); err != nil {
		return err
	}
	b.SerializedMeta = metaBytes
	return nil
}

// serializeEventTopicTable 序列化ContractEvents信息
// @Description:
// @receiver b
// @param buf
// @return error
func (b *BlockWithSerializedInfo) serializeEventTopicTable(buf *proto.Buffer) error {
	if err := buf.EncodeVarint(uint64(len(b.ContractEvents))); err != nil {
		return err
	}
	serializedEventList := make([][]byte, len(b.ContractEvents))
	batchSize := 1000
	taskNum := len(b.ContractEvents)/batchSize + 1
	errsChan := make(chan error, taskNum)
	wg := sync.WaitGroup{}
	wg.Add(taskNum)
	for taskId := 0; taskId < taskNum; taskId++ {
		startIndex := taskId * batchSize
		endIndex := (taskId + 1) * batchSize
		if endIndex > len(b.ContractEvents) {
			endIndex = len(b.ContractEvents)
		}
		go func(start int, end int) {
			defer wg.Done()
			for offset, et := range b.ContractEvents[start:end] {
				txBytes, err := proto.Marshal(et)
				if err != nil {
					errsChan <- err
				}
				serializedEventList[start+offset] = txBytes
			}
		}(startIndex, endIndex)
	}
	wg.Wait()
	if len(errsChan) > 0 {
		return <-errsChan
	}
	for _, txBytes := range serializedEventList {
		b.SerializedContractEvents = append(b.SerializedContractEvents, txBytes)
		if err := buf.EncodeRawBytes(txBytes); err != nil {
			return err
		}
	}
	return nil
}

// SerializeEventTopicTable 序列化ContractEvents信息
// @Description:
// @receiver b
// @param buf
// @return error
func (b *BlockWithSerializedInfo) SerializeEventTopicTable(buf *proto.Buffer) error {
	if err := buf.EncodeVarint(uint64(len(b.ContractEvents))); err != nil {
		return err
	}
	serializedEventList := make([][]byte, len(b.ContractEvents))
	batchSize := 1000
	taskNum := len(b.ContractEvents)/batchSize + 1
	errsChan := make(chan error, taskNum)
	wg := sync.WaitGroup{}
	wg.Add(taskNum)
	for taskId := 0; taskId < taskNum; taskId++ {
		startIndex := taskId * batchSize
		endIndex := (taskId + 1) * batchSize
		if endIndex > len(b.ContractEvents) {
			endIndex = len(b.ContractEvents)
		}
		go func(start int, end int) {
			defer wg.Done()
			for offset, et := range b.ContractEvents[start:end] {
				txBytes, err := proto.Marshal(et)
				if err != nil {
					errsChan <- err
				}
				serializedEventList[start+offset] = txBytes
			}
		}(startIndex, endIndex)
	}
	wg.Wait()
	if len(errsChan) > 0 {
		return <-errsChan
	}
	for _, txBytes := range serializedEventList {
		b.SerializedContractEvents = append(b.SerializedContractEvents, txBytes)
		if err := buf.EncodeRawBytes(txBytes); err != nil {
			return err
		}
	}
	return nil
}

// serializeTxs 序列化txs 交易信息
// @Description:
// @receiver b
// @param buf
// @return error
func (b *BlockWithSerializedInfo) serializeTxs(buf *proto.Buffer) error {
	if err := buf.EncodeVarint(uint64(len(b.Txs))); err != nil {
		return err
	}

	serializedTxList := make([][]byte, len(b.Txs))
	b.TxsIndex = make([]*storePb.StoreInfo, len(b.Txs))
	//txIds := make([]string, len(b.Txs))
	batchSize := 1000
	taskNum := len(b.Txs)/batchSize + 1
	errsChan := make(chan error, taskNum)
	wg := sync.WaitGroup{}
	wg.Add(taskNum)
	for taskId := 0; taskId < taskNum; taskId++ {
		startIndex := taskId * batchSize
		endIndex := (taskId + 1) * batchSize
		if endIndex > len(b.Txs) {
			endIndex = len(b.Txs)
		}
		go func(start int, end int) {
			defer wg.Done()
			for offset, tx := range b.Txs[start:end] {
				//txIds[start+offset] = tx.Payload.TxId
				txBytes, err := proto.Marshal(tx)
				if err != nil {
					errsChan <- err
				}
				serializedTxList[start+offset] = txBytes
			}
		}(startIndex, endIndex)
	}
	wg.Wait()
	if len(errsChan) > 0 {
		return <-errsChan
	}
	for i, txBytes := range serializedTxList {
		b.TxsIndex[i] = &storePb.StoreInfo{
			Offset:  uint64(len(buf.Bytes())) + getLenLeng(uint64(len(txBytes))),
			ByteLen: uint64(len(txBytes)),
		}
		b.SerializedTxs = append(b.SerializedTxs, txBytes)
		if err := buf.EncodeRawBytes(txBytes); err != nil {
			return err
		}
	}
	return nil
}

// getLenLeng 计算整型数据，被序列化后的长度
// @Description:
// @param x
// @return uint64
func getLenLeng(x uint64) uint64 {
	n := uint64(1)
	for x >= 1<<7 {
		x >>= 7
		n++
	}
	return n
}

// SerializeTxs 序列化txs信息
// @Description:
// @receiver b
// @param buf
// @return error
func (b *BlockWithSerializedInfo) SerializeTxs(buf *proto.Buffer) error {
	if err := buf.EncodeVarint(uint64(len(b.Txs))); err != nil {
		return err
	}

	serializedTxList := make([][]byte, len(b.Txs))
	b.TxsIndex = make([]*storePb.StoreInfo, len(b.Txs))
	batchSize := 1000
	taskNum := len(b.Txs)/batchSize + 1
	errsChan := make(chan error, taskNum)
	wg := sync.WaitGroup{}
	wg.Add(taskNum)

	//分组，并行序列化
	for taskId := 0; taskId < taskNum; taskId++ {
		startIndex := taskId * batchSize
		endIndex := (taskId + 1) * batchSize
		if endIndex > len(b.Txs) {
			endIndex = len(b.Txs)
		}
		go func(start int, end int) {
			defer wg.Done()
			for offset, tx := range b.Txs[start:end] {
				txBytes, err := proto.Marshal(tx)
				if err != nil {
					errsChan <- err
				}
				serializedTxList[start+offset] = txBytes
			}
		}(startIndex, endIndex)
	}
	wg.Wait()
	if len(errsChan) > 0 {
		return <-errsChan
	}
	//组合数据，记录索引信息
	for i, txBytes := range serializedTxList {
		b.TxsIndex[i] = &storePb.StoreInfo{
			Offset:  uint64(len(buf.Bytes())) + getLenLeng(uint64(len(txBytes))),
			ByteLen: uint64(len(txBytes)),
		}
		b.SerializedTxs = append(b.SerializedTxs, txBytes)
		if err := buf.EncodeRawBytes(txBytes); err != nil {
			return err
		}
	}
	return nil
}

// serializeTxRWSets 序列化读写集
// @Description:
// @receiver b
// @param buf
// @return error
func (b *BlockWithSerializedInfo) serializeTxRWSets(buf *proto.Buffer) error {

	if err := buf.EncodeVarint(uint64(len(b.TxRWSets))); err != nil {
		return err
	}

	serializedTxRWSets := make([][]byte, len(b.TxRWSets))
	b.RWSetsIndex = make([]*storePb.StoreInfo, len(b.TxRWSets))
	batchSize := 1000
	taskNum := len(b.TxRWSets)/batchSize + 1
	errsChan := make(chan error, taskNum)
	wg := sync.WaitGroup{}
	wg.Add(taskNum)
	//分组，并行序列化
	for taskId := 0; taskId < taskNum; taskId++ {
		startIndex := taskId * batchSize
		endIndex := (taskId + 1) * batchSize
		if endIndex > len(b.TxRWSets) {
			endIndex = len(b.TxRWSets)
		}
		go func(start int, end int) {
			defer wg.Done()
			for offset, txRWSet := range b.TxRWSets[start:end] {
				txRWSetBytes, err := proto.Marshal(txRWSet)
				if err != nil {
					errsChan <- err
				}
				serializedTxRWSets[start+offset] = txRWSetBytes
			}
		}(startIndex, endIndex)
	}
	wg.Wait()

	if len(errsChan) > 0 {
		return <-errsChan
	}
	//合并结果、获得索引信息
	for i, rwSetBytes := range serializedTxRWSets {
		// useless variant
		/*
			var rwsetIndex storePb.StoreInfo
			rwsetIndex.Offset = uint64(len(buf.Bytes())) + getLenLeng(uint64(len(rwSetBytes)))
		*/
		b.RWSetsIndex[i] = &storePb.StoreInfo{
			Offset:  uint64(len(buf.Bytes())) + getLenLeng(uint64(len(rwSetBytes))),
			ByteLen: uint64(len(rwSetBytes)),
		}
		b.SerializedTxRWSets = append(b.SerializedTxRWSets, rwSetBytes)
		if err := buf.EncodeRawBytes(rwSetBytes); err != nil {
			return err
		}
	}
	return nil
}

// SerializeTxRWSets 序列化读写集
// @Description:
// @receiver b
// @param buf
// @return error
func (b *BlockWithSerializedInfo) SerializeTxRWSets(buf *proto.Buffer) error {

	if err := buf.EncodeVarint(uint64(len(b.TxRWSets))); err != nil {
		return err
	}

	serializedTxRWSets := make([][]byte, len(b.TxRWSets))
	b.RWSetsIndex = make([]*storePb.StoreInfo, len(b.TxRWSets))
	batchSize := 1000
	taskNum := len(b.TxRWSets)/batchSize + 1
	errsChan := make(chan error, taskNum)
	wg := sync.WaitGroup{}
	wg.Add(taskNum)
	for taskId := 0; taskId < taskNum; taskId++ {
		startIndex := taskId * batchSize
		endIndex := (taskId + 1) * batchSize
		if endIndex > len(b.TxRWSets) {
			endIndex = len(b.TxRWSets)
		}
		go func(start int, end int) {
			defer wg.Done()
			for offset, txRWSet := range b.TxRWSets[start:end] {
				txRWSetBytes, err := proto.Marshal(txRWSet)
				if err != nil {
					errsChan <- err
				}
				serializedTxRWSets[start+offset] = txRWSetBytes
			}
		}(startIndex, endIndex)
	}
	wg.Wait()

	if len(errsChan) > 0 {
		return <-errsChan
	}
	// 更新索引信息
	for i, rwSetBytes := range serializedTxRWSets {
		var rwsetIndex storePb.StoreInfo
		rwsetIndex.Offset = uint64(len(buf.Bytes())) + getLenLeng(uint64(len(rwSetBytes)))
		b.RWSetsIndex[i] = &storePb.StoreInfo{
			Offset:  uint64(len(buf.Bytes())) + getLenLeng(uint64(len(rwSetBytes))),
			ByteLen: uint64(len(rwSetBytes)),
		}
		b.SerializedTxRWSets = append(b.SerializedTxRWSets, rwSetBytes)
		if err := buf.EncodeRawBytes(rwSetBytes); err != nil {
			return err
		}
	}

	return nil
}

// deserializeMeta 反序列化 meta 数据
// @Description:
// @receiver b
// @param buf
// @return *storePb.SerializedBlock
// @return error
func (b *BlockWithSerializedInfo) deserializeMeta(buf *proto.Buffer) (*storePb.SerializedBlock, error) {
	meta := &storePb.SerializedBlock{}
	serializedMeta, err := buf.DecodeRawBytes(false)
	if err != nil {
		return nil, err
	}
	err = proto.Unmarshal(serializedMeta, meta)
	if err != nil {
		return nil, err
	}
	return meta, nil
}

// DeserializeMeta 反序列化 meta 数据
// @Description:
// @param serializedBlock
// @return *storePb.SerializedBlock
// @return error
func DeserializeMeta(serializedBlock []byte) (*storePb.SerializedBlock, error) {
	info := &BlockWithSerializedInfo{}
	buf := proto.NewBuffer(serializedBlock)
	var err error
	meta, err := info.deserializeMeta(buf)
	if err != nil {
		return nil, err
	}
	return meta, nil
}

// deserializeTxs 反序列化 txs 数据
// @Description:
// @receiver b
// @param buf
// @return []*commonPb.Transaction
// @return error
func (b *BlockWithSerializedInfo) deserializeTxs(buf *proto.Buffer) ([]*commonPb.Transaction, error) {
	var txs []*commonPb.Transaction
	txNum, err := buf.DecodeVarint()
	if err != nil {
		return nil, err
	}
	for i := uint64(0); i < txNum; i++ {
		txBytes, err := buf.DecodeRawBytes(false)
		if err != nil {
			return nil, err
		}
		tx := &commonPb.Transaction{}
		if err = proto.Unmarshal(txBytes, tx); err != nil {
			return nil, err
		}
		txs = append(txs, tx)
	}
	return txs, nil
}

// deserializeRWSets 反序列化 读写集 数据
// @Description:
// @receiver b
// @param buf
// @return []*commonPb.TxRWSet
// @return error
func (b *BlockWithSerializedInfo) deserializeRWSets(buf *proto.Buffer) ([]*commonPb.TxRWSet, error) {
	var txRWSets []*commonPb.TxRWSet
	num, err := buf.DecodeVarint()
	if err != nil {
		return nil, err
	}
	for i := uint64(0); i < num; i++ {
		rwSetBytes, err := buf.DecodeRawBytes(false)
		if err != nil {
			return nil, err
		}
		rwSet := &commonPb.TxRWSet{}
		if err = proto.Unmarshal(rwSetBytes, rwSet); err != nil {
			return nil, err
		}
		txRWSets = append(txRWSets, rwSet)
	}
	return txRWSets, nil
}

// deserializeEventTopicTable 反序列化 ContractEvent 数据
// @Description:
// @receiver b
// @param buf
// @return []*commonPb.ContractEvent
// @return error
func (b *BlockWithSerializedInfo) deserializeEventTopicTable(buf *proto.Buffer) ([]*commonPb.ContractEvent, error) {
	var ets []*commonPb.ContractEvent
	txNum, err := buf.DecodeVarint()
	if err != nil {
		return nil, err
	}
	for i := uint64(0); i < txNum; i++ {
		txBytes, err := buf.DecodeRawBytes(false)
		if err != nil {
			return nil, err
		}
		et := &commonPb.ContractEvent{}
		if err = proto.Unmarshal(txBytes, et); err != nil {
			return nil, err
		}
		ets = append(ets, et)
	}
	return ets, nil
}

// ReSet 为sync.pool 重置 BlockWithSerializedInfo 状态时使用
// @Description:
// @receiver b
func (b *BlockWithSerializedInfo) ReSet() {
	//b.Meta not reset nil
	b.SerializedContractEvents = nil
	b.Block = nil
	b.Meta.AdditionalData = nil
	b.Meta.TxIds = nil
	b.Meta.Dag = nil
	b.Meta.Header = nil
	b.Txs = nil
	b.TxRWSets = nil
	b.ContractEvents = nil
	b.SerializedMeta = nil
	b.SerializedTxRWSets = nil
	b.SerializedTxs = nil
}
