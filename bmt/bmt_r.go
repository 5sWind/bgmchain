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
//
//
//
//
//
package bmt

import (
	"hash"
)

//
type RefHasher struct {
	span    int
	section int
	cap     int
	h       hash.Hash
}

//
func NewRefHasher(hasher BaseHasher, count int) *RefHasher {
	h := hasher()
	hashsize := h.Size()
	maxsize := hashsize * count
	c := 2
	for ; c < count; c *= 2 {
	}
	if c > 2 {
		c /= 2
	}
	return &RefHasher{
		section: 2 * hashsize,
		span:    c * hashsize,
		cap:     maxsize,
		h:       h,
	}
}

//
//
func (rh *RefHasher) Hash(d []byte) []byte {
	if len(d) > rh.cap {
		d = d[:rh.cap]
	}

	return rh.hash(d, rh.span)
}

func (rh *RefHasher) hash(d []byte, s int) []byte {
	l := len(d)
	left := d
	var right []byte
	if l > rh.section {
		for ; s >= l; s /= 2 {
		}
		left = rh.hash(d[:s], s)
		right = d[s:]
		if l-s > rh.section/2 {
			right = rh.hash(right, s)
		}
	}
	defer rh.h.Reset()
	rh.h.Write(left)
	rh.h.Write(right)
	h := rh.h.Sum(nil)
	return h
}
