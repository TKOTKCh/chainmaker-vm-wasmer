/*
 * Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
 *
 * SPDX-License-Identifier: Apache-2.0
 *
 */
package store

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"chainmaker.org/chainmaker/protocol/v2/test"
	"chainmaker.org/chainmaker/store/v2/conf"
	"github.com/stretchr/testify/assert"
)

func TestFactory_NewStore(t *testing.T) {
	config := &conf.StorageConfig{}
	cJson := `{
"StorePath":"../data/org1/ledgerData1",
"DbPrefix":"","WriteBufferSize":0,
"BloomFilterBits":0,
"BlockWriteBufferSize":0,
"DisableHistoryDB":false,
"DisableResultDB":false,
"DisableContractEventDB":true,
"LogDBWriteAsync":false,
"BlockDbConfig":{
	"Provider":"leveldb",
	"LevelDbConfig":{
		"store_path":"../data/org1/blocks"
	},
	"BadgerDbConfig":null,
	"SqlDbConfig":null
},
"TxExistDbConfig":{
	"Provider":"leveldb",
	"LevelDbConfig":{
		"store_path":"../data/org1/txexist"
	},
	"BadgerDbConfig":null,
	"SqlDbConfig":null
},
"StateDbConfig":{
	"Provider":"leveldb",
	"LevelDbConfig":{"store_path":"../data/org1/statedb"},
	"BadgerDbConfig":null,"SqlDbConfig":null
},
"HistoryDbConfig":{
	"Provider":"leveldb",
	"LevelDbConfig":{"store_path":"../data/org1/history"},
	"BadgerDbConfig":null,"SqlDbConfig":null
},
"ResultDbConfig":{
	"Provider":"leveldb",
	"LevelDbConfig":{"store_path":"../data/org1/result"},
	"BadgerDbConfig":null,
	"SqlDbConfig":null
},
"ContractEventDbConfig":{
	"Provider":"sql",
	"LevelDbConfig":null,
	"BadgerDbConfig":null,
	"SqlDbConfig":{
		"dsn":"root:password@tcp(127.0.0.1:3306)/",
		"sqldb_type":"mysql"
	}
},
"UnArchiveBlockHeight":0,
"Encryptor":"sm4",
"EncryptKey":"1234567890123456"
}`
	//config.DisableBigFilter = true
	err := json.Unmarshal([]byte(cJson), config)

	if err != nil {
		t.Log(err)
	}
	var f = NewFactory()
	bs, err := f.NewStore("chainId", config, &test.GoLogger{}, nil)
	assert.Nil(t, err)
	assert.NotNil(t, bs)
	t.Logf("%#v", bs)
}
func TestReadEncryptKey(t *testing.T) {
	password := "P@ssw0rd"
	assert.EqualValues(t, []byte(password), readEncryptKey(password))
	hexpwd := "0x" + hex.EncodeToString([]byte(password))
	t.Logf("hex password:%s", hexpwd)
	assert.EqualValues(t, []byte(password), readEncryptKey(hexpwd))

	path := filepath.Join(os.TempDir(), fmt.Sprintf("%d", time.Now().UnixNano()))
	t.Logf("path:%s", path)
	_ = ioutil.WriteFile(path, []byte(password), os.ModePerm)
	assert.EqualValues(t, []byte(password), readEncryptKey(path))

	noFilePath := filepath.Join(os.TempDir(), fmt.Sprintf("%d", time.Now().UnixNano()))
	t.Logf("path:%s", noFilePath)
	assert.EqualValues(t, []byte(noFilePath), readEncryptKey(noFilePath))

}
