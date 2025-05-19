/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0

Wacsi WebAssembly chainmaker system interface
*/

package vm

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"math/big"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"

	"chainmaker.org/chainmaker/common/v2/bytehelper"
	"chainmaker.org/chainmaker/common/v2/crypto/bulletproofs"
	"chainmaker.org/chainmaker/common/v2/crypto/paillier"
	"chainmaker.org/chainmaker/common/v2/serialize"
	"chainmaker.org/chainmaker/pb-go/v2/common"
	"chainmaker.org/chainmaker/protocol/v2"
	"chainmaker.org/chainmaker/utils/v2"

	commonCrt "chainmaker.org/chainmaker/common/v2/cert"
	"chainmaker.org/chainmaker/common/v2/crypto"
	"chainmaker.org/chainmaker/common/v2/crypto/asym"
	bcx509 "chainmaker.org/chainmaker/common/v2/crypto/x509"
	"chainmaker.org/chainmaker/common/v2/evmutils"
	"chainmaker.org/chainmaker/pb-go/v2/accesscontrol"
	configPb "chainmaker.org/chainmaker/pb-go/v2/config"
	"chainmaker.org/chainmaker/pb-go/v2/syscontract"
	vmPb "chainmaker.org/chainmaker/pb-go/v2/vm"
	"encoding/hex"
	"encoding/pem"
	"github.com/gogo/protobuf/proto"
)

// ErrorNotManageContract means the method is not init_contract or upgrade
var ErrorNotManageContract = fmt.Errorf("method is not init_contract or upgrade")

// Bool is the Type mapped from int to bool
type Bool int32

const (
	version2201 uint32 = 2201
	version2210 uint32 = 2210
	version2220 uint32 = 2220
	boolTrue    Bool   = 1
	boolFalse   Bool   = 0
)

//// Wacsi WebAssembly chainmaker system interface
//type Wacsi interface {
//	// state operation
//	PutState(requestBody []byte, contractName string, txSimContext protocol.TxSimContext) error
//	GetState(requestBody []byte, contractName string, txSimContext protocol.TxSimContext, memory []byte,
//		data []byte, isLen bool) ([]byte, error)
//	DeleteState(requestBody []byte, contractName string, txSimContext protocol.TxSimContext) error
//	// call other contract
//	CallContract(requestBody []byte, txSimContext protocol.TxSimContext, memory []byte, data []byte,
//		gasUsed uint64, isLen bool) (*common.ContractResult, uint64, protocol.ExecOrderTxType, error)
//	// result record
//	SuccessResult(contractResult *common.ContractResult, data []byte) int32
//	ErrorResult(contractResult *common.ContractResult, data []byte) int32
//	// emit event
//	EmitEvent(requestBody []byte, txSimContext protocol.TxSimContext, contractId *common.Contract,
//		log protocol.Logger) (*common.ContractEvent, error)
//	// paillier
//	PaillierOperation(requestBody []byte, memory []byte, data []byte, isLen bool) ([]byte, error)
//	// bulletproofs
//	BulletProofsOperation(requestBody []byte, memory []byte, data []byte, isLen bool) ([]byte, error)
//
//	// kv iterator
//	KvIterator(requestBody []byte, contractName string, txSimContext protocol.TxSimContext, memory []byte) error
//	KvPreIterator(requestBody []byte, contractName string, txSimContext protocol.TxSimContext, memory []byte) error
//	KvIteratorHasNext(requestBody []byte, txSimContext protocol.TxSimContext, memory []byte) error
//	KvIteratorNext(requestBody []byte, txSimContext protocol.TxSimContext, memory []byte, data []byte,
//		contractName string, isLen bool) ([]byte, error)
//	KvIteratorClose(requestBody []byte, contractName string, txSimContext protocol.TxSimContext, memory []byte) error
//
//	// sql operation
//	ExecuteQuery(requestBody []byte, contractName string, txSimContext protocol.TxSimContext, memory []byte,
//		chainId string) error
//	ExecuteQueryOne(requestBody []byte, contractName string, txSimContext protocol.TxSimContext, memory []byte,
//		data []byte, chainId string, isLen bool) ([]byte, error)
//	ExecuteUpdate(requestBody []byte, contractName string, method string, txSimContext protocol.TxSimContext,
//		memory []byte, chainId string) error
//	ExecuteDDL(requestBody []byte, contractName string, txSimContext protocol.TxSimContext, memory []byte,
//		method string) error
//	RSHasNext(requestBody []byte, txSimContext protocol.TxSimContext, memory []byte) error
//	RSNext(requestBody []byte, txSimContext protocol.TxSimContext, memory []byte, data []byte,
//		isLen bool) ([]byte, error)
//	RSClose(requestBody []byte, txSimContext protocol.TxSimContext, memory []byte) error
//}

// WacsiImpl implements the Wacsi interface(WebAssembly chainmaker system interface)
type WacsiImpl struct {
	verifySql protocol.SqlVerifier
	rowIndex  int32
	enableSql map[string]bool // map[chainId]enableSql
	lock      *sync.Mutex
	logger    protocol.Logger
}

// NewWacsi get wacsi instance
func NewWacsi(logger protocol.Logger, verifySql protocol.SqlVerifier) protocol.Wacsi {
	return &WacsiImpl{
		verifySql: verifySql,
		rowIndex:  0,
		enableSql: make(map[string]bool),
		lock:      &sync.Mutex{},
		logger:    logger,
	}
}

// PutState is used to put state into simContext cache
func (w *WacsiImpl) PutState(requestBody []byte, contractName string, txSimContext protocol.TxSimContext) error {
	ec := serialize.NewEasyCodecWithBytes(requestBody)
	key, _ := ec.GetString("key")
	field, _ := ec.GetString("field")
	value, _ := ec.GetBytes("value")
	w.logger.Debugf("wacsiImpl::PutState() ==> key = %s, field = %s, value = %s \n", key, field, value)
	if err := protocol.CheckKeyFieldStr(key, field); err != nil {
		return err
	}
	err := txSimContext.Put(contractName, protocol.GetKeyStr(key, field), value)
	return err
}

// GetState is used to get state from simContext cache
func (w *WacsiImpl) GetState(requestBody []byte, contractName string, txSimContext protocol.TxSimContext, memory []byte,
	data []byte, isLen bool) ([]byte, error) {
	ec := serialize.NewEasyCodecWithBytes(requestBody)
	key, _ := ec.GetString("key")
	field, _ := ec.GetString("field")
	valuePtr, _ := ec.GetInt32("value_ptr")

	if err := protocol.CheckKeyFieldStr(key, field); err != nil {
		return nil, err
	}

	if !isLen {
		copy(memory[valuePtr:valuePtr+int32(len(data))], data)
		return nil, nil
	}
	w.logger.Debugf("wacsiImpl::GetState() ==> key = %s, field = %s \n", key, field)
	value, err := txSimContext.Get(contractName, protocol.GetKeyStr(key, field))
	if err != nil {
		msg := fmt.Errorf("[get state] fail. key=%s, field=%s, error:%s", key, field, err.Error())
		return nil, msg
	}
	w.logger.Debugf("wacsiImpl::GetState() ==> value = %s \n", value)
	copy(memory[valuePtr:valuePtr+4], bytehelper.IntToBytes(int32(len(value))))
	if len(value) == 0 {
		return nil, nil
	}
	return value, nil
}

// GetBatchState is used to get state from simContext cache
func (w *WacsiImpl) GetBatchState(requestBody []byte, contractName string, txSimContext protocol.TxSimContext, memory []byte,
	data []byte, isLen bool) ([]byte, error) {
	ec := serialize.NewEasyCodecWithBytes(requestBody)
	temp, _ := ec.GetBytes("BatchKeys")
	keys := &vmPb.BatchKeys{}
	if err := keys.Unmarshal(temp); err != nil {
		return nil, err
	}
	valuePtr, _ := ec.GetInt32("value_ptr")
	batchKeys := keys.Keys
	for _, key := range batchKeys {
		if err := protocol.CheckKeyFieldStr(key.Key, key.Field); err != nil {
			return nil, err
		}
	}

	if !isLen {
		copy(memory[valuePtr:valuePtr+int32(len(data))], data)
		return nil, nil
	}
	getKeys, err := txSimContext.GetKeys(keys.Keys)
	if err != nil {
		msg := fmt.Errorf("[get batch]error:%s", err.Error())
		return nil, msg
	}
	w.logger.Debugf("wacsiImpl::GetBatchState() ==> value = %v \n", getKeys)
	resp := vmPb.BatchKeys{Keys: getKeys}
	value, err := resp.Marshal()
	if err != nil {
		return nil, err
	}
	copy(memory[valuePtr:valuePtr+4], bytehelper.IntToBytes(int32(len(value))))
	if len(value) == 0 {
		return nil, nil
	}
	return value, nil
}

// GetSenderAddress
func (w *WacsiImpl) GetSenderAddress(requestBody []byte, contractName string, txSimContext protocol.TxSimContext, memory []byte,
	data []byte, isLen bool) ([]byte, error) {
	ec := serialize.NewEasyCodecWithBytes(requestBody)
	valuePtr, _ := ec.GetInt32("value_ptr")
	var bytes []byte
	bytes, err := txSimContext.Get(chainConfigContractName, []byte(keyChainConfig))
	if !isLen {
		copy(memory[valuePtr:valuePtr+int32(len(data))], data)
		return nil, nil
	}
	if err != nil {
		w.logger.Errorf("txSimContext get failed, name[%s] key[%s] err: %s",
			chainConfigContractName, keyChainConfig, err.Error())
		return nil, err
	}
	var chainConfig configPb.ChainConfig

	if err = proto.Unmarshal(bytes, &chainConfig); err != nil {
		w.logger.Errorf("unmarshal chainConfig failed, contractName %s err: %+v", chainConfigContractName, err)
		return nil, err
	}
	var address string
	address, err = w.getSenderAddrWithBlockVersion(txSimContext.GetBlockVersion(), chainConfig, txSimContext)
	if err != nil {
		w.logger.Error(err.Error())
		return nil, err
	}
	value := []byte(address)
	copy(memory[valuePtr:valuePtr+4], bytehelper.IntToBytes(int32(len(value))))
	if len(value) == 0 {
		return nil, nil
	}
	return value, nil
}
func (w *WacsiImpl) getSenderAddrWithBlockVersion(blockVersion uint32, chainConfig configPb.ChainConfig,
	txSimContext protocol.TxSimContext) (string, error) {
	var address string
	var err error

	sender := txSimContext.GetSender()

	switch sender.MemberType {
	case accesscontrol.MemberType_CERT:
		address, err = w.getSenderAddressFromCert(blockVersion, sender.MemberInfo, chainConfig.Vm.AddrType)
		if err != nil {
			w.logger.Errorf("getSenderAddressFromCert failed, %s", err.Error())
			return "", err
		}
	case accesscontrol.MemberType_CERT_HASH,
		accesscontrol.MemberType_ALIAS:
		if blockVersion < version2201 && sender.MemberType == accesscontrol.MemberType_ALIAS {
			w.logger.Error("handleGetSenderAddress failed, invalid member type")
			return "", err
		}

		address, err = w.getSenderAddressFromCertHash(
			blockVersion,
			sender.MemberInfo,
			chainConfig.Vm.AddrType,
			txSimContext,
		)
		if err != nil {
			w.logger.Errorf("getSenderAddressFromCert failed, %s", err.Error())
			return "", err
		}

	case accesscontrol.MemberType_PUBLIC_KEY:
		address, err = w.getSenderAddressFromPublicKeyPEM(blockVersion, sender.MemberInfo, chainConfig.Vm.AddrType,
			crypto.HashAlgoMap[chainConfig.GetCrypto().Hash])
		if err != nil {
			w.logger.Errorf("getSenderAddressFromPublicKeyPEM failed, %s", err.Error())
			return "", err
		}

	default:
		w.logger.Errorf("getSenderAddrWithBlockVersion failed, invalid member type")
		return "", err
	}

	return address, nil
}
func (w *WacsiImpl) getSenderAddressFromCertHash(blockVersion uint32, memberInfo []byte,
	addressType configPb.AddrType, txSimContext protocol.TxSimContext) (string, error) {
	var certBytes []byte
	var err error
	certBytes, err = w.getCertFromChain(memberInfo, txSimContext)
	if err != nil {
		return "", err
	}

	var address string
	address, err = w.getSenderAddressFromCert(blockVersion, certBytes, addressType)
	if err != nil {
		w.logger.Errorf("getSenderAddressFromCert failed, %s", err.Error())
		return "", err
	}

	return address, nil
}
func (w *WacsiImpl) getCertFromChain(memberInfo []byte, txSimContext protocol.TxSimContext) ([]byte, error) {
	certHashKey := hex.EncodeToString(memberInfo)
	certBytes, err := txSimContext.Get(syscontract.SystemContract_CERT_MANAGE.String(), []byte(certHashKey))
	if err != nil {
		w.logger.Errorf("get cert from chain failed, %s", err.Error())
		return nil, err
	}

	return certBytes, nil
}
func (w *WacsiImpl) getSenderAddressFromCert(blockVersion uint32, certPem []byte,
	addressType configPb.AddrType) (string, error) {
	if addressType == configPb.AddrType_ZXL {
		address, err := evmutils.ZXAddressFromCertificatePEM(certPem)
		if err != nil {
			return "", fmt.Errorf("ParseCertificate failed, %s", err.Error())
		}

		return address, nil
	}

	if blockVersion >= version2220 {
		if addressType == configPb.AddrType_CHAINMAKER || addressType == configPb.AddrType_ETHEREUM {
			return w.calculateCertAddr2220(certPem)
		}

		return "", errors.New(fmt.Sprintf("getSenderAddressFromCert >=version2220:invalid address type %d addressType %+v", blockVersion, addressType))
	}

	if blockVersion == version2201 || blockVersion == version2210 {
		if addressType == configPb.AddrType_CHAINMAKER {
			return w.calculateCertAddrBefore2220(certPem)
		}

		return "", errors.New(fmt.Sprintf("getSenderAddressFromCert == version2201 || blockVersion == version2210:invalid address type %d", blockVersion))
	}

	if blockVersion < version2201 {
		if addressType == configPb.AddrType_ETHEREUM {
			return w.calculateCertAddrBefore2220(certPem)
		}

		return "", errors.New(fmt.Sprintf("getSenderAddressFromCert < version2201:invalid address type %d", blockVersion))
	}

	return "", errors.New(fmt.Sprintf("getSenderAddressFromCert :invalid address type %d", blockVersion))
}

func (w *WacsiImpl) calculateCertAddrBefore2220(certPem []byte) (string, error) {
	blockCrt, _ := pem.Decode(certPem)
	crt, err := bcx509.ParseCertificate(blockCrt.Bytes)
	if err != nil {
		return "", fmt.Errorf("MakeAddressFromHex failed, %s", err.Error())
	}

	ski := hex.EncodeToString(crt.SubjectKeyId)
	addrInt, err := evmutils.MakeAddressFromHex(ski)
	if err != nil {
		return "", fmt.Errorf("MakeAddressFromHex failed, %s", err.Error())
	}

	return addrInt.String(), nil
}

func (w *WacsiImpl) calculateCertAddr2220(certPem []byte) (string, error) {
	blockCrt, _ := pem.Decode(certPem)
	crt, err := bcx509.ParseCertificate(blockCrt.Bytes)
	if err != nil {
		return "", fmt.Errorf("MakeAddressFromHex failed, %s", err.Error())
	}

	ski := hex.EncodeToString(crt.SubjectKeyId)
	addrInt, err := evmutils.MakeAddressFromHex(ski)
	if err != nil {
		return "", fmt.Errorf("MakeAddressFromHex failed, %s", err.Error())
	}

	addr := evmutils.BigToAddress(addrInt)
	addrBytes := addr[:]

	return hex.EncodeToString(addrBytes), nil
}
func (w *WacsiImpl) getSenderAddressFromPublicKeyPEM(blockVersion uint32, publicKeyPem []byte,
	addressType configPb.AddrType, hashType crypto.HashType) (string, error) {
	if addressType == configPb.AddrType_ZXL {
		address, err := evmutils.ZXAddressFromPublicKeyPEM(publicKeyPem)
		if err != nil {
			w.logger.Errorf("ZXAddressFromPublicKeyPEM, failed, %s", err.Error())
		}
		return address, err
	}

	if blockVersion >= version2220 {
		if addressType == configPb.AddrType_CHAINMAKER {
			return w.calculatePubKeyAddr2220(publicKeyPem, hashType)
		}

		return "", errors.New(fmt.Sprintf("getSenderAddressFromPublicKeyPEM blockVersion >= version2220:invalid address type %d %+v", blockVersion, addressType))
	}

	if blockVersion == version2201 || blockVersion == version2210 {
		if addressType == configPb.AddrType_CHAINMAKER {
			return w.calculatePubKeyAddrBefore2220(publicKeyPem, hashType)
		}

		return "", errors.New(fmt.Sprintf("getSenderAddressFromPublicKeyPEM blockVersion == version2201 || blockVersion == version2210:invalid address type %d", blockVersion))
	}

	if blockVersion < version2201 {
		if addressType == configPb.AddrType_ETHEREUM {
			return w.calculatePubKeyAddrBefore2220(publicKeyPem, hashType)
		}

		return "", errors.New(fmt.Sprintf("getSenderAddressFromPublicKeyPEM blockVersion < version2201:invalid address type %d", blockVersion))
	}

	return "", errors.New(fmt.Sprintf("getSenderAddressFromPublicKeyPEM :invalid address type %d", blockVersion))
}

func (w *WacsiImpl) calculatePubKeyAddrBefore2220(publicKeyPem []byte, hashType crypto.HashType) (string, error) {
	publicKey, err := asym.PublicKeyFromPEM(publicKeyPem)
	if err != nil {
		return "", fmt.Errorf("ParsePublicKey failed, %s", err.Error())
	}

	ski, err := commonCrt.ComputeSKI(hashType, publicKey.ToStandardKey())
	if err != nil {
		return "", fmt.Errorf("computeSKI from public key failed, %s", err.Error())
	}

	addr, err := evmutils.MakeAddressFromHex(hex.EncodeToString(ski))
	if err != nil {
		return "", fmt.Errorf("make address from cert SKI failed, %s", err)
	}
	return addr.String(), nil
}

func (w *WacsiImpl) calculatePubKeyAddr2220(publicKeyPem []byte, hashType crypto.HashType) (string, error) {
	publicKey, err := asym.PublicKeyFromPEM(publicKeyPem)
	if err != nil {
		return "", fmt.Errorf("ParsePublicKey failed, %s", err.Error())
	}

	ski, err := commonCrt.ComputeSKI(hashType, publicKey.ToStandardKey())
	if err != nil {
		return "", fmt.Errorf("computeSKI from public key failed, %s", err.Error())
	}

	addrInt, err := evmutils.MakeAddressFromHex(hex.EncodeToString(ski))
	if err != nil {
		return "", fmt.Errorf("make address from public key failed, %s", err)
	}

	addr := evmutils.BigToAddress(addrInt)
	addrBytes := addr[:]

	return hex.EncodeToString(addrBytes), nil
}

// sha256
func (w *WacsiImpl) Sha256(requestBody []byte, contractName string, memory []byte) ([]byte, error) {
	ec := serialize.NewEasyCodecWithBytes(requestBody)
	hashInput, _ := ec.GetBytes("hashInput")
	valuePtr, _ := ec.GetInt32("value_ptr")

	//w.logger.Infof("wacsiImpl::sha256() ==> hashInput = %s", string(hashInput))
	value := sha256.Sum256([]byte(hashInput))

	//w.logger.Infof("wacsiImpl::sha256() ==> value = %x", string(value[:]))
	copy(memory[valuePtr:valuePtr+32], value[:])
	if len(value) == 0 {
		return nil, nil
	}
	return value[:], nil
}

// NativeSha256
func (w *WacsiImpl) NativeSha256(hashInput []byte) [32]byte {
	value := sha256.Sum256([]byte(hashInput))
	return value
}

// DeleteState is used to delete state from simContext cache
func (w *WacsiImpl) DeleteState(requestBody []byte, contractName string, txSimContext protocol.TxSimContext) error {
	ec := serialize.NewEasyCodecWithBytes(requestBody)
	key, _ := ec.GetString("key")
	field, _ := ec.GetString("field")
	if err := protocol.CheckKeyFieldStr(key, field); err != nil {
		return err
	}

	err := txSimContext.Del(contractName, protocol.GetKeyStr(key, field))
	if err != nil {
		return err
	}
	return nil
}

// CallContract implement syscall for call contract, it is for gasm and wasmer
func (w *WacsiImpl) CallContract(requestBody []byte, txSimContext protocol.TxSimContext, memory []byte,
	data []byte, gasUsed uint64, isLen bool) (*common.ContractResult, uint64, protocol.ExecOrderTxType, error) {
	ec := serialize.NewEasyCodecWithBytes(requestBody)
	valuePtr, _ := ec.GetInt32("value_ptr")
	contractName, _ := ec.GetString("contract_name")
	method, _ := ec.GetString("method")
	param, _ := ec.GetBytes("param")

	ecData := serialize.NewEasyCodecWithBytes(param)
	paramItem := ecData.GetItems()

	if !isLen { // get value from cache
		result := txSimContext.GetCurrentResult()
		copy(memory[valuePtr:valuePtr+int32(len(result))], result)
		return nil, gasUsed, protocol.ExecOrderTxTypeNormal, nil
	}

	// check param
	if len(contractName) == 0 {
		return nil, gasUsed, protocol.ExecOrderTxTypeNormal, fmt.Errorf("[call contract] contract_name is null")
	}
	if len(method) == 0 {
		return nil, gasUsed, protocol.ExecOrderTxTypeNormal, fmt.Errorf("[call contract] method is null")
	}
	if len(paramItem) > protocol.ParametersKeyMaxCount {
		return nil, gasUsed, protocol.ExecOrderTxTypeNormal, fmt.Errorf("[call contract] expect less than %d "+
			"parameters, but got %d", protocol.ParametersKeyMaxCount, len(paramItem))
	}
	paramMap := ecData.ToMap()
	for key, val := range paramMap {
		if len(key) > protocol.DefaultMaxStateKeyLen {
			return nil, gasUsed, protocol.ExecOrderTxTypeNormal, fmt.Errorf("[call contract] param expect "+
				"key length less than %d, but got %d", protocol.DefaultMaxStateKeyLen, len(key))
		}

		re, err := regexp.Compile(protocol.DefaultStateRegex)
		if err != nil {
			return nil, gasUsed, protocol.ExecOrderTxTypeNormal, err
		}
		match := re.MatchString(key)
		if !match {
			return nil, gasUsed, protocol.ExecOrderTxTypeNormal, fmt.Errorf("[call contract] param expect key "+
				"no special characters, but got %s. letter, number, dot and underline are allowed", key)
		}
		if len(val) > int(protocol.ParametersValueMaxLength) {
			return nil, gasUsed, protocol.ExecOrderTxTypeNormal, fmt.Errorf("[call contract] expect value "+
				"length less than %d, but got %d", protocol.ParametersValueMaxLength, len(val))
		}
	}
	if err := protocol.CheckKeyFieldStr(contractName, method); err != nil {
		return nil, gasUsed, protocol.ExecOrderTxTypeNormal, err
	}

	// call contract
	gasUsed += protocol.CallContractGasOnce

	contract, err := txSimContext.GetContractByName(contractName)
	if err != nil {
		return nil, gasUsed, protocol.ExecOrderTxTypeNormal, fmt.Errorf(
			"[call contract] failed to get contract by [%s], err: %s",
			contractName,
			err.Error(),
		)
	}

	result, specialTxType, code := txSimContext.CallContract(contract, method,
		nil, paramMap, gasUsed, txSimContext.GetTx().Payload.TxType)
	gasUsed += result.GasUsed
	if code != common.TxStatusCode_SUCCESS {
		return nil, gasUsed, specialTxType, fmt.Errorf("[call contract] execute error code: %s, msg: %s",
			code.String(), result.Message)
	}
	// set value length to memory
	l := bytehelper.IntToBytes(int32(len(result.Result)))
	copy(memory[valuePtr:valuePtr+4], l)
	if len(result.Result) == 0 {
		return nil, gasUsed, specialTxType, nil
	}
	return result, gasUsed, specialTxType, nil
}

// SuccessResult is used to construct successful result
func (w *WacsiImpl) SuccessResult(contractResult *common.ContractResult, data []byte) int32 {
	if contractResult.Code == uint32(1) {
		return protocol.ContractSdkSignalResultFail
	}
	contractResult.Code = 0
	contractResult.Result = data
	return protocol.ContractSdkSignalResultSuccess
}

// ErrorResult is used to construct failed result
func (w *WacsiImpl) ErrorResult(contractResult *common.ContractResult, data []byte) int32 {
	contractResult.Code = uint32(1)
	if len(contractResult.Message) > 0 {
		contractResult.Message += ". contract message:" + string(data)
	} else {
		contractResult.Message = "contract message:" + string(data)
	}
	return protocol.ContractSdkSignalResultSuccess
}

// EmitEvent emit event to chain
func (w *WacsiImpl) EmitEvent(requestBody []byte, txSimContext protocol.TxSimContext, contractId *common.Contract,
	log protocol.Logger) (*common.ContractEvent, error) {
	ec := serialize.NewEasyCodecWithBytes(requestBody)
	topic, err := ec.GetString("topic")
	if err != nil {
		return nil, fmt.Errorf("[emit event] get topic err")
	}
	if err := protocol.CheckTopicStr(topic); err != nil {
		return nil, err
	}

	req := ec.GetItems()
	var eventData []string
	for i := 1; i < len(req); i++ {
		data, _ := req[i].Value.(string)
		eventData = append(eventData, data)
		log.Debugf("[emit event] event data :%v", data)
	}

	if err := protocol.CheckEventData(eventData); err != nil {
		return nil, err
	}

	contractEvent := &common.ContractEvent{
		ContractName:    contractId.Name,
		ContractVersion: contractId.Version,
		Topic:           topic,
		TxId:            txSimContext.GetTx().Payload.TxId,
		EventData:       eventData,
	}
	ddl := utils.GenerateSaveContractEventDdl(contractEvent, "chainId", 1, 1)
	count := utils.GetSqlStatementCount(ddl)
	if count != 1 {
		return nil, fmt.Errorf("[emit event] contract event parameter error, exist sql injection")
	}

	return contractEvent, nil
}

// KvIterator construct a kv iterator
func (w *WacsiImpl) KvIterator(requestBody []byte, contractName string, txSimContext protocol.TxSimContext,
	memory []byte) error {
	ec := serialize.NewEasyCodecWithBytes(requestBody)
	startKey, _ := ec.GetString("start_key")
	startField, _ := ec.GetString("start_field")
	limitKey, _ := ec.GetString("limit_key")
	limitField, _ := ec.GetString("limit_field")
	valuePtr, _ := ec.GetInt32("value_ptr")
	if err := protocol.CheckKeyFieldStr(startKey, startField); err != nil {
		return err
	}
	if err := protocol.CheckKeyFieldStr(limitKey, limitField); err != nil {
		return err
	}

	key := protocol.GetKeyStr(startKey, startField)
	limit := protocol.GetKeyStr(limitKey, limitField)
	iter, err := txSimContext.Select(contractName, key, limit)
	if err != nil {
		return fmt.Errorf("[kv iterator] select error, %s", err.Error())
	}

	index := atomic.AddInt32(&w.rowIndex, 1)
	txSimContext.SetIterHandle(index, iter)
	copy(memory[valuePtr:valuePtr+4], bytehelper.IntToBytes(index))
	return nil
}

// KvPreIterator construct a kV iterator based on prefix matching
func (w *WacsiImpl) KvPreIterator(requestBody []byte, contractName string, txSimContext protocol.TxSimContext,
	memory []byte) error {
	ec := serialize.NewEasyCodecWithBytes(requestBody)
	startKey, _ := ec.GetString("start_key")
	startField, _ := ec.GetString("start_field")
	valuePtr, _ := ec.GetInt32("value_ptr")
	if err := protocol.CheckKeyFieldStr(startKey, startField); err != nil {
		return err
	}

	key := string(protocol.GetKeyStr(startKey, startField))

	limitLast := key[len(key)-1] + 1
	limit := key[:len(key)-1] + string(limitLast)

	iter, err := txSimContext.Select(contractName, []byte(key), []byte(limit))
	if err != nil {
		return fmt.Errorf("[kv pre iterator] select error, %s", err.Error())
	}

	index := atomic.AddInt32(&w.rowIndex, 1)
	txSimContext.SetIterHandle(index, iter)
	copy(memory[valuePtr:valuePtr+4], bytehelper.IntToBytes(index))
	return nil
}

// KvIteratorHasNext is used to determine whether there is another element
func (w *WacsiImpl) KvIteratorHasNext(requestBody []byte, txSimContext protocol.TxSimContext, memory []byte) error {
	ec := serialize.NewEasyCodecWithBytes(requestBody)
	kvIndex, _ := ec.GetInt32("rs_index")
	valuePtr, _ := ec.GetInt32("value_ptr")

	// get
	iter, ok := txSimContext.GetIterHandle(kvIndex)
	if !ok {
		return fmt.Errorf("[kv iterator has next] can not found rs_index[%d]", kvIndex)
	}

	kvRows, ok := iter.(protocol.StateIterator)
	if !ok {
		return fmt.Errorf("[kv iterator has next] failed, iterator %d assertion failed", kvIndex)
	}

	index := boolFalse
	if kvRows.Next() {
		index = boolTrue
	}
	copy(memory[valuePtr:valuePtr+4], bytehelper.IntToBytes(int32(index)))
	return nil
}

// KvIteratorNext get next element
func (*WacsiImpl) KvIteratorNext(requestBody []byte, txSimContext protocol.TxSimContext, memory []byte, data []byte,
	contractname string, isLen bool) ([]byte, error) {
	ec := serialize.NewEasyCodecWithBytes(requestBody)
	kvIndex, _ := ec.GetInt32("rs_index")
	ptr, _ := ec.GetInt32("value_ptr")

	// get handle
	iter, ok := txSimContext.GetIterHandle(kvIndex)
	if !ok {
		return nil, fmt.Errorf("[kv iterator next] can not found rs_index[%d]", kvIndex)
	}
	// get data
	if !isLen {
		copy(memory[ptr:ptr+int32(len(data))], data)
		return nil, nil
	}

	kvRows, ok := iter.(protocol.StateIterator)
	if !ok {
		return nil, fmt.Errorf("[kv iterator next] failed, iterator %d assertion failed", kvIndex)
	}
	// get len
	ec = serialize.NewEasyCodec()
	if kvRows != nil {
		kvRow, err := kvRows.Value()
		if err != nil {
			return nil, fmt.Errorf("[kv iterator next] iterator next data error, %s", err.Error())
		}

		arrKey := strings.Split(string(kvRow.Key), "#")
		key := arrKey[0]
		field := ""
		if len(arrKey) > 1 {
			field = arrKey[1]
		}

		value := kvRow.Value
		ec.AddString("key", key)
		ec.AddString("field", field)
		ec.AddBytes("value", value)
	}
	kvBytes := ec.Marshal()
	copy(memory[ptr:ptr+4], bytehelper.IntToBytes(int32(len(kvBytes))))
	return kvBytes, nil
}

// KvIteratorClose close iteraotr
func (w *WacsiImpl) KvIteratorClose(requestBody []byte, contractName string, txSimContext protocol.TxSimContext,
	memory []byte) error {
	ec := serialize.NewEasyCodecWithBytes(requestBody)
	kvIndex, _ := ec.GetInt32("rs_index")
	valuePtr, _ := ec.GetInt32("value_ptr")
	// get
	iter, ok := txSimContext.GetIterHandle(kvIndex)
	if !ok {
		return fmt.Errorf("[kv iterator close] ctx can not found rs_index[%d]", kvIndex)
	}

	kvRows, ok := iter.(protocol.StateIterator)
	if !ok {
		return fmt.Errorf("[kv iterator close] failed, iterator %d assertion failed", kvIndex)
	}

	kvRows.Release()
	copy(memory[valuePtr:valuePtr+4], bytehelper.IntToBytes(1))
	return nil
}

// HistoryKvIterator construct a kv iterator
func (w *WacsiImpl) HistoryKvIterator(requestBody []byte, contractName string, txSimContext protocol.TxSimContext,
	memory []byte) error {
	ec := serialize.NewEasyCodecWithBytes(requestBody)
	startKey, _ := ec.GetString("start_key")
	startField, _ := ec.GetString("start_field")

	valuePtr, _ := ec.GetInt32("value_ptr")
	if err := protocol.CheckKeyFieldStr(startKey, startField); err != nil {
		return err
	}

	key := protocol.GetKeyStr(startKey, startField)

	iter, err := txSimContext.GetHistoryIterForKey(contractName, key)
	if err != nil {
		return fmt.Errorf("[History kv iterator] select error, %s", err.Error())
	}
	index := atomic.AddInt32(&w.rowIndex, 1)
	txSimContext.SetIterHandle(index, iter)
	copy(memory[valuePtr:valuePtr+4], bytehelper.IntToBytes(index))
	return nil
}

// HistoryKvIterHasNext is used to determine whether there is another element
func (w *WacsiImpl) HistoryKvIterHasNext(requestBody []byte, txSimContext protocol.TxSimContext, memory []byte) error {
	ec := serialize.NewEasyCodecWithBytes(requestBody)
	kvIndex, _ := ec.GetInt32("ks_index")
	valuePtr, _ := ec.GetInt32("value_ptr")

	// get
	iter, ok := txSimContext.GetIterHandle(kvIndex)
	if !ok {
		return fmt.Errorf("[History kv iterator has next] can not found rs_index[%d]", kvIndex)
	}

	keyHistoryIterator, ok := iter.(protocol.KeyHistoryIterator)
	if !ok {
		return fmt.Errorf("[History kv iterator has next] failed, iterator %d assertion failed", kvIndex)
	}

	index := boolFalse
	if keyHistoryIterator.Next() {
		index = boolTrue
	}
	copy(memory[valuePtr:valuePtr+4], bytehelper.IntToBytes(int32(index)))
	return nil
}

// HistoryKvIterNext get next element
func (w *WacsiImpl) HistoryKvIterNext(requestBody []byte, txSimContext protocol.TxSimContext, memory []byte, data []byte,
	contractname string, isLen bool) ([]byte, error) {
	ec := serialize.NewEasyCodecWithBytes(requestBody)
	kvIndex, _ := ec.GetInt32("ks_index")
	ptr, _ := ec.GetInt32("value_ptr")

	// get handle
	iter, ok := txSimContext.GetIterHandle(kvIndex)
	if !ok {
		return nil, fmt.Errorf("[History kv iterator next] can not found rs_index[%d]", kvIndex)
	}
	// get data
	if !isLen {
		copy(memory[ptr:ptr+int32(len(data))], data)
		return nil, nil
	}

	keyHistoryIterator, ok := iter.(protocol.KeyHistoryIterator)
	if !ok {
		return nil, fmt.Errorf("[History kv iterator next] failed, iterator %d assertion failed", kvIndex)
	}
	// get len
	ec = serialize.NewEasyCodec()
	if keyHistoryIterator != nil {
		historyValue, err := keyHistoryIterator.Value()
		if err != nil {
			return nil, fmt.Errorf("[History kv iterator next] iterator next data error, %s", err.Error())
		}

		value := historyValue.Value
		blockHeight := int32(historyValue.BlockHeight)
		timestampStr := strconv.FormatInt(historyValue.Timestamp, 10)
		isDelete := 1
		if !historyValue.IsDelete {
			isDelete = 0
		}
		txId := txSimContext.GetTx().Payload.TxId
		ec.AddBytes("value", value)
		ec.AddString("txId", txId)
		ec.AddInt32("blockHeight", blockHeight)
		ec.AddString("timestamp", timestampStr)
		ec.AddInt32("isDelete", int32(isDelete))
	}
	kvBytes := ec.Marshal()
	copy(memory[ptr:ptr+4], bytehelper.IntToBytes(int32(len(kvBytes))))
	return kvBytes, nil
}

// HistoryKvIterClose close iteraotr
func (w *WacsiImpl) HistoryKvIterClose(requestBody []byte, contractName string, txSimContext protocol.TxSimContext,
	memory []byte) error {
	ec := serialize.NewEasyCodecWithBytes(requestBody)
	kvIndex, _ := ec.GetInt32("ks_index")
	valuePtr, _ := ec.GetInt32("value_ptr")
	// get
	iter, ok := txSimContext.GetIterHandle(kvIndex)
	if !ok {
		return fmt.Errorf("[History kv iterator close] ctx can not found rs_index[%d]", kvIndex)
	}

	keyHistoryIterator, ok := iter.(protocol.KeyHistoryIterator)
	if !ok {
		return fmt.Errorf("[History kv iterator close] failed, iterator %d assertion failed", kvIndex)
	}

	keyHistoryIterator.Release()
	copy(memory[valuePtr:valuePtr+4], bytehelper.IntToBytes(1))
	return nil
}

// BulletProofsOperation is used to handle bulletproofs operations
func (*WacsiImpl) BulletProofsOperation(requestBody []byte, memory []byte, data []byte, isLen bool) ([]byte, error) {
	ec := serialize.NewEasyCodecWithBytes(requestBody)

	/*	bulletproofsFuncName:
		| func  									|
		|-------------------------------------------|
		| BulletProofsOpTypePedersenAddNum 			|
		| BulletProofsOpTypePedersenAddCommitment   |
		| BulletProofsOpTypePedersenSubNum 			|
		| BulletProofsOpTypePedersenSubCommitment   |
		| BulletProofsOpTypePedersenMulNum 			|
		| BulletProofsVerify					    |
	*/
	opTypeStr, _ := ec.GetString("bulletproofsFuncName")

	/*  param1, param2:
	| func  									| param1 	 | param2	  |
	|-------------------------------------------|------------|------------|
	| BulletProofsOpTypePedersenAddNum 			| commitment | num		  |
	| BulletProofsOpTypePedersenAddCommitment   | commitment | commitment |
	| BulletProofsOpTypePedersenSubNum 			| commitment | num		  |
	| BulletProofsOpTypePedersenSubCommitment   | commitment | commitment |
	| BulletProofsOpTypePedersenMulNum 			| commitment | num		  |
	| BulletProofsVerify					    | proof 	 | commitment |
	*/
	param1, _ := ec.GetBytes("param1")
	param2, _ := ec.GetBytes("param2")
	valuePtr, _ := ec.GetInt32("value_ptr")

	if !isLen {
		copy(memory[valuePtr:valuePtr+int32(len(data))], data)
		return nil, nil
	}

	resultBytes := make([]byte, 0)
	var err error
	switch opTypeStr {
	case protocol.BulletProofsOpTypePedersenAddNum:
		resultBytes, err = pedersenAddNum(param1, param2)
	case protocol.BulletProofsOpTypePedersenAddCommitment:
		resultBytes, err = pedersenAddCommitment(param1, param2)
	case protocol.BulletProofsOpTypePedersenSubNum:
		resultBytes, err = pedersenSubNum(param1, param2)
	case protocol.BulletProofsOpTypePedersenSubCommitment:
		resultBytes, err = pedersenSubCommitment(param1, param2)
	case protocol.BulletProofsOpTypePedersenMulNum:
		resultBytes, err = pedersenMulNum(param1, param2)
	case protocol.BulletProofsVerify:
		resultBytes, err = bulletproofsVerify(param1, param2)
	}

	if err != nil {
		return nil, err
	}

	copy(memory[valuePtr:valuePtr+4], bytehelper.IntToBytes(int32(len(resultBytes))))
	return resultBytes, nil
}

func pedersenAddNum(commitment, num interface{}) ([]byte, error) {
	//bulletproofs.Helper().NewBulletproofs()
	c, _ := commitment.([]byte)
	x, err := strconv.ParseUint(string(num.([]byte)), 10, 64)
	if err != nil {
		return nil, err
	}

	return bulletproofs.PedersenAddNum(c, x)
}

func pedersenAddCommitment(commitment1, commitment2 interface{}) ([]byte, error) {
	commitmentX, _ := commitment1.([]byte)
	commitmentY, _ := commitment2.([]byte)

	return bulletproofs.PedersenAddCommitment(commitmentX, commitmentY)
}

func pedersenSubNum(commitment, num interface{}) ([]byte, error) {
	c, _ := commitment.([]byte)
	x, err := strconv.ParseUint(string(num.([]byte)), 10, 64)
	if err != nil {
		return nil, err
	}

	return bulletproofs.PedersenSubNum(c, x)
}

func pedersenSubCommitment(commitment1, commitment2 interface{}) ([]byte, error) {
	commitmentX, _ := commitment1.([]byte)
	commitmentY, _ := commitment2.([]byte)
	return bulletproofs.PedersenSubCommitment(commitmentX, commitmentY)

}

func pedersenMulNum(commitment, num interface{}) ([]byte, error) {
	c, _ := commitment.([]byte)
	x, err := strconv.ParseUint(string(num.([]byte)), 10, 64)
	if err != nil {
		return nil, err
	}

	return bulletproofs.PedersenMulNum(c, x)
}

func bulletproofsVerify(proof, commitment interface{}) ([]byte, error) {
	p, _ := proof.([]byte)
	c, _ := commitment.([]byte)
	ok, err := bulletproofs.Verify(p, c)
	if err != nil {
		return nil, err
	}

	if !ok {
		// verify failed
		return []byte("0"), nil
	}

	// verify success
	return []byte("1"), nil
}

// PaillierOperation is used to handle paillier operations
func (*WacsiImpl) PaillierOperation(requestBody []byte, memory []byte, data []byte, isLen bool) ([]byte, error) {
	ec := serialize.NewEasyCodecWithBytes(requestBody)
	opTypeStr, _ := ec.GetString("opType")
	operandOne, _ := ec.GetBytes("operandOne")
	operandTwo, _ := ec.GetBytes("operandTwo")
	pubKeyBytes, _ := ec.GetBytes("pubKey")
	valuePtr, _ := ec.GetInt32("value_ptr")

	pubKey := new(paillier.PubKey)
	err := pubKey.Unmarshal(pubKeyBytes)
	if err != nil {
		return nil, err
	}

	if !isLen {
		copy(memory[valuePtr:valuePtr+int32(len(data))], data)
		return nil, nil
	}

	var resultBytes []byte
	switch opTypeStr {
	case protocol.PaillierOpTypeAddCiphertext:
		resultBytes, err = addCiphertext(operandOne, operandTwo, pubKey)
	case protocol.PaillierOpTypeAddPlaintext:
		resultBytes, err = addPlaintext(operandOne, operandTwo, pubKey)
	case protocol.PaillierOpTypeSubCiphertext:
		resultBytes, err = subCiphertext(operandOne, operandTwo, pubKey)
	case protocol.PaillierOpTypeSubPlaintext:
		resultBytes, err = subPlaintext(operandOne, operandTwo, pubKey)
	case protocol.PaillierOpTypeNumMul:
		resultBytes, err = numMul(operandOne, operandTwo, pubKey)
	default:
		return nil, errors.New("[paillier operate] the operate type is illegal, opTypeStr =" + opTypeStr)
	}

	if err != nil {
		return nil, err
	}

	copy(memory[valuePtr:valuePtr+4], bytehelper.IntToBytes(int32(len(resultBytes))))
	return resultBytes, nil
}

func addCiphertext(operandOne interface{}, operandTwo interface{}, pubKey *paillier.PubKey) ([]byte, error) {
	ct1 := new(paillier.Ciphertext)
	ct2 := new(paillier.Ciphertext)
	err := ct1.Unmarshal(operandOne.([]byte))
	if err != nil {
		return nil, err
	}
	err = ct2.Unmarshal(operandTwo.([]byte))
	if err != nil {
		return nil, err
	}

	result, err := pubKey.AddCiphertext(ct1, ct2)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, errors.New("[add ciphertext] operate failed")
	}
	return result.Marshal()
}

func addPlaintext(operandOne interface{}, operandTwo interface{}, pubKey *paillier.PubKey) ([]byte, error) {
	ct := new(paillier.Ciphertext)
	err := ct.Unmarshal(operandOne.([]byte))
	if err != nil {
		return nil, err
	}

	ptInt64, err := strconv.ParseInt(string(operandTwo.([]byte)), 10, 64)
	if err != nil {
		return nil, err
	}
	pt := new(big.Int).SetInt64(ptInt64)

	result, err := pubKey.AddPlaintext(ct, pt)
	if err != nil {
		return nil, err
	}

	return result.Marshal()
}

func subCiphertext(operandOne interface{}, operandTwo interface{}, pubKey *paillier.PubKey) ([]byte, error) {
	ct1 := new(paillier.Ciphertext)
	ct2 := new(paillier.Ciphertext)
	err := ct1.Unmarshal(operandOne.([]byte))
	if err != nil {
		return nil, err
	}
	err = ct2.Unmarshal(operandTwo.([]byte))
	if err != nil {
		return nil, err
	}
	result, err := pubKey.SubCiphertext(ct1, ct2)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, errors.New("[sub ciphertext] operate failed")
	}

	return result.Marshal()
}

func subPlaintext(operandOne interface{}, operandTwo interface{}, pubKey *paillier.PubKey) ([]byte, error) {
	ct := new(paillier.Ciphertext)
	err := ct.Unmarshal(operandOne.([]byte))
	if err != nil {
		return nil, err
	}

	ptInt64, err := strconv.ParseInt(string(operandTwo.([]byte)), 10, 64)
	if err != nil {
		return nil, err
	}
	pt := new(big.Int).SetInt64(ptInt64)

	result, err := pubKey.SubPlaintext(ct, pt)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, errors.New("[sub plaintext] operate failed")
	}

	return result.Marshal()
}

func numMul(operandOne interface{}, operandTwo interface{}, pubKey *paillier.PubKey) ([]byte, error) {
	ct := new(paillier.Ciphertext)
	err := ct.Unmarshal(operandOne.([]byte))
	if err != nil {
		return nil, err
	}

	ptInt64, err := strconv.ParseInt(string(operandTwo.([]byte)), 10, 64)
	if err != nil {
		return nil, err
	}
	pt := new(big.Int).SetInt64(ptInt64)

	result, err := pubKey.NumMul(ct, pt)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, errors.New("[num mul] operate failed")
	}

	return result.Marshal()
}

// ExecuteQuery execute query operation
func (w *WacsiImpl) ExecuteQuery(requestBody []byte, contractName string, txSimContext protocol.TxSimContext,
	memory []byte, chainId string) error {
	if !w.isSupportSql(txSimContext) {
		return fmt.Errorf("not support sql, you must set chainConfig[contract.enable_sql_support=true]")
	}
	ec := serialize.NewEasyCodecWithBytes(requestBody)
	sql, _ := ec.GetString("sql")
	ptr, _ := ec.GetInt32("value_ptr")
	// verify
	var err error
	if err = w.verifySql.VerifyDQLSql(sql); err != nil {
		return fmt.Errorf("[execute query] verify sql error, %s", err.Error())
	}
	// execute query
	var rows protocol.SqlRows
	if txSimContext.GetTx().Payload.TxType == common.TxType_QUERY_CONTRACT {
		rows, err = txSimContext.GetBlockchainStore().QueryMulti(contractName, sql)
		if err != nil {
			return fmt.Errorf("[execute query] error, %s", err.Error())
		}
	} else {
		txKey := common.GetTxKeyWith(txSimContext.GetBlockProposer().MemberInfo, txSimContext.GetBlockHeight())
		var transaction protocol.SqlDBTransaction
		transaction, err = txSimContext.GetBlockchainStore().GetDbTransaction(txKey)
		if err != nil {
			return fmt.Errorf("[execute query] get db transaction error, [%s]", err.Error())
		}
		dbName := txSimContext.GetBlockchainStore().GetContractDbName(contractName)
		changeCurrentDB(chainId, dbName, transaction)
		rows, err = transaction.QueryMulti(sql)
		if err != nil {
			return fmt.Errorf("[execute query] query multi error, [%s]", err.Error())
		}
	}

	index := atomic.AddInt32(&w.rowIndex, 1)
	txSimContext.SetIterHandle(index, rows)
	copy(memory[ptr:ptr+4], bytehelper.IntToBytes(index))
	return nil
}

// ExecuteQueryOne  query a record
func (w *WacsiImpl) ExecuteQueryOne(requestBody []byte, contractName string, txSimContext protocol.TxSimContext,
	memory []byte, data []byte, chainId string, isLen bool) ([]byte, error) {
	if !w.isSupportSql(txSimContext) {
		return nil, fmt.Errorf("not support sql, you must set chainConfig[contract.enable_sql_support=true]")
	}
	ec := serialize.NewEasyCodecWithBytes(requestBody)
	sql, _ := ec.GetString("sql")
	ptr, _ := ec.GetInt32("value_ptr")

	// verify
	if err := w.verifySql.VerifyDQLSql(sql); err != nil {
		return nil, fmt.Errorf("[execute query one] verify sql error, %s", err.Error())
	}

	// get len
	if isLen {
		// execute
		var row protocol.SqlRow
		var err error
		if txSimContext.GetTx().Payload.TxType == common.TxType_QUERY_CONTRACT {
			row, err = txSimContext.GetBlockchainStore().QuerySingle(contractName, sql)
			if err != nil {
				return nil, fmt.Errorf("[execute query one] error, %s", err.Error())
			}
		} else {
			txKey := common.GetTxKeyWith(txSimContext.GetBlockProposer().MemberInfo, txSimContext.GetBlockHeight())
			var transaction protocol.SqlDBTransaction
			transaction, err = txSimContext.GetBlockchainStore().GetDbTransaction(txKey)
			if err != nil {
				return nil, fmt.Errorf("[execute query one] get db transaction error, [%s]", err.Error())
			}
			dbName := txSimContext.GetBlockchainStore().GetContractDbName(contractName)
			changeCurrentDB(chainId, dbName, transaction)
			row, err = transaction.QuerySingle(sql)
			if err != nil {
				return nil, fmt.Errorf("[execute query one] query single error, [%s]", err.Error())
			}
		}
		var dataRow map[string][]byte
		if row.IsEmpty() {
			dataRow = make(map[string][]byte)
		} else {
			dataRow, err = row.Data()
			if err != nil {
				return nil, fmt.Errorf("[execute query one] ctx query get data to map error, %s", err.Error())
			}
		}
		ec := serialize.NewEasyCodecWithMap(dataRow)
		rsBytes := ec.Marshal()
		copy(memory[ptr:ptr+4], bytehelper.IntToBytes(int32(len(rsBytes))))
		if len(rsBytes) == 0 {
			return nil, nil
		}
		return rsBytes, nil
	}
	// get data
	if len(data) > 0 {
		copy(memory[ptr:ptr+int32(len(data))], data)
	}

	return nil, nil
}

// RSHasNext is used to judge whether there is a next record
func (w *WacsiImpl) RSHasNext(requestBody []byte, txSimContext protocol.TxSimContext, memory []byte) error {
	if !w.isSupportSql(txSimContext) {
		return fmt.Errorf("not support sql, you must set chainConfig[contract.enable_sql_support=true]")
	}
	ec := serialize.NewEasyCodecWithBytes(requestBody)
	rsIndex, _ := ec.GetInt32("rs_index")
	valuePtr, _ := ec.GetInt32("value_ptr")

	// get
	iter, ok := txSimContext.GetIterHandle(rsIndex)
	if !ok {
		return fmt.Errorf("[rs has next] ctx can not found rs_index[%d]", rsIndex)
	}

	rows, ok := iter.(protocol.SqlRows)
	if !ok {
		return fmt.Errorf("[rs has next] failed, rows %d assertion failed", rsIndex)
	}
	index := boolFalse
	if rows.Next() {
		index = boolTrue
	}
	copy(memory[valuePtr:valuePtr+4], bytehelper.IntToBytes(int32(index)))
	return nil
}

// RSNext get next record
func (w *WacsiImpl) RSNext(requestBody []byte, txSimContext protocol.TxSimContext, memory []byte, data []byte,
	isLen bool) ([]byte, error) {
	if !w.isSupportSql(txSimContext) {
		return nil, fmt.Errorf("not support sql, you must set chainConfig[contract.enable_sql_support=true]")
	}
	ec := serialize.NewEasyCodecWithBytes(requestBody)
	rsIndex, _ := ec.GetInt32("rs_index")
	ptr, _ := ec.GetInt32("value_ptr")

	// get handle
	iter, ok := txSimContext.GetIterHandle(rsIndex)
	if !ok {
		return nil, fmt.Errorf("[rs next] ctx can not found rs_index[%d]", rsIndex)
	}

	rows, ok := iter.(protocol.SqlRows)
	if !ok {
		return nil, fmt.Errorf("[rs next] failed, rows %d assertion failed", rsIndex)
	}

	// get len
	if isLen {
		var dataRow map[string][]byte
		var err error
		if rows == nil {
			dataRow = make(map[string][]byte)
		} else {
			dataRow, err = rows.Data()
			if err != nil {
				return nil, fmt.Errorf("[rs next] query next data error, %s", err.Error())
			}
		}
		ec := serialize.NewEasyCodecWithMap(dataRow)
		rsBytes := ec.Marshal()
		copy(memory[ptr:ptr+4], bytehelper.IntToBytes(int32(len(rsBytes))))
		if len(rsBytes) == 0 {
			return nil, nil
		}
		return rsBytes, nil
	}
	// get data
	if len(data) > 0 {
		copy(memory[ptr:ptr+int32(len(data))], data)
	}
	return nil, nil
}

// RSClose close sql iterator
func (w *WacsiImpl) RSClose(requestBody []byte, txSimContext protocol.TxSimContext, memory []byte) error {
	if !w.isSupportSql(txSimContext) {
		return fmt.Errorf("not support sql, you must set chainConfig[contract.enable_sql_support=true]")
	}
	ec := serialize.NewEasyCodecWithBytes(requestBody)
	rsIndex, _ := ec.GetInt32("rs_index")
	valuePtr, _ := ec.GetInt32("value_ptr")

	// get
	iter, ok := txSimContext.GetIterHandle(rsIndex)
	if !ok {
		return fmt.Errorf("[rs close] ctx can not found rs_index[%d]", rsIndex)
	}

	rows, ok := iter.(protocol.SqlRows)
	if !ok {
		return fmt.Errorf("[rs close] failed, rows %d assertion failed", rsIndex)
	}

	var index int32 = 1
	if err := rows.Close(); err != nil {
		return fmt.Errorf("[rs close] close rows error, [%s]", err.Error())
	}
	copy(memory[valuePtr:valuePtr+4], bytehelper.IntToBytes(index))
	return nil
}

// ExecuteUpdate execute udpate
func (w *WacsiImpl) ExecuteUpdate(requestBody []byte, contractName string, method string,
	txSimContext protocol.TxSimContext, memory []byte, chainId string) error {
	if txSimContext.GetTx().Payload.TxType == common.TxType_QUERY_CONTRACT {
		return fmt.Errorf("[execute update] query transaction cannot be execute dml")
	}
	if method == protocol.ContractUpgradeMethod {
		return fmt.Errorf("[execute update] upgrade contract transaction cannot be execute dml")
	}
	if !w.isSupportSql(txSimContext) {
		return fmt.Errorf("not support sql, you must set chainConfig[contract.enable_sql_support=true]")
	}
	ec := serialize.NewEasyCodecWithBytes(requestBody)
	sql, _ := ec.GetString("sql")
	ptr, _ := ec.GetInt32("value_ptr")

	// verify
	if err := w.verifySql.VerifyDMLSql(sql); err != nil {
		return fmt.Errorf("[execute update] verify update sql error, [%s], sql [%s]", err.Error(), sql)
	}

	txKey := common.GetTxKeyWith(txSimContext.GetBlockProposer().MemberInfo, txSimContext.GetBlockHeight())
	transaction, err := txSimContext.GetBlockchainStore().GetDbTransaction(txKey)
	if err != nil {
		return fmt.Errorf("[execute update] ctx get db transaction error, [%s]", err.Error())
	}

	// execute
	dbName := txSimContext.GetBlockchainStore().GetContractDbName(contractName)
	changeCurrentDB(chainId, dbName, transaction)
	affectedCount, err := transaction.ExecSql(sql)
	if err != nil {
		return fmt.Errorf("[execute update] execute error, [%s], sql[%s]", err.Error(), sql)
	}
	txSimContext.PutRecord(contractName, []byte(sql), protocol.SqlTypeDml)
	copy(memory[ptr:ptr+4], bytehelper.IntToBytes(int32(affectedCount)))
	return nil
}

// ExecuteDDL execute DDL statement
func (w *WacsiImpl) ExecuteDDL(requestBody []byte, contractName string, txSimContext protocol.TxSimContext,
	memory []byte, method string) error {
	if !w.isSupportSql(txSimContext) {
		return fmt.Errorf("not support sql, you must set chainConfig[contract.enable_sql_support=true]")
	}
	if !w.isManageContract(method) {
		return ErrorNotManageContract
	}
	ec := serialize.NewEasyCodecWithBytes(requestBody)
	sql, _ := ec.GetString("sql")
	ptr, _ := ec.GetInt32("value_ptr")

	// verify
	if err := w.verifySql.VerifyDDLSql(sql); err != nil {
		return fmt.Errorf("[execute ddl] verify ddl sql error,  [%s], sql[%s]", err.Error(), sql)
	}
	if err := txSimContext.GetBlockchainStore().ExecDdlSql(contractName, sql, "1"); err != nil {
		return fmt.Errorf("[execute ddl] execute error, %s, sql[%s]", err.Error(), sql)
	}
	txSimContext.PutRecord(contractName, []byte(sql), protocol.SqlTypeDdl)
	copy(memory[ptr:ptr+4], bytehelper.IntToBytes(0))
	return nil
}

func (w *WacsiImpl) isManageContract(method string) bool {
	return method == protocol.ContractInitMethod || method == protocol.ContractUpgradeMethod
}

func changeCurrentDB(chainId string, dbName string, transaction protocol.SqlDBTransaction) {
	//currentDbName := getCurrentDb(chainId)
	//if contractName != "" && dbName != currentDbName {
	_ = transaction.ChangeContextDb(dbName)
	//setCurrentDb(chainId, dbName)
	//}
}
func (w *WacsiImpl) isSupportSql(txSimContext protocol.TxSimContext) bool {
	w.lock.Lock()
	defer w.lock.Unlock()
	chainId := txSimContext.GetTx().Payload.ChainId
	if b, ok := w.enableSql[chainId]; ok {
		return b
	}
	cc, _ := txSimContext.GetBlockchainStore().GetLastChainConfig()
	w.enableSql[chainId] = cc.Contract.EnableSqlSupport
	return cc.Contract.EnableSqlSupport
}
