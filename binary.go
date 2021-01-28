package astikit

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
)

// BitsWriter represents an object that can write individual bits into a writer
// in a developer-friendly way. Check out the Write method for more information.
// This is particularly helpful when you want to build a slice of bytes based
// on individual bits for testing purposes.
type BitsWriter struct {
	bo       binary.ByteOrder
	cache    byte
	cacheLen byte
	//rs []rune
	w io.Writer
}

// BitsWriterOptions represents BitsWriter options
type BitsWriterOptions struct {
	ByteOrder binary.ByteOrder
	Writer    io.Writer
}

// NewBitsWriter creates a new BitsWriter
func NewBitsWriter(o BitsWriterOptions) (w *BitsWriter) {
	w = &BitsWriter{
		bo: o.ByteOrder,
		w:  o.Writer,
	}
	if w.bo == nil {
		w.bo = binary.BigEndian
	}
	return
}

// Write writes bits into the writer. Bits are only written when there are
// enough to create a byte. When using a string or a bool, bits are added
// from left to right as if
// Available types are:
//   - string("10010"): processed as n bits, n being the length of the input
//   - []byte: processed as n bytes, n being the length of the input
//   - bool: processed as one bit
//   - uint8/uint16/uint32/uint64: processed as n bits, if type is uintn
func (w *BitsWriter) Write(i interface{}) error {
	// Transform input into "10010" format

	switch a := i.(type) {
	case string:
		for _, r := range a {
			var err error
			if r == '1' {
				err = w.writeBit(1)
			} else {
				err = w.writeBit(0)
			}
			if err != nil {
				return err
			}
		}
	case []byte:
		for _, b := range a {
			if err := w.writeFullByte(b); err != nil {
				return err
			}
		}
	case bool:
		if a {
			return w.writeBit(1)
		} else {
			return w.writeBit(0)
		}
	case uint8:
		return w.writeFullByte(a)
	case uint16:
		return w.writeFullInt(uint64(a), 2)
	case uint32:
		return w.writeFullInt(uint64(a), 4)
	case uint64:
		return w.writeFullInt(a, 8)
	default:
		return errors.New("astikit: invalid type")
	}

	return nil
}

func (w *BitsWriter) writeFullInt(in uint64, len int) error {
	if w.bo == binary.BigEndian {
		for i := len - 1; i >= 0; i-- {
			err := w.writeFullByte(byte((in >> (i * 8)) & 0xff))
			if err != nil {
				return err
			}
		}
	} else {
		for i := 0; i < len; i++ {
			err := w.writeFullByte(byte((in >> (i * 8)) & 0xff))
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (w *BitsWriter) writeFullByte(b byte) error {
	if w.cacheLen == 0 {
		_, err := w.w.Write([]byte{b})
		return err
	}

	toWrite := w.cache | (b >> w.cacheLen)
	if _, err := w.w.Write([]byte{toWrite}); err != nil {
		return err
	}

	w.cache = b << (8 - w.cacheLen)
	return nil
}

func (w *BitsWriter) writeBit(bit byte) error {
	w.cache = w.cache | (bit)<<(7-w.cacheLen)
	w.cacheLen++
	if w.cacheLen == 8 {
		_, err := w.w.Write([]byte{w.cache})
		if err != nil {
			return err
		}
		w.cacheLen = 0
		w.cache = 0
	}
	return nil
}

// WriteN writes the input into n bits
func (w *BitsWriter) WriteN(i interface{}, n int) error {
	switch i.(type) {
	case uint8, uint16, uint32, uint64:
		return w.Write(fmt.Sprintf(fmt.Sprintf("%%.%db", n), i))
	default:
		return errors.New("astikit: invalid type")
	}
}

var byteHamming84Tab = [256]uint8{
	0x01, 0xff, 0xff, 0x08, 0xff, 0x0c, 0x04, 0xff, 0xff, 0x08, 0x08, 0x08, 0x06, 0xff, 0xff, 0x08,
	0xff, 0x0a, 0x02, 0xff, 0x06, 0xff, 0xff, 0x0f, 0x06, 0xff, 0xff, 0x08, 0x06, 0x06, 0x06, 0xff,
	0xff, 0x0a, 0x04, 0xff, 0x04, 0xff, 0x04, 0x04, 0x00, 0xff, 0xff, 0x08, 0xff, 0x0d, 0x04, 0xff,
	0x0a, 0x0a, 0xff, 0x0a, 0xff, 0x0a, 0x04, 0xff, 0xff, 0x0a, 0x03, 0xff, 0x06, 0xff, 0xff, 0x0e,
	0x01, 0x01, 0x01, 0xff, 0x01, 0xff, 0xff, 0x0f, 0x01, 0xff, 0xff, 0x08, 0xff, 0x0d, 0x05, 0xff,
	0x01, 0xff, 0xff, 0x0f, 0xff, 0x0f, 0x0f, 0x0f, 0xff, 0x0b, 0x03, 0xff, 0x06, 0xff, 0xff, 0x0f,
	0x01, 0xff, 0xff, 0x09, 0xff, 0x0d, 0x04, 0xff, 0xff, 0x0d, 0x03, 0xff, 0x0d, 0x0d, 0xff, 0x0d,
	0xff, 0x0a, 0x03, 0xff, 0x07, 0xff, 0xff, 0x0f, 0x03, 0xff, 0x03, 0x03, 0xff, 0x0d, 0x03, 0xff,
	0xff, 0x0c, 0x02, 0xff, 0x0c, 0x0c, 0xff, 0x0c, 0x00, 0xff, 0xff, 0x08, 0xff, 0x0c, 0x05, 0xff,
	0x02, 0xff, 0x02, 0x02, 0xff, 0x0c, 0x02, 0xff, 0xff, 0x0b, 0x02, 0xff, 0x06, 0xff, 0xff, 0x0e,
	0x00, 0xff, 0xff, 0x09, 0xff, 0x0c, 0x04, 0xff, 0x00, 0x00, 0x00, 0xff, 0x00, 0xff, 0xff, 0x0e,
	0xff, 0x0a, 0x02, 0xff, 0x07, 0xff, 0xff, 0x0e, 0x00, 0xff, 0xff, 0x0e, 0xff, 0x0e, 0x0e, 0x0e,
	0x01, 0xff, 0xff, 0x09, 0xff, 0x0c, 0x05, 0xff, 0xff, 0x0b, 0x05, 0xff, 0x05, 0xff, 0x05, 0x05,
	0xff, 0x0b, 0x02, 0xff, 0x07, 0xff, 0xff, 0x0f, 0x0b, 0x0b, 0xff, 0x0b, 0xff, 0x0b, 0x05, 0xff,
	0xff, 0x09, 0x09, 0x09, 0x07, 0xff, 0xff, 0x09, 0x00, 0xff, 0xff, 0x09, 0xff, 0x0d, 0x05, 0xff,
	0x07, 0xff, 0xff, 0x09, 0x07, 0x07, 0x07, 0xff, 0xff, 0x0b, 0x03, 0xff, 0x07, 0xff, 0xff, 0x0e,
}

// ByteHamming84Decode hamming 8/4 decodes
func ByteHamming84Decode(i uint8) (o uint8, ok bool) {
	o = byteHamming84Tab[i]
	if o == 0xff {
		return
	}
	ok = true
	return
}

var byteParityTab = [256]uint8{
	0x00, 0x01, 0x01, 0x00, 0x01, 0x00, 0x00, 0x01, 0x01, 0x00, 0x00, 0x01, 0x00, 0x01, 0x01, 0x00,
	0x01, 0x00, 0x00, 0x01, 0x00, 0x01, 0x01, 0x00, 0x00, 0x01, 0x01, 0x00, 0x01, 0x00, 0x00, 0x01,
	0x01, 0x00, 0x00, 0x01, 0x00, 0x01, 0x01, 0x00, 0x00, 0x01, 0x01, 0x00, 0x01, 0x00, 0x00, 0x01,
	0x00, 0x01, 0x01, 0x00, 0x01, 0x00, 0x00, 0x01, 0x01, 0x00, 0x00, 0x01, 0x00, 0x01, 0x01, 0x00,
	0x01, 0x00, 0x00, 0x01, 0x00, 0x01, 0x01, 0x00, 0x00, 0x01, 0x01, 0x00, 0x01, 0x00, 0x00, 0x01,
	0x00, 0x01, 0x01, 0x00, 0x01, 0x00, 0x00, 0x01, 0x01, 0x00, 0x00, 0x01, 0x00, 0x01, 0x01, 0x00,
	0x00, 0x01, 0x01, 0x00, 0x01, 0x00, 0x00, 0x01, 0x01, 0x00, 0x00, 0x01, 0x00, 0x01, 0x01, 0x00,
	0x01, 0x00, 0x00, 0x01, 0x00, 0x01, 0x01, 0x00, 0x00, 0x01, 0x01, 0x00, 0x01, 0x00, 0x00, 0x01,
	0x01, 0x00, 0x00, 0x01, 0x00, 0x01, 0x01, 0x00, 0x00, 0x01, 0x01, 0x00, 0x01, 0x00, 0x00, 0x01,
	0x00, 0x01, 0x01, 0x00, 0x01, 0x00, 0x00, 0x01, 0x01, 0x00, 0x00, 0x01, 0x00, 0x01, 0x01, 0x00,
	0x00, 0x01, 0x01, 0x00, 0x01, 0x00, 0x00, 0x01, 0x01, 0x00, 0x00, 0x01, 0x00, 0x01, 0x01, 0x00,
	0x01, 0x00, 0x00, 0x01, 0x00, 0x01, 0x01, 0x00, 0x00, 0x01, 0x01, 0x00, 0x01, 0x00, 0x00, 0x01,
	0x00, 0x01, 0x01, 0x00, 0x01, 0x00, 0x00, 0x01, 0x01, 0x00, 0x00, 0x01, 0x00, 0x01, 0x01, 0x00,
	0x01, 0x00, 0x00, 0x01, 0x00, 0x01, 0x01, 0x00, 0x00, 0x01, 0x01, 0x00, 0x01, 0x00, 0x00, 0x01,
	0x01, 0x00, 0x00, 0x01, 0x00, 0x01, 0x01, 0x00, 0x00, 0x01, 0x01, 0x00, 0x01, 0x00, 0x00, 0x01,
	0x00, 0x01, 0x01, 0x00, 0x01, 0x00, 0x00, 0x01, 0x01, 0x00, 0x00, 0x01, 0x00, 0x01, 0x01, 0x00,
}

// ByteParity returns the byte parity
func ByteParity(i uint8) (o uint8, ok bool) {
	ok = byteParityTab[i] == 1
	o = i & 0x7f
	return
}
