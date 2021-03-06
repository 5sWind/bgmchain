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
//
package light

import (
	"context"
	"math/big"

	"github.com/5sWind/bgmchain/common"
	"github.com/5sWind/bgmchain/core"
	"github.com/5sWind/bgmchain/core/types"
	"github.com/5sWind/bgmchain/bgmdb"
)

//
//
var NoOdr = context.Background()

//
type OdrBackend interface {
	Database() bgmdb.Database
	ChtIndexer() *core.ChainIndexer
	BloomTrieIndexer() *core.ChainIndexer
	BloomIndexer() *core.ChainIndexer
	Retrieve(ctx context.Context, req OdrRequest) error
}

//
type OdrRequest interface {
	StoreResult(db bgmdb.Database)
}

//
type TrieID struct {
	BlockHash, Root common.Hash
	BlockNumber     uint64
	AccKey          []byte
}

//
//
func StateTrieID(header *types.Header) *TrieID {
	return &TrieID{
		BlockHash:   header.Hash(),
		BlockNumber: header.Number.Uint64(),
		AccKey:      nil,
		Root:        header.Root,
	}
}

//
//
//
func StorageTrieID(state *TrieID, addrHash, root common.Hash) *TrieID {
	return &TrieID{
		BlockHash:   state.BlockHash,
		BlockNumber: state.BlockNumber,
		AccKey:      addrHash[:],
		Root:        root,
	}
}

//
type TrieRequest struct {
	OdrRequest
	Id    *TrieID
	Key   []byte
	Proof *NodeSet
}

//
func (req *TrieRequest) StoreResult(db bgmdb.Database) {
	req.Proof.Store(db)
}

//
type CodeRequest struct {
	OdrRequest
	Id   *TrieID //
	Hash common.Hash
	Data []byte
}

//
func (req *CodeRequest) StoreResult(db bgmdb.Database) {
	db.Put(req.Hash[:], req.Data)
}

//
type BlockRequest struct {
	OdrRequest
	Hash   common.Hash
	Number uint64
	Rlp    []byte
}

//
func (req *BlockRequest) StoreResult(db bgmdb.Database) {
	core.WriteBodyRLP(db, req.Hash, req.Number, req.Rlp)
}

//
type ReceiptsRequest struct {
	OdrRequest
	Hash     common.Hash
	Number   uint64
	Receipts types.Receipts
}

//
func (req *ReceiptsRequest) StoreResult(db bgmdb.Database) {
	core.WriteBlockReceipts(db, req.Hash, req.Number, req.Receipts)
}

//
type ChtRequest struct {
	OdrRequest
	ChtNum, BlockNum uint64
	ChtRoot          common.Hash
	Header           *types.Header
	Td               *big.Int
	Proof            *NodeSet
}

//
func (req *ChtRequest) StoreResult(db bgmdb.Database) {
//
	core.WriteHeader(db, req.Header)
	hash, num := req.Header.Hash(), req.Header.Number.Uint64()
	core.WriteTd(db, hash, num, req.Td)
	core.WriteCanonicalHash(db, hash, num)
}

//
type BloomRequest struct {
	OdrRequest
	BloomTrieNum   uint64
	BitIdx         uint
	SectionIdxList []uint64
	BloomTrieRoot  common.Hash
	BloomBits      [][]byte
	Proofs         *NodeSet
}

//
func (req *BloomRequest) StoreResult(db bgmdb.Database) {
	for i, sectionIdx := range req.SectionIdxList {
		sectionHead := core.GetCanonicalHash(db, (sectionIdx+1)*BloomTrieFrequency-1)
//
//
//
//
		core.WriteBloomBits(db, req.BitIdx, sectionIdx, sectionHead, req.BloomBits[i])
	}
}
