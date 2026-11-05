package live

import (
	"testing"
	"time"
)

func TestHubPublishSubscribe(t *testing.T) {
	h := NewHub()
	ch, cancel := h.Subscribe(42)
	defer cancel()
	if h.SubscriberCount(42) != 1 {
		t.Fatalf("abone sayısı 1 olmalı")
	}
	h.Publish(Event{ContentID: 42, Version: 2, Status: "yayinda"})
	select {
	case ev := <-ch:
		if ev.ContentID != 42 || ev.Version != 2 || ev.Status != "yayinda" {
			t.Fatalf("beklenmeyen olay: %+v", ev)
		}
	case <-time.After(time.Second):
		t.Fatal("olay gelmedi")
	}
	cancel()
	if h.SubscriberCount(42) != 0 {
		t.Fatalf("abonelik kapanmalı")
	}
}

func TestHubDoesNotBlockOnSlowSubscriber(t *testing.T) {
	h := NewHub()
	_, cancel := h.Subscribe(7)
	defer cancel()
	for i := 0; i < 20; i++ {
		h.Publish(Event{ContentID: 7, Version: i + 1})
	}
}
