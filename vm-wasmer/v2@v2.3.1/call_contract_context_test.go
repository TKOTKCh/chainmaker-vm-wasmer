package vm

import (
	"reflect"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"

	"chainmaker.org/chainmaker/pb-go/v2/common"
)

func TestCallContractContext_AddLayer(t *testing.T) {
	type fields struct {
		ctxBitmap uint64
	}
	type args struct {
		runtimeType common.RuntimeType
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		{
			name:   "TestCallContractContext_AddLayer_EVM",
			fields: fields{ctxBitmap: 0},
			args:   args{runtimeType: common.RuntimeType_EVM},
		},
		{
			name:   "TestCallContractContext_AddLayer_DOCKER_GO",
			fields: fields{ctxBitmap: 0},
			args:   args{runtimeType: common.RuntimeType_GO},
		},
		{
			name:   "TestCallContractContext_AddLayer_EVM_DULPLICATE",
			fields: fields{ctxBitmap: 0},
			args:   args{runtimeType: common.RuntimeType_EVM},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := CallContractContext{
				ctxBitmap: tt.fields.ctxBitmap,
			}
			c.AddLayer(tt.args.runtimeType)
			assert.Equal(t, false, c.HasUsed(tt.args.runtimeType))
		})
	}
}

func TestCallContractContext_ClearLatestLayer(t *testing.T) {
	ctx := NewCallContractContext(0)
	ctx.AddLayer(common.RuntimeType_JAVA)
	ctx.AddLayer(common.RuntimeType_GO)
	ctx.AddLayer(common.RuntimeType_EVM)
	ctx.AddLayer(common.RuntimeType_JAVA)

	type input struct {
		preType []common.RuntimeType
	}
	type want struct {
		depth      uint64
		hasPreType []bool
		topType    common.RuntimeType
	}
	tests := []struct {
		input input
		want  want
	}{
		{
			input{
				preType: []common.RuntimeType{
					common.RuntimeType_EVM,
					common.RuntimeType_GO,
					common.RuntimeType_JAVA,
				},
			},
			want{
				depth: 3,
				hasPreType: []bool{
					false,
					true,
					true,
				},
				topType: common.RuntimeType_EVM,
			},
		},
		{
			input{
				preType: []common.RuntimeType{
					common.RuntimeType_EVM,
					common.RuntimeType_GO,
					common.RuntimeType_JAVA,
				},
			},
			want{
				depth: 2,
				hasPreType: []bool{
					false,
					false,
					true,
				},
				topType: common.RuntimeType_GO,
			},
		},
		{
			input{
				preType: []common.RuntimeType{
					common.RuntimeType_EVM,
					common.RuntimeType_GO,
					common.RuntimeType_JAVA,
				},
			},
			want{
				depth: 1,
				hasPreType: []bool{
					false,
					false,
					false,
				},
				topType: common.RuntimeType_JAVA,
			},
		},
	}

	for index, tt := range tests {
		t.Run(strconv.Itoa(index), func(t *testing.T) {
			ctx.ClearLatestLayer()
			depth := ctx.GetDepth()
			assert.Equal(t, depth, tt.want.depth)
			for index, preType := range tt.input.preType {
				assert.Equal(t, ctx.HasUsed(preType), tt.want.hasPreType[index])
			}
			topType, err := ctx.GetTypeByDepth(depth)
			assert.Nil(t, err)
			assert.Equal(t, topType, tt.want.topType)
		})
	}

}

func TestCallContractContext_GetAllTypes(t *testing.T) {

	ctx := NewCallContractContext(0)
	ctx.AddLayer(common.RuntimeType_GO)
	ctx.AddLayer(common.RuntimeType_EVM)
	ctx.AddLayer(common.RuntimeType_GO)
	ctx.AddLayer(common.RuntimeType_JAVA)

	type fields struct {
		context *CallContractContext
	}
	tests := []struct {
		name   string
		fields fields
		want   []common.RuntimeType
	}{
		{
			name:   "TestCallContractContext_GetAllTypes",
			fields: fields{context: ctx},
			want: []common.RuntimeType{
				common.RuntimeType_GO,
				common.RuntimeType_EVM,
				common.RuntimeType_GO,
				common.RuntimeType_JAVA,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := ctx
			if got := c.GetAllTypes(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetAllTypes() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCallContractContext_GetDepth(t *testing.T) {

	ctx := NewCallContractContext(0)
	ctx.AddLayer(common.RuntimeType_GO)
	ctx.AddLayer(common.RuntimeType_EVM)
	ctx.AddLayer(common.RuntimeType_GO)
	ctx.AddLayer(common.RuntimeType_JAVA)

	type fields struct {
		context *CallContractContext
	}
	tests := []struct {
		name   string
		fields fields
		want   uint64
	}{
		{
			name:   "TestCallContractContext_GetDepth",
			fields: fields{context: ctx},
			want:   4,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := ctx
			if got := c.GetDepth(); got != tt.want {
				t.Errorf("GetDepth() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCallContractContext_GetTypeByDepth(t *testing.T) {

	ctx := NewCallContractContext(0)
	ctx.AddLayer(common.RuntimeType_GO)
	ctx.AddLayer(common.RuntimeType_EVM)
	ctx.AddLayer(common.RuntimeType_GO)
	ctx.AddLayer(common.RuntimeType_JAVA)

	type fields struct {
		context *CallContractContext
	}
	type args struct {
		depth uint64
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    common.RuntimeType
		wantErr bool
	}{
		{
			name:    "TestCallContractContext_GetTypeByDepth_0",
			fields:  fields{context: ctx},
			args:    args{depth: 0},
			want:    0,
			wantErr: true,
		},
		{
			name:    "TestCallContractContext_GetTypeByDepth_1",
			fields:  fields{context: ctx},
			args:    args{depth: 1},
			want:    common.RuntimeType_GO,
			wantErr: false,
		},
		{
			name:    "TestCallContractContext_GetTypeByDepth_2",
			fields:  fields{context: ctx},
			args:    args{depth: 2},
			want:    common.RuntimeType_EVM,
			wantErr: false,
		},
		{
			name:    "TestCallContractContext_GetTypeByDepth_3",
			fields:  fields{context: ctx},
			args:    args{depth: 3},
			want:    common.RuntimeType_GO,
			wantErr: false,
		},
		{
			name:    "TestCallContractContext_GetTypeByDepth_4",
			fields:  fields{context: ctx},
			args:    args{depth: 4},
			want:    common.RuntimeType_JAVA,
			wantErr: false,
		},
		{
			name:    "TestCallContractContext_GetTypeByDepth_5",
			fields:  fields{context: ctx},
			args:    args{depth: 5},
			want:    0,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := ctx
			got, err := c.GetTypeByDepth(tt.args.depth)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetTypeByDepth() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("GetTypeByDepth() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCallContractContext_HasUsed(t *testing.T) {

	ctx := NewCallContractContext(0)
	ctx.AddLayer(common.RuntimeType_GO)
	ctx.AddLayer(common.RuntimeType_EVM)
	ctx.AddLayer(common.RuntimeType_JAVA)
	ctx.AddLayer(common.RuntimeType_WASMER)

	type fields struct {
		context *CallContractContext
	}
	type args struct {
		runtimeType common.RuntimeType
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		{
			name:   "TestCallContractContext_HasUsed_RuntimeType_INVALID",
			fields: fields{context: ctx},
			args:   args{runtimeType: common.RuntimeType_INVALID},
			want:   false,
		},
		{
			name:   "TestCallContractContext_HasUsed_RuntimeType_NATIVE",
			fields: fields{context: ctx},
			args:   args{runtimeType: common.RuntimeType_NATIVE},
			want:   false,
		},
		{
			name:   "TestCallContractContext_HasUsed_RuntimeType_WASMER",
			fields: fields{context: ctx},
			args:   args{runtimeType: common.RuntimeType_WASMER},
			want:   false,
		},
		{
			name:   "TestCallContractContext_HasUsed_RuntimeType_WXVM",
			fields: fields{context: ctx},
			args:   args{runtimeType: common.RuntimeType_WXVM},
			want:   false,
		},
		{
			name:   "TestCallContractContext_HasUsed_RuntimeType_GASM",
			fields: fields{context: ctx},
			args:   args{runtimeType: common.RuntimeType_GASM},
			want:   false,
		},
		{
			name:   "TestCallContractContext_HasUsed_RuntimeType_EVM",
			fields: fields{context: ctx},
			args:   args{runtimeType: common.RuntimeType_EVM},
			want:   true,
		},
		{
			name:   "TestCallContractContext_HasUsed_RuntimeType_GO",
			fields: fields{context: ctx},
			args:   args{runtimeType: common.RuntimeType_GO},
			want:   true,
		},
		{
			name:   "TestCallContractContext_HasUsed_RuntimeType_JAVA",
			fields: fields{context: ctx},
			args:   args{runtimeType: common.RuntimeType_JAVA},
			want:   true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.fields.context.HasUsed(tt.args.runtimeType); got != tt.want {
				t.Errorf("HasUsed() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewCallContractContext(t *testing.T) {
	type args struct {
		ctxBitmap uint64
	}
	tests := []struct {
		name string
		args args
		want *CallContractContext
	}{
		{
			name: "TestNewCallContractContext",
			args: args{ctxBitmap: 0},
			want: &CallContractContext{ctxBitmap: 0},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewCallContractContext(tt.args.ctxBitmap); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewCallContractContext() = %v, want %v", got, tt.want)
			}
		})
	}
}
