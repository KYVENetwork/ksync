GO_VERSION := $(shell go version | cut -c 14- | cut -d' ' -f1 | cut -d'.' -f1,2)

###############################################################################
###                                  Build                                  ###
###############################################################################

build: ensure_version
	go build -mod=readonly -o ./build/ksync ./cmd/ksync/main.go

###############################################################################
###                                  Tests                                  ###
###############################################################################

test-setup:
	docker build --platform linux/amd64 -t docker-ksync-test .

test:
	docker run --platform linux/amd64 --rm docker-ksync-test

###############################################################################
###                                 Checks                                  ###
###############################################################################

ensure_version:
ifneq ($(GO_VERSION),1.22)
	$(error ❌  Please run Go v1.22.x..)
endif
