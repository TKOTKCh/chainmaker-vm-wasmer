/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0

*/

package historysqldb

import (
	commonPb "chainmaker.org/chainmaker/pb-go/v2/common"
	"chainmaker.org/chainmaker/protocol/v2"
	"chainmaker.org/chainmaker/store/v2/conf"
	"chainmaker.org/chainmaker/store/v2/historydb"
	"chainmaker.org/chainmaker/store/v2/serialization"
	"chainmaker.org/chainmaker/store/v2/types"
)

// HistorySqlDB provider a implementation of `history.HistoryDB`
// @Description:
// This implementation provides a mysql based data model
type HistorySqlDB struct {
	db     protocol.SqlDBHandle
	logger protocol.Logger
	dbName string
	config *conf.HistoryDbConfig
}

// NewHistorySqlDB construct a new `HistoryDB` for given chainId
// @Description:
// @param dbName
// @param config
// @param db
// @param logger
// @return *HistorySqlDB
func NewHistorySqlDB(dbName string, config *conf.HistoryDbConfig, db protocol.SqlDBHandle,
	logger protocol.Logger) *HistorySqlDB {
	historyDB := &HistorySqlDB{
		db:     db,
		logger: logger,
		dbName: dbName,
		config: config,
	}
	return historyDB
}

// initDb 如果数据库不存在，则创建数据库，然后切换到这个数据库，创建表
// 如果数据库存在，则切换数据库，检查表是否存在，不存在则创建表。
// @Description:
// @receiver db
// @param dbName
func (db *HistorySqlDB) initDb(dbName string) {
	db.logger.Debugf("create history database:%s", dbName)
	_, err := db.db.CreateDatabaseIfNotExist(dbName)
	if err != nil {
		db.logger.Panicf("init state sql db fail,error:%s", err)
	}
	db.logger.Debug("create table[state_history_infos] to save history")
	err = db.db.CreateTableIfNotExist(&StateHistoryInfo{})
	if err != nil {
		db.logger.Panicf("init state sql db table `state_history_infos` fail, error:%s", err)
	}
	err = db.db.CreateTableIfNotExist(&AccountTxHistoryInfo{})
	if err != nil {
		db.logger.Panicf("init state sql db table `account_tx_history_infos` fail, error:%s", err)
	}
	err = db.db.CreateTableIfNotExist(&ContractTxHistoryInfo{})
	if err != nil {
		db.logger.Panicf("init state sql db table `contract_tx_history_infos` fail, error:%s", err)
	}
	err = db.db.CreateTableIfNotExist(&types.SavePoint{})
	if err != nil {
		db.logger.Panicf("init state sql db table `save_points` fail, error:%s", err)
	}
	_, err = db.db.Save(&types.SavePoint{})
	if err != nil {
		db.logger.Panicf("insert new SavePoint get an error:%s", err)
	}

}

// InitGenesis init genesis block
// @Description:
// @receiver h
// @param genesisBlock
// @return error
func (h *HistorySqlDB) InitGenesis(genesisBlock *serialization.BlockWithSerializedInfo) error {
	h.initDb(h.dbName)
	return h.CommitBlock(genesisBlock, false)
}

// saveKeyHistory save all txWrite keys
// @Description:
// @receiver h
// @param dbtx
// @param txRWSets
// @param height
// @return error
func (h *HistorySqlDB) saveKeyHistory(dbtx protocol.SqlDBTransaction, txRWSets []*commonPb.TxRWSet,
	height uint64) error {
	for _, txRWSet := range txRWSets {
		for _, w := range txRWSet.TxWrites {
			if len(w.Key) == 0 {
				continue
			}
			historyInfo := NewStateHistoryInfo(w.ContractName, txRWSet.TxId, w.Key, height)
			_, err := dbtx.Save(historyInfo)
			if err != nil {
				h.logger.Errorf("save tx[%s] state key[%s] history info fail,rollback history save transaction,%s",
					txRWSet.TxId, w.Key, err.Error())
				return err
			}
		}
	}
	return nil
}

// saveTxHistory save account history and contract history , according to config
// @Description:
// @receiver h
// @param dbtx
// @param txs
// @param height
// @return error
func (h *HistorySqlDB) saveTxHistory(dbtx protocol.SqlDBTransaction, txs []*commonPb.Transaction,
	height uint64) error {
	for _, tx := range txs {
		txSender := tx.GetSenderAccountId()
		if len(txSender) == 0 {
			continue //genesis block tx don't have sender
		}
		if !h.config.DisableAccountHistory {
			accountTxInfo := &AccountTxHistoryInfo{
				AccountId:   txSender,
				BlockHeight: height,
				TxId:        tx.Payload.TxId,
			}
			_, err := dbtx.Save(accountTxInfo)
			if err != nil {
				h.logger.Errorf("save account[%s] and tx[%s] info fail,rollback history save transaction,%s",
					txSender, tx.Payload.TxId, err.Error())
				return err
			}
		}
		if !h.config.DisableContractHistory {
			contractName := tx.Payload.ContractName
			contractTxInfo := &ContractTxHistoryInfo{
				ContractName: contractName,
				BlockHeight:  height,
				TxId:         tx.Payload.TxId,
				AccountId:    txSender,
			}
			_, err := dbtx.Save(contractTxInfo)
			if err != nil {
				h.logger.Errorf(
					"save contract[%s] and tx[%s] history info fail,rollback history save transaction,%s",
					contractName, tx.Payload.TxId, err.Error())
				return err
			}
		}
	}
	return nil
}

// CommitBlock save write key history、account tx history、contract tx history, accourding to config
// and update save point (block height)
// @Description:
// @receiver h
// @param blockInfo
// @param isCache
// @return error
func (h *HistorySqlDB) CommitBlock(blockInfo *serialization.BlockWithSerializedInfo, isCache bool) error {
	block := blockInfo.Block
	blockHashStr := block.GetBlockHashStr()
	dbtx, err := h.db.BeginDbTransaction(blockHashStr)
	if err != nil {
		return err
	}
	if !h.config.DisableKeyHistory {
		if err = h.saveKeyHistory(dbtx, blockInfo.TxRWSets, block.Header.BlockHeight); err != nil {
			err2 := h.db.RollbackDbTransaction(blockHashStr)
			if err2 != nil {
				return err2
			}
		}
	}
	if !h.config.DisableAccountHistory || !h.config.DisableContractHistory {
		if err = h.saveTxHistory(dbtx, block.Txs, block.Header.BlockHeight); err != nil {
			err2 := h.db.RollbackDbTransaction(blockHashStr)
			if err2 != nil {
				return err2
			}
		}
	}

	//save last point
	_, err = dbtx.ExecSql("update save_points set block_height=?", block.Header.BlockHeight)
	if err != nil {
		h.logger.Errorf("update save point error:%s", err)
		err2 := h.db.RollbackDbTransaction(blockHashStr)
		if err2 != nil {
			return err2
		}
		return err
	}
	err = h.db.CommitDbTransaction(blockHashStr)
	if err != nil {
		return err
	}

	h.logger.Debugf("chain[%s]: commit block[%d] sql historydb",
		block.Header.ChainId, block.Header.BlockHeight)
	return nil

}

// GetLastSavepoint query last save point (block height)
// @Description:
// @receiver s
// @return uint64
// @return error
func (s *HistorySqlDB) GetLastSavepoint() (uint64, error) {
	sql := "select block_height from save_points"
	row, err := s.db.QuerySingle(sql)
	if err != nil {
		return 0, err
	}
	var height *uint64
	err = row.ScanColumns(&height)
	if err != nil {
		return 0, err
	}
	if height == nil {
		return 0, nil
	}
	return *height, nil
}

// Close close db
// @Description:
// @receiver h
func (h *HistorySqlDB) Close() {
	h.logger.Info("close history sql db")
	h.db.Close()
}

// hisIter implement iterator pattern
// @Description:
type hisIter struct {
	rows protocol.SqlRows
}

// Next check have value
// @Description:
// @receiver hi
// @return bool
func (hi *hisIter) Next() bool {
	return hi.rows.Next()
}

// Value get current value
// @Description:
// @receiver hi
// @return *historydb.BlockHeightTxId
// @return error
func (hi *hisIter) Value() (*historydb.BlockHeightTxId, error) {
	var txId string
	var blockHeight uint64
	err := hi.rows.ScanColumns(&txId, &blockHeight)
	if err != nil {
		return nil, err
	}
	return &historydb.BlockHeightTxId{TxId: txId, BlockHeight: blockHeight}, nil
}

// Release release the iterator
// @Description:
// @receiver hi
func (hi *hisIter) Release() {
	hi.rows.Close()
}

// newHisIter initiate an iterator
// @Description:
// @param rows
// @return *hisIter
func newHisIter(rows protocol.SqlRows) *hisIter {
	return &hisIter{rows: rows}
}

// GetHistoryForKey construct an iterator, using data from state_history_infos
// @Description:
// @receiver h
// @param contractName
// @param key
// @return historydb.HistoryIterator
// @return error
func (h *HistorySqlDB) GetHistoryForKey(contractName string, key []byte) (historydb.HistoryIterator, error) {
	sql := `select tx_id,block_height 
from state_history_infos 
where contract_name=? and state_key=? 
order by block_height desc`
	rows, err := h.db.QueryMulti(sql, contractName, key)
	if err != nil {
		return nil, err
	}
	return newHisIter(rows), nil
}

// GetAccountTxHistory construct an iterator, using data from account_tx_history_infos
// @Description:
// @receiver h
// @param account
// @return historydb.HistoryIterator
// @return error
func (h *HistorySqlDB) GetAccountTxHistory(account []byte) (historydb.HistoryIterator, error) {
	sql := `select tx_id,block_height 
from account_tx_history_infos 
where account_id=? 
order by block_height desc`
	rows, err := h.db.QueryMulti(sql, account)
	if err != nil {
		return nil, err
	}
	return newHisIter(rows), nil
}

// GetContractTxHistory construct an iterator, using data from contract_tx_history_infos
// @Description:
// @receiver h
// @param contractName
// @return historydb.HistoryIterator
// @return error
func (h *HistorySqlDB) GetContractTxHistory(contractName string) (historydb.HistoryIterator, error) {
	sql := `select tx_id,block_height 
from contract_tx_history_infos 
where contract_name=? 
order by block_height desc`
	rows, err := h.db.QueryMulti(sql, contractName)
	if err != nil {
		return nil, err
	}
	return newHisIter(rows), nil
}
