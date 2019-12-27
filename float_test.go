package astikit

import (
	"testing"
)

func TestRational(t *testing.T) {
	r := &Rational{}
	err := r.UnmarshalText([]byte(""))
	if err != nil {
		t.Errorf("expected no error, got %+v", err)
	}
	if e, g := 0.0, r.ToFloat64(); e != g {
		t.Errorf("expected %+v, got %+v", e, g)
	}
	err = r.UnmarshalText([]byte("test"))
	if err == nil {
		t.Error("expected error, got nil")
	}
	err = r.UnmarshalText([]byte("1/test"))
	if err == nil {
		t.Error("expected error, got nil")
	}
	err = r.UnmarshalText([]byte("0"))
	if err != nil {
		t.Errorf("expected no error, got %+v", err)
	}
	if e, g := 0, r.Num(); e != g {
		t.Errorf("expected %+v, got %+v", e, g)
	}
	if e, g := 1, r.Den(); e != g {
		t.Errorf("expected %+v, got %+v", e, g)
	}
	err = r.UnmarshalText([]byte("1/2"))
	if err != nil {
		t.Errorf("expected no error, got %+v", err)
	}
	if e, g := 1, r.Num(); e != g {
		t.Errorf("expected %+v, got %+v", e, g)
	}
	if e, g := 2, r.Den(); e != g {
		t.Errorf("expected %+v, got %+v", e, g)
	}
	if e, g := 0.5, r.ToFloat64(); e != g {
		t.Errorf("expected %+v, got %+v", e, g)
	}
}
