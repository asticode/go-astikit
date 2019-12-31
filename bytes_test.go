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
