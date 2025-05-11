/*
 * Copyright (C) BABEC. All rights reserved.
 * Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
 *
 * SPDX-License-Identifier: Apache-2.0
 */

package conf

import (
	"strings"
	"time"

	"github.com/mitchellh/mapstructure"
)

// nolint
const (
	CommonWriteBlockType     = 0
	QuickWriteBlockType      = 1
	DbconfigProviderSql      = "sql"
	DbconfigProviderSqlKV    = "sqlkv" //虽然是用SQL数据库，但是当KV数据库使
	DbconfigProviderLeveldb  = "leveldb"
	DbconfigProviderMemdb    = "memdb" //内存数据库，只用于测试
	DbconfigProviderBadgerdb = "badgerdb"
	DbconfigProviderTikvdb   = "tikvdb"
)

// StorageConfig engine config
//  @Description:
type StorageConfig struct {
	//默认的Leveldb配置，如果每个DB有不同的设置，可以在自己的DB中进行设置
	StorePath string `mapstructure:"store_path"`
	//写入区块数据存储方式，0-wal和db双写，1-只写wal，2-只写db
	//BlockDbType          uint8  `mapstructure:"block_db_type"`
	DbPrefix             string `mapstructure:"db_prefix"`
	WriteBufferSize      int    `mapstructure:"write_buffer_size"`
	BloomFilterBits      int    `mapstructure:"bloom_filter_bits"`
	BlockWriteBufferSize int    `mapstructure:"block_write_buffer_size"`
	//数据库模式：light只存区块头,normal存储区块头和交易以及生成的State,full存储了区块头、交易、状态和交易收据（读写集、日志等）
	//Mode string `mapstructure:"mode"`
	DisableBlockFileDb     bool             `mapstructure:"disable_block_file_db"`
	DisableHistoryDB       bool             `mapstructure:"disable_historydb"`
	DisableResultDB        bool             `mapstructure:"disable_resultdb"`
	DisableContractEventDB bool             `mapstructure:"disable_contract_eventdb"`
	DisableStateCache      bool             `mapstructure:"disable_state_cache"`
	EnableBigFilter        bool             `mapstructure:"enable_bigfilter"`
	LogDBSegmentAsync      bool             `mapstructure:"logdb_segment_async"`
	LogDBSegmentSize       int              `mapstructure:"logdb_segment_size"`
	DisableLogDBMmap       bool             `mapstructure:"disable_logdb_mmap"`
	BlockDbConfig          *DbConfig        `mapstructure:"blockdb_config"`
	StateDbConfig          *DbConfig        `mapstructure:"statedb_config"`
	HistoryDbConfig        *HistoryDbConfig `mapstructure:"historydb_config"`
	ResultDbConfig         *DbConfig        `mapstructure:"resultdb_config"`
	ContractEventDbConfig  *DbConfig        `mapstructure:"contract_eventdb_config"`
	TxExistDbConfig        *DbConfig        `mapstructure:"txexistdb_config"`
	UnArchiveBlockHeight   uint64           `mapstructure:"unarchive_block_height"`
	Async                  bool             `mapstructure:"is_async"`
	WriteBatchSize         uint64           `mapstructure:"write_batch_size"`
	Encryptor              string           `mapstructure:"encryptor"` //SM4  AES
	EncryptKey             string           `mapstructure:"encrypt_key"`

	WriteBlockType             int              `mapstructure:"write_block_type"` //0 普通写, 1 高速写
	StateCache                 *CacheConfig     `mapstructure:"state_cache_config"`
	BigFilter                  *BigFilterConfig `mapstructure:"bigfilter_config"`
	RollingWindowCacheCapacity uint64           `mapstructure:"rolling_window_cache_capacity"`
	EnableRWC                  bool             `mapstructure:"enable_rwc"`
	//记录慢操作的日志，默认是0，表示不记录，如果>0则是毫秒数，大于该毫秒数的读操作被记录
	SlowLog int64 `mapstructure:"slow_log"`
}

// NewStorageConfig new storage config from map structure
//  @Description:
//  @param cfg
//  @return *StorageConfig
//  @return error
func NewStorageConfig(cfg map[string]interface{}) (*StorageConfig, error) {
	sconfig := &StorageConfig{
		DisableBlockFileDb: true, //为了保持v2.1的兼容，默认BlockFileDB是不开启的，必须通过配置文件开启
		WriteBatchSize:     100,  //默认按一个批次100条记录算
	}
	if cfg == nil {
		return sconfig, nil
	}
	err := mapstructure.Decode(cfg, sconfig)
	if err != nil {
		return nil, err
	}
	return sconfig, nil
}

//  setDefault do nothing
//  @Description:
//  @receiver config
func (config *StorageConfig) setDefault() {
	//if config.DbPrefix != "" {
	//	if config.BlockDbConfig != nil && config.BlockDbConfig.SqlDbConfig != nil &&
	//		config.BlockDbConfig.SqlDbConfig.DbPrefix == "" {
	//		config.BlockDbConfig.SqlDbConfig.DbPrefix = config.DbPrefix
	//	}
	//	if config.StateDbConfig != nil && config.StateDbConfig.SqlDbConfig != nil &&
	//		config.StateDbConfig.SqlDbConfig.DbPrefix == "" {
	//		config.StateDbConfig.SqlDbConfig.DbPrefix = config.DbPrefix
	//	}
	//	if config.HistoryDbConfig != nil && config.HistoryDbConfig.SqlDbConfig != nil &&
	//		config.HistoryDbConfig.SqlDbConfig.DbPrefix == "" {
	//		config.HistoryDbConfig.SqlDbConfig.DbPrefix = config.DbPrefix
	//	}
	//	if config.ResultDbConfig != nil && config.ResultDbConfig.SqlDbConfig != nil &&
	//		config.ResultDbConfig.SqlDbConfig.DbPrefix == "" {
	//		config.ResultDbConfig.SqlDbConfig.DbPrefix = config.DbPrefix
	//	}
	//	if config.ContractEventDbConfig != nil && config.ContractEventDbConfig.SqlDbConfig != nil &&
	//		config.ContractEventDbConfig.SqlDbConfig.DbPrefix == "" {
	//		config.ContractEventDbConfig.SqlDbConfig.DbPrefix = config.DbPrefix
	//	}
	//}
}

// GetBlockDbConfig 返回blockdb config
//  @Description:
//  @receiver config
//  @return *DbConfig
func (config *StorageConfig) GetBlockDbConfig() *DbConfig {
	if config.BlockDbConfig == nil {
		return config.GetDefaultDBConfig()
	}
	config.setDefault()
	return config.BlockDbConfig
}

// GetStateDbConfig 返回statedb config
//  @Description:
//  @receiver config
//  @return *DbConfig
func (config *StorageConfig) GetStateDbConfig() *DbConfig {
	if config.StateDbConfig == nil {
		return config.GetDefaultDBConfig()
	}
	config.setDefault()
	return config.StateDbConfig
}

// GetHistoryDbConfig 返回historydb config
//  @Description:
//  @receiver config
//  @return *HistoryDbConfig
func (config *StorageConfig) GetHistoryDbConfig() *HistoryDbConfig {
	if config.HistoryDbConfig == nil {
		return NewHistoryDbConfig(config.GetDefaultDBConfig())
	}
	config.setDefault()
	return config.HistoryDbConfig
}

// GetResultDbConfig 返回resultdb config
//  @Description:
//  @receiver config
//  @return *DbConfig
func (config *StorageConfig) GetResultDbConfig() *DbConfig {
	if config.ResultDbConfig == nil {
		return config.GetDefaultDBConfig()
	}
	config.setDefault()
	return config.ResultDbConfig
}

// GetContractEventDbConfig 返回contractdb config
//  @Description:
//  @receiver config
//  @return *DbConfig
func (config *StorageConfig) GetContractEventDbConfig() *DbConfig {
	if config.ContractEventDbConfig == nil {
		return config.GetDefaultDBConfig()
	}
	config.setDefault()
	return config.ContractEventDbConfig
}

// GetTxExistDbConfig 返回txExistdb config
//  @Description:
//  @receiver config
//  @return *DbConfig
func (config *StorageConfig) GetTxExistDbConfig() *DbConfig {
	if config.TxExistDbConfig == nil {
		return config.GetDefaultDBConfig()
	}
	config.setDefault()
	return config.TxExistDbConfig
}

// GetDefaultDBConfig 返回一个默认config
//  @Description:
//  @receiver config
//  @return *DbConfig
func (config *StorageConfig) GetDefaultDBConfig() *DbConfig {
	lconfig := make(map[string]interface{})
	lconfig["store_path"] = config.StorePath
	//lconfig := &LevelDbConfig{
	//	StorePath:            config.StorePath,
	//	WriteBufferSize:      config.WriteBufferSize,
	//	BloomFilterBits:      config.BloomFilterBits,
	//	BlockWriteBufferSize: config.WriteBufferSize,
	//}

	//bconfig := &BadgerDbConfig{
	//	StorePath: config.StorePath,
	//}

	return &DbConfig{
		Provider:      "leveldb",
		LevelDbConfig: lconfig,
		//BadgerDbConfig: bconfig,
	}
}

// GetActiveDBCount 根据配置的DisableDB的情况，确定当前配置活跃的数据库数量
//  @Description:
//  @receiver config
//  @return int
func (config *StorageConfig) GetActiveDBCount() int {
	count := 5
	if config.DisableContractEventDB {
		count--
	}
	if config.DisableHistoryDB {
		count--
	}
	if config.DisableResultDB {
		count--
	}
	return count
}

// DbConfig encapsulate db config
//  @Description:
type DbConfig struct {
	//leveldb,badgerdb,tikvdb,sql
	Provider       string                 `mapstructure:"provider"`
	LevelDbConfig  map[string]interface{} `mapstructure:"leveldb_config"`
	BadgerDbConfig map[string]interface{} `mapstructure:"badgerdb_config"`
	TikvDbConfig   map[string]interface{} `mapstructure:"tikvdb_config"`
	SqlDbConfig    map[string]interface{} `mapstructure:"sqldb_config"`
}

// HistoryDbConfig config history db
//  @Description:
type HistoryDbConfig struct {
	DbConfig               `mapstructure:",squash"`
	DisableKeyHistory      bool `mapstructure:"disable_key_history"`
	DisableContractHistory bool `mapstructure:"disable_contract_history"`
	DisableAccountHistory  bool `mapstructure:"disable_account_history"`
}

// NewHistoryDbConfig  创建historydb config
//  @Description:
//  @param config
//  @return *HistoryDbConfig
func NewHistoryDbConfig(config *DbConfig) *HistoryDbConfig {
	return &HistoryDbConfig{
		DbConfig:               *config,
		DisableKeyHistory:      false,
		DisableContractHistory: false,
		DisableAccountHistory:  false,
	}
}

// GetDbConfig get concrete db config according to type
//  @Description:
//  @receiver c
//  @return map[string]interface{}
func (c *DbConfig) GetDbConfig() map[string]interface{} {
	switch strings.ToLower(c.Provider) {
	case DbconfigProviderLeveldb:
		return c.LevelDbConfig
	case DbconfigProviderBadgerdb:
		return c.BadgerDbConfig
	case DbconfigProviderTikvdb:
		return c.TikvDbConfig
	case DbconfigProviderSql:
	case DbconfigProviderSqlKV:
		return c.SqlDbConfig
	default:
		return map[string]interface{}{}
	}
	return nil
}

// IsKVDB check db satisfy kv standard
//  @Description:
//  @receiver dbc
//  @return bool
func (dbc *DbConfig) IsKVDB() bool {
	return dbc.Provider == DbconfigProviderLeveldb ||
		dbc.Provider == DbconfigProviderTikvdb ||
		dbc.Provider == DbconfigProviderBadgerdb ||
		dbc.Provider == DbconfigProviderSqlKV
}

// IsSqlDB check db satisfy sql standard
//  @Description:
//  @receiver dbc
//  @return bool
func (dbc *DbConfig) IsSqlDB() bool {
	return dbc.Provider == DbconfigProviderSql || dbc.Provider == "sqldb" ||
		dbc.Provider == "mysql" || dbc.Provider == "rdbms" //兼容其他配置情况
}

// nolint
const (
	SqldbconfigSqldbtypeMysql  = "mysql"
	SqldbconfigSqldbtypeSqlite = "sqlite"
)

// CacheConfig cache config option
//  @Description:
type CacheConfig struct {
	//Cache提供方，包括：bigcache，redis，等
	Provider string `mapstructure:"provider"`
	// Time after which entry can be evicted
	LifeWindow time.Duration `mapstructure:"life_window"`
	// Interval between removing expired entries (clean up).
	// If set to <= 0 then no action is performed.
	//Setting to < 1 second is counterproductive — bigcache has a one second resolution.
	CleanWindow time.Duration `mapstructure:"clean_window"`
	// Max size of entry in bytes. Used only to calculate initial size for cache shards.
	MaxEntrySize int `mapstructure:"max_entry_size"`
	// Max size of cache.
	HardMaxCacheSize int `mapstructure:"hard_max_cache_size"`
}

// BigFilterConfig bigfilter 配置信息
//  @Description:
type BigFilterConfig struct {
	RedisHosts string  `mapstructure:"redis_hosts_port"`
	Pass       string  `mapstructure:"redis_password"`
	TxCapacity uint    `mapstructure:"tx_capacity"`
	FpRate     float64 `mapstructure:"fp_rate"`
}
