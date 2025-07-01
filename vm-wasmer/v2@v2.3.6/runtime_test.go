/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package wasmer

import (
	"bytes"
	"chainmaker.org/chainmaker/pb-go/v2/common"
	"chainmaker.org/chainmaker/protocol/v2"
	"encoding/hex"
	"fmt"
	"math/big"
	"math/rand"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

var (
	sdkConfigPaths       []string      // 被压测的cert模式链配置文件数组
	sdkPKConfigPaths     []string      // 被压测的public模式链配置文件数组
	SdkPWKConfigPaths    []string      // 被压测的permissionedWithKey模式链配置文件数组
	Clientslen           int           // 被压测的链个数
	ContractByteCodePath string        // 存证类合约智能合约路径
	txCounts             uint32    = 0 // 成功上链交易数
	timeStart            time.Time     // 压测开始时间
	timeEnd              time.Time     // 压测开始时间
	BlockStart           int64         // 压测开始区块
	Wg                   = sync.WaitGroup{}
	Model                string             // 身份认证模型
	ContractName         string             // 智能合约名称
	ContractType         string             // 智能合约类型
	RuntimeTypeString    string             // 智能合约语言类型, string类型
	RuntimeType          common.RuntimeType // 智能合约语言类型, common.RuntimeType类型
	contractMethod       string             // 智能合约被压测方法
	Params               string             // 智能合约被压测方法参数
	ThreadNum            int                // 单次并发进程数
	LoopNum              int                // 压测并发次数
	SleepTime            int                // 并发间隔,单位ms
	ClimbTime            int                // 爬坡时间,单位s
	AddOption            string
	RandomSeed           int64 // 生成tokenid的随机种子
	lastTokenId          = new(big.Int)
	mu                   sync.Mutex
)

// 生成随机长度地址（用于生成参数）
func randomHexString(length int) (string, error) {
	bytes := make([]byte, length/2)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// 生成指定长度的随机数字字符串（首位不为0），用于erc721的tokenid生成 （用于生成参数）
func RandomNumberString(length int) string {
	//加锁这一步对性能影响不大
	mu.Lock()         // 加锁确保并发安全
	defer mu.Unlock() // 解锁
	lastTokenId.Add(lastTokenId, big.NewInt(1))

	return lastTokenId.String()
}

// 处理参数 （用于生成参数）
func handleParams(paramString string) map[string]string {
	FunctionParametersMap := make(map[string]string)
	initParametersList := strings.Split(paramString, "||")
	for i := 0; i < len(initParametersList); i++ {
		index := strings.Index(initParametersList[i], ":")
		if index != -1 {
			firstPart := initParametersList[i][:index]
			secondPart := initParametersList[i][index+1:]
			FunctionParametersMap[firstPart] = secondPart
		}
	}
	return FunctionParametersMap
}

// RandParams 随机化参数 （用于生成参数）
func RandParams(FunctionParametersMap map[string]string) map[string][]byte {
	normalParams := make(map[string]string)
	params := make(map[string][]byte)
	// 获取带压测方法的参数列表
	set := FunctionParametersMap
	curTime := strconv.FormatInt(time.Now().Unix(), 10)
	for k, v := range set {
		value := v
		if strings.Contains(strings.ToLower(k), strings.ToLower("time")) || strings.Contains(strings.ToLower(k), strings.ToLower("timestamp")) {
			value = curTime
		}
		if strings.Contains(strings.ToLower(k), strings.ToLower("tokenId")) {
			value = RandomNumberString(len(value))
		}
		normalParams[k] = value
		params[k] = []byte(value)
	}

	return params
}

// 生成调用函数参数
func getMethodParams(ContractName, contractMethod string) string {
	var Params string
	if ContractName == "identity" {
		if contractMethod == "callerAddress" || contractMethod == "address" {
			Params = ""
		} else {
			Params = "address:"
			for i := 0; i < 100; i++ {
				s, _ := randomHexString(40)
				Params += s
				if i != 99 {
					Params += ","
				}
			}
		}
	}
	if ContractName == "erc721" {
		if contractMethod == "tokenURI" || contractMethod == "ownerOf" || contractMethod == "tokenMetadata" || contractMethod == "tokenLatestTxInfo" || contractMethod == "getApprove" {
			Params = "tokenId:111111111111111111111112"
		}
		if contractMethod == "balanceOf" || contractMethod == "accountTokens" {
			Params = "account:c0d8e4ce07a48081eff14a3016699b1c839c4375"
		}
		if contractMethod == "mint" {
			Params = "to:8acfaca5eeec9f6f7c23c4ffac969b86f27799b0||tokenId:111111111111111111111111||metadata:http://chainmaker.org.cn/"
		}
		//if contractMethod == "approve" {
		//	Params = "to:818fac1ac51525aeedf619a9a339b95854930159||tokenId:111111111111111111111111"
		//}
		if contractMethod == "setApprovalForAll2" {
			Params = "approvalFrom:8acfaca5eeec9f6f7c23c4ffac969b86f27799b0"
		}
		if contractMethod == "transferFrom" {
			Params = "from:8acfaca5eeec9f6f7c23c4ffac969b86f27799b0||to:818fac1ac51525aeedf619a9a339b95854930159||tokenId:11111111111111111111111||metadata:http://chainmaker.org.cn/"
		}

	}
	if ContractName == "enc_data" || ContractName == "enc_data_modify" { // 比如你的合约名字叫 enc_contract
		if contractMethod == "enc_data" {
			Params = `data_key:dataKey||data_value:dataValue||enc_key:encKey||authorized_person:-----BEGIN CERTIFICATE-----
MIICeDCCAh6gAwIBAgIDDmp3MAoGCCqGSM49BAMCMIGKMQswCQYDVQQGEwJDTjEQ
MA4GA1UECBMHQmVpamluZzEQMA4GA1UEBxMHQmVpamluZzEfMB0GA1UEChMWd3gt
b3JnMS5jaGFpbm1ha2VyLm9yZzESMBAGA1UECxMJcm9vdC1jZXJ0MSIwIAYDVQQD
ExljYS53eC1vcmcxLmNoYWlubWFrZXIub3JnMB4XDTI1MDQxODE1NDQyOVoXDTMw
MDQxNzE1NDQyOVowgZExCzAJBgNVBAYTAkNOMRAwDgYDVQQIEwdCZWlqaW5nMRAw
DgYDVQQHEwdCZWlqaW5nMR8wHQYDVQQKExZ3eC1vcmcxLmNoYWlubWFrZXIub3Jn
MQ8wDQYDVQQLEwZjb21tb24xLDAqBgNVBAMTI2NvbW1vbjEuc2lnbi53eC1vcmcx
LmNoYWlubWFrZXIub3JnMFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAEn4ZMa251
acwZkmZQ/HBWGyy1hMr40ChHJ29aNvlCp9xUBjl3SEema3Zl8J33iXv9BNGyKH1/
7p+yHYj2ougY2KNqMGgwDgYDVR0PAQH/BAQDAgbAMCkGA1UdDgQiBCCsMh3Xbs+H
qbb7iYyi3G2RhZG0+l8GmYPa/i7NSkIxcDArBgNVHSMEJDAigCDStB+0gbNWFT1p
iPW8+XzJ+vS0m3JZ1gKYSUESt7n/pzAKBggqhkjOPQQDAgNIADBFAiAG3fYB1HEu
Gi7aUUNBIOizWBCtOuWWvmR5FMVSuuUYdAIhALqbClSD9Kt2gYwYucCE7iPajc3H
wyi1e7ZVkH5vjHP8
-----END CERTIFICATE-----
`
		}
		if contractMethod == "enc_auth" {
			Params = `data_key:dataKey||enc_key:encKey||authorized_person:-----BEGIN CERTIFICATE-----
MIICfjCCAiSgAwIBAgIDCgn6MAoGCCqGSM49BAMCMIGKMQswCQYDVQQGEwJDTjEQ
MA4GA1UECBMHQmVpamluZzEQMA4GA1UEBxMHQmVpamluZzEfMB0GA1UEChMWd3gt
b3JnMS5jaGFpbm1ha2VyLm9yZzESMBAGA1UECxMJcm9vdC1jZXJ0MSIwIAYDVQQD
ExljYS53eC1vcmcxLmNoYWlubWFrZXIub3JnMB4XDTI1MDQxODE1NDQyOVoXDTMw
MDQxNzE1NDQyOVowgZcxCzAJBgNVBAYTAkNOMRAwDgYDVQQIEwdCZWlqaW5nMRAw
DgYDVQQHEwdCZWlqaW5nMR8wHQYDVQQKExZ3eC1vcmcxLmNoYWlubWFrZXIub3Jn
MRIwEAYDVQQLEwljb25zZW5zdXMxLzAtBgNVBAMTJmNvbnNlbnN1czEuc2lnbi53
eC1vcmcxLmNoYWlubWFrZXIub3JnMFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAE
XJBsVjVS5zcdQk2RhdA7eRs1DXdVq8xXRCD8G9CQ+YoDp/3bWLTBj7nw2ZYQHdxq
Bp1iPP0tIbv4S/LAw1WbCqNqMGgwDgYDVR0PAQH/BAQDAgbAMCkGA1UdDgQiBCB0
oajU1EwCPAWpcyBwnuaUUo98H4W75/0IyqmbvrXuEDArBgNVHSMEJDAigCDStB+0
gbNWFT1piPW8+XzJ+vS0m3JZ1gKYSUESt7n/pzAKBggqhkjOPQQDAgNIADBFAiEA
zQIb4bTapNnTqbEyr0B2VahFunoFThRZrZG1PXSicTUCIBk3x7Z/PRR9Q/agNuJI
NaH1gyFpD5XW1nlTQa4xdrML
-----END CERTIFICATE-----||authorizer:-----BEGIN CERTIFICATE-----
MIICeDCCAh6gAwIBAgIDDmp3MAoGCCqGSM49BAMCMIGKMQswCQYDVQQGEwJDTjEQ
MA4GA1UECBMHQmVpamluZzEQMA4GA1UEBxMHQmVpamluZzEfMB0GA1UEChMWd3gt
b3JnMS5jaGFpbm1ha2VyLm9yZzESMBAGA1UECxMJcm9vdC1jZXJ0MSIwIAYDVQQD
ExljYS53eC1vcmcxLmNoYWlubWFrZXIub3JnMB4XDTI1MDQxODE1NDQyOVoXDTMw
MDQxNzE1NDQyOVowgZExCzAJBgNVBAYTAkNOMRAwDgYDVQQIEwdCZWlqaW5nMRAw
DgYDVQQHEwdCZWlqaW5nMR8wHQYDVQQKExZ3eC1vcmcxLmNoYWlubWFrZXIub3Jn
MQ8wDQYDVQQLEwZjb21tb24xLDAqBgNVBAMTI2NvbW1vbjEuc2lnbi53eC1vcmcx
LmNoYWlubWFrZXIub3JnMFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAEn4ZMa251
acwZkmZQ/HBWGyy1hMr40ChHJ29aNvlCp9xUBjl3SEema3Zl8J33iXv9BNGyKH1/
7p+yHYj2ougY2KNqMGgwDgYDVR0PAQH/BAQDAgbAMCkGA1UdDgQiBCCsMh3Xbs+H
qbb7iYyi3G2RhZG0+l8GmYPa/i7NSkIxcDArBgNVHSMEJDAigCDStB+0gbNWFT1p
iPW8+XzJ+vS0m3JZ1gKYSUESt7n/pzAKBggqhkjOPQQDAgNIADBFAiAG3fYB1HEu
Gi7aUUNBIOizWBCtOuWWvmR5FMVSuuUYdAIhALqbClSD9Kt2gYwYucCE7iPajc3H
wyi1e7ZVkH5vjHP8
-----END CERTIFICATE-----
||auth_sign:-----BEGIN EC PRIVATE KEY-----
MHcCAQEEIK0M179niQ0F5+iZAjIWSa+frPiYGyrktwUKln/gGOCWoAoGCCqGSM49
AwEHoUQDQgAEn4ZMa251acwZkmZQ/HBWGyy1hMr40ChHJ29aNvlCp9xUBjl3SEem
a3Zl8J33iXv9BNGyKH1/7p+yHYj2ougY2A==
-----END EC PRIVATE KEY-----
||auth_level:2`
		}
		if contractMethod == "get_enc_data" {
			Params = "data_key:dataKey"
		}
		if contractMethod == "get_enc_auth" {
			Params = `data_key:dataKey||authorizer:-----BEGIN CERTIFICATE-----
MIICeDCCAh6gAwIBAgIDDmp3MAoGCCqGSM49BAMCMIGKMQswCQYDVQQGEwJDTjEQ
MA4GA1UECBMHQmVpamluZzEQMA4GA1UEBxMHQmVpamluZzEfMB0GA1UEChMWd3gt
b3JnMS5jaGFpbm1ha2VyLm9yZzESMBAGA1UECxMJcm9vdC1jZXJ0MSIwIAYDVQQD
ExljYS53eC1vcmcxLmNoYWlubWFrZXIub3JnMB4XDTI1MDQxODE1NDQyOVoXDTMw
MDQxNzE1NDQyOVowgZExCzAJBgNVBAYTAkNOMRAwDgYDVQQIEwdCZWlqaW5nMRAw
DgYDVQQHEwdCZWlqaW5nMR8wHQYDVQQKExZ3eC1vcmcxLmNoYWlubWFrZXIub3Jn
MQ8wDQYDVQQLEwZjb21tb24xLDAqBgNVBAMTI2NvbW1vbjEuc2lnbi53eC1vcmcx
LmNoYWlubWFrZXIub3JnMFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAEn4ZMa251
acwZkmZQ/HBWGyy1hMr40ChHJ29aNvlCp9xUBjl3SEema3Zl8J33iXv9BNGyKH1/
7p+yHYj2ougY2KNqMGgwDgYDVR0PAQH/BAQDAgbAMCkGA1UdDgQiBCCsMh3Xbs+H
qbb7iYyi3G2RhZG0+l8GmYPa/i7NSkIxcDArBgNVHSMEJDAigCDStB+0gbNWFT1p
iPW8+XzJ+vS0m3JZ1gKYSUESt7n/pzAKBggqhkjOPQQDAgNIADBFAiAG3fYB1HEu
Gi7aUUNBIOizWBCtOuWWvmR5FMVSuuUYdAIhALqbClSD9Kt2gYwYucCE7iPajc3H
wyi1e7ZVkH5vjHP8
-----END CERTIFICATE-----
`
		}
		if contractMethod == "update_enc_auth" {
			Params = `data_key:dataKey||authorized_person:-----BEGIN CERTIFICATE-----
MIICfjCCAiSgAwIBAgIDCgn6MAoGCCqGSM49BAMCMIGKMQswCQYDVQQGEwJDTjEQ
MA4GA1UECBMHQmVpamluZzEQMA4GA1UEBxMHQmVpamluZzEfMB0GA1UEChMWd3gt
b3JnMS5jaGFpbm1ha2VyLm9yZzESMBAGA1UECxMJcm9vdC1jZXJ0MSIwIAYDVQQD
ExljYS53eC1vcmcxLmNoYWlubWFrZXIub3JnMB4XDTI1MDQxODE1NDQyOVoXDTMw
MDQxNzE1NDQyOVowgZcxCzAJBgNVBAYTAkNOMRAwDgYDVQQIEwdCZWlqaW5nMRAw
DgYDVQQHEwdCZWlqaW5nMR8wHQYDVQQKExZ3eC1vcmcxLmNoYWlubWFrZXIub3Jn
MRIwEAYDVQQLEwljb25zZW5zdXMxLzAtBgNVBAMTJmNvbnNlbnN1czEuc2lnbi53
eC1vcmcxLmNoYWlubWFrZXIub3JnMFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAE
XJBsVjVS5zcdQk2RhdA7eRs1DXdVq8xXRCD8G9CQ+YoDp/3bWLTBj7nw2ZYQHdxq
Bp1iPP0tIbv4S/LAw1WbCqNqMGgwDgYDVR0PAQH/BAQDAgbAMCkGA1UdDgQiBCB0
oajU1EwCPAWpcyBwnuaUUo98H4W75/0IyqmbvrXuEDArBgNVHSMEJDAigCDStB+0
gbNWFT1piPW8+XzJ+vS0m3JZ1gKYSUESt7n/pzAKBggqhkjOPQQDAgNIADBFAiEA
zQIb4bTapNnTqbEyr0B2VahFunoFThRZrZG1PXSicTUCIBk3x7Z/PRR9Q/agNuJI
NaH1gyFpD5XW1nlTQa4xdrML
-----END CERTIFICATE-----||authorizer:-----BEGIN CERTIFICATE-----
MIICeDCCAh6gAwIBAgIDDmp3MAoGCCqGSM49BAMCMIGKMQswCQYDVQQGEwJDTjEQ
MA4GA1UECBMHQmVpamluZzEQMA4GA1UEBxMHQmVpamluZzEfMB0GA1UEChMWd3gt
b3JnMS5jaGFpbm1ha2VyLm9yZzESMBAGA1UECxMJcm9vdC1jZXJ0MSIwIAYDVQQD
ExljYS53eC1vcmcxLmNoYWlubWFrZXIub3JnMB4XDTI1MDQxODE1NDQyOVoXDTMw
MDQxNzE1NDQyOVowgZExCzAJBgNVBAYTAkNOMRAwDgYDVQQIEwdCZWlqaW5nMRAw
DgYDVQQHEwdCZWlqaW5nMR8wHQYDVQQKExZ3eC1vcmcxLmNoYWlubWFrZXIub3Jn
MQ8wDQYDVQQLEwZjb21tb24xLDAqBgNVBAMTI2NvbW1vbjEuc2lnbi53eC1vcmcx
LmNoYWlubWFrZXIub3JnMFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAEn4ZMa251
acwZkmZQ/HBWGyy1hMr40ChHJ29aNvlCp9xUBjl3SEema3Zl8J33iXv9BNGyKH1/
7p+yHYj2ougY2KNqMGgwDgYDVR0PAQH/BAQDAgbAMCkGA1UdDgQiBCCsMh3Xbs+H
qbb7iYyi3G2RhZG0+l8GmYPa/i7NSkIxcDArBgNVHSMEJDAigCDStB+0gbNWFT1p
iPW8+XzJ+vS0m3JZ1gKYSUESt7n/pzAKBggqhkjOPQQDAgNIADBFAiAG3fYB1HEu
Gi7aUUNBIOizWBCtOuWWvmR5FMVSuuUYdAIhALqbClSD9Kt2gYwYucCE7iPajc3H
wyi1e7ZVkH5vjHP8
-----END CERTIFICATE-----
||auth_sign:-----BEGIN EC PRIVATE KEY-----
MHcCAQEEIK0M179niQ0F5+iZAjIWSa+frPiYGyrktwUKln/gGOCWoAoGCCqGSM49
AwEHoUQDQgAEn4ZMa251acwZkmZQ/HBWGyy1hMr40ChHJ29aNvlCp9xUBjl3SEem
a3Zl8J33iXv9BNGyKH1/7p+yHYj2ougY2A==
-----END EC PRIVATE KEY-----
||auth_level:2`
		}
	}
	if ContractName == "compute" {
		Params = ""
	}
	if ContractName == "exchange" {
		if contractMethod == "buyNow" {
			Params = "from:8acfaca5eeec9f6f7c23c4ffac969b86f27799b0||to:818fac1ac51525aeedf619a9a339b95854930159||tokenId:11111111111111111111111||metadata:http://chainmaker.org.cn/"
		}
	}
	if ContractName == "counter" {
		Params = "key:test_key"

	}
	if ContractName == "raffle" {
		if contractMethod == "registRaffle" {
			Params = "peoples:{\"peoples\":[{\"num\":1,\"name\":\"Chris\"},{\"num\":2,\"name\":\"Linus\"}]}||timestamp:13235432||level:1"
		}
	}
	if ContractName == "standard-evidence" {
		if contractMethod == "evidenceAndFindByHash" {
			Params = "evidences:[{\"id\":\"id1\",\"hash\":\"hash1\",\"txId\":\"\",\"blockHeight\":0,\"timestamp\":\"\",\"metadata\":\"11\"},{\"id\":\"id2\",\"hash\":\"hash2\",\"txId\":\"\",\"blockHeight\":0,\"timestamp\":\"\",\"metadata\":\"11\"}]||hash:hash1"

		}
	}
	if ContractName == "itinerary" {
		if contractMethod == "queryHistory" {
			Params = "phone:18892352495||itinerary:{\"ip\":\"117.107.131.195\",\"city\":\"Beijing\",\"region\":\"Beijing\",\"country\":\"CN\",\"loc\":\"39.9075,116.3972\",\"org\":\"\",\"timezone\":\"Asia/Shanghai\",\"asn\":{\"asn\":\"AS4847\",\"name\":\"China Networks Inter-Exchange\",\"domain\":\"bta.net.cn\",\"route\":\"117.107.128.0/18\",\"type\":\"isp\"},\"company\":{\"name\":\"Beijing Sinnet Technology Co., Ltd.\",\"domain\":\"ghidc.net\",\"type\":\"business\"},\"privacy\":{\"vpn\":false,\"proxy\":false,\"tor\":false,\"relay\":false,\"hosting\":false,\"service\":\"\"},\"abuse\":{\"address\":\"Beijing, China\",\"country\":\"CN\",\"email\":\"ipas@cnnic.cn\",\"name\":\"Chen hao\",\"network\":\"117.107.128.0/17\",\"phone\":\"+86-13311166160\"},\"domains\":{\"total\":0,\"domains\":[]}}"

		}

	}
	if ContractName == "fact" {
		if contractMethod == "saveAndFindByFileHash" {
			Params = "file_hash:005521f27d745a04999c6d09f559764f9c44376a||file_name:aoteman.jpg||time:16456254"

		}
	}
	//fmt.Println(Params)
	return Params
}

// 生成调用函数参数列表
func prepareFunc(ContractName, contractMethod string) map[string][]byte {
	Params := getMethodParams(ContractName, contractMethod)
	ParamsList := handleParams(Params)
	parameters := RandParams(ParamsList)
	return parameters
}

// 生成合约文件地址
func prepareFile(ContractName, contractType string) string {
	var filePath string
	filePath = "./testdata/" + ContractName + "-" + contractType + ".wasm"
	return filePath
}

// 这个函数是用来检测上下文写入putstate，getstate是否正常的，可不用，只要最终返回结果是successresult即可
func readWriteSet(txSimContext protocol.TxSimContext) ([]byte, error) {
	rwSet := txSimContext.GetTxRWSet(true)
	fmt.Printf("rwSet = %v \n", rwSet)

	var result []byte
	for _, w := range rwSet.TxWrites {
		if bytes.Equal(w.Key, []byte("count#test_key")) {
			result = w.Value
		}
	}
	if result == nil {
		return nil, fmt.Errorf("write set contain no 'count#test_key'")
	}

	return result, nil
}

// TestInvoke comment at next version
func TestInvoke(t *testing.T) {
	contractMethod := "address"
	ContractName := "identity"
	contractType := "go"
	testTime := 1
	filePath := prepareFile(ContractName, contractType)
	wasmBytes, contractId, logger := prepareContract(filePath, t)
	vmPool, err := newVmPool(&contractId, wasmBytes, logger)
	if err != nil {
		t.Fatalf("create vmPool error: %v", err)
	}

	defer func() {
		vmPool.close()
	}()

	runtimeInst := RuntimeInstance{
		pool:    vmPool,
		log:     logger,
		chainId: ChainId,
	}

	parameters := prepareFunc(ContractName, contractMethod)
	fillingBaseParams(parameters)
	successCnt := 0
	totalExecutionTime := float64(0)
	//txSimContext := prepareTxSimContext(ChainId, BlockVersion, ContractName, contractMethod, parameters, SnapshotMock{})
	//bigNumCalRe := regexp.MustCompile(`contractResult.*executionTime (\d+)`)
	for j := 0; j < testTime; j++ {
		txSimContext := prepareTxSimContext(ChainId, BlockVersion, ContractName, contractMethod, parameters, SnapshotMock{})
		contractResult, _, _, _, executionTime := runtimeInst.InvokeTime(&contractId, contractMethod, wasmBytes, parameters, txSimContext, 0)
		log.Infof("contractResult = %v \n", contractResult)
		resultList := strings.Split(string(contractResult.Result), ",")
		result := resultList[len(resultList)-1]
		index := strings.Index(result, " ")
		ContractTime, _ := strconv.Atoi(result[index+1:])
		TotalContractTime += int64(ContractTime)
		//contractResult.Code为1表示合约函数执行失败
		if contractResult.Code != 0 {
			t.Fatalf("invoke contract failed, contract code")
		} else {
			successCnt += 1
			totalExecutionTime += executionTime
		}
	}
	TPS := float64(successCnt) / totalExecutionTime
	fmt.Printf("successCnt=%d totalExecutionTime=%v TPS = %v \n", successCnt, totalExecutionTime, TPS)
	exRatio := float64(ExportMemoryTime) / float64(totalExecutionTime*1e9) * 100
	rfRatio := float64(RealFuncTime) / float64(totalExecutionTime*1e9) * 100
	rrRatio := float64(ReturnResultTime) / float64(totalExecutionTime*1e9) * 100
	rpRatio := float64(ReadParamTime) / float64(totalExecutionTime*1e9) * 100
	//tmRatio := float64(TotalFuncTime) / float64(TotalContractTime) * 100
	ttRatio := float64(TotalFuncTime) / float64(totalExecutionTime*1e9) * 100
	fmt.Printf("内存导入 平均占比: %.2f%%\n", exRatio)
	fmt.Printf("读取参数 平均占比: %.2f%%\n", rpRatio)
	fmt.Printf("实际函数 平均占比: %.2f%%\n", rfRatio)
	fmt.Printf("结果拷贝 平均占比: %.2f%%\n", rrRatio)
	//fmt.Printf("TotalFuncTime/TotalContractTime 平均占比: %.2f%%\n", tmRatio)
	fmt.Printf("TotalFuncTime/totalExecutionTime 平均占比: %.2f%%\n", ttRatio)
	minus := totalExecutionTime - float64(TotalContractTime)/1e9
	fmt.Printf("minus %f", minus)
	//contractMethod = "enc_auth"
	//parameters = prepareFunc(ContractName, contractMethod)
	//fillingBaseParams(parameters)
	//successCnt = 0
	//totalExecutionTime = float64(0)
	//for j := 0; j < testTime; j++ {
	//	//txSimContext := prepareTxSimContext(ChainId, BlockVersion, ContractName, contractMethod, parameters, SnapshotMock{})
	//	contractResult, _, _, _, executionTime := runtimeInst.InvokeTime(&contractId, contractMethod, wasmBytes, parameters, txSimContext, 0)
	//	fmt.Printf("contractResult = %v \n", contractResult)
	//	//contractResult.Code为1表示合约函数执行失败
	//	if contractResult.Code != 0 {
	//		t.Fatalf("invoke contract failed, contract code")
	//	} else {
	//		successCnt += 1
	//		totalExecutionTime += executionTime
	//	}
	//}
	//TPS = float64(successCnt) / totalExecutionTime
	//fmt.Printf("successCnt=%d totalExecutionTime=%v TPS = %v \n", successCnt, totalExecutionTime, TPS)
	//
	//contractMethod = "get_enc_data"
	//parameters = prepareFunc(ContractName, contractMethod)
	//fillingBaseParams(parameters)
	//successCnt = 0
	//totalExecutionTime = float64(0)
	//for j := 0; j < testTime; j++ {
	//	//txSimContext := prepareTxSimContext(ChainId, BlockVersion, ContractName, contractMethod, parameters, SnapshotMock{})
	//	contractResult, _, _, _, executionTime := runtimeInst.InvokeTime(&contractId, contractMethod, wasmBytes, parameters, txSimContext, 0)
	//	fmt.Printf("contractResult = %v \n", contractResult)
	//	//contractResult.Code为1表示合约函数执行失败
	//	if contractResult.Code != 0 {
	//		t.Fatalf("invoke contract failed, contract code")
	//	} else {
	//		successCnt += 1
	//		totalExecutionTime += executionTime
	//	}
	//}
	//TPS = float64(successCnt) / totalExecutionTime
	//fmt.Printf("successCnt=%d totalExecutionTime=%v TPS = %v \n", successCnt, totalExecutionTime, TPS)
	//
	//contractMethod = "get_enc_auth"
	//parameters = prepareFunc(ContractName, contractMethod)
	//fillingBaseParams(parameters)
	//successCnt = 0
	//totalExecutionTime = float64(0)
	//for j := 0; j < testTime; j++ {
	//	//txSimContext := prepareTxSimContext(ChainId, BlockVersion, ContractName, contractMethod, parameters, SnapshotMock{})
	//	contractResult, _, _, _, executionTime := runtimeInst.InvokeTime(&contractId, contractMethod, wasmBytes, parameters, txSimContext, 0)
	//	fmt.Printf("contractResult = %v \n", contractResult)
	//	//contractResult.Code为1表示合约函数执行失败
	//	if contractResult.Code != 0 {
	//		t.Fatalf("invoke contract failed, contract code")
	//	} else {
	//		successCnt += 1
	//		totalExecutionTime += executionTime
	//	}
	//}
	//TPS = float64(successCnt) / totalExecutionTime
	//fmt.Printf("successCnt=%d totalExecutionTime=%v TPS = %v \n", successCnt, totalExecutionTime, TPS)
	//
	//contractMethod = "update_enc_auth"
	//parameters = prepareFunc(ContractName, contractMethod)
	//fillingBaseParams(parameters)
	//successCnt = 0
	//totalExecutionTime = float64(0)
	//for j := 0; j < testTime; j++ {
	//	//txSimContext := prepareTxSimContext(ChainId, BlockVersion, ContractName, contractMethod, parameters, SnapshotMock{})
	//	contractResult, _, _, _, executionTime := runtimeInst.InvokeTime(&contractId, contractMethod, wasmBytes, parameters, txSimContext, 0)
	//	fmt.Printf("contractResult = %v \n", contractResult)
	//	//contractResult.Code为1表示合约函数执行失败
	//	if contractResult.Code != 0 {
	//		t.Fatalf("invoke contract failed, contract code")
	//	} else {
	//		successCnt += 1
	//		totalExecutionTime += executionTime
	//	}
	//}
	//TPS = float64(successCnt) / totalExecutionTime
	//fmt.Printf("successCnt=%d totalExecutionTime=%v TPS = %v \n", successCnt, totalExecutionTime, TPS)
}
