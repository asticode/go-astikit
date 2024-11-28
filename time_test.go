package astikit

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"sync"
	"testing"
	"time"
)

func TestSleep(t *testing.T) {
	var ctx, cancel = context.WithCancel(context.Background())
	var err error
	var wg = &sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		err = Sleep(ctx, time.Minute)
	}()
	cancel()
	wg.Wait()
	if e, g := context.Canceled, err; !errors.Is(g, e) {
		t.Fatalf("err should be %s, got %s", e, g)
	}
}

func TestTimestamp(t *testing.T) {
	const j = `{"value":1495290215}`
	v := struct {
		Value Timestamp `json:"value"`
	}{}
	err := json.Unmarshal([]byte(j), &v)
	if err != nil {
		t.Fatalf("err should be nil, got %s", err)
	}
	if e, g := int64(1495290215), v.Value.Unix(); g != e {
		t.Fatalf("timestamp should be %v, got %v", e, g)
	}
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("err should be nil, got %s", err)
	}
	if string(b) != j {
		t.Fatalf("json should be %s, got %s", j, b)
	}
}

func isAbsoluteTime(t time.Time) bool {
	return t.Year() >= time.Now().Year()-1
}

func TestNow(t *testing.T) {
	if g := Now(); !isAbsoluteTime(Now()) {
		t.Fatalf("expected %s to be an absolute time", g)
	}
	var count int64
	m := MockNow(func() time.Time {
		count++
		return time.Unix(count, 0)
	})
	if e, g := time.Unix(1, 0), Now(); !reflect.DeepEqual(e, g) {
		t.Fatalf("expected %s, got %s", e, g)
	}
	m.Close()
	if g := Now(); !isAbsoluteTime(Now()) {
		t.Fatalf("expected %s to be an absolute time", g)
	}
}

func TestTimestampNano(t *testing.T) {
	const j = `{"value":1732636645443709000}`
	v := struct {
		Value TimestampNano `json:"value"`
	}{}
	err := json.Unmarshal([]byte(j), &v)
	if err != nil {
		t.Fatalf("err should be nil, got %s", err)
	}
	if e, g := int64(1732636645443709000), v.Value.UnixNano(); g != e {
		t.Fatalf("timestamp should be %v, got %v", e, g)
	}
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("err should be nil, got %s", err)
	}
	if string(b) != j {
		t.Fatalf("json should be %s, got %s", j, b)
	}
}

func TestStopwatch(t *testing.T) {
	var count int64
	defer MockNow(func() time.Time {
		count++
		return time.Unix(count, 0)
	}).Close()

	s1 := NewStopwatch()
	s2 := s1.NewChild("1")
	if e, g := 2*time.Second, s1.Duration(); e != g {
		t.Fatalf("expected %s, got %s", e, g)
	}
	s2.Done()
	s1.NewChild("2")
	s3 := s1.NewChild("3")
	s3.NewChild("3-1")
	s4 := s3.NewChild("3-2")
	s4.NewChild("3-2-1")
	s5 := NewStopwatch()
	s5.NewChild("3-2-2")
	s6 := s5.NewChild("3-2-3")
	s6.NewChild("3-2-3-1")
	s5.Done()
	s4.Merge(s5)
	s3.NewChild("3-3")
	s1.NewChild("4")
	s1.Done()
	if e, g := `16s
  [1s]1: 2s
  [4s]2: 1s
  [5s]3: 10s
    [6s]3-1: 1s
    [7s]3-2: 7s
      [8s]3-2-1: 2s
      [10s]3-2-2: 1s
      [11s]3-2-3: 2s
        [12s]3-2-3-1: 1s
    [14s]3-3: 1s
  [15s]4: 1s`, s1.Dump(); e != g {
		t.Fatalf("expected %s, got %s", e, g)
	}
	if e, g := 16*time.Second, s1.Duration(); e != g {
		t.Fatalf("expected %s, got %s", e, g)
	}
	b, err := s5.MarshalJSON()
	if err != nil {
		t.Fatalf("expected no error, got %s", err)
	}
	if e, g := []byte(`{"children":[{"children":[],"created_at":11000000000,"done_at":12000000000,"label":"3-2-2"},{"children":[{"children":[],"created_at":13000000000,"done_at":14000000000,"label":"3-2-3-1"}],"created_at":12000000000,"done_at":14000000000,"label":"3-2-3"}],"created_at":10000000000,"done_at":14000000000,"label":""}`), b; !bytes.Equal(e, g) {
		t.Fatalf("expected %s, got %s", e, g)
	}
	var s7 Stopwatch
	if err = s7.UnmarshalJSON(b); err != nil {
		t.Fatalf("expected no error, got %s", err)
	}
	if e, g := *s5, s7; !reflect.DeepEqual(e, g) {
		t.Fatalf("expected %+v, got %+v", e, g)
	}
}

func TestDurationMinimalistFormat(t *testing.T) {
	for _, v := range []struct {
		d time.Duration
		e string
	}{
		{
			d: 123 * time.Nanosecond,
			e: "123ns",
		},
		{
			d: 123456 * time.Nanosecond,
			e: "123µs",
		},
		{
			d: 123456789 * time.Nanosecond,
			e: "123ms",
		},
		{
			d: 123456789123 * time.Nanosecond,
			e: "123s",
		},
	} {
		if g := DurationMinimalistFormat(v.d); v.e != g {
			t.Fatalf("expected %s, got %s", v.e, g)
		}
	}
}
