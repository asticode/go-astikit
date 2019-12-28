package astikit

import (
	"reflect"
	"testing"
)

func TestSortInt64(t *testing.T) {
	i := []int64{3, 2, 4, 1}
	SortInt64(i)
	if e := []int64{1, 2, 3, 4}; !reflect.DeepEqual(e, i) {
		t.Errorf("expected %+v, got %+v", e, i)
	}
}
