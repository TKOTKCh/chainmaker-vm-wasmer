/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0

*/

package historykvdb

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"chainmaker.org/chainmaker/store/v2/historydb"
)

// constructKey
// format : k{contractName}#{key}#{blockHeight}#{txIndex}${txId}
// @Description: k+ContractName+StateKey+BlockHeight+TxId
// @param contractName
// @param key
// @param blockHeight
// @param txId
// @param txIndex
// @return []byte
func constructKey(contractName string, key []byte, blockHeight uint64, txId string, txIndex uint64) []byte {
	dbkey := fmt.Sprintf(keyHistoryPrefix+"%s"+splitChar+"%s"+splitChar+"%d"+splitChar+
		"%d"+splitCharPlusOne+"%s",
		contractName, key, blockHeight, txIndex, txId)
	return []byte(dbkey)
}

//出于sqlkv下key是varchar类型的考虑，key由字符串构成，不用bytes构成
////k+ContractName+StateKey+BlockHeight+DAGIndex+TxId
//func constructKeyWithDAGIndex(contractName string, key []byte, blockHeight uint64, txId string,
//	dagIndex uint64) []byte {
//	// blockHeight、DAGIndex转bytes表示
//	dbkey := make([]byte, 0)
//	dbkey = append(dbkey, []byte(keyHistoryPrefix+contractName)...)
//	dbkey = appendWithSplitChar(dbkey, key, splitChar)
//	blockHeightBytes, err := bytehelper.Uint64ToBytes(blockHeight)
//	if err != nil {
//		return nil
//	}
//	dbkey = appendWithSplitChar(dbkey, blockHeightBytes, splitChar)
//
//	dagIndexBytes, err := bytehelper.Uint64ToBytes(dagIndex)
//	if err != nil {
//		return nil
//	}
//	dbkey = appendWithSplitChar(dbkey, dagIndexBytes, splitChar)
//	dbkey = appendWithSplitChar(dbkey, []byte(txId), splitCharPlusOne)
//	return dbkey
//}

//func appendWithSplitChar(s1 []byte, s2 []byte, splitChar string) []byte {
//	var buf bytes.Buffer
//	buf.Write(s1)
//	if len(s2) > 0 {
//		buf.Write([]byte(splitChar))
//		buf.Write(s2)
//	}
//	return buf.Bytes()
//}

// constructKeyPrefix construct key prefix
// format : k{contractName}#{key}#
// @Description:
// @param contractName
// @param key
// @return []byte
func constructKeyPrefix(contractName string, key []byte) []byte {
	dbkey := fmt.Sprintf(keyHistoryPrefix+"%s"+splitChar+"%s"+splitChar, contractName, key)
	return []byte(dbkey)
}

// splitKey
// format :
// @Description: 将key分割成4个字段
// @param dbKey
// @return contractName
// @return key
// @return blockHeight
// @return txId
// @return err
func splitKey(dbKey []byte) (contractName string, key []byte, blockHeight uint64, txId string, err error) {
	if len(dbKey) == 0 {
		return "", nil, 0, "", errors.New("empty dbKey")
	}
	array := strings.Split(string(dbKey[1:]), splitChar)
	/*
		| index | description           |
		| ---   | ---                   |
		| 0     | kContractName         |
		| 1     | key                   |
		| 2     | field                 |
		| 3     | blockHeight           |
		| 4     | txId OR DAGIndex$TxId |
	*/
	if len(array) == 5 {
		contractName = array[0]
		key = []byte(array[1] + splitChar + array[2])
		var heightFromString int
		heightFromString, err = strconv.Atoi(array[3])
		if err != nil {
			return "", nil, 0, "", errors.New("invalid dbKey format")
		}
		blockHeight = uint64(heightFromString)

		// 兼容两个版本
		// 1. DAGIndex$TxId
		// 2. TxId
		array2 := strings.Split(array[4], splitCharPlusOne)
		txId = array2[len(array2)-1]
		return
	}

	/*
		| index | description           |
		| ---   | ---                   |
		| 0     | kContractName         |
		| 1     | key                   |
		| 2     | blockHeight           |
		| 3     | txId OR DAGIndex$TxId |
	*/
	if len(array) == 4 {
		contractName = array[0]
		key = []byte(array[1])
		var heightFromString int
		heightFromString, err = strconv.Atoi(array[2])
		if err != nil {
			return "", nil, 0, "", errors.
				New("invalid dbKey format")
		}
		blockHeight = uint64(heightFromString)

		array2 := strings.Split(array[3], splitCharPlusOne)
		txId = array2[len(array2)-1]
		return
	}

	return "", nil, 0, "", errors.New("invalid dbKey format")
}

// GetHistoryForKey construct an iterator,according to contractName and key
// @Description:
// @receiver h
// @param contractName
// @param key
// @return historydb.HistoryIterator
// @return error
func (h *HistoryKvDB) GetHistoryForKey(contractName string, key []byte) (historydb.HistoryIterator, error) {
	iter, erro := h.dbHandle.NewIteratorWithPrefix(constructKeyPrefix(contractName, key))
	if erro != nil {
		return nil, erro
	}
	splitKeyFunc := func(key []byte) (*historydb.BlockHeightTxId, error) {
		_, _, height, txId, err := splitKey(key)
		if err != nil {
			return nil, err
		}
		return &historydb.BlockHeightTxId{
			BlockHeight: height,
			TxId:        txId,
		}, nil
	}
	return &historyKeyIterator{dbIter: iter, buildFunc: splitKeyFunc}, nil
}
