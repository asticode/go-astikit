package astikit

import "fmt"

// BytesIterator represents an object capable of iterating sequentially and safely
// through a slice of bytes. This is particularly useful when you need to iterate
// through a slice of bytes and don't want to check for "index out of range" errors
// manually.
type BytesIterator struct {
	bs     []byte
	offset int
}

// NewBytesIterator creates a new BytesIterator
func NewBytesIterator(bs []byte) *BytesIterator {
	return &BytesIterator{bs: bs}
}

// NextByte returns the next byte
func (i *BytesIterator) NextByte() (b byte, err error) {
	if len(i.bs) < i.offset+1 {
		err = fmt.Errorf("astikit: slice length is %d, offset %d is invalid", len(i.bs), i.offset)
		return
	}
	b = i.bs[i.offset]
	i.offset++
	return
}

// NextBytes returns the n next bytes
func (i *BytesIterator) NextBytes(n int) (bs []byte, err error) {
	if len(i.bs) < i.offset+n {
		err = fmt.Errorf("astikit: slice length is %d, offset %d is invalid", len(i.bs), i.offset+n)
		return
	}
	bs = make([]byte, n)
	copy(bs, i.bs[i.offset:i.offset+n])
	i.offset += n
	return
}

// Seek seeks to the nth byte
func (i *BytesIterator) Seek(n int) {
	i.offset = n
}

// Skip skips the n previous/next bytes
func (i *BytesIterator) Skip(n int) {
	i.offset += n
}

// HasBytesLeft checks whether there are bytes left
func (i *BytesIterator) HasBytesLeft() bool {
	return i.offset < len(i.bs)
}

// Offset returns the offset
func (i *BytesIterator) Offset() int {
	return i.offset
}

// Dump dumps the rest of the slice
func (i *BytesIterator) Dump() (bs []byte) {
	if !i.HasBytesLeft() {
		return
	}
	bs = make([]byte, len(i.bs)-i.offset)
	copy(bs, i.bs[i.offset:len(i.bs)])
	i.offset = len(i.bs)
	return
}

// Len returns the slice length
func (i *BytesIterator) Len() int {
	return len(i.bs)
}
