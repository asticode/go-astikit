package astikit

import (
	"flag"
	"os"
	"reflect"
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

func TestFlagStrings(t *testing.T) {
	f := NewFlagStrings()
	flag.Var(f, "t", "")
	flag.CommandLine.Parse([]string{"-t", "1", "-t", "2", "-t", "1"}) //nolint:errcheck
	if e := (FlagStrings{
		Map: map[string]bool{
			"1": true,
			"2": true,
		},
		Slice: &[]string{"1", "2"},
	}); !reflect.DeepEqual(e, f) {
		t.Errorf("expected %+v, got %+v", e, f)
	}
}
