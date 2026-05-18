package outlook

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestSearchEmailsReturnsErrInvalidParamsWhenQueryEmpty(t *testing.T) {
	store := &outlookMailStore{executor: &fakeCOMExecutor{started: true}}

	_, err := store.SearchEmails(context.Background(), SearchEmailsParams{Query: "   "})

	if !errors.Is(err, ErrInvalidParams) {
		t.Fatalf("SearchEmails() error = %v, want %v", err, ErrInvalidParams)
	}
}

func TestCreateDraftReturnsErrInvalidParamsWhenToEmpty(t *testing.T) {
	store := &outlookMailStore{executor: &fakeCOMExecutor{started: true}}

	_, err := store.CreateDraft(context.Background(), CreateDraftParams{Subject: "Subject", Body: "Body"})

	if !errors.Is(err, ErrInvalidParams) {
		t.Fatalf("CreateDraft() error = %v, want %v", err, ErrInvalidParams)
	}
}

func TestCreateDraftReturnsErrInvalidParamsWhenSubjectEmpty(t *testing.T) {
	store := &outlookMailStore{executor: &fakeCOMExecutor{started: true}}

	_, err := store.CreateDraft(context.Background(), CreateDraftParams{To: []string{"dev@example.com"}, Subject: "  ", Body: "Body"})

	if !errors.Is(err, ErrInvalidParams) {
		t.Fatalf("CreateDraft() error = %v, want %v", err, ErrInvalidParams)
	}
}

func TestSearchEmailsReturnsErrInvalidParamsWhenDateRangeIsReversed(t *testing.T) {
	store := &outlookMailStore{executor: &fakeCOMExecutor{started: true}}
	since := time.Date(2026, time.May, 17, 0, 0, 0, 0, time.UTC)
	until := time.Date(2026, time.May, 16, 0, 0, 0, 0, time.UTC)

	_, err := store.SearchEmails(context.Background(), SearchEmailsParams{Query: "kubernetes", Since: since, Until: until})

	if !errors.Is(err, ErrInvalidParams) {
		t.Fatalf("SearchEmails() error = %v, want %v", err, ErrInvalidParams)
	}
}

func TestMailStoreReturnsErrNotConnectedWhenExecutorNotStarted(t *testing.T) {
	tests := []struct {
		name string
		call func(store *outlookMailStore) error
	}{
		{
			name: "search emails",
			call: func(store *outlookMailStore) error {
				_, err := store.SearchEmails(context.Background(), SearchEmailsParams{Query: "kubernetes"})
				return err
			},
		},
		{
			name: "get email",
			call: func(store *outlookMailStore) error {
				_, err := store.GetEmail(context.Background(), "mail-id")
				return err
			},
		},
		{
			name: "list attachments",
			call: func(store *outlookMailStore) error {
				_, err := store.ListAttachments(context.Background(), ListAttachmentsParams{EmailID: "mail-id"})
				return err
			},
		},
		{
			name: "create draft",
			call: func(store *outlookMailStore) error {
				_, err := store.CreateDraft(context.Background(), CreateDraftParams{To: []string{"dev@example.com"}, Subject: "Subject", Body: "Body"})
				return err
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &outlookMailStore{executor: &fakeCOMExecutor{started: false}}

			err := tt.call(store)

			if !errors.Is(err, ErrNotConnected) {
				t.Fatalf("error = %v, want %v", err, ErrNotConnected)
			}
		})
	}
}

func TestMapMailRecordToEmailMapsAllFields(t *testing.T) {
	receivedAt := time.Date(2026, time.May, 16, 10, 30, 0, 0, time.UTC)
	record := mailRecord{
		ID:      "mail-123",
		Subject: "Status update",
		Body:    "Plain text body",
		From:    "alice@example.com",
		To:      []string{"bob@example.com", "carol@example.com"},
		CC:      []string{"dave@example.com"},
		Date:    receivedAt,
		Attachments: []attachmentRecord{
			{ID: "att-1", Name: "report.pdf", Size: 2048},
		},
	}

	email := mapMailRecordToEmail(record)

	if email.ID != record.ID || email.Subject != record.Subject || email.Body != record.Body || email.From != record.From {
		t.Fatalf("mapMailRecordToEmail() returned %+v, want core fields from %+v", email, record)
	}
	if len(email.To) != 2 || email.To[0] != "bob@example.com" || email.To[1] != "carol@example.com" {
		t.Fatalf("To = %#v, want %#v", email.To, record.To)
	}
	if len(email.CC) != 1 || email.CC[0] != "dave@example.com" {
		t.Fatalf("CC = %#v, want %#v", email.CC, record.CC)
	}
	if !email.Date.Equal(receivedAt) {
		t.Fatalf("Date = %v, want %v", email.Date, receivedAt)
	}
	if !email.HasAttachments {
		t.Fatal("HasAttachments = false, want true")
	}
	if len(email.Attachments) != 1 {
		t.Fatalf("len(Attachments) = %d, want 1", len(email.Attachments))
	}
	if email.Attachments[0].ContentType != "application/pdf" {
		t.Fatalf("Attachments[0].ContentType = %q, want %q", email.Attachments[0].ContentType, "application/pdf")
	}
}

func TestMapMailRecordToEmailHandlesEmailsWithoutAttachments(t *testing.T) {
	record := mailRecord{
		ID:      "mail-456",
		Subject: "No attachments",
		From:    "alice@example.com",
		To:      []string{"bob@example.com"},
	}

	email := mapMailRecordToEmail(record)

	if email.HasAttachments {
		t.Fatal("HasAttachments = true, want false")
	}
	if len(email.Attachments) != 0 {
		t.Fatalf("len(Attachments) = %d, want 0", len(email.Attachments))
	}
}

func TestNormalizeMailSearchMaxResultsDefaultsToTwenty(t *testing.T) {
	if got := normalizeMailSearchMaxResults(0); got != 20 {
		t.Fatalf("normalizeMailSearchMaxResults(0) = %d, want 20", got)
	}
	if got := normalizeMailSearchMaxResults(7); got != 7 {
		t.Fatalf("normalizeMailSearchMaxResults(7) = %d, want 7", got)
	}
}

func TestReplyDraftReturnsErrNotConnectedWhenExecutorNotStarted(t *testing.T) {
	store := &outlookMailStore{executor: &fakeCOMExecutor{started: false}}

	_, err := store.ReplyDraft(context.Background(), ReplyDraftParams{EmailID: "id-1", Body: "body"})

	if !errors.Is(err, ErrNotConnected) {
		t.Fatalf("ReplyDraft() error = %v, want %v", err, ErrNotConnected)
	}
}

func TestReplyDraftReturnsErrInvalidParamsWhenEmailIDEmpty(t *testing.T) {
	store := &outlookMailStore{executor: &fakeCOMExecutor{started: true}}

	_, err := store.ReplyDraft(context.Background(), ReplyDraftParams{EmailID: "", Body: "body"})

	if !errors.Is(err, ErrInvalidParams) {
		t.Fatalf("ReplyDraft() error = %v, want %v", err, ErrInvalidParams)
	}
}

func TestReplyDraftReturnsErrInvalidParamsWhenBodyEmpty(t *testing.T) {
	store := &outlookMailStore{executor: &fakeCOMExecutor{started: true}}

	_, err := store.ReplyDraft(context.Background(), ReplyDraftParams{EmailID: "id-1", Body: "   "})

	if !errors.Is(err, ErrInvalidParams) {
		t.Fatalf("ReplyDraft() error = %v, want %v", err, ErrInvalidParams)
	}
}

func TestForwardDraftReturnsErrNotConnectedWhenExecutorNotStarted(t *testing.T) {
	store := &outlookMailStore{executor: &fakeCOMExecutor{started: false}}

	_, err := store.ForwardDraft(context.Background(), ForwardDraftParams{EmailID: "id-1", To: []string{"a@b.com"}})

	if !errors.Is(err, ErrNotConnected) {
		t.Fatalf("ForwardDraft() error = %v, want %v", err, ErrNotConnected)
	}
}

func TestForwardDraftReturnsErrInvalidParamsWhenEmailIDEmpty(t *testing.T) {
	store := &outlookMailStore{executor: &fakeCOMExecutor{started: true}}

	_, err := store.ForwardDraft(context.Background(), ForwardDraftParams{EmailID: "", To: []string{"a@b.com"}})

	if !errors.Is(err, ErrInvalidParams) {
		t.Fatalf("ForwardDraft() error = %v, want %v", err, ErrInvalidParams)
	}
}

func TestForwardDraftReturnsErrInvalidParamsWhenToEmpty(t *testing.T) {
	store := &outlookMailStore{executor: &fakeCOMExecutor{started: true}}

	_, err := store.ForwardDraft(context.Background(), ForwardDraftParams{EmailID: "id-1", To: []string{}})

	if !errors.Is(err, ErrInvalidParams) {
		t.Fatalf("ForwardDraft() error = %v, want %v", err, ErrInvalidParams)
	}
}

func TestMarkReadReturnsErrNotConnectedWhenExecutorNotStarted(t *testing.T) {
	store := &outlookMailStore{executor: &fakeCOMExecutor{started: false}}

	err := store.MarkRead(context.Background(), MarkReadParams{EmailID: "id-1", Read: true})

	if !errors.Is(err, ErrNotConnected) {
		t.Fatalf("MarkRead() error = %v, want %v", err, ErrNotConnected)
	}
}

func TestMarkReadReturnsErrInvalidParamsWhenEmailIDEmpty(t *testing.T) {
	store := &outlookMailStore{executor: &fakeCOMExecutor{started: true}}

	err := store.MarkRead(context.Background(), MarkReadParams{EmailID: "  ", Read: true})

	if !errors.Is(err, ErrInvalidParams) {
		t.Fatalf("MarkRead() error = %v, want %v", err, ErrInvalidParams)
	}
}

func TestFlagEmailReturnsErrNotConnectedWhenExecutorNotStarted(t *testing.T) {
	store := &outlookMailStore{executor: &fakeCOMExecutor{started: false}}

	err := store.FlagEmail(context.Background(), FlagEmailParams{EmailID: "id-1", Flagged: true})

	if !errors.Is(err, ErrNotConnected) {
		t.Fatalf("FlagEmail() error = %v, want %v", err, ErrNotConnected)
	}
}

func TestFlagEmailReturnsErrInvalidParamsWhenEmailIDEmpty(t *testing.T) {
	store := &outlookMailStore{executor: &fakeCOMExecutor{started: true}}

	err := store.FlagEmail(context.Background(), FlagEmailParams{EmailID: "", Flagged: true})

	if !errors.Is(err, ErrInvalidParams) {
		t.Fatalf("FlagEmail() error = %v, want %v", err, ErrInvalidParams)
	}
}

func TestMoveEmailReturnsErrNotConnectedWhenExecutorNotStarted(t *testing.T) {
	store := &outlookMailStore{executor: &fakeCOMExecutor{started: false}}

	err := store.MoveEmail(context.Background(), MoveEmailParams{EmailID: "id-1", Folder: "Archive"})

	if !errors.Is(err, ErrNotConnected) {
		t.Fatalf("MoveEmail() error = %v, want %v", err, ErrNotConnected)
	}
}

func TestMoveEmailReturnsErrInvalidParamsWhenEmailIDEmpty(t *testing.T) {
	store := &outlookMailStore{executor: &fakeCOMExecutor{started: true}}

	err := store.MoveEmail(context.Background(), MoveEmailParams{EmailID: "", Folder: "Archive"})

	if !errors.Is(err, ErrInvalidParams) {
		t.Fatalf("MoveEmail() error = %v, want %v", err, ErrInvalidParams)
	}
}

func TestMoveEmailReturnsErrInvalidParamsWhenFolderEmpty(t *testing.T) {
	store := &outlookMailStore{executor: &fakeCOMExecutor{started: true}}

	err := store.MoveEmail(context.Background(), MoveEmailParams{EmailID: "id-1", Folder: "  "})

	if !errors.Is(err, ErrInvalidParams) {
		t.Fatalf("MoveEmail() error = %v, want %v", err, ErrInvalidParams)
	}
}

func TestListFoldersReturnsErrNotConnectedWhenExecutorNotStarted(t *testing.T) {
	store := &outlookMailStore{executor: &fakeCOMExecutor{started: false}}

	_, err := store.ListFolders(context.Background())

	if !errors.Is(err, ErrNotConnected) {
		t.Fatalf("ListFolders() error = %v, want %v", err, ErrNotConnected)
	}
}

func TestDownloadAttachmentReturnsErrNotConnectedWhenExecutorNotStarted(t *testing.T) {
	store := &outlookMailStore{executor: &fakeCOMExecutor{started: false}}

	_, err := store.DownloadAttachment(context.Background(), DownloadAttachmentParams{EmailID: "id-1", AttachmentID: "1", DestDir: "C:\\tmp"})

	if !errors.Is(err, ErrNotConnected) {
		t.Fatalf("DownloadAttachment() error = %v, want %v", err, ErrNotConnected)
	}
}

func TestDownloadAttachmentReturnsErrInvalidParamsWhenEmailIDEmpty(t *testing.T) {
	store := &outlookMailStore{executor: &fakeCOMExecutor{started: true}}

	_, err := store.DownloadAttachment(context.Background(), DownloadAttachmentParams{EmailID: "", AttachmentID: "1", DestDir: "C:\\tmp"})

	if !errors.Is(err, ErrInvalidParams) {
		t.Fatalf("DownloadAttachment() error = %v, want %v", err, ErrInvalidParams)
	}
}

func TestDownloadAttachmentReturnsErrInvalidParamsWhenAttachmentIDEmpty(t *testing.T) {
	store := &outlookMailStore{executor: &fakeCOMExecutor{started: true}}

	_, err := store.DownloadAttachment(context.Background(), DownloadAttachmentParams{EmailID: "id-1", AttachmentID: "", DestDir: "C:\\tmp"})

	if !errors.Is(err, ErrInvalidParams) {
		t.Fatalf("DownloadAttachment() error = %v, want %v", err, ErrInvalidParams)
	}
}

func TestDeleteEmailReturnsErrNotConnectedWhenExecutorNotStarted(t *testing.T) {
	store := &outlookMailStore{executor: &fakeCOMExecutor{started: false}}

	err := store.DeleteEmail(context.Background(), "id-1")

	if !errors.Is(err, ErrNotConnected) {
		t.Fatalf("DeleteEmail() error = %v, want %v", err, ErrNotConnected)
	}
}

func TestDeleteEmailReturnsErrInvalidParamsWhenIDEmpty(t *testing.T) {
	store := &outlookMailStore{executor: &fakeCOMExecutor{started: true}}

	err := store.DeleteEmail(context.Background(), "   ")

	if !errors.Is(err, ErrInvalidParams) {
		t.Fatalf("DeleteEmail() error = %v, want %v", err, ErrInvalidParams)
	}
}

func TestListEmailsInRangeReturnsErrInvalidParamsWhenSinceIsZero(t *testing.T) {
	store := &outlookMailStore{executor: &fakeCOMExecutor{started: true}}

	_, err := store.ListEmailsInRange(context.Background(), ListEmailsInRangeParams{
		Since:      time.Time{}, // zero
		MaxResults: 10,
	})

	if !errors.Is(err, ErrInvalidParams) {
		t.Fatalf("ListEmailsInRange() error = %v, want %v", err, ErrInvalidParams)
	}
}

func TestListEmailsInRangeReturnsErrInvalidParamsWhenDateRangeIsReversed(t *testing.T) {
	store := &outlookMailStore{executor: &fakeCOMExecutor{started: true}}
	since := time.Date(2026, time.May, 18, 0, 0, 0, 0, time.UTC)
	until := time.Date(2026, time.May, 17, 0, 0, 0, 0, time.UTC) // before since

	_, err := store.ListEmailsInRange(context.Background(), ListEmailsInRangeParams{
		Since:      since,
		Until:      until,
		MaxResults: 10,
	})

	if !errors.Is(err, ErrInvalidParams) {
		t.Fatalf("ListEmailsInRange() error = %v, want %v", err, ErrInvalidParams)
	}
}

func TestListEmailsInRangeReturnsErrNotConnectedWhenExecutorNotStarted(t *testing.T) {
	store := &outlookMailStore{executor: &fakeCOMExecutor{started: false}}
	since := time.Date(2026, time.May, 17, 0, 0, 0, 0, time.UTC)

	_, err := store.ListEmailsInRange(context.Background(), ListEmailsInRangeParams{
		Since:      since,
		MaxResults: 10,
	})

	if !errors.Is(err, ErrNotConnected) {
		t.Fatalf("ListEmailsInRange() error = %v, want %v", err, ErrNotConnected)
	}
}

type fakeCOMExecutor struct{ started bool }

func (f *fakeCOMExecutor) Submit(ctx context.Context, fn func() error) error {
	if !f.started {
		return ErrNotConnected
	}
	return fn()
}
