/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package vm

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
	"time"

	"chainmaker.org/chainmaker/common/v2/msgbus"
	acPb "chainmaker.org/chainmaker/pb-go/v2/accesscontrol"
	commonPb "chainmaker.org/chainmaker/pb-go/v2/common"
	"chainmaker.org/chainmaker/pb-go/v2/config"
	"chainmaker.org/chainmaker/pb-go/v2/syscontract"
	"chainmaker.org/chainmaker/protocol/v2"
	native "chainmaker.org/chainmaker/vm-native/v2"
	"github.com/gogo/protobuf/proto"
)

const (
	blockVersion220  = 220
	blockVersion2300 = 2300
	blockVersion2310 = 2030100
	blockVersion2312 = 2030102
	blockVersion233  = 2030300
)

// NewVmManager get vm runtime manager
func NewVmManager(instanceManagers map[commonPb.RuntimeType]protocol.VmInstancesManager, wxvmCodePathPrefix string,
	accessControl protocol.AccessControlProvider, chainNodesInfoProvider protocol.ChainNodesInfoProvider,
	chainConf protocol.ChainConf, log protocol.Logger) protocol.VmManager {

	chainId := chainConf.ChainConfig().ChainId
	//log := logger.GetLoggerByChain(logger.MODULE_VM, chainId)

	//instance VmManagerImpl
	return &VmManagerImpl{
		ChainId:                chainId,
		InstanceManagers:       instanceManagers,
		aliveManagers:          make(map[commonPb.RuntimeType]struct{}),
		WxvmCodePath:           wxvmCodePathPrefix,
		AccessControl:          accessControl,
		ChainNodesInfoProvider: chainNodesInfoProvider,
		Log:                    log,
		ChainConf:              chainConf,
	}
}

// VmManagerImpl implements the VmManager interface, manage vm runtime
type VmManagerImpl struct {
	InstanceManagers       map[commonPb.RuntimeType]protocol.VmInstancesManager
	aliveManagers          map[commonPb.RuntimeType]struct{}
	WxvmCodePath           string
	SnapshotManager        protocol.SnapshotManager
	AccessControl          protocol.AccessControlProvider
	ChainNodesInfoProvider protocol.ChainNodesInfoProvider
	ChainId                string
	Log                    protocol.Logger
	ChainConf              protocol.ChainConf // chain config
}

// GetAccessControl get accessControl manages policies and principles
func (m *VmManagerImpl) GetAccessControl() protocol.AccessControlProvider {
	return m.AccessControl
}

// GetChainNodesInfoProvider get ChainNodesInfoProvider provide base node info list of chain.
func (m *VmManagerImpl) GetChainNodesInfoProvider() protocol.ChainNodesInfoProvider {
	return m.ChainNodesInfoProvider
}

// Start all vm instance
func (m *VmManagerImpl) Start() error {
	var err error
	// check every instance manager
	for runtimeType, instanceManager := range m.InstanceManagers {
		// instance manager not exists, or instance manager is not alive, start it
		if _, ok := m.aliveManagers[runtimeType]; !ok {
			m.Log.Infof("starting vm instance manager: %v", runtimeType)
			if err = instanceManager.StartVM(); err != nil {
				return fmt.Errorf("failed to start %v vm manager", runtimeType)
			}
			m.aliveManagers[runtimeType] = struct{}{}
		}
	}
	return nil
}

// Stop all vm instance
func (m *VmManagerImpl) Stop() error {
	var err error
	// check every instance manager
	for runtimeType, instanceManager := range m.InstanceManagers {
		//if the manager is alive, stop it
		if _, ok := m.aliveManagers[runtimeType]; ok {
			m.Log.Infof("stopping vm instance manager: %v", runtimeType)
			err = instanceManager.StopVM()
			if err != nil {
				m.Log.Errorf("failed to stop vm instance manager: %v", runtimeType)
				continue
			}
			delete(m.aliveManagers, runtimeType)
		}
	}
	return err
}

// BeforeSchedule do sth. before schedule a block
func (m *VmManagerImpl) BeforeSchedule(blockFingerprint string, blockHeight uint64) {
	//prepare docker go if its instance manager not null
	if m.InstanceManagers[commonPb.RuntimeType_DOCKER_GO] != nil {
		m.InstanceManagers[commonPb.RuntimeType_DOCKER_GO].BeforeSchedule(blockFingerprint, blockHeight)
	}

	//prepare go vm if its instance manager not null
	if m.InstanceManagers[commonPb.RuntimeType_GO] != nil {
		m.InstanceManagers[commonPb.RuntimeType_GO].BeforeSchedule(blockFingerprint, blockHeight)
	}
}

// AfterSchedule do sth. after schedule a block
func (m *VmManagerImpl) AfterSchedule(blockFingerprint string, blockHeight uint64) {
	//prepare docker go if its instance manager not null
	if m.InstanceManagers[commonPb.RuntimeType_DOCKER_GO] != nil {
		m.InstanceManagers[commonPb.RuntimeType_DOCKER_GO].AfterSchedule(blockFingerprint, blockHeight)
	}

	//prepare go vm if its instance manager not null
	if m.InstanceManagers[commonPb.RuntimeType_GO] != nil {
		m.InstanceManagers[commonPb.RuntimeType_GO].AfterSchedule(blockFingerprint, blockHeight)
	}
}

// OnMessage obtain chain configuration check all vm runtime readiness
// if not ready, init and start the instance manager, then add to vm manager InstanceManagers
func (m *VmManagerImpl) OnMessage(msg *msgbus.Message) {
	m.Log.Infof("vm manager get message,msg.topic:%d", msg.Topic)
	switch msg.Topic {
	case msgbus.ChainConfig:
		m.Log.Info(" get the message, msg topic:msgbus.ChainConfig")
		dataStr, ok := msg.Payload.([]string)
		if !ok {
			m.Log.Error("the type of msg.Payload is not []string")
			return
		}
		if len(dataStr) == 0 {
			m.Log.Error("the msg.Payload is empty")
			return
		}
		dataBytes, err := hex.DecodeString(dataStr[0])
		if err != nil {
			m.Log.Error(err)
			return
		}
		chainConfig := &config.ChainConfig{}
		err = proto.Unmarshal(dataBytes, chainConfig)

		if err != nil {
			m.Log.Error(err)
			return
		}

		for _, vmConfig := range chainConfig.GetVm().GetSupportList() {
			vmConfig = strings.ToUpper(vmConfig)
			runtimeType := protocol.VmTypeToRunTimeType[vmConfig]

			// if the vm instance manager is alive, continue
			if _, ok := m.aliveManagers[runtimeType]; ok {
				continue
			}

			m.addNewRuntimeType(chainConfig.GetChainId(), runtimeType)

			if runtimeType == commonPb.RuntimeType_DOCKER_GO {
				m.addNewRuntimeType(chainConfig.GetChainId(), commonPb.RuntimeType_GO)
			}
		}
	}
}

// OnQuit nothing to do
func (m *VmManagerImpl) OnQuit() {
	// nothing for implement interface msgbus.Subscriber
}

// RunContract run native or user contract according ContractName in contractId, and call the specified function
func (m *VmManagerImpl) RunContract(contract *commonPb.Contract, method string, byteCode []byte,
	parameters map[string][]byte, txContext protocol.TxSimContext, gasUsed uint64, refTxType commonPb.TxType) (
	*commonPb.ContractResult, protocol.ExecOrderTxType, commonPb.TxStatusCode) {

	//prepare contract result
	contractResult := &commonPb.ContractResult{
		Code:    uint32(1),
		Result:  nil,
		Message: "",
	}

	if txContext.GetBlockVersion() >= v235 {
		contractResult.GasUsed = gasUsed
	}

	contractName := contract.Name
	//return err if name is empty
	if contractName == "" {
		contractResult.Message = "contractName not found"
		m.Log.Error(contractResult.Message)
		return contractResult, protocol.ExecOrderTxTypeNormal,
			commonPb.TxStatusCode_INVALID_CONTRACT_PARAMETER_CONTRACT_NAME
	}

	if parameters == nil {
		parameters = make(map[string][]byte)
	}

	//if contract version is 0, check contract
	if len(contract.Version) == 0 {
		var err error
		contract, err = txContext.GetContractByName(contractName)
		if err != nil {
			m.Log.Warn(err)
			contractResult.Message = fmt.Sprintf("query contract[%s] error", contractName)
			return contractResult, protocol.ExecOrderTxTypeNormal,
				commonPb.TxStatusCode_INVALID_CONTRACT_PARAMETER_CONTRACT_NAME
		}
	}

	// starting from version 2030100(v2.3.1), frozen contracts can be upgraded
	if txContext.GetBlockVersion() < blockVersion2310 {
		if contract.Status == commonPb.ContractStatus_FROZEN {
			contractResult.Message = fmt.Sprintf("failed to run user contract, %s has been frozen.", contractName)
			return contractResult, protocol.ExecOrderTxTypeNormal, commonPb.TxStatusCode_CONTRACT_FREEZE_FAILED
		}
	} else {
		if contract.Status == commonPb.ContractStatus_FROZEN && method != protocol.ContractUpgradeMethod {
			contractResult.Message = fmt.Sprintf("failed to run user contract, %s has been frozen.", contractName)
			return contractResult, protocol.ExecOrderTxTypeNormal, commonPb.TxStatusCode_CONTRACT_FREEZE_FAILED
		}
	}

	//check if it revoked
	if contract.Status == commonPb.ContractStatus_REVOKED {
		contractResult.Message = fmt.Sprintf("failed to run user contract, %s has been revoked.", contractName)
		return contractResult, protocol.ExecOrderTxTypeNormal, commonPb.TxStatusCode_CONTRACT_REVOKE_FAILED
	}

	// record and remove crossInfo
	txContext.RecordRuntimeTypeIntoCrossInfo(contract.RuntimeType)
	defer txContext.RemoveRuntimeTypeFromCrossInfo()

	//check if it is native contract
	if native.IsNative(contractName, refTxType) {
		if method == "" {
			contractResult.Message = "require param method not found."
			return contractResult, protocol.ExecOrderTxTypeNormal,
				commonPb.TxStatusCode_INVALID_CONTRACT_PARAMETER_METHOD
		}

		return m.runNativeContract(contract, method, parameters, txContext, gasUsed)
	}

	//check if its is user contract
	if !m.isUserContract(refTxType) {
		contractResult.Message = fmt.Sprintf("bad contract call %s, transaction type %s", contractName, refTxType)
		return contractResult, protocol.ExecOrderTxTypeNormal, commonPb.TxStatusCode_INVALID_CONTRACT_TRANSACTION_TYPE
	}

	// byteCode should have value
	if contract.RuntimeType != commonPb.RuntimeType_DOCKER_GO &&
		contract.RuntimeType != commonPb.RuntimeType_GO &&
		contract.RuntimeType != commonPb.RuntimeType_DOCKER_JAVA &&
		len(byteCode) == 0 {
		contractResult.Message = fmt.Sprintf("contract %s has no byte code, transaction type %s",
			contractName, refTxType)
		m.Log.Error(contractResult.Message)
		return contractResult, protocol.ExecOrderTxTypeNormal, commonPb.TxStatusCode_CONTRACT_BYTE_CODE_NOT_EXIST_FAILED
	}

	//run user contract
	return m.runUserContract(contract, method, byteCode, parameters, txContext, gasUsed, refTxType)
}

// runNativeContract invoke native contract
func (m *VmManagerImpl) runNativeContract(contract *commonPb.Contract, method string, parameters map[string][]byte,
	txContext protocol.TxSimContext, gasUsed uint64) (*commonPb.ContractResult,
	protocol.ExecOrderTxType,
	commonPb.TxStatusCode) {
	txId := txContext.GetTx().Payload.TxId
	runtimeType := contract.RuntimeType
	m.Log.Infof("invoke native contract[%s], tx id:%s, runtime:%+v, method:%s",
		contract.Name, txId, contract.RuntimeType.String(), method)

	//defaultGas := uint64(0)
	//check if enable gas
	//if m.ChainConf.ChainConfig().AccountConfig != nil && m.ChainConf.ChainConfig().AccountConfig.EnableGas {
	//	defaultGas = m.ChainConf.ChainConfig().AccountConfig.DefaultGas
	//}
	//runtimeInstance := native.GetRuntimeInstance(m.ChainId, defaultGas, m.Log)

	runtimeInstance := native.GetRuntimeInstance(m.ChainId)
	//记录开始时间
	startTime := time.Now().UnixNano()
	runtimeContractResult := runtimeInstance.Invoke(contract, method, nil, parameters, txContext)
	//记录结束时间。并写入日志
	endTime := time.Now().UnixNano()
	executionTime := float64(endTime-startTime) / 1e9

	m.Log.Infof("invoke vm, tx id:%s, runtime type:%+v, runtimeContractResult:%d ,startTime:%d, endTime:%d, executionTime:%.6f s",
		txId, runtimeType, runtimeContractResult.Code, startTime, endTime, executionTime)
	runtimeContractResult.GasUsed += gasUsed
	//check if result code succeed
	if runtimeContractResult.Code == uint32(0) {
		if contract.Name == syscontract.SystemContract_ACCOUNT_MANAGER.String() &&
			method == syscontract.GasAccountFunction_CHARGE_GAS_FOR_MULTI_ACCOUNT.String() {
			return runtimeContractResult, protocol.ExecOrderTxTypeChargeGas, commonPb.TxStatusCode_SUCCESS
		}
		return runtimeContractResult, protocol.ExecOrderTxTypeNormal, commonPb.TxStatusCode_SUCCESS
	}
	if contract.Name == syscontract.SystemContract_ACCOUNT_MANAGER.String() &&
		method == syscontract.GasAccountFunction_CHARGE_GAS_FOR_MULTI_ACCOUNT.String() {
		return runtimeContractResult, protocol.ExecOrderTxTypeChargeGas, commonPb.TxStatusCode_CONTRACT_FAIL
	}
	return runtimeContractResult, protocol.ExecOrderTxTypeNormal, commonPb.TxStatusCode_CONTRACT_FAIL
}

// runUserContract invoke user contract
func (m *VmManagerImpl) runUserContract(contract *commonPb.Contract, method string, byteCode []byte,
	parameters map[string][]byte, txContext protocol.TxSimContext, gasUsed uint64, refTxType commonPb.TxType) (
	contractResult *commonPb.ContractResult, specialTxType protocol.ExecOrderTxType, code commonPb.TxStatusCode) {

	var (
		myContract = contract
		//contractName = contract.Name
		//status       = contract.Status
	)
	//nolint:ineffassign
	contractResult = &commonPb.ContractResult{Code: uint32(1)}
	//if status == commonPb.ContractStatus_ALL {
	//	dbContract, err := utils.GetContractByName(txContext.Get, contractName)
	//	if err != nil {
	//		return nil, commonPb.TxStatusCode_CONTRACT_FAIL
	//	}
	//	myContract = dbContract
	//}

	return m.invokeUserContractByRuntime(myContract, method, parameters, txContext, byteCode, gasUsed)
}

// invokeUserContractByRuntime invoke user contract by runtime
func (m *VmManagerImpl) invokeUserContractByRuntime(contract *commonPb.Contract, method string,
	parameters map[string][]byte, txContext protocol.TxSimContext, byteCode []byte, gasUsed uint64) (
	*commonPb.ContractResult, protocol.ExecOrderTxType, commonPb.TxStatusCode) {
	contractResult := &commonPb.ContractResult{Code: uint32(1)}
	txId := txContext.GetTx().Payload.TxId
	txType := txContext.GetTx().Payload.TxType
	runtimeType := contract.RuntimeType
	contractName := contract.Name
	m.Log.Infof("invoke user contract[%s], tx id:%s, runtime:%s, method:%s",
		contract.Name, txId, contract.RuntimeType.String(), method)

	//get vm instance manager byt runtime type
	vmInstancesManager := m.InstanceManagers[runtimeType]
	if vmInstancesManager == nil {

		// enable go runtime type
		goExists := m.InstanceManagers[commonPb.RuntimeType_GO]
		// enable docker_go runtime type
		dockerGoExists := m.InstanceManagers[commonPb.RuntimeType_DOCKER_GO]

		// enable docker_go runtime type && disable go runtime type && use go runtime type
		if runtimeType == commonPb.RuntimeType_GO && dockerGoExists != nil {
			contractResult.Message = "incorrect vm go runtime version, you only have docker_go part configured in " +
				"chainmaker.yml's vm module, but your contract sdk version >= v2.3.0, it needs go part"
		} else if runtimeType == commonPb.RuntimeType_DOCKER_GO && goExists != nil {
			// enable go runtime type && disable docker_go runtime type && use docker_go runtime type
			contractResult.Message = "incorrect vm go runtime version, you only have go part configured in " +
				"chainmaker.yml's vm module, but your contract sdk version < v2.3.0, it needs docker_go part"
		} else {
			// runtime type not exist
			contractResult.Message = fmt.Sprintf("no such vm runtime %q", runtimeType)
		}
		return contractResult, protocol.ExecOrderTxTypeNormal, commonPb.TxStatusCode_INVALID_CONTRACT_PARAMETER_RUNTIME_TYPE
	}

	//new instance
	runtimeInstance, err := vmInstancesManager.NewRuntimeInstance(txContext, m.ChainId, method, m.WxvmCodePath,
		contract, byteCode, m.Log)
	if err != nil {
		contractResult.Message = fmt.Sprintf("failed to create vm runtime, contract: %s, %s",
			contract.Name, err.Error())
		return contractResult, protocol.ExecOrderTxTypeNormal, commonPb.TxStatusCode_CREATE_RUNTIME_INSTANCE_FAILED
	}

	sender, creator, origin, orderTxType, statusCode, err := m.getMember(contract, txContext, runtimeType)
	if err != nil {
		contractResult.Message = err.Error()
		return contractResult, orderTxType, statusCode
	}

	m.setParams(parameters, txContext, sender, creator, origin, txId)

	// calc the gas used by byte code
	// gasUsed := uint64(GasPerByte * len(byteCode))

	m.Log.Debugf("invoke vm, tx id:%s, tx type:%+v, contractId:%+v, method:%+v,"+
		" runtime type:%+v, byte code len:%+v, params:%+v",
		txId, txType, contract, method, runtimeType, len(byteCode), len(parameters))

	// begin save point for sql
	var dbTransaction protocol.SqlDBTransaction
	if m.ChainConf.ChainConfig().Contract.EnableSqlSupport && txType != commonPb.TxType_QUERY_CONTRACT {
		txKey := commonPb.GetTxKeyWith(txContext.GetBlockProposer().MemberInfo, txContext.GetBlockHeight())
		dbTransaction, err = txContext.GetBlockchainStore().GetDbTransaction(txKey)
		if err != nil {
			contractResult.Message = fmt.Sprintf("get db transaction from [%s] error %+v", txKey, err)
			return contractResult, protocol.ExecOrderTxTypeNormal, commonPb.TxStatusCode_INTERNAL_ERROR
		}
		err := dbTransaction.BeginDbSavePoint(txId)
		if err != nil {
			m.Log.Warn("[%s] begin db save point error, %s", txId, err.Error())
		}
		//txContext.Put(contractId.Name, []byte("target"), []byte("mysql")) // for dag
	}
	//记录开始时间
	startTime := time.Now().UnixNano()
	runtimeContractResult, specialTxType := runtimeInstance.Invoke(contract, method, byteCode, parameters, txContext,
		gasUsed)
	//记录结束时间。并写入日志
	endTime := time.Now().UnixNano()
	executionTime := float64(endTime-startTime) / 1e9
	m.Log.Infof("invoke vm, tx id:%s, contractName:%+v, contractMethod:%+v, runtime type:%+v, runtimeContractResult:%d ,startTime:%d, endTime:%d, executionTime:%.6f s",
		txId, contractName, method, runtimeType, runtimeContractResult.Code, startTime, endTime, executionTime)
	if runtimeContractResult.Code == 0 {
		return runtimeContractResult, specialTxType, commonPb.TxStatusCode_SUCCESS
	}
	if m.ChainConf.ChainConfig().Contract.EnableSqlSupport && txType != commonPb.TxType_QUERY_CONTRACT {
		err := dbTransaction.RollbackDbSavePoint(txId)
		if err != nil {
			m.Log.Warn("[%s] rollback db save point error, %s", txId, err.Error())
		}
	}
	return runtimeContractResult, specialTxType, commonPb.TxStatusCode_CONTRACT_FAIL
}

// setParams sets parameters
func (m *VmManagerImpl) setParams(parameters map[string][]byte, txContext protocol.TxSimContext,
	sender *acPb.Member, creator *acPb.MemberFull, origin protocol.Member, txId string) {
	//check if current call is cross contract call
	if string(parameters[syscontract.CrossParams_CALL_TYPE.String()]) == syscontract.CallType_CROSS.String() {
		//parameters[protocol.ContractSenderTypeParam] = []byte(strconv.Itoa(int(acPb.MemberType_ADDR)))
		parameters[protocol.ContractSenderRoleParam] = []byte(protocol.RoleContract)
	} else {
		parameters[protocol.ContractSenderRoleParam] = []byte(origin.GetRole()) //non-cross-call sender==origin
	}

	parameters[protocol.ContractSenderOrgIdParam] = []byte(sender.OrgId)
	parameters[protocol.ContractCreatorOrgIdParam] = []byte(creator.OrgId)
	parameters[protocol.ContractCreatorRoleParam] = []byte(creator.Role)

	typeParam := make([]byte, 4)
	addrType := m.ChainConf.ChainConfig().GetVm().GetAddrType()
	if txContext.GetBlockVersion() < blockVersion2300 && addrType == config.AddrType_ZXL {
		binary.BigEndian.PutUint32(typeParam, uint32(config.AddrType_ZXL))
		parameters[protocol.ContractCreatorPkParam] = creator.MemberInfo
		parameters[protocol.ContractSenderPkParam] = sender.MemberInfo
		parameters[protocol.ContractCreatorTypeParam] = make([]byte, 4)
		parameters[protocol.ContractSenderTypeParam] = make([]byte, 4)

		memberType := make([]byte, 4)
		binary.BigEndian.PutUint32(memberType, uint32(acPb.MemberType_PUBLIC_KEY))
		if creator.MemberType == acPb.MemberType_PUBLIC_KEY {
			parameters[protocol.ContractCreatorTypeParam] = memberType
		}
		if sender.MemberType == acPb.MemberType_PUBLIC_KEY {
			parameters[protocol.ContractSenderTypeParam] = memberType
		}
	} else {
		//ATTENTION: uid is SKI(SubjectKeyId) in any mode(include cert/pk/pwk etc.)
		//SKI generation -- encode public key by DER, then calculate it's by chainmaker default hash
		parameters[protocol.ContractCreatorPkParam] = []byte(creator.GetUid())
		parameters[protocol.ContractSenderPkParam] = []byte(origin.GetUid())
	}

	parameters[protocol.ContractTxIdParam] = []byte(txId)
	parameters[protocol.ContractBlockHeightParam] = []byte(strconv.FormatUint(txContext.GetBlockHeight(), 10))
	parameters[protocol.ContractTxTimeStamp] = []byte(strconv.FormatInt(txContext.GetTx().GetPayload().Timestamp, 10))
}

// getMember get member by contract
func (m *VmManagerImpl) getMember(contract *commonPb.Contract,
	txContext protocol.TxSimContext,
	runtimeType commonPb.RuntimeType) (
	sender *acPb.Member, creator *acPb.MemberFull, senderMember protocol.Member,
	tp protocol.ExecOrderTxType, code commonPb.TxStatusCode, err error) {
	//先计算Sender
	sender = txContext.GetSender()
	sender, code = m.getFullCertMember(sender, txContext)
	if code != commonPb.TxStatusCode_SUCCESS {
		return nil, nil, nil, protocol.ExecOrderTxTypeNormal, code, err
	}
	// Get three items in the certificate: orgid PK role
	senderMember, err = m.AccessControl.NewMember(sender)
	if err != nil {
		err = fmt.Errorf("failed to unmarshal sender %q", runtimeType)
		return nil, nil, nil, protocol.ExecOrderTxTypeNormal, commonPb.TxStatusCode_UNMARSHAL_SENDER_FAILED, err
	}
	//再计算creator
	creator = contract.Creator
	if creator == nil {
		err = fmt.Errorf("creatorTmp is empty for contract:%s", contract.Name)
		return nil, nil, nil, protocol.ExecOrderTxTypeNormal, commonPb.TxStatusCode_GET_CREATOR_FAILED, err
	}
	//creator.MemberId == "" 这个判断是因为有可能合约是在v2.2.0_alpha或者之前创建的，然后是在v2.2.0正式版运行的
	//这个时候需要重新通过ac模块获取Role和PK，但是不记录读集
	if txContext.GetBlockVersion() < blockVersion220 || contract.Creator.MemberId == "" {
		//在v2.2.0_alpha版中，creator并没有Role，PubKey等信息，所以需要再次查询
		//在>220的版本也就是v2.2.0正式版及之后的版本中，creator有了完整的Role，PubKey信息，所以不需要再次查询了

		creatorPbMember := &acPb.Member{
			OrgId:      creator.OrgId,
			MemberType: creator.MemberType,
			MemberInfo: creator.MemberInfo,
		}
		creatorPbMember, code = m.getFullCertMember(creatorPbMember, txContext)
		if code != commonPb.TxStatusCode_SUCCESS {
			return nil, nil, nil, protocol.ExecOrderTxTypeNormal, code, err
		}

		// Get three items in the certificate: orgid PK role
		creatorMember, err2 := m.AccessControl.NewMember(creatorPbMember)
		if err2 != nil {
			m.Log.Error("failed to unmarshal creator " + err2.Error())
			err = fmt.Errorf("failed to unmarshal creator %q", creatorPbMember)
			return nil, nil, nil, protocol.ExecOrderTxTypeNormal, commonPb.TxStatusCode_UNMARSHAL_CREATOR_FAILED, err
		}

		//construct creator member instance
		creator = &acPb.MemberFull{
			OrgId:      creatorPbMember.OrgId,
			MemberType: creatorPbMember.MemberType,
			MemberInfo: creatorPbMember.MemberInfo,
			MemberId:   creatorMember.GetMemberId(),
			Role:       string(creatorMember.GetRole()),
			Uid:        creatorMember.GetUid(),
		}
	}
	return sender, creator, senderMember, 0, 0, err
}

// getFullCertMember 如果是证书Hash，重新获取完整的证书Member
func (m *VmManagerImpl) getFullCertMember(sender *acPb.Member, txContext protocol.TxSimContext) (
	*acPb.Member, commonPb.TxStatusCode) {
	//在v2.2.0版之前，如果是CertHash，在取完整的Cert的时候，会记录到读集中
	//在v2.2.0正式版及之后，取完整Cert不会再放入读集
	recordIntoReadSet := txContext.GetBlockVersion() <= blockVersion220
	//特殊处理，因为v2.2.0_alpha版的时候对CertAlias没有GetCert，所以没有读集，这是一个Bug
	//现在只能将错就错，不然就会不兼容。所以在BlockVersion==220而且是CertAlias的时候，不产生读集
	if txContext.GetBlockVersion() == blockVersion220 && sender.MemberType == acPb.MemberType_ALIAS {
		return sender, commonPb.TxStatusCode_SUCCESS
	}

	// If the certificate in the transaction is hash, the original certificate is retrieved
	if sender.MemberType == acPb.MemberType_CERT_HASH || sender.MemberType == acPb.MemberType_ALIAS {
		var fullCertMemberInfo []byte
		var err error
		if txContext.GetBlockVersion() >= blockVersion233 {
			fullCertMemberInfo, err = m.AccessControl.GetCertFromCache(sender.MemberInfo)

		} else {
			memberInfoHex := hex.EncodeToString(sender.MemberInfo)
			if recordIntoReadSet {
				fullCertMemberInfo, err = txContext.Get(syscontract.SystemContract_CERT_MANAGE.String(),
					[]byte(memberInfoHex))
			} else {
				fullCertMemberInfo, err = txContext.GetNoRecord(syscontract.SystemContract_CERT_MANAGE.String(),
					[]byte(memberInfoHex))
			}

		}

		if err != nil {
			fmt.Println(err)
			return nil, commonPb.TxStatusCode_GET_SENDER_CERT_FAILED
		}

		sender = &acPb.Member{
			OrgId:      sender.OrgId,
			MemberInfo: fullCertMemberInfo,
			MemberType: acPb.MemberType_CERT,
		}
	}
	return sender, commonPb.TxStatusCode_SUCCESS
}

// isUserContract check if it is user contract
func (m *VmManagerImpl) isUserContract(refTxType commonPb.TxType) bool {
	switch refTxType {
	case
		commonPb.TxType_INVOKE_CONTRACT,
		commonPb.TxType_QUERY_CONTRACT:
		return true
	default:
		return false
	}
}

// addNewRuntimeType add a  new runtime type
func (m *VmManagerImpl) addNewRuntimeType(chainId string, runtimeType commonPb.RuntimeType) {
	m.Log.Infof("add new runtime type:%d,chainId:%s", runtimeType, chainId)
	// init vm engine
	vmInstancesManagerProvider := GetVmProvider(protocol.RunTimeTypeToVmType[runtimeType])
	if vmInstancesManagerProvider == nil {
		m.Log.Warnf("get vm provider nil,runtimeType:%d, runtime:%s", runtimeType,
			protocol.RunTimeTypeToVmType[runtimeType])
		return
	}
	vmInstancesManager, err := vmInstancesManagerProvider(chainId, nil)
	if err != nil {
		m.Log.Errorf("create instance manager failed, %v", err)
		return
	}
	// if the config in chainmaker.yml does not enable the vm(like docker go/java vm),
	// the vmInstancesManager will be nil.
	if vmInstancesManager == nil {
		m.Log.Debugf("vm instances manager of %v is nil", runtimeType)
		return
	}

	m.InstanceManagers[runtimeType] = vmInstancesManager

	m.Log.Infof("starting vm instance manager: %v", runtimeType)
	// use a go routine to start vm,
	// for in docker vm, it will block the main thread if the connection to engine is not established
	go func() {
		if err = vmInstancesManager.StartVM(); err != nil {
			m.Log.Errorf("failed to start %v vm manager", runtimeType)
		}
	}()
	m.aliveManagers[runtimeType] = struct{}{}
}
