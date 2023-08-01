#!/usr/bin/make -f

ksync:
	go build -mod=readonly -o ./build/ksync ./cmd/ksync/main.go