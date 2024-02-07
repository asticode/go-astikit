package astikit

import (
	"context"
	"reflect"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestStater(t *testing.T) {
	// Update the now function so that it increments by 5s every time stats are computed
	var c int64
	mc := &sync.Mutex{} // Locks c
	nowPrevious := now
	defer func() { now = nowPrevious }()
	mn := &sync.Mutex{} // Locks nowV
	nowV := time.Unix(c*5, 0)
	now = func() time.Time {
		mn.Lock()
		defer mn.Unlock()
		return nowV
	}

	// Add stats
	var u1 uint64
	v1 := NewAtomicUint64RateStat(&u1)
	m1 := &StatMetadata{Description: "1"}
	o1 := StatOptions{Metadata: m1, Valuer: v1}
	d2 := NewAtomicDuration(0)
	v2 := NewAtomicDurationPercentageStat(d2)
	m2 := &StatMetadata{Description: "2"}
	o2 := StatOptions{Metadata: m2, Valuer: v2}
	d3 := NewAtomicDuration(0)
	v3 := NewAtomicDurationAvgStat(d3, &u1)
	m3 := &StatMetadata{Description: "3"}
	o3 := StatOptions{Metadata: m3, Valuer: v3}
	v4 := StatValuerFunc(func(d time.Duration) interface{} { return 42 })
	m4 := &StatMetadata{Description: "4"}
	o4 := StatOptions{Metadata: m4, Valuer: v4}

	// First time stats are computed, it actually acts as if stats were being updated
	// Second time stats are computed, results are stored and context is cancelled
	var ss []StatValue
	ctx, cancel := context.WithCancel(context.Background())
	s := NewStater(StaterOptions{
		HandleFunc: func(stats []StatValue) {
			mc.Lock()
			defer mc.Unlock()
			c++
			switch c {
			case 1:
				atomic.AddUint64(&u1, 10)
				d2.Add(4 * time.Second)
				d3.Add(10 * time.Second)
				mn.Lock()
				nowV = time.Unix(5, 0)
				mn.Unlock()
			case 2:
				ss = stats
				cancel()
			}
		},
		Period: time.Millisecond,
	})
	s.AddStats(o1, o2, o3, o4)
	s.Start(ctx)
	defer s.Stop()
	for _, e := range []StatValue{
		{StatMetadata: m1, Value: 2.0},
		{StatMetadata: m2, Value: 80.0},
		{StatMetadata: m3, Value: time.Second},
		{StatMetadata: m4, Value: 42},
	} {
		found := false
		for _, s := range ss {
			if reflect.DeepEqual(s, e) {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("expected %+v, not found", e)
		}
	}
}
