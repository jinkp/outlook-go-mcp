package outlook

import (
	"context"
	"fmt"
	"strings"
	"time"
)

const (
	defaultCalendarListMaxResults = 20
	defaultCalendarListWindow     = 30 * 24 * time.Hour
)

type outlookCalendarStore struct {
	executor mailSubmitter
	session  OutlookSession
}

type appointmentRecord struct {
	ID       string
	Title    string
	Start    time.Time
	End      time.Time
	Location string
	Body     string
}

func NewCalendarStore(executor *COMExecutor) CalendarStore {
	return &outlookCalendarStore{
		executor: executor,
		session:  calendarStoreSession(executor),
	}
}

func (s *outlookCalendarStore) submit(ctx context.Context, fn func() error) error {
	if s.executor == nil {
		return ErrNotConnected
	}

	return s.executor.Submit(ctx, fn)
}

func validateCreateEventParams(params CreateEventParams) error {
	if strings.TrimSpace(params.Title) == "" {
		return fmt.Errorf("%w: title is required", ErrInvalidParams)
	}
	if params.Start.IsZero() {
		return fmt.Errorf("%w: start is required", ErrInvalidParams)
	}
	if params.End.IsZero() {
		return fmt.Errorf("%w: end is required", ErrInvalidParams)
	}
	if !params.End.After(params.Start) {
		return fmt.Errorf("%w: end must be after start", ErrInvalidParams)
	}
	return nil
}

func validateListEventsParams(params ListEventsParams) error {
	if params.Start.IsZero() {
		return fmt.Errorf("%w: start is required", ErrInvalidParams)
	}
	if params.End.IsZero() {
		return fmt.Errorf("%w: end is required", ErrInvalidParams)
	}
	if params.Start.After(params.End) {
		return fmt.Errorf("%w: start must be before end", ErrInvalidParams)
	}
	return nil
}

func normalizeListEventsParams(params ListEventsParams, now time.Time) ListEventsParams {
	if params.MaxResults <= 0 {
		params.MaxResults = defaultCalendarListMaxResults
	}

	return params
}

func mapAppointmentRecord(record appointmentRecord) CalendarEvent {
	return CalendarEvent{
		ID:       record.ID,
		Title:    record.Title,
		Start:    record.Start,
		End:      record.End,
		Location: record.Location,
		Body:     record.Body,
	}
}
