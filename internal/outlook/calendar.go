//go:build windows

package outlook

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/go-ole/go-ole"
	"github.com/go-ole/go-ole/oleutil"
)

const (
	olAppointmentItem = 1
	olFolderCalendar  = 9
	olNonMeeting      = 0
)

func calendarStoreSession(executor *COMExecutor) OutlookSession {
	if executor == nil {
		return nil
	}
	return executor.session
}

func (s *outlookCalendarStore) ListEvents(ctx context.Context, params ListEventsParams) ([]CalendarEvent, error) {
	if err := validateListEventsParams(params); err != nil {
		return nil, err
	}

	params = normalizeListEventsParams(params, time.Now())

	results := make([]CalendarEvent, 0, params.MaxResults)
	err := s.submit(ctx, func() error {
		session, err := s.connectedSession()
		if err != nil {
			return err
		}

		folder, err := dispatchCall(session.mapi, "GetDefaultFolder", olFolderCalendar)
		if err != nil {
			return wrapCOMError("get calendar folder", err)
		}
		defer folder.Release()

		items, err := dispatchProperty(folder, "Items")
		if err != nil {
			return err
		}
		defer items.Release()

		if err := putProperty(items, "IncludeRecurrences", true); err != nil {
			return err
		}
		if _, err := oleutil.CallMethod(items, "Sort", "[Start]"); err != nil {
			return wrapCOMError("sort calendar items", err)
		}

		filter := buildCalendarRangeFilter(params.Start, params.End)
		restricted, err := dispatchCall(items, "Restrict", filter)
		if err != nil {
			return wrapCOMError("restrict calendar items", err)
		}
		defer restricted.Release()

		count, err := intProperty(restricted, "Count")
		if err != nil {
			return err
		}

		for i := 1; i <= count && len(results) < params.MaxResults; i++ {
			item, err := dispatchIndexedProperty(restricted, "Item", i)
			if err != nil {
				continue
			}

			record, mapErr := mapAppointmentSummary(item)
			item.Release()
			if mapErr != nil {
				continue
			}

			results = append(results, mapAppointmentRecord(record))
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return results, nil
}

func (s *outlookCalendarStore) GetEvent(ctx context.Context, id string) (*CalendarEvent, error) {
	if strings.TrimSpace(id) == "" {
		return nil, fmt.Errorf("%w: event id is required", ErrInvalidParams)
	}

	var event *CalendarEvent
	err := s.submit(ctx, func() error {
		session, err := s.connectedSession()
		if err != nil {
			return err
		}

		item, err := getAppointmentItemByID(session, id)
		if err != nil {
			return err
		}
		defer item.Release()

		record, err := mapAppointmentDetails(item)
		if err != nil {
			return err
		}

		mapped := mapAppointmentRecord(record)
		event = &mapped
		return nil
	})
	if err != nil {
		return nil, err
	}

	return event, nil
}

func (s *outlookCalendarStore) CreateEvent(ctx context.Context, params CreateEventParams) (*CalendarEvent, error) {
	if err := validateCreateEventParams(params); err != nil {
		return nil, err
	}

	var event *CalendarEvent
	err := s.submit(ctx, func() error {
		session, err := s.connectedSession()
		if err != nil {
			return err
		}

		item, err := dispatchCall(session.ole, "CreateItem", olAppointmentItem)
		if err != nil {
			return wrapCOMError("create appointment item", err)
		}
		defer item.Release()

		_ = putProperty(item, "BodyFormat", olFormatPlain)
		if err := putProperty(item, "MeetingStatus", olNonMeeting); err != nil {
			return err
		}
		if err := putProperty(item, "Subject", params.Title); err != nil {
			return err
		}
		if err := putProperty(item, "Start", params.Start); err != nil {
			return err
		}
		if err := putProperty(item, "End", params.End); err != nil {
			return err
		}
		if err := putProperty(item, "Location", params.Location); err != nil {
			return err
		}
		if err := putProperty(item, "Body", params.Body); err != nil {
			return err
		}
		if _, err := oleutil.CallMethod(item, "Save"); err != nil {
			return wrapCOMError("save appointment", err)
		}

		record, err := mapAppointmentDetails(item)
		if err != nil {
			return err
		}

		mapped := mapAppointmentRecord(record)
		event = &mapped
		return nil
	})
	if err != nil {
		return nil, err
	}

	return event, nil
}

func (s *outlookCalendarStore) connectedSession() (*outlookSession, error) {
	session, ok := s.session.(*outlookSession)
	if !ok || session == nil || !session.IsConnected() || session.mapi == nil || session.ole == nil {
		return nil, ErrNotConnected
	}
	return session, nil
}

func buildCalendarRangeFilter(since, until time.Time) string {
	return fmt.Sprintf("[Start] <= '%s' AND [End] >= '%s'", formatOutlookTime(until), formatOutlookTime(since))
}

func mapAppointmentSummary(item *ole.IDispatch) (appointmentRecord, error) {
	id, err := stringProperty(item, "EntryID")
	if err != nil {
		return appointmentRecord{}, err
	}

	start, err := timeProperty(item, "Start")
	if err != nil {
		return appointmentRecord{}, err
	}
	end, err := timeProperty(item, "End")
	if err != nil {
		return appointmentRecord{}, err
	}

	title, _ := stringProperty(item, "Subject")
	location, _ := stringProperty(item, "Location")

	return appointmentRecord{
		ID:       id,
		Title:    title,
		Start:    start,
		End:      end,
		Location: location,
	}, nil
}

func mapAppointmentDetails(item *ole.IDispatch) (appointmentRecord, error) {
	record, err := mapAppointmentSummary(item)
	if err != nil {
		return appointmentRecord{}, err
	}

	record.Body, _ = stringProperty(item, "Body")
	return record, nil
}

func getAppointmentItemByID(session *outlookSession, id string) (*ole.IDispatch, error) {
	item, err := dispatchCall(session.mapi, "GetItemFromID", id)
	if err != nil {
		return nil, fmt.Errorf("%w: event %q", ErrNotFound, id)
	}
	if item == nil {
		return nil, fmt.Errorf("%w: event %q", ErrNotFound, id)
	}
	return item, nil
}
