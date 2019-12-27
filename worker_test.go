package astikit

import (
	"reflect"
	"testing"
)

func TestWorker(t *testing.T) {
	w := NewWorker(WorkerOptions{})
	ts := w.NewTask()
	var o []int
	ts.Do(func() {
		w.Stop()
		o = append(o, 1)
	})
	w.Wait()
	o = append(o, 2)
	if e := []int{1, 2}; !reflect.DeepEqual(o, e) {
		t.Errorf("expected %+v, got %+v", e, o)
	}
}
