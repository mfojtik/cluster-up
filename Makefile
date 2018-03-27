BINARY = cluster
GOOS = $(shell go env GOOS)
OUTPUT_DIR = _output/local/${GOOS}/bin

build:
	mkdir -p ${OUTPUT_DIR} && \
	go build -o ${OUTPUT_DIR}/${BINARY} ./cmd/cluster

install:
	go install -v ./cmd/cluster

all: build
.PHONY: build
