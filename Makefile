.PHONY: run build test clean

APP_NAME=biscoli-backend

run:
	@go run cmd/api/main.go

build:
	@mkdir -p bin
	@go build -o bin/$(APP_NAME) cmd/api/main.go

test:
	@go test -v ./...

clean:
	@rm -rf bin
