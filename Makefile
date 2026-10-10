PROTOC ?= protoc
PROTO_DIR := api/proto
GEN_DIR := api/gen
PROTO_FILES := $(PROTO_DIR)/locky/v1/locky.proto
GO_BIN_DIR := $(shell go env GOPATH)/bin
SQLC ?= sqlc

.DEFAULT_GOAL := generate

.PHONY: generate generate-pb proto generate-sql install-tools

ifeq ($(OS),Windows_NT)
MKDIR_GEN = if not exist "$(GEN_DIR)" mkdir "$(GEN_DIR)"
export PATH := $(PATH);$(GO_BIN_DIR)
else
MKDIR_GEN = mkdir -p $(GEN_DIR)
export PATH := $(PATH):$(GO_BIN_DIR)
endif

generate: proto generate-sql

proto:
	$(MKDIR_GEN)
	$(PROTOC) -I $(PROTO_DIR) \
		--go_out=$(GEN_DIR) --go_opt=paths=source_relative \
		--go-grpc_out=$(GEN_DIR) --go-grpc_opt=paths=source_relative \
		$(PROTO_FILES)

generate-sql:
	$(SQLC) generate

install-tools:
	go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.34.2
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.5.1
