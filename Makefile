SHELL := /bin/bash

GOBIN ?= $$(go env GOPATH)/bin

.PHONY: *

mocks:
	rm -rf pkg/mocks
	find . -type f -name 'mock_*.go' -delete
	mockery

lint:
	golangci-lint run --fix

fmt:
	golangci-lint fmt

proto:
	buf lint
	buf generate

gen: mocks proto
