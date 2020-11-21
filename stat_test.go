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
	h1 := NewCounterRateStat()
	m1 := &StatMetadata{Description: "1"}
	o1 := StatOptions{Handler: h1, Metadata: m1}
	h2 := NewDurationPercentageStat()
	m2 := &StatMetadata{Description: "2"}
	o2 := StatOptions{Handler: h2, Metadata: m2}
	h3 := NewCounterAvgStat()
	m3 := &StatMetadata{Description: "3"}
	o3 := StatOptions{Handler: h3, Metadata: m3}

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
				h1.Add(10)
				mn.Lock()
				nowV = time.Unix(0, 0)
				mn.Unlock()
				h2.Begin()
				mn.Lock()
				nowV = time.Unix(5, 0)
				mn.Unlock()
				h2.End()
				h3.Add(10)
				h3.Add(20)
				h3.Add(30)
			default:
				ss = stats
				cancel()
			}
		},
		Period: time.Millisecond,
	})
	s.AddStats(o1, o2, o3)
	for _, o := range []StatOptions{o1, o2, o3} {
		o.Handler.Start()
		defer o.Handler.Stop()
	}
	s.Start(ctx)
	defer s.Stop()
	if e := []StatValue{{StatMetadata: m1, Value: 2.0}, {StatMetadata: m2, Value: 100.0}, {StatMetadata: m3, Value: 20.0}}; !reflect.DeepEqual(e, ss) {
		t.Errorf("expected %+v, got %+v", e, ss)
	}
}
