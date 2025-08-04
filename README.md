# chainmaker-vm-wasmer

针对长安链2.3.1、2.3.6、2.4.0的vm-wasmer虚拟机进行修改，使其可支持go标准编译的wasm程序

## 支持go-wasm：

**修改简介**：修改vm-wasmer模块，使其可支持标准go编译的wasm程序

**涉及模块**：vm-wasmer

**涉及文件**：vm_pool.go，SimContext.go、exports.go

**详细修改内容：**

1. 修改newInstanceFromModule函数：
   - 创建wasmer实例时判断是否wasm文件是否引入了wasi，如引入需要开启wasmer对wasi的支持。(标准go编译默认依赖wasi)
   - 将wasmer实例的内存指针存入env.memory对象（CMEnvironment结构体），便于后续函数下沉时调用
2. 修改CallMethod函数：
   - 修改运行时检查，长安链原本wasmer只用来跑rust的wasm程序存在运行时检查，我添加了判断条件int32(commonPb.RuntimeType_GASM) == runtimeSdkType，让wasmer也可以跑go的wasm程序
   - 长安链在合约部署阶段会把编译后wasm的字节码当作上下文参数传过来，由于标准go编译的wasm太长了，导致上下文创建不出来，为此手动把这个参数设置为空，该参数无实际作用。
3. 添加GetWasiStartRawFunction函数，用户获取wasm实例的wasi_get_start_function函数接口



## 添加函数下沉：

**修改简介**：使部分函数接口下沉，当合约执行到对应函数时会脱离wasmer环境跳转到对应go代码，用正常go执行对应函数

**涉及模块**：vm-wasmer

**涉及文件**：vm-bridge.go

**详细修改内容：**

1. 添加nativeSha256实现，sha256函数下沉
2. 添加nativeBigExp实现，大数BigExp运算函数下沉
3. 添加nativeBcx实现，crypto/bx509库函数下沉，bx509库部分使用了C代码，标准wasm不支持这样的混合编译。
4. 修改CMEnvironment结构体，引入memory  *wasmer.Memory 作为整个wasmer实例内存的指针，方便函数下沉时的调用
5. 修改GetImports函数，将上述添加的函数下沉接口暴露给wasmer实例，使其可以正常调用上述函数。



## 补充sdk相关接口：

**修改简介**：补充部分DockerGo实现了而wasmer没实现的接口，供合约sdk（contract-sdk-go-wasm）调用。

**涉及模块**：vm-wasmer,vm,protocol

**涉及文件**：

vm-wasmer：vm-bridge.go，vm-bridge-kv.go

vm：wasi,go

protocol：vm_interface.go

**详细修改内容：**

1. 添加GetSenderAddress接口，包含GetSenderAddressLen、GetSenderAddress、getSenderAddressCore三个函数接口。
2. 添加GetBatchState接口，包含GetBatchStateLen、GetBatchState、GetBatchState三个函数接口。
3. 添加history Kv相关接口，包含HistoryKvIterator、HistoryKvIterHasNext、HistoryKvIterNextLen、HistoryKvIterNext、HistoryKvIterClose等函数接口



## 虚拟机测试相关：

**修改简介**：修改vm-wasmer模块中的_text.go测试脚本，支持虚拟机侧更多合约的测试。

**涉及模块**：vm-wasmer

**详细修改内容：**

1. runtime_test.go、init_test.go和testdata文件夹：添加更多合约，支持虚拟机侧更多合约的测试
2. runtime.go 添加InvokeTime函数，较Invoke函数添加了时间打点，便于虚拟机侧测试。
