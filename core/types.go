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
	"math/big"

	"github.com/5sWind/bgmchain/core/state"
	"github.com/5sWind/bgmchain/core/types"
	"github.com/5sWind/bgmchain/core/vm"
)

//
//
//
//
type Validator interface {
//
	ValidateBody(block *types.Block) error

//
//
	ValidateState(block, parent *types.Block, state *state.StateDB, receipts types.Receipts, usedGas *big.Int) error
//
	ValidateDposState(block *types.Block) error
}

//
//
//
//
//
//
type Processor interface {
	Process(block *types.Block, statedb *state.StateDB, cfg vm.Config) (types.Receipts, []*types.Log, *big.Int, error)
}
