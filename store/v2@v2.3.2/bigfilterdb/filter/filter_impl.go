package filter

import (
	"strings"
	"sync"

	"chainmaker.org/chainmaker/protocol/v2"
	"github.com/spaolacci/murmur3"

	"strconv"

	redisbloom "github.com/RedisBloom/redisbloom-go"
)

const (
	lastBigFilterNumKeyStr = "lastBigFilterNumKey"
	redisBloomName         = "bigfilter_"
)

//var (
//redisErr = errors.New("redis error,when new bigfilter")
//)

type bigFilter struct {
	//filters  []*BoomFilters.StableBloomFilter
	//filters  []*BoomFilters.CuckooFilter
	//filters  []*BoomFilters.BloomFilter
	name    string
	filters []*redisbloom.Client
	logger  protocol.Logger
}

// NewBigFilter 创建一个 bigFilter
// @Description:
// args: n 子过滤器个数，m 每个子过滤器容量，fpRate 过滤器误判概率
// @param n
// @param m
// @param fpRate
// @param redisServerCluster
// @param pass
// @param name
// @param logger
// @return Filter
// @return error
func NewBigFilter(n int, m uint, fpRate float64, redisServerCluster []string, pass *string,
	name string, logger protocol.Logger) (
	Filter, error) {
	//filters := []*BoomFilters.StableBloomFilter{}
	//filters := []*BoomFilters.CuckooFilter{}
	//filters := []*BoomFilters.BloomFilter{}
	filters := []*redisbloom.Client{}

	for i := 0; i < n; i++ {
		//把过滤器，映射到对应的redisServerCluster 中，一共n个redis
		redisNum := i % (len(redisServerCluster))
		bloomFilter := redisbloom.NewClient(redisServerCluster[redisNum], redisServerCluster[redisNum], pass)
		//创建redisBloom 过滤器
		idxStr := strconv.Itoa(i)
		bloomName := name + redisBloomName + idxStr

		//filter已存在则创建失败，不会重新创建
		err := bloomFilter.Reserve(bloomName, fpRate, uint64(m))
		if err != nil {
			if !strings.Contains(err.Error(), "item exists") {
				return nil, err
			}
			logger.Infof("create filter [%s] in redis server [%s] err:[%s]", bloomName,
				redisServerCluster[redisNum], err)
		}
		filters = append(filters, bloomFilter)
	}

	b := bigFilter{
		name:    name,
		filters: filters,
		logger:  logger,
	}
	return b, nil

}

// 添加一个交易是否存在
func (b bigFilter) Add(key []byte) error {
	idx := Hash(key) % uint64(len(b.filters))
	idxStr := strconv.Itoa(int(idx))
	//b.filters[idx].Add(key)
	bloomName := b.name + redisBloomName + idxStr
	_, err := b.filters[idx].Add(bloomName, string(key))
	if err != nil {
		b.logger.Errorf("add key [%s] in bloomfilter:[%s] redisHost:[%s] err:[%s]", string(key), bloomName,
			b.filters[idx].Name, err)
		//fmt.Println("add key error:",err)
		return err
	}
	return nil
}

// 判断一批交易是否存在
func (b bigFilter) ExistMult(arr [][]byte) []bool {
	r := []bool{}
	m := make(map[string]bool)
	lock := &sync.RWMutex{}
	//keyArr := make([][]string,len(b.filters))
	keyArr := make([][]string, len(b.filters))

	wg := &sync.WaitGroup{}

	//先分组，再并发判断，提升速度
	for i := 0; i < len(arr); i++ {
		idx := Hash(arr[i]) % uint64(len(b.filters))
		//r = append(r,b.filters[idx].Test(arr[i]))
		keyArr[idx] = append(keyArr[idx], string(arr[i]))
	}

	wg.Add(len(keyArr))
	//按组，每组批量判断
	for j := 0; j < len(keyArr); j++ {
		go func(j int) {
			defer wg.Done()
			idxStr := strconv.Itoa(int(j))
			bloomName := b.name + redisBloomName + idxStr

			//如果为空，则不发请求
			if len(keyArr[j]) > 0 {
				multi, err := b.filters[j].BfExistsMulti(bloomName, keyArr[j])
				if err != nil {
					b.logger.Errorf("error,when ExistMult in redis:[%s], error:[%s]", b.filters[j].Name, err)
					//fmt.Println("error,when ExistMult, error :",err)
					for _, keyStr := range keyArr[j] {
						//fmt.Println("error,keyStr=",keyStr)
						b.logger.Errorf("error,keyStr:[%s]", keyStr)
					}
				}
				//归并一次结果集
				for k := 0; k < len(multi); k++ {
					lock.Lock()
					if multi[k] > 0 {
						m[keyArr[j][k]] = true
					} else {
						m[keyArr[j][k]] = false
					}
					lock.Unlock()
				}
			}

		}(j)
	}
	wg.Wait()

	//把结果集从map中读取出来，写到返回值中
	for h := 0; h < len(arr); h++ {
		r = append(r, m[string(arr[h])])
	}
	//
	return r
}

// 判断一个key 是否存在
func (b bigFilter) Exist(key []byte) (bool, error) {
	idx := Hash(key) % uint64(len(b.filters))

	idxStr := strconv.Itoa(int(idx))
	//b.filters[idx].Add(key)
	bloomName := b.name + redisBloomName + idxStr

	//return b.filters[idx].Test(key)
	exists, err := b.filters[idx].Exists(bloomName, string(key))
	if err != nil {
		b.logger.Errorf("redis bloomFiter exists in redishost:[%s],key: [%s] error:[%s]", b.filters[idx].Name,
			string(key), err)
		//fmt.Println("redis bloomFiter exists error:",err)
	}
	return exists, err

}

// 添加一批key到过滤器
func (b bigFilter) AddMult(arr [][]byte) error {
	keyArr := make([][]string, len(b.filters))

	wg := &sync.WaitGroup{}
	errChan := make(chan error, len(keyArr))

	//先分组，再并发判断，提升速度
	for i := 0; i < len(arr); i++ {
		idx := Hash(arr[i]) % uint64(len(b.filters))
		keyArr[idx] = append(keyArr[idx], string(arr[i]))
	}

	wg.Add(len(keyArr))
	//按组，每组批量判断
	for j := 0; j < len(keyArr); j++ {
		go func(j int) {
			defer wg.Done()
			idxStr := strconv.Itoa(int(j))
			bloomName := b.name + redisBloomName + idxStr

			//如果为空，则不发请求
			if len(keyArr[j]) > 0 {
				_, err := b.filters[j].BfAddMulti(bloomName, keyArr[j])
				if err != nil {
					b.logger.Errorf("error,when BfAddMulti, errorInfo:[%s]", err)
					//fmt.Println("error,when BfAddMulti, error :",err)
					for _, keyStr := range keyArr[j] {
						//fmt.Println("error,keyStr=",keyStr)
						b.logger.Errorf("error,keyStr:[%s],errorInfo:[%s]", keyStr, err)
					}
					errChan <- err
				}
			}
		}(j)
	}
	wg.Wait()

	if len(errChan) > 0 {
		return <-errChan
	}
	return nil
}

// Save last point to redis
func (b bigFilter) SaveLastPoint(pointK string, pointV string) error {
	lastSaveName := b.name + pointK
	wg := &sync.WaitGroup{}
	wg.Add(len(b.filters))
	errChan := make(chan error, len(b.filters))
	for _, r := range b.filters {
		go func(r *redisbloom.Client) {
			defer wg.Done()
			_, err := r.SetKV(lastSaveName, pointV)
			if err != nil {
				//fmt.Println("save last point error",pointK,pointV,err)
				b.logger.Errorf("save last point,[%s],[%s], error:[%s]", pointK, pointV, err)
				//return err
				errChan <- err
			}
		}(r)
	}
	wg.Wait()
	if len(errChan) > 0 {
		return <-errChan
	}
	return nil
}

// GetLastPoint add next time
// @Description:
// @receiver b
// @param pointK
// @return []string
// @return error
// todo: 改成并发进行，性能更好, 本函数调用频率很低，只有奔溃恢复时调用1次
func (b bigFilter) GetLastPoint(pointK string) ([]string, error) {
	lastSaveName := b.name + pointK
	re := []string{}
	for _, r := range b.filters {
		data, err := r.GetKey(lastSaveName)
		if err != nil {
			//fmt.Println("get last point error",pointK,err)
			b.logger.Errorf("get last point,[%s], error:[%s]", pointK, err)
			return nil, err
		}
		re = append(re, data)
	}
	return re, nil
}

// Hash compute hash
// @Description:
// @param bys
// @return uint64
// todo: 测试这样分片是否均匀,已测试，很均匀
func Hash(bys []byte) uint64 {
	return murmur3.Sum64WithSeed(bys, 127)
}
