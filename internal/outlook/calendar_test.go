package outlook

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestListEventsReturnsErrNotConnectedWhenExecutorNotStarted(t *testing.T) {
	store := &outlookCalendarStore{executor: &fakeCOMExecutor{started: false}}

	_, err := store.ListEvents(context.Background(), ListEventsParams{Start: time.Now(), End: time.Now().Add(time.Hour)})

	if !errors.Is(err, ErrNotConnected) {
		t.Fatalf("ListEvents() error = %v, want %v", err, ErrNotConnected)
	}
}

func TestGetEventReturnsErrNotConnectedWhenExecutorNotStarted(t *testing.T) {
	store := &outlookCalendarStore{executor: &fakeCOMExecutor{started: false}}

	_, err := store.GetEvent(context.Background(), "event-id")

	if !errors.Is(err, ErrNotConnected) {
		t.Fatalf("GetEvent() error = %v, want %v", err, ErrNotConnected)
	}
}

func TestListEventsReturnsErrInvalidParamsWhenRangeReversed(t *testing.T) {
	store := &outlookCalendarStore{executor: &fakeCOMExecutor{started: true}}
	since := time.Date(2026, time.May, 17, 9, 0, 0, 0, time.UTC)
	until := time.Date(2026, time.May, 16, 9, 0, 0, 0, time.UTC)

	_, err := store.ListEvents(context.Background(), ListEventsParams{Start: since, End: until})

	if !errors.Is(err, ErrInvalidParams) {
		t.Fatalf("ListEvents() error = %v, want %v", err, ErrInvalidParams)
	}
}

func TestCreateEventReturnsErrInvalidParamsWhenTitleEmpty(t *testing.T) {
	store := &outlookCalendarStore{executor: &fakeCOMExecutor{started: true}}
	now := time.Date(2026, time.May, 16, 9, 0, 0, 0, time.UTC)

	_, err := store.CreateEvent(context.Background(), CreateEventParams{Start: now, End: now.Add(time.Hour)})

	if !errors.Is(err, ErrInvalidParams) {
		t.Fatalf("CreateEvent() error = %v, want %v", err, ErrInvalidParams)
	}
}

func TestCreateEventReturnsErrInvalidParamsWhenStartZero(t *testing.T) {
	store := &outlookCalendarStore{executor: &fakeCOMExecutor{started: true}}
	end := time.Date(2026, time.May, 16, 10, 0, 0, 0, time.UTC)

	_, err := store.CreateEvent(context.Background(), CreateEventParams{Title: "Focus time", End: end})

	if !errors.Is(err, ErrInvalidParams) {
		t.Fatalf("CreateEvent() error = %v, want %v", err, ErrInvalidParams)
	}
}

func TestCreateEventReturnsErrInvalidParamsWhenEndZero(t *testing.T) {
	store := &outlookCalendarStore{executor: &fakeCOMExecutor{started: true}}
	start := time.Date(2026, time.May, 16, 9, 0, 0, 0, time.UTC)

	_, err := store.CreateEvent(context.Background(), CreateEventParams{Title: "Focus time", Start: start})

	if !errors.Is(err, ErrInvalidParams) {
		t.Fatalf("CreateEvent() error = %v, want %v", err, ErrInvalidParams)
	}
}

func TestCreateEventReturnsErrInvalidParamsWhenEndNotAfterStart(t *testing.T) {
	store := &outlookCalendarStore{executor: &fakeCOMExecutor{started: true}}
	start := time.Date(2026, time.May, 16, 9, 0, 0, 0, time.UTC)

	_, err := store.CreateEvent(context.Background(), CreateEventParams{Title: "Focus time", Start: start, End: start})

	if !errors.Is(err, ErrInvalidParams) {
		t.Fatalf("CreateEvent() error = %v, want %v", err, ErrInvalidParams)
	}
}

func TestMapAppointmentRecordMapsAllFields(t *testing.T) {
	start := time.Date(2026, time.May, 16, 14, 0, 0, 0, time.UTC)
	end := start.Add(45 * time.Minute)
	record := appointmentRecord{
		ID:       "event-123",
		Title:    "Architecture review",
		Start:    start,
		End:      end,
		Location: "Room 5",
		Body:     "Plain text notes",
	}

	event := mapAppointmentRecord(record)

	if event.ID != record.ID || event.Title != record.Title || event.Location != record.Location || event.Body != record.Body {
		t.Fatalf("mapAppointmentRecord() returned %+v, want core fields from %+v", event, record)
	}
	if !event.Start.Equal(start) {
		t.Fatalf("Start = %v, want %v", event.Start, start)
	}
	if !event.End.Equal(end) {
		t.Fatalf("End = %v, want %v", event.End, end)
	}
}

func TestNormalizeListEventsParamsAppliesDefaults(t *testing.T) {
	now := time.Date(2026, time.May, 16, 15, 4, 5, 0, time.UTC)

	params := normalizeListEventsParams(ListEventsParams{Start: now, End: now.Add(defaultCalendarListWindow)}, now)

	if !params.Start.Equal(now) {
		t.Fatalf("Start = %v, want %v", params.Start, now)
	}
	wantUntil := now.Add(defaultCalendarListWindow)
	if !params.End.Equal(wantUntil) {
		t.Fatalf("End = %v, want %v", params.End, wantUntil)
	}
	if params.MaxResults != defaultCalendarListMaxResults {
		t.Fatalf("MaxResults = %d, want %d", params.MaxResults, defaultCalendarListMaxResults)
	}

	explicitUntil := now.Add(2 * time.Hour)
	params = normalizeListEventsParams(ListEventsParams{Start: now, End: explicitUntil, MaxResults: 7}, now.Add(-time.Hour))
	if !params.Start.Equal(now) || !params.End.Equal(explicitUntil) || params.MaxResults != 7 {
		t.Fatalf("normalizeListEventsParams() = %+v, want explicit values preserved", params)
	}
}
