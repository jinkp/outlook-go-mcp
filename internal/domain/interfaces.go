package domain

import "context"

type MailStore interface {
	// Ping validates MAPI connectivity by accessing the Inbox folder name.
	// Returns nil if Outlook is reachable and the mail store is functional.
	Ping(ctx context.Context) error
	SearchEmails(ctx context.Context, params SearchEmailsParams) ([]Email, error)
	GetEmail(ctx context.Context, id string) (*Email, error)
	ListAttachments(ctx context.Context, params ListAttachmentsParams) ([]Attachment, error)
	CreateDraft(ctx context.Context, params CreateDraftParams) (*Email, error)
	ReplyDraft(ctx context.Context, params ReplyDraftParams) (*Email, error)
	ForwardDraft(ctx context.Context, params ForwardDraftParams) (*Email, error)
	MarkRead(ctx context.Context, params MarkReadParams) error
	FlagEmail(ctx context.Context, params FlagEmailParams) error
	MoveEmail(ctx context.Context, params MoveEmailParams) error
	ListFolders(ctx context.Context) ([]MailFolder, error)
	DownloadAttachment(ctx context.Context, params DownloadAttachmentParams) (*DownloadedAttachment, error)
	DeleteEmail(ctx context.Context, id string) error
	ListEmailsInRange(ctx context.Context, params ListEmailsInRangeParams) ([]Email, error)
}

type CalendarStore interface {
	ListEvents(ctx context.Context, params ListEventsParams) ([]CalendarEvent, error)
	GetEvent(ctx context.Context, id string) (*CalendarEvent, error)
	CreateEvent(ctx context.Context, params CreateEventParams) (*CalendarEvent, error)
}
