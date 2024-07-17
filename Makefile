#!/usr/bin/make -f

DOCKER := $(shell which docker)


###############################################################################
###                                Linting                                  ###
###############################################################################

lint:
	golangci-lint run --out-format=tab --timeout=15m

lint-fix:
	golangci-lint run --fix --out-format=tab --timeout=15m

.PHONY: lint lint-fix


###############################################################################
###                           Tests 
###############################################################################

test: test-unit

test-all: test-unit test-race test-cover

test-unit:
	@VERSION=$(VERSION) go test -mod=readonly -tags='ledger test_ledger_mock' ./...

test-race:
	@VERSION=$(VERSION) go test -mod=readonly -race -tags='ledger test_ledger_mock' ./...

test-cover:
	@go test -mod=readonly -timeout 30m -race -coverprofile=coverage.txt -covermode=atomic -tags='ledger test_ledger_mock' ./...

benchmark:
	@go test -timeout 20m -mod=readonly -bench=. ./... 

.PHONY: test test-all test-cover test-unit test-race benchmark
