package astikit

import (
	"context"
	"reflect"
	"testing"
)

func TestChan(t *testing.T) {
	// Setup
	c := NewChan(ChanOptions{})

	// Do not process all
	var o []int
	go func() {
		c.Add(func() {
			o = append(o, 1)
		})
		c.Stop()
	}()
	c.Start(context.Background())
	if len(o) != 0 {
		t.Errorf("expected %+v, got %+v", 0, len(o))
	}

	// Process all
	c = NewChan(ChanOptions{ProcessAll: true})
	o = []int{}
	go func() {
		c.Add(func() {
			o = append(o, 1)
		})
		c.Stop()
	}()
	c.Start(context.Background())
	if len(o) != 1 {
		t.Errorf("expected %+v, got %+v", 1, len(o))
	}

	// Default order
	c.Reset()
	o = []int{}
	go func() {
		c.Add(func() {
			o = append(o, 1)
		})
		c.Add(func() {
			o = append(o, 2)
		})
		c.Stop()
	}()
	c.Start(context.Background())
	if !reflect.DeepEqual(o, []int{1, 2}) {
		t.Errorf("expected %+v, got %+v", []int{1, 2}, o)
	}

	// FILO order
	c = NewChan(ChanOptions{
		Order:      ChanFILOOrder,
		ProcessAll: true,
	})
	o = []int{}
	go func() {
		c.Add(func() {
			o = append(o, 1)
		})
		c.Add(func() {
			o = append(o, 2)
		})
		c.Stop()
	}()
	c.Start(context.Background())
	if !reflect.DeepEqual(o, []int{2, 1}) {
		t.Errorf("expected %+v, got %+v", []int{2, 1}, o)
	}
}
