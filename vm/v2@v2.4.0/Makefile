VERSION=v2.4.0_qc

build:
	go mod tidy && go build ./...

ut:
	mkdir -p ../ut
	go test -covermode=count -coverprofile=../ut/vm.out ./...
	go tool cover -html=../ut/vm.out -o=../ut/vm.html

lint:
	golangci-lint run ./...

gomod:
	go get chainmaker.org/chainmaker/pb-go/v2@$(VERSION)
	go get chainmaker.org/chainmaker/common/v2@$(VERSION)
	go get chainmaker.org/chainmaker/protocol/v2@$(VERSION)
	go get chainmaker.org/chainmaker/utils/v2@$(VERSION)
	go get chainmaker.org/chainmaker/vm-native/v2@$(VERSION)
	go get chainmaker.org/chainmaker/logger/v2@$(VERSION)
	go mod tidy
