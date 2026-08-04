package repositories

import "testing"

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
