SHELL := /bin/bash

GOBIN ?= $$(go env GOPATH)/bin

.PHONY: *

all:
	cd cmd/skiff && go build -o ../../bin/skiff main.go

gen: mocks testdata

mocks:
	rm -rf pkg/mocks
	find . -type f -name 'mock_*.go' -delete
	mockery

lint:
	golangci-lint run --fix

fmt:
	golangci-lint fmt

test:
	go test -count=1 -v ./... -coverprofile=./cover.out -covermode=atomic -coverpkg=./...

cover:
	${GOBIN}/go-test-coverage --config=./.testcoverage.yml

test.cover: test cover

croc.send:
	 CROC_SECRET=skiff123 croc send -c skiff123 --git --exclude  "api,.git,.idea,mocks" ./*

croc.receive:
	croc --yes --overwrite skiff123

croc: croc.receive mocks

update.api:
	cd cmd && go get github.com/skiff-sh/api/go
	cd sdk-go && go get github.com/skiff-sh/api/go
	cd examples/go-fiber-controller && go get github.com/skiff-sh/api/go

testdata:
	rm -f pkg/plugin/testdata/*.wasm
	GOOS=wasip1 GOARCH=wasm go build -target=wasi -buildmode=c-shared -o pkg/plugin/testdata/basic_plugin.wasm pkg/plugin/testdata/basic_plugin.go
