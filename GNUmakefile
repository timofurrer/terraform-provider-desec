GOBIN=$(shell pwd)/bin

ifdef RUN
TESTARGS += -test.run $(RUN)
endif

default: fmt lint install generate

build:
	go build -v ./...

playground:
	GOBIN=$(GOBIN) go install -v

install: build
	go install -v ./...

lint:
	golangci-lint run

generate:
	cd tools; go generate ./...

fmt:
	gofmt -s -w -e .

test:
	go test -v -cover -timeout=120s -parallel=10 $(TESTARGS) ./...

testacc:
	TF_ACC=1 go test -v -cover -timeout 120m $(TESTARGS) ./...

.PHONY: fmt lint test testacc build install generate playground
