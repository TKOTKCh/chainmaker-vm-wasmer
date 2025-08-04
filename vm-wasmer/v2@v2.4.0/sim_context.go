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
	//runtimeSdkType := int32(2)
	//长安链原本go的wasm程序用的是GASM（wazero）跑的，这里是个语言和运行时检查，我添加了int32(commonPb.RuntimeType_GASM) == runtimeSdkType
	//让wasmer也可以跑go的wasm程序
	if int32(commonPb.RuntimeType_WASMER) == runtimeSdkType || int32(commonPb.RuntimeType_GASM) == runtimeSdkType {

		//这个ctxptr是递增的，导致后面ec.Marshal序列化长度不一样，从而导致原本后面在allocateFunc.Call(lengthOfSubject)把参数写入内存的gas计量会有小误差
		//sc.Log.Debugf("length of ctx_ptr %d", len(sc.parameters[protocol.ContractContextPtrParam]))
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

	sc.Log.Debugf("sc.Contract = %v", sc.Contract)
	sc.Log.Debugf("sc.method = %v", sc.method)
	//sc.Log.Debugf("sc.parameters = %v", sc.parameters)

	//不知道为什么反复调用情况下，这个lengthOfSubject并不是一个固定值，会有一些大小差异，导致allocateFunc.Call(lengthOfSubject)这个函数它的gas开销有所出入
	//如果按照之前的方法在runtime.go中进行插桩的话对gas结果存在影响，是否应该对我们需要的exportFunc进行插桩？
	//已解决，有个参数ctxptr是递增的，导致后面ec.Marshal序列化长度不一样，从而导致原本后面在allocateFunc.Call(lengthOfSubject)把参数写入内存的gas计量会有小误差
	lengthOfSubject := len(bytes)

	allocateFunc, err := instance.Exports.GetRawFunction(protocol.ContractAllocateMethod)
	if err != nil {
		return fmt.Errorf("method [%s] not export, err = %v", protocol.ContractAllocateMethod, err)
	}
	defer allocateFunc.Close()
	sc.Log.Debugf("lengthOfSubject: %d", lengthOfSubject)

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
	// 经测试instance.Exports.GetMemory("memory")并不会影响开销，应该只有Call的会影响
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
	gasLimit := uint64(1e15)
	instance.SetGasLimit(gasLimit - 0)
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
	sc.Log.Debugf("contract invoke success")

	return err
}

// CallDeallocate deallocate vm memory before closing the instance
func CallDeallocate(instance *wasmer.Instance) error {
	// TODO 这里setgaslimit是否会影响gas计量？
	instance.SetGasLimit(protocol.GasLimit)
	// TODO 这里deallocate，主要是因为原本做法参数是写在合约实例的全局变量，这里把合约实例的参数argsmap设置为空
	deallocFunc, err := instance.Exports.GetFunction(protocol.ContractDeallocateMethod)
	if err != nil {
		return err
	}
	_, err = deallocFunc(0)
	return err
	//return nil
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
