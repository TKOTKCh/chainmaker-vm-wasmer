/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

// Package protocol is a protocol package, which is base.
package protocol

import (
	"bytes"
	"fmt"
	"regexp"

	pbac "chainmaker.org/chainmaker/pb-go/v2/accesscontrol"
	"chainmaker.org/chainmaker/pb-go/v2/common"
	"chainmaker.org/chainmaker/pb-go/v2/config"
	vmPb "chainmaker.org/chainmaker/pb-go/v2/vm"
)

// constant data for VM
const (
	GasLimit            = 1e10    // invoke user contract max gas
	TimeLimit           = 1 * 1e9 // 1s
	CallContractGasOnce = 1e5     // Gas consumed per cross call contract
	CallContractDepth   = 5       // cross call contract stack depth, must less than vm pool min size
	EvmGasPrice         = 1
	EvmMaxStackDepth    = 1024

	ContractSdkSignalResultSuccess = 0 // sdk call chain method success result
	ContractSdkSignalResultFail    = 1 // sdk call chain method success result

	DefaultMaxStateKeyLen = 1024                // key & name for contract state length
	DefaultStateRegex     = "^[a-zA-Z0-9._-]+$" // key & name for contract state regex

	DefaultVersionLen   = 64                  // key & name for contract state length
	DefaultVersionRegex = "^[a-zA-Z0-9._-]+$" // key & name for contract state regex

	ParametersKeyMaxCount         = 50 //
	DefaultParametersValueMaxSize = 10 // default size transaction's per parameter (MB)

	TopicMaxLen       = 255
	EventDataMaxLen   = 65535
	EventDataMaxCount = 16

	ContractStoreSeparator = "#"

	// special parameters passed to contract

	ContractCreatorOrgIdParam = "__creator_org_id__"
	ContractCreatorRoleParam  = "__creator_role__"
	ContractCreatorPkParam    = "__creator_pk__"
	ContractSenderOrgIdParam  = "__sender_org_id__"
	ContractSenderRoleParam   = "__sender_role__"
	ContractSenderPkParam     = "__sender_pk__"
	ContractBlockHeightParam  = "__block_height__"
	ContractTxIdParam         = "__tx_id__"
	ContractTxTimeStamp       = "__tx_time_stamp__"
	ContractContextPtrParam   = "__context_ptr__"
	ContractAddrTypeParam     = "__addr_type__"
	ContractSenderTypeParam   = "__sender_type__"
	ContractCreatorTypeParam  = "__creator_type__"
	ContractCrossCallerParam  = "__cross_caller__"

	// user contract must implement such method

	ContractInitMethod        = "init_contract"
	ContractUpgradeMethod     = "upgrade"
	ContractAllocateMethod    = "allocate"
	ContractDeallocateMethod  = "deallocate"
	ContractRuntimeTypeMethod = "runtime_type"
	ContractEvmParamKey       = "data"
	// method name used by smart contract sdk

	// common

	ContractMethodLogMessage      = "LogMessage"
	ContractMethodSuccessResult   = "SuccessResult"
	ContractMethodErrorResult     = "ErrorResult"
	ContractMethodCallContract    = "CallContract"
	ContractMethodCallContractLen = "CallContractLen"

	// kv

	ContractMethodGetStateLen = "GetStateLen"
	ContractMethodGetState    = "GetState"
	ContractMethodPutState    = "PutState"
	ContractMethodDeleteState = "DeleteState"

	ContractMethodGetBatchStateLen = "GetBatchStateLen"
	ContractMethodGetBatchState    = "GetBatchState"
	//address
	ContractMethodSenderAddress    = "GetSenderAddress"
	ContractMethodSenderAddressLen = "GetSenderAddressLen"

	// kv iterator author:whang1234

	ContractMethodKvIterator        = "KvIterator"
	ContractMethodKvPreIterator     = "KvPreIterator"
	ContractMethodKvIteratorHasNext = "KvIteratorHasNext"
	ContractMethodKvIteratorNextLen = "KvIteratorNextLen"
	ContractMethodKvIteratorNext    = "KvIteratorNext"
	ContractMethodKvIteratorClose   = "KvIteratorClose"

	// keyHistoryKvIterator method
	ContractHistoryKvIterator        = "HistoryKvIterator"
	ContractHistoryKvIteratorHasNext = "HistoryKvIterHasNext"
	ContractHistoryKvIteratorNextLen = "HistoryKvIterNextLen"
	ContractHistoryKvIteratorNext    = "HistoryKvIterNext"
	ContractHistoryKvIteratorClose   = "HistoryKvIterClose"

	//lib
	ContractMethodSha256 = "Sha256"

	// sql

	ContractMethodExecuteQuery       = "ExecuteQuery"
	ContractMethodExecuteQueryOne    = "ExecuteQueryOne"
	ContractMethodExecuteQueryOneLen = "ExecuteQueryOneLen"
	ContractMethodRSNext             = "RSNext"
	ContractMethodRSNextLen          = "RSNextLen"
	ContractMethodRSHasNext          = "RSHasNext"
	ContractMethodRSClose            = "RSClose"
	ContractMethodExecuteUpdate      = "ExecuteUpdate"
	ContractMethodExecuteDdl         = "ExecuteDDL"
	ContractMethodEmitEvent          = "EmitEvent"

	// paillier

	ContractMethodGetPaillierOperationResult    = "GetPaillierOperationResult"
	ContractMethodGetPaillierOperationResultLen = "GetPaillierOperationResultLen"
	PaillierOpTypeAddCiphertext                 = "AddCiphertext"
	PaillierOpTypeAddPlaintext                  = "AddPlaintext"
	PaillierOpTypeSubCiphertext                 = "SubCiphertext"
	PaillierOpTypeSubPlaintext                  = "SubPlaintext"
	PaillierOpTypeNumMul                        = "NumMul"

	// bulletproofs

	ContractMethodGetBulletproofsResult     = "GetBulletproofsResult"
	ContractMethodGetBulletproofsResultLen  = "GetBulletproofsResultLen"
	BulletProofsOpTypePedersenAddNum        = "PedersenAddNum"
	BulletProofsOpTypePedersenAddCommitment = "PedersenAddCommitment"
	BulletProofsOpTypePedersenSubNum        = "PedersenSubNum"
	BulletProofsOpTypePedersenSubCommitment = "PedersenSubCommitment"
	BulletProofsOpTypePedersenMulNum        = "PedersenMulNum"
	BulletProofsVerify                      = "BulletproofsVerify"

	//Log level
	DebugLevel = -1
	InfoLevel  = 0
	WarnLevel  = 1
	ErrorLevel = 2
)

var (
	//ParametersValueMaxLength 参数Value允许的最大长度
	ParametersValueMaxLength uint32
)

// ExecOrderTxType 执行排序类型
type ExecOrderTxType int

// ExecOrderTxType list
const (
	ExecOrderTxTypeNormal ExecOrderTxType = iota
	ExecOrderTxTypeIterator
	ExecOrderTxTypeChargeGas
)

// SqlType sql语句的类型
type SqlType int8

// Sql types: ddl,dml,dql
const (
	SqlTypeDdl SqlType = iota
	SqlTypeDml
	SqlTypeDql
)

// VmManager manage vm runtime
type VmManager interface {
	// GetAccessControl get accessControl manages policies and principles
	GetAccessControl() AccessControlProvider
	// GetChainNodesInfoProvider get ChainNodesInfoProvider provide base node info list of chain.
	GetChainNodesInfoProvider() ChainNodesInfoProvider
	// RunContract run native or user contract according ContractName in contractId, and call the specified function
	RunContract(contract *common.Contract, method string, byteCode []byte, parameters map[string][]byte,
		txContext TxSimContext, gasUsed uint64, refTxType common.TxType) (
		*common.ContractResult, ExecOrderTxType, common.TxStatusCode)
	// Start all vm instance
	Start() error
	// Stop all vm instance
	Stop() error

	BeforeSchedule(blockFingerprint string, blockHeight uint64)

	AfterSchedule(blockFingerprint string, blockHeight uint64)
}

// RuntimeInstance of smart contract engine runtime
type RuntimeInstance interface {
	// start vm runtime with invoke, call “method”
	Invoke(contractId *common.Contract, method string, byteCode []byte, parameters map[string][]byte,
		txContext TxSimContext, gasUsed uint64) (*common.ContractResult, ExecOrderTxType)
}

// VmInstancesManager VM实例的管理接口
type VmInstancesManager interface {
	NewRuntimeInstance(txSimContext TxSimContext, chainId, method, codePath string, contract *common.Contract,
		byteCode []byte, log Logger) (RuntimeInstance, error)
	// StartVM Start vm
	StartVM() error
	// StopVM Stop vm
	StopVM() error

	BeforeSchedule(blockFingerprint string, blockHeight uint64)

	AfterSchedule(blockFingerprint string, blockHeight uint64)
}

// ContractWacsiCommon 合约提供的公共接口
type ContractWacsiCommon interface {
	LogMessage() int32
	SuccessResult() int32
	ErrorResult() int32
	CallContract() int32
}

// ContractWacsiKV 合约在支持KV模式下的接口
type ContractWacsiKV interface {
	ContractWacsiCommon
	GetState() int32
	PutState() int32
	DeleteState() int32
	KvIterator() int32
	KvPreIterator() int32
	KvIteratorClose() int32
	KvIteratorNext() int32
	KvIteratorHasNext() int32
}

// ContractWacsiSQL 合约支持SQL模式下的接口
type ContractWacsiSQL interface {
	ContractWacsiCommon
	ExecuteQuery() int32
	ExecuteQueryOne() int32
	RSHasNext() int32
	RSNext() int32
	RSClose() int32
	ExecuteUpdate() int32
	ExecuteDDL() int32
}

// Wacsi WebAssembly chainmaker system interface
type Wacsi interface {
	// state operation
	PutState(requestBody []byte, contractName string, txSimContext TxSimContext) error
	GetState(requestBody []byte, contractName string, txSimContext TxSimContext, memory []byte,
		data []byte, isLen bool) ([]byte, error)
	GetBatchState(requestBody []byte, contractName string, txSimContext TxSimContext, memory []byte,
		data []byte, isLen bool) ([]byte, error)

	DeleteState(requestBody []byte, contractName string, txSimContext TxSimContext) error
	// call other contract
	CallContract(caller *common.Contract, requestBody []byte, txSimContext TxSimContext, memory []byte, data []byte,
		gasUsed uint64, isLen bool) (*common.ContractResult, uint64, ExecOrderTxType, error)
	// result record
	SuccessResult(contractResult *common.ContractResult, data []byte) int32
	ErrorResult(contractResult *common.ContractResult, data []byte) int32
	// emit event
	EmitEvent(requestBody []byte, txSimContext TxSimContext, contractId *common.Contract,
		log Logger) (*common.ContractEvent, error)
	// paillier
	PaillierOperation(requestBody []byte, memory []byte, data []byte, isLen bool) ([]byte, error)
	// bulletproofs
	BulletProofsOperation(requestBody []byte, memory []byte, data []byte, isLen bool) ([]byte, error)

	// kv iterator
	KvIterator(requestBody []byte, contractName string, txSimContext TxSimContext, memory []byte) error
	KvPreIterator(requestBody []byte, contractName string, txSimContext TxSimContext, memory []byte) error
	KvIteratorHasNext(requestBody []byte, txSimContext TxSimContext, memory []byte) error
	KvIteratorNext(requestBody []byte, txSimContext TxSimContext, memory []byte, data []byte,
		contractName string, isLen bool) ([]byte, error)
	KvIteratorClose(requestBody []byte, contractName string, txSimContext TxSimContext, memory []byte) error

	// History kv iterator
	HistoryKvIterator(requestBody []byte, contractName string, txSimContext TxSimContext, memory []byte) error
	HistoryKvIterHasNext(requestBody []byte, txSimContext TxSimContext, memory []byte) error
	HistoryKvIterNext(requestBody []byte, txSimContext TxSimContext, memory []byte, data []byte,
		contractName string, isLen bool) ([]byte, error)
	HistoryKvIterClose(requestBody []byte, contractName string, txSimContext TxSimContext, memory []byte) error

	// sql operation
	ExecuteQuery(requestBody []byte, contractName string, txSimContext TxSimContext, memory []byte,
		chainId string) error
	ExecuteQueryOne(requestBody []byte, contractName string, txSimContext TxSimContext, memory []byte,
		data []byte, chainId string, isLen bool) ([]byte, error)
	ExecuteUpdate(requestBody []byte, contractName string, method string, txSimContext TxSimContext,
		memory []byte, chainId string) error
	ExecuteDDL(requestBody []byte, contractName string, txSimContext TxSimContext, memory []byte,
		method string) error
	RSHasNext(requestBody []byte, txSimContext TxSimContext, memory []byte) error
	RSNext(requestBody []byte, txSimContext TxSimContext, memory []byte, data []byte,
		isLen bool) ([]byte, error)
	RSClose(requestBody []byte, txSimContext TxSimContext, memory []byte) error

	Sha256(requestBody []byte, contractName string, memory []byte) ([]byte, error)
	NativeSha256(hashInput []byte) [32]byte
	GetSenderAddress(requestBody []byte, contractName string, txSimContext TxSimContext, memory []byte,
		data []byte, isLen bool) ([]byte, error)
}

// WacsiWithGas WebAssembly chainmaker system interface
type WacsiWithGas interface {
	LogMessage(requestBody []byte, txSimContext TxSimContext) int32
	// state operation
	PutState(requestBody []byte, contractName string, txSimContext TxSimContext) error
	GetState(requestBody []byte, contractName string, txSimContext TxSimContext, memory []byte,
		data []byte, isLen bool) ([]byte, error)
	DeleteState(requestBody []byte, contractName string, txSimContext TxSimContext) error
	// call other contract
	CallContract(caller *common.Contract, requestBody []byte, txSimContext TxSimContext, memory []byte, data []byte,
		gasUsed uint64, isLen bool) (*common.ContractResult, uint64, ExecOrderTxType, error)
	// result record
	SuccessResult(contractResult *common.ContractResult, txSimContext TxSimContext, data []byte) int32
	ErrorResult(contractResult *common.ContractResult, txSimContext TxSimContext, data []byte) int32
	// emit event
	EmitEvent(requestBody []byte, txSimContext TxSimContext, contractId *common.Contract,
		log Logger) (*common.ContractEvent, error)
	// paillier
	PaillierOperation(
		requestBody []byte, txSimContext TxSimContext, memory []byte, data []byte, isLen bool) ([]byte, error)
	// bulletproofs
	BulletProofsOperation(
		requestBody []byte, txSimContext TxSimContext, memory []byte, data []byte, isLen bool) ([]byte, error)

	// kv iterator
	KvIterator(requestBody []byte, contractName string, txSimContext TxSimContext, memory []byte) error
	KvPreIterator(requestBody []byte, contractName string, txSimContext TxSimContext, memory []byte) error
	KvIteratorHasNext(requestBody []byte, txSimContext TxSimContext, memory []byte) error
	KvIteratorNext(requestBody []byte, txSimContext TxSimContext, memory []byte, data []byte,
		contractName string, isLen bool) ([]byte, error)
	KvIteratorClose(requestBody []byte, contractName string, txSimContext TxSimContext, memory []byte) error

	// sql operation
	ExecuteQuery(requestBody []byte, contractName string, txSimContext TxSimContext, memory []byte,
		chainId string) error
	ExecuteQueryOne(requestBody []byte, contractName string, txSimContext TxSimContext, memory []byte,
		data []byte, chainId string, isLen bool) ([]byte, error)
	ExecuteUpdate(requestBody []byte, contractName string, method string, txSimContext TxSimContext,
		memory []byte, chainId string) error
	ExecuteDDL(requestBody []byte, contractName string, txSimContext TxSimContext, memory []byte,
		method string) error
	RSHasNext(requestBody []byte, txSimContext TxSimContext, memory []byte) error
	RSNext(requestBody []byte, txSimContext TxSimContext, memory []byte, data []byte,
		isLen bool) ([]byte, error)
	RSClose(requestBody []byte, txSimContext TxSimContext, memory []byte) error
}

// GetKeyStr get state key from string
func GetKeyStr(key string, field string) []byte {
	return GetKey([]byte(key), []byte(field))
}

// GetKey get state key from byte
func GetKey(key []byte, field []byte) []byte {
	var buf bytes.Buffer
	buf.Write(key)
	if len(field) > 0 {
		buf.Write([]byte(ContractStoreSeparator))
		buf.Write(field)
	}
	return buf.Bytes()
}

// CheckKeyFieldStr verify param
func CheckKeyFieldStr(key string, field string) error {
	{
		s := key
		if len(s) > DefaultMaxStateKeyLen {
			return fmt.Errorf("key[%s] too long", s)
		}
		match, err := regexp.MatchString(DefaultStateRegex, s)
		if err != nil || !match {
			return fmt.Errorf("key[%s] can only consist of numbers, dot, letters and underscores", s)
		}
	}
	{
		s := field
		if len(s) == 0 {
			return nil
		}
		if len(s) > DefaultMaxStateKeyLen {
			return fmt.Errorf("key field[%s] too long", s)
		}
		match, err := regexp.MatchString(DefaultStateRegex, s)
		if err != nil || !match {
			return fmt.Errorf("key field[%s] can only consist of numbers, dot, letters and underscores", s)
		}
	}
	return nil
}

// CheckTopicStr check topic length
func CheckTopicStr(topic string) error {
	topicLen := len(topic)
	if topicLen == 0 {
		return fmt.Errorf("topic can not empty")
	}
	if topicLen > TopicMaxLen {
		return fmt.Errorf("topic too long,longer than %v", TopicMaxLen)
	}
	return nil

}

// CheckEventData verify event data
func CheckEventData(eventData []string) error {

	eventDataNum := len(eventData)
	if eventDataNum == 0 {
		return fmt.Errorf("event data can not empty")

	}
	if eventDataNum > EventDataMaxCount {
		return fmt.Errorf("too many event data")
	}
	for _, data := range eventData {
		if len(data) > EventDataMaxLen {
			return fmt.Errorf("event data too long,longer than %v", EventDataMaxLen)
		}
	}
	return nil

}

// TxSimContext The simulated execution context of the transaction,
// providing a cache for the transaction to read and write
type TxSimContext interface {
	// Get key from cache, record this operation to read set
	Get(contractName string, key []byte) ([]byte, error)
	// GetKeys key from cache, record this operation to read set
	GetKeys(keys []*vmPb.BatchKey) ([]*vmPb.BatchKey, error)
	//GetNoRecord read data from state, but not record into read set, only used for framework
	GetNoRecord(contractName string, key []byte) ([]byte, error)
	// GetSnapshot get the snapshot in TxSimContext, just for calculating gas
	GetSnapshot() Snapshot
	// Put key into cache
	Put(name string, key []byte, value []byte) error
	// PutRecord put sql state into cache
	PutRecord(contractName string, value []byte, sqlType SqlType)
	// PutIntoReadSet put kv to readset
	PutIntoReadSet(contractName string, key []byte, value []byte)
	// Del Delete key from cache
	Del(name string, key []byte) error
	// Select range query for key [start, limit)
	Select(name string, startKey []byte, limit []byte) (StateIterator, error)
	// GetHistoryIterForKey query the change history of a key in a contract
	GetHistoryIterForKey(contractName string, key []byte) (KeyHistoryIterator, error)
	// CallContract Cross contract call, return (contract result, gas used)
	CallContract(caller, contract *common.Contract, method string, byteCode []byte,
		parameter map[string][]byte, gasUsed uint64, refTxType common.TxType) (
		*common.ContractResult, ExecOrderTxType, common.TxStatusCode)
	// GetCurrentResult Get cross contract call result, cache for len
	GetCurrentResult() []byte
	// GetTx get related transaction
	GetTx() *common.Transaction
	// GetBlockHeight returns current block height
	GetBlockHeight() uint64
	// GetBlockFingerprint returns unique id for block
	GetBlockFingerprint() string
	// GetBlockTimestamp returns current block timestamp
	GetBlockTimestamp() int64
	// GetBlockProposer returns current block proposer
	GetBlockProposer() *pbac.Member
	// GetTxResult returns the tx result
	GetTxResult() *common.Result
	// SetTxResult set the tx result
	SetTxResult(*common.Result)
	// GetTxRWSet returns the read and write set completed by the current transaction
	GetTxRWSet(runVmSuccess bool) *common.TxRWSet
	// GetCreator returns the creator of the contract
	GetCreator(namespace string) *pbac.Member
	// GetSender returns the invoker of the transaction
	GetSender() *pbac.Member
	// GetBlockchainStore returns related blockchain store
	GetBlockchainStore() BlockchainStore
	// GetLastChainConfig returns last chain config
	GetLastChainConfig() *config.ChainConfig
	// GetAccessControl returns access control service
	GetAccessControl() (AccessControlProvider, error)
	// GetChainNodesInfoProvider returns organization service
	GetChainNodesInfoProvider() (ChainNodesInfoProvider, error)
	// The execution sequence of the transaction, used to construct the dag,
	// indicating the number of completed transactions during transaction scheduling
	GetTxExecSeq() int
	SetTxExecSeq(int)
	// Get cross contract call deep
	GetDepth() int
	SetIterHandle(index int32, iter interface{})
	GetIterHandle(index int32) (interface{}, bool)
	GetBlockVersion() uint32
	//GetContractByName get contract info by name
	GetContractByName(name string) (*common.Contract, error)
	//GetContractBytecode get contract bytecode
	GetContractBytecode(name string) ([]byte, error)
	// GetTxRWMapByContractName get the read-write map of the specified contract of the current transaction
	GetTxRWMapByContractName(contractName string) (map[string]*common.TxRead, map[string]*common.TxWrite)
	// GetCrossInfo get contract call link information
	GetCrossInfo() uint64
	// HasUsed judge whether the specified common.RuntimeType has appeared in the previous depth
	// in the current cross-link
	HasUsed(runtimeType common.RuntimeType) bool
	// RecordRuntimeTypeIntoCrossInfo record the new contract call information to the top of crossInfo
	RecordRuntimeTypeIntoCrossInfo(runtimeType common.RuntimeType)
	// RemoveRuntimeTypeFromCrossInfo remove the top-level information from the crossInfo
	RemoveRuntimeTypeFromCrossInfo()
	// GetStrAddrFromPbMember calculate string address from pb Member
	GetStrAddrFromPbMember(pbMember *pbac.Member) (string, error)
	// SubtractGas charge gas used for this tx
	SubtractGas(gasUsed uint64) error
	// GetGasRemaining return gas remaining for this tx
	GetGasRemaining() uint64
}
