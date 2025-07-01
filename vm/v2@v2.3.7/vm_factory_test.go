/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package vm

import (
	"encoding/hex"
	"errors"
	"reflect"
	"testing"

	acPb "chainmaker.org/chainmaker/pb-go/v2/accesscontrol"
	"chainmaker.org/chainmaker/pb-go/v2/common"
	commonPb "chainmaker.org/chainmaker/pb-go/v2/common"
	"chainmaker.org/chainmaker/pb-go/v2/config"
	"chainmaker.org/chainmaker/pb-go/v2/syscontract"
	"chainmaker.org/chainmaker/protocol/v2"
	"chainmaker.org/chainmaker/protocol/v2/mock"
	"chainmaker.org/chainmaker/protocol/v2/test"
	"github.com/golang/mock/gomock"
)

const (
	wxVmCodePathPrefix = "wxVmCodePathPrefix"
	contractName       = "contractName1"
	chainId            = "chainId"
	orgId              = "orgId"
	uid                = "uid1"
	role               = "role"
	method             = "method1"
	byteCode           = "bytecode1"
)

var (
	certString = `-----BEGIN CERTIFICATE-----
MIICiDCCAi2gAwIBAgIDDBDWMAoGCCqGSM49BAMCMIGKMQswCQYDVQQGEwJDTjEQ
MA4GA1UECBMHQmVpamluZzEQMA4GA1UEBxMHQmVpamluZzEfMB0GA1UEChMWd3gt
b3JnMi5jaGFpbm1ha2VyLm9yZzESMBAGA1UECxMJcm9vdC1jZXJ0MSIwIAYDVQQD
ExljYS53eC1vcmcyLmNoYWlubWFrZXIub3JnMB4XDTIwMTIwODA2NTM0M1oXDTI1
MTIwNzA2NTM0M1owgY8xCzAJBgNVBAYTAkNOMRAwDgYDVQQIEwdCZWlqaW5nMRAw
DgYDVQQHEwdCZWlqaW5nMR8wHQYDVQQKExZ3eC1vcmcyLmNoYWlubWFrZXIub3Jn
MQ4wDAYDVQQLEwVhZG1pbjErMCkGA1UEAxMiYWRtaW4xLnNpZ24ud3gtb3JnMi5j
aGFpbm1ha2VyLm9yZzBZMBMGByqGSM49AgEGCCqGSM49AwEHA0IABKzGp+roB4gO
nELoK/yWLlZX7z7+5AaGlsUC+TFP+pTuJ/A62nn1mBEN9dRKn/NlBDTfUko3P1BR
qu6JSWKzD96jezB5MA4GA1UdDwEB/wQEAwIBpjAPBgNVHSUECDAGBgRVHSUAMCkG
A1UdDgQiBCAaDND0hAYYe3kCtnoFbk0T/39GLD5+Tndnz6WkESqp7jArBgNVHSME
JDAigCDxj9Wz0+Py57Nj52LnWff6h7HJtKRi+in8vtQgBVhtITAKBggqhkjOPQQD
AgNJADBGAiEAziGp+kpd7UwxDLql8vFYOv94mwveOzRkYK2VJi5fmMQCIQDqXUO+
TNa0muhbqAMEqj7eLpuzh5mVUcTo2aHnleUcXw==
-----END CERTIFICATE-----
`

	txId = "0x86fa049857e0209aa7d9e616f7eb3b3b78ecfqb0"
)

func TestNewVmManager(t *testing.T) {
	log := &test.GoLogger{}
	c := gomock.NewController(t)
	defer c.Finish()
	rc := &config.ChainConfig{
		ChainId: chainId,
	}
	conf := mock.NewMockChainConf(c)
	conf.EXPECT().ChainConfig().Return(rc).AnyTimes()
	instanceManagers := make(map[commonPb.RuntimeType]protocol.VmInstancesManager)
	accessControl := &mock.MockAccessControlProvider{}
	chainNodesInfoProvider := mock.NewMockChainNodesInfoProvider(c)

	type args struct {
		instanceManagers       map[commonPb.RuntimeType]protocol.VmInstancesManager
		wxvmCodePathPrefix     string
		accessControl          protocol.AccessControlProvider
		chainNodesInfoProvider protocol.ChainNodesInfoProvider
		chainConf              protocol.ChainConf
	}

	tests := []struct {
		name string
		args args
		want protocol.VmManager
	}{
		{
			name: "good",
			args: args{
				instanceManagers:       instanceManagers,
				wxvmCodePathPrefix:     wxVmCodePathPrefix,
				accessControl:          accessControl,
				chainNodesInfoProvider: chainNodesInfoProvider,
				chainConf:              conf,
			},
			want: NewVmManager(instanceManagers, wxVmCodePathPrefix, accessControl, chainNodesInfoProvider, conf, log),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewVmManager(tt.args.instanceManagers, tt.args.wxvmCodePathPrefix, tt.args.accessControl,
				tt.args.chainNodesInfoProvider, tt.args.chainConf, log); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewVmManager() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestVmManagerImpl_GetAccessControl(t *testing.T) {
	accessControl := &mock.MockAccessControlProvider{}
	type fields struct {
		InstanceManagers       map[commonPb.RuntimeType]protocol.VmInstancesManager
		WxvmCodePath           string
		SnapshotManager        protocol.SnapshotManager
		AccessControl          protocol.AccessControlProvider
		ChainNodesInfoProvider protocol.ChainNodesInfoProvider
		ChainId                string
		Log                    protocol.Logger
		ChainConf              protocol.ChainConf
	}
	tests := []struct {
		name   string
		fields fields
		want   protocol.AccessControlProvider
	}{
		{
			name: "good",
			fields: fields{
				AccessControl: accessControl,
			},
			want: accessControl,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &VmManagerImpl{
				AccessControl: tt.fields.AccessControl,
			}
			if got := m.GetAccessControl(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetAccessControl() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestVmManagerImpl_GetChainNodesInfoProvider(t *testing.T) {
	c := gomock.NewController(t)
	defer c.Finish()
	chainNodesInfoProvider := mock.NewMockChainNodesInfoProvider(c)
	type fields struct {
		InstanceManagers       map[commonPb.RuntimeType]protocol.VmInstancesManager
		WxvmCodePath           string
		SnapshotManager        protocol.SnapshotManager
		AccessControl          protocol.AccessControlProvider
		ChainNodesInfoProvider protocol.ChainNodesInfoProvider
		ChainId                string
		Log                    protocol.Logger
		ChainConf              protocol.ChainConf
	}
	tests := []struct {
		name   string
		fields fields
		want   protocol.ChainNodesInfoProvider
	}{
		{
			name: "good",
			fields: fields{
				ChainNodesInfoProvider: chainNodesInfoProvider,
			},
			want: chainNodesInfoProvider,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &VmManagerImpl{
				InstanceManagers:       tt.fields.InstanceManagers,
				WxvmCodePath:           tt.fields.WxvmCodePath,
				SnapshotManager:        tt.fields.SnapshotManager,
				AccessControl:          tt.fields.AccessControl,
				ChainNodesInfoProvider: tt.fields.ChainNodesInfoProvider,
				ChainId:                tt.fields.ChainId,
				Log:                    tt.fields.Log,
				ChainConf:              tt.fields.ChainConf,
			}
			if got := m.GetChainNodesInfoProvider(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetChainNodesInfoProvider() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestVmManagerImpl_RunContract(t *testing.T) {

	c := gomock.NewController(t)
	defer c.Finish()

	log := &test.GoLogger{}

	chainConfig := &config.ChainConfig{
		AccountConfig: &config.GasAccountConfig{
			EnableGas:       true,
			DefaultGas:      uint64(10000),
			DefaultGasPrice: float32(0.1),
			InstallBaseGas:  uint64(200),
			InstallGasPrice: float32(0.3),
		},
	}

	m := mock.NewMockMember(c)
	ac := mock.NewMockAccessControlProvider(c)
	ac.EXPECT().NewMember(gomock.Any()).DoAndReturn(func(*acPb.Member) (protocol.Member, error) { return m, nil }).AnyTimes()
	context := mock.NewMockTxSimContext(c)
	context.EXPECT().GetBlockVersion().Return(uint32(220)).AnyTimes()
	context.EXPECT().GetLastChainConfig().Return(chainConfig).AnyTimes()
	context.EXPECT().GetAccessControl().Return(ac, nil).AnyTimes()
	context.EXPECT().RecordRuntimeTypeIntoCrossInfo(commonPb.RuntimeType_GO).Do(
		func(runtimeType commonPb.RuntimeType) {
			//todo
		}).AnyTimes()
	context.EXPECT().RemoveRuntimeTypeFromCrossInfo().Do(
		func() {
			//todo
		}).AnyTimes()
	vmInstancesManager := mock.NewMockVmInstancesManager(c)
	runtimeInstance := mock.NewMockRuntimeInstance(c)
	accessControlProvider := mock.NewMockAccessControlProvider(c)
	member := mock.NewMockMember(c)
	chainConf := mock.NewMockChainConf(c)

	chainConf.EXPECT().ChainConfig().Return(&config.ChainConfig{
		Contract: &config.ContractConfig{
			EnableSqlSupport:       false,
			DisabledNativeContract: nil,
		},
		Vm: &config.Vm{
			SupportList: nil,
			AddrType:    config.AddrType_ZXL,
		},
	}).AnyTimes()
	member.EXPECT().GetOrgId().Return(orgId).AnyTimes()
	member.EXPECT().GetRole().Return(protocol.Role(role)).AnyTimes()
	member.EXPECT().GetUid().Return(uid).AnyTimes()
	member.EXPECT().GetMemberId().Return("memberid").AnyTimes()

	accessControlProvider.EXPECT().NewMember(&acPb.Member{
		OrgId:      orgId,
		MemberType: acPb.MemberType_PUBLIC_KEY,
		MemberInfo: []byte(certString),
	}).Return(member, nil).AnyTimes()

	transaction := &common.Transaction{
		Payload: &common.Payload{
			ChainId:      chainId,
			TxId:         txId,
			Timestamp:    int64(1641532736),
			ContractName: contractName,
			TxType:       commonPb.TxType_QUERY_CONTRACT,
		},
	}
	context.EXPECT().GetBlockVersion().Return(uint32(220)).AnyTimes()
	context.EXPECT().GetTx().Return(transaction).AnyTimes()
	context.EXPECT().GetBlockHeight().Return(uint64(0)).AnyTimes()
	context.EXPECT().GetSender().Return(&acPb.Member{
		OrgId:      orgId,
		MemberType: acPb.MemberType_PUBLIC_KEY,
		MemberInfo: []byte(certString),
	}).AnyTimes()
	contract := &commonPb.Contract{
		Name:        contractName,
		Version:     version,
		RuntimeType: commonPb.RuntimeType_GO,
		Status:      0,
		Creator: &acPb.MemberFull{
			OrgId:      orgId,
			MemberType: acPb.MemberType_PUBLIC_KEY,
			MemberInfo: []byte(certString),
		}}
	context.EXPECT().GetContractByName(contractName).Return(contract, nil).AnyTimes()
	vmInstancesManager.EXPECT().NewRuntimeInstance(context, chainId, method, wxVmCodePathPrefix, contract, []byte(byteCode), log).Return(runtimeInstance, nil).AnyTimes()
	runtimeInstance.EXPECT().Invoke(contract, method, []byte(byteCode), gomock.Any(), context, uint64(0)).Return(&common.ContractResult{}, protocol.ExecOrderTxTypeNormal).AnyTimes()

	type fields struct {
		InstanceManagers       map[commonPb.RuntimeType]protocol.VmInstancesManager
		WxvmCodePath           string
		SnapshotManager        protocol.SnapshotManager
		AccessControl          protocol.AccessControlProvider
		ChainNodesInfoProvider protocol.ChainNodesInfoProvider
		ChainId                string
		Log                    protocol.Logger
		ChainConf              protocol.ChainConf
	}
	type args struct {
		contract   *commonPb.Contract
		method     string
		byteCode   []byte
		parameters map[string][]byte
		txContext  protocol.TxSimContext
		gasUsed    uint64
		refTxType  common.TxType
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   *commonPb.ContractResult
		want1  protocol.ExecOrderTxType
		want2  common.TxStatusCode
	}{
		{
			name: "bad1",
			fields: fields{
				InstanceManagers: map[commonPb.RuntimeType]protocol.VmInstancesManager{
					commonPb.RuntimeType_GO: vmInstancesManager,
				},
				WxvmCodePath:           wxVmCodePathPrefix,
				SnapshotManager:        nil,
				AccessControl:          accessControlProvider,
				ChainNodesInfoProvider: nil,
				ChainId:                chainId,
				Log:                    log,
				ChainConf:              chainConf,
			},

			args: args{
				contract: &common.Contract{
					Name:        contractName,
					Version:     "",
					RuntimeType: commonPb.RuntimeType_GO,
					Status:      0,
					Creator:     nil,
				},
				method:     method,
				byteCode:   []byte(byteCode),
				parameters: nil,
				txContext:  context,
				gasUsed:    0,
				refTxType:  0,
			},
			want:  &common.ContractResult{},
			want1: protocol.ExecOrderTxTypeNormal,
			want2: common.TxStatusCode_SUCCESS,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &VmManagerImpl{
				InstanceManagers:       tt.fields.InstanceManagers,
				WxvmCodePath:           tt.fields.WxvmCodePath,
				SnapshotManager:        tt.fields.SnapshotManager,
				AccessControl:          tt.fields.AccessControl,
				ChainNodesInfoProvider: tt.fields.ChainNodesInfoProvider,
				ChainId:                tt.fields.ChainId,
				Log:                    tt.fields.Log,
				ChainConf:              tt.fields.ChainConf,
			}

			got, got1, got2 := m.RunContract(tt.args.contract, tt.args.method, tt.args.byteCode, tt.args.parameters, tt.args.txContext, tt.args.gasUsed, tt.args.refTxType)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("RunContract() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("RunContract() got1 = %v, want %v", got1, tt.want1)
			}
			if got2 != tt.want2 {
				t.Errorf("RunContract() got2 = %v, want %v", got2, tt.want2)
			}
		})
	}
}

func TestVmManagerImpl_Start(t *testing.T) {
	type fields struct {
		InstanceManagers       map[commonPb.RuntimeType]protocol.VmInstancesManager
		WxvmCodePath           string
		SnapshotManager        protocol.SnapshotManager
		AccessControl          protocol.AccessControlProvider
		ChainNodesInfoProvider protocol.ChainNodesInfoProvider
		ChainId                string
		Log                    protocol.Logger
		ChainConf              protocol.ChainConf
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "good",
			fields: fields{
				InstanceManagers:       nil,
				WxvmCodePath:           "",
				SnapshotManager:        nil,
				AccessControl:          nil,
				ChainNodesInfoProvider: nil,
				ChainId:                "",
				Log:                    nil,
				ChainConf:              nil,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &VmManagerImpl{
				InstanceManagers:       tt.fields.InstanceManagers,
				WxvmCodePath:           tt.fields.WxvmCodePath,
				SnapshotManager:        tt.fields.SnapshotManager,
				AccessControl:          tt.fields.AccessControl,
				ChainNodesInfoProvider: tt.fields.ChainNodesInfoProvider,
				ChainId:                tt.fields.ChainId,
				Log:                    tt.fields.Log,
				ChainConf:              tt.fields.ChainConf,
			}
			if err := m.Start(); (err != nil) != tt.wantErr {
				t.Errorf("Start() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestVmManagerImpl_Stop(t *testing.T) {
	type fields struct {
		InstanceManagers       map[commonPb.RuntimeType]protocol.VmInstancesManager
		WxvmCodePath           string
		SnapshotManager        protocol.SnapshotManager
		AccessControl          protocol.AccessControlProvider
		ChainNodesInfoProvider protocol.ChainNodesInfoProvider
		ChainId                string
		Log                    protocol.Logger
		ChainConf              protocol.ChainConf
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "bad",
			fields: fields{
				InstanceManagers:       nil,
				WxvmCodePath:           "",
				SnapshotManager:        nil,
				AccessControl:          nil,
				ChainNodesInfoProvider: nil,
				ChainId:                "",
				Log:                    nil,
				ChainConf:              nil,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &VmManagerImpl{
				InstanceManagers:       tt.fields.InstanceManagers,
				WxvmCodePath:           tt.fields.WxvmCodePath,
				SnapshotManager:        tt.fields.SnapshotManager,
				AccessControl:          tt.fields.AccessControl,
				ChainNodesInfoProvider: tt.fields.ChainNodesInfoProvider,
				ChainId:                tt.fields.ChainId,
				Log:                    tt.fields.Log,
				ChainConf:              tt.fields.ChainConf,
			}
			if err := m.Stop(); (err != nil) != tt.wantErr {
				t.Errorf("Stop() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestVmManagerImpl_invokeUserContractByRuntime(t *testing.T) {
	c := gomock.NewController(t)
	defer c.Finish()
	log := &test.GoLogger{}
	context := mock.NewMockTxSimContext(c)
	manager := mock.NewMockVmInstancesManager(c)
	runtimeInstance := mock.NewMockRuntimeInstance(c)
	accessControlProvider := mock.NewMockAccessControlProvider(c)
	member := mock.NewMockMember(c)
	chainConf := mock.NewMockChainConf(c)

	chainConf.EXPECT().ChainConfig().Return(&config.ChainConfig{
		Contract: &config.ContractConfig{
			EnableSqlSupport:       false,
			DisabledNativeContract: nil,
		},
		Vm: &config.Vm{
			SupportList: nil,
			AddrType:    config.AddrType_ZXL,
		},
	}).AnyTimes()
	member.EXPECT().GetOrgId().Return(orgId).AnyTimes()
	member.EXPECT().GetRole().Return(protocol.Role(role)).AnyTimes()
	member.EXPECT().GetUid().Return(uid).AnyTimes()
	member.EXPECT().GetMemberId().Return("memberid").AnyTimes()

	accessControlProvider.EXPECT().NewMember(&acPb.Member{
		OrgId:      orgId,
		MemberType: acPb.MemberType_PUBLIC_KEY,
		MemberInfo: []byte(certString),
	}).Return(member, nil).AnyTimes()

	transaction := &common.Transaction{
		Payload: &common.Payload{
			ChainId:      chainId,
			TxId:         txId,
			Timestamp:    int64(1641532736),
			ContractName: contractName,
		},
	}
	context.EXPECT().GetTx().Return(transaction).AnyTimes()
	context.EXPECT().GetBlockHeight().Return(uint64(0)).AnyTimes()
	context.EXPECT().GetSender().Return(&acPb.Member{
		OrgId:      orgId,
		MemberType: acPb.MemberType_PUBLIC_KEY,
		MemberInfo: []byte(certString),
	})
	context.EXPECT().GetBlockVersion().Return(uint32(220)).AnyTimes()

	contract := &commonPb.Contract{
		Name:        contractName,
		Version:     version,
		RuntimeType: commonPb.RuntimeType_GO,
		Status:      0,
		Creator: &acPb.MemberFull{
			OrgId:      orgId,
			MemberType: acPb.MemberType_PUBLIC_KEY,
			MemberInfo: []byte(certString),
		}}
	manager.EXPECT().NewRuntimeInstance(context, chainId, method, wxVmCodePathPrefix, contract, []byte(byteCode), log).Return(runtimeInstance, nil).AnyTimes()

	runtimeInstance.EXPECT().Invoke(contract, method, []byte(byteCode), gomock.Any(), context, uint64(0)).Return(&common.ContractResult{
		Code:          uint32(1),
		Result:        nil,
		Message:       "",
		GasUsed:       0,
		ContractEvent: nil,
	}, protocol.ExecOrderTxTypeNormal).AnyTimes()

	type fields struct {
		InstanceManagers       map[commonPb.RuntimeType]protocol.VmInstancesManager
		WxvmCodePath           string
		SnapshotManager        protocol.SnapshotManager
		AccessControl          protocol.AccessControlProvider
		ChainNodesInfoProvider protocol.ChainNodesInfoProvider
		ChainId                string
		Log                    protocol.Logger
		ChainConf              protocol.ChainConf
	}
	type args struct {
		contract   *commonPb.Contract
		method     string
		parameters map[string][]byte
		txContext  protocol.TxSimContext
		byteCode   []byte
		gasUsed    uint64
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   *commonPb.ContractResult
		want1  protocol.ExecOrderTxType
		want2  common.TxStatusCode
	}{
		{
			name: "good",
			fields: fields{
				InstanceManagers: map[commonPb.RuntimeType]protocol.VmInstancesManager{
					commonPb.RuntimeType_GO: manager,
				},
				WxvmCodePath:           wxVmCodePathPrefix,
				SnapshotManager:        nil,
				AccessControl:          accessControlProvider,
				ChainNodesInfoProvider: nil,
				ChainId:                chainId,
				Log:                    log,
				ChainConf:              chainConf,
			},
			args: args{
				contract: &common.Contract{
					Name:        contractName,
					Version:     version,
					RuntimeType: commonPb.RuntimeType_GO,
					Status:      0,
					Creator: &acPb.MemberFull{
						OrgId:      orgId,
						MemberType: acPb.MemberType_PUBLIC_KEY,
						MemberInfo: []byte(certString),
					},
				},
				method:     method,
				parameters: make(map[string][]byte),
				txContext:  context,
				byteCode:   []byte(byteCode),
				gasUsed:    0,
			},
			want: &commonPb.ContractResult{
				Code:          1,
				Result:        nil,
				Message:       "",
				GasUsed:       0,
				ContractEvent: nil,
			},
			want1: protocol.ExecOrderTxTypeNormal,
			want2: common.TxStatusCode_CONTRACT_FAIL,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &VmManagerImpl{
				InstanceManagers:       tt.fields.InstanceManagers,
				WxvmCodePath:           tt.fields.WxvmCodePath,
				SnapshotManager:        tt.fields.SnapshotManager,
				AccessControl:          tt.fields.AccessControl,
				ChainNodesInfoProvider: tt.fields.ChainNodesInfoProvider,
				ChainId:                tt.fields.ChainId,
				Log:                    tt.fields.Log,
				ChainConf:              tt.fields.ChainConf,
			}
			got, got1, got2 := m.invokeUserContractByRuntime(tt.args.contract, tt.args.method, tt.args.parameters, tt.args.txContext, tt.args.byteCode, tt.args.gasUsed)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("invokeUserContractByRuntime() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("invokeUserContractByRuntime() got1 = %v, want %v", got1, tt.want1)
			}
			if got2 != tt.want2 {
				t.Errorf("invokeUserContractByRuntime() got2 = %v, want %v", got2, tt.want2)
			}
		})
	}
}

func TestVmManagerImpl_isUserContract(t *testing.T) {
	type fields struct {
		InstanceManagers       map[commonPb.RuntimeType]protocol.VmInstancesManager
		WxvmCodePath           string
		SnapshotManager        protocol.SnapshotManager
		AccessControl          protocol.AccessControlProvider
		ChainNodesInfoProvider protocol.ChainNodesInfoProvider
		ChainId                string
		Log                    protocol.Logger
		ChainConf              protocol.ChainConf
	}
	type args struct {
		refTxType common.TxType
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		{
			name: "true",
			args: args{refTxType: common.TxType_INVOKE_CONTRACT},
			want: true,
		},
		{
			name: "false",
			args: args{refTxType: common.TxType_QUERY_CONTRACT},
			want: true,
		},
		{
			name: "false",
			args: args{refTxType: common.TxType_ARCHIVE},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &VmManagerImpl{
				InstanceManagers:       tt.fields.InstanceManagers,
				WxvmCodePath:           tt.fields.WxvmCodePath,
				SnapshotManager:        tt.fields.SnapshotManager,
				AccessControl:          tt.fields.AccessControl,
				ChainNodesInfoProvider: tt.fields.ChainNodesInfoProvider,
				ChainId:                tt.fields.ChainId,
				Log:                    tt.fields.Log,
				ChainConf:              tt.fields.ChainConf,
			}
			if got := m.isUserContract(tt.args.refTxType); got != tt.want {
				t.Errorf("isUserContract() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestVmManagerImpl_runNativeContract(t *testing.T) {
	c := gomock.NewController(t)
	defer c.Finish()
	conf := mock.NewMockChainConf(c)
	rconf := &config.ChainConfig{}
	conf.EXPECT().ChainConfig().Return(rconf).AnyTimes()
	initInstance(c)

	type fields struct {
		InstanceManagers       map[commonPb.RuntimeType]protocol.VmInstancesManager
		WxvmCodePath           string
		SnapshotManager        protocol.SnapshotManager
		AccessControl          protocol.AccessControlProvider
		ChainNodesInfoProvider protocol.ChainNodesInfoProvider
		ChainId                string
		Log                    protocol.Logger
		ChainConf              protocol.ChainConf
	}
	type args struct {
		contract   *commonPb.Contract
		method     string
		parameters map[string][]byte
		txContext  protocol.TxSimContext
		gasUsed    uint64
	}

	tx := &commonPb.Transaction{
		Payload: &commonPb.Payload{
			TxId: "test-transaction-id-123",
		},
	}

	snapshot := mock.NewMockSnapshot(c)
	snapshot.EXPECT().GetSnapshotSize().Return(0).AnyTimes()
	ctx := NewTxSimContext(nil, snapshot, tx, 220, &test.GoLogger{})
	tests := []struct {
		name   string
		fields fields
		args   args
		want   *commonPb.ContractResult
		want1  protocol.ExecOrderTxType
		want2  common.TxStatusCode
	}{
		{
			name: "bad1",
			fields: fields{
				ChainConf: conf,
				Log:       &test.GoLogger{},
				ChainId:   chainId,
			},
			args: args{
				txContext: ctx,
				contract:  &commonPb.Contract{Name: "bad1"},
			},
			want: &common.ContractResult{
				Code:    uint32(1),
				Message: "the contractName is not exist",
			},
			want1: protocol.ExecOrderTxTypeNormal,
			want2: common.TxStatusCode_CONTRACT_FAIL,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &VmManagerImpl{
				InstanceManagers:       tt.fields.InstanceManagers,
				WxvmCodePath:           tt.fields.WxvmCodePath,
				SnapshotManager:        tt.fields.SnapshotManager,
				AccessControl:          tt.fields.AccessControl,
				ChainNodesInfoProvider: tt.fields.ChainNodesInfoProvider,
				ChainId:                tt.fields.ChainId,
				Log:                    tt.fields.Log,
				ChainConf:              tt.fields.ChainConf,
			}
			got, got1, got2 := m.runNativeContract(tt.args.contract, tt.args.method, tt.args.parameters, tt.args.txContext, tt.args.gasUsed)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("runNativeContract() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("runNativeContract() got1 = %v, want %v", got1, tt.want1)
			}
			if got2 != tt.want2 {
				t.Errorf("runNativeContract() got2 = %v, want %v", got2, tt.want2)
			}
		})
	}
}

func TestVmManagerImpl_runUserContract(t *testing.T) {
	c := gomock.NewController(t)
	defer c.Finish()
	context := mock.NewMockTxSimContext(c)
	payload := &common.Payload{
		ChainId:        "",
		TxType:         commonPb.TxType_INVOKE_CONTRACT,
		TxId:           txId,
		Timestamp:      0,
		ExpirationTime: 0,
		ContractName:   "",
		Method:         "",
		Parameters:     nil,
		Sequence:       0,
		Limit:          nil,
	}

	transaction := &common.Transaction{
		Payload:   payload,
		Sender:    nil,
		Endorsers: nil,
		Result:    nil,
	}

	chainConfig := &config.ChainConfig{
		AccountConfig: &config.GasAccountConfig{
			EnableGas:       true,
			DefaultGas:      uint64(10000),
			DefaultGasPrice: float32(0.1),
			InstallBaseGas:  uint64(200),
			InstallGasPrice: float32(0.3),
		},
	}

	context.EXPECT().GetTx().Return(transaction).AnyTimes()
	context.EXPECT().GetBlockVersion().Return(uint32(220)).AnyTimes()
	context.EXPECT().GetLastChainConfig().Return(chainConfig).AnyTimes()

	acPbMember := &acPb.Member{
		OrgId:      "",
		MemberType: acPb.MemberType_CERT,
		MemberInfo: []byte(certString),
	}
	acPbFull := &acPb.MemberFull{
		OrgId:      acPbMember.OrgId,
		MemberType: acPbMember.MemberType,
		MemberInfo: acPbMember.MemberInfo,
	}
	context.EXPECT().GetSender().Return(acPbMember).AnyTimes()

	contract := &common.Contract{
		Name:        "",
		Version:     "",
		RuntimeType: common.RuntimeType_GO,
		Status:      0,
		Creator:     acPbFull,
	}

	log := &test.GoLogger{}
	instanceManagers := make(map[commonPb.RuntimeType]protocol.VmInstancesManager)
	newMockVmManager := mock.NewMockVmInstancesManager(c)
	instance := mock.NewMockRuntimeInstance(c)
	newMockVmManager.EXPECT().NewRuntimeInstance(context, chainId, method, wxVmCodePathPrefix, contract, []byte(byteCode), log).Return(instance, nil).AnyTimes()
	instanceManagers[common.RuntimeType_GO] = newMockVmManager

	accessControl := mock.NewMockAccessControlProvider(c)
	//accessControl.EXPECT().GetCertFromCache(gomock.Any()).Return([]byte(certString), nil)
	mockMember := mock.NewMockMember(c)
	m := &VmManagerImpl{
		InstanceManagers:       instanceManagers,
		WxvmCodePath:           wxVmCodePathPrefix,
		SnapshotManager:        nil,
		AccessControl:          accessControl,
		ChainNodesInfoProvider: nil,
		ChainId:                chainId,
		Log:                    log,
		ChainConf:              nil,
	}

	sender := context.GetSender()
	member, _ := m.getFullCertMember(sender, context)
	accessControl.EXPECT().NewMember(member).Return(mockMember, errors.New("stop")).AnyTimes()
	type fields struct {
		InstanceManagers       map[commonPb.RuntimeType]protocol.VmInstancesManager
		WxvmCodePath           string
		SnapshotManager        protocol.SnapshotManager
		AccessControl          protocol.AccessControlProvider
		ChainNodesInfoProvider protocol.ChainNodesInfoProvider
		ChainId                string
		Log                    protocol.Logger
		ChainConf              protocol.ChainConf
	}
	type args struct {
		contract   *commonPb.Contract
		method     string
		byteCode   []byte
		parameters map[string][]byte
		txContext  protocol.TxSimContext
		gasUsed    uint64
		refTxType  common.TxType
	}
	tests := []struct {
		name               string
		fields             fields
		args               args
		wantContractResult *commonPb.ContractResult
		wantSpecialTxType  protocol.ExecOrderTxType
		wantCode           common.TxStatusCode
	}{
		{
			name: "bad1",
			args: args{
				contract:   contract,
				method:     method,
				byteCode:   []byte(byteCode),
				parameters: nil,
				txContext:  context,
				gasUsed:    0,
				refTxType:  0,
			},
			wantContractResult: &common.ContractResult{
				Code:          uint32(1),
				Result:        nil,
				Message:       "failed to unmarshal sender \"GO\"",
				GasUsed:       0,
				ContractEvent: nil,
			},
			wantSpecialTxType: protocol.ExecOrderTxTypeNormal,
			wantCode:          common.TxStatusCode_UNMARSHAL_SENDER_FAILED,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotContractResult, gotSpecialTxType, gotCode := m.runUserContract(tt.args.contract, tt.args.method, tt.args.byteCode, tt.args.parameters, tt.args.txContext, tt.args.gasUsed, tt.args.refTxType)
			if !reflect.DeepEqual(gotContractResult, tt.wantContractResult) {
				t.Errorf("runUserContract() gotContractResult = %v, want %v", gotContractResult, tt.wantContractResult)
			}
			if gotSpecialTxType != tt.wantSpecialTxType {
				t.Errorf("runUserContract() gotSpecialTxType = %v, want %v", gotSpecialTxType, tt.wantSpecialTxType)
			}
			if gotCode != tt.wantCode {
				t.Errorf("runUserContract() gotCode = %v, want %v", gotCode, tt.wantCode)
			}
		})
	}
}

func Test_getFullCertMember(t *testing.T) {

	memberInfoHex := hex.EncodeToString([]byte(certString))
	c := gomock.NewController(t)
	defer c.Finish()
	context := mock.NewMockTxSimContext(c)
	context.EXPECT().Get(syscontract.SystemContract_CERT_MANAGE.String(), []byte(memberInfoHex)).Return([]byte(certString), nil).AnyTimes()
	context.EXPECT().GetBlockVersion().Return(uint32(220)).AnyTimes()
	type args struct {
		sender    *acPb.Member
		txContext protocol.TxSimContext
	}

	accessControl := mock.NewMockAccessControlProvider(c)
	//accessControl.EXPECT().GetCertFromCache(gomock.Any()).Return([]byte(certString), nil)
	m := &VmManagerImpl{
		AccessControl: accessControl,
	}

	tests := []struct {
		name  string
		args  args
		want  *acPb.Member
		want1 common.TxStatusCode
	}{
		{
			name: "good",
			args: args{
				sender: &acPb.Member{
					OrgId:      orgId,
					MemberType: acPb.MemberType_CERT_HASH,
					MemberInfo: []byte(certString),
				},
				txContext: context,
			},
			want: &acPb.Member{
				OrgId:      orgId,
				MemberType: acPb.MemberType_CERT,
				MemberInfo: []byte(certString),
			},
			want1: commonPb.TxStatusCode_SUCCESS,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := m.getFullCertMember(tt.args.sender, tt.args.txContext)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getFullCertMember() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("getFullCertMember() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}
