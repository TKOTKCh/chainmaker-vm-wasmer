module chainmaker.org/chainmaker/vm/v2

go 1.16

require (
	chainmaker.org/chainmaker/common/v2 v2.3.6
	chainmaker.org/chainmaker/logger/v2 v2.3.4
	chainmaker.org/chainmaker/pb-go/v2 v2.3.6
	chainmaker.org/chainmaker/protocol/v2 v2.3.7
	chainmaker.org/chainmaker/utils/v2 v2.3.6
	chainmaker.org/chainmaker/vm-native/v2 v2.3.6
	github.com/go-sql-driver/mysql v1.6.0 // indirect
	github.com/golang/mock v1.6.0
	github.com/google/uuid v1.3.0 // indirect
	github.com/pingcap/errors v0.11.5-0.20201126102027-b0a155152ca3 // indirect
	github.com/pingcap/log v0.0.0-20201112100606-8f1e84a3abc8 // indirect
	github.com/prometheus/procfs v0.6.0 // indirect
	github.com/shirou/gopsutil v3.21.4-0.20210419000835-c7a38de76ee5+incompatible // indirect
	github.com/stretchr/testify v1.7.0
	github.com/tklauser/go-sysconf v0.3.10 // indirect
)

replace github.com/linvon/cuckoo-filter => chainmaker.org/third_party/cuckoo-filter v1.0.0
