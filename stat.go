package astikit

import (
	"context"
	"sync"
	"sync/atomic"
	"time"
)

// Stater is an object that can compute and handle stats
type Stater struct {
	cancel  context.CancelFunc
	ctx     context.Context
	h       StatsHandleFunc
	m       *sync.Mutex // Locks ss
	period  time.Duration
	running uint32
	ss      map[*StatMetadata]StatOptions
}

// StatOptions represents stat options
type StatOptions struct {
	Metadata *StatMetadata
	// Either a StatValuer or StatValuerOverTime
	Valuer interface{}
}

// StatsHandleFunc is a method that can handle stat values
type StatsHandleFunc func(stats []StatValue)

// StatMetadata represents a stat metadata
type StatMetadata struct {
	Description string
	Label       string
	Name        string
	Unit        string
}

// StatValuer represents a stat valuer
type StatValuer interface {
	Value(delta time.Duration) interface{}
}

type StatValuerFunc func(d time.Duration) interface{}

func (f StatValuerFunc) Value(d time.Duration) interface{} {
	return f(d)
}

// StatValue represents a stat value
type StatValue struct {
	*StatMetadata
	Value interface{}
}

// StaterOptions represents stater options
type StaterOptions struct {
	HandleFunc StatsHandleFunc
	Period     time.Duration
}

// NewStater creates a new stater
func NewStater(o StaterOptions) *Stater {
	return &Stater{
		h:      o.HandleFunc,
		m:      &sync.Mutex{},
		period: o.Period,
		ss:     make(map[*StatMetadata]StatOptions),
	}
}

// Start starts the stater
func (s *Stater) Start(ctx context.Context) {
	// Check context
	if ctx.Err() != nil {
		return
	}

	// Make sure to start only once
	if atomic.CompareAndSwapUint32(&s.running, 0, 1) {
		// Update status
		defer atomic.StoreUint32(&s.running, 0)

		// Reset context
		s.ctx, s.cancel = context.WithCancel(ctx)

		// Create ticker
		t := time.NewTicker(s.period)
		defer t.Stop()

		// Loop
		lastStatAt := now()
		for {
			select {
			case <-t.C:
				// Get delta
				n := now()
				delta := n.Sub(lastStatAt)
				lastStatAt = n

				// Loop through stats
				var stats []StatValue
				s.m.Lock()
				for _, o := range s.ss {
					// Get value
					var v interface{}
					if h, ok := o.Valuer.(StatValuer); ok {
						v = h.Value(delta)
					} else {
						continue
					}

					// Append
					stats = append(stats, StatValue{
						StatMetadata: o.Metadata,
						Value:        v,
					})
				}
				s.m.Unlock()

				// Handle stats
				go s.h(stats)
			case <-s.ctx.Done():
				return
			}
		}
	}
}

// Stop stops the stater
func (s *Stater) Stop() {
	if s.cancel != nil {
		s.cancel()
	}
}

// AddStats adds stats
func (s *Stater) AddStats(os ...StatOptions) {
	s.m.Lock()
	defer s.m.Unlock()
	for _, o := range os {
		s.ss[o.Metadata] = o
	}
}

// DelStats deletes stats
func (s *Stater) DelStats(os ...StatOptions) {
	s.m.Lock()
	defer s.m.Unlock()
	for _, o := range os {
		delete(s.ss, o.Metadata)
	}
}

type AtomicUint64RateStat struct {
	last *uint64
	v    *uint64
}

func NewAtomicUint64RateStat(v *uint64) *AtomicUint64RateStat {
	return &AtomicUint64RateStat{v: v}
}

func (s *AtomicUint64RateStat) Value(d time.Duration) interface{} {
	current := atomic.LoadUint64(s.v)
	defer func() { s.last = &current }()
	if d <= 0 {
		return 0.0
	}
	var last uint64
	if s.last != nil {
		last = *s.last
	}
	return float64(current-last) / d.Seconds()
}

type AtomicDurationPercentageStat struct {
	d    *AtomicDuration
	last *time.Duration
}

func NewAtomicDurationPercentageStat(d *AtomicDuration) *AtomicDurationPercentageStat {
	return &AtomicDurationPercentageStat{d: d}
}

func (s *AtomicDurationPercentageStat) Value(d time.Duration) interface{} {
	current := s.d.Duration()
	defer func() { s.last = &current }()
	if d <= 0 {
		return 0.0
	}
	var last time.Duration
	if s.last != nil {
		last = *s.last
	}
	return float64(current-last) / float64(d) * 100
}

type AtomicDurationAvgStat struct {
	count     *uint64
	d         *AtomicDuration
	last      *time.Duration
	lastCount *uint64
}

func NewAtomicDurationAvgStat(d *AtomicDuration, count *uint64) *AtomicDurationAvgStat {
	return &AtomicDurationAvgStat{
		count: count,
		d:     d,
	}
}

func (s *AtomicDurationAvgStat) Value(_ time.Duration) interface{} {
	current := s.d.Duration()
	currentCount := atomic.LoadUint64(s.count)
	defer func() {
		s.last = &current
		s.lastCount = &currentCount
	}()
	var last time.Duration
	var lastCount uint64
	if s.last != nil {
		last = *s.last
	}
	if s.lastCount != nil {
		lastCount = *s.lastCount
	}
	if currentCount-lastCount <= 0 {
		return time.Duration(0)
	}
	return time.Duration(float64(current-last) / float64(currentCount-lastCount))
}
