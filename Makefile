
export GOOS    := $(shell go env GOOS)
export GOARCH  := $(shell go env GOARCH)

PRODUCT := pubsub-cli

TARGET   		 ?= pubsub
TARGET_DIR   ?= $(PWD)/target
TARGET_NAME  ?= $(GOOS)_$(GOARCH)
TARGET_BUILD := $(TARGET_DIR)/$(TARGET_NAME)
TARGET_BIN   := $(TARGET_BUILD)/bin/$(TARGET)

RELEASE_ARCHIVE := $(PRODUCT)-$(GOOS)-$(GOARCH).tgz
RELEASE_PACKAGE := $(TARGET_BUILD)
RELEASE_BIN 		:= bin/$(TARGET)

.PHONY: all
all: build

$(TARGET_BUILD):
	mkdir -p $(TARGET_BUILD)

.PHONY: pubsub
pubsub: $(TARGET_BUILD)
	go build -o $(TARGET_BIN) $(PWD)/cmd

.PHONY: build
build: pubsub

.PHONY: archive
archive: build
	(cd $(TARGET_BUILD) && tar -zcf $(RELEASE_ARCHIVE) $(RELEASE_BIN))

.PHONY: install
install: build
	sudo install $(TARGET_BIN) /usr/local/bin

.PHONY: clean
clean:
	rm -rf $(TARGET_DIR)
