MODULE = github.com/from-cero/cero-id

format-tools:
	go install mvdan.cc/gofumpt@latest
	go install golang.org/x/tools/cmd/goimports@latest
	go install github.com/daixiang0/gci@latest
	go install github.com/segmentio/golines@latest

format:
	goimports -w .
	gci write --custom-order -s standard -s default -s "prefix($(MODULE))" -s blank \
		--no-lex-order --skip-generated --skip-vendor .
	golines -w -m 120 .
	gofumpt -l -w -extra .

lint-tools:
	curl -sSfL https://golangci-lint.run/install.sh | sh -s -- -b $(go env GOPATH)/bin v2.12.1

lint:
	golangci-lint run ./...

example-basic:
	go run examples/basic/main.go

example-burst:
	go run examples/burst/main.go

loadtest:
	go run examples/loadtest/main.go $(ARGS)

