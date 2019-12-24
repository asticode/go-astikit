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
	if !reflect.DeepEqual(o, []int{1, 2}) {
		t.Errorf("expected %+v, got %+v", []int{1, 2}, o)
	}
}
