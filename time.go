package astikit

import (
	"context"
	"encoding"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"
)

// Sleep is a cancellable sleep
func Sleep(ctx context.Context, d time.Duration) (err error) {
	for {
		select {
		case <-time.After(d):
			return
		case <-ctx.Done():
			err = ctx.Err()
			return
		}
	}
}

var now = time.Now

func Now() time.Time {
	return now()
}

type mockedNow struct {
	previous func() time.Time
}

func newMockedNow() *mockedNow {
	return &mockedNow{previous: now}
}

func (m *mockedNow) Close() error {
	now = m.previous
	return nil
}

func MockNow(fn func() time.Time) io.Closer {
	m := newMockedNow()
	now = fn
	return m
}

var (
	_ encoding.TextMarshaler   = (*Timestamp)(nil)
	_ encoding.TextUnmarshaler = (*Timestamp)(nil)
	_ json.Marshaler           = (*Timestamp)(nil)
	_ json.Unmarshaler         = (*Timestamp)(nil)
)

type Timestamp struct {
	time.Time
}

func NewTimestamp(t time.Time) *Timestamp {
	return &Timestamp{Time: t}
}

func (t *Timestamp) UnmarshalJSON(text []byte) error {
	return t.UnmarshalText(text)
}

func (t *Timestamp) UnmarshalText(text []byte) (err error) {
	var i int
	if i, err = strconv.Atoi(string(text)); err != nil {
		return
	}
	t.Time = time.Unix(int64(i), 0)
	return
}

func (t Timestamp) MarshalJSON() ([]byte, error) {
	return t.MarshalText()
}

func (t Timestamp) MarshalText() (text []byte, err error) {
	text = []byte(strconv.Itoa(int(t.UTC().Unix())))
	return
}

var (
	_ encoding.TextMarshaler   = (*TimestampNano)(nil)
	_ encoding.TextUnmarshaler = (*TimestampNano)(nil)
	_ json.Marshaler           = (*TimestampNano)(nil)
	_ json.Unmarshaler         = (*TimestampNano)(nil)
)

type TimestampNano struct {
	time.Time
}

func NewTimestampNano(t time.Time) *TimestampNano {
	return &TimestampNano{Time: t}
}

func (t *TimestampNano) UnmarshalJSON(text []byte) error {
	return t.UnmarshalText(text)
}

func (t *TimestampNano) UnmarshalText(text []byte) (err error) {
	var i int
	if i, err = strconv.Atoi(string(text)); err != nil {
		return
	}
	t.Time = time.Unix(0, int64(i))
	return
}

func (t TimestampNano) MarshalJSON() ([]byte, error) {
	return t.MarshalText()
}

func (t TimestampNano) MarshalText() (text []byte, err error) {
	text = []byte(strconv.Itoa(int(t.UTC().UnixNano())))
	return
}

var (
	_ json.Marshaler   = (*Stopwatch)(nil)
	_ json.Unmarshaler = (*Stopwatch)(nil)
)

type Stopwatch struct {
	children  []*Stopwatch
	createdAt time.Time
	doneAt    time.Time
	label     string
}

func NewStopwatch() *Stopwatch {
	return newStopwatch("")
}

func newStopwatch(label string) *Stopwatch {
	return &Stopwatch{
		createdAt: Now(),
		label:     label,
	}
}

func (s *Stopwatch) NewChild(label string) *Stopwatch {
	// Create stopwatch
	dst := newStopwatch(label)

	// Make sure to propagate done to children
	s.propagateDone(dst.createdAt)

	// Append
	s.children = append(s.children, dst)
	return dst
}

func (s *Stopwatch) propagateDone(doneAt time.Time) {
	// No children
	if len(s.children) == 0 {
		return
	}

	// Get child
	c := s.children[len(s.children)-1]

	// Update done at
	if c.doneAt.IsZero() {
		c.doneAt = doneAt
	}

	// Make sure to propagate done to children
	c.propagateDone(doneAt)
}

func (s *Stopwatch) Done() {
	// Update done at
	if s.doneAt.IsZero() {
		s.doneAt = Now()
	}

	// Make sure to propagate done to children
	s.propagateDone(s.doneAt)
}

func (s *Stopwatch) Duration() time.Duration {
	if !s.doneAt.IsZero() {
		return s.doneAt.Sub(s.createdAt)
	}
	return Now().Sub(s.createdAt)
}

func (s *Stopwatch) Merge(i *Stopwatch) {
	// No children
	if len(i.children) == 0 {
		return
	}

	// Make sure to propagate done to children
	s.propagateDone(i.children[0].createdAt)

	// Append
	s.children = append(s.children, i.children...)
}

func (s *Stopwatch) Dump() string {
	return s.dump("", s.createdAt)
}

func (s *Stopwatch) dump(ident string, rootCreatedAt time.Time) string {
	// Dump stopwatch
	var ss []string
	if ident == "" {
		ss = append(ss, DurationMinimalistFormat(s.doneAt.Sub(s.createdAt)))
	} else {
		ss = append(ss, fmt.Sprintf("%s[%s]%s: %s", ident, DurationMinimalistFormat(s.createdAt.Sub(rootCreatedAt)), s.label, DurationMinimalistFormat(s.doneAt.Sub(s.createdAt))))
	}

	// Loop through children
	ident += "  "
	for _, c := range s.children {
		// Dump child
		ss = append(ss, c.dump(ident, rootCreatedAt))
	}
	return strings.Join(ss, "\n")
}

type stopwatchJSON struct {
	Children  []stopwatchJSON `json:"children"`
	CreatedAt TimestampNano   `json:"created_at"`
	DoneAt    TimestampNano   `json:"done_at"`
	Label     string          `json:"label"`
}

func (sj stopwatchJSON) toStopwatch(s *Stopwatch) {
	s.createdAt = sj.CreatedAt.Time
	s.doneAt = sj.DoneAt.Time
	s.label = sj.Label
	for _, cj := range sj.Children {
		c := &Stopwatch{}
		cj.toStopwatch(c)
		s.children = append(s.children, c)
	}
}

func (s *Stopwatch) toStopwatchJSON() (sj stopwatchJSON) {
	sj.Children = []stopwatchJSON{}
	sj.CreatedAt = *NewTimestampNano(s.createdAt)
	sj.DoneAt = *NewTimestampNano(s.doneAt)
	sj.Label = s.label
	for _, c := range s.children {
		sj.Children = append(sj.Children, c.toStopwatchJSON())
	}
	return
}

func (s *Stopwatch) UnmarshalJSON(text []byte) error {
	var j stopwatchJSON
	if err := json.Unmarshal(text, &j); err != nil {
		return err
	}
	j.toStopwatch(s)
	return nil
}

func (s Stopwatch) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.toStopwatchJSON())
}

func DurationMinimalistFormat(d time.Duration) string {
	if d < time.Microsecond {
		return strconv.Itoa(int(d)) + "ns"
	} else if d < time.Millisecond {
		return strconv.Itoa(int(d/1e3)) + "µs"
	} else if d < time.Second {
		return strconv.Itoa(int(d/1e6)) + "ms"
	}
	return strconv.Itoa(int(d/1e9)) + "s"
}
