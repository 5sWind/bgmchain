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

package trie

import (
	"bytes"
	"fmt"

	"github.com/5sWind/bgmchain/common"
	"github.com/5sWind/bgmchain/crypto"
	"github.com/5sWind/bgmchain/log"
	"github.com/5sWind/bgmchain/rlp"
)

//
//
//
//
//
//
//
//
//
func (t *Trie) Prove(key []byte, fromLevel uint, proofDb DatabaseWriter) error {
//
	key = keybytesToHex(key)
	nodes := []node{}
	tn := t.root
	for len(key) > 0 && tn != nil {
		switch n := tn.(type) {
		case *shortNode:
			if len(key) < len(n.Key) || !bytes.Equal(n.Key, key[:len(n.Key)]) {
//
				tn = nil
			} else {
				tn = n.Val
				key = key[len(n.Key):]
			}
			nodes = append(nodes, n)
		case *fullNode:
			tn = n.Children[key[0]]
			key = key[1:]
			nodes = append(nodes, n)
		case hashNode:
			var err error
			tn, err = t.resolveHash(n, nil)
			if err != nil {
				log.Error(fmt.Sprintf("Unhandled trie error: %v", err))
				return err
			}
		default:
			panic(fmt.Sprintf("%T: invalid node: %v", tn, tn))
		}
	}
	hasher := newHasher(0, 0)
	for i, n := range nodes {
//
//
		n, _, _ = hasher.hashChildren(n, nil)
		hn, _ := hasher.store(n, nil, false)
		if hash, ok := hn.(hashNode); ok || i == 0 {
//
//
			if fromLevel > 0 {
				fromLevel--
			} else {
				enc, _ := rlp.EncodeToBytes(n)
				if !ok {
					hash = crypto.Keccak256(enc)
				}
				proofDb.Put(hash, enc)
			}
		}
	}
	return nil
}

//
//
//
//
func VerifyProof(rootHash common.Hash, key []byte, proofDb DatabaseReader) (value []byte, err error, nodes int) {
	key = keybytesToHex(key)
	wantHash := rootHash[:]
	for i := 0; ; i++ {
		buf, _ := proofDb.Get(wantHash)
		if buf == nil {
			return nil, fmt.Errorf("proof node %d (hash %064x) missing", i, wantHash[:]), i
		}
		n, err := decodeNode(wantHash, buf, 0)
		if err != nil {
			return nil, fmt.Errorf("bad proof node %d: %v", i, err), i
		}
		keyrest, cld := get(n, key)
		switch cld := cld.(type) {
		case nil:
//
			return nil, nil, i
		case hashNode:
			key = keyrest
			wantHash = cld
		case valueNode:
			return cld, nil, i + 1
		}
	}
}

func get(tn node, key []byte) ([]byte, node) {
	for {
		switch n := tn.(type) {
		case *shortNode:
			if len(key) < len(n.Key) || !bytes.Equal(n.Key, key[:len(n.Key)]) {
				return nil, nil
			}
			tn = n.Val
			key = key[len(n.Key):]
		case *fullNode:
			tn = n.Children[key[0]]
			key = key[1:]
		case hashNode:
			return key, n
		case nil:
			return key, nil
		case valueNode:
			return nil, n
		default:
			panic(fmt.Sprintf("%T: invalid node: %v", tn, tn))
		}
	}
}
