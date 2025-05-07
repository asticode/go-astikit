package astikit

import (
	"context"
	"errors"
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
		t.Fatalf("expected %+v, got %+v", e, g)
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
		t.Fatalf("expected %+v, got %+v", e, g)
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
		t.Fatalf("expected %+v, got %+v", e, o)
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
		t.Fatalf("expected %+v, got %+v", e, o)
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
		t.Fatalf("expected %+v, got %+v", e, o)
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
		t.Fatalf("expected %+v, got %+v", e, max)
	}
}

func TestEventer(t *testing.T) {
	e := NewEventer(EventerOptions{Chan: ChanOptions{ProcessAll: true}})
	var o []string
	e.On("1", func(payload any) { o = append(o, payload.(string)) })
	e.On("2", func(payload any) { o = append(o, payload.(string)) })
	go func() {
		time.Sleep(10 * time.Millisecond)
		e.Dispatch("1", "1.1")
		e.Dispatch("2", "2")
		e.Dispatch("1", "1.2")
		e.Stop()
	}()
	e.Start(context.Background())
	if e := []string{"1.1", "2", "1.2"}; !reflect.DeepEqual(e, o) {
		t.Fatalf("expected %+v, got %+v", e, o)
	}
}

type mockedStdLogger struct {
	m  sync.Mutex
	ss []string
}

func (l *mockedStdLogger) Fatal(v ...any) {
	l.m.Lock()
	defer l.m.Unlock()
	l.ss = append(l.ss, "fatal: "+fmt.Sprint(v...))
}
func (l *mockedStdLogger) Fatalf(format string, v ...any) {
	l.m.Lock()
	defer l.m.Unlock()
	l.ss = append(l.ss, "fatal: "+fmt.Sprintf(format, v...))
}
func (l *mockedStdLogger) Print(v ...any) {
	l.m.Lock()
	defer l.m.Unlock()
	l.ss = append(l.ss, "print: "+fmt.Sprint(v...))
}
func (l *mockedStdLogger) Printf(format string, v ...any) {
	l.m.Lock()
	defer l.m.Unlock()
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
	l.m.Lock()
	ss := l.ss
	l.m.Unlock()
	if e, g := 1, len(ss); e != g {
		t.Fatalf("expected %d, got %d", e, g)
	}
	if s, g := "sync_test.go:177", ss[0]; !strings.Contains(g, s) {
		t.Fatalf("%s doesn't contain %s", g, s)
	}
	if s, g := "sync_test.go:182", ss[0]; !strings.Contains(g, s) {
		t.Fatalf("%s doesn't contain %s", g, s)
	}
}

func TestFIFOMutex(t *testing.T) {
	m := FIFOMutex{}
	var r []int
	m.Lock()
	wg := sync.WaitGroup{}
	testFIFOMutex(1, &m, &r, &wg)
	m.Unlock()
	wg.Wait()
	if e, g := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}, r; !reflect.DeepEqual(e, g) {
		t.Fatalf("expected %v, got %v", e, g)
	}
}

func testFIFOMutex(i int, m *FIFOMutex, r *[]int, wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		defer wg.Done()
		if i < 10 {
			testFIFOMutex(i+1, m, r, wg)
		}
		m.Lock()
		*r = append(*r, i)
		m.Unlock()
	}()
}

func TestBufferedBatcher(t *testing.T) {
	var count int
	var batches []map[any]int
	var bb1 *BufferedBatcher
	ctx1, cancel1 := context.WithCancel(context.Background())
	defer cancel1()
	bb1 = NewBufferedBatcher(BufferedBatcherOptions{OnBatch: func(ctx context.Context, batch []any) {
		count++
		if len(batch) > 0 {
			m := make(map[any]int)
			for _, i := range batch {
				m[i]++
			}
			batches = append(batches, m)
		}
		switch count {
		case 1:
			bb1.Add(1)
			bb1.Add(1)
			bb1.Add(2)
		case 2:
			bb1.Add(2)
			bb1.Add(2)
			bb1.Add(3)
		case 3:
			bb1.Add(1)
			bb1.Add(1)
			bb1.Add(2)
			bb1.Add(2)
			bb1.Add(3)
			bb1.Add(3)
		case 4:
			go func() {
				time.Sleep(100 * time.Millisecond)
				bb1.Add(1)
			}()
		case 5:
			cancel1()
		}
	}})
	bb1.Add(1)
	ctx2, cancel2 := context.WithTimeout(context.Background(), time.Second)
	defer cancel2()
	go func() {
		defer cancel2()
		bb1.Start(ctx1)
	}()
	<-ctx2.Done()
	if errors.Is(ctx2.Err(), context.DeadlineExceeded) {
		t.Fatal("expected nothing, got timeout")
	}
	if e, g := []map[any]int{
		{1: 1},
		{1: 1, 2: 1},
		{2: 1, 3: 1},
		{1: 1, 2: 1, 3: 1},
		{1: 1},
	}, batches; !reflect.DeepEqual(e, g) {
		t.Fatalf("expected %+v, got %+v", e, g)
	}

	var bb2 *BufferedBatcher
	bb2 = NewBufferedBatcher(BufferedBatcherOptions{OnBatch: func(ctx context.Context, batch []any) {
		bb2.Start(context.Background())
		bb2.Stop()
		bb2.Stop()
	}})
	bb2.Add(1)
	ctx3, cancel3 := context.WithTimeout(context.Background(), time.Second)
	defer cancel3()
	go func() {
		defer cancel3()
		bb2.Start(context.Background())
	}()
	<-ctx3.Done()
	if errors.Is(ctx3.Err(), context.DeadlineExceeded) {
		t.Fatal("expected nothing, got timeout")
	}
}
