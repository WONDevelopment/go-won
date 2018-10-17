.PHONY: build cluster test

default: build

build:
	go build -o bin/swarm    github.com/worldopennetwork/go-won/cmd/swarm
	go build -o bin/geth     github.com/worldopennetwork/go-won/cmd/geth
	go build -o bin/bootnode github.com/worldopennetwork/go-won/cmd/bootnode

cluster: build
	scripts/boot-cluster.sh

test:
	go test -v github.com/worldopennetwork/go-won/swarm/...
