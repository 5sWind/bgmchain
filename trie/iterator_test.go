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
	"math/rand"
	"testing"

	"github.com/5sWind/bgmchain/common"
	"github.com/5sWind/bgmchain/bgmdb"
)

func TestIterator(t *testing.T) {
	trie := newEmpty()
	vals := []struct{ k, v string }{
		{"do", "verb"},
		{"bgmchain", "wookiedoo"},
		{"horse", "stallion"},
		{"shaman", "horse"},
		{"doge", "coin"},
		{"dog", "puppy"},
		{"sombgmingveryoddindeedthis is", "myothernodedata"},
	}
	all := make(map[string]string)
	for _, val := range vals {
		all[val.k] = val.v
		trie.Update([]byte(val.k), []byte(val.v))
	}
	trie.Commit()

	found := make(map[string]string)
	it := NewIterator(trie.NodeIterator(nil))
	for it.Next() {
		found[string(it.Key)] = string(it.Value)
	}

	for k, v := range all {
		if found[k] != v {
			t.Errorf("iterator value mismatch for %s: got %q want %q", k, found[k], v)
		}
	}
}

type kv struct {
	k, v []byte
	t    bool
}

func TestIteratorLargeData(t *testing.T) {
	trie := newEmpty()
	vals := make(map[string]*kv)

	for i := byte(0); i < 255; i++ {
		value := &kv{common.LeftPadBytes([]byte{i}, 32), []byte{i}, false}
		value2 := &kv{common.LeftPadBytes([]byte{10, i}, 32), []byte{i}, false}
		trie.Update(value.k, value.v)
		trie.Update(value2.k, value2.v)
		vals[string(value.k)] = value
		vals[string(value2.k)] = value2
	}

	it := NewIterator(trie.NodeIterator(nil))
	for it.Next() {
		vals[string(it.Key)].t = true
	}

	var untouched []*kv
	for _, value := range vals {
		if !value.t {
			untouched = append(untouched, value)
		}
	}

	if len(untouched) > 0 {
		t.Errorf("Missed %d nodes", len(untouched))
		for _, value := range untouched {
			t.Error(value)
		}
	}
}

//
func TestNodeIteratorCoverage(t *testing.T) {
//
	db, trie, _ := makeTestTrie()

//
	hashes := make(map[common.Hash]struct{})
	for it := trie.NodeIterator(nil); it.Next(true); {
		if it.Hash() != (common.Hash{}) {
			hashes[it.Hash()] = struct{}{}
		}
	}
//
	for hash := range hashes {
		if _, err := db.Get(hash.Bytes()); err != nil {
			t.Errorf("failed to retrieve reported node %x: %v", hash, err)
		}
	}
	for _, key := range db.(*bgmdb.MemDatabase).Keys() {
		if _, ok := hashes[common.BytesToHash(key)]; !ok {
			t.Errorf("state entry not reported %x", key)
		}
	}
}

type kvs struct{ k, v string }

var testdata1 = []kvs{
	{"barb", "ba"},
	{"bard", "bc"},
	{"bars", "bb"},
	{"bar", "b"},
	{"fab", "z"},
	{"food", "ab"},
	{"foos", "aa"},
	{"foo", "a"},
}

var testdata2 = []kvs{
	{"aardvark", "c"},
	{"bar", "b"},
	{"barb", "bd"},
	{"bars", "be"},
	{"fab", "z"},
	{"foo", "a"},
	{"foos", "aa"},
	{"food", "ab"},
	{"jars", "d"},
}

func TestIteratorSeek(t *testing.T) {
	trie := newEmpty()
	for _, val := range testdata1 {
		trie.Update([]byte(val.k), []byte(val.v))
	}

//
	it := NewIterator(trie.NodeIterator([]byte("fab")))
	if err := checkIteratorOrder(testdata1[4:], it); err != nil {
		t.Fatal(err)
	}

//
	it = NewIterator(trie.NodeIterator([]byte("barc")))
	if err := checkIteratorOrder(testdata1[1:], it); err != nil {
		t.Fatal(err)
	}

//
	it = NewIterator(trie.NodeIterator([]byte("z")))
	if err := checkIteratorOrder(nil, it); err != nil {
		t.Fatal(err)
	}
}

func checkIteratorOrder(want []kvs, it *Iterator) error {
	for it.Next() {
		if len(want) == 0 {
			return fmt.Errorf("didn't expect any more values, got key %q", it.Key)
		}
		if !bytes.Equal(it.Key, []byte(want[0].k)) {
			return fmt.Errorf("wrong key: got %q, want %q", it.Key, want[0].k)
		}
		want = want[1:]
	}
	if len(want) > 0 {
		return fmt.Errorf("iterator ended early, want key %q", want[0])
	}
	return nil
}

func TestDifferenceIterator(t *testing.T) {
	triea := newEmpty()
	for _, val := range testdata1 {
		triea.Update([]byte(val.k), []byte(val.v))
	}
	triea.Commit()

	trieb := newEmpty()
	for _, val := range testdata2 {
		trieb.Update([]byte(val.k), []byte(val.v))
	}
	trieb.Commit()

	found := make(map[string]string)
	di, _ := NewDifferenceIterator(triea.NodeIterator(nil), trieb.NodeIterator(nil))
	it := NewIterator(di)
	for it.Next() {
		found[string(it.Key)] = string(it.Value)
	}

	all := []struct{ k, v string }{
		{"aardvark", "c"},
		{"barb", "bd"},
		{"bars", "be"},
		{"jars", "d"},
	}
	for _, item := range all {
		if found[item.k] != item.v {
			t.Errorf("iterator value mismatch for %s: got %v want %v", item.k, found[item.k], item.v)
		}
	}
	if len(found) != len(all) {
		t.Errorf("iterator count mismatch: got %d values, want %d", len(found), len(all))
	}
}

func TestUnionIterator(t *testing.T) {
	triea := newEmpty()
	for _, val := range testdata1 {
		triea.Update([]byte(val.k), []byte(val.v))
	}
	triea.Commit()

	trieb := newEmpty()
	for _, val := range testdata2 {
		trieb.Update([]byte(val.k), []byte(val.v))
	}
	trieb.Commit()

	di, _ := NewUnionIterator([]NodeIterator{triea.NodeIterator(nil), trieb.NodeIterator(nil)})
	it := NewIterator(di)

	all := []struct{ k, v string }{
		{"aardvark", "c"},
		{"barb", "ba"},
		{"barb", "bd"},
		{"bard", "bc"},
		{"bars", "bb"},
		{"bars", "be"},
		{"bar", "b"},
		{"fab", "z"},
		{"food", "ab"},
		{"foos", "aa"},
		{"foo", "a"},
		{"jars", "d"},
	}

	for i, kv := range all {
		if !it.Next() {
			t.Errorf("Iterator ends prematurely at element %d", i)
		}
		if kv.k != string(it.Key) {
			t.Errorf("iterator value mismatch for element %d: got key %s want %s", i, it.Key, kv.k)
		}
		if kv.v != string(it.Value) {
			t.Errorf("iterator value mismatch for element %d: got value %s want %s", i, it.Value, kv.v)
		}
	}
	if it.Next() {
		t.Errorf("Iterator returned extra values.")
	}
}

func TestIteratorNoDups(t *testing.T) {
	var tr Trie
	for _, val := range testdata1 {
		tr.Update([]byte(val.k), []byte(val.v))
	}
	checkIteratorNoDups(t, tr.NodeIterator(nil), nil)
}

//
func TestIteratorContinueAfterError(t *testing.T) {
	db, _ := bgmdb.NewMemDatabase()
	tr, _ := New(common.Hash{}, db)
	for _, val := range testdata1 {
		tr.Update([]byte(val.k), []byte(val.v))
	}
	tr.Commit()
	wantNodeCount := checkIteratorNoDups(t, tr.NodeIterator(nil), nil)
	keys := db.Keys()
	t.Log("node count", wantNodeCount)

	for i := 0; i < 20; i++ {
//
		tr, _ := New(tr.Hash(), db)

//
//
		var rkey []byte
		for {
			if rkey = keys[rand.Intn(len(keys))]; !bytes.Equal(rkey, tr.Hash().Bytes()) {
				break
			}
		}
		rval, _ := db.Get(rkey)
		db.Delete(rkey)

//
		seen := make(map[string]bool)
		it := tr.NodeIterator(nil)
		checkIteratorNoDups(t, it, seen)
		missing, ok := it.Error().(*MissingNodeError)
		if !ok || !bytes.Equal(missing.NodeHash[:], rkey) {
			t.Fatal("didn't hit missing node, got", it.Error())
		}

//
		db.Put(rkey, rval)
		checkIteratorNoDups(t, it, seen)
		if it.Error() != nil {
			t.Fatal("unexpected error", it.Error())
		}
		if len(seen) != wantNodeCount {
			t.Fatal("wrong node iteration count, got", len(seen), "want", wantNodeCount)
		}
	}
}

//
//
//
func TestIteratorContinueAfterSeekError(t *testing.T) {
//
	db, _ := bgmdb.NewMemDatabase()
	ctr, _ := New(common.Hash{}, db)
	for _, val := range testdata1 {
		ctr.Update([]byte(val.k), []byte(val.v))
	}
	root, _ := ctr.Commit()
	barNodeHash := common.HexToHash("05041990364eb72fcb1127652ce40d8bab765f2bfe53225b1170d276cc101c2e")
	barNode, _ := db.Get(barNodeHash[:])
	db.Delete(barNodeHash[:])

//
//
	tr, _ := New(root, db)
	it := tr.NodeIterator([]byte("bars"))
	missing, ok := it.Error().(*MissingNodeError)
	if !ok {
		t.Fatal("want MissingNodeError, got", it.Error())
	} else if missing.NodeHash != barNodeHash {
		t.Fatal("wrong node missing")
	}

//
	db.Put(barNodeHash[:], barNode[:])

//
	if err := checkIteratorOrder(testdata1[2:], NewIterator(it)); err != nil {
		t.Fatal(err)
	}
}

func checkIteratorNoDups(t *testing.T, it NodeIterator, seen map[string]bool) int {
	if seen == nil {
		seen = make(map[string]bool)
	}
	for it.Next(true) {
		if seen[string(it.Path())] {
			t.Fatalf("iterator visited node path %x twice", it.Path())
		}
		seen[string(it.Path())] = true
	}
	return len(seen)
}

func TestPrefixIterator(t *testing.T) {
	trie := newEmpty()
	vals := []struct{ k, v string }{
		{"do", "verb"},
		{"bgmchain", "wookiedoo"},
		{"horse", "stallion"},
		{"shaman", "horse"},
		{"doge", "coin"},
		{"dog", "puppy"},
		{"sombgmingveryoddindeedthis is", "myothernodedata"},
		{"drive", "car"},
		{"dollar", "cny"},
		{"dxracer", "chair"},
	}
	all := make(map[string]string)
	for _, val := range vals {
		all[val.k] = val.v
		trie.Update([]byte(val.k), []byte(val.v))
	}
	trie.Commit()

	expect := map[string]string{
		"doge":   "coin",
		"dog":    "puppy",
		"dollar": "cny",
		"do":     "verb",
	}
	found := make(map[string]string)
	it := NewIterator(trie.PrefixIterator([]byte("do")))
	for it.Next() {
		found[string(it.Key)] = string(it.Value)
	}

	for k, v := range found {
		if expect[k] != v {
			t.Errorf("iterator value mismatch for %s: got %v want %v", k, v, expect[k])
		}
	}
	for k, v := range expect {
		if found[k] != v {
			t.Errorf("iterator value mismatch for %s: got %v want %v", k, found[k], v)
		}
	}

	expect = map[string]string{
		"doge": "coin",
		"dog":  "puppy",
	}
	found = make(map[string]string)
	it = NewIterator(trie.PrefixIterator([]byte("dog")))
	for it.Next() {
		found[string(it.Key)] = string(it.Value)
	}

	for k, v := range found {
		if expect[k] != v {
			t.Errorf("iterator value mismatch for %s: got %v want %v", k, v, expect[k])
		}
	}
	for k, v := range expect {
		if found[k] != v {
			t.Errorf("iterator value mismatch for %s: got %v want %v", k, found[k], v)
		}
	}

	found = make(map[string]string)
	it = NewIterator(trie.PrefixIterator([]byte("test")))
	for it.Next() {
		found[string(it.Key)] = string(it.Value)
	}
	if len(found) > 0 {
		t.Errorf("iterator value count mismatch: got %v want %v", len(found), 0)
	}

	expect = map[string]string{
		"do":     "verb",
		"bgmchain":  "wookiedoo",
		"horse":  "stallion",
		"shaman": "horse",
		"doge":   "coin",
		"dog":    "puppy",
		"sombgmingveryoddindeedthis is": "myothernodedata",
		"drive":   "car",
		"dollar":  "cny",
		"dxracer": "chair",
	}
	found = make(map[string]string)
	it = NewIterator(trie.PrefixIterator(nil))
	for it.Next() {
		found[string(it.Key)] = string(it.Value)
	}
	for k, v := range found {
		if expect[k] != v {
			t.Errorf("iterator value mismatch for %s: got %v want %v", k, v, expect[k])
		}
	}
	for k, v := range expect {
		if found[k] != v {
			t.Errorf("iterator value mismatch for %s: got %v want %v", k, found[k], v)
		}
	}
}
