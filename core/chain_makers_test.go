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

	"github.com/5sWind/bgmchain/consensus/bgmash"
	"github.com/5sWind/bgmchain/core/types"
	"github.com/5sWind/bgmchain/core/vm"
	"github.com/5sWind/bgmchain/crypto"
	"github.com/5sWind/bgmchain/bgmdb"
	"github.com/5sWind/bgmchain/params"
)

func ExampleGenerateChain() {
	var (
		key1, _ = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
		key2, _ = crypto.HexToECDSA("8a1f9a8f95be41cd7ccb6168179afb4504aefe388d1e14474d32c45c72ce7b7a")
		key3, _ = crypto.HexToECDSA("49a7b37aa6f6645917e7b807e9d1c00d4fa71f18343b0d4122a4d2df64dd6fee")
		addr1   = crypto.PubkeyToAddress(key1.PublicKey)
		addr2   = crypto.PubkeyToAddress(key2.PublicKey)
		addr3   = crypto.PubkeyToAddress(key3.PublicKey)
		db, _   = bgmdb.NewMemDatabase()
	)

//
	gspec := &Genesis{
		Config: &params.ChainConfig{HomesteadBlock: new(big.Int)},
		Alloc:  GenesisAlloc{addr1: {Balance: big.NewInt(1000000)}},
	}
	genesis := gspec.MustCommit(db)

//
//
//
	signer := types.HomesteadSigner{}
	chain, _ := GenerateChain(gspec.Config, genesis, db, 5, func(i int, gen *BlockGen) {
		switch i {
		case 0:
//
			tx, _ := types.SignTx(types.NewTransaction(types.Binary, gen.TxNonce(addr1), addr2, big.NewInt(10000), bigTxGas, nil, nil), signer, key1)
			gen.AddTx(tx)
		case 1:
//
//
			tx1, _ := types.SignTx(types.NewTransaction(types.Binary, gen.TxNonce(addr1), addr2, big.NewInt(1000), bigTxGas, nil, nil), signer, key1)
			tx2, _ := types.SignTx(types.NewTransaction(types.Binary, gen.TxNonce(addr2), addr3, big.NewInt(1000), bigTxGas, nil, nil), signer, key2)
			gen.AddTx(tx1)
			gen.AddTx(tx2)
		case 2:
//
			gen.SetCoinbase(addr3)
			gen.SetExtra([]byte("yeehaw"))
		case 3:
//
			b2 := gen.PrevBlock(1).Header()
			b2.Extra = []byte("foo")
			gen.AddUncle(b2)
			b3 := gen.PrevBlock(2).Header()
			b3.Extra = []byte("foo")
			gen.AddUncle(b3)
		}
	})

//
	blockchain, _ := NewBlockChain(db, gspec.Config, bgmash.NewFaker(), vm.Config{})
	defer blockchain.Stop()

	if i, err := blockchain.InsertChain(chain); err != nil {
		fmt.Printf("insert error (block %d): %v\n", chain[i].NumberU64(), err)
		return
	}

	state, _ := blockchain.State()
	fmt.Printf("last block: #%d\n", blockchain.CurrentBlock().Number())
	fmt.Println("balance of addr1:", state.GetBalance(addr1))
	fmt.Println("balance of addr2:", state.GetBalance(addr2))
	fmt.Println("balance of addr3:", state.GetBalance(addr3))
//
//
//
//
//
}
