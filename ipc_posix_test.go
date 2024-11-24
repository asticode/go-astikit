//go:build !windows

package astikit

import (
	"bytes"
	"testing"
)

func TestPosixSharedMemory(t *testing.T) {
	sm1, err := CreatePosixSharedMemory("/test", 8)
	if err != nil {
		t.Fatalf("expected no error, got %s", err)
	}
	defer sm1.Close()
	if sm1.Addr() == nil {
		t.Fatal("expected not nil, got nil")
	}
	if g := sm1.Size(); g <= 0 {
		t.Fatalf("expected > 0, got %d", g)
	}
	if e, g := "test", sm1.Name(); e != g {
		t.Fatalf("expected %v, got %v", e, g)
	}
	if _, err = CreatePosixSharedMemory("/test", 8); err == nil {
		t.Fatal("expected error, got nil")
	}

	b1 := []byte("test")
	if err := sm1.WriteBytes(b1); err != nil {
		t.Fatalf("expected no error, got %s", err)
	}

	sm2, err := OpenPosixSharedMemory("test")
	if err != nil {
		t.Fatalf("expected no error, got %s", err)
	}
	defer sm2.Close()
	b2, err := sm2.ReadBytes(len(b1))
	if err != nil {
		t.Fatalf("expected no error, got %s", err)
	}
	if e, g := b1, b2; !bytes.Equal(b1, b2) {
		t.Fatalf("expected %s, got %s", e, g)
	}

	if err = sm1.Close(); err != nil {
		t.Fatalf("expected no error, got %s", err)
	}
	if err = sm1.WriteBytes(b1); err == nil {
		t.Fatal("expected error, got nil")
	}
	if err = sm1.Close(); err != nil {
		t.Fatalf("expected no error, got %s", err)
	}

	if err = sm2.Close(); err != nil {
		t.Fatalf("expected no error, got %s", err)
	}
	if _, err = sm2.ReadBytes(len(b1)); err == nil {
		t.Fatal("expected error, got nil")
	}
	if err = sm2.Close(); err != nil {
		t.Fatalf("expected no error, got %s", err)
	}
}

func TestPosixVariableSizeSharedMemory(t *testing.T) {
	w := NewPosixVariableSizeSharedMemoryWriter("test-1")
	defer w.Close()
	r := NewPosixVariableSizeSharedMemoryReader()
	defer r.Close()

	b1 := []byte("test")
	ro1, err := w.WriteBytes(b1)
	if err != nil {
		t.Fatalf("expected no error, got %s", err)
	}
	if e, g := w.shm.Name(), ro1.Name; e != g {
		t.Fatalf("expected %s, got %s", e, g)
	}
	if e, g := len(b1), ro1.Size; e != g {
		t.Fatalf("expected %d, got %d", e, g)
	}
	b2, err := r.ReadBytes(ro1)
	if err != nil {
		t.Fatalf("expected no error, got %s", err)
	}
	if !bytes.Equal(b1, b2) {
		t.Fatalf("expected %s, got %s", b1, b2)
	}

	b3 := make([]byte, w.shm.Size()+1)
	b3[0] = 'a'
	b3[len(b3)-1] = 'b'
	ro2, err := w.WriteBytes(b3)
	if err != nil {
		t.Fatalf("expected no error, got %s", err)
	}
	if ro1.Name == ro2.Name {
		t.Fatal("expected different, got equalt")
	}
	b4, err := r.ReadBytes(ro2)
	if err != nil {
		t.Fatalf("expected no error, got %s", err)
	}
	if !bytes.Equal(b3, b4) {
		t.Fatalf("expected %s, got %s", b3, b4)
	}
}
