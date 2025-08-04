package vm

import (
	"reflect"
	"testing"

	"chainmaker.org/chainmaker/protocol/v2"
	"chainmaker.org/chainmaker/protocol/v2/mock"
	"github.com/golang/mock/gomock"
)

func TestNewGasWacsi(t *testing.T) {
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

			if got := NewWacsiWithGas(tt.args.logger, tt.args.verifySql); reflect.TypeOf(tt.want).Kind() != reflect.TypeOf(got).Kind() {
				t.Errorf("NewWacsi() = %v, want %v", got, tt.want)
			}
		})
	}
}
