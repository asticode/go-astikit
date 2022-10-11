package astikit

import (
	"context"
	"reflect"
	"sync"
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
	v1 := NewCounterRateStat()
	m1 := &StatMetadata{Description: "1"}
	o1 := StatOptions{Metadata: m1, Valuer: v1}
	v2 := NewDurationPercentageStat()
	m2 := &StatMetadata{Description: "2"}
	o2 := StatOptions{Metadata: m2, Valuer: v2}
	v3 := NewCounterAvgStat()
	m3 := &StatMetadata{Description: "3"}
	o3 := StatOptions{Metadata: m3, Valuer: v3}
	v4 := NewCounterStat()
	v4.Add(1)
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
				v1.Add(10)
				mn.Lock()
				nowV = time.Unix(0, 0)
				mn.Unlock()
				v2.Begin()
				mn.Lock()
				nowV = time.Unix(5, 0)
				mn.Unlock()
				v2.End()
				v3.Add(10)
				v3.Add(20)
				v3.Add(30)
				v4.Add(1)
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
		{StatMetadata: m2, Value: 100.0},
		{StatMetadata: m3, Value: 20.0},
		{StatMetadata: m4, Value: 2.0},
	} {
		found := false
		for _, s := range ss {
			if reflect.DeepEqual(s, e) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected %+v, not found", e)
		}
	}
}
