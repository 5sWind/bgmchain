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
	"testing"

	"github.com/5sWind/bgmchain/consensus/bgmash"
	"github.com/5sWind/bgmchain/core/vm"
	"github.com/5sWind/bgmchain/bgmdb"
	"github.com/5sWind/bgmchain/params"
)

//
//
func TestDAOForkRangeExtradata(t *testing.T) {
	forkBlock := big.NewInt(32)

//
	db, _ := bgmdb.NewMemDatabase()
	gspec := new(Genesis)
	genesis := gspec.MustCommit(db)
	prefix, _ := GenerateChain(params.TestChainConfig, genesis, db, int(forkBlock.Int64()-1), func(i int, gen *BlockGen) {})

//
	proDb, _ := bgmdb.NewMemDatabase()
	gspec.MustCommit(proDb)

	proConf := *params.TestChainConfig
	proConf.DAOForkBlock = forkBlock
	proConf.DAOForkSupport = true

	proBc, _ := NewBlockChain(proDb, &proConf, bgmash.NewFaker(), vm.Config{})
	defer proBc.Stop()

	conDb, _ := bgmdb.NewMemDatabase()
	gspec.MustCommit(conDb)

	conConf := *params.TestChainConfig
	conConf.DAOForkBlock = forkBlock
	conConf.DAOForkSupport = false

	conBc, _ := NewBlockChain(conDb, &conConf, bgmash.NewFaker(), vm.Config{})
	defer conBc.Stop()

	if _, err := proBc.InsertChain(prefix); err != nil {
		t.Fatalf("pro-fork: failed to import chain prefix: %v", err)
	}
	if _, err := conBc.InsertChain(prefix); err != nil {
		t.Fatalf("con-fork: failed to import chain prefix: %v", err)
	}
//
	for i := int64(0); i < params.DAOForkExtraRange.Int64(); i++ {
//
		db, _ = bgmdb.NewMemDatabase()
		gspec.MustCommit(db)
		bc, _ := NewBlockChain(db, &conConf, bgmash.NewFaker(), vm.Config{})
		defer bc.Stop()

		blocks := conBc.GetBlocksFromHash(conBc.CurrentBlock().Hash(), int(conBc.CurrentBlock().NumberU64()))
		for j := 0; j < len(blocks)/2; j++ {
			blocks[j], blocks[len(blocks)-1-j] = blocks[len(blocks)-1-j], blocks[j]
		}
		if _, err := bc.InsertChain(blocks); err != nil {
			t.Fatalf("failed to import contra-fork chain for expansion: %v", err)
		}
		blocks, _ = GenerateChain(&proConf, conBc.CurrentBlock(), db, 1, func(i int, gen *BlockGen) {})
		if _, err := conBc.InsertChain(blocks); err == nil {
			t.Fatalf("contra-fork chain accepted pro-fork block: %v", blocks[0])
		}
//
		blocks, _ = GenerateChain(&conConf, conBc.CurrentBlock(), db, 1, func(i int, gen *BlockGen) {})
		if _, err := conBc.InsertChain(blocks); err != nil {
			t.Fatalf("contra-fork chain didn't accepted no-fork block: %v", err)
		}
//
		db, _ = bgmdb.NewMemDatabase()
		gspec.MustCommit(db)
		bc, _ = NewBlockChain(db, &proConf, bgmash.NewFaker(), vm.Config{})
		defer bc.Stop()

		blocks = proBc.GetBlocksFromHash(proBc.CurrentBlock().Hash(), int(proBc.CurrentBlock().NumberU64()))
		for j := 0; j < len(blocks)/2; j++ {
			blocks[j], blocks[len(blocks)-1-j] = blocks[len(blocks)-1-j], blocks[j]
		}
		if _, err := bc.InsertChain(blocks); err != nil {
			t.Fatalf("failed to import pro-fork chain for expansion: %v", err)
		}
		blocks, _ = GenerateChain(&conConf, proBc.CurrentBlock(), db, 1, func(i int, gen *BlockGen) {})
		if _, err := proBc.InsertChain(blocks); err == nil {
			t.Fatalf("pro-fork chain accepted contra-fork block: %v", blocks[0])
		}
//
		blocks, _ = GenerateChain(&proConf, proBc.CurrentBlock(), db, 1, func(i int, gen *BlockGen) {})
		if _, err := proBc.InsertChain(blocks); err != nil {
			t.Fatalf("pro-fork chain didn't accepted pro-fork block: %v", err)
		}
	}
//
	db, _ = bgmdb.NewMemDatabase()
	gspec.MustCommit(db)
	bc, _ := NewBlockChain(db, &conConf, bgmash.NewFaker(), vm.Config{})
	defer bc.Stop()

	blocks := conBc.GetBlocksFromHash(conBc.CurrentBlock().Hash(), int(conBc.CurrentBlock().NumberU64()))
	for j := 0; j < len(blocks)/2; j++ {
		blocks[j], blocks[len(blocks)-1-j] = blocks[len(blocks)-1-j], blocks[j]
	}
	if _, err := bc.InsertChain(blocks); err != nil {
		t.Fatalf("failed to import contra-fork chain for expansion: %v", err)
	}
	blocks, _ = GenerateChain(&proConf, conBc.CurrentBlock(), db, 1, func(i int, gen *BlockGen) {})
	if _, err := conBc.InsertChain(blocks); err != nil {
		t.Fatalf("contra-fork chain didn't accept pro-fork block post-fork: %v", err)
	}
//
	db, _ = bgmdb.NewMemDatabase()
	gspec.MustCommit(db)
	bc, _ = NewBlockChain(db, &proConf, bgmash.NewFaker(), vm.Config{})
	defer bc.Stop()

	blocks = proBc.GetBlocksFromHash(proBc.CurrentBlock().Hash(), int(proBc.CurrentBlock().NumberU64()))
	for j := 0; j < len(blocks)/2; j++ {
		blocks[j], blocks[len(blocks)-1-j] = blocks[len(blocks)-1-j], blocks[j]
	}
	if _, err := bc.InsertChain(blocks); err != nil {
		t.Fatalf("failed to import pro-fork chain for expansion: %v", err)
	}
	blocks, _ = GenerateChain(&conConf, proBc.CurrentBlock(), db, 1, func(i int, gen *BlockGen) {})
	if _, err := proBc.InsertChain(blocks); err != nil {
		t.Fatalf("pro-fork chain didn't accept contra-fork block post-fork: %v", err)
	}
}
