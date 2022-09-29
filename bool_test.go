package astikit

import "testing"

func TestBoolToUInt32(t *testing.T) {
	if e, g := uint32(0), BoolToUInt32(false); e != g {
		t.Errorf("expected %d, got %d", e, g)
	}
	if e, g := uint32(1), BoolToUInt32(true); e != g {
		t.Errorf("expected %d, got %d", e, g)
	}
}
