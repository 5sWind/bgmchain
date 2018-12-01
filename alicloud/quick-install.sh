#!/bin/bash

# install git
sudo yum upgrade
sudo yum install git

## tag version
GO_VERSION=1.11.2

# install go
wget https://dl.google.com/go/go${GO_VERSION}.linux-amd64.tar.gz
tar -C /usr/local -xzf go${GO_VERSION}.linux-amd64.tar.gz
mkdir -p ~/go; echo "export GOPATH=$HOME/go" >> ~/.bashrc
echo "export PATH=$PATH:$HOME/go/bin:/usr/local/go/bin" >> ~/.bashrc
source ~/.bashrc

# download repo
git clone https://github.com/5sWind/dposBgmchain

# start
cd dposBgmchain && make gbgm
./build/bin/gbgm init --datadir /path/to/datadir dpos_genesis.json
./build/bin/gbgm --datadir /path/to/datadir --keystore /path/to/keystore console