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

package bgm

import (
	"context"
	"math/big"

	"github.com/5sWind/bgmchain"
	"github.com/5sWind/bgmchain/common"
	"github.com/5sWind/bgmchain/common/hexutil"
	"github.com/5sWind/bgmchain/core/types"
	"github.com/5sWind/bgmchain/internal/bgmapi"
	"github.com/5sWind/bgmchain/rlp"
	"github.com/5sWind/bgmchain/rpc"
)

//
//
//
//
//
//
//
type ContractBackend struct {
	eapi  *bgmapi.PublicBgmchainAPI        //
	bcapi *bgmapi.PublicBlockChainAPI      //
	txapi *bgmapi.PublicTransactionPoolAPI //
}

//
//
func NewContractBackend(apiBackend bgmapi.Backend) *ContractBackend {
	return &ContractBackend{
		eapi:  bgmapi.NewPublicBgmchainAPI(apiBackend),
		bcapi: bgmapi.NewPublicBlockChainAPI(apiBackend),
		txapi: bgmapi.NewPublicTransactionPoolAPI(apiBackend, new(bgmapi.AddrLocker)),
	}
}

//
func (b *ContractBackend) CodeAt(ctx context.Context, contract common.Address, blockNum *big.Int) ([]byte, error) {
	return b.bcapi.GetCode(ctx, contract, toBlockNumber(blockNum))
}

//
func (b *ContractBackend) PendingCodeAt(ctx context.Context, contract common.Address) ([]byte, error) {
	return b.bcapi.GetCode(ctx, contract, rpc.PendingBlockNumber)
}

//
//
//
func (b *ContractBackend) CallContract(ctx context.Context, msg bgmchain.CallMsg, blockNum *big.Int) ([]byte, error) {
	out, err := b.bcapi.Call(ctx, toCallArgs(msg), toBlockNumber(blockNum))
	return out, err
}

//
//
//
func (b *ContractBackend) PendingCallContract(ctx context.Context, msg bgmchain.CallMsg) ([]byte, error) {
	out, err := b.bcapi.Call(ctx, toCallArgs(msg), rpc.PendingBlockNumber)
	return out, err
}

func toCallArgs(msg bgmchain.CallMsg) bgmapi.CallArgs {
	args := bgmapi.CallArgs{
		To:   msg.To,
		From: msg.From,
		Data: msg.Data,
	}
	if msg.Gas != nil {
		args.Gas = hexutil.Big(*msg.Gas)
	}
	if msg.GasPrice != nil {
		args.GasPrice = hexutil.Big(*msg.GasPrice)
	}
	if msg.Value != nil {
		args.Value = hexutil.Big(*msg.Value)
	}
	return args
}

func toBlockNumber(num *big.Int) rpc.BlockNumber {
	if num == nil {
		return rpc.LatestBlockNumber
	}
	return rpc.BlockNumber(num.Int64())
}

//
//
func (b *ContractBackend) PendingNonceAt(ctx context.Context, account common.Address) (nonce uint64, err error) {
	out, err := b.txapi.GetTransactionCount(ctx, account, rpc.PendingBlockNumber)
	if out != nil {
		nonce = uint64(*out)
	}
	return nonce, err
}

//
//
func (b *ContractBackend) SuggestGasPrice(ctx context.Context) (*big.Int, error) {
	return b.eapi.GasPrice(ctx)
}

//
//
//
//
//
func (b *ContractBackend) EstimateGas(ctx context.Context, msg bgmchain.CallMsg) (*big.Int, error) {
	out, err := b.bcapi.EstimateGas(ctx, toCallArgs(msg))
	return out.ToInt(), err
}

//
//
func (b *ContractBackend) SendTransaction(ctx context.Context, tx *types.Transaction) error {
	raw, _ := rlp.EncodeToBytes(tx)
	_, err := b.txapi.SendRawTransaction(ctx, raw)
	return err
}
