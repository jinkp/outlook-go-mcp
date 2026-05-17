package outlook

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/jinkp/outlook-go-mcp/internal/domain"
)

var (
	_ domain.MailStore     = (*mailStoreContract)(nil)
	_ domain.CalendarStore = (*calendarStoreContract)(nil)
	_ OutlookSession       = (*sessionContract)(nil)
)

type mailStoreContract struct{}

func (mailStoreContract) SearchEmails(context.Context, SearchEmailsParams) ([]Email, error) {
	return nil, nil
}

func (mailStoreContract) GetEmail(context.Context, string) (*Email, error) {
	return nil, nil
}

func (mailStoreContract) ListAttachments(context.Context, ListAttachmentsParams) ([]Attachment, error) {
	return nil, nil
}

func (mailStoreContract) CreateDraft(context.Context, CreateDraftParams) (*Email, error) {
	return nil, nil
}

type calendarStoreContract struct{}

func (calendarStoreContract) ListEvents(context.Context, ListEventsParams) ([]CalendarEvent, error) {
	return nil, nil
}

func (calendarStoreContract) GetEvent(context.Context, string) (*CalendarEvent, error) {
	return nil, nil
}

func (calendarStoreContract) CreateEvent(context.Context, CreateEventParams) (*CalendarEvent, error) {
	return nil, nil
}

type sessionContract struct{}

func (sessionContract) Connect() error {
	return nil
}

func (sessionContract) Close() error {
	return nil
}

func (sessionContract) IsConnected() bool {
	return true
}

func TestDomainErrorsRemainDistinct(t *testing.T) {
	tests := []struct {
		name   string
		target error
		other  error
	}{
		{name: "not found", target: ErrNotFound, other: ErrPolicyDenied},
		{name: "not connected", target: ErrNotConnected, other: ErrCOMFailure},
		{name: "invalid params", target: ErrInvalidParams, other: ErrNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wrapped := fmt.Errorf("wrapped: %w", tt.target)

			if !errors.Is(wrapped, tt.target) {
				t.Fatalf("errors.Is(%v, %v) = false, want true", wrapped, tt.target)
			}

			if errors.Is(wrapped, tt.other) {
				t.Fatalf("errors.Is(%v, %v) = true, want false", wrapped, tt.other)
			}
		})
	}
}
