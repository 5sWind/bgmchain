.PHONY: build cluster test

default: build

build:
	go build -o bin/swarm    github.com/bgmchain/go-bgmchain/cmd/swarm
	go build -o bin/gbgm     github.com/bgmchain/go-bgmchain/cmd/gbgm
	go build -o bin/bootnode github.com/bgmchain/go-bgmchain/cmd/bootnode

cluster: build
	scripts/boot-cluster.sh

test:
	go test -v github.com/bgmchain/go-bgmchain/swarm/...
