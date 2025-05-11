/*
 * Copyright (C) BABEC. All rights reserved.
 * Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
 *
 * SPDX-License-Identifier: Apache-2.0
 */

package dbprovider

import (
	"fmt"
	"strings"

	"chainmaker.org/chainmaker/common/v2/crypto"
	"chainmaker.org/chainmaker/protocol/v2"
	badgerdbprovider "chainmaker.org/chainmaker/store-badgerdb/v2"
	leveldbprovider "chainmaker.org/chainmaker/store-leveldb/v2"
	rawsqlprovider "chainmaker.org/chainmaker/store-sqldb/v2"
	tikvdbprovider "chainmaker.org/chainmaker/store-tikv/v2"
	"chainmaker.org/chainmaker/store/v2/conf"
	"github.com/mitchellh/mapstructure"
)

// NewDBFactory new factory
// @Description:
// @return *DBFactory
func NewDBFactory() *DBFactory {
	return &DBFactory{}
}

// DBFactory db factory struct
// @Description:
type DBFactory struct {
	//ioc container.Container
}

// NewKvDB 创建kvdb 对象，根据配置中，配置的不同的provider，创建不同的db对象
//  @Description:
//  @receiver f
//  @return unc
func (f *DBFactory) NewKvDB(chainId, providerName, dbFolder string, config map[string]interface{},
	logger protocol.Logger, encryptor crypto.SymmetricKey) (protocol.DBHandle, error) {
	providerName = strings.ToLower(providerName)
	switch providerName {
	//leveldb
	case conf.DbconfigProviderLeveldb:
		dbConfig := &leveldbprovider.LevelDbConfig{}
		err := mapstructure.Decode(config, dbConfig)
		if err != nil {
			return nil, err
		}
		input := &leveldbprovider.NewLevelDBOptions{
			Config:    dbConfig,
			Logger:    logger,
			Encryptor: encryptor,
			ChainId:   chainId,
			DbFolder:  dbFolder,
		}
		return leveldbprovider.NewLevelDBHandle(input), nil
	//badgerdb
	case conf.DbconfigProviderBadgerdb:
		dbConfig := &badgerdbprovider.BadgerDbConfig{}
		err := mapstructure.Decode(config, dbConfig)
		if err != nil {
			return nil, err
		}
		input := &badgerdbprovider.NewBadgerDBOptions{
			Config:    dbConfig,
			Logger:    logger,
			Encryptor: encryptor,
			ChainId:   chainId,
			DbFolder:  dbFolder,
		}
		return badgerdbprovider.NewBadgerDBHandle(input), nil
	//tikvdb
	case conf.DbconfigProviderTikvdb:
		dbConfig := &tikvdbprovider.TiKVDbConfig{}
		err := mapstructure.Decode(config, dbConfig)
		if err != nil {
			return nil, err
		}
		input := &tikvdbprovider.NewTikvDBOptions{
			Config:    dbConfig,
			Logger:    logger,
			Encryptor: encryptor,
			ChainId:   chainId,
			DbName:    dbFolder,
		}
		return tikvdbprovider.NewTiKVDBHandle(input), nil
	//sqlkv,sqlkv 可以用mysql，但是mysql当kvdb来使用
	case conf.DbconfigProviderSqlKV:
		dbConfig := &rawsqlprovider.SqlDbConfig{}
		err := mapstructure.Decode(config, dbConfig)
		if err != nil {
			return nil, err
		}
		input := &rawsqlprovider.NewSqlDBOptions{
			Config:    dbConfig,
			Logger:    logger,
			Encryptor: encryptor,
			ChainId:   chainId,
			DbName:    dbFolder,
		}
		return rawsqlprovider.NewKVDBHandle(input), nil
	//memdb,测试时，ut时使用
	case conf.DbconfigProviderMemdb:
		return leveldbprovider.NewMemdbHandle(), nil
	default:
		return nil, fmt.Errorf("unsupported provider:%s", providerName)
	}
}

// NewSqlDB 创建 sqldb 对象
//  @Description:
//  @receiver f
//  @return unc
func (f *DBFactory) NewSqlDB(chainId, dbName string, config map[string]interface{},
	logger protocol.Logger) (
	protocol.SqlDBHandle, error) {
	dbConfig := &rawsqlprovider.SqlDbConfig{}
	//读取配置，初始化dbconfig对象
	err := mapstructure.Decode(config, dbConfig)
	if err != nil {
		return nil, err
	}
	input := &rawsqlprovider.NewSqlDBOptions{
		Config:    dbConfig,
		Logger:    logger,
		Encryptor: nil,
		ChainId:   chainId,
		DbName:    dbName,
	}
	logger.Debugf("initial new sql db for %s", dbName)
	return rawsqlprovider.NewSqlDBHandle(input), nil
}
