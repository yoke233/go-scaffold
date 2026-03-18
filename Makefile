.PHONY: help init bootstrap add-feature doctor upgrade-check docs fmt ci proto-breaking generate gen-proto gen-gorm gen-code gen-wire build run test lint lint-go lint-proto db-up db-down clean

PROTOC_GEN_GO_VERSION := v1.36.11
PROTOC_GEN_GO_GRPC_VERSION := v1.5.1
PROTOC_GEN_GO_HTTP_VERSION := v2.0.0-20260228034312-fe9258d38fd4
PROTOC_GEN_GO_ERRORS_VERSION := v2.0.0-20260105075216-c7a58ff59f80
PROTOC_GEN_OPENAPI_VERSION := v0.7.1
WIRE_VERSION := v0.7.0
BUF_VERSION := v1.65.0
GOLANGCI_LINT_VERSION := v1.64.8
BUF_BREAKING_AGAINST ?= .git#branch=main,subdir=api

help:
	@echo "Targets:"
	@echo "  make bootstrap   安装工具 + 生成代码 + 运行测试"
	@echo "  make add-feature name=order   新增业务域骨架"
	@echo "  make doctor      执行项目自检"
	@echo "  make upgrade-check   检查脚手架版本与关键产物"
	@echo "  make docs        生成 OpenAPI 文档"
	@echo "  make fmt         格式化 Go 代码"
	@echo "  make ci          本地执行 CI 流程"
	@echo "  make proto-breaking   检查 proto breaking change"
	@echo "  make generate    执行全部代码生成"
	@echo "  make test        运行 go test ./..."
	@echo "  make run         启动服务"
	@echo "  make db-up       启动本地 PostgreSQL"
	@echo "  make db-down     停止本地 PostgreSQL"
	@echo "  make clean       清理生成产物"

# Install all required tools with pinned versions.
init:
	go install github.com/bufbuild/buf/cmd/buf@$(BUF_VERSION)
	go install github.com/go-kratos/kratos/cmd/protoc-gen-go-http/v2@$(PROTOC_GEN_GO_HTTP_VERSION)
	go install github.com/go-kratos/kratos/cmd/protoc-gen-go-errors/v2@$(PROTOC_GEN_GO_ERRORS_VERSION)
	go install github.com/google/gnostic/cmd/protoc-gen-openapi@$(PROTOC_GEN_OPENAPI_VERSION)
	go install google.golang.org/protobuf/cmd/protoc-gen-go@$(PROTOC_GEN_GO_VERSION)
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@$(PROTOC_GEN_GO_GRPC_VERSION)
	go install github.com/google/wire/cmd/wire@$(WIRE_VERSION)
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)

bootstrap: init generate test

add-feature:
	go run ./cmd/scaffold add-feature -name $(name)

doctor:
	go run ./cmd/scaffold doctor

upgrade-check:
	go run ./cmd/scaffold upgrade --check

docs: gen-proto

fmt:
	gofmt -w $$(rg --files . -g '*.go')

ci: generate test lint proto-breaking

proto-breaking:
	buf breaking --against $(BUF_BREAKING_AGAINST)

# Run all code generation steps.
generate: gen-proto gen-gorm gen-code gen-wire

# Step 1: Generate proto code and OpenAPI docs.
gen-proto:
	powershell -Command "Remove-Item docs\\openapi -Recurse -Force -ErrorAction SilentlyContinue"
	buf generate

# Step 2: Generate GORM models and queries from schema SQL.
gen-gorm:
	go run ./cmd/gormgen

# Step 3: Generate feature scaffolding (service/usecase/repo/wire).
gen-code:
	go run ./cmd/codegen

# Step 4: Generate Wire dependency injection.
gen-wire:
	wire ./cmd/server

build:
	go build -o bin/server.exe ./cmd/server

run:
	go run ./cmd/server -conf configs/config.yaml

test:
	go test ./...

lint: lint-go lint-proto

lint-go:
	golangci-lint run ./...

lint-proto:
	buf lint

db-up:
	docker compose up -d postgres

db-down:
	docker compose down

clean:
	powershell -Command "Remove-Item gen -Recurse -Force -ErrorAction SilentlyContinue; Remove-Item docs\\openapi -Recurse -Force -ErrorAction SilentlyContinue; Remove-Item cmd/server/wire_gen.go -Force -ErrorAction SilentlyContinue; Remove-Item bin -Recurse -Force -ErrorAction SilentlyContinue"
