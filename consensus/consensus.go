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

//
package consensus

import (
	"github.com/5sWind/bgmchain/common"
	"github.com/5sWind/bgmchain/core/state"
	"github.com/5sWind/bgmchain/core/types"
	"github.com/5sWind/bgmchain/params"
	"github.com/5sWind/bgmchain/rpc"
)

//
//
type ChainReader interface {
//
	Config() *params.ChainConfig

//
	CurrentHeader() *types.Header

//
	GetHeader(hash common.Hash, number uint64) *types.Header

//
	GetHeaderByNumber(number uint64) *types.Header

//
	GetHeaderByHash(hash common.Hash) *types.Header

//
	GetBlock(hash common.Hash, number uint64) *types.Block
}

//
type Engine interface {
//
//
//
	Author(header *types.Header) (common.Address, error)

//
//
//
	VerifyHeader(chain ChainReader, header *types.Header, seal bool) error

//
//
//
//
	VerifyHeaders(chain ChainReader, headers []*types.Header, seals []bool) (chan<- struct{}, <-chan error)

//
//
	VerifyUncles(chain ChainReader, block *types.Block) error

//
//
	VerifySeal(chain ChainReader, header *types.Header) error

//
//
	Prepare(chain ChainReader, header *types.Header) error

//
//
//
//
	Finalize(chain ChainReader, header *types.Header, state *state.StateDB, txs []*types.Transaction,
		uncles []*types.Header, receipts []*types.Receipt, dposContext *types.DposContext) (*types.Block, error)

//
//
	Seal(chain ChainReader, block *types.Block, stop <-chan struct{}) (*types.Block, error)

//
	APIs(chain ChainReader) []rpc.API
}

//
type PoW interface {
	Engine

//
	Hashrate() float64
}
