SHELL = /bin/bash
BINARY = cluster
GOOS = $(shell go env GOOS)
OUTPUT_DIR = _output/local/${GOOS}/bin

clean:
	rm -rf _output

all:
	mkdir -p ${OUTPUT_DIR} && \
	go build -o ${OUTPUT_DIR}/${BINARY} ./cmd/cluster