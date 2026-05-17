//go:build !windows

package outlook

import (
	"context"
	"fmt"
	"strings"
	"time"
)

const (
	olAppointmentItem = 1
	olFolderCalendar  = 9
	olNonMeeting      = 0
)

func calendarStoreSession(executor *COMExecutor) OutlookSession {
	return nil
}

func (s *outlookCalendarStore) ListEvents(ctx context.Context, params ListEventsParams) ([]CalendarEvent, error) {
	if err := validateListEventsParams(params); err != nil {
		return nil, err
	}

	params = normalizeListEventsParams(params, nowUTC())

	if err := s.submit(ctx, func() error { return ErrNotConnected }); err != nil {
		return nil, err
	}

	return nil, ErrNotConnected
}

func (s *outlookCalendarStore) GetEvent(ctx context.Context, id string) (*CalendarEvent, error) {
	if strings.TrimSpace(id) == "" {
		return nil, fmt.Errorf("%w: event id is required", ErrInvalidParams)
	}

	if err := s.submit(ctx, func() error { return ErrNotConnected }); err != nil {
		return nil, err
	}

	return nil, ErrNotConnected
}

func (s *outlookCalendarStore) CreateEvent(ctx context.Context, params CreateEventParams) (*CalendarEvent, error) {
	if err := validateCreateEventParams(params); err != nil {
		return nil, err
	}

	if err := s.submit(ctx, func() error { return ErrNotConnected }); err != nil {
		return nil, err
	}

	return nil, ErrNotConnected
}

func nowUTC() time.Time {
	return time.Now().UTC()
}
