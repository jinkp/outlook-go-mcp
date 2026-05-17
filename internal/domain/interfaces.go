package domain

import "context"

type MailStore interface {
	SearchEmails(ctx context.Context, params SearchEmailsParams) ([]Email, error)
	GetEmail(ctx context.Context, id string) (*Email, error)
	ListAttachments(ctx context.Context, params ListAttachmentsParams) ([]Attachment, error)
	CreateDraft(ctx context.Context, params CreateDraftParams) (*Email, error)
}

type CalendarStore interface {
	ListEvents(ctx context.Context, params ListEventsParams) ([]CalendarEvent, error)
	GetEvent(ctx context.Context, id string) (*CalendarEvent, error)
	CreateEvent(ctx context.Context, params CreateEventParams) (*CalendarEvent, error)
}
