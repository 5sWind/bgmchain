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
	"encoding/binary"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/5sWind/bgmchain/common"
	"github.com/5sWind/bgmchain/common/bitutil"
	"github.com/5sWind/bgmchain/core"
	"github.com/5sWind/bgmchain/core/types"
	"github.com/5sWind/bgmchain/bgmdb"
	"github.com/5sWind/bgmchain/log"
	"github.com/5sWind/bgmchain/params"
	"github.com/5sWind/bgmchain/rlp"
	"github.com/5sWind/bgmchain/trie"
)

const (
	ChtFrequency                   = 32768
	ChtV1Frequency                 = 4096 //
	HelperTrieConfirmations        = 2048 //
	HelperTrieProcessConfirmations = 256  //
)

//
//
//
type trustedCheckpoint struct {
	name                                string
	sectionIdx                          uint64
	sectionHead, chtRoot, bloomTrieRoot common.Hash
}

var (
	mainnetCheckpoint = trustedCheckpoint{
		name:          "BGM mainnet",
		sectionIdx:    129,
		sectionHead:   common.HexToHash("64100587c8ec9a76870056d07cb0f58622552d16de6253a59cac4b580c899501"),
		chtRoot:       common.HexToHash("bb4fb4076cbe6923c8a8ce8f158452bbe19564959313466989fda095a60884ca"),
		bloomTrieRoot: common.HexToHash("0db524b2c4a2a9520a42fd842b02d2e8fb58ff37c75cf57bd0eb82daeace6716"),
	}

	ropstenCheckpoint = trustedCheckpoint{
		name:          "Ropsten testnet",
		sectionIdx:    50,
		sectionHead:   common.HexToHash("00bd65923a1aa67f85e6b4ae67835784dd54be165c37f056691723c55bf016bd"),
		chtRoot:       common.HexToHash("6f56dc61936752cc1f8c84b4addabdbe6a1c19693de3f21cb818362df2117f03"),
		bloomTrieRoot: common.HexToHash("aca7d7c504d22737242effc3fdc604a762a0af9ced898036b5986c3a15220208"),
	}
)

//
var trustedCheckpoints = map[common.Hash]trustedCheckpoint{
	params.MainnetGenesisHash: mainnetCheckpoint,
}

var (
	ErrNoTrustedCht       = errors.New("No trusted canonical hash trie")
	ErrNoTrustedBloomTrie = errors.New("No trusted bloom trie")
	ErrNoHeader           = errors.New("Header not found")
	chtPrefix             = []byte("chtRoot-") //
	ChtTablePrefix        = "cht-"
)

//
type ChtNode struct {
	Hash common.Hash
	Td   *big.Int
}

//
//
func GetChtRoot(db bgmdb.Database, sectionIdx uint64, sectionHead common.Hash) common.Hash {
	var encNumber [8]byte
	binary.BigEndian.PutUint64(encNumber[:], sectionIdx)
	data, _ := db.Get(append(append(chtPrefix, encNumber[:]...), sectionHead.Bytes()...))
	return common.BytesToHash(data)
}

//
//
func GetChtV2Root(db bgmdb.Database, sectionIdx uint64, sectionHead common.Hash) common.Hash {
	return GetChtRoot(db, (sectionIdx+1)*(ChtFrequency/ChtV1Frequency)-1, sectionHead)
}

//
//
func StoreChtRoot(db bgmdb.Database, sectionIdx uint64, sectionHead, root common.Hash) {
	var encNumber [8]byte
	binary.BigEndian.PutUint64(encNumber[:], sectionIdx)
	db.Put(append(append(chtPrefix, encNumber[:]...), sectionHead.Bytes()...), root.Bytes())
}

//
type ChtIndexerBackend struct {
	db, cdb              bgmdb.Database
	section, sectionSize uint64
	lastHash             common.Hash
	trie                 *trie.Trie
}

//
func NewChtIndexer(db bgmdb.Database, clientMode bool) *core.ChainIndexer {
	cdb := bgmdb.NewTable(db, ChtTablePrefix)
	idb := bgmdb.NewTable(db, "chtIndex-")
	var sectionSize, confirmReq uint64
	if clientMode {
		sectionSize = ChtFrequency
		confirmReq = HelperTrieConfirmations
	} else {
		sectionSize = ChtV1Frequency
		confirmReq = HelperTrieProcessConfirmations
	}
	return core.NewChainIndexer(db, idb, &ChtIndexerBackend{db: db, cdb: cdb, sectionSize: sectionSize}, sectionSize, confirmReq, time.Millisecond*100, "cht")
}

//
func (c *ChtIndexerBackend) Reset(section uint64, lastSectionHead common.Hash) error {
	var root common.Hash
	if section > 0 {
		root = GetChtRoot(c.db, section-1, lastSectionHead)
	}
	var err error
	c.trie, err = trie.New(root, c.cdb)
	c.section = section
	return err
}

//
func (c *ChtIndexerBackend) Process(header *types.Header) {
	hash, num := header.Hash(), header.Number.Uint64()
	c.lastHash = hash

	td := core.GetTd(c.db, hash, num)
	if td == nil {
		panic(nil)
	}
	var encNumber [8]byte
	binary.BigEndian.PutUint64(encNumber[:], num)
	data, _ := rlp.EncodeToBytes(ChtNode{hash, td})
	c.trie.Update(encNumber[:], data)
}

//
func (c *ChtIndexerBackend) Commit() error {
	batch := c.cdb.NewBatch()
	root, err := c.trie.CommitTo(batch)
	if err != nil {
		return err
	} else {
		batch.Write()
		if ((c.section+1)*c.sectionSize)%ChtFrequency == 0 {
			log.Info("Storing CHT", "idx", c.section*c.sectionSize/ChtFrequency, "sectionHead", fmt.Sprintf("%064x", c.lastHash), "root", fmt.Sprintf("%064x", root))
		}
		StoreChtRoot(c.db, c.section, c.lastHash, root)
	}
	return nil
}

const (
	BloomTrieFrequency        = 32768
	bgmBloomBitsSection       = 4096
	bgmBloomBitsConfirmations = 256
)

var (
	bloomTriePrefix      = []byte("bltRoot-") //
	BloomTrieTablePrefix = "blt-"
)

//
func GetBloomTrieRoot(db bgmdb.Database, sectionIdx uint64, sectionHead common.Hash) common.Hash {
	var encNumber [8]byte
	binary.BigEndian.PutUint64(encNumber[:], sectionIdx)
	data, _ := db.Get(append(append(bloomTriePrefix, encNumber[:]...), sectionHead.Bytes()...))
	return common.BytesToHash(data)
}

//
func StoreBloomTrieRoot(db bgmdb.Database, sectionIdx uint64, sectionHead, root common.Hash) {
	var encNumber [8]byte
	binary.BigEndian.PutUint64(encNumber[:], sectionIdx)
	db.Put(append(append(bloomTriePrefix, encNumber[:]...), sectionHead.Bytes()...), root.Bytes())
}

//
type BloomTrieIndexerBackend struct {
	db, cdb                                    bgmdb.Database
	section, parentSectionSize, bloomTrieRatio uint64
	trie                                       *trie.Trie
	sectionHeads                               []common.Hash
}

//
func NewBloomTrieIndexer(db bgmdb.Database, clientMode bool) *core.ChainIndexer {
	cdb := bgmdb.NewTable(db, BloomTrieTablePrefix)
	idb := bgmdb.NewTable(db, "bltIndex-")
	backend := &BloomTrieIndexerBackend{db: db, cdb: cdb}
	var confirmReq uint64
	if clientMode {
		backend.parentSectionSize = BloomTrieFrequency
		confirmReq = HelperTrieConfirmations
	} else {
		backend.parentSectionSize = bgmBloomBitsSection
		confirmReq = HelperTrieProcessConfirmations
	}
	backend.bloomTrieRatio = BloomTrieFrequency / backend.parentSectionSize
	backend.sectionHeads = make([]common.Hash, backend.bloomTrieRatio)
	return core.NewChainIndexer(db, idb, backend, BloomTrieFrequency, confirmReq-bgmBloomBitsConfirmations, time.Millisecond*100, "bloomtrie")
}

//
func (b *BloomTrieIndexerBackend) Reset(section uint64, lastSectionHead common.Hash) error {
	var root common.Hash
	if section > 0 {
		root = GetBloomTrieRoot(b.db, section-1, lastSectionHead)
	}
	var err error
	b.trie, err = trie.New(root, b.cdb)
	b.section = section
	return err
}

//
func (b *BloomTrieIndexerBackend) Process(header *types.Header) {
	num := header.Number.Uint64() - b.section*BloomTrieFrequency
	if (num+1)%b.parentSectionSize == 0 {
		b.sectionHeads[num/b.parentSectionSize] = header.Hash()
	}
}

//
func (b *BloomTrieIndexerBackend) Commit() error {
	var compSize, decompSize uint64

	for i := uint(0); i < types.BloomBitLength; i++ {
		var encKey [10]byte
		binary.BigEndian.PutUint16(encKey[0:2], uint16(i))
		binary.BigEndian.PutUint64(encKey[2:10], b.section)
		var decomp []byte
		for j := uint64(0); j < b.bloomTrieRatio; j++ {
			data, err := core.GetBloomBits(b.db, i, b.section*b.bloomTrieRatio+j, b.sectionHeads[j])
			if err != nil {
				return err
			}
			decompData, err2 := bitutil.DecompressBytes(data, int(b.parentSectionSize/8))
			if err2 != nil {
				return err2
			}
			decomp = append(decomp, decompData...)
		}
		comp := bitutil.CompressBytes(decomp)

		decompSize += uint64(len(decomp))
		compSize += uint64(len(comp))
		if len(comp) > 0 {
			b.trie.Update(encKey[:], comp)
		} else {
			b.trie.Delete(encKey[:])
		}
	}

	batch := b.cdb.NewBatch()
	root, err := b.trie.CommitTo(batch)
	if err != nil {
		return err
	} else {
		batch.Write()
		sectionHead := b.sectionHeads[b.bloomTrieRatio-1]
		log.Info("Storing BloomTrie", "section", b.section, "sectionHead", fmt.Sprintf("%064x", sectionHead), "root", fmt.Sprintf("%064x", root), "compression ratio", float64(compSize)/float64(decompSize))
		StoreBloomTrieRoot(b.db, b.section, sectionHead, root)
	}

	return nil
}
