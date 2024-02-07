package astikit

import (
	"testing"
)

func TestBitFlags(t *testing.T) {
	f := BitFlags(2 | 4)
	r := f.Add(1)
	if e, g := uint64(7), r; e != g {
		t.Fatalf("expected %d, got %d", e, g)
	}
	r = f.Del(2)
	if e, g := uint64(4), r; e != g {
		t.Fatalf("expected %d, got %d", e, g)
	}
	if e, g := false, f.Has(1); e != g {
		t.Fatalf("expected %v, got %v", e, g)
	}
	if e, g := true, f.Has(4); e != g {
		t.Fatalf("expected %v, got %v", e, g)
	}
}
