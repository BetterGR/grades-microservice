PROTO_DIR := protos
PROTO_FILE := $(PROTO_DIR)/grades_services.proto
OUT_DIR := $(PROTO_DIR)

.PHONY: all
all: generate

.PHONY: generate
generate:
	protoc --go_out=paths=source_relative:$(OUT_DIR) --go-grpc_out=paths=source_relative:$(OUT_DIR) -I $(PROTO_DIR) $(PROTO_FILE)