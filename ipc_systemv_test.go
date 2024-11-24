//go:build !windows

package astikit

import (
	"bytes"
	"testing"
)

func TestNewSystemVKey(t *testing.T) {
	_, err := NewSystemVKey(1, "testdata/ipc/invalid")
	if err == nil {
		t.Fatal("expected an error, got none")
	}
	if _, err = NewSystemVKey(1, "testdata/ipc/f"); err != nil {
		t.Fatalf("expected no error, got %s", err)
	}
}

func TestSystemVSemaphore(t *testing.T) {
	const key = 1
	s1, err := CreateSystemVSemaphore(key, IpcCreate|IpcExclusive|0666)
	if err != nil {
		t.Fatalf("expected no error, got %s", err)
	}
	defer s1.Close()
	if e, g := key, s1.Key(); e != g {
		t.Fatalf("expected %v, got %v", e, g)
	}
	if err = s1.Lock(); err != nil {
		t.Fatalf("expected no error, got %s", err)
	}
	if err = s1.Unlock(); err != nil {
		t.Fatalf("expected no error, got %s", err)
	}
	s2, err := OpenSystemVSemaphore(key)
	if err != nil {
		t.Fatalf("expected no error, got %s", err)
	}
	defer s2.Close()
	if e, g := key, s2.Key(); e != g {
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

func TestSystemVSharedMemory(t *testing.T) {
	const key = 1
	sm1, err := CreateSystemVSharedMemory(key, 10, IpcCreate|IpcExclusive|0666)
	if err != nil {
		t.Fatalf("expected no error, got %s", err)
	}
	defer sm1.Close()
	if sm1.Addr() == nil {
		t.Fatal("expected not nil, got nil")
	}
	if e, g := key, sm1.Key(); e != g {
		t.Fatalf("expected %v, got %v", e, g)
	}
	b1 := []byte("test")
	if err := sm1.WriteBytes(b1); err != nil {
		t.Fatalf("expected no error, got %s", err)
	}
	sm2, err := OpenSystemVSharedMemory(key)
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
}

func TestSystemVSemaphoredSharedMemory(t *testing.T) {
	w1 := NewSystemVSemaphoredSharedMemoryWriter()
	defer w1.Close()
	w2 := NewSystemVSemaphoredSharedMemoryWriter()
	defer w2.Close()
	r1 := NewSystemVSemaphoredSharedMemoryReader()
	defer r1.Close()
	r2 := NewSystemVSemaphoredSharedMemoryReader()
	defer r2.Close()

	b1 := []byte("test")
	semKeys := make(map[int]bool)
	shmAts := make(map[*SystemVSemaphoredSharedMemoryWriter]int64)
	shmKeys := make(map[int]bool)
	for _, v := range []struct {
		r *SystemVSemaphoredSharedMemoryReader
		w *SystemVSemaphoredSharedMemoryWriter
	}{
		{
			r: r1,
			w: w1,
		},
		{
			r: r2,
			w: w2,
		},
	} {
		ro, err := v.w.WriteBytes(b1)
		if err != nil {
			t.Fatalf("expected no error, got %s", err)
		}
		if e, g := len(b1), ro.Size; e != g {
			t.Fatalf("expected %d, got %d", e, g)
		}
		if e, g := v.w.sem.Key(), ro.SemaphoreKey; e != g {
			t.Fatalf("expected %d, got %d", e, g)
		}
		if _, ok := semKeys[ro.SemaphoreKey]; ok {
			t.Fatal("expected false, got true")
		}
		semKeys[ro.SemaphoreKey] = true
		if g := ro.SharedMemoryAt; g <= 0 {
			t.Fatalf("expected > 0, got %d", g)
		}
		shmAts[v.w] = ro.SharedMemoryAt
		if e, g := v.w.shm.Key(), ro.SharedMemoryKey; e != g {
			t.Fatalf("expected %d, got %d", e, g)
		}
		if _, ok := shmKeys[ro.SharedMemoryKey]; ok {
			t.Fatal("expected false, got true")
		}
		shmKeys[ro.SharedMemoryKey] = true

		b, err := v.r.ReadBytes(ro)
		if err != nil {
			t.Fatalf("expected no error, got %s", err)
		}
		if !bytes.Equal(b1, b) {
			t.Fatalf("expected %s, got %s", b1, b)
		}
	}

	b3 := append(b1, []byte("1")...)
	ro, err := w1.WriteBytes(b3)
	if err != nil {
		t.Fatalf("expected no error, got %s", err)
	}
	at, ok := shmAts[w1]
	if !ok {
		t.Fatal("expected false, got true")
	}
	if ne, g := at, ro.SharedMemoryAt; ne == g {
		t.Fatalf("didn't expect %d, got %d", ne, g)
	}

	b4, err := r1.ReadBytes(ro)
	if err != nil {
		t.Fatalf("expected no error, got %s", err)
	}
	if !bytes.Equal(b3, b4) {
		t.Fatalf("expected %s, got %s", b3, b4)
	}
}
