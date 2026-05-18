package outlook

import (
	"context"
	"fmt"
	"mime"
	"path/filepath"
	"strings"
	"time"
)

const (
	olMailItem    = 0
	olFormatPlain = 1
	olFolderInbox = 6
	olFolderSentMail = 5

	defaultMailSearchMaxResults = 20

	olFlagNone   = 0
	olFlagMarked = 2
)

type mailSubmitter interface {
	Submit(ctx context.Context, fn func() error) error
}

type outlookMailStore struct {
	executor mailSubmitter
	session  OutlookSession
}

type mailRecord struct {
	ID             string
	Subject        string
	Body           string
	From           string
	To             []string
	CC             []string
	Date           time.Time
	HasAttachments bool
	Attachments    []attachmentRecord
}

type attachmentRecord struct {
	ID          string
	Name        string
	Size        int64
	ContentType string
}

func NewMailStore(executor *COMExecutor) MailStore {
	return &outlookMailStore{
		executor: executor,
		session:  mailStoreSession(executor),
	}
}

func (s *outlookMailStore) submit(ctx context.Context, fn func() error) error {
	if s.executor == nil {
		return ErrNotConnected
	}

	return s.executor.Submit(ctx, fn)
}

func validateSearchEmailsParams(params SearchEmailsParams) error {
	if strings.TrimSpace(params.Query) == "" {
		return fmt.Errorf("%w: query is required", ErrInvalidParams)
	}
	if !params.Since.IsZero() && !params.Until.IsZero() && params.Since.After(params.Until) {
		return fmt.Errorf("%w: since must be before until", ErrInvalidParams)
	}
	return nil
}

func validateCreateDraftParams(params CreateDraftParams) error {
	if len(params.To) == 0 {
		return fmt.Errorf("%w: at least one recipient is required", ErrInvalidParams)
	}
	if strings.TrimSpace(params.Subject) == "" {
		return fmt.Errorf("%w: subject is required", ErrInvalidParams)
	}
	return nil
}

func normalizeMailSearchMaxResults(maxResults int) int {
	if maxResults <= 0 {
		return defaultMailSearchMaxResults
	}
	return maxResults
}

func mapMailRecordToEmail(record mailRecord) Email {
	attachments := make([]Attachment, 0, len(record.Attachments))
	for _, attachment := range record.Attachments {
		attachments = append(attachments, mapAttachmentRecord(attachment))
	}

	to := append([]string(nil), record.To...)
	hasAttachments := record.HasAttachments || len(attachments) > 0

	return Email{
		ID:             record.ID,
		Subject:        record.Subject,
		Body:           record.Body,
		From:           record.From,
		To:             to,
		CC:             append([]string(nil), record.CC...),
		Date:           record.Date,
		HasAttachments: hasAttachments,
		Attachments:    attachments,
	}
}

func mapAttachmentRecord(record attachmentRecord) Attachment {
	contentType := strings.TrimSpace(record.ContentType)
	if contentType == "" {
		contentType = inferContentType(record.Name)
	}

	return Attachment{
		ID:          record.ID,
		Name:        record.Name,
		Size:        record.Size,
		ContentType: contentType,
	}
}

func inferContentType(name string) string {
	contentType := mime.TypeByExtension(strings.ToLower(filepath.Ext(name)))
	if contentType == "" {
		return "application/octet-stream"
	}
	return contentType
}

func splitRecipients(raw string) []string {
	parts := strings.Split(raw, ";")
	recipients := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			recipients = append(recipients, part)
		}
	}
	return recipients
}
