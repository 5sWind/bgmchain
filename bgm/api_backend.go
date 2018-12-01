// Copyright 2015 The go-bgmchain Authors
// This file is part of the go-bgmchain library.
//
// The go-bgmchain library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-bgmchain library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-bgmchain library. If not, see <http://www.gnu.org/licenses/>.

package bgm

import (
	"context"
	"math/big"

	"github.com/5sWind/bgmchain/accounts"
	"github.com/5sWind/bgmchain/common"
	"github.com/5sWind/bgmchain/common/math"
	"github.com/5sWind/bgmchain/core"
	"github.com/5sWind/bgmchain/core/bloombits"
	"github.com/5sWind/bgmchain/core/state"
	"github.com/5sWind/bgmchain/core/types"
	"github.com/5sWind/bgmchain/core/vm"
	"github.com/5sWind/bgmchain/bgm/downloader"
	"github.com/5sWind/bgmchain/bgm/gasprice"
	"github.com/5sWind/bgmchain/bgmdb"
	"github.com/5sWind/bgmchain/event"
	"github.com/5sWind/bgmchain/params"
	"github.com/5sWind/bgmchain/rpc"
)

// BgmApiBackend implements bgmapi.Backend for full nodes
type BgmApiBackend struct {
	bgm *Bgmchain
	gpo *gasprice.Oracle
}

func (b *BgmApiBackend) ChainConfig() *params.ChainConfig {
	return b.bgm.chainConfig
}

func (b *BgmApiBackend) CurrentBlock() *types.Block {
	return b.bgm.blockchain.CurrentBlock()
}

func (b *BgmApiBackend) SetHead(number uint64) {
	b.bgm.protocolManager.downloader.Cancel()
	b.bgm.blockchain.SetHead(number)
}

func (b *BgmApiBackend) HeaderByNumber(ctx context.Context, blockNr rpc.BlockNumber) (*types.Header, error) {
	// Pending block is only known by the miner
	if blockNr == rpc.PendingBlockNumber {
		block := b.bgm.miner.PendingBlock()
		return block.Header(), nil
	}
	// Otherwise resolve and return the block
	if blockNr == rpc.LatestBlockNumber {
		return b.bgm.blockchain.CurrentBlock().Header(), nil
	}
	return b.bgm.blockchain.GetHeaderByNumber(uint64(blockNr)), nil
}

func (b *BgmApiBackend) BlockByNumber(ctx context.Context, blockNr rpc.BlockNumber) (*types.Block, error) {
	// Pending block is only known by the miner
	if blockNr == rpc.PendingBlockNumber {
		block := b.bgm.miner.PendingBlock()
		return block, nil
	}
	// Otherwise resolve and return the block
	if blockNr == rpc.LatestBlockNumber {
		return b.bgm.blockchain.CurrentBlock(), nil
	}
	return b.bgm.blockchain.GetBlockByNumber(uint64(blockNr)), nil
}

func (b *BgmApiBackend) StateAndHeaderByNumber(ctx context.Context, blockNr rpc.BlockNumber) (*state.StateDB, *types.Header, error) {
	// Pending state is only known by the miner
	if blockNr == rpc.PendingBlockNumber {
		block, state := b.bgm.miner.Pending()
		return state, block.Header(), nil
	}
	// Otherwise resolve the block number and return its state
	header, err := b.HeaderByNumber(ctx, blockNr)
	if header == nil || err != nil {
		return nil, nil, err
	}
	stateDb, err := b.bgm.BlockChain().StateAt(header.Root)
	return stateDb, header, err
}

func (b *BgmApiBackend) GetBlock(ctx context.Context, blockHash common.Hash) (*types.Block, error) {
	return b.bgm.blockchain.GetBlockByHash(blockHash), nil
}

func (b *BgmApiBackend) GetReceipts(ctx context.Context, blockHash common.Hash) (types.Receipts, error) {
	return core.GetBlockReceipts(b.bgm.chainDb, blockHash, core.GetBlockNumber(b.bgm.chainDb, blockHash)), nil
}

func (b *BgmApiBackend) GetTd(blockHash common.Hash) *big.Int {
	return b.bgm.blockchain.GetTdByHash(blockHash)
}

func (b *BgmApiBackend) GetEVM(ctx context.Context, msg core.Message, state *state.StateDB, header *types.Header, vmCfg vm.Config) (*vm.EVM, func() error, error) {
	state.SetBalance(msg.From(), math.MaxBig256)
	vmError := func() error { return nil }

	context := core.NewEVMContext(msg, header, b.bgm.BlockChain(), nil)
	return vm.NewEVM(context, state, b.bgm.chainConfig, vmCfg), vmError, nil
}

func (b *BgmApiBackend) SubscribeRemovedLogsEvent(ch chan<- core.RemovedLogsEvent) event.Subscription {
	return b.bgm.BlockChain().SubscribeRemovedLogsEvent(ch)
}

func (b *BgmApiBackend) SubscribeChainEvent(ch chan<- core.ChainEvent) event.Subscription {
	return b.bgm.BlockChain().SubscribeChainEvent(ch)
}

func (b *BgmApiBackend) SubscribeChainHeadEvent(ch chan<- core.ChainHeadEvent) event.Subscription {
	return b.bgm.BlockChain().SubscribeChainHeadEvent(ch)
}

func (b *BgmApiBackend) SubscribeLogsEvent(ch chan<- []*types.Log) event.Subscription {
	return b.bgm.BlockChain().SubscribeLogsEvent(ch)
}

func (b *BgmApiBackend) SendTx(ctx context.Context, signedTx *types.Transaction) error {
	return b.bgm.txPool.AddLocal(signedTx)
}

func (b *BgmApiBackend) GetPoolTransactions() (types.Transactions, error) {
	pending, err := b.bgm.txPool.Pending()
	if err != nil {
		return nil, err
	}
	var txs types.Transactions
	for _, batch := range pending {
		txs = append(txs, batch...)
	}
	return txs, nil
}

func (b *BgmApiBackend) GetPoolTransaction(hash common.Hash) *types.Transaction {
	return b.bgm.txPool.Get(hash)
}

func (b *BgmApiBackend) GetPoolNonce(ctx context.Context, addr common.Address) (uint64, error) {
	return b.bgm.txPool.State().GetNonce(addr), nil
}

func (b *BgmApiBackend) Stats() (pending int, queued int) {
	return b.bgm.txPool.Stats()
}

func (b *BgmApiBackend) TxPoolContent() (map[common.Address]types.Transactions, map[common.Address]types.Transactions) {
	return b.bgm.TxPool().Content()
}

func (b *BgmApiBackend) SubscribeTxPreEvent(ch chan<- core.TxPreEvent) event.Subscription {
	return b.bgm.TxPool().SubscribeTxPreEvent(ch)
}

func (b *BgmApiBackend) Downloader() *downloader.Downloader {
	return b.bgm.Downloader()
}

func (b *BgmApiBackend) ProtocolVersion() int {
	return b.bgm.BgmVersion()
}

func (b *BgmApiBackend) SuggestPrice(ctx context.Context) (*big.Int, error) {
	return b.gpo.SuggestPrice(ctx)
}

func (b *BgmApiBackend) ChainDb() bgmdb.Database {
	return b.bgm.ChainDb()
}

func (b *BgmApiBackend) EventMux() *event.TypeMux {
	return b.bgm.EventMux()
}

func (b *BgmApiBackend) AccountManager() *accounts.Manager {
	return b.bgm.AccountManager()
}

func (b *BgmApiBackend) BloomStatus() (uint64, uint64) {
	sections, _, _ := b.bgm.bloomIndexer.Sections()
	return params.BloomBitsBlocks, sections
}

func (b *BgmApiBackend) ServiceFilter(ctx context.Context, session *bloombits.MatcherSession) {
	for i := 0; i < bloomFilterThreads; i++ {
		go session.Multiplex(bloomRetrievalBatch, bloomRetrievalWait, b.bgm.bloomRequests)
	}
}
