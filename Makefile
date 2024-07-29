#!/usr/bin/make -f

BRANCH := $(shell git rev-parse --abbrev-ref HEAD)
COMMIT := $(shell git log -1 --format='%H')
BUILDDIR ?= $(CURDIR)/build

# don't override user values
ifeq (,$(VERSION))
  VERSION := $(shell git describe --tags)
  # if VERSION is empty, then populate it with branch's name and raw commit hash
  ifeq (,$(VERSION))
    VERSION := $(BRANCH)-$(COMMIT)
  endif
endif

export GO111MODULE = on

# Build target
BINARY_NAME = opinitd

build_tags = netgo
build_tags += $(BUILD_TAGS)
build_tags := $(strip $(build_tags))

ldflags = -X github.com/initia-labs/opinit-bots-go/version.Version=$(VERSION) \
		  -X github.com/initia-labs/opinit-bots-go/version.GitCommit=$(COMMIT)

ldflags += $(LDFLAGS)
ldflags := $(strip $(ldflags))
ldflags += -w -s

BUILD_FLAGS := -tags "$(build_tags)" -ldflags '$(ldflags)'

all: install test

build: go.sum
ifeq ($(OS),Windows_NT)
	exit 1
else
	go build -mod=readonly $(BUILD_FLAGS) -o build/$(BINARY_NAME) ./cmd/$(BINARY_NAME)
endif

install: go.sum 
	go install -mod=readonly $(BUILD_FLAGS) ./cmd/$(BINARY_NAME)

.PHONY: build install

########################################
### Tools & dependencies

go-mod-cache: go.sum
	@echo "--> Download go modules to local cache"
	@go mod download

go.sum: go.mod
	@echo "--> Ensure dependencies have not been modified"
	@go mod verify

draw-deps:
	@# requires brew install graphviz or apt-get install graphviz
	go install github.com/RobotsAndPencils/goviz
	@goviz -i ./cmd/initiad -d 2 | dot -Tpng -o dependency-graph.png

clean:
	rm -rf \
    $(BUILDDIR)/ 

.PHONY: clean

###############################################################################
###                                Linting                                  ###
###############################################################################

lint:
	golangci-lint run --out-format=tab --timeout=15m

lint-fix:
	golangci-lint run --fix --out-format=tab --timeout=15m
.PHONY: lint lint-fix

format:
	find . -name '*.go' -type f -not -path "./vendor*" -not -path "*.git*" -not -path "./client/docs/statik/statik.go" -not -path "./tests/mocks/*" -not -name '*.pb.go' | xargs gofmt -w -s
	find . -name '*.go' -type f -not -path "./vendor*" -not -path "*.git*" -not -path "./client/docs/statik/statik.go" -not -path "./tests/mocks/*" -not -name '*.pb.go' | xargs misspell -w
	find . -name '*.go' -type f -not -path "./vendor*" -not -path "*.git*" -not -path "./client/docs/statik/statik.go" -not -path "./tests/mocks/*" -not -name '*.pb.go' | xargs goimports -w -local github.com/cosmos/cosmos-sdk
.PHONY: format
