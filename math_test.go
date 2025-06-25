package astikit

import (
	"testing"
)

func TestRational(t *testing.T) {
	r := &Rational{}
	err := r.UnmarshalText([]byte(""))
	if err != nil {
		t.Fatalf("expected no error, got %+v", err)
	}
	if e, g := 0.0, r.ToFloat64(); e != g {
		t.Fatalf("expected %+v, got %+v", e, g)
	}
	err = r.UnmarshalText([]byte("test"))
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	err = r.UnmarshalText([]byte("1/test"))
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	err = r.UnmarshalText([]byte("0"))
	if err != nil {
		t.Fatalf("expected no error, got %+v", err)
	}
	if e, g := 0, r.Num(); e != g {
		t.Fatalf("expected %+v, got %+v", e, g)
	}
	if e, g := 1, r.Den(); e != g {
		t.Fatalf("expected %+v, got %+v", e, g)
	}
	err = r.UnmarshalText([]byte("1/2"))
	if err != nil {
		t.Fatalf("expected no error, got %+v", err)
	}
	if e, g := 1, r.Num(); e != g {
		t.Fatalf("expected %+v, got %+v", e, g)
	}
	if e, g := 2, r.Den(); e != g {
		t.Fatalf("expected %+v, got %+v", e, g)
	}
	if e, g := 0.5, r.ToFloat64(); e != g {
		t.Fatalf("expected %+v, got %+v", e, g)
	}
	r = NewRational(1, 2)
	b, err := r.MarshalText()
	if err != nil {
		t.Fatalf("expected no error, got %+v", err)
	}
	if e, g := "1/2", string(b); e != g {
		t.Fatalf("expected %s, got %s", e, g)
	}
}

func TestMinMaxInt(t *testing.T) {
	if e, g := 0, MinMaxInt(-1, 0, 2); e != g {
		t.Fatalf("expected %+v, got %+v", e, g)
	}
	if e, g := 1, MinMaxInt(1, 0, 2); e != g {
		t.Fatalf("expected %+v, got %+v", e, g)
	}
	if e, g := 2, MinMaxInt(3, 0, 2); e != g {
		t.Fatalf("expected %+v, got %+v", e, g)
	}
}
