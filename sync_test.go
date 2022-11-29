package astikit

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestChan(t *testing.T) {
	// Do not process all
	c := NewChan(ChanOptions{})
	var o []int
	c.Add(func() {
		o = append(o, 1)
		c.Stop()
	})
	c.Add(func() {
		o = append(o, 2)
	})
	c.Start(context.Background())
	if e, g := 1, len(o); e != g {
		t.Errorf("expected %+v, got %+v", e, g)
	}

	// Process all
	c = NewChan(ChanOptions{ProcessAll: true})
	o = []int{}
	c.Add(func() {
		o = append(o, 1)
		c.Stop()
	})
	c.Add(func() {
		o = append(o, 2)
	})
	c.Start(context.Background())
	if e, g := 2, len(o); e != g {
		t.Errorf("expected %+v, got %+v", e, g)
	}

	// Default order
	c = NewChan(ChanOptions{ProcessAll: true})
	o = []int{}
	c.Add(func() {
		o = append(o, 1)
	})
	c.Add(func() {
		o = append(o, 2)
		c.Stop()
	})
	c.Start(context.Background())
	if e := []int{1, 2}; !reflect.DeepEqual(o, e) {
		t.Errorf("expected %+v, got %+v", e, o)
	}

	// FILO order
	c = NewChan(ChanOptions{
		Order:      ChanOrderFILO,
		ProcessAll: true,
	})
	o = []int{}
	c.Add(func() {
		o = append(o, 1)
	})
	c.Add(func() {
		o = append(o, 2)
		c.Stop()
	})
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
	m := &sync.Mutex{}
	var c, max int
	const n = 4
	wg := &sync.WaitGroup{}
	wg.Add(n)
	fn := func() {
		defer wg.Done()
		defer func() {
			m.Lock()
			c--
			m.Unlock()
		}()
		m.Lock()
		c++
		if c > max {
			max = c
		}
		m.Unlock()
		time.Sleep(time.Millisecond)
	}
	for idx := 0; idx < n; idx++ {
		l.Do(fn) //nolint:errcheck
	}
	wg.Wait()
	if e := 2; e != max {
		t.Errorf("expected %+v, got %+v", e, max)
	}
}

func TestEventer(t *testing.T) {
	e := NewEventer(EventerOptions{Chan: ChanOptions{ProcessAll: true}})
	var o []string
	e.On("1", func(payload interface{}) { o = append(o, payload.(string)) })
	e.On("2", func(payload interface{}) { o = append(o, payload.(string)) })
	go func() {
		time.Sleep(10 * time.Millisecond)
		e.Dispatch("1", "1.1")
		e.Dispatch("2", "2")
		e.Dispatch("1", "1.2")
		e.Stop()
	}()
	e.Start(context.Background())
	if e := []string{"1.1", "2", "1.2"}; !reflect.DeepEqual(e, o) {
		t.Errorf("expected %+v, got %+v", e, o)
	}
}

type mockedStdLogger struct {
	ss []string
}

func (l *mockedStdLogger) Fatal(v ...interface{}) { l.ss = append(l.ss, "fatal: "+fmt.Sprint(v...)) }
func (l *mockedStdLogger) Fatalf(format string, v ...interface{}) {
	l.ss = append(l.ss, "fatal: "+fmt.Sprintf(format, v...))
}
func (l *mockedStdLogger) Print(v ...interface{}) { l.ss = append(l.ss, "print: "+fmt.Sprint(v...)) }
func (l *mockedStdLogger) Printf(format string, v ...interface{}) {
	l.ss = append(l.ss, "print: "+fmt.Sprintf(format, v...))
}

func TestDebugMutex(t *testing.T) {
	l := &mockedStdLogger{}
	m := NewDebugMutex("test", l, DebugMutexWithDeadlockDetection(time.Millisecond))
	m.Lock()
	go func() {
		time.Sleep(100 * time.Millisecond)
		m.Unlock()
	}()
	m.Lock()
	if e, g := 1, len(l.ss); e != g {
		t.Errorf("expected %d, got %d", e, g)
	}
	if s, g := "sync_test.go:163", l.ss[0]; !strings.Contains(g, s) {
		t.Errorf("%s doesn't contain %s", g, s)
	}
	if s, g := "sync_test.go:168", l.ss[0]; !strings.Contains(g, s) {
		t.Errorf("%s doesn't contain %s", g, s)
	}
}
