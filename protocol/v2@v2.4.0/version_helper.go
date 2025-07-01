/*
 * Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
 *
 * SPDX-License-Identifier: Apache-2.0

 * Due to historical reasons, compatibility was not initially considered. Therefore, will show the correspondence
 * between the blockVersion and the open-source chainmaker-go version.
 * blockVersion: 20 -> chainmaker-go: v2.0.0/v2.1.0
 * blockVersion: 220 -> chainmaker-go: v2.2.0_alpha
 * blockVersion: 2201 -> chainmaker-go: v2.2.0
 * blockVersion: 2210 -> chainmaker-go: v2.2.1
 * blockVersion: 2220 -> chainmaker-go: v2.2.2
 * blockVersion: 2300 -> chainmaker-go: v2.3.0_alpha
 * blockVersion: 2301 -> chainmaker-go: v2.3.0
 * blockVersion: 2030100 -> chainmaker-go: v2.3.1
 * blockVersion: 2030200 -> chainmaker-go: v2.3.2
 * blockVersion: 2030300 -> chainmaker-go: v2.3.3
 * blockVersion: 2040000 -> chainmaker-go: v2.4.0_alpha
 * blockVersion: 2040100 -> chainmaker-go: v2.4.0
 * 2030100 ->  v2.3.1
 * 2030200 ->  v2.3.2
 * 3000000 -> v3.0.0_alpha
 * 3000001 -> v3.0.0_beta // maybe
 * 3000002 -> v3.0.0
 * before 230：blockVersion is changed to 4 digits (uint), the first 3 digits correspond to the 3 digits
 * released by chainmaker,
 * and the last digit corresponds to the internal iteration version (alpha, beta, gamma, delta).
 * after 231: blockVersion is changed to 7/8 digits (uint), where each two digits represent a version number (
 * the first 0 is omitted),
 * and the last two digits correspond to the internal iteration version, such as (alpha[01], beta[02], gamma, delta)
 *
 */

package protocol

import (
	"fmt"
	"strconv"
	"strings"
)

// DefaultBlockVersion default blockHeader.Version value
const DefaultBlockVersion = uint32(2040000) // default version of chain

const (
	// BlockVersion20 - BlockVersion2220 相关版本
	BlockVersion20   = uint32(20)
	BlockVersion220  = uint32(220)
	BlockVersion2201 = uint32(2201)
	BlockVersion2210 = uint32(2210)
	BlockVersion2211 = uint32(2211)
	BlockVersion2212 = uint32(2212)
	BlockVersion2216 = uint32(2216)
	BlockVersion2217 = uint32(2217)
	BlockVersion2218 = uint32(2218)
	BlockVersion2220 = uint32(2220)

	// BlockVersion230 - BlockVersion239 v2.3.x相关版本
	BlockVersion230  = uint32(2300)
	BlockVersion2301 = uint32(2301)
	BlockVersion231  = uint32(2030100)
	BlockVersion2311 = uint32(2030101)
	BlockVersion2312 = uint32(2030102)
	BlockVersion232  = uint32(2030200)
	BlockVersion2321 = uint32(2030201)
	BlockVersion233  = uint32(2030300)
	BlockVersion234  = uint32(2030400)
	BlockVersion235  = uint32(2030500)
	BlockVersion236  = uint32(2030600)
	BlockVersion2361 = uint32(2030601)
	BlockVersion2362 = uint32(2030602)
	BlockVersion237  = uint32(2030700)
	BlockVersion238  = uint32(2030800)
	BlockVersion239  = uint32(2030900)

	// BlockVersion240 - BlockVersion242 v2.4.x相关版本
	BlockVersion240 = uint32(2040000) // 该版本会对应发版的v2.4.0_alpha版本
	BlockVersion241 = uint32(2040100)
	BlockVersion242 = uint32(2040200)
	BlockVersion243 = uint32(2040300)
)

// ToVersionInt 将版本号转换为整数，仅支持v2.3.1之后版本，不支持alpha版本
func ToVersionInt(s string) (uint32, error) {
	if strings.Contains(s, "_") {
		return 0, fmt.Errorf("invalid version format, alpha or beta... version is not supported")
	}
	// 移除前缀 'v'
	version := strings.TrimPrefix(s, "v")
	// 分割版本号各部分
	parts := strings.Split(version, ".")
	// 检查版本号格式是否合法（最多4部分）
	if len(parts) < 3 || len(parts) > 4 {
		return 0, fmt.Errorf("invalid version format, expected 3 or 4 parts")
	}
	// 定义一个切片来存储各部分的整数值
	var nums [4]int
	// 解析各部分数字，并确保每部分至少有两位数
	for i, part := range parts {
		num, err := strconv.Atoi(part)
		if err != nil {
			return 0, fmt.Errorf("non-numeric version part: %s", part)
		}
		nums[i] = num
	}
	// 构建最终的数字表示
	// 使用 Sprintf 来确保每个部分至少有两位数，不足的用前导零补齐
	formatted := fmt.Sprintf("%02d%02d%02d%02d", nums[0], nums[1], nums[2], nums[3])
	// 将格式化后的字符串转换为整数
	number, err := strconv.Atoi(formatted)
	if err != nil {
		return 0, fmt.Errorf("failed to convert formatted string to integer: %v", err)
	}
	return uint32(number), nil
}
