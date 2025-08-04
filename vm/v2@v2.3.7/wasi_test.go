/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package vm

import (
	"reflect"
	"sync"
	"testing"

	"chainmaker.org/chainmaker/common/v2/crypto/paillier"
	"chainmaker.org/chainmaker/common/v2/serialize"
	"chainmaker.org/chainmaker/pb-go/v2/accesscontrol"
	"chainmaker.org/chainmaker/pb-go/v2/common"
	"chainmaker.org/chainmaker/pb-go/v2/config"
	"chainmaker.org/chainmaker/pb-go/v2/store"
	"chainmaker.org/chainmaker/protocol/v2"
	"chainmaker.org/chainmaker/protocol/v2/mock"
	"chainmaker.org/chainmaker/protocol/v2/test"
	"github.com/golang/mock/gomock"
)

const (
	version = "2.x.x"
	message = "message1"
	dbName  = "dbName1"
	sql     = "select *from users"
)

func init() {
	protocol.ParametersValueMaxLength = protocol.DefaultParametersValueMaxSize * 1024 * 1024
}

// type WacsiImplTestFields struct {
type WacsiImplTestFields struct {
	verifySql protocol.SqlVerifier
	rowIndex  int32
	enableSql map[string]bool
	lock      *sync.Mutex
	logger    protocol.Logger
}

func TestNewWacsi(t *testing.T) {
	c := gomock.NewController(t)
	defer c.Finish()

	wasi := mock.NewMockWacsi(c)
	verifier := mock.NewMockSqlVerifier(c)
	type args struct {
		logger    protocol.Logger
		verifySql protocol.SqlVerifier
	}

	tests := []struct {
		name string
		args args
		want protocol.Wacsi
	}{
		{
			name: "t1",
			args: args{
				verifySql: verifier,
			},
			want: wasi,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			if got := NewWacsi(tt.args.logger, tt.args.verifySql); reflect.TypeOf(tt.want).Kind() != reflect.TypeOf(got).Kind() {
				t.Errorf("NewWacsi() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestWacsiImpl_BulletProofsOperation(t *testing.T) {
	c := gomock.NewController(t)
	defer c.Finish()

	codec := serialize.NewEasyCodec()
	codec.AddString("bulletproofsFuncName", protocol.BulletProofsOpTypePedersenAddNum)
	codec.AddBytes("param1", []byte(value))
	codec.AddBytes("param2", []byte("66"))
	codec.AddInt32("value_ptr", int32(1))

	codec1 := serialize.NewEasyCodec()
	codec1.AddString("bulletproofsFuncName", protocol.BulletProofsOpTypePedersenAddCommitment)
	codec1.AddBytes("param1", []byte(value))
	codec1.AddBytes("param2", []byte("66"))
	codec1.AddInt32("value_ptr", int32(1))

	codec2 := serialize.NewEasyCodec()
	codec2.AddString("bulletproofsFuncName", protocol.BulletProofsOpTypePedersenSubNum)
	codec2.AddBytes("param1", []byte(value))
	codec2.AddBytes("param2", []byte("66"))
	codec2.AddInt32("value_ptr", int32(1))

	codec3 := serialize.NewEasyCodec()
	codec3.AddString("bulletproofsFuncName", protocol.BulletProofsOpTypePedersenSubCommitment)
	codec3.AddBytes("param1", []byte(value))
	codec3.AddBytes("param2", []byte("66"))
	codec3.AddInt32("value_ptr", int32(1))

	codec4 := serialize.NewEasyCodec()
	codec4.AddString("bulletproofsFuncName", protocol.BulletProofsOpTypePedersenMulNum)
	codec4.AddBytes("param1", []byte(value))
	codec4.AddBytes("param2", []byte("66"))
	codec4.AddInt32("value_ptr", int32(1))

	codec5 := serialize.NewEasyCodec()
	codec5.AddString("bulletproofsFuncName", protocol.BulletProofsVerify)
	codec5.AddBytes("param1", []byte(value))
	codec5.AddBytes("param2", []byte("66"))
	codec5.AddInt32("value_ptr", int32(1))

	log := &test.GoLogger{}
	type args struct {
		requestBody []byte
		memory      []byte
		data        []byte
		isLen       bool
	}
	tests := []struct {
		name    string
		fields  WacsiImplTestFields
		args    args
		want    []byte
		wantErr bool
	}{
		{
			name: "t0",
			fields: WacsiImplTestFields{
				logger: log,
			},
			args: args{
				requestBody: codec.Marshal(),
				memory:      []byte(byteCode),
				data:        nil,
				isLen:       true,
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "t1",
			fields: WacsiImplTestFields{
				logger: log,
			},
			args: args{
				requestBody: codec1.Marshal(),
				memory:      []byte(byteCode),
				data:        nil,
				isLen:       true,
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "t2",
			fields: WacsiImplTestFields{
				logger: log,
			},
			args: args{
				requestBody: codec2.Marshal(),
				memory:      []byte(byteCode),
				data:        nil,
				isLen:       true,
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "t3",
			fields: WacsiImplTestFields{
				logger: log,
			},
			args: args{
				requestBody: codec3.Marshal(),
				memory:      []byte(byteCode),
				data:        nil,
				isLen:       true,
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "t4",
			fields: WacsiImplTestFields{
				logger: log,
			},
			args: args{
				requestBody: codec4.Marshal(),
				memory:      []byte(byteCode),
				data:        nil,
				isLen:       true,
			},
			want:    nil,
			wantErr: true,
		},

		{
			name: "t5",
			fields: WacsiImplTestFields{
				logger: log,
			},
			args: args{
				requestBody: codec5.Marshal(),
				memory:      []byte(byteCode),
				data:        nil,
				isLen:       true,
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wa := &WacsiImpl{
				verifySql: tt.fields.verifySql,
				rowIndex:  tt.fields.rowIndex,
				enableSql: tt.fields.enableSql,
				lock:      tt.fields.lock,
				logger:    tt.fields.logger,
			}
			got, err := wa.BulletProofsOperation(tt.args.requestBody, tt.args.memory, tt.args.data, tt.args.isLen)
			if (err != nil) != tt.wantErr {
				t.Errorf("BulletProofsOperation() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(reflect.TypeOf(got).Kind(), reflect.TypeOf(tt.want).Kind()) {
				t.Errorf("BulletProofsOperation() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestWacsiImpl_CallContract(t *testing.T) {
	c := gomock.NewController(t)
	defer c.Finish()
	context := mock.NewMockTxSimContext(c)
	context.EXPECT().Del(contractName, protocol.GetKeyStr(key, value)).Return(nil).AnyTimes()
	context.EXPECT().GetContractByName(contractName).Return(&common.Contract{Name: contractName}, nil).AnyTimes()
	context.EXPECT().GetBlockVersion().Return(uint32(v233)).AnyTimes()

	transaction := &common.Transaction{
		Payload: &common.Payload{
			ChainId:      chainId,
			TxType:       common.TxType_INVOKE_CONTRACT,
			TxId:         txId,
			ContractName: contractName,
		},
	}
	context.EXPECT().GetTx().Return(transaction).AnyTimes()
	context.EXPECT().CallContract(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(&common.ContractResult{}, protocol.ExecOrderTxTypeNormal, common.TxStatusCode_SUCCESS).AnyTimes()
	log := &test.GoLogger{}

	codec := serialize.NewEasyCodec()
	codec.AddString("key", key)
	codec.AddString("field", value)
	codec.AddString("value", value)
	codec.AddString("method", value)
	codec.AddString("contract_name", contractName)
	codec.AddBytes("param", codec.Marshal())
	type args struct {
		requestBody  []byte
		txSimContext protocol.TxSimContext
		memory       []byte
		data         []byte
		gasUsed      uint64
		isLen        bool
	}
	tests := []struct {
		name    string
		fields  WacsiImplTestFields
		args    args
		want    *common.ContractResult
		want1   uint64
		want2   protocol.ExecOrderTxType
		wantErr bool
	}{
		{
			name: "t1",
			fields: WacsiImplTestFields{
				logger: log,
			},
			args: args{
				requestBody:  codec.Marshal(),
				txSimContext: context,
				memory:       []byte(value),
				isLen:        true,
			},
			want:    nil,
			want1:   uint64(0),
			want2:   protocol.ExecOrderTxTypeNormal,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := &WacsiImpl{
				verifySql: tt.fields.verifySql,
				rowIndex:  tt.fields.rowIndex,
				enableSql: tt.fields.enableSql,
				lock:      tt.fields.lock,
				logger:    tt.fields.logger,
			}
			got, got1, got2, _ := w.CallContract(nil, tt.args.requestBody, tt.args.txSimContext, tt.args.memory, tt.args.data, tt.args.gasUsed, tt.args.isLen)
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

func TestWacsiImpl_DeleteState(t *testing.T) {
	c := gomock.NewController(t)
	defer c.Finish()
	context := mock.NewMockTxSimContext(c)
	context.EXPECT().Del(contractName, protocol.GetKeyStr(key, value)).Return(nil).AnyTimes()
	log := &test.GoLogger{}

	codec := serialize.NewEasyCodec()
	codec.AddString("key", key)
	codec.AddString("field", value)
	codec.AddString("value", value)

	type args struct {
		requestBody  []byte
		contractName string
		txSimContext protocol.TxSimContext
	}
	tests := []struct {
		name    string
		fields  WacsiImplTestFields
		args    args
		wantErr bool
	}{
		{
			name: "t1",
			fields: WacsiImplTestFields{
				logger: log,
			},
			args: args{
				requestBody:  codec.Marshal(),
				contractName: contractName,
				txSimContext: context,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := &WacsiImpl{
				logger: tt.fields.logger,
			}
			if err := w.DeleteState(tt.args.requestBody, tt.args.contractName, tt.args.txSimContext); (err != nil) != tt.wantErr {
				t.Errorf("DeleteState() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestWacsiImpl_EmitEvent(t *testing.T) {
	c := gomock.NewController(t)
	defer c.Finish()
	context := mock.NewMockTxSimContext(c)
	transaction := &common.Transaction{
		Payload: &common.Payload{
			ChainId:      chainId,
			TxType:       common.TxType_INVOKE_CONTRACT,
			TxId:         txId,
			ContractName: contractName,
		},
	}
	context.EXPECT().GetTx().Return(transaction).AnyTimes()
	log := &test.GoLogger{}

	codec := serialize.NewEasyCodec()
	codec.AddString("key", key)
	codec.AddString("field", value)
	codec.AddString("value", value)
	codec.AddString("topic", value)
	type args struct {
		requestBody  []byte
		txSimContext protocol.TxSimContext
		contractId   *common.Contract
		log          protocol.Logger
	}
	tests := []struct {
		name    string
		fields  WacsiImplTestFields
		args    args
		want    *common.ContractEvent
		wantErr bool
	}{
		{
			name: "t1",
			fields: WacsiImplTestFields{
				logger: log,
			},
			args: args{
				requestBody:  codec.Marshal(),
				txSimContext: context,
				contractId: &common.Contract{
					Name:    contractName,
					Version: version,
				},
				log: log,
			},
			want: &common.ContractEvent{
				Topic:           value,
				TxId:            txId,
				ContractName:    contractName,
				ContractVersion: version,
				EventData:       []string{value, value, value},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := &WacsiImpl{
				verifySql: tt.fields.verifySql,
				rowIndex:  tt.fields.rowIndex,
				enableSql: tt.fields.enableSql,
				lock:      tt.fields.lock,
				logger:    tt.fields.logger,
			}
			got, err := w.EmitEvent(tt.args.requestBody, tt.args.txSimContext, tt.args.contractId, tt.args.log)
			if (err != nil) != tt.wantErr {
				t.Errorf("EmitEvent() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("EmitEvent() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestWacsiImpl_ErrorResult(t *testing.T) {
	type args struct {
		contractResult *common.ContractResult
		data           []byte
	}
	tests := []struct {
		name   string
		fields WacsiImplTestFields
		args   args
		want   int32
	}{
		{
			name:   "t1",
			fields: WacsiImplTestFields{},
			args: args{
				contractResult: &common.ContractResult{
					Message: "",
				},
				data: nil,
			},
			want: int32(0),
		},
		{
			name:   "t2",
			fields: WacsiImplTestFields{},
			args: args{
				contractResult: &common.ContractResult{
					Message: message,
				},
				data: nil,
			},
			want: int32(0),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := &WacsiImpl{
				verifySql: tt.fields.verifySql,
				rowIndex:  tt.fields.rowIndex,
				enableSql: tt.fields.enableSql,
				lock:      tt.fields.lock,
				logger:    tt.fields.logger,
			}
			if got := w.ErrorResult(tt.args.contractResult, tt.args.data); got != tt.want {
				t.Errorf("ErrorResult() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestWacsiImpl_ExecuteDDL(t *testing.T) {
	c := gomock.NewController(t)
	defer c.Finish()
	context := mock.NewMockTxSimContext(c)
	blockchainStore := mock.NewMockBlockchainStore(c)
	sqlVerifier := mock.NewMockSqlVerifier(c)

	sqlVerifier.EXPECT().VerifyDDLSql(sql).Return(nil).AnyTimes()
	blockchainStore.EXPECT().GetLastChainConfig().Return(&config.ChainConfig{
		Contract: &config.ContractConfig{
			EnableSqlSupport: true,
		},
	}, nil).AnyTimes()

	blockchainStore.EXPECT().ExecDdlSql(contractName, sql, "1").Return(nil).AnyTimes()
	transaction := &common.Transaction{
		Payload: &common.Payload{
			ChainId:      chainId,
			TxType:       common.TxType_INVOKE_CONTRACT,
			TxId:         txId,
			ContractName: contractName,
		},
	}
	context.EXPECT().GetTx().Return(transaction).AnyTimes()
	context.EXPECT().GetBlockVersion().Return(uint32(v233)).AnyTimes()
	context.EXPECT().GetBlockchainStore().Return(blockchainStore).AnyTimes()
	context.EXPECT().PutRecord(contractName, []byte(sql), protocol.SqlTypeDdl)
	codec := serialize.NewEasyCodec()
	codec.AddString("sql", sql)
	codec.AddInt32("value_ptr", int32(1))
	log := &test.GoLogger{}
	type args struct {
		requestBody  []byte
		contractName string
		txSimContext protocol.TxSimContext
		memory       []byte
		method       string
	}
	tests := []struct {
		name    string
		fields  WacsiImplTestFields
		args    args
		wantErr bool
	}{
		{
			name: "t1",
			fields: WacsiImplTestFields{
				verifySql: sqlVerifier,
				enableSql: map[string]bool{},
				lock:      &sync.Mutex{},
				logger:    log,
			},
			args: args{
				requestBody:  codec.Marshal(),
				txSimContext: context,
				memory:       []byte(byteCode),
				contractName: contractName,
				method:       protocol.ContractInitMethod,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := &WacsiImpl{
				verifySql: tt.fields.verifySql,
				rowIndex:  tt.fields.rowIndex,
				enableSql: tt.fields.enableSql,
				lock:      tt.fields.lock,
				logger:    tt.fields.logger,
			}
			if err := w.ExecuteDDL(tt.args.requestBody, tt.args.contractName, tt.args.txSimContext, tt.args.memory, tt.args.method); (err != nil) != tt.wantErr {
				t.Errorf("ExecuteDDL() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestWacsiImpl_ExecuteQuery(t *testing.T) {
	c := gomock.NewController(t)
	defer c.Finish()
	context := mock.NewMockTxSimContext(c)
	blockchainStore := mock.NewMockBlockchainStore(c)
	sqlVerifier := mock.NewMockSqlVerifier(c)
	sqlRows := mock.NewMockSqlRows(c)

	sqlVerifier.EXPECT().VerifyDQLSql(sql).Return(nil).AnyTimes()
	blockchainStore.EXPECT().GetLastChainConfig().Return(&config.ChainConfig{
		Contract: &config.ContractConfig{
			EnableSqlSupport: true,
		},
	}, nil).AnyTimes()
	blockchainStore.EXPECT().QueryMulti(contractName, sql).Return(sqlRows, nil).AnyTimes()
	context.EXPECT().Del(contractName, protocol.GetKeyStr(key, value)).Return(nil).AnyTimes()
	context.EXPECT().GetBlockVersion().Return(uint32(v233)).AnyTimes()
	transaction := &common.Transaction{
		Payload: &common.Payload{
			ChainId:      chainId,
			TxType:       common.TxType_QUERY_CONTRACT,
			TxId:         txId,
			ContractName: contractName,
		},
	}
	context.EXPECT().GetTx().Return(transaction).AnyTimes()
	context.EXPECT().GetBlockchainStore().Return(blockchainStore).AnyTimes()
	context.EXPECT().SetIterHandle(int32(1), sqlRows).Return().AnyTimes()

	log := &test.GoLogger{}

	codec := serialize.NewEasyCodec()
	codec.AddString("sql", sql)
	codec.AddInt32("value_ptr", int32(1))

	type args struct {
		requestBody  []byte
		contractName string
		txSimContext protocol.TxSimContext
		memory       []byte
		chainId      string
	}
	tests := []struct {
		name    string
		fields  WacsiImplTestFields
		args    args
		wantErr bool
	}{
		{
			name: "t1",
			fields: WacsiImplTestFields{
				verifySql: sqlVerifier,
				rowIndex:  0,
				enableSql: make(map[string]bool),
				lock:      &sync.Mutex{},
				logger:    log,
			},
			args: args{
				requestBody:  codec.Marshal(),
				contractName: contractName,
				txSimContext: context,
				memory:       []byte(byteCode),
				chainId:      chainId,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := &WacsiImpl{
				verifySql: tt.fields.verifySql,
				rowIndex:  tt.fields.rowIndex,
				enableSql: tt.fields.enableSql,
				lock:      tt.fields.lock,
				logger:    tt.fields.logger,
			}
			if err := w.ExecuteQuery(tt.args.requestBody, tt.args.contractName, tt.args.txSimContext, tt.args.memory, tt.args.chainId); (err != nil) != tt.wantErr {
				t.Errorf("ExecuteQuery() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestWacsiImpl_ExecuteQueryOne(t *testing.T) {
	c := gomock.NewController(t)
	defer c.Finish()
	context := mock.NewMockTxSimContext(c)
	blockchainStore := mock.NewMockBlockchainStore(c)
	sqlVerifier := mock.NewMockSqlVerifier(c)
	sqlRows := mock.NewMockSqlRows(c)
	sqlRow := mock.NewMockSqlRow(c)
	stateSqlOperation := mock.NewMockStateSqlOperation(c)
	sqlRow.EXPECT().IsEmpty().Return(true).AnyTimes()
	stateSqlOperation.EXPECT().QuerySingle(contractName, sql).Return(sqlRow, nil).AnyTimes()
	sqlVerifier.EXPECT().VerifyDQLSql(sql).Return(nil).AnyTimes()

	blockchainStore.EXPECT().GetLastChainConfig().Return(&config.ChainConfig{
		Contract: &config.ContractConfig{
			EnableSqlSupport: true,
		},
	}, nil).AnyTimes()
	blockchainStore.EXPECT().QuerySingle(contractName, sql).Return(sqlRow, nil).AnyTimes()
	context.EXPECT().Del(contractName, protocol.GetKeyStr(key, value)).Return(nil).AnyTimes()
	context.EXPECT().GetBlockVersion().Return(uint32(v233)).AnyTimes()
	transaction := &common.Transaction{
		Payload: &common.Payload{
			ChainId:      chainId,
			TxType:       common.TxType_QUERY_CONTRACT,
			TxId:         txId,
			ContractName: contractName,
		},
	}
	context.EXPECT().GetTx().Return(transaction).AnyTimes()
	context.EXPECT().GetBlockchainStore().Return(blockchainStore).AnyTimes()
	context.EXPECT().SetIterHandle(int32(1), sqlRows).Return().AnyTimes()
	codec := serialize.NewEasyCodec()
	codec.AddString("sql", sql)
	codec.AddInt32("value_ptr", int32(1))
	log := &test.GoLogger{}

	type args struct {
		requestBody  []byte
		contractName string
		txSimContext protocol.TxSimContext
		memory       []byte
		data         []byte
		chainId      string
		isLen        bool
	}
	tests := []struct {
		name    string
		fields  WacsiImplTestFields
		args    args
		want    []byte
		wantErr bool
	}{
		{
			name: "t1",
			fields: WacsiImplTestFields{
				verifySql: sqlVerifier,
				enableSql: map[string]bool{},
				lock:      &sync.Mutex{},
				logger:    log,
			},
			args: args{
				requestBody:  codec.Marshal(),
				contractName: contractName,
				txSimContext: context,
				memory:       []byte(byteCode),
				isLen:        true,
			},
			want:    []byte("\u0000\u0000\u0000\u0000"),
			wantErr: false,
		},

		{
			name: "t1",
			fields: WacsiImplTestFields{
				verifySql: sqlVerifier,
				enableSql: map[string]bool{},
				lock:      &sync.Mutex{},
				logger:    log,
			},
			args: args{
				requestBody:  codec.Marshal(),
				contractName: contractName,
				txSimContext: context,
				memory:       []byte(byteCode),
				isLen:        false,
			},
			want:    []byte(""),
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := &WacsiImpl{
				verifySql: tt.fields.verifySql,
				rowIndex:  tt.fields.rowIndex,
				enableSql: tt.fields.enableSql,
				lock:      tt.fields.lock,
				logger:    tt.fields.logger,
			}
			got, err := w.ExecuteQueryOne(tt.args.requestBody, tt.args.contractName, tt.args.txSimContext, tt.args.memory, tt.args.data, tt.args.chainId, tt.args.isLen)
			if (err != nil) != tt.wantErr {
				t.Errorf("ExecuteQueryOne() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(reflect.TypeOf(got).Kind(), reflect.TypeOf(tt.want).Kind()) {
				t.Errorf("ExecuteQueryOne() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestWacsiImpl_ExecuteUpdate(t *testing.T) {
	c := gomock.NewController(t)
	defer c.Finish()
	context := mock.NewMockTxSimContext(c)
	blockchainStore := mock.NewMockBlockchainStore(c)
	sqlRows := mock.NewMockSqlRows(c)
	sqlVerifier := mock.NewMockSqlVerifier(c)
	sqlDBTransaction := mock.NewMockSqlDBTransaction(c)
	sqlDBTransaction.EXPECT().ChangeContextDb(dbName).Return(nil).AnyTimes()
	sqlDBTransaction.EXPECT().ExecSql(sql).Return(int64(1), nil).AnyTimes()

	sqlVerifier.EXPECT().VerifyDMLSql(sql).Return(nil).AnyTimes()
	sqlRows.EXPECT().Data().Return(map[string][]byte{}, nil).AnyTimes()
	blockchainStore.EXPECT().GetLastChainConfig().Return(&config.ChainConfig{
		Contract: &config.ContractConfig{
			EnableSqlSupport: true,
		},
	}, nil).AnyTimes()

	blockchainStore.EXPECT().GetContractDbName(contractName).Return(dbName).AnyTimes()

	context.EXPECT().Del(contractName, protocol.GetKeyStr(key, value)).Return(nil).AnyTimes()
	context.EXPECT().GetBlockVersion().Return(uint32(v233)).AnyTimes()
	transaction := &common.Transaction{
		Payload: &common.Payload{
			ChainId:      chainId,
			TxType:       common.TxType_INVOKE_CONTRACT,
			TxId:         txId,
			ContractName: contractName,
		},
	}
	context.EXPECT().GetTx().Return(transaction).AnyTimes()
	context.EXPECT().GetBlockchainStore().Return(blockchainStore).AnyTimes()
	context.EXPECT().GetIterHandle(int32(0)).Return(sqlRows, true).AnyTimes()
	context.EXPECT().GetBlockProposer().Return(&accesscontrol.Member{
		OrgId:      orgId,
		MemberType: 0,
		MemberInfo: []byte(certString),
	}).AnyTimes()
	context.EXPECT().GetBlockHeight().Return(uint64(1)).AnyTimes()
	context.EXPECT().PutRecord(contractName, []byte(sql), protocol.SqlTypeDml)
	txKey := common.GetTxKeyWith([]byte(certString), uint64(1))
	blockchainStore.EXPECT().GetDbTransaction(txKey).Return(sqlDBTransaction, nil).AnyTimes()
	codec := serialize.NewEasyCodec()
	codec.AddString("sql", sql)
	codec.AddInt32("value_ptr", int32(1))
	log := &test.GoLogger{}
	type args struct {
		requestBody  []byte
		contractName string
		method       string
		txSimContext protocol.TxSimContext
		memory       []byte
		chainId      string
	}
	tests := []struct {
		name    string
		fields  WacsiImplTestFields
		args    args
		wantErr bool
	}{
		{
			name: "t1",
			fields: WacsiImplTestFields{
				verifySql: sqlVerifier,
				enableSql: map[string]bool{},
				lock:      &sync.Mutex{},
				logger:    log,
			},
			args: args{
				requestBody:  codec.Marshal(),
				txSimContext: context,
				memory:       []byte(byteCode),
				contractName: contractName,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := &WacsiImpl{
				verifySql: tt.fields.verifySql,
				rowIndex:  tt.fields.rowIndex,
				enableSql: tt.fields.enableSql,
				lock:      tt.fields.lock,
				logger:    tt.fields.logger,
			}
			if err := w.ExecuteUpdate(tt.args.requestBody, tt.args.contractName, tt.args.method, tt.args.txSimContext, tt.args.memory, tt.args.chainId); (err != nil) != tt.wantErr {
				t.Errorf("ExecuteUpdate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestWacsiImpl_GetState(t *testing.T) {
	c := gomock.NewController(t)
	defer c.Finish()
	context := mock.NewMockTxSimContext(c)
	context.EXPECT().Get(contractName, protocol.GetKeyStr(key, value)).Return([]byte(value), nil).AnyTimes()
	context.EXPECT().GetBlockVersion().Return(uint32(v233)).AnyTimes()
	log := &test.GoLogger{}

	codec := serialize.NewEasyCodec()
	codec.AddString("key", key)
	codec.AddString("field", value)
	codec.AddString("value", value)
	type args struct {
		requestBody  []byte
		contractName string
		txSimContext protocol.TxSimContext
		memory       []byte
		data         []byte
		isLen        bool
	}
	tests := []struct {
		name    string
		fields  WacsiImplTestFields
		args    args
		want    []byte
		wantErr bool
	}{
		{
			name: "t1",
			fields: WacsiImplTestFields{
				verifySql: nil,
				rowIndex:  0,
				enableSql: nil,
				lock:      nil,
				logger:    log,
			},
			args: args{
				requestBody:  codec.Marshal(),
				contractName: contractName,
				txSimContext: context,
				memory:       []byte(byteCode),
				data:         nil,
				isLen:        true,
			},
			want:    []byte(value),
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := &WacsiImpl{
				verifySql: tt.fields.verifySql,
				rowIndex:  tt.fields.rowIndex,
				enableSql: tt.fields.enableSql,
				lock:      tt.fields.lock,
				logger:    tt.fields.logger,
			}
			got, err := w.GetState(tt.args.requestBody, tt.args.contractName, tt.args.txSimContext, tt.args.memory, tt.args.data, tt.args.isLen)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetState() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetState() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestWacsiImpl_KvIterator(t *testing.T) {
	c := gomock.NewController(t)
	defer c.Finish()
	context := mock.NewMockTxSimContext(c)
	iterator := mock.NewMockStateIterator(c)
	codec := serialize.NewEasyCodec()
	codec.AddString("start_key", key)
	codec.AddString("start_field", key)
	codec.AddString("limit_key", limit)
	codec.AddString("limit_field", limit)
	codec.AddString("value_ptr", value)
	context.EXPECT().Select(contractName, []byte(key+protocol.ContractStoreSeparator+key), []byte(limit+protocol.ContractStoreSeparator+limit)).Return(iterator, nil).AnyTimes()
	context.EXPECT().GetBlockVersion().Return(uint32(v233)).AnyTimes()
	context.EXPECT().SetIterHandle(int32(1), iterator).Return().AnyTimes()
	log := &test.GoLogger{}

	type args struct {
		requestBody  []byte
		contractName string
		txSimContext protocol.TxSimContext
		memory       []byte
	}
	tests := []struct {
		name    string
		fields  WacsiImplTestFields
		args    args
		wantErr bool
	}{
		{
			name: "t1",
			fields: WacsiImplTestFields{
				logger: log,
			},
			args: args{
				requestBody:  codec.Marshal(),
				contractName: contractName,
				txSimContext: context,
				memory:       []byte(value),
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := &WacsiImpl{
				logger: tt.fields.logger,
			}
			if err := w.KvIterator(tt.args.requestBody, tt.args.contractName, tt.args.txSimContext, tt.args.memory); (err != nil) != tt.wantErr {
				t.Errorf("KvIterator() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestWacsiImpl_KvIteratorClose(t *testing.T) {
	c := gomock.NewController(t)
	defer c.Finish()
	context := mock.NewMockTxSimContext(c)
	iterator := mock.NewMockStateIterator(c)
	iterator.EXPECT().Release().Return().AnyTimes()
	codec := serialize.NewEasyCodec()
	codec.AddInt32("rs_index", int32(1))
	codec.AddInt32("value_ptr", int32(2))
	context.EXPECT().GetIterHandle(int32(1)).Return(iterator, true).AnyTimes()
	log := &test.GoLogger{}

	type args struct {
		requestBody  []byte
		contractName string
		txSimContext protocol.TxSimContext
		memory       []byte
	}
	tests := []struct {
		name    string
		fields  WacsiImplTestFields
		args    args
		wantErr bool
	}{
		{
			name: "t1",
			fields: WacsiImplTestFields{
				logger: log,
			},
			args: args{
				requestBody:  codec.Marshal(),
				contractName: contractName,
				txSimContext: context,
				memory:       []byte(byteCode),
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := &WacsiImpl{
				verifySql: tt.fields.verifySql,
				rowIndex:  tt.fields.rowIndex,
				enableSql: tt.fields.enableSql,
				lock:      tt.fields.lock,
				logger:    tt.fields.logger,
			}
			if err := w.KvIteratorClose(tt.args.requestBody, tt.args.contractName, tt.args.txSimContext, tt.args.memory); (err != nil) != tt.wantErr {
				t.Errorf("KvIteratorClose() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestWacsiImpl_KvIteratorHasNext(t *testing.T) {
	c := gomock.NewController(t)
	defer c.Finish()
	context := mock.NewMockTxSimContext(c)
	iterator := mock.NewMockStateIterator(c)
	iterator.EXPECT().Next().Return(true).AnyTimes()
	codec := serialize.NewEasyCodec()
	codec.AddInt32("rs_index", int32(1))
	codec.AddInt32("value_ptr", int32(2))
	context.EXPECT().GetIterHandle(int32(1)).Return(iterator, true).AnyTimes()
	log := &test.GoLogger{}

	type args struct {
		requestBody  []byte
		txSimContext protocol.TxSimContext
		memory       []byte
	}
	tests := []struct {
		name    string
		fields  WacsiImplTestFields
		args    args
		wantErr bool
	}{
		{
			name: "t1",
			fields: WacsiImplTestFields{
				logger: log,
			},
			args: args{
				requestBody:  codec.Marshal(),
				txSimContext: context,
				memory:       []byte(byteCode),
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := &WacsiImpl{
				verifySql: tt.fields.verifySql,
				rowIndex:  tt.fields.rowIndex,
				enableSql: tt.fields.enableSql,
				lock:      tt.fields.lock,
				logger:    tt.fields.logger,
			}
			if err := w.KvIteratorHasNext(tt.args.requestBody, tt.args.txSimContext, tt.args.memory); (err != nil) != tt.wantErr {
				t.Errorf("KvIteratorHasNext() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestWacsiImpl_KvIteratorNext(t *testing.T) {
	c := gomock.NewController(t)
	defer c.Finish()
	context := mock.NewMockTxSimContext(c)
	iterator := mock.NewMockStateIterator(c)
	iterator.EXPECT().Value().Return(&store.KV{}, nil).AnyTimes()
	codec := serialize.NewEasyCodec()
	codec.AddInt32("rs_index", int32(1))
	codec.AddInt32("value_ptr", int32(2))
	context.EXPECT().GetIterHandle(int32(1)).Return(iterator, true).AnyTimes()
	log := &test.GoLogger{}

	type args struct {
		requestBody  []byte
		txSimContext protocol.TxSimContext
		memory       []byte
		data         []byte
		contractname string
		isLen        bool
	}
	tests := []struct {
		name    string
		fields  WacsiImplTestFields
		args    args
		want    []byte
		wantErr bool
	}{
		{
			name: "t1",
			fields: WacsiImplTestFields{
				logger: log,
			},
			args: args{
				requestBody:  codec.Marshal(),
				txSimContext: context,
				memory:       []byte(byteCode),
				data:         nil,
				contractname: contractName,
				isLen:        true,
			},
			want:    []byte("\u0003\u0000\u0000\u0000\u0001\u0000\u0000\u0000\u0003\u0000\u0000\u0000key\u0001\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0001\u0000\u0000\u0000\u0005\u0000\u0000\u0000field\u0001\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0001\u0000\u0000\u0000\u0005\u0000\u0000\u0000value\u0002\u0000\u0000\u0000\u0000\u0000\u0000\u0000"),
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wa := &WacsiImpl{
				verifySql: tt.fields.verifySql,
				rowIndex:  tt.fields.rowIndex,
				enableSql: tt.fields.enableSql,
				lock:      tt.fields.lock,
				logger:    tt.fields.logger,
			}
			got, err := wa.KvIteratorNext(tt.args.requestBody, tt.args.txSimContext, tt.args.memory, tt.args.data, tt.args.contractname, tt.args.isLen)
			if (err != nil) != tt.wantErr {
				t.Errorf("KvIteratorNext() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("KvIteratorNext() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestWacsiImpl_KvPreIterator(t *testing.T) {
	c := gomock.NewController(t)
	defer c.Finish()
	context := mock.NewMockTxSimContext(c)
	iterator := mock.NewMockStateIterator(c)
	codec := serialize.NewEasyCodec()
	codec.AddString("start_key", key)
	codec.AddString("start_field", key)
	codec.AddString("value_ptr", value)
	context.EXPECT().Select(contractName, []byte(key+protocol.ContractStoreSeparator+key), gomock.Any()).Return(iterator, nil).AnyTimes()
	context.EXPECT().GetBlockVersion().Return(uint32(v233)).AnyTimes()
	context.EXPECT().SetIterHandle(int32(1), iterator).Return().AnyTimes()
	log := &test.GoLogger{}

	type args struct {
		requestBody  []byte
		contractName string
		txSimContext protocol.TxSimContext
		memory       []byte
	}
	tests := []struct {
		name    string
		fields  WacsiImplTestFields
		args    args
		wantErr bool
	}{
		{
			name: "t1",
			fields: WacsiImplTestFields{
				logger: log,
			},
			args: args{
				requestBody:  codec.Marshal(),
				contractName: contractName,
				txSimContext: context,
				memory:       []byte(byteCode),
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := &WacsiImpl{
				verifySql: tt.fields.verifySql,
				rowIndex:  tt.fields.rowIndex,
				enableSql: tt.fields.enableSql,
				lock:      tt.fields.lock,
				logger:    tt.fields.logger,
			}
			if err := w.KvPreIterator(tt.args.requestBody, tt.args.contractName, tt.args.txSimContext, tt.args.memory); (err != nil) != tt.wantErr {
				t.Errorf("KvPreIterator() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestWacsiImpl_PaillierOperation(t *testing.T) {
	c := gomock.NewController(t)
	defer c.Finish()

	genKey, _ := paillier.GenKey()
	pubKey, _ := genKey.GetPubKey()
	pubKeyBytes, _ := pubKey.Marshal()

	codec0 := serialize.NewEasyCodec()
	codec0.AddString("opType", protocol.PaillierOpTypeAddCiphertext)
	codec0.AddBytes("operandOne", []byte(value))
	codec0.AddBytes("operandTwo", []byte(value))
	codec0.AddBytes("pubKey", pubKeyBytes)
	codec0.AddInt32("value_ptr", int32(1))

	codec1 := serialize.NewEasyCodec()
	codec1.AddString("opType", protocol.PaillierOpTypeAddPlaintext)
	codec1.AddBytes("operandOne", []byte(value))
	codec1.AddBytes("operandTwo", []byte(value))
	codec1.AddBytes("pubKey", pubKeyBytes)
	codec1.AddInt32("value_ptr", int32(1))

	codec2 := serialize.NewEasyCodec()
	codec2.AddString("opType", protocol.PaillierOpTypeSubCiphertext)
	codec2.AddBytes("operandOne", []byte(value))
	codec2.AddBytes("operandTwo", []byte(value))
	codec2.AddBytes("pubKey", pubKeyBytes)
	codec2.AddInt32("value_ptr", int32(1))

	codec3 := serialize.NewEasyCodec()
	codec3.AddString("opType", protocol.PaillierOpTypeSubPlaintext)
	codec3.AddBytes("operandOne", []byte(value))
	codec3.AddBytes("operandTwo", []byte(value))
	codec3.AddBytes("pubKey", pubKeyBytes)
	codec3.AddInt32("value_ptr", int32(1))

	codec4 := serialize.NewEasyCodec()
	codec4.AddString("opType", protocol.PaillierOpTypeNumMul)
	codec4.AddBytes("operandOne", []byte(value))
	codec4.AddBytes("operandTwo", []byte(value))
	codec4.AddBytes("pubKey", pubKeyBytes)
	codec4.AddInt32("value_ptr", int32(1))

	log := &test.GoLogger{}
	type args struct {
		requestBody []byte
		memory      []byte
		data        []byte
		isLen       bool
	}
	tests := []struct {
		name    string
		fields  WacsiImplTestFields
		args    args
		want    []byte
		wantErr bool
	}{
		{
			name: "t0",
			fields: WacsiImplTestFields{
				logger: log,
			},
			args: args{
				requestBody: codec0.Marshal(),
				memory:      []byte(byteCode),
				isLen:       true,
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "t1",
			fields: WacsiImplTestFields{
				logger: log,
			},
			args: args{
				requestBody: codec1.Marshal(),
				memory:      []byte(byteCode),
				isLen:       true,
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "t2",
			fields: WacsiImplTestFields{
				logger: log,
			},
			args: args{
				requestBody: codec2.Marshal(),
				memory:      []byte(byteCode),
				isLen:       true,
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "t3",
			fields: WacsiImplTestFields{
				logger: log,
			},
			args: args{
				requestBody: codec3.Marshal(),
				memory:      []byte(byteCode),
				isLen:       true,
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "t4",
			fields: WacsiImplTestFields{
				logger: log,
			},
			args: args{
				requestBody: codec4.Marshal(),
				memory:      []byte(byteCode),
				isLen:       true,
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wa := &WacsiImpl{
				verifySql: tt.fields.verifySql,
				rowIndex:  tt.fields.rowIndex,
				enableSql: tt.fields.enableSql,
				lock:      tt.fields.lock,
				logger:    tt.fields.logger,
			}
			got, err := wa.PaillierOperation(tt.args.requestBody, tt.args.memory, tt.args.data, tt.args.isLen)
			if (err != nil) != tt.wantErr {
				t.Errorf("PaillierOperation() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("PaillierOperation() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestWacsiImpl_PutState(t *testing.T) {
	c := gomock.NewController(t)
	defer c.Finish()
	context := mock.NewMockTxSimContext(c)
	context.EXPECT().Put(contractName, protocol.GetKeyStr(key, value), gomock.Any()).Return(nil).AnyTimes()
	context.EXPECT().GetBlockVersion().Return(uint32(v233)).AnyTimes()
	log := &test.GoLogger{}

	codec := serialize.NewEasyCodec()
	codec.AddString("key", key)
	codec.AddString("field", value)
	codec.AddString("value", value)
	type args struct {
		requestBody  []byte
		contractName string
		txSimContext protocol.TxSimContext
	}
	tests := []struct {
		name    string
		fields  WacsiImplTestFields
		args    args
		wantErr bool
	}{
		{
			name: "t1",
			fields: WacsiImplTestFields{
				logger: log,
			},
			args: args{
				requestBody:  serialize.NewEasyCodecWithMap(nil).Marshal(),
				contractName: contractName,
				txSimContext: context,
			},
			wantErr: true,
		},
		{
			name: "t2",
			fields: WacsiImplTestFields{
				logger: log,
			},
			args: args{
				requestBody:  codec.Marshal(),
				contractName: contractName,
				txSimContext: context,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := &WacsiImpl{
				logger: tt.fields.logger,
			}
			if err := w.PutState(tt.args.requestBody, tt.args.contractName, tt.args.txSimContext); (err != nil) != tt.wantErr {
				t.Errorf("PutState() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestWacsiImpl_RSClose(t *testing.T) {
	c := gomock.NewController(t)
	defer c.Finish()
	context := mock.NewMockTxSimContext(c)
	blockchainStore := mock.NewMockBlockchainStore(c)
	sqlRows := mock.NewMockSqlRows(c)

	sqlRows.EXPECT().Close().Return(nil).AnyTimes()
	blockchainStore.EXPECT().GetLastChainConfig().Return(&config.ChainConfig{
		Contract: &config.ContractConfig{
			EnableSqlSupport: true,
		},
	}, nil).AnyTimes()
	context.EXPECT().Del(contractName, protocol.GetKeyStr(key, value)).Return(nil).AnyTimes()
	transaction := &common.Transaction{
		Payload: &common.Payload{
			ChainId:      chainId,
			TxType:       common.TxType_QUERY_CONTRACT,
			TxId:         txId,
			ContractName: contractName,
		},
	}
	context.EXPECT().GetTx().Return(transaction).AnyTimes()
	context.EXPECT().GetBlockchainStore().Return(blockchainStore).AnyTimes()
	context.EXPECT().GetIterHandle(int32(0)).Return(sqlRows, true).AnyTimes()
	context.EXPECT().GetBlockVersion().Return(uint32(v233)).AnyTimes()

	codec := serialize.NewEasyCodec()
	codec.AddInt32("rs_index", int32(0))
	codec.AddInt32("value_ptr", int32(1))
	log := &test.GoLogger{}
	type args struct {
		requestBody  []byte
		txSimContext protocol.TxSimContext
		memory       []byte
	}
	tests := []struct {
		name    string
		fields  WacsiImplTestFields
		args    args
		wantErr bool
	}{
		{
			name: "t1",
			fields: WacsiImplTestFields{
				enableSql: map[string]bool{},
				lock:      &sync.Mutex{},
				logger:    log,
			},
			args: args{
				requestBody:  codec.Marshal(),
				txSimContext: context,
				memory:       []byte(byteCode),
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := &WacsiImpl{
				verifySql: tt.fields.verifySql,
				rowIndex:  tt.fields.rowIndex,
				enableSql: tt.fields.enableSql,
				lock:      tt.fields.lock,
				logger:    tt.fields.logger,
			}
			if err := w.RSClose(tt.args.requestBody, tt.args.txSimContext, tt.args.memory); (err != nil) != tt.wantErr {
				t.Errorf("RSClose() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestWacsiImpl_RSHasNext(t *testing.T) {
	c := gomock.NewController(t)
	defer c.Finish()
	context := mock.NewMockTxSimContext(c)
	blockchainStore := mock.NewMockBlockchainStore(c)
	sqlRows := mock.NewMockSqlRows(c)
	sqlRows.EXPECT().Next().Return(true).AnyTimes()

	blockchainStore.EXPECT().GetLastChainConfig().Return(&config.ChainConfig{
		Contract: &config.ContractConfig{
			EnableSqlSupport: true,
		},
	}, nil).AnyTimes()
	context.EXPECT().Del(contractName, protocol.GetKeyStr(key, value)).Return(nil).AnyTimes()
	transaction := &common.Transaction{
		Payload: &common.Payload{
			ChainId:      chainId,
			TxType:       common.TxType_QUERY_CONTRACT,
			TxId:         txId,
			ContractName: contractName,
		},
	}
	context.EXPECT().GetTx().Return(transaction).AnyTimes()
	context.EXPECT().GetBlockchainStore().Return(blockchainStore).AnyTimes()
	context.EXPECT().GetIterHandle(int32(0)).Return(sqlRows, true).AnyTimes()
	context.EXPECT().GetBlockVersion().Return(uint32(v233)).AnyTimes()

	codec := serialize.NewEasyCodec()
	codec.AddInt32("rs_index", int32(0))
	codec.AddInt32("value_ptr", int32(1))
	log := &test.GoLogger{}
	type args struct {
		requestBody  []byte
		txSimContext protocol.TxSimContext
		memory       []byte
	}
	tests := []struct {
		name    string
		fields  WacsiImplTestFields
		args    args
		wantErr bool
	}{
		{
			name: "t1",
			fields: WacsiImplTestFields{
				enableSql: map[string]bool{},
				lock:      &sync.Mutex{},
				logger:    log,
			},
			args: args{
				requestBody:  codec.Marshal(),
				txSimContext: context,
				memory:       []byte(byteCode),
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := &WacsiImpl{
				verifySql: tt.fields.verifySql,
				rowIndex:  tt.fields.rowIndex,
				enableSql: tt.fields.enableSql,
				lock:      tt.fields.lock,
				logger:    tt.fields.logger,
			}
			if err := w.RSHasNext(tt.args.requestBody, tt.args.txSimContext, tt.args.memory); (err != nil) != tt.wantErr {
				t.Errorf("RSHasNext() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestWacsiImpl_RSNext(t *testing.T) {
	c := gomock.NewController(t)
	defer c.Finish()
	context := mock.NewMockTxSimContext(c)
	blockchainStore := mock.NewMockBlockchainStore(c)
	sqlRows := mock.NewMockSqlRows(c)

	sqlRows.EXPECT().Data().Return(map[string][]byte{}, nil).AnyTimes()
	blockchainStore.EXPECT().GetLastChainConfig().Return(&config.ChainConfig{
		Contract: &config.ContractConfig{
			EnableSqlSupport: true,
		},
	}, nil).AnyTimes()
	context.EXPECT().Del(contractName, protocol.GetKeyStr(key, value)).Return(nil).AnyTimes()
	transaction := &common.Transaction{
		Payload: &common.Payload{
			ChainId:      chainId,
			TxType:       common.TxType_QUERY_CONTRACT,
			TxId:         txId,
			ContractName: contractName,
		},
	}
	context.EXPECT().GetTx().Return(transaction).AnyTimes()
	context.EXPECT().GetBlockchainStore().Return(blockchainStore).AnyTimes()
	context.EXPECT().GetIterHandle(int32(0)).Return(sqlRows, true).AnyTimes()
	context.EXPECT().GetBlockVersion().Return(uint32(v233)).AnyTimes()

	codec := serialize.NewEasyCodec()
	codec.AddInt32("rs_index", int32(0))
	codec.AddInt32("value_ptr", int32(1))
	log := &test.GoLogger{}
	type args struct {
		requestBody  []byte
		txSimContext protocol.TxSimContext
		memory       []byte
		data         []byte
		isLen        bool
	}
	tests := []struct {
		name    string
		fields  WacsiImplTestFields
		args    args
		want    []byte
		wantErr bool
	}{
		{
			name: "t1",
			fields: WacsiImplTestFields{
				enableSql: map[string]bool{},
				lock:      &sync.Mutex{},
				logger:    log,
			},
			args: args{
				requestBody:  codec.Marshal(),
				txSimContext: context,
				memory:       []byte(byteCode),
				isLen:        true,
			},
			want:    nil,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := &WacsiImpl{
				verifySql: tt.fields.verifySql,
				rowIndex:  tt.fields.rowIndex,
				enableSql: tt.fields.enableSql,
				lock:      tt.fields.lock,
				logger:    tt.fields.logger,
			}
			got, err := w.RSNext(tt.args.requestBody, tt.args.txSimContext, tt.args.memory, tt.args.data, tt.args.isLen)
			if (err != nil) != tt.wantErr {
				t.Errorf("RSNext() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(reflect.TypeOf(got).Kind(), reflect.TypeOf(tt.want).Kind()) {
				t.Errorf("RSNext() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestWacsiImpl_SuccessResult(t *testing.T) {
	type args struct {
		contractResult *common.ContractResult
		data           []byte
	}
	tests := []struct {
		name   string
		fields WacsiImplTestFields
		args   args
		want   int32
	}{
		{
			name:   "t1",
			fields: WacsiImplTestFields{},
			args: args{
				contractResult: &common.ContractResult{
					Code: uint32(0),
				},
				data: nil,
			},
			want: 0,
		},
		{
			name:   "t2",
			fields: WacsiImplTestFields{},
			args: args{
				contractResult: &common.ContractResult{
					Code: uint32(1),
				},
				data: nil,
			},
			want: 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := &WacsiImpl{
				verifySql: tt.fields.verifySql,
				rowIndex:  tt.fields.rowIndex,
				enableSql: tt.fields.enableSql,
				lock:      tt.fields.lock,
				logger:    tt.fields.logger,
			}
			if got := w.SuccessResult(tt.args.contractResult, tt.args.data); got != tt.want {
				t.Errorf("SuccessResult() = %v, want %v", got, tt.want)
			}
		})
	}
}
