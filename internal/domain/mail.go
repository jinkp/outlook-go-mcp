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

type ReplyDraftParams struct {
	EmailID string
	Body    string
}

type ForwardDraftParams struct {
	EmailID string
	To      []string
	Body    string
}

type MarkReadParams struct {
	EmailID string
	Read    bool
}

type FlagEmailParams struct {
	EmailID string
	Flagged bool
}

type MoveEmailParams struct {
	EmailID string
	Folder  string
}

type DownloadAttachmentParams struct {
	EmailID      string
	AttachmentID string
	DestDir      string
}

type DownloadedAttachment struct {
	Name string
	Path string
	Size int64
}

type MailFolder struct {
	Name          string
	EntryID       string
	ParentEntryID string
	FolderType    int
}

type ListEmailsInRangeParams struct {
	Since      time.Time
	Until      time.Time
	MaxResults int
}
