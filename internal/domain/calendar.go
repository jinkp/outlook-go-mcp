package domain

import "time"

type CalendarEvent struct {
	ID       string
	Title    string
	Start    time.Time
	End      time.Time
	Location string
	Body     string
}

type ListEventsParams struct {
	Start      time.Time
	End        time.Time
	MaxResults int
}

type CreateEventParams struct {
	Title    string
	Start    time.Time
	End      time.Time
	Location string
	Body     string
}
