package astikit

import (
	"errors"
	"reflect"
	"testing"
)

func TestCloser(t *testing.T) {
	var c int
	var o []string
	c1 := NewCloser()
	c1.OnClosed(func(err error) { c++ })
	c2 := c1.NewChild()
	c1.Add(func() { o = append(o, "1") })
	c1.AddWithError(func() error {
		o = append(o, "2")
		return errors.New("1")
	})
	c1.AddWithError(func() error { return errors.New("2") })
	c2.AddWithError(func() error {
		o = append(o, "3")
		return errors.New("3")
	})
	err := c1.Close()
	if e := []string{"2", "1", "3"}; !reflect.DeepEqual(o, e) {
		t.Fatalf("expected %+v, got %+v", e, o)
	}
	if e, g := "2 && 1 && 3", err.Error(); !reflect.DeepEqual(g, e) {
		t.Fatalf("expected %+v, got %+v", e, g)
	}
	c1.AddWithError(func() error { return nil })
	if err = c1.Close(); err != nil {
		t.Fatalf("expected no error, got %+v", err)
	}
	if e, g := 2, c; e != g {
		t.Fatalf("expected %v, got %v", e, g)
	}
	if !c1.IsClosed() {
		t.Fatal("expected true, got false")
	}
}
