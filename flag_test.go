package astikit

import (
	"os"
	"testing"
)

func TestFlagCmd(t *testing.T) {
	os.Args = []string{"name"}
	if e, g := "", FlagCmd(); g != e {
		t.Errorf("expected %v, got %v", e, g)
	}
	os.Args = []string{"name", "-flag"}
	if e, g := "", FlagCmd(); g != e {
		t.Errorf("expected %v, got %v", e, g)
	}
	os.Args = []string{"name", "cmd"}
	if e, g := "cmd", FlagCmd(); g != e {
		t.Errorf("expected %v, got %v", e, g)
	}
}
