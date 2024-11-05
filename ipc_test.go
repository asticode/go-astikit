//go:build !windows

package astikit

import (
	"bytes"
	"testing"
)

func TestSystemVIpcKey(t *testing.T) {
	_, err := NewSystemVIpcKey(1, "testdata/ipc/invalid")
	if err == nil {
		t.Fatal("expected an error, got none")
	}
	if _, err = NewSystemVIpcKey(1, "testdata/ipc/f"); err != nil {
		t.Fatalf("expected no error, got %s", err)
	}
}

func TestSemaphore(t *testing.T) {
	s1, err := CreateSemaphore(1, IpcFlagCreat|IpcFlagExcl|0666)
	if err != nil {
		t.Fatalf("expected no error, got %s", err)
	}
	defer s1.Close()
	if e, g := 1, s1.Key(); e != g {
		t.Fatalf("expected %v, got %v", e, g)
	}
	if err = s1.Lock(); err != nil {
		t.Fatalf("expected no error, got %s", err)
	}
	if err = s1.Unlock(); err != nil {
		t.Fatalf("expected no error, got %s", err)
	}
	s2, err := OpenSemaphore(1)
	if err != nil {
		t.Fatalf("expected no error, got %s", err)
	}
	defer s2.Close()
	if e, g := 1, s2.Key(); e != g {
		t.Fatalf("expected %v, got %v", e, g)
	}
	if err = s2.Lock(); err != nil {
		t.Fatalf("expected no error, got %s", err)
	}
	if err = s2.Unlock(); err != nil {
		t.Fatalf("expected no error, got %s", err)
	}
	if err = s1.Close(); err != nil {
		t.Fatalf("expected no error, got %s", err)
	}
	if err = s1.Lock(); err == nil {
		t.Fatal("expected error, got nil")
	}
	if err = s1.Unlock(); err == nil {
		t.Fatal("expected error, got nil")
	}
	if err = s1.Close(); err != nil {
		t.Fatalf("expected no error, got %s", err)
	}
	if err = s2.Close(); err == nil {
		t.Fatal("expected error, got nil")
	}
	if err = s2.Lock(); err == nil {
		t.Fatal("expected error, got nil")
	}
	if err = s2.Unlock(); err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestSharedMemory(t *testing.T) {
	sm1, err := CreateSharedMemory(1, 10, IpcFlagCreat|IpcFlagExcl|0666)
	if err != nil {
		t.Fatalf("expected no error, got %s", err)
	}
	defer sm1.Close()
	if g := sm1.Pointer(); g == nil {
		t.Fatal("expected not nil, got nil")
	}
	if e, g := 1, sm1.Key(); e != g {
		t.Fatalf("expected %v, got %v", e, g)
	}
	b1 := []byte("test")
	if err := sm1.WriteBytes(b1); err != nil {
		t.Fatalf("expected no error, got %s", err)
	}
	sm2, err := OpenSharedMemory(1)
	if err != nil {
		t.Fatalf("expected no error, got %s", err)
	}
	defer sm2.Close()
	b2, err := sm2.ReadBytes(4)
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
}

func TestSemaphoredSharedMemory(t *testing.T) {
	w := NewSemaphoredSharedMemoryWriter("test")
	defer w.Close()
	r := NewSemaphoredSharedMemoryReader()
	defer r.Close()

	b1 := []byte("test")
	ro, err := w.WriteBytes(b1)
	if err != nil {
		t.Fatalf("expected no error, got %s", err)
	}
	if e, g := 4, ro.Size; e != g {
		t.Fatalf("expected %d, got %d", e, g)
	}
	if ro.SemaphoreKey == 0 {
		t.Fatalf("expected > 0, got 0")
	}
	if ro.SharedMemoryKey == 0 {
		t.Fatalf("expected > 0, got 0")
	}

	b2, err := r.ReadBytes(ro)
	if err != nil {
		t.Fatalf("expected no error, got %s", err)
	}
	if !bytes.Equal(b1, b2) {
		t.Fatalf("expected %s, got %s", b1, b2)
	}
}
