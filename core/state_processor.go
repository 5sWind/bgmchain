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

	"github.com/5sWind/bgmchain/common"
	"github.com/5sWind/bgmchain/consensus"
	"github.com/5sWind/bgmchain/consensus/misc"
	"github.com/5sWind/bgmchain/core/state"
	"github.com/5sWind/bgmchain/core/types"
	"github.com/5sWind/bgmchain/core/vm"
	"github.com/5sWind/bgmchain/crypto"
	"github.com/5sWind/bgmchain/params"
)

//
//
//
//
type StateProcessor struct {
	config *params.ChainConfig //
	bc     *BlockChain         //
	engine consensus.Engine    //
}

//
func NewStateProcessor(config *params.ChainConfig, bc *BlockChain, engine consensus.Engine) *StateProcessor {
	return &StateProcessor{
		config: config,
		bc:     bc,
		engine: engine,
	}
}

//
//
//
//
//
//
//
func (p *StateProcessor) Process(block *types.Block, statedb *state.StateDB, cfg vm.Config) (types.Receipts, []*types.Log, *big.Int, error) {
	var (
		receipts     types.Receipts
		totalUsedGas = big.NewInt(0)
		header       = block.Header()
		allLogs      []*types.Log
		gp           = new(GasPool).AddGas(block.GasLimit())
	)
//
	if p.config.DAOForkSupport && p.config.DAOForkBlock != nil && p.config.DAOForkBlock.Cmp(block.Number()) == 0 {
		misc.ApplyDAOHardFork(statedb)
	}
//
//
	for i, tx := range block.Transactions() {
		statedb.Prepare(tx.Hash(), block.Hash(), i)
		receipt, _, err := ApplyTransaction(p.config, block.DposCtx(), p.bc, nil, gp, statedb, header, tx, totalUsedGas, cfg)
		if err != nil {
			return nil, nil, nil, err
		}
		receipts = append(receipts, receipt)
		allLogs = append(allLogs, receipt.Logs...)
	}
//
	p.engine.Finalize(p.bc, header, statedb, block.Transactions(), block.Uncles(), receipts, block.DposCtx())

	return receipts, allLogs, totalUsedGas, nil
}

//
//
//
//
func ApplyTransaction(config *params.ChainConfig, dposContext *types.DposContext, bc *BlockChain, author *common.Address, gp *GasPool, statedb *state.StateDB, header *types.Header, tx *types.Transaction, usedGas *big.Int, cfg vm.Config) (*types.Receipt, *big.Int, error) {
	msg, err := tx.AsMessage(types.MakeSigner(config, header.Number))
	if err != nil {
		return nil, nil, err
	}

	if msg.To() == nil && msg.Type() != types.Binary {
		return nil, nil, types.ErrInvalidType
	}

//
	context := NewEVMContext(msg, header, bc, author)
//
//
	vmenv := vm.NewEVM(context, statedb, config, cfg)
//
	_, gas, failed, err := ApplyMessage(vmenv, msg, gp)
	if err != nil {
		return nil, nil, err
	}
	if msg.Type() != types.Binary {
		if err = applyDposMessage(dposContext, msg); err != nil {
			return nil, nil, err
		}
	}

//
	var root []byte
	if config.IsByzantium(header.Number) {
		statedb.Finalise(true)
	} else {
		root = statedb.IntermediateRoot(config.IsEIP158(header.Number)).Bytes()
	}
	usedGas.Add(usedGas, gas)

//
//
	receipt := types.NewReceipt(root, failed, usedGas)
	receipt.TxHash = tx.Hash()
	receipt.GasUsed = new(big.Int).Set(gas)
//
	if msg.To() == nil {
		receipt.ContractAddress = crypto.CreateAddress(vmenv.Context.Origin, tx.Nonce())
	}

//
	receipt.Logs = statedb.GetLogs(tx.Hash())
	receipt.Bloom = types.CreateBloom(types.Receipts{receipt})

	return receipt, gas, err
}

func applyDposMessage(dposContext *types.DposContext, msg types.Message) error {
	switch msg.Type() {
	case types.LoginCandidate:
		dposContext.BecomeCandidate(msg.From())
	case types.LogoutCandidate:
		dposContext.KickoutCandidate(msg.From())
	case types.Delegate:
		dposContext.Delegate(msg.From(), *(msg.To()))
	case types.UnDelegate:
		dposContext.UnDelegate(msg.From(), *(msg.To()))
	default:
		return types.ErrInvalidType
	}
	return nil
}
