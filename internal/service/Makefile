MODULE = github.com/from-cero/csid/service
BUILD_FLAGS = -ldflags="-s -w" -trimpath
ENV = CGO_ENABLED=0 GOOS=linux GOARCH=amd64

format-tools:
	go install mvdan.cc/gofumpt@latest
	go install golang.org/x/tools/cmd/goimports@latest
	go install github.com/daixiang0/gci@latest
	go install github.com/segmentio/golines@latest

format:
	go mod tidy
	goimports -w .
	gci write --custom-order -s standard -s default -s "prefix($(MODULE))" -s blank \
		--no-lex-order --skip-generated --skip-vendor .
	golines -w -m 120 .
	gofumpt -l -w -extra .

lint-tools:
	curl -sSfL https://golangci-lint.run/install.sh | sh -s -- -b $$(go env GOPATH)/bin v2.12.1

lint:
	golangci-lint run ./...

build:
	$(ENV) go build $(BUILD_FLAGS) -o bin/abc ./cmd/abc

dev:
	go run cmd/abc/main.go

test:
	go test -race -v --cover ./...

