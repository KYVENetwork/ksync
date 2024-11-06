GO_VERSION := $(shell go version | cut -c 14- | cut -d' ' -f1 | cut -d'.' -f1,2)

###############################################################################
###                                  Build                                  ###
###############################################################################

build: ensure_version
	go build -mod=readonly -o ./build/ksync ./cmd/ksync/main.go

test: ensure_version
	go test -v ./test/*

###############################################################################
###                                 Checks                                  ###
###############################################################################

ensure_version:
ifneq ($(GO_VERSION),1.22)
	$(error ❌  Please run Go v1.22.x..)
endif