package filter

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"reflect"
	"strconv"
	"testing"

	"chainmaker.org/chainmaker/protocol/v2"
	"chainmaker.org/chainmaker/protocol/v2/test"
	redisbloom "github.com/RedisBloom/redisbloom-go"
)

var (
	redisServerCluster                     = []string{"127.0.0.1:6500", "127.0.0.1:6501", "127.0.0.1:6502"}
	redisPassword                          *string
	logger                                 = &test.GoLogger{}
	key1                                   = "key1"
	key2                                   = "key2"
	key3                                   = "key3"
	key4                                   = "key4"
	new_big_filter_bloomfilter_redis_error = "new_big_filter_bloomfilter_redis_error"
	tChainID1                              = "chainID1"
)

func initRedisHosts() {
	cmd := exec.Command("/bin/bash", "start_redis.sh")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		fmt.Println("start redis error :", err)
	}
	//outStr, errStr := string(stdout.Bytes()), string(stderr.Bytes())
	outStr, errStr := stdout.String(), stderr.String()
	fmt.Printf("out:%s\n", outStr)
	if errStr != "" {
		fmt.Printf("err:%s\n", errStr)
	}
}

func destroyRedisHosts() {
	cmd := exec.Command("/bin/bash", "stop_redis.sh")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		fmt.Println("stop redis error: ", err)
	}
	//outStr, errStr := string(stdout.Bytes()), string(stderr.Bytes())
	outStr, errStr := stdout.String(), stderr.String()
	fmt.Printf("out:%s\n", outStr)
	if errStr != "" {
		fmt.Printf("err:%s\n", errStr)
	}

}

func createRedisClient() []*redisbloom.Client {
	filters := []*redisbloom.Client{}

	for i := 0; i < len(redisServerCluster); i++ {
		//把过滤器，映射到对应的redisServerCluster 中，一共n个redis
		redisNum := i % (len(redisServerCluster))
		bloomFilter := redisbloom.NewClient(redisServerCluster[redisNum], "nohelp", redisPassword)
		//创建redisBloom 过滤器
		idxStr := strconv.Itoa(i)
		bloomName := redisBloomName + idxStr

		//filter已存在则创建失败，不会重新创建
		err := bloomFilter.Reserve(bloomName, 0.1, 100*100*100)
		if err != nil {
			logger.Infof("create filter [%s] in redis server [%s] err:", bloomName,
				redisServerCluster[redisNum], err)
		}
		filters = append(filters, bloomFilter)
	}
	return filters

}

func Test_bigFilter_Add(t *testing.T) {
	initRedisHosts()
	type fields struct {
		filters []*redisbloom.Client
		logger  protocol.Logger
	}
	type args struct {
		key []byte
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
		{
			name: "add1",
			fields: fields{
				filters: createRedisClient(),
				logger:  logger,
			},
			args: args{
				key: []byte("input1"),
			},
			wantErr: false,
		},
		{
			name: "add2", //不启动redis，模拟错误情况
			fields: fields{
				filters: createRedisClient(),
				logger:  logger,
			},
			args: args{
				key: []byte("input2"),
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		if tt.name == "add2" {
			destroyRedisHosts()
		}
		t.Run(tt.name, func(t *testing.T) {
			b := bigFilter{
				filters: tt.fields.filters,
				logger:  tt.fields.logger,
			}
			if err := b.Add(tt.args.key); (err != nil) != tt.wantErr {
				t.Errorf("Add() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_bigFilter_ExistMult(t *testing.T) {
	//启动redis
	initRedisHosts()

	type fields struct {
		filters []*redisbloom.Client
		logger  protocol.Logger
	}
	type args struct {
		arr [][]byte
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   []bool
	}{
		// TODO: Add test cases.
		{
			name: "notExistMult1", //测试不存在的情况
			fields: fields{
				filters: createRedisClient(),
				logger:  logger,
			},
			args: args{
				arr: [][]byte{[]byte(key1), []byte(key2), []byte(key3), []byte(key4)},
			},
			want: []bool{
				false, false, false, false,
			},
		},
		{
			name: "existMult", //测试存在的情况
			fields: fields{
				filters: createRedisClient(),
				logger:  logger,
			},
			args: args{
				arr: [][]byte{[]byte(key1), []byte(key2), []byte(key3), []byte(key4)},
			},
			want: []bool{
				true, true, true, true,
			},
		},
		{
			name: "redis_error_ExistMult", //测试redis出错的情况
			fields: fields{
				filters: createRedisClient(),
				logger:  logger,
			},
			args: args{
				arr: [][]byte{[]byte(key1), []byte(key2), []byte(key3), []byte(key4)},
			},
			want: []bool{
				false, false, false, false,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := bigFilter{
				filters: tt.fields.filters,
				logger:  tt.fields.logger,
			}
			//err := b.AddMult(tt.args.arr)
			//if err != nil {
			//	t.Errorf("error,when addMult:",err)
			//}
			if tt.name == "existMult" {
				err := b.AddMult(tt.args.arr)
				if err != nil {
					t.Errorf("error,when addMult:[%s]", err)
				}
			}
			if tt.name == "redis_error_ExistMult" {
				destroyRedisHosts()
			}
			if got := b.ExistMult(tt.args.arr); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ExistMult() = %v, want %v", got, tt.want)
			}
		})
	}
	destroyRedisHosts()
}

func Test_bigFilter_Exist(t *testing.T) {
	//启动 redis
	initRedisHosts()

	type fields struct {
		filters []*redisbloom.Client
		logger  protocol.Logger
	}
	type args struct {
		key []byte
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    bool
		wantErr bool
	}{
		// TODO: Add test cases.
		{
			name: "exist",
			fields: fields{
				filters: createRedisClient(),
				logger:  logger,
			},
			args: args{
				key: []byte(key1),
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "exist_not",
			fields: fields{
				filters: createRedisClient(),
				logger:  logger,
			},
			args: args{
				key: []byte(key2),
			},
			want:    false,
			wantErr: false,
		},
		{
			name: "exist_redis_err",
			fields: fields{
				filters: createRedisClient(),
				logger:  logger,
			},
			args: args{
				key: []byte(key2),
			},
			want:    false,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := bigFilter{
				filters: tt.fields.filters,
				logger:  tt.fields.logger,
			}
			if tt.name == "exist" {
				err := b.Add(tt.args.key)
				if err != nil {
					t.Errorf("add key error:[%s]", err)
				}
			}
			if tt.name == "exist_redis_err" {
				destroyRedisHosts()
			}
			got, err := b.Exist(tt.args.key)
			if (err != nil) != tt.wantErr {
				t.Errorf("Exist() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Exist() got = %v, want %v", got, tt.want)
			}
		})
	}
	destroyRedisHosts()
}

func Test_bigFilter_AddMult(t *testing.T) {
	initRedisHosts()

	type fields struct {
		filters []*redisbloom.Client
		logger  protocol.Logger
	}
	type args struct {
		arr [][]byte
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
		{
			name: "addMult1",
			fields: fields{
				filters: createRedisClient(),
				logger:  logger,
			},
			args: args{
				arr: [][]byte{[]byte(key1), []byte(key2), []byte(key3), []byte(key4)},
			},
			wantErr: false,
		},
		{
			name: "addMult2", //测试redis 故障的情况
			fields: fields{
				filters: createRedisClient(),
				logger:  logger,
			},
			args: args{
				arr: [][]byte{[]byte(key1), []byte(key2), []byte(key3), []byte(key4)},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := bigFilter{
				filters: tt.fields.filters,
				logger:  tt.fields.logger,
			}
			if tt.name == "addMult2" {
				destroyRedisHosts()
			}
			if err := b.AddMult(tt.args.arr); (err != nil) != tt.wantErr {
				t.Errorf("AddMult() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_bigFilter_SaveLastPoint(t *testing.T) {
	initRedisHosts()
	type fields struct {
		filters []*redisbloom.Client
		logger  protocol.Logger
	}
	type args struct {
		pointK string
		pointV string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
		{
			name: "save_point",
			fields: fields{
				filters: createRedisClient(),
				logger:  logger,
			},
			args: args{
				pointK: lastBigFilterNumKeyStr,
				pointV: lastBigFilterNumKeyStr,
			},
			wantErr: false,
		},
		{
			name: "save_point_err",
			fields: fields{
				filters: createRedisClient(),
				logger:  logger,
			},
			args: args{
				pointK: lastBigFilterNumKeyStr,
				pointV: lastBigFilterNumKeyStr,
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := bigFilter{
				filters: tt.fields.filters,
				logger:  tt.fields.logger,
			}
			if tt.name == "save_point_err" {
				destroyRedisHosts()
			}
			if err := b.SaveLastPoint(tt.args.pointK, tt.args.pointV); (err != nil) != tt.wantErr {
				t.Errorf("SaveLastPoint() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_bigFilter_GetLastPoint(t *testing.T) {
	initRedisHosts()
	type fields struct {
		filters []*redisbloom.Client
		logger  protocol.Logger
	}
	type args struct {
		pointK string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    []string
		wantErr bool
	}{
		// TODO: Add test cases.
		{
			name: "get_last_point",
			fields: fields{
				filters: createRedisClient(),
				logger:  logger,
			},
			args: args{
				pointK: lastBigFilterNumKeyStr,
			},
			want:    []string{lastBigFilterNumKeyStr, lastBigFilterNumKeyStr, lastBigFilterNumKeyStr},
			wantErr: false,
		},
		{
			name: "err_get_last_point",
			fields: fields{
				filters: createRedisClient(),
				logger:  logger,
			},
			args: args{
				pointK: lastBigFilterNumKeyStr,
			},
			want:    nil, //出错时，会返回nil
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := bigFilter{
				filters: tt.fields.filters,
				logger:  tt.fields.logger,
			}
			//先写一个
			b.SaveLastPoint(tt.args.pointK, tt.args.pointK)
			if tt.name == "err_get_last_point" {
				destroyRedisHosts()
			}
			got, err := b.GetLastPoint(tt.args.pointK)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetLastPoint() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetLastPoint() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHash(t *testing.T) {
	type args struct {
		bys []byte
	}
	tests := []struct {
		name string
		args args
		want uint64
	}{
		// TODO: Add test cases.
		{
			name: "hash1",
			args: args{
				bys: []byte("in1"),
			},
			want: 5481223373228092137,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Hash(tt.args.bys); got != tt.want {
				fmt.Println(got)
				t.Errorf("Hash() = %v, want %v", got, tt.want)
			}
		})
	}
}
func TestNewBigFilter(t *testing.T) {
	initRedisHosts()
	type args struct {
		n                  int
		m                  uint
		fpRate             float64
		redisServerCluster []string
		pass               *string
		logger             protocol.Logger
	}
	tests := []struct {
		name    string
		args    args
		want    Filter
		wantErr bool
	}{
		// TODO: Add test cases.
		{
			name: "new_big_filter",
			args: args{
				n:                  3,
				m:                  100 * 100 * 100,
				fpRate:             0.00001,
				redisServerCluster: redisServerCluster,
				pass:               nil,
				logger:             logger,
			},
			want: bigFilter{
				filters: createRedisClient(),
				logger:  logger,
			},
			wantErr: false,
		},
		{
			name: "new_big_filter_bloomfilter_exist",
			args: args{
				n:                  3,
				m:                  100 * 100 * 100,
				fpRate:             0.00001,
				redisServerCluster: redisServerCluster,
				pass:               nil,
				logger:             logger,
			},
			want: bigFilter{
				filters: createRedisClient(),
				logger:  logger,
			},
			wantErr: false,
		},
		{
			name: new_big_filter_bloomfilter_redis_error,
			args: args{
				n:                  3,
				m:                  100 * 100 * 100,
				fpRate:             0.00001,
				redisServerCluster: redisServerCluster,
				pass:               nil,
				logger:             logger,
			},
			want: bigFilter{
				filters: createRedisClient(),
				logger:  logger,
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.name == new_big_filter_bloomfilter_redis_error {
				destroyRedisHosts()
			}
			got, err := NewBigFilter(tt.args.n, tt.args.m, tt.args.fpRate, tt.args.redisServerCluster, tt.args.pass,
				tChainID1, tt.args.logger)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewBigFilter() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.name == new_big_filter_bloomfilter_redis_error {
				return
			}
			//如果正常，说明返回结果正确，如果不正常，说明创建过滤器失败
			k := key1 + key1
			err = got.Add([]byte(k))
			if err != nil {
				t.Errorf("NewBigFilter() error, err=[%s]", err)
			}
			//if !reflect.DeepEqual(got, tt.want) {
			//	t.Errorf("NewBigFilter() got = %v, want %v", got, tt.want)
			//}
		})
	}
	destroyRedisHosts()
}

//func TestNewBigFilter(t *testing.T) {
//	initRedisHosts()
//	type args struct {
//		n                  int
//		m                  uint
//		fpRate             float64
//		redisServerCluster []string
//		pass               *string
//		logger             protocol.Logger
//	}
//	tests := []struct {
//		name string
//		args args
//		want Filter
//	}{
//		// TODO: Add test cases.
//		{
//			name: "new_big_filter",
//			args: args{
//				n:                  3,
//				m:                  100 * 100 * 100,
//				fpRate:             0.00001,
//				redisServerCluster: redisServerCluster,
//				pass:               nil,
//				logger:             logger,
//			},
//			want: bigFilter{
//				filters: createRedisClient(),
//				logger:  logger,
//			},
//		},
//		{
//			name: "new_big_filter_bloomfilter_exist",
//			args: args{
//				n:                  3,
//				m:                  100 * 100 * 100,
//				fpRate:             0.00001,
//				redisServerCluster: redisServerCluster,
//				pass:               nil,
//				logger:             logger,
//			},
//			want: bigFilter{
//				filters: createRedisClient(),
//				logger:  logger,
//			},
//		},
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			//判断
//			got,_ := NewBigFilter(tt.args.n, tt.args.m, tt.args.fpRate, tt.args.redisServerCluster, tt.args.pass, tt.args.logger)
//			//todo:判断过滤器，是否可以add,exist,addmult,existmult,getlastpoint,savelastpoint几个方法是否正常
//			//如果正常，说明返回结果正确，如果不正常，说明创建过滤器失败
//			k := key1 + key1
//			err := got.Add([]byte(k))
//			if err != nil {
//				t.Errorf("NewBigFilter() error, err=[%s]", err)
//			}
//		})
//	}
//	destroyRedisHosts()
//}

func TestMain(m *testing.M) {
	if len(os.Getenv("BUILD_ID")) == 0 { //jenkins has it
		return
	}
	run := m.Run()
	os.Exit(run)
}
