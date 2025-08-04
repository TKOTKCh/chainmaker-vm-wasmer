VERSION=v2.3.5

build:
	go mod tidy && go build ./...

gomod:
	go get chainmaker.org/chainmaker/chainconf/v2@v2.3.4
	go get chainmaker.org/chainmaker/common/v2@$(VERSION)
	go get chainmaker.org/chainmaker/logger/v2@v2.3.4
	go get chainmaker.org/chainmaker/pb-go/v2@v2.3.6
	go get chainmaker.org/chainmaker/protocol/v2@v2.3.6
	go get chainmaker.org/chainmaker/store/v2@v2.3.6
	go get chainmaker.org/chainmaker/utils/v2@$(VERSION)
	go get chainmaker.org/chainmaker/vm/v2@v2.3.6
	go mod tidy

lint:
	golangci-lint run ./...