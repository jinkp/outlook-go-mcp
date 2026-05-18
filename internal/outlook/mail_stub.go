//go:build !windows

package outlook

import (
	"context"
	"strings"

	"github.com/jinkp/outlook-go-mcp/internal/domain"
)

func mailStoreSession(executor *COMExecutor) OutlookSession {
	return nil
}

func (s *outlookMailStore) SearchEmails(ctx context.Context, params SearchEmailsParams) ([]Email, error) {
	if err := validateSearchEmailsParams(params); err != nil {
		return nil, err
	}

	if err := s.submit(ctx, func() error { return ErrNotConnected }); err != nil {
		return nil, err
	}

	return nil, ErrNotConnected
}

func (s *outlookMailStore) GetEmail(ctx context.Context, id string) (*Email, error) {
	if err := s.submit(ctx, func() error { return ErrNotConnected }); err != nil {
		return nil, err
	}

	return nil, ErrNotConnected
}

func (s *outlookMailStore) ListAttachments(ctx context.Context, params ListAttachmentsParams) ([]Attachment, error) {
	if strings.TrimSpace(params.EmailID) == "" {
		return nil, ErrInvalidParams
	}
	if err := s.submit(ctx, func() error { return ErrNotConnected }); err != nil {
		return nil, err
	}

	return nil, ErrNotConnected
}

func (s *outlookMailStore) CreateDraft(ctx context.Context, params CreateDraftParams) (*Email, error) {
	if err := validateCreateDraftParams(params); err != nil {
		return nil, err
	}

	if err := s.submit(ctx, func() error { return ErrNotConnected }); err != nil {
		return nil, err
	}

	return nil, ErrNotConnected
}

func (s *outlookMailStore) ReplyDraft(ctx context.Context, params domain.ReplyDraftParams) (*Email, error) {
	if err := s.submit(ctx, func() error { return ErrNotConnected }); err != nil {
		return nil, err
	}
	return nil, ErrNotConnected
}

func (s *outlookMailStore) ForwardDraft(ctx context.Context, params domain.ForwardDraftParams) (*Email, error) {
	if err := s.submit(ctx, func() error { return ErrNotConnected }); err != nil {
		return nil, err
	}
	return nil, ErrNotConnected
}

func (s *outlookMailStore) MarkRead(ctx context.Context, params domain.MarkReadParams) error {
	if err := s.submit(ctx, func() error { return ErrNotConnected }); err != nil {
		return err
	}
	return ErrNotConnected
}

func (s *outlookMailStore) FlagEmail(ctx context.Context, params domain.FlagEmailParams) error {
	if err := s.submit(ctx, func() error { return ErrNotConnected }); err != nil {
		return err
	}
	return ErrNotConnected
}

func (s *outlookMailStore) MoveEmail(ctx context.Context, params domain.MoveEmailParams) error {
	if err := s.submit(ctx, func() error { return ErrNotConnected }); err != nil {
		return err
	}
	return ErrNotConnected
}

func (s *outlookMailStore) ListFolders(ctx context.Context) ([]domain.MailFolder, error) {
	if err := s.submit(ctx, func() error { return ErrNotConnected }); err != nil {
		return nil, err
	}
	return nil, ErrNotConnected
}

func (s *outlookMailStore) DownloadAttachment(ctx context.Context, params domain.DownloadAttachmentParams) (*domain.DownloadedAttachment, error) {
	if err := s.submit(ctx, func() error { return ErrNotConnected }); err != nil {
		return nil, err
	}
	return nil, ErrNotConnected
}

func (s *outlookMailStore) DeleteEmail(ctx context.Context, id string) error {
	if err := s.submit(ctx, func() error { return ErrNotConnected }); err != nil {
		return err
	}
	return ErrNotConnected
}

func (s *outlookMailStore) ListEmailsInRange(ctx context.Context, params domain.ListEmailsInRangeParams) ([]domain.Email, error) {
	if err := s.submit(ctx, func() error { return ErrNotConnected }); err != nil {
		return nil, err
	}
	return nil, ErrNotConnected
}
