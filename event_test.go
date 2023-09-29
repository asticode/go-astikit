package astikit_test

import (
	"reflect"
	"testing"

	"github.com/asticode/go-astikit"
)

func TestEvent(t *testing.T) {
	const (
		eventName1 astikit.EventName = "event-name-1"
		eventName2 astikit.EventName = "event-name-2"
		eventName3 astikit.EventName = "event-name-3"
	)
	m := astikit.NewEventManager()
	ons := make(map[astikit.EventName][]interface{})
	m.On(eventName1, func(payload interface{}) (delete bool) {
		ons[eventName1] = append(ons[eventName1], payload)
		return true
	})
	m.On(eventName3, func(payload interface{}) (delete bool) {
		ons[eventName3] = append(ons[eventName3], payload)
		return false
	})

	m.Emit(eventName1, 1)
	m.Emit(eventName1, 2)
	m.Emit(eventName2, 1)
	m.Emit(eventName2, 2)
	m.Emit(eventName3, 1)
	m.Emit(eventName3, 2)

	if e, g := map[astikit.EventName][]interface{}{
		eventName1: {1},
		eventName3: {1, 2},
	}, ons; !reflect.DeepEqual(e, g) {
		t.Errorf("expected %+v, got %+v", e, g)
	}
}
