package report

import (
	"context"
	"testing"
	"time"

	"github.com/jinkp/outlook-go-mcp/internal/config"
	"github.com/jinkp/outlook-go-mcp/internal/domain"
)

// mockMailStore implements domain.MailStore for testing.
type mockMailStore struct {
	listEmailsInRangeResult []domain.Email
	listEmailsInRangeErr    error
	searchEmailsResult      []domain.Email
	searchEmailsErr         error
}

func (m *mockMailStore) SearchEmails(_ context.Context, _ domain.SearchEmailsParams) ([]domain.Email, error) {
	return m.searchEmailsResult, m.searchEmailsErr
}

func (m *mockMailStore) GetEmail(_ context.Context, _ string) (*domain.Email, error) { return nil, nil }

func (m *mockMailStore) ListAttachments(_ context.Context, _ domain.ListAttachmentsParams) ([]domain.Attachment, error) {
	return nil, nil
}

func (m *mockMailStore) CreateDraft(_ context.Context, _ domain.CreateDraftParams) (*domain.Email, error) {
	return nil, nil
}

func (m *mockMailStore) ReplyDraft(_ context.Context, _ domain.ReplyDraftParams) (*domain.Email, error) {
	return nil, nil
}

func (m *mockMailStore) ForwardDraft(_ context.Context, _ domain.ForwardDraftParams) (*domain.Email, error) {
	return nil, nil
}

func (m *mockMailStore) MarkRead(_ context.Context, _ domain.MarkReadParams) error { return nil }

func (m *mockMailStore) FlagEmail(_ context.Context, _ domain.FlagEmailParams) error { return nil }

func (m *mockMailStore) MoveEmail(_ context.Context, _ domain.MoveEmailParams) error { return nil }

func (m *mockMailStore) ListFolders(_ context.Context) ([]domain.MailFolder, error) {
	return nil, nil
}

func (m *mockMailStore) DownloadAttachment(_ context.Context, _ domain.DownloadAttachmentParams) (*domain.DownloadedAttachment, error) {
	return nil, nil
}

func (m *mockMailStore) DeleteEmail(_ context.Context, _ string) error { return nil }

func (m *mockMailStore) ListEmailsInRange(_ context.Context, _ domain.ListEmailsInRangeParams) ([]domain.Email, error) {
	return m.listEmailsInRangeResult, m.listEmailsInRangeErr
}

// mockCalendarStore implements domain.CalendarStore for testing.
type mockCalendarStore struct {
	listEventsResult []domain.CalendarEvent
	listEventsErr    error
}

func (m *mockCalendarStore) ListEvents(_ context.Context, _ domain.ListEventsParams) ([]domain.CalendarEvent, error) {
	return m.listEventsResult, m.listEventsErr
}

func (m *mockCalendarStore) GetEvent(_ context.Context, _ string) (*domain.CalendarEvent, error) {
	return nil, nil
}

func (m *mockCalendarStore) CreateEvent(_ context.Context, _ domain.CreateEventParams) (*domain.CalendarEvent, error) {
	return nil, nil
}

var fixedNow = time.Date(2026, time.May, 18, 12, 0, 0, 0, time.UTC)

func newTestEngine(mail domain.MailStore, calendar domain.CalendarStore, cfg config.ReportConfig) *Engine {
	return NewEngine(mail, calendar, cfg, nil, func() time.Time { return fixedNow })
}

func defaultTestConfig() config.ReportConfig {
	return config.ReportConfig{
		SinceHours:    24,
		MaxPerSection: 20,
	}
}

func TestRunReturnsAllSections(t *testing.T) {
	mail := &mockMailStore{
		listEmailsInRangeResult: []domain.Email{
			{ID: "1", Subject: "Budget Review", From: "alice@vip.com", HasAttachments: true, Date: fixedNow.Add(-1 * time.Hour)},
		},
		searchEmailsResult: nil, // no reply found → unanswered
	}
	calendar := &mockCalendarStore{
		listEventsResult: []domain.CalendarEvent{
			{ID: "evt1", Title: "Budget Meeting", Start: fixedNow, End: fixedNow.Add(time.Hour)},
		},
	}
	cfg := config.ReportConfig{
		SinceHours:    24,
		MaxPerSection: 20,
		VIPSenders:    []string{"@vip.com"},
	}

	engine := newTestEngine(mail, calendar, cfg)
	rpt, err := engine.Run(context.Background())

	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if rpt == nil {
		t.Fatal("Run() = nil report")
	}
	if len(rpt.Unanswered) == 0 {
		t.Error("Unanswered section is empty, want at least 1 email")
	}
	if len(rpt.VIPEmails) == 0 {
		t.Error("VIPEmails section is empty, want at least 1 email")
	}
	if len(rpt.Attachments) == 0 {
		t.Error("Attachments section is empty, want at least 1 email")
	}
	if rpt.GeneratedAt.IsZero() {
		t.Error("GeneratedAt is zero")
	}
}

func TestRunUnansweredFiltersReplied(t *testing.T) {
	// Two emails, one has a Sent match → only 1 unanswered
	mail := &mockMailStore{
		listEmailsInRangeResult: []domain.Email{
			{ID: "1", Subject: "Budget Review", From: "a@b.com", Date: fixedNow.Add(-2 * time.Hour)},
			{ID: "2", Subject: "Sync meeting", From: "c@d.com", Date: fixedNow.Add(-3 * time.Hour)},
		},
	}
	// SearchEmails is called per email to check Sent Items.
	// We make it return a match only for the first call (Budget Review) → email 1 is replied
	callCount := 0
	mail.searchEmailsResult = nil
	searchFn := func(ctx context.Context, params domain.SearchEmailsParams) ([]domain.Email, error) {
		callCount++
		if callCount == 1 {
			// First email has a reply
			return []domain.Email{{ID: "sent-1", Subject: "RE: Budget Review"}}, nil
		}
		return nil, nil
	}
	_ = searchFn

	// Since we can't override a method on a struct easily, use a custom store
	customMail := &customMockMailStore{
		listResult:  mail.listEmailsInRangeResult,
		searchCalls: make([]domain.SearchEmailsParams, 0),
		searchFn:    searchFn,
	}

	engine := newTestEngine(customMail, &mockCalendarStore{}, defaultTestConfig())
	rpt, err := engine.Run(context.Background())
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if len(rpt.Unanswered) != 1 {
		t.Fatalf("Unanswered count = %d, want 1", len(rpt.Unanswered))
	}
	if rpt.Unanswered[0].ID != "2" {
		t.Fatalf("Unanswered[0].ID = %q, want %q", rpt.Unanswered[0].ID, "2")
	}
}

func TestRunVIPFiltersCorrectly(t *testing.T) {
	mail := &mockMailStore{
		listEmailsInRangeResult: []domain.Email{
			{ID: "1", Subject: "VIP email", From: "boss@vip.com"},
			{ID: "2", Subject: "Normal email", From: "stranger@other.com"},
		},
	}
	cfg := config.ReportConfig{
		SinceHours:    24,
		MaxPerSection: 20,
		VIPSenders:    []string{"@vip.com"},
	}

	engine := newTestEngine(mail, &mockCalendarStore{}, cfg)
	rpt, err := engine.Run(context.Background())
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if len(rpt.VIPEmails) != 1 {
		t.Fatalf("VIPEmails count = %d, want 1", len(rpt.VIPEmails))
	}
	if rpt.VIPEmails[0].ID != "1" {
		t.Fatalf("VIPEmails[0].ID = %q, want %q", rpt.VIPEmails[0].ID, "1")
	}
}

func TestRunAttachmentsFiltersHasAttachments(t *testing.T) {
	mail := &mockMailStore{
		listEmailsInRangeResult: []domain.Email{
			{ID: "1", Subject: "With attachment", HasAttachments: true},
			{ID: "2", Subject: "Without attachment", HasAttachments: false},
		},
	}

	engine := newTestEngine(mail, &mockCalendarStore{}, defaultTestConfig())
	rpt, err := engine.Run(context.Background())
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if len(rpt.Attachments) != 1 {
		t.Fatalf("Attachments count = %d, want 1", len(rpt.Attachments))
	}
	if rpt.Attachments[0].ID != "1" {
		t.Fatalf("Attachments[0].ID = %q, want %q", rpt.Attachments[0].ID, "1")
	}
}

func TestRunThreadsGroupsAndFilters(t *testing.T) {
	// 5 emails with same normalized subject, 2 emails with another subject → 1 group
	t1 := fixedNow.Add(-1 * time.Hour)
	t2 := fixedNow.Add(-2 * time.Hour)
	mail := &mockMailStore{
		listEmailsInRangeResult: []domain.Email{
			{ID: "1", Subject: "RE: Budget", From: "a@b.com", Date: t1},
			{ID: "2", Subject: "RE: Budget", From: "b@c.com", Date: t2},
			{ID: "3", Subject: "Budget", From: "c@d.com", Date: t2},
			{ID: "4", Subject: "FWD: Budget", From: "d@e.com", Date: t2},
			{ID: "5", Subject: "RES: Budget", From: "e@f.com", Date: t2},
			{ID: "6", Subject: "Standalone", From: "f@g.com", Date: t2},
			{ID: "7", Subject: "Standalone", From: "g@h.com", Date: t2},
		},
	}

	engine := newTestEngine(mail, &mockCalendarStore{}, defaultTestConfig())
	rpt, err := engine.Run(context.Background())
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	// "Budget" has 5 messages (threshold ≥3), "Standalone" has 2 (below threshold)
	if len(rpt.Threads) != 1 {
		t.Fatalf("Threads count = %d, want 1", len(rpt.Threads))
	}
	if rpt.Threads[0].NormalizedSubject != "BUDGET" {
		t.Fatalf("Threads[0].NormalizedSubject = %q, want %q", rpt.Threads[0].NormalizedSubject, "BUDGET")
	}
	if rpt.Threads[0].Count != 5 {
		t.Fatalf("Threads[0].Count = %d, want 5", rpt.Threads[0].Count)
	}
}

func TestRunCalendarHintCounts(t *testing.T) {
	mail := &mockMailStore{
		listEmailsInRangeResult: []domain.Email{
			{ID: "1", Subject: "Budget review meeting"},
			{ID: "2", Subject: "Budget proposal discussion"},
			{ID: "3", Subject: "Other topic"},
		},
	}
	calendar := &mockCalendarStore{
		listEventsResult: []domain.CalendarEvent{
			{ID: "evt1", Title: "Budget Review", Start: fixedNow, End: fixedNow.Add(time.Hour)},
		},
	}

	engine := newTestEngine(mail, calendar, defaultTestConfig())
	rpt, err := engine.Run(context.Background())
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if len(rpt.Events) != 1 {
		t.Fatalf("Events count = %d, want 1", len(rpt.Events))
	}
	// "Budget" (>4 chars) appears in subjects of emails 1 and 2
	if rpt.Events[0].RelatedEmails != 2 {
		t.Fatalf("Events[0].RelatedEmails = %d, want 2", rpt.Events[0].RelatedEmails)
	}
}

// customMockMailStore allows injecting a custom SearchEmails function for testing.
type customMockMailStore struct {
	listResult  []domain.Email
	searchCalls []domain.SearchEmailsParams
	searchFn    func(context.Context, domain.SearchEmailsParams) ([]domain.Email, error)
}

func (m *customMockMailStore) SearchEmails(ctx context.Context, params domain.SearchEmailsParams) ([]domain.Email, error) {
	m.searchCalls = append(m.searchCalls, params)
	if m.searchFn != nil {
		return m.searchFn(ctx, params)
	}
	return nil, nil
}

func (m *customMockMailStore) GetEmail(_ context.Context, _ string) (*domain.Email, error) {
	return nil, nil
}

func (m *customMockMailStore) ListAttachments(_ context.Context, _ domain.ListAttachmentsParams) ([]domain.Attachment, error) {
	return nil, nil
}

func (m *customMockMailStore) CreateDraft(_ context.Context, _ domain.CreateDraftParams) (*domain.Email, error) {
	return nil, nil
}

func (m *customMockMailStore) ReplyDraft(_ context.Context, _ domain.ReplyDraftParams) (*domain.Email, error) {
	return nil, nil
}

func (m *customMockMailStore) ForwardDraft(_ context.Context, _ domain.ForwardDraftParams) (*domain.Email, error) {
	return nil, nil
}

func (m *customMockMailStore) MarkRead(_ context.Context, _ domain.MarkReadParams) error {
	return nil
}

func (m *customMockMailStore) FlagEmail(_ context.Context, _ domain.FlagEmailParams) error {
	return nil
}

func (m *customMockMailStore) MoveEmail(_ context.Context, _ domain.MoveEmailParams) error {
	return nil
}

func (m *customMockMailStore) ListFolders(_ context.Context) ([]domain.MailFolder, error) {
	return nil, nil
}

func (m *customMockMailStore) DownloadAttachment(_ context.Context, _ domain.DownloadAttachmentParams) (*domain.DownloadedAttachment, error) {
	return nil, nil
}

func (m *customMockMailStore) DeleteEmail(_ context.Context, _ string) error { return nil }

func (m *customMockMailStore) ListEmailsInRange(_ context.Context, _ domain.ListEmailsInRangeParams) ([]domain.Email, error) {
	return m.listResult, nil
}
