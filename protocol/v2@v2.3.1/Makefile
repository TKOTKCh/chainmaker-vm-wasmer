VERSION=v2.3.0_qc

mockgen:
	mockgen -destination ./mock/access_control_mock.go -package mock -source access_control_interface.go
	mockgen -destination ./mock/cache_mock.go -package mock -source cache_interface.go
	mockgen -destination ./mock/consensus_mock.go -package mock -source consensus_interface.go
	mockgen -destination ./mock/core_mock.go -package mock -source core_interface.go
	mockgen -destination ./mock/dpos_mock.go -package mock -source dpos_interface.go
	mockgen -destination ./mock/net_mock.go -package mock -source net_interface.go
	mockgen -destination ./mock/scheduler_mock.go -package mock -source scheduler_interface.go
	mockgen -destination ./mock/snapshot_mock.go -package mock -source snapshot_interface.go
	mockgen -destination ./mock/store_mock.go -package mock -source store_interface.go
	mockgen -destination ./mock/sync_mock.go -package mock -source sync_interface.go
	mockgen -destination ./mock/tx_pool_mock.go -package mock -source tx_pool_interface.go
	mockgen -destination ./mock/vm_mock.go -package mock -source vm_interface.go
	mockgen -destination ./mock/chainconf_mock.go -package mock -source chainconf_interface.go
	mockgen -destination ./mock/government_mock.go -package mock -source government_interface.go
	mockgen -destination ./mock/logger_mock.go -package mock -source logger_interface.go
	mockgen -destination ./mock/txfilter_mock.go -package mock -source tx_filiter_interface.go
	go mod tidy

mockgen-dep:
	go get -u github.com/golang/mock/gomock
	go get -u github.com/golang/mock/mockgen

gomod:
	go get chainmaker.org/chainmaker/common/v2@$(VERSION)
	go get chainmaker.org/chainmaker/pb-go/v2@$(VERSION)
	go mod tidy

stringer:
	go install golang.org/x/tools/cmd/stringer@latest
	stringer -type=VerifyMode

lint:
	golangci-lint run ./...