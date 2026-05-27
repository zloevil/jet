.PHONY: dep build test test-with-coverage test-integration vet fmt lint mock proto

dep: ## tidy dependencies
	go mod tidy

build: ## build the library
	go build ./...

test: ## run unit tests (skips integration)
	go test -count=1 ./...

test-with-coverage: ## run unit tests with a coverage profile
	go test -count=1 -coverprofile=.testCoverage.txt ./...

test-integration: ## run integration tests (require real services: Postgres, Kafka, etc.)
	go test -count=1 -tags integration ./...

vet: ## run go vet
	go vet ./...

fmt: ## format the code
	go fmt ./...

lint: vet fmt ## vet + format

mock: ## (re)generate mocks (requires mockery)
	@rm -rf ./mocks 2>/dev/null; mockery

proto: ## generate protobuf code (requires protoc + go/grpc plugins)
	protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative ./grpc/*.proto ./centrifugo/proto/*.proto
