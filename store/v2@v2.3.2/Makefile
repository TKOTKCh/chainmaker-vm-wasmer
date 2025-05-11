VERSION=v2.3.0_qc
gomod:
	go get chainmaker.org/chainmaker/common/v2@$(VERSION)
	go get chainmaker.org/chainmaker/pb-go/v2@$(VERSION)
	go get chainmaker.org/chainmaker/protocol/v2@$(VERSION)
	go get chainmaker.org/chainmaker/store-badgerdb/v2@$(VERSION)
	go get chainmaker.org/chainmaker/store-leveldb/v2@$(VERSION)
	go get chainmaker.org/chainmaker/store-sqldb/v2@$(VERSION)
	go get chainmaker.org/chainmaker/store-tikv/v2@$(VERSION)
	go get chainmaker.org/chainmaker/utils/v2@$(VERSION)
	go get chainmaker.org/chainmaker/lws@v1.1.0_qc
	go mod tidy
ut:
	go test ./...
	(gocloc --include-lang=Go --output-type=json . | jq '(.total.comment-.total.files*6)/(.total.code+.total.comment)*100')

mockgen:
	mockgen -destination ./statedb/statedb_mock.go -package statedb -source ./statedb/statedb.go
	mockgen -destination ./blockdb/blockdb_mock.go -package blockdb -source ./blockdb/blockdb.go
