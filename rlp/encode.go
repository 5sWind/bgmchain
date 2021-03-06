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

package rlp

import (
	"fmt"
	"io"
	"math/big"
	"reflect"
	"sync"
)

var (
//
//
	EmptyString = []byte{0x80}
	EmptyList   = []byte{0xC0}
)

//
//
type Encoder interface {
//
//
//
	//
//
//
//
//
	EncodeRLP(io.Writer) error
}

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
func Encode(w io.Writer, val interface{}) error {
	if outer, ok := w.(*encbuf); ok {
//
//
		return outer.encode(val)
	}
	eb := encbufPool.Get().(*encbuf)
	defer encbufPool.Put(eb)
	eb.reset()
	if err := eb.encode(val); err != nil {
		return err
	}
	return eb.toWriter(w)
}

//
//
func EncodeToBytes(val interface{}) ([]byte, error) {
	eb := encbufPool.Get().(*encbuf)
	defer encbufPool.Put(eb)
	eb.reset()
	if err := eb.encode(val); err != nil {
		return nil, err
	}
	return eb.toBytes(), nil
}

//
//
//
//
//
func EncodeToReader(val interface{}) (size int, r io.Reader, err error) {
	eb := encbufPool.Get().(*encbuf)
	eb.reset()
	if err := eb.encode(val); err != nil {
		return 0, nil, err
	}
	return eb.size(), &encReader{buf: eb}, nil
}

type encbuf struct {
	str     []byte      //
	lheads  []*listhead //
	lhsize  int         //
	sizebuf []byte      //
}

type listhead struct {
	offset int //
	size   int //
}

//
//
func (head *listhead) encode(buf []byte) []byte {
	return buf[:puthead(buf, 0xC0, 0xF7, uint64(head.size))]
}

//
//
func headsize(size uint64) int {
	if size < 56 {
		return 1
	}
	return 1 + intsize(size)
}

//
//
func puthead(buf []byte, smalltag, largetag byte, size uint64) int {
	if size < 56 {
		buf[0] = smalltag + byte(size)
		return 1
	} else {
		sizesize := putint(buf[1:], size)
		buf[0] = largetag + byte(sizesize)
		return sizesize + 1
	}
}

//
var encbufPool = sync.Pool{
	New: func() interface{} { return &encbuf{sizebuf: make([]byte, 9)} },
}

func (w *encbuf) reset() {
	w.lhsize = 0
	if w.str != nil {
		w.str = w.str[:0]
	}
	if w.lheads != nil {
		w.lheads = w.lheads[:0]
	}
}

//
func (w *encbuf) Write(b []byte) (int, error) {
	w.str = append(w.str, b...)
	return len(b), nil
}

func (w *encbuf) encode(val interface{}) error {
	rval := reflect.ValueOf(val)
	ti, err := cachedTypeInfo(rval.Type(), tags{})
	if err != nil {
		return err
	}
	return ti.writer(rval, w)
}

func (w *encbuf) encodeStringHeader(size int) {
	if size < 56 {
		w.str = append(w.str, 0x80+byte(size))
	} else {
//
		sizesize := putint(w.sizebuf[1:], uint64(size))
		w.sizebuf[0] = 0xB7 + byte(sizesize)
		w.str = append(w.str, w.sizebuf[:sizesize+1]...)
	}
}

func (w *encbuf) encodeString(b []byte) {
	if len(b) == 1 && b[0] <= 0x7F {
//
		w.str = append(w.str, b[0])
	} else {
		w.encodeStringHeader(len(b))
		w.str = append(w.str, b...)
	}
}

func (w *encbuf) list() *listhead {
	lh := &listhead{offset: len(w.str), size: w.lhsize}
	w.lheads = append(w.lheads, lh)
	return lh
}

func (w *encbuf) listEnd(lh *listhead) {
	lh.size = w.size() - lh.offset - lh.size
	if lh.size < 56 {
		w.lhsize += 1 //
	} else {
		w.lhsize += 1 + intsize(uint64(lh.size))
	}
}

func (w *encbuf) size() int {
	return len(w.str) + w.lhsize
}

func (w *encbuf) toBytes() []byte {
	out := make([]byte, w.size())
	strpos := 0
	pos := 0
	for _, head := range w.lheads {
//
		n := copy(out[pos:], w.str[strpos:head.offset])
		pos += n
		strpos += n
//
		enc := head.encode(out[pos:])
		pos += len(enc)
	}
//
	copy(out[pos:], w.str[strpos:])
	return out
}

func (w *encbuf) toWriter(out io.Writer) (err error) {
	strpos := 0
	for _, head := range w.lheads {
//
		if head.offset-strpos > 0 {
			n, err := out.Write(w.str[strpos:head.offset])
			strpos += n
			if err != nil {
				return err
			}
		}
//
		enc := head.encode(w.sizebuf)
		if _, err = out.Write(enc); err != nil {
			return err
		}
	}
	if strpos < len(w.str) {
//
		_, err = out.Write(w.str[strpos:])
	}
	return err
}

//
//
type encReader struct {
	buf    *encbuf //
	lhpos  int     //
	strpos int     //
	piece  []byte  //
}

func (r *encReader) Read(b []byte) (n int, err error) {
	for {
		if r.piece = r.next(); r.piece == nil {
//
//
//
			if r.buf != nil {
				encbufPool.Put(r.buf)
				r.buf = nil
			}
			return n, io.EOF
		}
		nn := copy(b[n:], r.piece)
		n += nn
		if nn < len(r.piece) {
//
			r.piece = r.piece[nn:]
			return n, nil
		}
		r.piece = nil
	}
}

//
//
func (r *encReader) next() []byte {
	switch {
	case r.buf == nil:
		return nil

	case r.piece != nil:
//
		return r.piece

	case r.lhpos < len(r.buf.lheads):
//
		head := r.buf.lheads[r.lhpos]
		sizebefore := head.offset - r.strpos
		if sizebefore > 0 {
//
			p := r.buf.str[r.strpos:head.offset]
			r.strpos += sizebefore
			return p
		} else {
			r.lhpos++
			return head.encode(r.buf.sizebuf)
		}

	case r.strpos < len(r.buf.str):
//
		p := r.buf.str[r.strpos:]
		r.strpos = len(r.buf.str)
		return p

	default:
		return nil
	}
}

var (
	encoderInterface = reflect.TypeOf(new(Encoder)).Elem()
	big0             = big.NewInt(0)
)

//
func makeWriter(typ reflect.Type, ts tags) (writer, error) {
	kind := typ.Kind()
	switch {
	case typ == rawValueType:
		return writeRawValue, nil
	case typ.Implements(encoderInterface):
		return writeEncoder, nil
	case kind != reflect.Ptr && reflect.PtrTo(typ).Implements(encoderInterface):
		return writeEncoderNoPtr, nil
	case kind == reflect.Interface:
		return writeInterface, nil
	case typ.AssignableTo(reflect.PtrTo(bigInt)):
		return writeBigIntPtr, nil
	case typ.AssignableTo(bigInt):
		return writeBigIntNoPtr, nil
	case isUint(kind):
		return writeUint, nil
	case kind == reflect.Bool:
		return writeBool, nil
	case kind == reflect.String:
		return writeString, nil
	case kind == reflect.Slice && isByte(typ.Elem()):
		return writeBytes, nil
	case kind == reflect.Array && isByte(typ.Elem()):
		return writeByteArray, nil
	case kind == reflect.Slice || kind == reflect.Array:
		return makeSliceWriter(typ, ts)
	case kind == reflect.Struct:
		return makeStructWriter(typ)
	case kind == reflect.Ptr:
		return makePtrWriter(typ)
	default:
		return nil, fmt.Errorf("rlp: type %v is not RLP-serializable", typ)
	}
}

func isByte(typ reflect.Type) bool {
	return typ.Kind() == reflect.Uint8 && !typ.Implements(encoderInterface)
}

func writeRawValue(val reflect.Value, w *encbuf) error {
	w.str = append(w.str, val.Bytes()...)
	return nil
}

func writeUint(val reflect.Value, w *encbuf) error {
	i := val.Uint()
	if i == 0 {
		w.str = append(w.str, 0x80)
	} else if i < 128 {
//
		w.str = append(w.str, byte(i))
	} else {
//
		s := putint(w.sizebuf[1:], i)
		w.sizebuf[0] = 0x80 + byte(s)
		w.str = append(w.str, w.sizebuf[:s+1]...)
	}
	return nil
}

func writeBool(val reflect.Value, w *encbuf) error {
	if val.Bool() {
		w.str = append(w.str, 0x01)
	} else {
		w.str = append(w.str, 0x80)
	}
	return nil
}

func writeBigIntPtr(val reflect.Value, w *encbuf) error {
	ptr := val.Interface().(*big.Int)
	if ptr == nil {
		w.str = append(w.str, 0x80)
		return nil
	}
	return writeBigInt(ptr, w)
}

func writeBigIntNoPtr(val reflect.Value, w *encbuf) error {
	i := val.Interface().(big.Int)
	return writeBigInt(&i, w)
}

func writeBigInt(i *big.Int, w *encbuf) error {
	if cmp := i.Cmp(big0); cmp == -1 {
		return fmt.Errorf("rlp: cannot encode negative *big.Int")
	} else if cmp == 0 {
		w.str = append(w.str, 0x80)
	} else {
		w.encodeString(i.Bytes())
	}
	return nil
}

func writeBytes(val reflect.Value, w *encbuf) error {
	w.encodeString(val.Bytes())
	return nil
}

func writeByteArray(val reflect.Value, w *encbuf) error {
	if !val.CanAddr() {
//
//
		copy := reflect.New(val.Type()).Elem()
		copy.Set(val)
		val = copy
	}
	size := val.Len()
	slice := val.Slice(0, size).Bytes()
	w.encodeString(slice)
	return nil
}

func writeString(val reflect.Value, w *encbuf) error {
	s := val.String()
	if len(s) == 1 && s[0] <= 0x7f {
//
		w.str = append(w.str, s[0])
	} else {
		w.encodeStringHeader(len(s))
		w.str = append(w.str, s...)
	}
	return nil
}

func writeEncoder(val reflect.Value, w *encbuf) error {
	return val.Interface().(Encoder).EncodeRLP(w)
}

//
//
func writeEncoderNoPtr(val reflect.Value, w *encbuf) error {
	if !val.CanAddr() {
//
//
//
		//
//
//
//
		return fmt.Errorf("rlp: game over: unadressable value of type %v, EncodeRLP is pointer method", val.Type())
	}
	return val.Addr().Interface().(Encoder).EncodeRLP(w)
}

func writeInterface(val reflect.Value, w *encbuf) error {
	if val.IsNil() {
//
//
//
		w.str = append(w.str, 0xC0)
		return nil
	}
	eval := val.Elem()
	ti, err := cachedTypeInfo(eval.Type(), tags{})
	if err != nil {
		return err
	}
	return ti.writer(eval, w)
}

func makeSliceWriter(typ reflect.Type, ts tags) (writer, error) {
	etypeinfo, err := cachedTypeInfo1(typ.Elem(), tags{})
	if err != nil {
		return nil, err
	}
	writer := func(val reflect.Value, w *encbuf) error {
		if !ts.tail {
			defer w.listEnd(w.list())
		}
		vlen := val.Len()
		for i := 0; i < vlen; i++ {
			if err := etypeinfo.writer(val.Index(i), w); err != nil {
				return err
			}
		}
		return nil
	}
	return writer, nil
}

func makeStructWriter(typ reflect.Type) (writer, error) {
	fields, err := structFields(typ)
	if err != nil {
		return nil, err
	}
	writer := func(val reflect.Value, w *encbuf) error {
		lh := w.list()
		for _, f := range fields {
			if err := f.info.writer(val.Field(f.index), w); err != nil {
				return err
			}
		}
		w.listEnd(lh)
		return nil
	}
	return writer, nil
}

func makePtrWriter(typ reflect.Type) (writer, error) {
	etypeinfo, err := cachedTypeInfo1(typ.Elem(), tags{})
	if err != nil {
		return nil, err
	}

//
	var nilfunc func(*encbuf) error
	kind := typ.Elem().Kind()
	switch {
	case kind == reflect.Array && isByte(typ.Elem().Elem()):
		nilfunc = func(w *encbuf) error {
			w.str = append(w.str, 0x80)
			return nil
		}
	case kind == reflect.Struct || kind == reflect.Array:
		nilfunc = func(w *encbuf) error {
//
//
			w.listEnd(w.list())
			return nil
		}
	default:
		zero := reflect.Zero(typ.Elem())
		nilfunc = func(w *encbuf) error {
			return etypeinfo.writer(zero, w)
		}
	}

	writer := func(val reflect.Value, w *encbuf) error {
		if val.IsNil() {
			return nilfunc(w)
		} else {
			return etypeinfo.writer(val.Elem(), w)
		}
	}
	return writer, err
}

//
//
func putint(b []byte, i uint64) (size int) {
	switch {
	case i < (1 << 8):
		b[0] = byte(i)
		return 1
	case i < (1 << 16):
		b[0] = byte(i >> 8)
		b[1] = byte(i)
		return 2
	case i < (1 << 24):
		b[0] = byte(i >> 16)
		b[1] = byte(i >> 8)
		b[2] = byte(i)
		return 3
	case i < (1 << 32):
		b[0] = byte(i >> 24)
		b[1] = byte(i >> 16)
		b[2] = byte(i >> 8)
		b[3] = byte(i)
		return 4
	case i < (1 << 40):
		b[0] = byte(i >> 32)
		b[1] = byte(i >> 24)
		b[2] = byte(i >> 16)
		b[3] = byte(i >> 8)
		b[4] = byte(i)
		return 5
	case i < (1 << 48):
		b[0] = byte(i >> 40)
		b[1] = byte(i >> 32)
		b[2] = byte(i >> 24)
		b[3] = byte(i >> 16)
		b[4] = byte(i >> 8)
		b[5] = byte(i)
		return 6
	case i < (1 << 56):
		b[0] = byte(i >> 48)
		b[1] = byte(i >> 40)
		b[2] = byte(i >> 32)
		b[3] = byte(i >> 24)
		b[4] = byte(i >> 16)
		b[5] = byte(i >> 8)
		b[6] = byte(i)
		return 7
	default:
		b[0] = byte(i >> 56)
		b[1] = byte(i >> 48)
		b[2] = byte(i >> 40)
		b[3] = byte(i >> 32)
		b[4] = byte(i >> 24)
		b[5] = byte(i >> 16)
		b[6] = byte(i >> 8)
		b[7] = byte(i)
		return 8
	}
}

//
func intsize(i uint64) (size int) {
	for size = 1; ; size++ {
		if i >>= 8; i == 0 {
			return size
		}
	}
}
