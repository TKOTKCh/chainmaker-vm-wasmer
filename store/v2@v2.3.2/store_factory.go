/*
 * Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
 *
 * SPDX-License-Identifier: Apache-2.0
 *
 */
//
//store_factory 完成对存储对象的初始化，主要包括：
//	1.解析配置
//	2.根据配置文件创建 wal/bfdb、blockdb、statedb、resultdb、txExistdb、historydb、resultdb、cache、bigfilter 等对象
//	3.完成logger初始化
//	4.主要通过ioc 容器完成构造

package store

import (
	"encoding/hex"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"chainmaker.org/chainmaker/store/v2/rolling_window_cache"

	"chainmaker.org/chainmaker/store/v2/bigfilterdb"
	"chainmaker.org/chainmaker/store/v2/bigfilterdb/bigfilterkvdb"

	"chainmaker.org/chainmaker/common/v2/container"
	"chainmaker.org/chainmaker/common/v2/crypto"
	"chainmaker.org/chainmaker/common/v2/crypto/pkcs11"
	"chainmaker.org/chainmaker/common/v2/crypto/sym/aes"
	"chainmaker.org/chainmaker/common/v2/crypto/sym/sm4"
	"chainmaker.org/chainmaker/common/v2/wal"
	"chainmaker.org/chainmaker/protocol/v2"
	"chainmaker.org/chainmaker/store/v2/binlog"
	"chainmaker.org/chainmaker/store/v2/blockdb"
	"chainmaker.org/chainmaker/store/v2/blockdb/blockfiledb"
	"chainmaker.org/chainmaker/store/v2/blockdb/blockkvdb"
	"chainmaker.org/chainmaker/store/v2/blockdb/blocksqldb"
	"chainmaker.org/chainmaker/store/v2/cache"
	"chainmaker.org/chainmaker/store/v2/conf"
	"chainmaker.org/chainmaker/store/v2/contracteventdb"
	"chainmaker.org/chainmaker/store/v2/contracteventdb/eventsqldb"
	"chainmaker.org/chainmaker/store/v2/dbprovider"
	"chainmaker.org/chainmaker/store/v2/historydb"
	"chainmaker.org/chainmaker/store/v2/historydb/historykvdb"
	"chainmaker.org/chainmaker/store/v2/historydb/historysqldb"
	"chainmaker.org/chainmaker/store/v2/resultdb"
	"chainmaker.org/chainmaker/store/v2/resultdb/resultfiledb"
	"chainmaker.org/chainmaker/store/v2/resultdb/resultkvdb"
	"chainmaker.org/chainmaker/store/v2/resultdb/resultsqldb"
	"chainmaker.org/chainmaker/store/v2/serialization"
	"chainmaker.org/chainmaker/store/v2/statedb"
	"chainmaker.org/chainmaker/store/v2/statedb/statekvdb"
	"chainmaker.org/chainmaker/store/v2/statedb/statesqldb"
	"chainmaker.org/chainmaker/store/v2/test"
	"chainmaker.org/chainmaker/store/v2/txexistdb"
	"chainmaker.org/chainmaker/store/v2/txexistdb/txexistkvdb"
	"github.com/allegro/bigcache/v3"
)

// nolint
const (
	//StoreBlockDBDir blockdb folder name
	StoreBlockDBDir = "store_block"
	//StoreStateDBDir statedb folder name
	StoreStateDBDir = "store_state"
	//StoreHistoryDBDir historydb folder name
	StoreHistoryDBDir = "store_history"
	//StoreResultDBDir resultdb folder name
	StoreResultDBDir   = "store_result"
	StoreEventLogDBDir = "store_event_log"
	StoreLocalDBDir    = "localdb"
	StoreTxExistDbDir  = "store_txexist"

	DBName_BlockDB   = "blockdb"
	DBName_StateDB   = "statedb"
	DBName_HistoryDB = "historydb"
	DBName_ResultDB  = "resultdb"
	DBName_EventDB   = "eventdb"
	DBName_LocalDB   = "localdb"
	DBName_TxExistDB = "txexistdb"
)

// Factory is a factory function to create an instance of the block store
// which commits block into the ledger.
// @Description:
type Factory struct {
	ioc       *container.Container
	dbFactory *dbprovider.DBFactory
}

// NewFactory add next time
// @Description:
// @return *Factory
func NewFactory() *Factory {
	ioc := container.NewContainer()
	return &Factory{
		ioc:       ioc,
		dbFactory: dbprovider.NewDBFactory(),
	}
}

// NewStore constructs new BlockStore
// @Description:
// @receiver m
// @param chainId
// @param storeConfig
// @param logger
// @param p11Handle
// @return protocol.BlockchainStore
// @return error
func (m *Factory) NewStore(chainId string, storeConfig *conf.StorageConfig,
	logger protocol.Logger, p11Handle *pkcs11.P11Handle) (protocol.BlockchainStore, error) {

	dbConfig := storeConfig.BlockDbConfig
	if strings.ToLower(dbConfig.Provider) == "simple" {
		db, err := m.dbFactory.NewKvDB(chainId, conf.DbconfigProviderLeveldb, StoreBlockDBDir,
			dbConfig.LevelDbConfig, logger, nil)

		if err != nil {
			return nil, err
		}

		return test.NewDebugStore(logger, storeConfig, db), nil
	}
	if strings.ToLower(dbConfig.Provider) == "memory" {
		db, err := m.dbFactory.NewKvDB(chainId, conf.DbconfigProviderMemdb, StoreBlockDBDir,
			dbConfig.LevelDbConfig, logger, nil)

		if err != nil {
			return nil, err
		}

		return test.NewDebugStore(logger, storeConfig, db), nil
	}
	m.ioc = container.NewContainer()
	err := m.register(chainId, storeConfig, logger, p11Handle)
	if err != nil {
		return nil, err
	}
	var store protocol.BlockchainStore
	err = m.ioc.Resolve(&store)
	return store, err
}

// register 注册函数，存储对象中，所有成员变量 初始化入口
// @Description:
// @receiver m
// @param chainId
// @param storeConfig
// @param logger
// @param p11Handle
// @return error
func (m *Factory) register(chainId string, storeConfig *conf.StorageConfig, logger protocol.Logger,
	p11Handle *pkcs11.P11Handle) error {
	//注册logger
	err := m.ioc.Register(func() protocol.Logger { return logger })
	if err != nil {
		return err
	}
	//注册数据落盘对称数据加密对象
	if len(storeConfig.Encryptor) > 0 && len(storeConfig.EncryptKey) > 0 {
		//encryptor, err = buildEncryptor(storeConfig.Encryptor, storeConfig.EncryptKey, p11Handle)
		logger.Debugf("ioc register store encryptor")
		if err = m.ioc.Register(buildEncryptor, container.Parameters(map[int]interface{}{
			0: storeConfig.Encryptor,
			1: storeConfig.EncryptKey,
			2: p11Handle})); err != nil {
			return err
		}
	}
	// 注册wal 或者bfdb
	if err = m.registerFileOrLogDB(chainId, storeConfig, logger); err != nil {
		return err
	}

	//注册StateDB缓存
	if !storeConfig.DisableStateCache {
		if err = m.registerCache(storeConfig.StateCache); err != nil {
			return err
		}
	}
	//注册StateDB缓存
	if !storeConfig.DisableStateCache {
		if err = m.registerCache(storeConfig.StateCache); err != nil {
			return err
		}
	}
	//注册bigFilter
	if storeConfig.EnableBigFilter {
		if err = m.registerBigFilter(storeConfig.BigFilter, logger, chainId); err != nil {
			return err
		}
	}
	//注册 rollingWindowCache
	if err = m.registerRollingWindowCache(storeConfig, logger); err != nil {
		return err
	}
	if err = m.registerAllDbHandle(chainId, storeConfig); err != nil {
		return err
	}
	if err = m.registerBusinessDB(chainId, storeConfig); err != nil {
		return err
	}
	var bs protocol.BlockchainStore
	// 注册 store 对象，Optional 表示非必选的可选参数在 NewBlockStoreImpl 传入参数的编号顺序
	if err = m.ioc.Register(NewBlockStoreImpl, container.Parameters(map[int]interface{}{0: chainId, 1: storeConfig}),
		container.DependsOn(map[int]string{8: DBName_LocalDB}),
		//HistoryDB,ResultDB,EventLogDB,TxExistDB,binlog,bigFilter可选
		container.Optional(4, 5, 6, 7, 10, 11, 12),
		container.Interface(&bs),
		container.Name("sync_bs")); err != nil {
		return err
	}
	// 异步时，使用AsyncBlockStoreImpl
	if storeConfig.Async {
		if err = m.ioc.Register(NewAsyncBlockStoreImpl,
			container.DependsOn(map[int]string{0: "sync_bs"}),
			container.Default()); err != nil {
			return err
		}
	}
	return nil
}

// buildEncryptor 构建加密
// @Description:
// @param encryptor
// @param key
// @param p11Handle
// @return crypto.SymmetricKey
// @return error
func buildEncryptor(encryptor, key string, p11Handle *pkcs11.P11Handle) (crypto.SymmetricKey, error) {
	switch strings.ToLower(encryptor) {
	case "sm4":
		if p11Handle != nil {
			return pkcs11.NewSecretKey(p11Handle, key, crypto.SM4)
		}
		return &sm4.SM4Key{Key: readEncryptKey(key)}, nil
	case "aes":
		if p11Handle != nil {
			return pkcs11.NewSecretKey(p11Handle, key, crypto.AES)
		}
		return &aes.AESKey{Key: readEncryptKey(key)}, nil
	default:
		return nil, errors.New("unsupported encryptor:" + encryptor)
	}
}

// readEncryptKey 读加密key
// @Description:
// @param key
// @return []byte
func readEncryptKey(key string) []byte {
	reg := regexp.MustCompile("^0[xX][0-9a-fA-F]+$") //is hex
	if reg.Match([]byte(key)) {
		b, _ := hex.DecodeString(key[2:])
		return b
	}

	removeFile := func(path string) {
		if errw := ioutil.WriteFile(path, []byte(""), 0600); errw != nil {
			fmt.Printf("remove encrypt key file: %s failed! you can manual remove. err: %v \n", key, errw)
		}
	}

	if key[0] == '/' && pathExists(key) {
		f, err := ioutil.ReadFile(key)
		if err == nil {
			removeFile(key)
			return f
		}
	}
	//对于windows系统，临时文件以C开头
	if key[0] == 'C' && pathExists(key) {
		f, err := ioutil.ReadFile(key)
		if err == nil {
			removeFile(key)
			return f
		}
	}
	return []byte(key)
}

// pathExists 判断目录是否存在
// @Description:
// @param path
// @return bool
func pathExists(path string) bool {
	_, err := os.Stat(path)
	if err == nil {
		return true
	}
	return os.IsExist(err)
}

// registerCache  注册Cache 目前只支持bigcache
// @Description:
// @receiver m
// @param config
// @return error
func (m *Factory) registerCache(config *conf.CacheConfig) error {
	if config == nil || len(config.Provider) == 0 || strings.ToLower(config.Provider) == "bigcache" {
		err := m.ioc.Register(newBigCache,
			container.Parameters(map[int]interface{}{0: config}))
		if err != nil {
			return err
		}
	} else {
		panic("cache provider[" + config.Provider + "] not support")
	}
	return nil
}

// newBigCache 创建 bigCache
// @Description:
// @param cacheConfig
// @return cache.Cache
// @return error
func newBigCache(cacheConfig *conf.CacheConfig) (cache.Cache, error) {
	bigCacheDefaultConfig := bigcache.Config{
		// number of shards (must be a power of 2)
		Shards: 1024,
		// time after which entry can be evicted
		LifeWindow: 10 * time.Minute,
		// rps * lifeWindow, used only in initial memory allocation
		CleanWindow: 10 * time.Second,
		// max entry size in bytes, used only in initial memory allocation
		MaxEntrySize: 500,
		// prints information about additional memory allocation
		Verbose: true,
		// cache will not allocate more memory than this limit, value in MB
		// if value is reached then the oldest entries can be overridden for the new ones
		// 0 value means no size limit
		HardMaxCacheSize: 128,
		// callback fired when the oldest entry is removed because of its
		// expiration time or no space left for the new entry. Default value is nil which
		// means no callback and it prevents from unwrapping the oldest entry.
		OnRemove: nil,
	}
	// 配置文件不为空，使用给定配置
	if cacheConfig != nil {
		bigCacheDefaultConfig.LifeWindow = cacheConfig.LifeWindow
		bigCacheDefaultConfig.CleanWindow = cacheConfig.CleanWindow
		bigCacheDefaultConfig.MaxEntrySize = cacheConfig.MaxEntrySize
		bigCacheDefaultConfig.HardMaxCacheSize = cacheConfig.HardMaxCacheSize
	}
	return bigcache.NewBigCache(bigCacheDefaultConfig)
}

// registerBigFilter 注册 bigfilter
// @Description:
// @receiver m
// @param config
// @param logger
// @param name
// @return error
func (m *Factory) registerBigFilter(config *conf.BigFilterConfig, logger protocol.Logger, name string) error {
	err := m.ioc.Register(newBigFilter,
		container.Parameters(map[int]interface{}{0: config, 1: logger, 2: name}))
	if err != nil {
		return err
	}

	return nil
}

// newBigFilter 创建一个bigFilter
// @Description:
// @param bigFilterConfig
// @param logger
// @param name
// @return bigfilterdb.BigFilterDB
// @return error
func newBigFilter(bigFilterConfig *conf.BigFilterConfig, logger protocol.Logger,
	name string) (bigfilterdb.BigFilterDB, error) {
	redisHosts := strings.Split(bigFilterConfig.RedisHosts, ",")
	filterNum := len(redisHosts)
	// 密码为空
	if bigFilterConfig.Pass == "" {
		b, err := bigfilterkvdb.NewBigFilterKvDB(filterNum, bigFilterConfig.TxCapacity, bigFilterConfig.FpRate,
			logger, redisHosts, nil, name)
		if err != nil {
			return b, err
		}
	}
	b, err := bigfilterkvdb.NewBigFilterKvDB(filterNum, bigFilterConfig.TxCapacity, bigFilterConfig.FpRate,
		logger, redisHosts, &bigFilterConfig.Pass, name)
	if err != nil {
		return b, err
	}
	return b, nil

}

// registerRollingWindowCache
// @Description: 注册newRollingWindowCache 函数，主要 将 rwCache中的 rolling_window_cache_capacity 初始化赋值
// txIdCount 通过 配置文件中的rolling_window_cache_capacity 赋值
// @receiver m
// @param storeConfig
// @param logger
// @return error
func (m *Factory) registerRollingWindowCache(storeConfig *conf.StorageConfig, logger protocol.Logger) error {
	err := m.ioc.Register(newRollingWindowCache,
		container.Parameters(map[int]interface{}{0: storeConfig, 1: logger}))
	if err != nil {
		return err
	}

	return nil
}

// newRollingWindowCache
// @Description: 创建一个 RWCache
// @param storeConfig
// @param logger
// @return rolling_window_cache.RollingWindowCache
// @return error
func newRollingWindowCache(storeConfig *conf.StorageConfig,
	logger protocol.Logger) (rolling_window_cache.RollingWindowCache, error) {
	rollingWindowCacheCapacity := storeConfig.RollingWindowCacheCapacity
	// 未配置或者配置为0，则给默认 10000
	if storeConfig.RollingWindowCacheCapacity == 0 {
		rollingWindowCacheCapacity = 10000
	}
	r := rolling_window_cache.NewRollingWindowCacher(rollingWindowCacheCapacity,
		0, 0, 0, 0, logger)
	return r, nil

}

// registerDbHandle
// @Description: 注册 dbhandler
// @receiver m
// @param chainId
// @param dbConfig
// @param dbDir
// @param name
// @param dbPrefix
// @param enableCache
// @return error
func (m *Factory) registerDbHandle(chainId string, dbConfig *conf.DbConfig, dbDir, name, dbPrefix string,
	enableCache bool) error {
	if dbConfig.IsKVDB() {
		dbName := dbDir
		if dbConfig.Provider == conf.DbconfigProviderSqlKV {
			dbName = getDbName(dbPrefix, name, chainId)
		}
		config := dbConfig.GetDbConfig()
		var constructor interface{} = m.dbFactory.NewKvDB
		if enableCache { //如果启用了State缓存，则使用WrapCacheToDBHandle来替换StateDB
			constructor = func(chainId, providerName, dbFolder string, config map[string]interface{},
				logger protocol.Logger, encryptor crypto.SymmetricKey, c cache.Cache) (protocol.DBHandle, error) {
				innerDb, err := m.dbFactory.NewKvDB(chainId, providerName, dbFolder, config, logger, encryptor)
				if err != nil {
					return nil, err
				}
				return cache.NewCacheWrapToDBHandle(c, innerDb, logger), nil
			}
		}
		//注册kvdb
		err := m.ioc.Register(constructor,
			container.Parameters(map[int]interface{}{0: chainId, 1: dbConfig.Provider,
				2: dbName, 3: config}),
			container.Name(name),
			container.Optional(5)) //透明数据加密是可选的
		if err != nil {
			return err
		}

	} else {
		//注册sqldb
		dbName := getDbName(dbPrefix, name, chainId)
		err := m.ioc.Register(m.dbFactory.NewSqlDB,
			container.Parameters(map[int]interface{}{0: chainId,
				1: dbName, 2: dbConfig.SqlDbConfig}),
			container.Name(name),
			container.Lifestyle(true))
		if err != nil {
			return err
		}
	}
	return nil
}

// getDbName
// @Description: db名字 由前缀、dbname、chainId组成，多链不重复
// @param dbPrefix
// @param dbName
// @param chainId
// @return string
func getDbName(dbPrefix, dbName, chainId string) string {
	return dbPrefix + dbName + "_" + chainId
}

// registerAllDbHandle
// @Description: 注册 所有db
// @receiver m
// @param chainId
// @param storeConfig
// @return error
func (m *Factory) registerAllDbHandle(chainId string, storeConfig *conf.StorageConfig) error {
	//注册blockdb
	err := m.registerDbHandle(chainId, storeConfig.BlockDbConfig, StoreBlockDBDir,
		DBName_BlockDB, storeConfig.DbPrefix, false)
	if err != nil {
		return err
	}
	//注册statedb
	err = m.registerDbHandle(chainId, storeConfig.StateDbConfig, StoreStateDBDir,
		DBName_StateDB, storeConfig.DbPrefix, !storeConfig.DisableStateCache)
	if err != nil {
		return err
	}
	//注册txExistdb
	if storeConfig.TxExistDbConfig != nil {
		err = m.registerDbHandle(chainId, storeConfig.TxExistDbConfig, StoreTxExistDbDir,
			DBName_TxExistDB, storeConfig.DbPrefix, false)
		if err != nil {
			return err
		}
	}
	//注册historydb
	if !storeConfig.DisableHistoryDB {
		err = m.registerDbHandle(chainId, &storeConfig.HistoryDbConfig.DbConfig, StoreHistoryDBDir,
			DBName_HistoryDB, storeConfig.DbPrefix, false)
		if err != nil {
			return err
		}
	}
	//注册resultdb
	if !storeConfig.DisableResultDB {
		err = m.registerDbHandle(chainId, storeConfig.ResultDbConfig, StoreResultDBDir,
			DBName_ResultDB, storeConfig.DbPrefix, false)
		if err != nil {
			return err
		}
	}
	//注册contractdb
	if !storeConfig.DisableContractEventDB {
		err = m.registerDbHandle(chainId, storeConfig.ContractEventDbConfig, StoreEventLogDBDir,
			DBName_EventDB, storeConfig.DbPrefix, false)
		if err != nil {
			return err
		}
	}
	//注册一个默认的db
	return m.registerDbHandle(chainId, storeConfig.GetDefaultDBConfig(), StoreLocalDBDir,
		DBName_LocalDB, storeConfig.DbPrefix, false)
}

// registerBlockDB
// @Description: 注册blockdb
// @receiver m
// @param chainId
// @param storeConfig
// @return error
func (m *Factory) registerBlockDB(chainId string, storeConfig *conf.StorageConfig) error {
	var (
		err     error
		blockDB blockdb.BlockDB
	)
	dbPrefix := storeConfig.DbPrefix
	// 处理kvdb
	if storeConfig.BlockDbConfig.IsKVDB() {
		// 处理bfdb 和非bfdb的情况
		if storeConfig.DisableBlockFileDb {
			if err = m.ioc.Register(blockkvdb.NewBlockKvDB,
				container.DependsOn(map[int]string{1: DBName_BlockDB}),
				container.Parameters(map[int]interface{}{0: chainId, 3: storeConfig}),
				container.Interface(&blockDB)); err != nil {
				return err
			}
		} else {
			if err = m.ioc.Register(blockfiledb.NewBlockFileDB,
				container.DependsOn(map[int]string{1: DBName_BlockDB}),
				container.Parameters(map[int]interface{}{0: chainId, 3: storeConfig}),
				container.Interface(&blockDB)); err != nil {
				return err
			}
		}
	} else {
		//处理 sqldb
		if err = m.ioc.Register(blocksqldb.NewBlockSqlDB,
			container.DependsOn(map[int]string{1: DBName_BlockDB}),
			container.Parameters(map[int]interface{}{0: getDbName(dbPrefix, DBName_BlockDB, chainId)}),
			container.Interface(&blockDB)); err != nil {
			return err
		}
	}
	return nil
}

// registerStateDB
// @Description:  注册statedb
// @receiver m
// @param chainId
// @param storeConfig
// @return error
func (m *Factory) registerStateDB(chainId string, storeConfig *conf.StorageConfig) error {
	var (
		err     error
		stateDB statedb.StateDB
	)
	//处理kvdb
	if storeConfig.StateDbConfig.IsKVDB() {
		if err = m.ioc.Register(statekvdb.NewStateKvDB, container.DependsOn(map[int]string{1: DBName_StateDB}),
			container.Parameters(map[int]interface{}{0: chainId, 3: storeConfig}),
			container.Interface(&stateDB)); err != nil {
			return err
		}
	} else {
		//处理非sqldb
		newDbFunc := func(dbName string) (protocol.SqlDBHandle, error) {
			var db protocol.SqlDBHandle
			err = m.ioc.Resolve(&db, container.ResolveName(DBName_StateDB),
				container.Arguments(map[int]interface{}{1: dbName}))
			return db, err
		}
		//配置默认连接池连接数
		connPoolSize := 90
		if maxConnSize, ok := storeConfig.StateDbConfig.SqlDbConfig["max_open_conns"]; ok {
			connPoolSize, _ = maxConnSize.(int)
		}
		if err = m.ioc.Register(statesqldb.NewStateSqlDB, container.DependsOn(map[int]string{2: DBName_StateDB}),
			container.Parameters(map[int]interface{}{0: storeConfig.DbPrefix, 1: chainId, 3: newDbFunc, 5: connPoolSize}),
			container.Interface(&stateDB)); err != nil {
			return err
		}
	}
	return nil
}

// registerHistoryDB
// @Description: 注册historydb
// @receiver m
// @param chainId
// @param storeConfig
// @return error
func (m *Factory) registerHistoryDB(chainId string, storeConfig *conf.StorageConfig) error {
	var (
		err       error
		historyDB historydb.HistoryDB
	)
	//注册kvdb
	if storeConfig.HistoryDbConfig.IsKVDB() {
		if err = m.ioc.Register(historykvdb.NewHistoryKvDB,
			container.DependsOn(map[int]string{2: DBName_HistoryDB}),
			container.Parameters(map[int]interface{}{0: chainId, 1: storeConfig.HistoryDbConfig}),
			container.Interface(&historyDB)); err != nil {
			return err
		}
	} else {
		//注册非kvdb
		if err = m.ioc.Register(historysqldb.NewHistorySqlDB,
			container.DependsOn(map[int]string{2: DBName_HistoryDB}),
			container.Parameters(map[int]interface{}{
				0: getDbName(storeConfig.DbPrefix, DBName_HistoryDB, chainId),
				1: storeConfig.HistoryDbConfig}),
			container.Interface(&historyDB)); err != nil {
			return err
		}
	}
	return nil
}

// registerResultDB
// @Description: 注册resultdb
// @receiver m
// @param chainId
// @param storeConfig
// @return error
func (m *Factory) registerResultDB(chainId string, storeConfig *conf.StorageConfig) error {
	var (
		err      error
		resultDB resultdb.ResultDB
	)
	//注册kvdb
	if storeConfig.ResultDbConfig.IsKVDB() {
		if storeConfig.DisableBlockFileDb {
			err = m.ioc.Register(resultkvdb.NewResultKvDB, container.DependsOn(map[int]string{1: DBName_ResultDB}),
				container.Parameters(map[int]interface{}{0: chainId, 3: storeConfig}),
				container.Interface(&resultDB))
			if err != nil {
				return err
			}
		} else {
			err = m.ioc.Register(resultfiledb.NewResultFileDB, container.DependsOn(map[int]string{1: DBName_ResultDB}),
				container.Parameters(map[int]interface{}{0: chainId, 3: storeConfig}),
				container.Interface(&resultDB))
			if err != nil {
				return err
			}
		}
	} else {
		//注册非kvdb
		if err = m.ioc.Register(resultsqldb.NewResultSqlDB, container.DependsOn(map[int]string{1: DBName_ResultDB}),
			container.Parameters(map[int]interface{}{0: getDbName(storeConfig.DbPrefix, DBName_ResultDB, chainId)}),
			container.Interface(&resultDB)); err != nil {
			return err
		}
	}
	return nil
}

// registerTxExistDB
// @Description: 注册txExistdb
// @receiver m
// @param chainId
// @param storeConfig
// @return error
func (m *Factory) registerTxExistDB(chainId string, storeConfig *conf.StorageConfig) error {
	if storeConfig.TxExistDbConfig == nil {
		//如果没有配置TxExistDB，那么就使用BlockDB来判断TxExist
		//WrapBlockDB2TxExistDB 中包含了 bigcache 和db
		err := m.ioc.Register(WrapBlockDB2TxExistDB)
		if err != nil {
			return err
		}
	} else if storeConfig.TxExistDbConfig.IsKVDB() {
		var txExistDb txexistdb.TxExistDB
		err := m.ioc.Register(txexistkvdb.NewTxExistKvDB, container.DependsOn(map[int]string{1: DBName_TxExistDB}),
			container.Parameters(map[int]interface{}{0: chainId}),
			container.Interface(&txExistDb))
		if err != nil {
			return err
		}
	}
	return nil
}

// registerBusinessDB
// @Description: 注册数据库
// @receiver m
// @param chainId
// @param storeConfig
// @return error
func (m *Factory) registerBusinessDB(chainId string, storeConfig *conf.StorageConfig) error {
	if err := m.registerBlockDB(chainId, storeConfig); err != nil {
		return err
	}
	if err := m.registerStateDB(chainId, storeConfig); err != nil {
		return err
	}
	dbPrefix := storeConfig.DbPrefix

	if !storeConfig.DisableHistoryDB {
		if err := m.registerHistoryDB(chainId, storeConfig); err != nil {
			return err
		}
	}
	if !storeConfig.DisableResultDB {
		if err := m.registerResultDB(chainId, storeConfig); err != nil {
			return err
		}
	}
	if !storeConfig.DisableContractEventDB {
		//event db
		var eventDB contracteventdb.ContractEventDB
		err := m.ioc.Register(eventsqldb.NewContractEventDB, container.DependsOn(map[int]string{1: DBName_EventDB}),
			container.Parameters(map[int]interface{}{0: getDbName(dbPrefix, DBName_EventDB, chainId)}),
			container.Interface(&eventDB))
		if err != nil {
			return err
		}
	}
	//register TxExistDB
	return m.registerTxExistDB(chainId, storeConfig)
}

// registerFileOrLogDB
// @Description: 注册bfdb或者wal
// @receiver m
// @param chainId
// @param storeConfig
// @param logger
// @return error
func (m *Factory) registerFileOrLogDB(chainId string, storeConfig *conf.StorageConfig, logger protocol.Logger) error {
	var (
		err    error
		bfdb   *blockfiledb.BlockFile
		walLog *wal.Log
	)
	//处理 bfdb，完成注册
	if !storeConfig.DisableBlockFileDb {
		opts := blockfiledb.DefaultOptions
		opts.NoCopy = true
		opts.NoSync = storeConfig.LogDBSegmentAsync
		if storeConfig.LogDBSegmentSize > 64 { // LogDBSegmentSize default is 64MB
			opts.SegmentSize = storeConfig.LogDBSegmentSize * 1024 * 1024
		}
		if storeConfig.DisableLogDBMmap {
			opts.UseMmap = false
		}

		bfdbPath := filepath.Join(storeConfig.StorePath, chainId, blockFilePath)
		bfdb, err = blockfiledb.Open(bfdbPath, opts, logger)
		if err != nil {
			panic(fmt.Sprintf("open block file db failed, path:%s, error:%s", bfdbPath, err))
		}
		return m.ioc.Register(func() binlog.BinLogger { return bfdb })
	}
	//处理 wal，完成注册
	opts := wal.DefaultOptions
	opts.NoCopy = true
	opts.NoSync = storeConfig.LogDBSegmentAsync
	if storeConfig.LogDBSegmentSize > 0 {
		// LogDBSegmentSize default is 20MB
		opts.SegmentSize = storeConfig.LogDBSegmentSize * 1024 * 1024
	}

	walPath := filepath.Join(storeConfig.StorePath, chainId, walLogPath)
	walLog, err = wal.Open(walPath, opts)
	if err != nil {
		panic(fmt.Sprintf("open wal log failed, path:%s, error:%s", walPath, err))
	}
	return m.ioc.Register(func() *wal.Log { return walLog })
}

// NewBlockDB 创建blockdb
// @Description:
// @receiver m
// @param chainId
// @param storeConfig
// @param logger
// @param p11Handle
// @return blockdb.BlockDB
// @return error
func (m *Factory) NewBlockDB(chainId string, storeConfig *conf.StorageConfig,
	logger protocol.Logger, p11Handle *pkcs11.P11Handle) (blockdb.BlockDB, error) {
	m.ioc = container.NewContainer()
	err := m.register(chainId, storeConfig, logger, p11Handle)
	if err != nil {
		return nil, err
	}
	var db blockdb.BlockDB
	err = m.ioc.Resolve(&db)
	return db, err
}

// noTxExistDB
// @Description:
type noTxExistDB struct {
	db blockdb.BlockDB
}

// InitGenesis
// @Description: 初始化创世块
// @receiver n
// @param genesisBlock
// @return error
func (n noTxExistDB) InitGenesis(genesisBlock *serialization.BlockWithSerializedInfo) error {
	return nil
}

// CommitBlock
// @Description: 提交创世块
// @receiver n
// @param blockWithRWSet
// @param isCache
// @return error
func (n noTxExistDB) CommitBlock(blockWithRWSet *serialization.BlockWithSerializedInfo, isCache bool) error {
	return nil
}

// GetLastSavepoint
// @Description: 保存最后写
// @receiver n
// @return uint64
// @return error
func (n noTxExistDB) GetLastSavepoint() (uint64, error) {
	return n.db.GetLastSavepoint()
}

// TxExists
// @Description: 判断交易是否存在
// @receiver n
// @param txId
// @return bool
// @return error
func (n noTxExistDB) TxExists(txId string) (bool, error) {
	return n.db.TxExists(txId)
}

// Close
// @Description: 关闭db
// @receiver n
func (n noTxExistDB) Close() {

}

// WrapBlockDB2TxExistDB 创建空db
// @Description:
// @param db
// @param log
// @return txexistdb.TxExistDB
func WrapBlockDB2TxExistDB(db blockdb.BlockDB, log protocol.Logger) txexistdb.TxExistDB {
	log.Info("no TxExistDB config, use BlockDB to replace TxExistDB")
	return &noTxExistDB{db: db}
}
