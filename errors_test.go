package astikit

import (
	"errors"
	"fmt"
	"strconv"
	"testing"
)

func TestErrors(t *testing.T) {
	errs := NewErrors()
	if !errs.IsNil() {
		t.Error("expected true, got false")
	}
	errs = NewErrors(errors.New("1"))
	if e, g := "1", errs.Error(); g != e {
		t.Errorf("expected %+v, got %+v", e, g)
	}
	errs.Add(errors.New("2"))
	if e, g := "1 && 2", errs.Error(); g != e {
		t.Errorf("expected %+v, got %+v", e, g)
	}
	errs.Loop(func(idx int, err error) bool {
		if e, g := strconv.Itoa(idx+1), err.Error(); g != e {
			t.Errorf("expected %v, got %v", e, g)
		}
		return false
	})
}

func TestErrorCause(t *testing.T) {
	err1 := errors.New("test 1")
	err2 := fmt.Errorf("test 2 failed: %w", err1)
	if e, g := err1, ErrorCause(err2); !errors.Is(g, e) {
		t.Errorf("expected %+v, got %+v", e, g)
	}
	err3 := fmt.Errorf("test 3 failed: %w", err2)
	if e, g := err1, ErrorCause(err3); !errors.Is(g, e) {
		t.Errorf("expected %+v, got %+v", e, g)
	}
}
