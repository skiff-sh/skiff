SHELL := /bin/bash

GOBIN ?= $$(go env GOPATH)/bin

.PHONY: *

all:
	cd cmd/skiff && go build -o ../../bin/skiff main.go

work:
	go work init
	go work use api/go

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

proto:
	buf lint
	rm -rf api/go/*.pb.go
	buf generate

gen: mocks proto

croc.send:
	 CROC_SECRET=skiff123 croc send --git --exclude  "api,.git,.idea,mocks" ./*

croc.receive:
	croc --yes --overwrite skiff123

croc: croc.receive gen

update.api:
	go get github.com/skiff-sh/skiff/api/go
	cd sdk-go && go get github.com/skiff-sh/skiff/api/go
	cd examples/go-fiber-controller && go get github.com/skiff-sh/skiff/api/go
