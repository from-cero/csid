APP = server
BUILD_FLAGS = -ldflags="-s -w" -trimpath
ENV = CGO_ENABLED=0 GOOS=linux GOARCH=amd64

build:
	$(ENV) go build $(BUILD_FLAGS) -o bin/$(APP) ./cmd/$(APP)

dev:
	go run cmd/$(APP)/main.go

test:
	go test -race -v --cover ./...

bench:
	./benchmark/bench.sh

bench-args:
	./benchmark/bench.sh -d $(or $(d),30s) -R $(or $(R),5000) -t $(or $(t),4) -c $(or $(c),50)

