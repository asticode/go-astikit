package astikit

import (
	"sync"
)

type EventHandler func(payload interface{}) (delete bool)

type EventName string

type EventManager struct {
	handlerCount uint64
	// We use a map[int]... so that deletion is as smooth as possible
	hs map[EventName]map[uint64]EventHandler
	m  *sync.Mutex
}

func NewEventManager() *EventManager {
	return &EventManager{
		hs: make(map[EventName]map[uint64]EventHandler),
		m:  &sync.Mutex{},
	}
}

func (m *EventManager) On(n EventName, h EventHandler) uint64 {
	// Lock
	m.m.Lock()
	defer m.m.Unlock()

	// Make sure event name exists
	if _, ok := m.hs[n]; !ok {
		m.hs[n] = make(map[uint64]EventHandler)
	}

	// Increment handler count
	m.handlerCount++

	// Add handler
	m.hs[n][m.handlerCount] = h

	// Return id
	return m.handlerCount
}

func (m *EventManager) Off(id uint64) {
	// Lock
	m.m.Lock()
	defer m.m.Unlock()

	// Loop through handlers
	for _, ids := range m.hs {
		// Loop through ids
		for v := range ids {
			// Id matches
			if id == v {
				delete(ids, id)
			}
		}
	}
}

func (m *EventManager) Emit(n EventName, payload interface{}) {
	// Loop through handlers
	for _, h := range m.handlers(n) {
		if h.h(payload) {
			m.Off(h.id)
		}
	}
}

type eventManagerHandler struct {
	h  EventHandler
	id uint64
}

func (m *EventManager) handlers(n EventName) (hs []eventManagerHandler) {
	// Lock
	m.m.Lock()
	defer m.m.Unlock()

	// Index handlers
	hsm := make(map[uint64]eventManagerHandler)
	var ids []uint64
	if _, ok := m.hs[n]; ok {
		for id, h := range m.hs[n] {
			hsm[id] = eventManagerHandler{
				h:  h,
				id: id,
			}
			ids = append(ids, id)
		}
	}

	// Sort ids
	SortUint64(ids)

	// Append
	for _, id := range ids {
		hs = append(hs, hsm[id])
	}
	return
}
