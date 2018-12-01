.PHONY: build cluster test

default: build

build:
	go build -o bin/swarm    github.com/bgmchain/bgmchain/cmd/swarm
	go build -o bin/gbgm     github.com/bgmchain/bgmchain/cmd/gbgm
	go build -o bin/bootnode github.com/bgmchain/bgmchain/cmd/bootnode

cluster: build
	scripts/boot-cluster.sh

test:
	go test -v github.com/bgmchain/bgmchain/swarm/...
