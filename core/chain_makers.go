//
//
//
//
//
//
//
//
//
//
//
//
//
//
//

package core

import (
	"fmt"
	"math/big"

	"github.com/5sWind/bgmchain/common"
	"github.com/5sWind/bgmchain/consensus/dpos"
	"github.com/5sWind/bgmchain/consensus/bgmash"
	"github.com/5sWind/bgmchain/consensus/misc"
	"github.com/5sWind/bgmchain/core/state"
	"github.com/5sWind/bgmchain/core/types"
	"github.com/5sWind/bgmchain/core/vm"
	"github.com/5sWind/bgmchain/bgmdb"
	"github.com/5sWind/bgmchain/params"
)

//
var (
	canonicalSeed = 1
	forkSeed      = 2
)

//
//
type BlockGen struct {
	i       int
	parent  *types.Block
	chain   []*types.Block
	header  *types.Header
	statedb *state.StateDB

	gasPool  *GasPool
	txs      []*types.Transaction
	receipts []*types.Receipt
	uncles   []*types.Header

	config *params.ChainConfig
}

//
//
func (b *BlockGen) SetCoinbase(addr common.Address) {
	if b.gasPool != nil {
		if len(b.txs) > 0 {
			panic("coinbase must be set before adding transactions")
		}
		panic("coinbase can only be set once")
	}
	b.header.Coinbase = addr
	b.gasPool = new(GasPool).AddGas(b.header.GasLimit)
}

//
func (b *BlockGen) SetExtra(data []byte) {
	b.header.Extra = data
}

//
//
//
//
//
//
//
//
func (b *BlockGen) AddTx(tx *types.Transaction) {
	if b.gasPool == nil {
		b.SetCoinbase(common.Address{})
	}
	b.statedb.Prepare(tx.Hash(), common.Hash{}, len(b.txs))
	receipt, _, err := ApplyTransaction(b.config, nil, nil, &b.header.Coinbase, b.gasPool, b.statedb, b.header, tx, b.header.GasUsed, vm.Config{})
	if err != nil {
		panic(err)
	}
	b.txs = append(b.txs, tx)
	b.receipts = append(b.receipts, receipt)
}

//
func (b *BlockGen) Number() *big.Int {
	return new(big.Int).Set(b.header.Number)
}

//
//
//
//
//
func (b *BlockGen) AddUncheckedReceipt(receipt *types.Receipt) {
	b.receipts = append(b.receipts, receipt)
}

//
//
func (b *BlockGen) TxNonce(addr common.Address) uint64 {
	if !b.statedb.Exist(addr) {
		panic("account does not exist")
	}
	return b.statedb.GetNonce(addr)
}

//
func (b *BlockGen) AddUncle(h *types.Header) {
	b.uncles = append(b.uncles, h)
}

//
//
//
func (b *BlockGen) PrevBlock(index int) *types.Block {
	if index >= b.i {
		panic("block index out of range")
	}
	if index == -1 {
		return b.parent
	}
	return b.chain[index]
}

//
//
//
func (b *BlockGen) OffsetTime(seconds int64) {
	b.header.Time.Add(b.header.Time, new(big.Int).SetInt64(seconds))
	if b.header.Time.Cmp(b.parent.Header().Time) <= 0 {
		panic("block time out of range")
	}
	b.header.Difficulty = bgmash.CalcDifficulty(b.config, b.header.Time.Uint64(), b.parent.Header())
}

//
//
//
//
//
//
//
//
//
//
//
//
func GenerateChain(config *params.ChainConfig, parent *types.Block, db bgmdb.Database, n int, gen func(int, *BlockGen)) ([]*types.Block, []types.Receipts) {
	if config == nil {
		config = params.DposChainConfig
	}
	blocks, receipts := make(types.Blocks, n), make([]types.Receipts, n)
	genblock := func(i int, h *types.Header, statedb *state.StateDB) (*types.Block, types.Receipts) {
		b := &BlockGen{parent: parent, i: i, chain: blocks, header: h, statedb: statedb, config: config}
//
		if daoBlock := config.DAOForkBlock; daoBlock != nil {
			limit := new(big.Int).Add(daoBlock, params.DAOForkExtraRange)
			if h.Number.Cmp(daoBlock) >= 0 && h.Number.Cmp(limit) < 0 {
				if config.DAOForkSupport {
					h.Extra = common.CopyBytes(params.DAOForkBlockExtra)
				}
			}
		}
		if config.DAOForkSupport && config.DAOForkBlock != nil && config.DAOForkBlock.Cmp(h.Number) == 0 {
			misc.ApplyDAOHardFork(statedb)
		}
//
		if gen != nil {
			gen(i, b)
		}
		dpos.AccumulateRewards(config, statedb, h, b.uncles)
		root, err := statedb.CommitTo(db, config.IsEIP158(h.Number))
		if err != nil {
			panic(fmt.Sprintf("state write error: %v", err))
		}
		h.Root = root
		h.DposContext = parent.Header().DposContext
		return types.NewBlock(h, b.txs, b.uncles, b.receipts), b.receipts
	}
	for i := 0; i < n; i++ {
		statedb, err := state.New(parent.Root(), state.NewDatabase(db))
		if err != nil {
			panic(err)
		}
		header := makeHeader(config, parent, statedb)
		block, receipt := genblock(i, header, statedb)
		blocks[i] = block
		receipts[i] = receipt
		parent = block
	}
	return blocks, receipts
}

func makeHeader(config *params.ChainConfig, parent *types.Block, state *state.StateDB) *types.Header {
	var time *big.Int
	if parent.Time() == nil {
		time = big.NewInt(10)
	} else {
		time = new(big.Int).Add(parent.Time(), big.NewInt(10)) //
	}

	return &types.Header{
		Root:        state.IntermediateRoot(config.IsEIP158(parent.Number())),
		ParentHash:  parent.Hash(),
		Coinbase:    parent.Coinbase(),
		Difficulty:  parent.Difficulty(),
		DposContext: &types.DposContextProto{},
		GasLimit:    CalcGasLimit(parent),
		GasUsed:     new(big.Int),
		Number:      new(big.Int).Add(parent.Number(), common.Big1),
		Time:        time,
	}
}

//
//
//
func newCanonical(n int, full bool) (bgmdb.Database, *BlockChain, error) {
//
	gspec := new(Genesis)
	db, _ := bgmdb.NewMemDatabase()
	genesis := gspec.MustCommit(db)

	blockchain, _ := NewBlockChain(db, params.AllBgmashProtocolChanges, bgmash.NewFaker(), vm.Config{})
//
	if n == 0 {
		return db, blockchain, nil
	}
	if full {
//
		blocks := makeBlockChain(genesis, n, db, canonicalSeed)
		_, err := blockchain.InsertChain(blocks)
		return db, blockchain, err
	}
//
	headers := makeHeaderChain(genesis.Header(), n, db, canonicalSeed)
	_, err := blockchain.InsertHeaderChain(headers, 1)
	return db, blockchain, err
}

//
func makeHeaderChain(parent *types.Header, n int, db bgmdb.Database, seed int) []*types.Header {
	blocks := makeBlockChain(types.NewBlockWithHeader(parent), n, db, seed)
	headers := make([]*types.Header, len(blocks))
	for i, block := range blocks {
		headers[i] = block.Header()
	}
	return headers
}

//
func makeBlockChain(parent *types.Block, n int, db bgmdb.Database, seed int) []*types.Block {
	blocks, _ := GenerateChain(params.DposChainConfig, parent, db, n, func(i int, b *BlockGen) {
		b.SetCoinbase(common.Address{0: byte(seed), 19: byte(i)})
	})
	return blocks
}
