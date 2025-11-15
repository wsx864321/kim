# KIM Project Makefile

# 变量定义
PROJECT_NAME := kim
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME := $(shell date +%Y-%m-%d\ %H:%M:%S)
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

# Go 相关变量
GO := go
GOFMT := gofmt
GOMOD := go mod
GOBUILD := $(GO) build
GOTEST := $(GO) test
GOVET := $(GO) vet


# 目录定义
CMD_DIR := cmd
BIN_DIR := bin
CONFIG_DIR := config
DEPLOYMENTS_DIR := deployments
IDL_DIR := idl


# Protobuf 相关
PROTOC := protoc
PROTOC_GEN_GO := protoc-gen-go
PROTOC_GEN_GO_GRPC := protoc-gen-go-grpc

# 帮助信息
.PHONY: help
help:
	@echo "KIM Project Makefile"
	@echo ""
	@echo "Usage:"
	@echo "  make <target>"
	@echo ""
	@echo "Targets:"
	@echo "  help          - 显示帮助信息"
	@echo "  proto         - 生成 protobuf 代码"
	@echo "  proto-gateway - 生成 gateway protobuf 代码"
	@echo "  proto-session - 生成 session protobuf 代码"
	@echo "  proto-push    - 生成 push protobuf 代码"
	@echo "  test          - 运行测试"
	@echo "  test-cover    - 运行测试并生成覆盖率报告"


# Protobuf 代码生成
.PHONY: proto
proto: proto-gateway proto-session proto-push

.PHONY: proto-gateway
proto-gateway:
	@echo "Generating gateway protobuf code..."
	@cd $(IDL_DIR)/gateway && \
		$(PROTOC) --go_out=. --go-grpc_out=. gateway.proto

.PHONY: proto-session
proto-session:
	@echo "Generating session protobuf code..."
	@cd $(IDL_DIR)/session && \
		$(PROTOC) --go_out=. --go-grpc_out=. session.proto

.PHONY: proto-push
proto-push:
	@echo "Generating push protobuf code..."
	@cd $(IDL_DIR)/push && \
		$(PROTOC) --go_out=. --go-grpc_out=. push.proto

# 测试
.PHONY: test
test:
	@echo "Running tests..."
	$(GOTEST) -v ./...

.PHONY: test-cover
test-cover:
	@echo "Running tests with coverage..."
	$(GOTEST) -v -coverprofile=coverage.out ./...
	$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"






