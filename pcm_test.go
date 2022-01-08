package astikit

import (
	"reflect"
	"testing"
	"time"
)

func TestPCMLevel(t *testing.T) {
	if e, g := 2.160246899469287, PCMLevel([]int{1, 2, 3}); g != e {
		t.Errorf("got %v, expected %v", g, e)
	}
}

func TestPCMNormalize(t *testing.T) {
	// Nothing to do
	i := []int{10000, maxPCMSample(16), -10000}
	if g := PCMNormalize(i, 16); !reflect.DeepEqual(i, g) {
		t.Errorf("got %+v, expected %+v", g, i)
	}

	// Normalize
	i = []int{10000, 0, -10000}
	if e, g := []int{32767, 0, -32767}, PCMNormalize(i, 16); !reflect.DeepEqual(e, g) {
		t.Errorf("got %+v, expected %+v", g, e)
	}
}

func TestConvertPCMBitDepth(t *testing.T) {
	// Nothing to do
	s, err := ConvertPCMBitDepth(1>>8, 16, 16)
	if err != nil {
		t.Errorf("expected no error, got %+v", err)
	}
	if e := 1 >> 8; !reflect.DeepEqual(s, e) {
		t.Errorf("got %+v, expected %+v", s, e)
	}

	// Src bit depth > Dst bit depth
	s, err = ConvertPCMBitDepth(1>>24, 32, 16)
	if err != nil {
		t.Errorf("expected no error, got %+v", err)
	}
	if e := 1 >> 8; !reflect.DeepEqual(s, e) {
		t.Errorf("got %+v, expected %+v", s, e)
	}

	// Src bit depth < Dst bit depth
	s, err = ConvertPCMBitDepth(1>>8, 16, 32)
	if err != nil {
		t.Errorf("expected no error, got %+v", err)
	}
	if e := 1 >> 24; !reflect.DeepEqual(s, e) {
		t.Errorf("got %+v, expected %+v", s, e)
	}
}

func TestPCMSampleRateConverter(t *testing.T) {
	// Create input
	var i []int
	for idx := 0; idx < 20; idx++ {
		i = append(i, idx+1)
	}

	// Create sample func
	var o []int
	var sampleFunc = func(s int) (err error) {
		o = append(o, s)
		return
	}

	// Nothing to do
	c := NewPCMSampleRateConverter(1, 1, 1, sampleFunc)
	for _, s := range i {
		c.Add(s) //nolint:errcheck
	}
	if !reflect.DeepEqual(o, i) {
		t.Errorf("got %+v, expected %+v", i, o)
	}

	// Simple src sample rate > dst sample rate
	o = []int{}
	c = NewPCMSampleRateConverter(5, 3, 1, sampleFunc)
	for _, s := range i {
		c.Add(s) //nolint:errcheck
	}
	if e := []int{1, 2, 4, 6, 7, 9, 11, 12, 14, 16, 17, 19}; !reflect.DeepEqual(e, o) {
		t.Errorf("got %+v, expected %+v", o, e)
	}

	// Multi channels
	o = []int{}
	c = NewPCMSampleRateConverter(4, 2, 2, sampleFunc)
	for _, s := range i {
		c.Add(s) //nolint:errcheck
	}
	if e := []int{1, 2, 4, 5, 8, 9, 12, 13, 16, 17}; !reflect.DeepEqual(e, o) {
		t.Errorf("got %+v, expected %+v", o, e)
	}

	// Realistic src sample rate > dst sample rate
	i = []int{}
	for idx := 0; idx < 4*44100; idx++ {
		i = append(i, idx+1)
	}
	o = []int{}
	c = NewPCMSampleRateConverter(44100, 16000, 2, sampleFunc)
	for _, s := range i {
		c.Add(s) //nolint:errcheck
	}
	if e, g := 4*16000, len(o); g != e {
		t.Errorf("invalid len, got %v, expected %v", g, e)
	}

	// Create input
	i = []int{}
	for idx := 0; idx < 10; idx++ {
		i = append(i, idx+1)
	}

	// Simple src sample rate < dst sample rate
	o = []int{}
	c = NewPCMSampleRateConverter(3, 5, 1, sampleFunc)
	for _, s := range i {
		c.Add(s) //nolint:errcheck
	}
	if e := []int{1, 1, 2, 2, 3, 4, 4, 5, 5, 6, 7, 7, 8, 8, 9, 10, 10}; !reflect.DeepEqual(e, o) {
		t.Errorf("got %+v, expected %+v", o, e)
	}

	// Multi channels
	o = []int{}
	c = NewPCMSampleRateConverter(3, 5, 2, sampleFunc)
	for _, s := range i {
		c.Add(s) //nolint:errcheck
	}
	if e := []int{1, 2, 1, 2, 3, 4, 3, 4, 5, 6, 7, 8, 7, 8, 9, 10, 9, 10}; !reflect.DeepEqual(e, o) {
		t.Errorf("got %+v, expected %+v", o, e)
	}
}

func TestPCMChannelsConverter(t *testing.T) {
	// Create input
	var i []int
	for idx := 0; idx < 20; idx++ {
		i = append(i, idx+1)
	}

	// Create sample func
	var o []int
	var sampleFunc = func(s int) (err error) {
		o = append(o, s)
		return
	}

	// Nothing to do
	c := NewPCMChannelsConverter(3, 3, sampleFunc)
	for _, s := range i {
		c.Add(s) //nolint:errcheck
	}
	if !reflect.DeepEqual(i, o) {
		t.Errorf("got %+v, expected %+v", o, i)
	}

	// Throw away data
	o = []int{}
	c = NewPCMChannelsConverter(3, 1, sampleFunc)
	for _, s := range i {
		c.Add(s) //nolint:errcheck
	}
	if e := []int{1, 4, 7, 10, 13, 16, 19}; !reflect.DeepEqual(e, o) {
		t.Errorf("got %+v, expected %+v", o, e)
	}

	// Repeat data
	o = []int{}
	c = NewPCMChannelsConverter(1, 2, sampleFunc)
	for _, s := range i {
		c.Add(s) //nolint:errcheck
	}
	if e := []int{1, 1, 2, 2, 3, 3, 4, 4, 5, 5, 6, 6, 7, 7, 8, 8, 9, 9, 10, 10, 11, 11, 12, 12, 13, 13, 14, 14, 15, 15, 16, 16, 17, 17, 18, 18, 19, 19, 20, 20}; !reflect.DeepEqual(o, e) {
		t.Errorf("got %+v, expected %+v", o, e)
	}
}

func TestPCMSilenceDetector(t *testing.T) {
	// Create silence detector
	sd := NewPCMSilenceDetector(PCMSilenceDetectorOptions{
		MaxSilenceLevel:    2,
		MinSilenceDuration: 400 * time.Millisecond, // 2 samples
		SampleRate:         5,
		StepDuration:       200 * time.Millisecond, // 1 sample
	})

	// Leading non silences + invalid leading silence + trailing silence is leftover
	vs := sd.Add([]int{3, 1, 3, 1})
	if e := [][]int(nil); !reflect.DeepEqual(vs, e) {
		t.Errorf("got %+v, expected %+v", vs, e)
	}
	if e, g := 1, len(sd.analyses); e != g {
		t.Errorf("got %v, expected %v", g, e)
	}

	// Valid leading silence but trailing silence is insufficient for now
	vs = sd.Add([]int{1, 3, 3, 1})
	if e := [][]int(nil); !reflect.DeepEqual(vs, e) {
		t.Errorf("got %+v, expected %+v", vs, e)
	}
	if e, g := 5, len(sd.analyses); e != g {
		t.Errorf("got %v, expected %v", g, e)
	}

	// Valid samples
	vs = sd.Add([]int{1})
	if e := [][]int{{1, 1, 3, 3, 1, 1}}; !reflect.DeepEqual(vs, e) {
		t.Errorf("got %+v, expected %+v", vs, e)
	}
	if e, g := 2, len(sd.analyses); e != g {
		t.Errorf("got %v, expected %v", g, e)
	}

	// Multiple valid samples + truncate leading and trailing silences
	vs = sd.Add([]int{1, 1, 1, 1, 3, 3, 1, 1, 1, 1, 3, 3, 1, 1, 1, 1})
	if e := [][]int{{1, 1, 3, 3, 1, 1}, {1, 1, 3, 3, 1, 1}}; !reflect.DeepEqual(vs, e) {
		t.Errorf("got %+v, expected %+v", vs, e)
	}
	if e, g := 2, len(sd.analyses); e != g {
		t.Errorf("got %v, expected %v", g, e)
	}

	// Invalid in-between silences that should be kept
	vs = sd.Add([]int{1, 1, 1, 3, 3, 1, 3, 3, 1, 3, 3, 1, 1, 1})
	if e := [][]int{{1, 1, 3, 3, 1, 3, 3, 1, 3, 3, 1, 1}}; !reflect.DeepEqual(vs, e) {
		t.Errorf("got %+v, expected %+v", vs, e)
	}
	if e, g := 2, len(sd.analyses); e != g {
		t.Errorf("got %v, expected %v", g, e)
	}
}
