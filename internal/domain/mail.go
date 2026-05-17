package domain

import "time"

type Email struct {
	ID             string
	Subject        string
	Body           string
	From           string
	To             []string
	CC             []string
	Date           time.Time
	HasAttachments bool
	Attachments    []Attachment
}

type Attachment struct {
	ID          string
	Name        string
	Size        int64
	ContentType string
}

type SearchEmailsParams struct {
	Query      string
	Folder     string
	Since      time.Time
	Until      time.Time
	MaxResults int
}

type CreateDraftParams struct {
	To      []string
	Subject string
	Body    string
}

type ListAttachmentsParams struct {
	EmailID string
}
