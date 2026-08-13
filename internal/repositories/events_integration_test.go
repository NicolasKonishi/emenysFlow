//go:build private_seeds

package repositories

import (
	"fmt"
	"testing"
	"time"

	"buffetflow/internal/models"
)

func sampleEvent(name string) models.Event {
	starts := time.Date(2026, 9, 1, 18, 0, 0, 0, time.UTC)
	return models.Event{
		ClientName: "Cliente teste",
		Name:       name,
		Venue:      "Salão principal",
		StartsAt:   starts,
		EndsAt:     starts.Add(4 * time.Hour),
		GuestCount: 80,
	}
}

func TestSaveEventUsesDefaultNameWhenMissing(t *testing.T) {
	store, _, ctx := newModelWorkflowStore(t)

	created := sampleEvent("")
	if err := store.SaveEvent(ctx, &created, 0); err != nil {
		t.Fatal(err)
	}
	wantName := fmt.Sprintf("evento-%d", created.ID)
	if created.Name != wantName {
		t.Fatalf("created event name got %q, want %q", created.Name, wantName)
	}

	stored, err := store.GetEvent(ctx, created.ID)
	if err != nil {
		t.Fatal(err)
	}
	if stored.Name != wantName {
		t.Fatalf("stored event name got %q, want %q", stored.Name, wantName)
	}

	updated := stored
	updated.Name = ""
	if err := store.SaveEvent(ctx, &updated, 0); err != nil {
		t.Fatal(err)
	}
	if updated.Name != wantName {
		t.Fatalf("updated event name got %q, want %q", updated.Name, wantName)
	}
}

func TestCancelledEventIsKeptInHistoryButHiddenFromEventList(t *testing.T) {
	store, _, ctx := newModelWorkflowStore(t)

	events, err := store.ListEvents(ctx, "")
	if err != nil {
		t.Fatal(err)
	}
	foundBefore := false
	for _, event := range events {
		foundBefore = foundBefore || event.ID == 1
	}
	if !foundBefore {
		t.Fatal("seed event should be visible before cancellation")
	}

	if err := store.CancelEvent(ctx, 1, 0); err != nil {
		t.Fatal(err)
	}
	events, err = store.ListEvents(ctx, "")
	if err != nil {
		t.Fatal(err)
	}
	for _, event := range events {
		if event.ID == 1 {
			t.Fatal("cancelled event should not remain in the active event list")
		}
	}

	event, err := store.GetEvent(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}
	if event.Status != "cancelled" {
		t.Fatalf("stored event status got %q, want cancelled", event.Status)
	}
}
