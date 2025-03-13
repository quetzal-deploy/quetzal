package events

import (
	"github.com/google/uuid"
	"sync"
)

type EventWithId struct {
	Id    string
	Event Event
}

type Manager struct {
	lock        sync.RWMutex
	events      []EventWithId
	subscribers []chan EventWithId
}

func NewManager() *Manager {
	return &Manager{
		lock:        sync.RWMutex{},
		events:      make([]EventWithId, 0),
		subscribers: make([]chan EventWithId, 0),
	}
}

func (m *Manager) SendEvent(event Event) {
	m.lock.Lock()
	defer m.lock.Unlock()

	eventWithId := EventWithId{Id: uuid.New().String(), Event: event}
	m.events = append(m.events, eventWithId)

	for _, subscriber := range m.subscribers {
		subscriber <- eventWithId
	}
}

func (m *Manager) Subscribe() chan EventWithId {
	// This is locked to avoid races with adding new subscribers and new events at the same time,
	// to avoid a new subscriber missing an event
	m.lock.Lock()
	defer m.lock.Unlock()

	subscriberChan := make(chan EventWithId, 1000000)

	m.subscribers = append(m.subscribers, subscriberChan)

	// spool all existing events to the new subscriber
	for _, event := range m.events {
		subscriberChan <- event
	}

	return subscriberChan
}

func (m *Manager) GetEvents(lastId string, batchSize int) ([]Event, string) {
	batch := make([]Event, 0)

	add := false
	nextLastId := lastId

	// stream from beginning if lastId is blank
	if lastId == "" {
		add = true
	}

	for _, event := range m.events {
		if add {
			batch = append(batch, event.Event)
			nextLastId = event.Id

			if len(batch) == batchSize {
				break
			}
		} else {
			if event.Id == lastId {
				add = true
			}
		}
	}

	return batch, nextLastId
}

func (m *Manager) NewLogWriter() LogWriter {
	return LogWriter{
		eventMgr: m,
	}
}
