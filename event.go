package astikit

import (
	"sort"
	"sync"
)

type EventHandler func(payload interface{}) (delete bool)

type EventName string

type EventManager struct {
	// We use a map[int]... so that deletion is as smooth as possible
	hs  map[EventName]map[int]EventHandler
	idx int
	m   *sync.Mutex
}

func NewEventManager() *EventManager {
	return &EventManager{
		hs: make(map[EventName]map[int]EventHandler),
		m:  &sync.Mutex{},
	}
}

func (m *EventManager) On(n EventName, h EventHandler) {
	// Lock
	m.m.Lock()
	defer m.m.Unlock()

	// Make sure event name exists
	if _, ok := m.hs[n]; !ok {
		m.hs[n] = make(map[int]EventHandler)
	}

	// Increment index
	m.idx++

	// Add handler
	m.hs[n][m.idx] = h
}

func (m *EventManager) del(n EventName, idx int) {
	// Lock
	m.m.Lock()
	defer m.m.Unlock()

	// Event name doesn't exist
	if _, ok := m.hs[n]; !ok {
		return
	}

	// Delete index
	delete(m.hs[n], idx)
}

func (m *EventManager) Emit(n EventName, payload interface{}) {
	// Loop through handlers
	for _, h := range m.handlers(n) {
		if h.h(payload) {
			m.del(n, h.idx)
		}
	}
}

type eventManagerHandler struct {
	h   EventHandler
	idx int
}

func (m *EventManager) handlers(n EventName) (hs []eventManagerHandler) {
	// Lock
	m.m.Lock()
	defer m.m.Unlock()

	// Index handlers
	hsm := make(map[int]eventManagerHandler)
	var idxs []int
	if _, ok := m.hs[n]; ok {
		for idx, h := range m.hs[n] {
			hsm[idx] = eventManagerHandler{
				h:   h,
				idx: idx,
			}
			idxs = append(idxs, idx)
		}
	}

	// Sort indexes
	sort.Ints(idxs)

	// Append
	for _, idx := range idxs {
		hs = append(hs, hsm[idx])
	}
	return
}
