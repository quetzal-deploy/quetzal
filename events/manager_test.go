package events

import (
	"fmt"
	"testing"
)

func TestSubscribeBeforeAddingEvent(t *testing.T) {
	manager := NewManager()

	data := "hello world"

	subscriber := manager.Subscribe()
	manager.SendEvent(Log{Data: data})

	event1 := <-subscriber

	if event1.Event.(Log).Data != data {
		t.Fatalf("Expected returned event data = %s, got %s", data, event1.Event.(Log).Data)
	}
}

func TestSubscribeAfterAddingEvent(t *testing.T) {
	manager := NewManager()

	data := "hello world"

	manager.SendEvent(Log{Data: data})
	subscriber := manager.Subscribe()

	event1 := <-subscriber

	if event1.Event.(Log).Data != data {
		t.Fatalf("Expected returned event data = %s, got %s", data, event1.Event.(Log).Data)
	}
}

func TestGetEventsBatching(t *testing.T) {
	manager := NewManager()

	events := make([]Event, 0)

	for i := range 9 {
		event := Log{Data: fmt.Sprintf("hello test event %d", i)}
		events = append(events, event)
		manager.SendEvent(event)
	}

	batchSize := 5

	batch1, lastId1 := manager.GetEvents("", batchSize)
	batch2, _ := manager.GetEvents(lastId1, batchSize)

	if len(batch1) != batchSize {
		t.Fatalf("Expected %d events, got %d", batchSize, len(batch1))
	}

	if len(batch2) != len(events)-batchSize {
		t.Fatalf("Expected %d events, got %d", len(events)-batchSize, len(batch1))
	}

	for i, event := range batch1 {
		expected := events[i].(Log).Data
		actual := event.(Log).Data

		if actual != expected {
			t.Fatalf("batch1: event %d: Expected \"%s\", got \"%s\"", i, expected, actual)
		}
	}

	for i, event := range batch2 {
		expected := events[i+batchSize].(Log).Data
		actual := event.(Log).Data

		if actual != expected {
			t.Fatalf("batch2: event %d: Expected \"%s\", got \"%s\"", i, expected, actual)
		}
	}

}
