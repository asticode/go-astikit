package astikit

import (
	"bytes"
	"fmt"
	"io"
	"reflect"
	"testing"
)

func TestBitsWriter(t *testing.T) {
	// TODO Need to test LittleEndian
	bw := &bytes.Buffer{}
	w := NewBitsWriter(BitsWriterOptions{Writer: bw})
	err := w.Write("000000")
	if err != nil {
		t.Errorf("expected no error, got %+v", err)
	}
	if e, g := 0, bw.Len(); e != g {
		t.Errorf("expected %d, got %d", e, g)
	}
	err = w.Write(false)
	if err != nil {
		t.Errorf("expected no error, got %+v", err)
	}
	err = w.Write(true)
	if err != nil {
		t.Errorf("expected no error, got %+v", err)
	}
	if e, g := []byte{1}, bw.Bytes(); !reflect.DeepEqual(e, g) {
		t.Errorf("expected %+v, got %+v", e, g)
	}
	err = w.Write([]byte{2, 3})
	if err != nil {
		t.Errorf("expected no error, got %+v", err)
	}
	if e, g := []byte{1, 2, 3}, bw.Bytes(); !reflect.DeepEqual(e, g) {
		t.Errorf("expected %+v, got %+v", e, g)
	}
	err = w.Write(uint8(4))
	if err != nil {
		t.Errorf("expected no error, got %+v", err)
	}
	if e, g := []byte{1, 2, 3, 4}, bw.Bytes(); !reflect.DeepEqual(e, g) {
		t.Errorf("expected %+v, got %+v", e, g)
	}
	err = w.Write(uint16(5))
	if err != nil {
		t.Errorf("expected no error, got %+v", err)
	}
	if e, g := []byte{1, 2, 3, 4, 0, 5}, bw.Bytes(); !reflect.DeepEqual(e, g) {
		t.Errorf("expected %+v, got %+v", e, g)
	}
	err = w.Write(uint32(6))
	if err != nil {
		t.Errorf("expected no error, got %+v", err)
	}
	if e, g := []byte{1, 2, 3, 4, 0, 5, 0, 0, 0, 6}, bw.Bytes(); !reflect.DeepEqual(e, g) {
		t.Errorf("expected %+v, got %+v", e, g)
	}
	err = w.Write(uint64(7))
	if err != nil {
		t.Errorf("expected no error, got %+v", err)
	}
	if e, g := []byte{1, 2, 3, 4, 0, 5, 0, 0, 0, 6, 0, 0, 0, 0, 0, 0, 0, 7}, bw.Bytes(); !reflect.DeepEqual(e, g) {
		t.Errorf("expected %+v, got %+v", e, g)
	}
	err = w.Write(1)
	if err == nil {
		t.Error("expected error")
	}
	bw.Reset()
	err = w.WriteN(uint8(4), 3)
	if err != nil {
		t.Errorf("expected no error, got %+v", err)
	}
	err = w.WriteN(uint16(4096), 13)
	if err != nil {
		t.Errorf("expected no error, got %+v", err)
	}
	if e, g := []byte{144, 0}, bw.Bytes(); !reflect.DeepEqual(e, g) {
		t.Errorf("expected %+v, got %+v", e, g)
	}
}

// testLimitedWriter is an implementation of io.Writer with max write size limit to test error handling
type testLimitedWriter struct {
	BytesLimit int
}

func (t *testLimitedWriter) Write(p []byte) (n int, err error) {
	t.BytesLimit -= len(p)
	if t.BytesLimit >= 0 {
		return len(p), nil
	}
	return len(p) + t.BytesLimit, io.EOF
}

func TestNewBitsWriterBatch(t *testing.T) {
	wr := &testLimitedWriter{BytesLimit: 1}
	w := NewBitsWriter(BitsWriterOptions{Writer: wr})
	b := NewBitsWriterBatch(w)

	b.Write(uint8(0))
	if err := b.Err(); err != nil {
		t.Errorf("expected no error, got %+v", err)
	}
	b.Write(uint8(1))
	if err := b.Err(); err == nil {
		t.Errorf("expected error, got %+v", err)
	}

	// let's check if the error is persisted
	b.Write(uint8(2))
	if err := b.Err(); err == nil {
		t.Errorf("expected error, got %+v", err)
	}
}

func BenchmarkBitsWriter_Write(b *testing.B) {
	benchmarks := []struct {
		input interface{}
	}{
		{"000000"},
		{false},
		{true},
		{[]byte{2, 3}},
		{uint8(4)},
		{uint16(5)},
		{uint32(6)},
		{uint64(7)},
	}

	bw := &bytes.Buffer{}
	bw.Grow(1024)
	w := NewBitsWriter(BitsWriterOptions{Writer: bw})

	for _, bm := range benchmarks {
		b.Run(fmt.Sprintf("%#v", bm.input), func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				bw.Reset()
				w.Write(bm.input)
			}
		})
	}
}

func BenchmarkBitsWriter_WriteN(b *testing.B) {
	type benchData struct {
		i interface{}
		n int
	}
	benchmarks := []benchData{}
	for i := 1; i <= 8; i++ {
		benchmarks = append(benchmarks, benchData{uint8(0xff), i})
	}
	for i := 1; i <= 16; i++ {
		benchmarks = append(benchmarks, benchData{uint16(0xffff), i})
	}
	for i := 1; i <= 32; i++ {
		benchmarks = append(benchmarks, benchData{uint32(0xffffffff), i})
	}
	for i := 1; i <= 64; i++ {
		benchmarks = append(benchmarks, benchData{uint64(0xffffffffffffffff), i})
	}

	bw := &bytes.Buffer{}
	bw.Grow(1024)
	w := NewBitsWriter(BitsWriterOptions{Writer: bw})

	for _, bm := range benchmarks {
		b.Run(fmt.Sprintf("%#v/%d", bm.i, bm.n), func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				bw.Reset()
				w.WriteN(bm.i, bm.n)
			}
		})
	}
}

func testByteHamming84Decode(i uint8) (o uint8, ok bool) {
	p1, d1, p2, d2, p3, d3, p4, d4 := i>>7&0x1, i>>6&0x1, i>>5&0x1, i>>4&0x1, i>>3&0x1, i>>2&0x1, i>>1&0x1, i&0x1
	testA := p1^d1^d3^d4 > 0
	testB := d1^p2^d2^d4 > 0
	testC := d1^d2^p3^d3 > 0
	testD := p1^d1^p2^d2^p3^d3^p4^d4 > 0
	if testA && testB && testC {
		// p4 may be incorrect
	} else if testD && (!testA || !testB || !testC) {
		return
	} else {
		if !testA && testB && testC {
			// p1 is incorrect
		} else if testA && !testB && testC {
			// p2 is incorrect
		} else if testA && testB && !testC {
			// p3 is incorrect
		} else if !testA && !testB && testC {
			// d4 is incorrect
			d4 ^= 1
		} else if testA && !testB && !testC {
			// d2 is incorrect
			d2 ^= 1
		} else if !testA && testB && !testC {
			// d3 is incorrect
			d3 ^= 1
		} else {
			// d1 is incorrect
			d1 ^= 1
		}
	}
	o = uint8(d4<<3 | d3<<2 | d2<<1 | d1)
	ok = true
	return
}

func TestByteHamming84Decode(t *testing.T) {
	for i := 0; i < 256; i++ {
		v, okV := ByteHamming84Decode(uint8(i))
		e, okE := testByteHamming84Decode(uint8(i))
		if !okE {
			if okV {
				t.Error("expected false, got true")
			}
		} else {
			if !okV {
				t.Error("expected true, got false")
			}
			if !reflect.DeepEqual(e, v) {
				t.Errorf("expected %+v, got %+v", e, v)
			}
		}
	}
}

func testByteParity(i uint8) bool {
	return (i&0x1)^(i>>1&0x1)^(i>>2&0x1)^(i>>3&0x1)^(i>>4&0x1)^(i>>5&0x1)^(i>>6&0x1)^(i>>7&0x1) > 0
}

func TestByteParity(t *testing.T) {
	for i := 0; i < 256; i++ {
		v, okV := ByteParity(uint8(i))
		okE := testByteParity(uint8(i))
		if !okE {
			if okV {
				t.Error("expected false, got true")
			}
		} else {
			if !okV {
				t.Error("expected true, got false")
			}
			if e := uint8(i) & 0x7f; e != v {
				t.Errorf("expected %+v, got %+v", e, v)
			}
		}
	}
}
