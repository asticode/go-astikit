package astikit

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
)

// BitsWriter represents an object that can write individual bits into a writer
// in a developer-friendly way. Check out the Write method for more information.
// This is particularly helpful when you want to build a slice of bytes based
// on individual bits for testing purposes.
type BitsWriter struct {
	bo binary.ByteOrder
	rs []rune
	w  io.Writer
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
func (w *BitsWriter) Write(i interface{}) (err error) {
	// Transform input into "10010" format
	var s string
	switch a := i.(type) {
	case string:
		s = a
	case []byte:
		for _, b := range a {
			s += fmt.Sprintf("%.8b", b)
		}
	case bool:
		if a {
			s = "1"
		} else {
			s = "0"
		}
	case uint8:
		s = fmt.Sprintf("%.8b", i)
	case uint16:
		s = fmt.Sprintf("%.16b", i)
	case uint32:
		s = fmt.Sprintf("%.32b", i)
	case uint64:
		s = fmt.Sprintf("%.64b", i)
	default:
		err = errors.New("astikit: invalid type")
		return
	}

	// Loop through runes
	for _, r := range s {
		// Append rune
		if w.bo == binary.LittleEndian {
			w.rs = append([]rune{r}, w.rs...)
		} else {
			w.rs = append(w.rs, r)
		}

		// There are enough bits to form a byte
		if len(w.rs) == 8 {
			// Get value
			v := w.val()

			// Remove runes
			w.rs = w.rs[8:]

			// Write
			if err = binary.Write(w.w, w.bo, v); err != nil {
				return
			}
		}
	}
	return
}

func (w *BitsWriter) val() (v uint8) {
	var power float64
	for idx := len(w.rs) - 1; idx >= 0; idx-- {
		if w.rs[idx] == '1' {
			v = v + uint8(math.Pow(2, power))
		}
		power++
	}
	return
}

// WriteN writes the input into n bits
func (w *BitsWriter) WriteN(i interface{}, n int) error {
	switch i.(type) {
	case uint8, uint16, uint32, uint64:
		fmt.Println(fmt.Sprintf(fmt.Sprintf("%%.%db", n), i))
		return w.Write(fmt.Sprintf(fmt.Sprintf("%%.%db", n), i))
	default:
		return errors.New("astikit: invalid type")
	}
}
