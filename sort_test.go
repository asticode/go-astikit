package astikit

import (
	"reflect"
	"testing"
)

func TestSort(t *testing.T) {
	i := []int64{3, 2, 4, 1}
	SortInt64(i)
	if e := []int64{1, 2, 3, 4}; !reflect.DeepEqual(e, i) {
		t.Fatalf("expected %+v, got %+v", e, i)
	}

	ui := []uint64{3, 2, 4, 1}
	SortUint64(ui)
	if e := []uint64{1, 2, 3, 4}; !reflect.DeepEqual(e, ui) {
		t.Fatalf("expected %+v, got %+v", e, ui)
	}
}
