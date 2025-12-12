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

update:
	go get github.com/skiff-sh/api/go@main
	go get github.com/skiff-sh/sdk-go@main
	cd examples/go-fiber-controller && go get github.com/skiff-sh/api/go@main
	cd examples/go-fiber-controller && go get github.com/skiff-sh/sdk-go@main
	cd examples/go-fiber-controller/.skiff/plugins && go get github.com/skiff-sh/api/go@main
	cd examples/go-fiber-controller/.skiff/plugins && go get github.com/skiff-sh/sdk-go@main

update.force:
	etc/update.sh ../api github.com/skiff-sh/api/go
	etc/update.sh ../sdk-go github.com/skiff-sh/sdk-go
	cd examples/go-fiber-controller && ../../etc/update.sh ../../../sdk-go github.com/skiff-sh/sdk-go
	cd examples/go-fiber-controller && ../../etc/update.sh ../../../api github.com/skiff-sh/api/go

testdata:
	rm -f pkg/plugin/testdata/*.wasm
	etc/testdata.sh
