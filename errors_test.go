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
		t.Fatal("expected true, got false")
	}
	errs = NewErrors(errors.New("1"))
	if e, g := "1", errs.Error(); g != e {
		t.Fatalf("expected %+v, got %+v", e, g)
	}
	errs.Add(errors.New("2"))
	if e, g := "1 && 2", errs.Error(); g != e {
		t.Fatalf("expected %+v, got %+v", e, g)
	}
	errs.Loop(func(idx int, err error) bool {
		if e, g := strconv.Itoa(idx+1), err.Error(); g != e {
			t.Fatalf("expected %v, got %v", e, g)
		}
		return false
	})
	err1 := errors.New("1")
	err2 := errors.New("2")
	err3 := errors.New("3")
	errs = NewErrors(err1, err3)
	for _, v := range []struct {
		err      error
		expected bool
	}{
		{
			err:      err1,
			expected: true,
		},
		{
			err:      err2,
			expected: false,
		},
		{
			err:      err3,
			expected: true,
		},
	} {
		if g := errors.Is(errs, v.err); g != v.expected {
			t.Fatalf("expected %v, got %v", v.expected, g)
		}
	}
}

func TestErrorCause(t *testing.T) {
	err1 := errors.New("test 1")
	err2 := fmt.Errorf("test 2 failed: %w", err1)
	if e, g := err1, ErrorCause(err2); !errors.Is(g, e) {
		t.Fatalf("expected %+v, got %+v", e, g)
	}
	err3 := fmt.Errorf("test 3 failed: %w", err2)
	if e, g := err1, ErrorCause(err3); !errors.Is(g, e) {
		t.Fatalf("expected %+v, got %+v", e, g)
	}
}
