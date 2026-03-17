.PHONY: init generate gen-proto gen-gorm gen-code gen-wire build run lint lint-go lint-proto clean

# Install all required tools
init:
	go install github.com/go-kratos/kratos/cmd/protoc-gen-go-http/v2@latest
	go install github.com/go-kratos/kratos/cmd/protoc-gen-go-errors/v2@latest
	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
	go install github.com/google/wire/cmd/wire@latest

# Run all code generation steps
generate: gen-proto gen-gorm gen-code gen-wire

# Step 1: Generate proto code (pb + grpc + http + errors)
gen-proto:
	buf generate

# Step 2: Generate GORM models and queries from schema SQL
gen-gorm:
	go run ./cmd/gormgen

# Step 3: Generate feature scaffolding (service/usecase/repo/wire)
gen-code:
	go run ./cmd/codegen

# Step 4: Generate Wire dependency injection
gen-wire:
	cd cmd/server && wire

# Build the server binary
build:
	go build -o bin/server.exe ./cmd/server

# Run the server
run:
	go run ./cmd/server -conf configs/config.yaml

# Lint everything
lint: lint-go lint-proto

# Go lint (architecture boundary + code quality)
lint-go:
	golangci-lint run ./...

# Proto lint
lint-proto:
	buf lint

# Remove generated files
clean:
	powershell -Command "Remove-Item gen -Recurse -Force -ErrorAction SilentlyContinue; Remove-Item cmd/server/wire_gen.go -Force -ErrorAction SilentlyContinue; Remove-Item bin -Recurse -Force -ErrorAction SilentlyContinue"
