BUILD_FLAGS = -ldflags="-s -w" -trimpath
ENV = CGO_ENABLED=0 GOOS=linux GOARCH=amd64

build:
	$(ENV) go build $(BUILD_FLAGS) -o bin/server ./cmd/server

dev:
	go run cmd/server/main.go

test:
	go test -race -v --cover ./...

