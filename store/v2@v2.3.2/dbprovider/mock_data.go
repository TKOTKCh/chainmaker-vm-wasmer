/*
 * Copyright (C) BABEC. All rights reserved.
 * Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
 *
 * SPDX-License-Identifier: Apache-2.0
 */

package dbprovider

import (
	"bytes"
	"fmt"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"time"

	"chainmaker.org/chainmaker/store/v2/conf"
)

var keyLen int

const hitKeyFormat = "%016d+"
const missingKeyFormat = "%016d-"

// init
// @Description:
func init() {
	var b bytes.Buffer
	keyLen, _ = fmt.Fprintf(&b, hitKeyFormat, math.MaxInt32)
	b.Reset()
	missingKeyLen, _ := fmt.Fprintf(&b, missingKeyFormat, math.MaxInt32)
	if keyLen != missingKeyLen {
		panic("len(key) != len(missingKey)")
	}
}

// keyGenerator
// @Description:
type keyGenerator interface {
	NKey() int
	Key(i int) []byte
}

// EntryGenerator interface
// @Description:
type EntryGenerator interface {
	keyGenerator
	Value(i int) []byte
}

// pairedEntryGenerator
// @Description:
type pairedEntryGenerator struct {
	keyGenerator
	randomValueGenerator
}

// randomValueGenerator
// @Description:
type randomValueGenerator struct {
	b []byte
	k int
}

// Value
// @Description: 返回一个随机数
// @receiver g
// @param i
// @return []byte
func (g *randomValueGenerator) Value(i int) []byte {
	i = (i * g.k) % len(g.b)
	return g.b[i : i+g.k]
}

// predefinedKeyGenerator
// @Description:
type predefinedKeyGenerator struct {
	keys [][]byte
}

// NKey
// @Description: 返回keys长度，元素个数
// @receiver g
// @return int
func (g *predefinedKeyGenerator) NKey() int {
	return len(g.keys)
}

// Key
// @Description: 返回第i个元素
// @receiver g
// @param i
// @return []byte
func (g *predefinedKeyGenerator) Key(i int) []byte {
	return g.keys[i]
}

// newFullRandomKeys
// @Description: 创建一些随机key
// @param n
// @param start
// @param format
// @return [][]byte
func newFullRandomKeys(n int, start int, format string) [][]byte {
	keys := newSequentialKeys(n, start, format)
	r := rand.New(rand.NewSource(time.Now().Unix())) //nolint: gosec
	for i := 0; i < n; i++ {
		j := r.Intn(n)
		keys[i], keys[j] = keys[j], keys[i]
	}
	return keys
}

// newSequentialKeys
// @Description: 按照顺序产生一些keys
// @param n
// @param start
// @param keyFormat
// @return [][]byte
//
func newSequentialKeys(n int, start int, keyFormat string) [][]byte {
	keys := make([][]byte, n)
	buffer := make([]byte, n*keyLen)
	for i := 0; i < n; i++ {
		begin, end := i*keyLen, (i+1)*keyLen
		key := buffer[begin:begin:end]
		n, _ := fmt.Fprintf(bytes.NewBuffer(key), keyFormat, start+i)
		if n != keyLen {
			panic("n != keyLen")
		}
		keys[i] = buffer[begin:end:end]
	}
	return keys
}

// newFullRandomKeyGenerator
// @Description:
// @param start
// @param n
// @return keyGenerator
func newFullRandomKeyGenerator(start, n int) keyGenerator {
	return &predefinedKeyGenerator{keys: newFullRandomKeys(n, start, hitKeyFormat)}
}

// makeRandomValueGenerator
// @Description:
// @param r
// @param ratio
// @param valueSize
// @return randomValueGenerator
func makeRandomValueGenerator(r *rand.Rand, ratio float64, valueSize int) randomValueGenerator {
	b := compressibleBytes(r, ratio, valueSize)
	max := maxInt(valueSize, 1024*1024)
	for len(b) < max {
		b = append(b, compressibleBytes(r, ratio, valueSize)...)
	}
	return randomValueGenerator{b: b, k: valueSize}
}

// compressibleBytes
// @Description:
// @param r
// @param ratio
// @param n
// @return []byte
func compressibleBytes(r *rand.Rand, ratio float64, n int) []byte { //nolint: gosec
	m := maxInt(int(float64(n)*ratio), 1)
	p := randomBytes(r, m)
	b := make([]byte, 0, n+n%m)
	for len(b) < n {
		b = append(b, p...)
	}
	return b[:n]
}

// maxInt
// @Description: 比较2个数，返回最大的
// @param a
// @param b
// @return int
func maxInt(a int, b int) int {
	if a >= b {
		return a
	}
	return b
}

//随机产生一个[]byte
func randomBytes(r *rand.Rand, n int) []byte { //nolint: gosec
	b := make([]byte, n)
	for i := 0; i < n; i++ {
		b[i] = ' ' + byte(r.Intn('~'-' '+1))
	}
	return b
}

// NewFullRandomEntryGenerator 创建迭代器
// @Description:
// @param start
// @param n
// @return EntryGenerator
func NewFullRandomEntryGenerator(start, n int) EntryGenerator {
	r := rand.New(rand.NewSource(time.Now().Unix())) //nolint: gosec
	return &pairedEntryGenerator{
		keyGenerator:         newFullRandomKeyGenerator(start, n),
		randomValueGenerator: makeRandomValueGenerator(r, 0.5, 100),
	}
}

// GetMockDBConfig 返回一个存储配置对象
// @Description:
// @param path
// @return *conf.StorageConfig
func GetMockDBConfig(path string) *conf.StorageConfig {
	config := &conf.StorageConfig{}
	if path == "" {
		path = filepath.Join(os.TempDir(), fmt.Sprintf("%d", time.Now().Nanosecond()))
	}
	config.StorePath = path

	lvlConfig := make(map[string]interface{})
	lvlConfig["store_path"] = path
	dbConfig := &conf.DbConfig{
		Provider:      "leveldb",
		LevelDbConfig: lvlConfig,
	}
	config.BlockDbConfig = dbConfig
	config.StateDbConfig = dbConfig
	config.HistoryDbConfig = conf.NewHistoryDbConfig(dbConfig)
	config.ResultDbConfig = dbConfig
	config.DisableContractEventDB = true
	return config
}
