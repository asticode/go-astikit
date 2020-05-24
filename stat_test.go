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
	nowPrevious := now
	defer func() { now = nowPrevious }()
	m := &sync.Mutex{}
	nowV := time.Unix(c*5, 0)
	now = func() time.Time {
		m.Lock()
		defer m.Unlock()
		return nowV
	}

	// Add stats
	s1 := NewCounterRateStat()
	m1 := StatMetadata{Description: "1"}
	s2 := NewDurationPercentageStat()
	m2 := StatMetadata{Description: "2"}
	s3 := NewCounterAvgStat()
	m3 := StatMetadata{Description: "3"}

	// First time stats are computed, it actually acts as if stats were being updated
	// Second time stats are computed, results are stored and context is cancelled
	var ss []Stat
	ctx, cancel := context.WithCancel(context.Background())
	s := NewStater(StaterOptions{
		HandleFunc: func(stats []Stat) {
			c++
			switch c {
			case 1:
				s1.Add(10)
				m.Lock()
				nowV = time.Unix(0, 0)
				m.Unlock()
				s2.Begin()
				m.Lock()
				nowV = time.Unix(5, 0)
				m.Unlock()
				s2.End()
				s3.Add(10)
				s3.Add(20)
				s3.Add(30)
			default:
				ss = stats
				cancel()
			}
		},
		Period: time.Millisecond,
	})
	s.AddStat(m1, s1)
	s.AddStat(m2, s2)
	s.AddStat(m3, s3)
	if e, g := []StatMetadata{m1, m2, m3}, s.StatsMetadata(); !reflect.DeepEqual(g, e) {
		t.Errorf("expected %+v, got %+v", e, g)
	}
	defer s.Stop()
	s.Start(ctx)
	if e := []Stat{{StatMetadata: m1, Value: 2.0}, {StatMetadata: m2, Value: 100.0}, {StatMetadata: m3, Value: 20.0}}; !reflect.DeepEqual(e, ss) {
		t.Errorf("expected %+v, got %+v", e, ss)
	}
}
