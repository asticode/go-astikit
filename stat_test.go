package astikit

import (
	"context"
	"reflect"
	"testing"
	"time"
)

func TestStater(t *testing.T) {
	// Update the now function so that it increments by 5s every time stats are computed
	var c int64
	nowPrevious := now
	defer func() { now = nowPrevious }()
	now = func() time.Time { return time.Unix(c*5, 0) }

	// Add stats
	s1 := NewCounterAvgStat()
	m1 := StatMetadata{Description: "1"}
	s2 := NewDurationPercentageStat()
	m2 := StatMetadata{Description: "2"}

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
				nowPrevious := now
				defer func() { now = nowPrevious }()
				now = func() time.Time { return time.Unix(0, 0) }
				s2.Begin()
				now = func() time.Time { return time.Unix(5, 0) }
				s2.End()
			default:
				ss = stats
				cancel()
			}
		},
		Period: time.Millisecond,
	})
	s.AddStat(m1, s1)
	s.AddStat(m2, s2)
	if e, g := []StatMetadata{m1, m2}, s.StatsMetadata(); !reflect.DeepEqual(g, e) {
		t.Errorf("expected %+v, got %+v", e, g)
	}
	defer s.Stop()
	s.Start(ctx)
	if e := []Stat{{StatMetadata: m1, Value: 2.0}, {StatMetadata: m2, Value: 100.0}}; !reflect.DeepEqual(e, ss) {
		t.Errorf("expected %+v, got %+v", e, ss)
	}
}
