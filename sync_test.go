package astikit

import (
	"context"
	"reflect"
	"sync"
	"testing"
	"time"
)

func TestChan(t *testing.T) {
	// Do not process all
	c := NewChan(ChanOptions{})
	var o []int
	go func() {
		c.Add(func() {
			time.Sleep(100 * time.Millisecond)
			o = append(o, 1)
		})
		c.Stop()
	}()
	c.Start(context.Background())
	if e, g := 0, len(o); e != g {
		t.Errorf("expected %+v, got %+v", e, g)
	}

	// Process all
	c = NewChan(ChanOptions{ProcessAll: true})
	o = []int{}
	go func() {
		c.Add(func() {
			time.Sleep(100 * time.Millisecond)
			o = append(o, 1)
		})
		c.Stop()
	}()
	c.Start(context.Background())
	if e, g := 1, len(o); e != g {
		t.Errorf("expected %+v, got %+v", e, g)
	}

	// Default order
	c.Reset()
	o = []int{}
	go func() {
		c.Add(func() {
			o = append(o, 1)
		})
		o = append(o, 2)
		c.Add(func() {
			o = append(o, 3)
		})
		o = append(o, 4)
		c.Stop()
	}()
	c.Start(context.Background())
	if e := []int{2, 4, 1, 3}; !reflect.DeepEqual(o, e) {
		t.Errorf("expected %+v, got %+v", e, o)
	}

	// FILO order
	c = NewChan(ChanOptions{
		Order:      ChanOrderFILO,
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
	if e := []int{2, 1}; !reflect.DeepEqual(o, e) {
		t.Errorf("expected %+v, got %+v", e, o)
	}

	// Block when started
	c = NewChan(ChanOptions{AddStrategy: ChanAddStrategyBlockWhenStarted})
	o = []int{}
	go func() {
		c.Add(func() {
			o = append(o, 1)
		})
		o = append(o, 2)
		c.Add(func() {
			o = append(o, 3)
		})
		o = append(o, 4)
		c.Stop()
	}()
	c.Start(context.Background())
	if e := []int{1, 2, 3, 4}; !reflect.DeepEqual(o, e) {
		t.Errorf("expected %+v, got %+v", e, o)
	}
}

func TestGoroutineLimiter(t *testing.T) {
	l := NewGoroutineLimiter(GoroutineLimiterOptions{Max: 2})
	defer l.Close()
	var c, max int
	const n = 4
	wg := &sync.WaitGroup{}
	wg.Add(n)
	fn := func() {
		defer wg.Done()
		defer func() {
			c--
		}()
		c++
		if c > max {
			max = c
		}
		time.Sleep(100 * time.Millisecond)
	}
	for idx := 0; idx < n; idx++ {
		l.Do(fn)
	}
	wg.Wait()
	if e := 2; e != max {
		t.Errorf("expected %+v, got %+v", e, max)
	}
}
