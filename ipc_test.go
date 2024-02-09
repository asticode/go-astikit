//go:build !windows

package astikit

import (
	"bytes"
	"os"
	"testing"
)

func TestSystemVIpcKey(t *testing.T) {
	_, err := NewSystemVIpcKey(1, "testdata/ipc/invalid")
	if err == nil {
		t.Fatal("expected an error, got none")
	}
	f, err := os.CreateTemp(t.TempDir(), "")
	if err != nil {
		t.Fatalf("expected no error, got %s", err)
	}
	defer f.Close()
	if _, err = NewSystemVIpcKey(1, "testdata/ipc/f"); err != nil {
		t.Fatalf("expected no error, got %s", err)
	}
}

func TestSemaphore(t *testing.T) {
	s, err := NewSemaphore(1, IpcFlagCreat|IpcFlagExcl|0666)
	if err != nil {
		t.Fatalf("expected no error, got %s", err)
	}
	defer s.Close()
	if e, g := 1, s.Key(); e != g {
		t.Fatalf("expected %v, got %v", e, g)
	}
	if err = s.Lock(); err != nil {
		t.Fatalf("expected no error, got %s", err)
	}
	if err = s.Unlock(); err != nil {
		t.Fatalf("expected no error, got %s", err)
	}
	if err = s.Close(); err != nil {
		t.Fatalf("expected no error, got %s", err)
	}
	if err = s.Lock(); err == nil {
		t.Fatal("expected error, got nil")
	}
	if err = s.Unlock(); err == nil {
		t.Fatal("expected error, got nil")
	}
	if err = s.Close(); err != nil {
		t.Fatalf("expected no error, got %s", err)
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
