package astikit

import (
	"errors"
	"reflect"
	"testing"
)

func TestCloser(t *testing.T) {
	var o []string
	c1 := NewCloser()
	c2 := c1.NewChild()
	c1.Add(func() error {
		o = append(o, "1")
		return nil
	})
	c1.Add(func() error {
		o = append(o, "2")
		return errors.New("1")
	})
	c1.Add(func() error { return errors.New("2") })
	c2.Add(func() error {
		o = append(o, "3")
		return errors.New("3")
	})
	err := c1.Close()
	if e := []string{"2", "1", "3"}; !reflect.DeepEqual(o, e) {
		t.Errorf("expected %+v, got %+v", e, o)
	}
	if e, g := "2 && 1 && 3", err.Error(); !reflect.DeepEqual(g, e) {
		t.Errorf("expected %+v, got %+v", e, g)
	}
	c1.Add(func() error { return nil })
	if err = c1.Close(); err != nil {
		t.Errorf("expected no error, got %+v", err)
	}
}
