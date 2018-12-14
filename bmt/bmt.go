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
package bmt

import (
	"fmt"
	"hash"
	"io"
	"strings"
	"sync"
	"sync/atomic"
)

/*
Binary Merkle Tree Hash is a hash function over arbitrary datachunks of limited size
It is defined as the root hash of the binary merkle tree built over fixed size segments
of the underlying chunk using any base hash function (e.g keccak 256 SHA3)

It is used as the chunk hash function in swarm which in turn is the basis for the
128 branching swarm hash http://swarm-guide.readthedocs.io/en/latest/architecture.html#swarm-hash

The BMT is optimal for providing compact inclusion proofs, i.e. prove that a
segment is a substring of a chunk starting at a particular offset
The size of the underlying segments is fixed at 32 bytes (called the resolution
of the BMT hash), the EVM word size to optimize for on-chain BMT verification
as well as the hash size optimal for inclusion proofs in the merkle tree of the swarm hash.

Two implementations are provided:

* RefHasher is optimized for code simplicity and meant as a reference implementation
* Hasher is optimized for speed taking advantage of concurrency with minimalistic
  control structure to coordinate the concurrent routines
  It implements the ChunkHash interface as well as the go standard hash.Hash interface

*/

const (
//
	DefaultSegmentCount = 128 //
//
//
	DefaultPoolSize = 8
)

//
type BaseHasher func() hash.Hash

//
//
//
//
//
//
//
//
//
type Hasher struct {
	pool        *TreePool   //
	bmt         *Tree       //
	blocksize   int         //
	count       int         //
	size        int         //
	cur         int         //
	segment     []byte      //
	depth       int         //
	result      chan []byte //
	hash        []byte      //
	max         int32       //
	blockLength []byte      //
}

//
//
//
func New(p *TreePool) *Hasher {
	return &Hasher{
		pool:      p,
		depth:     depth(p.SegmentCount),
		size:      p.SegmentSize,
		blocksize: p.SegmentSize,
		count:     p.SegmentCount,
		result:    make(chan []byte),
	}
}

//
//
//
type Node struct {
	level, index int   //
	initial      bool  //
	root         bool  //
	isLeft       bool  //
	unbalanced   bool  //
	parent       *Node //
	state        int32 //
	left, right  []byte
}

//
func NewNode(level, index int, parent *Node) *Node {
	return &Node{
		parent:  parent,
		level:   level,
		index:   index,
		initial: index == 0,
		isLeft:  index%2 == 0,
	}
}

//
//
//
//
type TreePool struct {
	lock         sync.Mutex
	c            chan *Tree
	hasher       BaseHasher
	SegmentSize  int
	SegmentCount int
	Capacity     int
	count        int
}

//
//
func NewTreePool(hasher BaseHasher, segmentCount, capacity int) *TreePool {
	return &TreePool{
		c:            make(chan *Tree, capacity),
		hasher:       hasher,
		SegmentSize:  hasher().Size(),
		SegmentCount: segmentCount,
		Capacity:     capacity,
	}
}

//
func (self *TreePool) Drain(n int) {
	self.lock.Lock()
	defer self.lock.Unlock()
	for len(self.c) > n {
		<-self.c
		self.count--
	}
}

//
//
func (self *TreePool) Reserve() *Tree {
	self.lock.Lock()
	defer self.lock.Unlock()
	var t *Tree
	if self.count == self.Capacity {
		return <-self.c
	}
	select {
	case t = <-self.c:
	default:
		t = NewTree(self.hasher, self.SegmentSize, self.SegmentCount)
		self.count++
	}
	return t
}

//
//
//
func (self *TreePool) Release(t *Tree) {
	self.c <- t //
}

//
//
//
//
type Tree struct {
	leaves []*Node
}

//
func (self *Tree) Draw(hash []byte, d int) string {
	var left, right []string
	var anc []*Node
	for i, n := range self.leaves {
		left = append(left, fmt.Sprintf("%v", hashstr(n.left)))
		if i%2 == 0 {
			anc = append(anc, n.parent)
		}
		right = append(right, fmt.Sprintf("%v", hashstr(n.right)))
	}
	anc = self.leaves
	var hashes [][]string
	for l := 0; len(anc) > 0; l++ {
		var nodes []*Node
		hash := []string{""}
		for i, n := range anc {
			hash = append(hash, fmt.Sprintf("%v|%v", hashstr(n.left), hashstr(n.right)))
			if i%2 == 0 && n.parent != nil {
				nodes = append(nodes, n.parent)
			}
		}
		hash = append(hash, "")
		hashes = append(hashes, hash)
		anc = nodes
	}
	hashes = append(hashes, []string{"", fmt.Sprintf("%v", hashstr(hash)), ""})
	total := 60
	del := "                             "
	var rows []string
	for i := len(hashes) - 1; i >= 0; i-- {
		var textlen int
		hash := hashes[i]
		for _, s := range hash {
			textlen += len(s)
		}
		if total < textlen {
			total = textlen + len(hash)
		}
		delsize := (total - textlen) / (len(hash) - 1)
		if delsize > len(del) {
			delsize = len(del)
		}
		row := fmt.Sprintf("%v: %v", len(hashes)-i-1, strings.Join(hash, del[:delsize]))
		rows = append(rows, row)

	}
	rows = append(rows, strings.Join(left, "  "))
	rows = append(rows, strings.Join(right, "  "))
	return strings.Join(rows, "\n") + "\n"
}

//
//
//
//
//
//
func NewTree(hasher BaseHasher, segmentSize, segmentCount int) *Tree {
	n := NewNode(0, 0, nil)
	n.root = true
	prevlevel := []*Node{n}
//
	level := 1
	count := 2
	for d := 1; d <= depth(segmentCount); d++ {
		nodes := make([]*Node, count)
		for i := 0; i < len(nodes); i++ {
			var parent *Node
			parent = prevlevel[i/2]
			t := NewNode(level, i, parent)
			nodes[i] = t
		}
		prevlevel = nodes
		level++
		count *= 2
	}
//
	return &Tree{
		leaves: prevlevel,
	}
}

//

//
func (self *Hasher) Size() int {
	return self.size
}

//
func (self *Hasher) BlockSize() int {
	return self.blocksize
}

//
//
//
func (self *Hasher) Sum(b []byte) (r []byte) {
	t := self.bmt
	i := self.cur
	n := t.leaves[i]
	j := i
//
//
	if len(self.segment) > self.size && i > 0 && n.parent != nil {
		n = n.parent
	} else {
		i *= 2
	}
	d := self.finalise(n, i)
	self.writeSegment(j, self.segment, d)
	c := <-self.result
	self.releaseTree()

//
	if self.blockLength == nil {
		return c
	}
	res := self.pool.hasher()
	res.Reset()
	res.Write(self.blockLength)
	res.Write(c)
	return res.Sum(nil)
}

//

//
//
func (self *Hasher) Hash() []byte {
	return <-self.result
}

//

//
//
//
func (self *Hasher) Write(b []byte) (int, error) {
	l := len(b)
	if l <= 0 {
		return 0, nil
	}
	s := self.segment
	i := self.cur
	count := (self.count + 1) / 2
	need := self.count*self.size - self.cur*2*self.size
	size := self.size
	if need > size {
		size *= 2
	}
	if l < need {
		need = l
	}
//
	rest := size - len(s)
	if need < rest {
		rest = need
	}
	s = append(s, b[:rest]...)
	need -= rest
//
	for need > 0 && i < count-1 {
//
		self.writeSegment(i, s, self.depth)
		need -= size
		if need < 0 {
			size += need
		}
		s = b[rest : rest+size]
		rest += size
		i++
	}
	self.segment = s
	self.cur = i
//
	return l, nil
}

//

//
//
//
func (self *Hasher) ReadFrom(r io.Reader) (m int64, err error) {
	bufsize := self.size*self.count - self.size*self.cur - len(self.segment)
	buf := make([]byte, bufsize)
	var read int
	for {
		var n int
		n, err = r.Read(buf)
		read += n
		if err == io.EOF || read == len(buf) {
			hash := self.Sum(buf[:n])
			if read == len(buf) {
				err = NewEOC(hash)
			}
			break
		}
		if err != nil {
			break
		}
		n, err = self.Write(buf[:n])
		if err != nil {
			break
		}
	}
	return int64(read), err
}

//
func (self *Hasher) Reset() {
	self.getTree()
	self.blockLength = nil
}

//

//
//
//
func (self *Hasher) ResetWithLength(l []byte) {
	self.Reset()
	self.blockLength = l

}

//
//
func (self *Hasher) releaseTree() {
	if self.bmt != nil {
		n := self.bmt.leaves[self.cur]
		for ; n != nil; n = n.parent {
			n.unbalanced = false
			if n.parent != nil {
				n.root = false
			}
		}
		self.pool.Release(self.bmt)
		self.bmt = nil

	}
	self.cur = 0
	self.segment = nil
}

func (self *Hasher) writeSegment(i int, s []byte, d int) {
	h := self.pool.hasher()
	n := self.bmt.leaves[i]

	if len(s) > self.size && n.parent != nil {
		go func() {
			h.Reset()
			h.Write(s)
			s = h.Sum(nil)

			if n.root {
				self.result <- s
				return
			}
			self.run(n.parent, h, d, n.index, s)
		}()
		return
	}
	go self.run(n, h, d, i*2, s)
}

func (self *Hasher) run(n *Node, h hash.Hash, d int, i int, s []byte) {
	isLeft := i%2 == 0
	for {
		if isLeft {
			n.left = s
		} else {
			n.right = s
		}
		if !n.unbalanced && n.toggle() {
			return
		}
		if !n.unbalanced || !isLeft || i == 0 && d == 0 {
			h.Reset()
			h.Write(n.left)
			h.Write(n.right)
			s = h.Sum(nil)

		} else {
			s = append(n.left, n.right...)
		}

		self.hash = s
		if n.root {
			self.result <- s
			return
		}

		isLeft = n.isLeft
		n = n.parent
		i++
	}
}

//
func (self *Hasher) getTree() *Tree {
	if self.bmt != nil {
		return self.bmt
	}
	t := self.pool.Reserve()
	self.bmt = t
	return t
}

//
//
//
func (self *Node) toggle() bool {
	return atomic.AddInt32(&self.state, 1)%2 == 1
}

func hashstr(b []byte) string {
	end := len(b)
	if end > 4 {
		end = 4
	}
	return fmt.Sprintf("%x", b[:end])
}

func depth(n int) (d int) {
	for l := (n - 1) / 2; l > 0; l /= 2 {
		d++
	}
	return d
}

//
//
func (self *Hasher) finalise(n *Node, i int) (d int) {
	isLeft := i%2 == 0
	for {
//
//
//
//
		n.unbalanced = isLeft
		n.right = nil
		if n.initial {
			n.root = true
			return d
		}
		isLeft = n.isLeft
		n = n.parent
		d++
	}
}

//
type EOC struct {
	Hash []byte //
}

//
func (self *EOC) Error() string {
	return fmt.Sprintf("hasher limit reached, chunk hash: %x", self.Hash)
}

//
func NewEOC(hash []byte) *EOC {
	return &EOC{hash}
}
