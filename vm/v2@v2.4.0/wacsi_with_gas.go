package vm

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"

	configPb "chainmaker.org/chainmaker/pb-go/v2/config"

	"chainmaker.org/chainmaker/common/v2/bytehelper"
	"chainmaker.org/chainmaker/common/v2/crypto/paillier"
	"chainmaker.org/chainmaker/common/v2/serialize"
	"chainmaker.org/chainmaker/pb-go/v2/common"
	"chainmaker.org/chainmaker/protocol/v2"
	"chainmaker.org/chainmaker/utils/v2"
	gaswasm "chainmaker.org/chainmaker/utils/v2/gas/wasm"
)

// WacsiWithGasImpl implements the Wacsi interface(WebAssembly chainmaker system interface)
type WacsiWithGasImpl struct {
	verifySql protocol.SqlVerifier
	rowIndex  int32
	enableSql map[string]bool // map[chainId]enableSql
	lock      *sync.Mutex
	logger    protocol.Logger
}

// NewWacsiWithGas get Wacsi instance with gas calculation
func NewWacsiWithGas(logger protocol.Logger, verifySql protocol.SqlVerifier) protocol.WacsiWithGas {
	return &WacsiWithGasImpl{
		verifySql: verifySql,
		rowIndex:  0,
		enableSql: make(map[string]bool),
		lock:      &sync.Mutex{},
		logger:    logger,
	}
}

// LogMessage is used to output debug info
func (w *WacsiWithGasImpl) LogMessage(message []byte, txSimContext protocol.TxSimContext) int32 {
	if err := gaswasm.SubtractGasForLogMessage(message, txSimContext); err != nil {
		w.logger.Debugf("wasmer log > > [%s] gas is not enough", txSimContext.GetTx().Payload.TxId)
		return protocol.ContractSdkSignalResultFail
	}
	w.logger.Debugf("wasmer log > > [%s] %s", txSimContext.GetTx().Payload.TxId, string(message))
	return protocol.ContractSdkSignalResultSuccess
}

// PutState is used to put state into simContext cache
func (w *WacsiWithGasImpl) PutState(
	requestBody []byte, contractName string, txSimContext protocol.TxSimContext) error {
	ec := serialize.NewEasyCodecWithBytes(requestBody)
	key, e1 := ec.GetString("key")
	if e1 != nil && txSimContext.GetBlockVersion() >= v235 {
		return e1
	}
	field, e2 := ec.GetString("field")
	if e2 != nil && txSimContext.GetBlockVersion() >= v235 {
		return e2
	}
	value, e3 := ec.GetBytes("value")
	if e3 != nil && txSimContext.GetBlockVersion() >= v235 {
		return e3
	}
	w.logger.Debugf("wacsiImpl::PutState() ==> key = %s, field = %s, value = %s \n", key, field, value)

	if err := protocol.CheckKeyFieldStr(key, field); err != nil {
		return err
	}

	stateKey := protocol.GetKeyStr(key, field)
	if err := gaswasm.SubtractGasForPutState(contractName, stateKey, value, txSimContext); err != nil {
		return err
	}

	err := txSimContext.Put(contractName, stateKey, value)
	return err
}

// GetState is used to get state from simContext cache
func (w *WacsiWithGasImpl) GetState(
	requestBody []byte, contractName string, txSimContext protocol.TxSimContext, memory []byte,
	data []byte, isLen bool) ([]byte, error) {

	ec := serialize.NewEasyCodecWithBytes(requestBody)
	key, e1 := ec.GetString("key")
	if e1 != nil && txSimContext.GetBlockVersion() >= v235 {
		return nil, e1
	}
	field, e2 := ec.GetString("field")
	if e2 != nil && txSimContext.GetBlockVersion() >= v235 {
		return nil, e2
	}
	valuePtr, e3 := ec.GetInt32("value_ptr")
	if e3 != nil && txSimContext.GetBlockVersion() >= v235 {
		return nil, e3
	}
	if err := protocol.CheckKeyFieldStr(key, field); err != nil {
		return nil, err
	}

	if !isLen {
		copy(memory[valuePtr:valuePtr+int32(len(data))], data)
		return nil, nil
	}

	stateKey := protocol.GetKeyStr(key, field)
	w.logger.Debugf("wacsiImpl::GetState() ==> key = %s, field = %s \n", key, field)
	value, err := txSimContext.Get(contractName, stateKey)
	if err != nil {
		msg := fmt.Errorf("[get state] fail. key=%s, field=%s, error:%s", key, field, err.Error())
		return nil, msg
	}
	if err1 := gaswasm.SubtractGasForGetState(contractName, stateKey, value, txSimContext); err != nil {
		return nil, err1
	}
	w.logger.Debugf("wacsiImpl::GetState() ==> value = %s \n", value)
	copy(memory[valuePtr:valuePtr+4], bytehelper.IntToBytes(int32(len(value))))
	if len(value) == 0 {
		return nil, nil
	}
	return value, nil
}

// DeleteState is used to delete state from simContext cache
func (w *WacsiWithGasImpl) DeleteState(
	requestBody []byte, contractName string, txSimContext protocol.TxSimContext) error {

	ec := serialize.NewEasyCodecWithBytes(requestBody)
	key, e1 := ec.GetString("key")
	if e1 != nil && txSimContext.GetBlockVersion() >= v235 {
		return e1
	}
	field, e2 := ec.GetString("field")
	if e2 != nil && txSimContext.GetBlockVersion() >= v235 {
		return e2
	}
	if err := protocol.CheckKeyFieldStr(key, field); err != nil {
		return err
	}

	stateKey := protocol.GetKeyStr(key, field)
	if err := gaswasm.SubtractGasForDeleteState(contractName, stateKey, txSimContext); err != nil {
		return err
	}

	err := txSimContext.Del(contractName, stateKey)
	if err != nil {
		return err
	}
	return nil
}

// ParseCallContractParams parse params for callContract func
func ParseCallContractParams(requestBody []byte) (int32, string, string,
	*serialize.EasyCodec, error) {

	ec := serialize.NewEasyCodecWithBytes(requestBody)
	valuePtr, e1 := ec.GetInt32("value_ptr")
	if e1 != nil {
		return 0, "", "", nil, fmt.Errorf("[call contract]value_ptr parsing failed")
	}
	contractName, e2 := ec.GetString("contract_name")
	if e2 != nil {
		return 0, "", "", nil, fmt.Errorf("[call contract]contract_name parsing failed")
	}
	method, e3 := ec.GetString("method")
	if e3 != nil {
		return 0, "", "", nil, fmt.Errorf("[call contract]method parsing failed")
	}
	param, e4 := ec.GetBytes("param")
	if e4 != nil {
		return 0, "", "", nil, fmt.Errorf("[call contract]param parsing failed")
	}

	ecData := serialize.NewEasyCodecWithBytes(param)
	return valuePtr, contractName, method, ecData, nil
}

// CallContract implement syscall for call contract, it is for gasm and wasmer
func (w *WacsiWithGasImpl) CallContract(
	caller *common.Contract,
	requestBody []byte,
	txSimContext protocol.TxSimContext,
	memory []byte,
	data []byte,
	gasUsed uint64,
	isLen bool,
) (*common.ContractResult, uint64, protocol.ExecOrderTxType, error) {

	valuePtr, contractName, method, ecData, err := ParseCallContractParams(requestBody)
	if err != nil && txSimContext.GetBlockVersion() >= v235 {
		return nil, gasUsed, protocol.ExecOrderTxTypeNormal, err
	}

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
	paramItem := ecData.GetItems()
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

		re, err1 := regexp.Compile(protocol.DefaultStateRegex)
		if err1 != nil {
			return nil, gasUsed, protocol.ExecOrderTxTypeNormal, err1
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
	if err1 := protocol.CheckKeyFieldStr(contractName, method); err1 != nil {
		return nil, gasUsed, protocol.ExecOrderTxTypeNormal, err1
	}

	// call contract
	gasUsed += protocol.CallContractGasOnce

	contract, err2 := txSimContext.GetContractByName(contractName)
	if err2 != nil {
		return nil, gasUsed, protocol.ExecOrderTxTypeNormal, fmt.Errorf(
			"[call contract] failed to get contract by [%s], err: %s",
			contractName,
			err2.Error(),
		)
	}

	result, specialTxType, code := txSimContext.CallContract(caller, contract, method,
		nil, paramMap, gasUsed, txSimContext.GetTx().Payload.TxType)
	if err3 := gaswasm.SubtractGasForCallContract(
		contractName, method, paramMap, result.Result, txSimContext); err3 != nil {
		return nil, gasUsed, protocol.ExecOrderTxTypeNormal, err3
	}
	gasUsed += result.GasUsed
	if code != common.TxStatusCode_SUCCESS {
		return nil, gasUsed, specialTxType,
			fmt.Errorf("[call contract] execute error code: %s, msg: %s",
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
func (w *WacsiWithGasImpl) SuccessResult(
	contractResult *common.ContractResult, txSimContext protocol.TxSimContext, data []byte) int32 {

	if err := gaswasm.SubtractGasForSuccessResult(data, txSimContext); err != nil {
		return protocol.ContractSdkSignalResultFail
	}
	if contractResult.Code == uint32(1) {
		return protocol.ContractSdkSignalResultFail
	}
	contractResult.Code = 0
	contractResult.Result = data
	return protocol.ContractSdkSignalResultSuccess
}

// ErrorResult is used to construct failed result
func (w *WacsiWithGasImpl) ErrorResult(
	contractResult *common.ContractResult, txSimContext protocol.TxSimContext, data []byte) int32 {

	if err := gaswasm.SubtractGasForErrorResult(data, txSimContext); err != nil {
		return protocol.ContractSdkSignalResultFail
	}
	contractResult.Code = uint32(1)
	if len(contractResult.Message) > 0 {
		contractResult.Message += ". contract message:" + string(data)
	} else {
		contractResult.Message = "contract message:" + string(data)
	}
	return protocol.ContractSdkSignalResultSuccess
}

// EmitEvent emit event to chain
func (w *WacsiWithGasImpl) EmitEvent(
	requestBody []byte, txSimContext protocol.TxSimContext, contractId *common.Contract,
	log protocol.Logger) (*common.ContractEvent, error) {

	ec := serialize.NewEasyCodecWithBytes(requestBody)
	topic, err := ec.GetString("topic")
	if err != nil {
		return nil, fmt.Errorf("[emit event] get topic err")
	}
	if err1 := protocol.CheckTopicStr(topic); err1 != nil {
		return nil, err1
	}

	req := ec.GetItems()
	var eventData []string
	for i := 1; i < len(req); i++ {
		data, ok := req[i].Value.(string)
		if !ok && txSimContext.GetBlockVersion() >= v235 {
			return nil, fmt.Errorf("[emit event] event data parsing failed")
		}
		eventData = append(eventData, data)
		log.Debugf("[emit event] event data :%v", data)
	}

	if err2 := protocol.CheckEventData(eventData); err2 != nil {
		return nil, err2
	}

	if err3 := gaswasm.SubtractGasForEmitEvent(topic, eventData, txSimContext); err3 != nil {
		return nil, err3
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
func (w *WacsiWithGasImpl) KvIterator(requestBody []byte, contractName string, txSimContext protocol.TxSimContext,
	memory []byte) error {

	ec := serialize.NewEasyCodecWithBytes(requestBody)
	startKey, e1 := ec.GetString("start_key")
	if e1 != nil && txSimContext.GetBlockVersion() >= v235 {
		return e1
	}
	startField, e2 := ec.GetString("start_field")
	if e2 != nil && txSimContext.GetBlockVersion() >= v235 {
		return e2
	}
	limitKey, e3 := ec.GetString("limit_key")
	if e3 != nil && txSimContext.GetBlockVersion() >= v235 {
		return e3
	}
	limitField, e4 := ec.GetString("limit_field")
	if e4 != nil && txSimContext.GetBlockVersion() >= v235 {
		return e4
	}
	valuePtr, e5 := ec.GetInt32("value_ptr")
	if e5 != nil && txSimContext.GetBlockVersion() >= v235 {
		return e5
	}

	if err := protocol.CheckKeyFieldStr(startKey, startField); err != nil {
		return err
	}
	if err := protocol.CheckKeyFieldStr(limitKey, limitField); err != nil {
		return err
	}

	key := protocol.GetKeyStr(startKey, startField)
	limit := protocol.GetKeyStr(limitKey, limitField)

	if err := gaswasm.SubtractGasForKvIterator(key, limit, txSimContext); err != nil {
		return err
	}

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
func (w *WacsiWithGasImpl) KvPreIterator(requestBody []byte, contractName string, txSimContext protocol.TxSimContext,
	memory []byte) error {

	ec := serialize.NewEasyCodecWithBytes(requestBody)
	startKey, e1 := ec.GetString("start_key")
	if e1 != nil && txSimContext.GetBlockVersion() >= v235 {
		return e1
	}
	startField, e2 := ec.GetString("start_field")
	if e2 != nil && txSimContext.GetBlockVersion() >= v235 {
		return e2
	}
	valuePtr, e3 := ec.GetInt32("value_ptr")
	if e3 != nil && txSimContext.GetBlockVersion() >= v235 {
		return e3
	}

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
	if err1 := gaswasm.SubtractGasForKvIterator([]byte(key), []byte(limit), txSimContext); err1 != nil {
		return err1
	}

	index := atomic.AddInt32(&w.rowIndex, 1)
	txSimContext.SetIterHandle(index, iter)
	copy(memory[valuePtr:valuePtr+4], bytehelper.IntToBytes(index))
	return nil
}

// KvIteratorHasNext is used to determine whether there is another element
func (w *WacsiWithGasImpl) KvIteratorHasNext(
	requestBody []byte, txSimContext protocol.TxSimContext, memory []byte) error {

	ec := serialize.NewEasyCodecWithBytes(requestBody)
	kvIndex, e1 := ec.GetInt32("rs_index")
	if e1 != nil && txSimContext.GetBlockVersion() >= v235 {
		return e1
	}
	valuePtr, e2 := ec.GetInt32("value_ptr")
	if e2 != nil && txSimContext.GetBlockVersion() >= v235 {
		return e2
	}

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
	if err := gaswasm.SubtractGasForKvIteratorHasNext(kvIndex, txSimContext); err != nil {
		return err
	}

	copy(memory[valuePtr:valuePtr+4], bytehelper.IntToBytes(int32(index)))
	return nil
}

// KvIteratorNext get next element
func (w *WacsiWithGasImpl) KvIteratorNext(
	requestBody []byte, txSimContext protocol.TxSimContext, memory []byte, data []byte,
	contractname string, isLen bool) ([]byte, error) {

	ec := serialize.NewEasyCodecWithBytes(requestBody)
	kvIndex, e1 := ec.GetInt32("rs_index")
	if e1 != nil && txSimContext.GetBlockVersion() >= v235 {
		return nil, e1
	}
	ptr, e2 := ec.GetInt32("value_ptr")
	if e2 != nil && txSimContext.GetBlockVersion() >= v235 {
		return nil, e2
	}

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
	var key, field string
	var value []byte
	ec = serialize.NewEasyCodec()
	if kvRows != nil {
		kvRow, err := kvRows.Value()
		if err != nil {
			return nil, fmt.Errorf("[kv iterator next] iterator next data error, %s", err.Error())
		}

		arrKey := strings.Split(string(kvRow.Key), "#")
		key = arrKey[0]
		field = ""
		if len(arrKey) > 1 {
			field = arrKey[1]
		}

		value = kvRow.Value
		ec.AddString("key", key)
		ec.AddString("field", field)
		ec.AddBytes("value", value)
	}
	kvBytes := ec.Marshal()
	if err := gaswasm.SubtractGasForKvIteratorNext(key, field, value, txSimContext); err != nil {
		return nil, err
	}

	copy(memory[ptr:ptr+4], bytehelper.IntToBytes(int32(len(kvBytes))))
	return kvBytes, nil
}

// KvIteratorClose close iteraotr
func (w *WacsiWithGasImpl) KvIteratorClose(requestBody []byte, contractName string, txSimContext protocol.TxSimContext,
	memory []byte) error {

	ec := serialize.NewEasyCodecWithBytes(requestBody)
	kvIndex, e1 := ec.GetInt32("rs_index")
	if e1 != nil && txSimContext.GetBlockVersion() >= v235 {
		return e1
	}
	valuePtr, e2 := ec.GetInt32("value_ptr")
	if e2 != nil && txSimContext.GetBlockVersion() >= v235 {
		return e2
	}

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
	if err := gaswasm.SubtractGasForKvIteratorClose(kvIndex, txSimContext); err != nil {
		return err
	}

	copy(memory[valuePtr:valuePtr+4], bytehelper.IntToBytes(1))
	return nil
}

// BulletProofsOperation is used to handle bulletproofs operations
func (w *WacsiWithGasImpl) BulletProofsOperation(requestBody []byte,
	txSimContext protocol.TxSimContext, memory []byte, data []byte, isLen bool) ([]byte, error) {

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
	opTypeStr, e1 := ec.GetString("bulletproofsFuncName")
	if e1 != nil && txSimContext.GetBlockVersion() >= v235 {
		return nil, e1
	}

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
	param1, e2 := ec.GetBytes("param1")
	if e2 != nil && txSimContext.GetBlockVersion() >= v235 {
		return nil, e2
	}
	param2, e3 := ec.GetBytes("param2")
	if e3 != nil && txSimContext.GetBlockVersion() >= v235 {
		return nil, e3
	}
	valuePtr, e4 := ec.GetInt32("value_ptr")
	if e4 != nil && txSimContext.GetBlockVersion() >= v235 {
		return nil, e4
	}

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

// PaillierOperation is used to handle paillier operations
func (w *WacsiWithGasImpl) PaillierOperation(requestBody []byte,
	txSimContext protocol.TxSimContext, memory []byte, data []byte, isLen bool) ([]byte, error) {

	ec := serialize.NewEasyCodecWithBytes(requestBody)
	opTypeStr, e1 := ec.GetString("opType")
	if e1 != nil && txSimContext.GetBlockVersion() >= v235 {
		return nil, e1
	}
	operandOne, e2 := ec.GetBytes("operandOne")
	if e2 != nil && txSimContext.GetBlockVersion() >= v235 {
		return nil, e2
	}
	operandTwo, e3 := ec.GetBytes("operandTwo")
	if e3 != nil && txSimContext.GetBlockVersion() >= v235 {
		return nil, e3
	}
	pubKeyBytes, e4 := ec.GetBytes("pubKey")
	if e4 != nil && txSimContext.GetBlockVersion() >= v235 {
		return nil, e4
	}
	valuePtr, e5 := ec.GetInt32("value_ptr")
	if e5 != nil && txSimContext.GetBlockVersion() >= v235 {
		return nil, e5
	}

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

// ExecuteQuery execute query operation
func (w *WacsiWithGasImpl) ExecuteQuery(requestBody []byte, contractName string, txSimContext protocol.TxSimContext,
	memory []byte, chainId string) error {

	if !w.isSupportSql(txSimContext) {
		return fmt.Errorf("not support sql, you must set chainConfig[contract.enable_sql_support=true]")
	}
	ec := serialize.NewEasyCodecWithBytes(requestBody)
	sql, e1 := ec.GetString("sql")
	if e1 != nil && txSimContext.GetBlockVersion() >= v235 {
		return e1
	}
	ptr, e2 := ec.GetInt32("value_ptr")
	if e2 != nil && txSimContext.GetBlockVersion() >= v235 {
		return e2
	}

	// verify
	var err error
	if err = w.verifySql.VerifyDQLSql(sql); err != nil {
		return fmt.Errorf("[execute query] verify sql error, %s", err.Error())
	}
	if err = gaswasm.SubtractGasForExecuteQuery(sql, txSimContext); err != nil {
		return err
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
func (w *WacsiWithGasImpl) ExecuteQueryOne(requestBody []byte, contractName string, txSimContext protocol.TxSimContext,
	memory []byte, data []byte, chainId string, isLen bool) ([]byte, error) {

	if !w.isSupportSql(txSimContext) {
		return nil, fmt.Errorf("not support sql, you must set chainConfig[contract.enable_sql_support=true]")
	}
	ec := serialize.NewEasyCodecWithBytes(requestBody)
	sql, e1 := ec.GetString("sql")
	if e1 != nil && txSimContext.GetBlockVersion() >= v235 {
		return nil, e1
	}
	ptr, e2 := ec.GetInt32("value_ptr")
	if e2 != nil && txSimContext.GetBlockVersion() >= v235 {
		return nil, e2
	}

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
		ecm := serialize.NewEasyCodecWithMap(dataRow)
		rsBytes := ecm.Marshal()
		if err1 := gaswasm.SubtractGasForExecuteQueryOne(sql, rsBytes, txSimContext); err1 != nil {
			return nil, err1
		}
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
func (w *WacsiWithGasImpl) RSHasNext(requestBody []byte, txSimContext protocol.TxSimContext, memory []byte) error {

	if !w.isSupportSql(txSimContext) {
		return fmt.Errorf("not support sql, you must set chainConfig[contract.enable_sql_support=true]")
	}
	ec := serialize.NewEasyCodecWithBytes(requestBody)
	rsIndex, e1 := ec.GetInt32("rs_index")
	if e1 != nil && txSimContext.GetBlockVersion() >= v235 {
		return e1
	}
	valuePtr, e2 := ec.GetInt32("value_ptr")
	if e2 != nil && txSimContext.GetBlockVersion() >= v235 {
		return e2
	}

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

	if err := gaswasm.SubtractGasForRSHasNext(rsIndex, txSimContext); err != nil {
		return err
	}
	copy(memory[valuePtr:valuePtr+4], bytehelper.IntToBytes(int32(index)))
	return nil
}

// RSNext get next record
func (w *WacsiWithGasImpl) RSNext(requestBody []byte, txSimContext protocol.TxSimContext, memory []byte, data []byte,
	isLen bool) ([]byte, error) {

	if !w.isSupportSql(txSimContext) {
		return nil, fmt.Errorf("not support sql, you must set chainConfig[contract.enable_sql_support=true]")
	}
	ec := serialize.NewEasyCodecWithBytes(requestBody)
	rsIndex, e1 := ec.GetInt32("rs_index")
	if e1 != nil && txSimContext.GetBlockVersion() >= v235 {
		return nil, e1
	}
	ptr, e2 := ec.GetInt32("value_ptr")
	if e2 != nil && txSimContext.GetBlockVersion() >= v235 {
		return nil, e2
	}

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
		ecm := serialize.NewEasyCodecWithMap(dataRow)
		rsBytes := ecm.Marshal()
		if err1 := gaswasm.SubtractGasForRSNext(rsIndex, rsBytes, txSimContext); err1 != nil {
			return nil, err1
		}

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
func (w *WacsiWithGasImpl) RSClose(requestBody []byte, txSimContext protocol.TxSimContext, memory []byte) error {

	if !w.isSupportSql(txSimContext) {
		return fmt.Errorf("not support sql, you must set chainConfig[contract.enable_sql_support=true]")
	}
	ec := serialize.NewEasyCodecWithBytes(requestBody)
	rsIndex, e1 := ec.GetInt32("rs_index")
	if e1 != nil && txSimContext.GetBlockVersion() >= v235 {
		return e1
	}
	valuePtr, e2 := ec.GetInt32("value_ptr")
	if e2 != nil && txSimContext.GetBlockVersion() >= v235 {
		return e2
	}

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

	if err := gaswasm.SubtractGasForRSClose(rsIndex, txSimContext); err != nil {
		return err
	}
	copy(memory[valuePtr:valuePtr+4], bytehelper.IntToBytes(index))
	return nil
}

// ExecuteUpdate execute udpate
func (w *WacsiWithGasImpl) ExecuteUpdate(requestBody []byte, contractName string, method string,
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
	sql, e1 := ec.GetString("sql")
	if e1 != nil && txSimContext.GetBlockVersion() >= v235 {
		return e1
	}
	ptr, e2 := ec.GetInt32("value_ptr")
	if e2 != nil && txSimContext.GetBlockVersion() >= v235 {
		return e2
	}

	// verify
	if err := w.verifySql.VerifyDMLSql(sql); err != nil {
		return fmt.Errorf("[execute update] verify update sql error, [%s], sql [%s]", err.Error(), sql)
	}
	if err := gaswasm.SubtractGasForExecuteUpdate(sql, txSimContext); err != nil {
		return err
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
func (w *WacsiWithGasImpl) ExecuteDDL(requestBody []byte, contractName string, txSimContext protocol.TxSimContext,
	memory []byte, method string) error {

	if !w.isSupportSql(txSimContext) {
		return fmt.Errorf("not support sql, you must set chainConfig[contract.enable_sql_support=true]")
	}
	if !w.isManageContract(method) {
		return ErrorNotManageContract
	}
	ec := serialize.NewEasyCodecWithBytes(requestBody)
	sql, e1 := ec.GetString("sql")
	if e1 != nil && txSimContext.GetBlockVersion() >= v235 {
		return e1
	}
	ptr, e2 := ec.GetInt32("value_ptr")
	if e2 != nil && txSimContext.GetBlockVersion() >= v235 {
		return e2
	}

	// verify
	if err := w.verifySql.VerifyDDLSql(sql); err != nil {
		return fmt.Errorf("[execute ddl] verify ddl sql error,  [%s], sql[%s]", err.Error(), sql)
	}
	if err := gaswasm.SubtractGasForExecuteDDL(sql, txSimContext); err != nil {
		return err
	}

	if err := txSimContext.GetBlockchainStore().ExecDdlSql(contractName, sql, "1"); err != nil {
		return fmt.Errorf("[execute ddl] execute error, %s, sql[%s]", err.Error(), sql)
	}
	txSimContext.PutRecord(contractName, []byte(sql), protocol.SqlTypeDdl)
	copy(memory[ptr:ptr+4], bytehelper.IntToBytes(0))
	return nil
}

func (w *WacsiWithGasImpl) isManageContract(method string) bool {
	return method == protocol.ContractInitMethod || method == protocol.ContractUpgradeMethod
}

func (w *WacsiWithGasImpl) isSupportSql(txSimContext protocol.TxSimContext) bool {
	w.lock.Lock()
	defer w.lock.Unlock()
	chainId := txSimContext.GetTx().Payload.ChainId
	if b, ok := w.enableSql[chainId]; ok {
		return b
	}
	var cc *configPb.ChainConfig
	if txSimContext.GetBlockVersion() < v235 {
		cc, _ = txSimContext.GetBlockchainStore().GetLastChainConfig()
	} else {
		cc = txSimContext.GetLastChainConfig()
	}

	w.enableSql[chainId] = cc.Contract.EnableSqlSupport
	return cc.Contract.EnableSqlSupport
}
