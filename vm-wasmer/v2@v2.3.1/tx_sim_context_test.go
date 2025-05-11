/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package vm

import (
	"errors"
	"reflect"
	"testing"

	configPb "chainmaker.org/chainmaker/pb-go/v2/config"

	"chainmaker.org/chainmaker/protocol/v2/test"

	acPb "chainmaker.org/chainmaker/pb-go/v2/accesscontrol"
	"chainmaker.org/chainmaker/pb-go/v2/common"
	"chainmaker.org/chainmaker/protocol/v2"
	"chainmaker.org/chainmaker/protocol/v2/mock"
	"github.com/golang/mock/gomock"
)

const (
	limit = "limit"
)

var log = &test.GoLogger{}

type txSimContextImplTestFields struct {
	txExecSeq int
	txResult  *common.Result
	tx        *common.Transaction
	// txReadKeyMap     map[string]*common.TxRead
	// txWriteKeyMap    map[string]*common.TxWrite
	txRWSetWithDepth []map[string]*rwSet // RWSet of a transaction
	txWriteKeySql    []*common.TxWrite
	snapshot         protocol.Snapshot
	vmManager        protocol.VmManager
	currentDepth     int
}

func TestNewTxSimContext(t *testing.T) {
	c := gomock.NewController(t)
	defer c.Finish()

	manager := mock.NewMockVmManager(c)
	snapshot := mock.NewMockSnapshot(c)
	snapshot.EXPECT().GetSnapshotSize().Return(0).AnyTimes()
	tx := &common.Transaction{}
	type args struct {
		vmManager    protocol.VmManager
		snapshot     protocol.Snapshot
		tx           *common.Transaction
		blockVersion uint32
	}
	tests := []struct {
		name string
		args args
		want protocol.TxSimContext
	}{
		{
			name: "t1",
			args: args{
				vmManager:    manager,
				snapshot:     snapshot,
				tx:           tx,
				blockVersion: 0,
			},
			want: NewTxSimContext(manager, snapshot, tx, 0, &test.GoLogger{}),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewTxSimContext(tt.args.vmManager, tt.args.snapshot, tt.args.tx, tt.args.blockVersion, &test.GoLogger{}); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewTxSimContext() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_appendWithSplitChar(t *testing.T) {
	type args struct {
		s1        []byte
		s2        []byte
		splitChar string
	}
	tests := []struct {
		name string
		args args
		want []byte
	}{
		{
			name: "t1",
			args: args{
				s1:        []byte("s1"),
				s2:        []byte("s2"),
				splitChar: "splitChar1",
			},
			want: []byte("s1splitChar1s2"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := appendWithSplitChar(tt.args.s1, tt.args.s2, tt.args.splitChar); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("appendWithSplitChar() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_constructKey(t *testing.T) {
	type args struct {
		contractName string
		key          []byte
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "t1",
			args: args{
				contractName: contractName,
				key:          []byte(key),
			},
			want: contractName + constructKeySeparator + key,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := constructKey(tt.args.contractName, tt.args.key); got != tt.want {
				t.Errorf("constructKey() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_constructKeyWithDAGIndex(t *testing.T) {
	type args struct {
		contractName string
		key          []byte
		blockHeight  uint64
		txId         string
		dagIndex     uint64
	}
	tests := []struct {
		name string
		args args
		want []byte
	}{
		{
			name: "t1",
			args: args{
				contractName: contractName,
				key:          []byte(key),
				txId:         txId,
			},
			want: []byte("kcontractName1#key1#\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000#\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000$0x86fa049857e0209aa7d9e616f7eb3b3b78ecfqb0"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := constructKeyWithDAGIndex(tt.args.contractName, tt.args.key, tt.args.blockHeight, tt.args.txId, tt.args.dagIndex); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("constructKeyWithDAGIndex() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_txSimContextImpl_CallContract(t *testing.T) {
	c := gomock.NewController(t)
	defer c.Finish()
	vmManager := mock.NewMockVmManager(c)
	contract := &common.Contract{
		Name:        contractName,
		Version:     "",
		RuntimeType: 0,
		Status:      0,
		Creator:     nil,
	}
	parameter := map[string][]byte{
		"key": []byte(value),
	}

	contractResult := &common.ContractResult{}
	execOrderTxType := protocol.ExecOrderTxTypeNormal
	codeStatus := common.TxStatusCode_SUCCESS
	vmManager.EXPECT().RunContract(contract, method, []byte(byteCode), parameter, gomock.Any(), gomock.Any(), gomock.Any()).Return(contractResult, execOrderTxType, codeStatus).AnyTimes()
	type args struct {
		contract  *common.Contract
		method    string
		byteCode  []byte
		parameter map[string][]byte
		gasUsed   uint64
		refTxType common.TxType
	}

	tests := []struct {
		name   string
		fields txSimContextImplTestFields
		args   args
		want   *common.ContractResult
		want1  protocol.ExecOrderTxType
		want2  common.TxStatusCode
	}{
		{
			name: "t1",
			fields: txSimContextImplTestFields{
				currentDepth: 0,
				vmManager:    vmManager,
			},
			args: args{
				contract:  contract,
				method:    method,
				byteCode:  []byte(byteCode),
				parameter: parameter,
			},
			want:  &common.ContractResult{},
			want1: protocol.ExecOrderTxTypeNormal,
			want2: common.TxStatusCode_SUCCESS,
		},

		{
			name: "t2",
			fields: txSimContextImplTestFields{
				vmManager:    vmManager,
				currentDepth: 6,
			},
			args: args{
				contract:  contract,
				method:    method,
				byteCode:  []byte(byteCode),
				parameter: parameter,
			},
			want: &common.ContractResult{
				Code:    uint32(1),
				Message: "CallContract too depth 7",
			},
			want1: protocol.ExecOrderTxTypeNormal,
			want2: common.TxStatusCode_CONTRACT_TOO_DEEP_FAILED,
		},
		{
			name: "t3",
			fields: txSimContextImplTestFields{
				vmManager: vmManager,
			},
			args: args{
				contract:  contract,
				method:    method,
				byteCode:  []byte(byteCode),
				parameter: parameter,
				gasUsed:   1e11,
			},
			want: &common.ContractResult{
				Code:    uint32(1),
				Message: "There is not enough gas, gasUsed 100000000000 GasLimit 10000000000 ",
			},
			want1: protocol.ExecOrderTxTypeNormal,
			want2: common.TxStatusCode_CONTRACT_FAIL,
		},
	}
	for _, tt := range tests {
		txRWSet := make([]map[string]*rwSet, 5)
		txRWSet[0] = make(map[string]*rwSet)
		t.Run(tt.name, func(t *testing.T) {
			s := &txSimContextImpl{
				vmManager:    tt.fields.vmManager,
				currentDepth: tt.fields.currentDepth,
				logger:       log,

				txRWSetWithDepth:                 txRWSet,
				crossInfo:                        0,
				rowCache:                         make([]map[int32]interface{}, 5),
				txWriteKeySql:                    make([]*common.TxWrite, 0),
				txWriteKeyDdlSql:                 make([]*common.TxWrite, 0),
				gasUsed:                          0,
				hisResult:                        make([]*callContractResult, 0),
				usedSimContextIterator:           make([]*SimContextIterator, 0),
				usedSimContextKeyHistoryIterator: make([]*SimContextKeyHistoryIterator, 0),
			}
			got, got1, got2 := s.CallContract(tt.args.contract, tt.args.method, tt.args.byteCode, tt.args.parameter, tt.args.gasUsed, tt.args.refTxType)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("CallContract() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("CallContract() got1 = %v, want %v", got1, tt.want1)
			}
			if got2 != tt.want2 {
				t.Errorf("CallContract() got2 = %v, want %v", got2, tt.want2)
			}
		})
	}
}

func Test_txSimContextImpl_Del(t *testing.T) {
	type args struct {
		contractName string
		key          []byte
	}
	tests := []struct {
		name    string
		fields  txSimContextImplTestFields
		args    args
		wantErr bool
	}{
		// map[string]*common.TxWrite
		{
			name: "t1",
			fields: txSimContextImplTestFields{
				currentDepth: 0,
				txRWSetWithDepth: []map[string]*rwSet{
					{
						contractName: &rwSet{
							txWriteKeyMap: map[string]*common.TxWrite{
								"": {
									Key:          []byte(key),
									Value:        []byte(value),
									ContractName: contractName,
								},
							},
						},
					},
				},
				// txWriteKeyMap: map[string]*common.TxWrite{
				// 	key: {
				// 		Key:          []byte(key),
				// 		Value:        []byte(value),
				// 		ContractName: contractName,
				// 	},
				// },
			},
			args: args{
				contractName: contractName,
				key:          []byte(key),
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &txSimContextImpl{
				txRWSetWithDepth: tt.fields.txRWSetWithDepth,
				txWriteKeySql:    tt.fields.txWriteKeySql,
				logger:           log,
			}
			if err := s.Del(tt.args.contractName, tt.args.key); (err != nil) != tt.wantErr {
				t.Errorf("Del() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_txSimContextImpl_Get(t *testing.T) {
	c := gomock.NewController(t)
	defer c.Finish()
	snapshot := mock.NewMockSnapshot(c)
	snapshot.EXPECT().GetKey(gomock.Any(), gomock.Any(), gomock.Any()).Return([]byte(value), nil).AnyTimes()

	type args struct {
		contractName string
		key          []byte
	}
	tests := []struct {
		name    string
		fields  txSimContextImplTestFields
		args    args
		want    []byte
		wantErr bool
	}{
		{
			name: "t1",
			fields: txSimContextImplTestFields{
				currentDepth: 0,
				txRWSetWithDepth: []map[string]*rwSet{
					{
						contractName: &rwSet{
							txWriteKeyMap: make(map[string]*common.TxWrite, 8),
							txReadKeyMap:  make(map[string]*common.TxRead, 8),
						},
					},
				},
				snapshot: snapshot,
			},
			args: args{
				contractName: contractName,
				key:          []byte(key),
			},
			want:    []byte("value1"),
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &txSimContextImpl{
				txRWSetWithDepth: tt.fields.txRWSetWithDepth,
				snapshot:         tt.fields.snapshot,
				logger:           log,
			}
			got, err := s.Get(tt.args.contractName, tt.args.key)
			if (err != nil) != tt.wantErr {
				t.Errorf("Get() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Get() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_txSimContextImpl_GetAccessControl(t *testing.T) {
	c := gomock.NewController(t)
	defer c.Finish()
	accessControlProvider := mock.NewMockAccessControlProvider(c)
	manager := mock.NewMockVmManager(c)
	manager.EXPECT().GetAccessControl().Return(accessControlProvider).AnyTimes()

	manager2 := mock.NewMockVmManager(c)
	manager2.EXPECT().GetAccessControl().Return(nil).AnyTimes()
	tests := []struct {
		name    string
		fields  txSimContextImplTestFields
		want    protocol.AccessControlProvider
		wantErr bool
	}{
		{
			name: "t1",
			fields: txSimContextImplTestFields{
				vmManager: manager,
			},
			want:    accessControlProvider,
			wantErr: false,
		},
		{
			name: "t1",
			fields: txSimContextImplTestFields{
				vmManager: manager2,
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &txSimContextImpl{
				vmManager: tt.fields.vmManager,
				logger:    log,
			}
			got, err := s.GetAccessControl()
			if (err != nil) != tt.wantErr {
				t.Errorf("GetAccessControl() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetAccessControl() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_txSimContextImpl_GetBlockHeight(t *testing.T) {
	c := gomock.NewController(t)
	defer c.Finish()
	snapshot := mock.NewMockSnapshot(c)
	snapshot.EXPECT().GetBlockHeight().Return(uint64(0)).AnyTimes()
	tests := []struct {
		name   string
		fields txSimContextImplTestFields
		want   uint64
	}{
		{
			name: "t1",
			fields: txSimContextImplTestFields{
				snapshot: snapshot,
			},
			want: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &txSimContextImpl{
				snapshot: tt.fields.snapshot,
				logger:   log,
			}
			if got := s.GetBlockHeight(); got != tt.want {
				t.Errorf("GetBlockHeight() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_txSimContextImpl_GetBlockProposer(t *testing.T) {
	c := gomock.NewController(t)
	defer c.Finish()
	snapshot := mock.NewMockSnapshot(c)
	member := &acPb.Member{
		OrgId:      "",
		MemberType: 0,
		MemberInfo: nil,
	}
	snapshot.EXPECT().GetBlockProposer().Return(member).AnyTimes()

	tests := []struct {
		name   string
		fields txSimContextImplTestFields
		want   *acPb.Member
	}{
		{
			name: "t1",
			fields: txSimContextImplTestFields{
				snapshot: snapshot,
			},
			want: member,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &txSimContextImpl{
				snapshot: tt.fields.snapshot,
				logger:   log,
			}
			if got := s.GetBlockProposer(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetBlockProposer() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_txSimContextImpl_GetBlockTimestamp(t *testing.T) {
	c := gomock.NewController(t)
	defer c.Finish()
	snapshot := mock.NewMockSnapshot(c)
	snapshot.EXPECT().GetBlockTimestamp().Return(int64(1641440597)).AnyTimes()

	tests := []struct {
		name   string
		fields txSimContextImplTestFields
		want   int64
	}{
		{
			name: "t1",
			fields: txSimContextImplTestFields{
				snapshot: snapshot,
			},
			want: 1641440597,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &txSimContextImpl{
				snapshot: tt.fields.snapshot,
				logger:   log,
			}
			if got := s.GetBlockTimestamp(); got != tt.want {
				t.Errorf("GetBlockTimestamp() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_txSimContextImpl_GetBlockchainStore(t *testing.T) {
	c := gomock.NewController(t)
	defer c.Finish()
	snapshot := mock.NewMockSnapshot(c)
	store := mock.NewMockBlockchainStore(c)
	snapshot.EXPECT().GetBlockchainStore().Return(store).AnyTimes()
	tests := []struct {
		name   string
		fields txSimContextImplTestFields
		want   protocol.BlockchainStore
	}{
		{
			name: "t1",
			fields: txSimContextImplTestFields{
				snapshot: snapshot,
			},
			want: store,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &txSimContextImpl{
				snapshot: tt.fields.snapshot,
				logger:   log,
			}
			if got := s.GetBlockchainStore(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetBlockchainStore() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_txSimContextImpl_GetChainNodesInfoProvider(t *testing.T) {
	c := gomock.NewController(t)
	defer c.Finish()
	vmManager := mock.NewMockVmManager(c)
	vmManager.EXPECT().GetChainNodesInfoProvider().Return(nil).AnyTimes()
	vmManager2 := mock.NewMockVmManager(c)
	infoProvider := mock.NewMockChainNodesInfoProvider(c)
	infoProvider.EXPECT().GetChainNodesInfo().Return([]*protocol.ChainNodeInfo{}, nil).AnyTimes()
	vmManager2.EXPECT().GetChainNodesInfoProvider().Return(infoProvider).AnyTimes()
	tests := []struct {
		name    string
		fields  txSimContextImplTestFields
		want    protocol.ChainNodesInfoProvider
		wantErr bool
	}{
		{
			name: "t1",
			fields: txSimContextImplTestFields{
				vmManager: vmManager,
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "t2",
			fields: txSimContextImplTestFields{
				vmManager: vmManager2,
			},
			want:    infoProvider,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &txSimContextImpl{
				vmManager: tt.fields.vmManager,
				logger:    log,
			}
			got, err := s.GetChainNodesInfoProvider()
			if (err != nil) != tt.wantErr {
				t.Errorf("GetChainNodesInfoProvider() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetChainNodesInfoProvider() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_txSimContextImpl_GetCreator(t *testing.T) {
	c := gomock.NewController(t)
	defer c.Finish()
	snapshot := mock.NewMockSnapshot(c)
	store := mock.NewMockBlockchainStore(c)
	store.EXPECT().GetContractByName(contractName).Return(&common.Contract{}, nil).AnyTimes()
	snapshot.EXPECT().GetBlockchainStore().Return(store).AnyTimes()
	cfg := &configPb.ChainConfig{Vm: &configPb.Vm{AddrType: configPb.AddrType_ZXL}}
	store.EXPECT().GetLastChainConfig().Return(cfg, nil).AnyTimes()

	snapshot2 := mock.NewMockSnapshot(c)
	store2 := mock.NewMockBlockchainStore(c)
	store2.EXPECT().GetContractByName(contractName).Return(&common.Contract{}, errors.New("error")).AnyTimes()
	snapshot2.EXPECT().GetBlockchainStore().Return(store2).AnyTimes()
	cfg2 := &configPb.ChainConfig{Vm: &configPb.Vm{AddrType: configPb.AddrType_ETHEREUM}}
	store2.EXPECT().GetLastChainConfig().Return(cfg2, nil).AnyTimes()

	type args struct {
		contractName string
	}
	tests := []struct {
		name   string
		fields txSimContextImplTestFields
		args   args
		want   *acPb.Member
	}{
		{
			name: "t1",
			fields: txSimContextImplTestFields{
				snapshot: snapshot,
			},
			args: args{
				contractName: contractName,
			},
			want: nil,
		},
		{
			name: "t2",
			fields: txSimContextImplTestFields{
				snapshot: snapshot2,
			},
			args: args{
				contractName: contractName,
			},
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &txSimContextImpl{
				snapshot: tt.fields.snapshot,
				logger:   log,
			}
			if got := s.GetCreator(tt.args.contractName); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetCreator() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_txSimContextImpl_GetHistoryIterForKey(t *testing.T) {
	c := gomock.NewController(t)
	defer c.Finish()
	iterator := mock.NewMockKeyHistoryIterator(c)
	store := mock.NewMockBlockchainStore(c)
	snapshot := mock.NewMockSnapshot(c)
	txWrite := []*common.TxRWSet{
		{
			TxId: txId,
			TxReads: []*common.TxRead{{
				Key:          []byte(key),
				Value:        []byte(value),
				ContractName: contractName,
			}},
			TxWrites: []*common.TxWrite{{
				Key:          []byte(key),
				Value:        []byte(value),
				ContractName: contractName,
			}},
		},
	}
	store.EXPECT().GetHistoryForKey(contractName, []byte(key)).Return(iterator, errors.New("error")).AnyTimes()
	snapshot.EXPECT().GetTxRWSetTable().Return(txWrite).AnyTimes()
	snapshot.EXPECT().GetBlockchainStore().Return(store).AnyTimes()
	snapshot.EXPECT().GetBlockHeight().Return(uint64(0)).AnyTimes()
	snapshot.EXPECT().GetBlockTimestamp().Return(int64(1641440597)).AnyTimes()

	type args struct {
		contractName string
		key          []byte
	}

	tests := []struct {
		name    string
		fields  txSimContextImplTestFields
		args    args
		want    protocol.KeyHistoryIterator
		wantErr bool
	}{
		{
			name: "t1",
			fields: txSimContextImplTestFields{
				currentDepth: 0,
				txRWSetWithDepth: []map[string]*rwSet{
					{
						contractName: &rwSet{
							txWriteKeyMap: map[string]*common.TxWrite{
								"": {
									Key:          []byte(key),
									Value:        []byte(value),
									ContractName: contractName,
								},
							},
						},
					},
				},
				snapshot: snapshot,
			},
			args: args{
				contractName: contractName,
				key:          []byte(key),
			},
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &txSimContextImpl{
				txRWSetWithDepth: tt.fields.txRWSetWithDepth,
				snapshot:         tt.fields.snapshot,
				logger:           log,
			}
			got, err := s.GetHistoryIterForKey(tt.args.contractName, tt.args.key)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetHistoryIterForKey() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetHistoryIterForKey() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_txSimContextImpl_GetTxExecSeq(t *testing.T) {
	tests := []struct {
		name   string
		fields txSimContextImplTestFields
		want   int
	}{
		{
			name: "t1",
			fields: txSimContextImplTestFields{
				txExecSeq: 0,
			},
			want: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &txSimContextImpl{
				txExecSeq: tt.fields.txExecSeq,
				logger:    log,
			}
			if got := s.GetTxExecSeq(); got != tt.want {
				t.Errorf("GetTxExecSeq() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_txSimContextImpl_GetTxRWSet(t *testing.T) {
	txRead := map[string]*common.TxRead{
		key: {
			Key:          []byte(key),
			Value:        []byte(value),
			ContractName: contractName,
		},
	}

	txWrite := map[string]*common.TxWrite{
		key: {
			Key:          []byte(key),
			Value:        []byte(value),
			ContractName: contractName,
		},
	}
	type args struct {
		runVmSuccess bool
	}
	tests := []struct {
		name   string
		fields txSimContextImplTestFields
		args   args
		want   *common.TxRWSet
	}{
		{
			name: "t1",
			fields: txSimContextImplTestFields{
				tx: &common.Transaction{
					Payload: &common.Payload{
						TxId: txId,
					},
				},

				currentDepth: 0,
				txRWSetWithDepth: []map[string]*rwSet{
					{
						contractName: &rwSet{
							txReadKeyMap:  txRead,
							txWriteKeyMap: txWrite,
						},
					},
				},
				// txReadKeyMap:  txRead,
				// txWriteKeyMap: txWrite,
			},
			args: args{
				runVmSuccess: true,
			},
			want: &common.TxRWSet{
				TxId: "0x86fa049857e0209aa7d9e616f7eb3b3b78ecfqb0",
				TxReads: []*common.TxRead{
					{
						Key:          []byte(key),
						Value:        []byte(value),
						ContractName: contractName,
					}},
				TxWrites: []*common.TxWrite{
					{
						Key:          []byte(key),
						Value:        []byte(value),
						ContractName: contractName,
					},
				},
			},
		},

		{
			name: "t2",
			fields: txSimContextImplTestFields{
				tx: &common.Transaction{
					Payload: &common.Payload{
						TxId: txId,
					},
				},

				currentDepth: 0,
				txRWSetWithDepth: []map[string]*rwSet{
					{
						contractName: &rwSet{
							txReadKeyMap:  txRead,
							txWriteKeyMap: txWrite,
						},
					},
				},
				// txReadKeyMap:  txRead,
				// txWriteKeyMap: txWrite,
			},
			args: args{
				runVmSuccess: false,
			},
			want: &common.TxRWSet{
				TxId: "0x86fa049857e0209aa7d9e616f7eb3b3b78ecfqb0",
				TxReads: []*common.TxRead{
					{
						Key:          []byte(key),
						Value:        []byte(value),
						ContractName: contractName,
					}},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &txSimContextImpl{
				tx:               tt.fields.tx,
				txRWSetWithDepth: tt.fields.txRWSetWithDepth,
				// txReadKeyMap:  tt.fields.txReadKeyMap,
				// txWriteKeyMap: tt.fields.txWriteKeyMap,
				logger: log,
			}
			if got := s.GetTxRWSet(tt.args.runVmSuccess); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetTxRWSet() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_txSimContextImpl_GetTxResult(t *testing.T) {
	result := &common.Result{
		Code: 0,
	}
	tests := []struct {
		name   string
		fields txSimContextImplTestFields
		want   *common.Result
	}{
		{
			name: "t1",
			fields: txSimContextImplTestFields{
				txResult: result,
			},
			want: result,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &txSimContextImpl{
				txResult: tt.fields.txResult,
				logger:   log,
			}
			if got := s.GetTxResult(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetTxResult() = %v, want %v", got, tt.want)
			}
			s.SetTxResult(tt.fields.txResult)
		})
	}
}

func Test_txSimContextImpl_Put(t *testing.T) {
	type args struct {
		contractName string
		key          []byte
		value        []byte
	}
	tests := []struct {
		name    string
		fields  txSimContextImplTestFields
		args    args
		wantErr bool
	}{
		{
			name: "t1",
			fields: txSimContextImplTestFields{
				currentDepth: 0,
				txRWSetWithDepth: []map[string]*rwSet{
					{
						contractName: &rwSet{
							txWriteKeyMap: map[string]*common.TxWrite{
								"": {
									Key:          []byte(key),
									Value:        []byte(value),
									ContractName: contractName,
								},
							},
						},
					},
				},

				// txWriteKeyMap: map[string]*common.TxWrite{
				// 	key: {
				// 		Key:          []byte(key),
				// 		Value:        []byte(value),
				// 		ContractName: contractName,
				// 	},
				// },
			},
			args: args{
				contractName: contractName,
				key:          []byte(key),
				value:        []byte(value),
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &txSimContextImpl{
				txRWSetWithDepth: tt.fields.txRWSetWithDepth,
				logger:           log,
			}
			if err := s.Put(tt.args.contractName, tt.args.key, tt.args.value); (err != nil) != tt.wantErr {
				t.Errorf("Put() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_txSimContextImpl_PutRecord(t *testing.T) {
	type args struct {
		contractName string
		value        []byte
		sqlType      protocol.SqlType
	}
	tests := []struct {
		name   string
		fields txSimContextImplTestFields
		args   args
	}{
		{
			name: "t1",
			fields: txSimContextImplTestFields{
				tx: &common.Transaction{
					Payload: &common.Payload{
						TxId: txId,
					},
				},
			},
			args: args{
				contractName: contractName,
				value:        []byte(value),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &txSimContextImpl{
				tx:     tt.fields.tx,
				logger: log,
			}
			s.PutRecord(tt.args.contractName, tt.args.value, tt.args.sqlType)
		})
	}
}

func Test_txSimContextImpl_Select(t *testing.T) {
	c := gomock.NewController(t)
	defer c.Finish()
	iterator := mock.NewMockStateIterator(c)
	store := mock.NewMockBlockchainStore(c)
	snapshot := mock.NewMockSnapshot(c)
	txWrite := []*common.TxRWSet{
		{
			TxId: txId,
			TxReads: []*common.TxRead{{
				Key:          []byte(key),
				Value:        []byte(value),
				ContractName: contractName,
			}},
			TxWrites: []*common.TxWrite{{
				Key:          []byte(key),
				Value:        []byte(value),
				ContractName: contractName,
			}},
		},
	}
	store.EXPECT().SelectObject(contractName, gomock.Any(), gomock.Any()).Return(iterator, errors.New("error")).AnyTimes()
	snapshot.EXPECT().GetTxRWSetTable().Return(txWrite).AnyTimes()
	snapshot.EXPECT().GetBlockchainStore().Return(store).AnyTimes()
	type args struct {
		contractName string
		startKey     []byte
		limit        []byte
	}
	tests := []struct {
		name    string
		fields  txSimContextImplTestFields
		args    args
		want    protocol.StateIterator
		wantErr bool
	}{
		{
			name: "t1",
			fields: txSimContextImplTestFields{
				currentDepth: 0,
				txRWSetWithDepth: []map[string]*rwSet{
					{
						contractName: &rwSet{
							txWriteKeyMap: map[string]*common.TxWrite{
								"": {
									Key:          []byte(key),
									Value:        []byte(value),
									ContractName: contractName,
								},
							},
						},
					},
				},

				// txWriteKeyMap: map[string]*common.TxWrite{
				// 	key: {
				// 		Key:          []byte(key),
				// 		Value:        []byte(value),
				// 		ContractName: contractName,
				// 	},
				// },
				snapshot: snapshot,
			},
			args: args{
				contractName: contractName,
				startKey:     []byte(key),
				limit:        []byte(limit),
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &txSimContextImpl{
				txRWSetWithDepth: tt.fields.txRWSetWithDepth,
				snapshot:         tt.fields.snapshot,
				logger:           log,
			}
			got, err := s.Select(tt.args.contractName, tt.args.startKey, tt.args.limit)
			if (err != nil) != tt.wantErr {
				t.Errorf("Select() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Select() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_txSimContextImpl_SetTxExecSeq(t *testing.T) {
	type args struct {
		txExecSeq int
	}
	tests := []struct {
		name   string
		fields txSimContextImplTestFields
		args   args
	}{
		{
			name: "t1",
			fields: txSimContextImplTestFields{
				txExecSeq: 0,
			},
			args: args{txExecSeq: 0},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &txSimContextImpl{
				txExecSeq: tt.fields.txExecSeq,
				logger:    log,
			}
			s.SetTxExecSeq(tt.args.txExecSeq)
		})
	}
}
