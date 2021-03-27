package astikit

import (
	"bytes"
	"testing"
)

func TestBytesIterator(t *testing.T) {
	i := NewBytesIterator([]byte("12345678"))
	if e, g := 8, i.Len(); e != g {
		t.Errorf("expected %v, got %v", e, g)
	}
	b, err := i.NextByte()
	if err != nil {
		t.Errorf("expected no error, got %+v", err)
	}
	if e := byte('1'); e != b {
		t.Errorf("expected %v, got %v", e, b)
	}
	bs, err := i.NextBytes(2)
	if err != nil {
		t.Errorf("expected no error, got %+v", err)
	}
	if e := []byte("23"); !bytes.Equal(e, bs) {
		t.Errorf("expected %+v, got %+v", e, bs)
	}
	i.Seek(1)
	bs, err = i.NextBytesNoCopy(2)
	if err != nil {
		t.Errorf("expected no error, got %+v", err)
	}
	if e := []byte("23"); !bytes.Equal(e, bs) {
		t.Errorf("expected %+v, got %+v", e, bs)
	}
	i.Seek(4)
	b, err = i.NextByte()
	if err != nil {
		t.Errorf("expected no error, got %+v", err)
	}
	if e := byte('5'); e != b {
		t.Errorf("expected %v, got %v", e, b)
	}
	i.Skip(1)
	b, err = i.NextByte()
	if err != nil {
		t.Errorf("expected no error, got %+v", err)
	}
	if e := byte('7'); e != b {
		t.Errorf("expected %v, got %v", e, b)
	}
	if e, g := 7, i.Offset(); e != g {
		t.Errorf("expected %v, got %v", e, g)
	}
	if !i.HasBytesLeft() {
		t.Error("expected true, got false")
	}
	bs = i.Dump()
	if e := []byte("8"); !bytes.Equal(e, bs) {
		t.Errorf("expected %+v, got %+v", e, bs)
	}
	if i.HasBytesLeft() {
		t.Error("expected false, got true")
	}
	_, err = i.NextByte()
	if err == nil {
		t.Error("expected error")
	}
	_, err = i.NextBytes(2)
	if err == nil {
		t.Error("expected error")
	}
	bs = i.Dump()
	if e, g := 0, len(bs); e != g {
		t.Errorf("expected %+v, got %+v", e, g)
	}
}
func TestBytesPad(t *testing.T) {
	if e, g := []byte("test"), BytesPad([]byte("test"), ' ', 4); !bytes.Equal(e, g) {
		t.Errorf("expected %+v, got %+v", e, g)
	}
	if e, g := []byte("testtest"), BytesPad([]byte("testtest"), ' ', 4); !bytes.Equal(e, g) {
		t.Errorf("expected %+v, got %+v", e, g)
	}
	if e, g := []byte("test"), BytesPad([]byte("testtest"), ' ', 4, PadCut); !bytes.Equal(e, g) {
		t.Errorf("expected %+v, got %+v", e, g)
	}
	if e, g := []byte("  test"), BytesPad([]byte("test"), ' ', 6); !bytes.Equal(e, g) {
		t.Errorf("expected %+v, got %+v", e, g)
	}
	if e, g := []byte("test  "), BytesPad([]byte("test"), ' ', 6, PadRight); !bytes.Equal(e, g) {
		t.Errorf("expected %+v, got %+v", e, g)
	}
	if e, g := []byte("    "), BytesPad([]byte{}, ' ', 4); !bytes.Equal(e, g) {
		t.Errorf("expected %+v, got %+v", e, g)
	}
}

func TestStrPad(t *testing.T) {
	if e, g := "test", StrPad("test", ' ', 4); e != g {
		t.Errorf("expected %+v, got %+v", e, g)
	}
	if e, g := "testtest", StrPad("testtest", ' ', 4); e != g {
		t.Errorf("expected %+v, got %+v", e, g)
	}
	if e, g := "test", StrPad("testtest", ' ', 4, PadCut); e != g {
		t.Errorf("expected %+v, got %+v", e, g)
	}
	if e, g := "  test", StrPad("test", ' ', 6); e != g {
		t.Errorf("expected %+v, got %+v", e, g)
	}
	if e, g := "test  ", StrPad("test", ' ', 6, PadRight); e != g {
		t.Errorf("expected %+v, got %+v", e, g)
	}
	if e, g := "    ", StrPad("", ' ', 4); e != g {
		t.Errorf("expected %+v, got %+v", e, g)
	}
}
