//go:build !windows

package outlook

import (
	"context"
	"strings"
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
