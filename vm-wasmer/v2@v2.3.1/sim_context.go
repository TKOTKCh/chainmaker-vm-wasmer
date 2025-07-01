/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package wasmer

import (
	"fmt"
	"strconv"
	"sync"

	"chainmaker.org/chainmaker/logger/v2"

	"chainmaker.org/chainmaker/common/v2/serialize"
	commonPb "chainmaker.org/chainmaker/pb-go/v2/common"
	"chainmaker.org/chainmaker/protocol/v2"
	"chainmaker.org/chainmaker/vm-wasmer/v2/wasmer-go"
)

// SimContext record the contract context
type SimContext struct {
	TxSimContext   protocol.TxSimContext
	Contract       *commonPb.Contract
	ContractResult *commonPb.ContractResult
	Log            *logger.CMLogger
	Instance       *wasmer.Instance

	method             string
	parameters         map[string][]byte
	CtxPtr             int32
	SenderAddressCache []byte
	GetStateCache      []byte // cache call method GetStateLen value result, one cache per transaction
	ChainId            string
	ContractEvent      []*commonPb.ContractEvent
	SpecialTxType      protocol.ExecOrderTxType
}

// NewSimContext for every transaction
func NewSimContext(method string, log *logger.CMLogger, chainId string) *SimContext {
	sc := SimContext{
		method:  method,
		Log:     log,
		ChainId: chainId,
	}

	sc.putCtxPointer()

	return &sc
}

// CallMethod will call contract method
func (sc *SimContext) CallMethod(instance *wasmer.Instance) error {
	var bytes []byte

	runtimeFn, err := instance.Exports.GetRawFunction(protocol.ContractRuntimeTypeMethod)
	if err != nil {
		return fmt.Errorf("method [%s] not export, err = %v", protocol.ContractRuntimeTypeMethod, err)
	}
	defer runtimeFn.Close()

	sdkType, err := runtimeFn.Call()
	if err != nil {
		return err
	}

	runtimeSdkType, ok := sdkType.(int32)
	if !ok {
		return fmt.Errorf("sdkType is not int32 type")
	}
	//长安链原本go的wasm程序用的是GASM（wazero）跑的，这里是个语言和运行时检查，我添加了int32(commonPb.RuntimeType_GASM) == runtimeSdkType
	//让wasmer也可以跑go的wasm程序
	if int32(commonPb.RuntimeType_WASMER) == runtimeSdkType || int32(commonPb.RuntimeType_GASM) == runtimeSdkType {
		sc.parameters[protocol.ContractContextPtrParam] = []byte(strconv.Itoa(int(sc.CtxPtr)))
		//这个CONTRACT_BYTECODE在合约部署阶段会把这个当作上下文参数传过来，由于标准go编译的wasm太长了，导致上下文创建不出来，为此手动把这个参数设置为空
		//rust、tinygo没有这个问题是因为他们的字节码比较短
		sc.parameters["CONTRACT_BYTECODE"] = []byte("")
		ec := serialize.NewEasyCodecWithMap(sc.parameters)
		bytes = ec.Marshal()
	} else {
		return fmt.Errorf("runtime type error, expect rust:[%d], but got %d",
			uint64(commonPb.RuntimeType_WASMER), runtimeSdkType)
	}

	return sc.callContract(instance, sc.method, bytes)
}

func (sc *SimContext) callContract(instance *wasmer.Instance, methodName string, bytes []byte) error {

	//sc.Log.Debugf("sc.Contract = %v", sc.Contract)
	//sc.Log.Debugf("sc.method = %v", sc.method)
	//sc.Log.Debugf("sc.parameters = %v", sc.parameters)

	lengthOfSubject := len(bytes)

	allocateFunc, err := instance.Exports.GetRawFunction(protocol.ContractAllocateMethod)
	if err != nil {
		return fmt.Errorf("method [%s] not export, err = %v", protocol.ContractAllocateMethod, err)
	}
	defer allocateFunc.Close()

	// Allocate memory for the subject, and get a pointer to it.
	allocateResult, err := allocateFunc.Call(lengthOfSubject)
	if err != nil {
		sc.Log.Errorf("contract invoke %s failed, %s", protocol.ContractAllocateMethod, err.Error())
		return fmt.Errorf("%s invoke failed. There may not be enough memory or CPU", protocol.ContractAllocateMethod)
	}
	dataPtr, ok := allocateResult.(int32)
	if !ok {
		return fmt.Errorf("allocateResult is not int32 type")
	}

	// Write the subject into the memory.
	exportMemory, err := instance.Exports.GetMemory("memory")
	if err != nil {
		return fmt.Errorf("[%s] can't get exported memory, err = %v", protocol.ContractAllocateMethod, err)
	}
	memory := exportMemory.Data()[dataPtr:]

	//copy(memory, bytes)
	for nth := 0; nth < lengthOfSubject; nth++ {
		memory[nth] = bytes[nth]
	}

	// Calls the `invoke` exported function. Given the pointer to the subject.
	exportFunc, err := instance.Exports.GetRawFunction(methodName)
	if err != nil {
		// add compatibility for wasmer-1.0
		if sc.TxSimContext.GetBlockVersion() < 2200 {
			return fmt.Errorf("method [%s] not export", methodName)
		}
		return fmt.Errorf("find method [%s] failed, err = %v", methodName, err)
	}
	defer exportFunc.Close()

	_, err = exportFunc.Call()
	if err != nil {
		return err
	}
	sc.Log.Infof("contract invoke success")
	return err
}

// CallDeallocate deallocate vm memory before closing the instance
func CallDeallocate(instance *wasmer.Instance) error {
	instance.SetGasLimit(protocol.GasLimit)
	deallocFunc, err := instance.Exports.GetFunction(protocol.ContractDeallocateMethod)
	if err != nil {
		return err
	}
	_, err = deallocFunc(0)
	return err
}

// putCtxPointer revmoe SimContext from cache
func (sc *SimContext) removeCtxPointer() {
	vbm := GetVmBridgeManager()
	vbm.remove(sc.CtxPtr)
}

var ctxIndex = int32(0)
var lock sync.Mutex

// putCtxPointer save SimContext to cache
func (sc *SimContext) putCtxPointer() {
	lock.Lock()
	ctxIndex++
	if ctxIndex > 1e8 {
		ctxIndex = 0
	}
	sc.CtxPtr = ctxIndex
	lock.Unlock()
	vbm := GetVmBridgeManager()
	vbm.put(sc.CtxPtr, sc)
}
