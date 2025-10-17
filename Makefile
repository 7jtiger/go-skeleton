# Go parameters
GO := go
GOBUILD := $(GO) build
GOCLEAN := $(GO) clean
GOTEST := $(GO) test
GOGET := $(GO) get
GORUN := $(GO) run

# Binary names
PACKAGE := ms-gateway
BASE := $(CURDIR)
BUILD_DIR := $(BASE)/build
BINARY := $(BUILD_DIR)/$(PACKAGE)

# Flags
LDFLAGS := -ldflags="-s -w"
BUILDFLAGS := -v

# Environment variables
export GOOS := linux
export GOARCH := amd64

# File list
SOURCES := $(shell find . -name '*.go')
CONFIGS := $(wildcard conf/*.toml)

# PHONY target
.PHONY: all build test clean run exe swg

# Default target
all: build

# Build target
build: $(BINARY)

$(BINARY): $(SOURCES)
	@echo "Building $(PACKAGE)..."
	@mkdir -p $(BUILD_DIR)/conf $(BUILD_DIR)/data $(BUILD_DIR)/logs
	$(GOBUILD) $(BUILDFLAGS) $(LDFLAGS) -o $@ 
	@cp -f $(CONFIGS) $(BUILD_DIR)/conf/

# Test target
test:
	$(GOTEST) -v ./...

# Clean target
clean:
	@echo "Cleaning..."
	@rm -rf $(BUILD_DIR)
	$(GOCLEAN)

# Run target
run:
	$(GORUN) main.go

# Build and run target
exe: build
	./$(BINARY)

# Swagger build target
swg:
	@echo "Building Swagger..."
	@swag init -g main.go

# Dependency management
$(GO_FILES): go.mod go.sum
	@echo "Checking dependencies..."
	@$(GO) mod tidy

# Parallel execution support
.NOTPARALLEL:
