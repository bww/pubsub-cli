
export GOOS    := $(shell go env GOOS)
export GOARCH  := $(shell go env GOARCH)

PRODUCT      := pubsub
TARGET_DIR   ?= $(PWD)/target
TARGET_NAME  ?= $(GOOS)_$(GOARCH)
TARGET_BUILD := $(TARGET_DIR)/$(TARGET_NAME)
TARGET_BIN   := $(TARGET_BUILD)/bin/$(PRODUCT)

.PHONY: all
all: build

$(TARGET_BUILD):
	mkdir -p $(TARGET_BUILD)

.PHONY: pubsub
pubsub: $(TARGET_BUILD)
	go build -o $(TARGET_BIN) $(PWD)/cmd

.PHONY: build
build: pubsub

.PHONY: install
install: build
	sudo install $(TARGET_BIN) /usr/local/bin

.PHONY: clean
clean:
	rm -rf $(TARGET_DIR)
