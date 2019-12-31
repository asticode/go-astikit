package astikit

import (
	"bytes"
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
	err = w.WriteN(uint8(8), 3)
	if err != nil {
		t.Errorf("expected no error, got %+v", err)
	}
	err = w.WriteN(uint16(4096), 13)
	if err != nil {
		t.Errorf("expected no error, got %+v", err)
	}
	if e, g := []byte{136, 0}, bw.Bytes(); !reflect.DeepEqual(e, g) {
		t.Errorf("expected %+v, got %+v", e, g)
	}
}
