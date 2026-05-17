package mcp

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/isai/outlook-mcp/internal/config"
	"github.com/isai/outlook-mcp/internal/domain"
	libmcp "github.com/mark3labs/mcp-go/mcp"
)

func TestHandleSearchEmailsReturnsInvalidParamsWhenQueryMissing(t *testing.T) {
	handlers := testHandlers()

	result, err := handlers.HandleSearchEmails(context.Background(), toolRequest("search_emails", map[string]any{}))
	if err != nil {
		t.Fatalf("HandleSearchEmails() unexpected error = %v", err)
	}

	assertToolError(t, result, libmcp.INVALID_PARAMS, "query")
}

func TestHandleCreateDraftReturnsPolicyDeniedBeforeStoreCall(t *testing.T) {
	mail := &mockMailStore{}
	handlers := testHandlersWithDeps(mail, &mockCalendarStore{}, &mockPolicyGate{err: errors.New("denied")})

	result, err := handlers.HandleCreateDraft(context.Background(), toolRequest("create_draft", map[string]any{
		"to":      []any{"dev@example.com"},
		"subject": "Draft subject",
		"body":    "Draft body",
	}))
	if err != nil {
		t.Fatalf("HandleCreateDraft() unexpected error = %v", err)
	}

	assertToolError(t, result, libmcp.INTERNAL_ERROR, "action denied by policy")
	if mail.createDraftCalled {
		t.Fatal("CreateDraft store call must not happen when policy denies")
	}
}

func TestHandleCreateDraftCallsStoreAndReturnsJSONWhenAllowed(t *testing.T) {
	mail := &mockMailStore{createDraftResult: &domain.Email{ID: "draft-123"}}
	handlers := testHandlersWithDeps(mail, &mockCalendarStore{}, &mockPolicyGate{})

	result, err := handlers.HandleCreateDraft(context.Background(), toolRequest("create_draft", map[string]any{
		"to":      []any{"dev@example.com", "ops@example.com"},
		"subject": "Draft subject",
		"body":    "Draft body",
	}))
	if err != nil {
		t.Fatalf("HandleCreateDraft() unexpected error = %v", err)
	}
	if result.IsError {
		t.Fatalf("HandleCreateDraft() returned error result: %+v", result)
	}
	if !mail.createDraftCalled {
		t.Fatal("CreateDraft store call was not executed")
	}
	if len(mail.createDraftParams.To) != 2 || mail.createDraftParams.To[0] != "dev@example.com" {
		t.Fatalf("CreateDraft params = %+v, want recipients preserved", mail.createDraftParams)
	}

	payload := result.StructuredContent.(map[string]any)
	if got := payload["draft_id"]; got != "draft-123" {
		t.Fatalf("draft_id = %#v, want %q", got, "draft-123")
	}
	if got := payload["saved"]; got != true {
		t.Fatalf("saved = %#v, want true", got)
	}
}

func TestHandleCreateEventMapsInvalidParamsWhenEndBeforeStart(t *testing.T) {
	handlers := testHandlers()

	result, err := handlers.HandleCreateEvent(context.Background(), toolRequest("create_event", map[string]any{
		"title": "Focus time",
		"start": "2026-05-16T10:00:00Z",
		"end":   "2026-05-16T09:00:00Z",
	}))
	if err != nil {
		t.Fatalf("HandleCreateEvent() unexpected error = %v", err)
	}

	assertToolError(t, result, libmcp.INVALID_PARAMS, "end must be after start")
}

func TestHandleGetEmailMapsNotFound(t *testing.T) {
	handlers := testHandlersWithDeps(&mockMailStore{getEmailErr: domain.ErrNotFound}, &mockCalendarStore{}, &mockPolicyGate{})

	result, err := handlers.HandleGetEmail(context.Background(), toolRequest("get_email", map[string]any{"id": "missing"}))
	if err != nil {
		t.Fatalf("HandleGetEmail() unexpected error = %v", err)
	}

	assertToolError(t, result, libmcp.RESOURCE_NOT_FOUND, "not found")
}

func TestHandleListEventsReturnsJSONOnSuccess(t *testing.T) {
	start := time.Date(2026, time.May, 16, 9, 0, 0, 0, time.UTC)
	end := start.Add(time.Hour)
	handlers := testHandlersWithDeps(&mockMailStore{}, &mockCalendarStore{
		listEventsResult: []domain.CalendarEvent{{
			ID:       "event-123",
			Title:    "Architecture review",
			Start:    start,
			End:      end,
			Location: "Room 5",
			Body:     "Notes",
		}},
	}, &mockPolicyGate{})

	result, err := handlers.HandleListEvents(context.Background(), toolRequest("list_events", map[string]any{
		"start":       "2026-05-16T09:00:00Z",
		"end":         "2026-05-16T11:00:00Z",
		"max_results": 5,
	}))
	if err != nil {
		t.Fatalf("HandleListEvents() unexpected error = %v", err)
	}
	if result.IsError {
		t.Fatalf("HandleListEvents() returned error result: %+v", result)
	}

	payload := result.StructuredContent.(map[string]any)
	if got := payload["count"]; got != float64(1) && got != 1 {
		t.Fatalf("count = %#v, want 1", got)
	}
}

func TestHandleListAttachmentsReturnsInvalidParamsWhenEmailIDMissing(t *testing.T) {
	handlers := testHandlers()

	result, err := handlers.HandleListAttachments(context.Background(), toolRequest("list_attachments", map[string]any{}))
	if err != nil {
		t.Fatalf("HandleListAttachments() unexpected error = %v", err)
	}

	assertToolError(t, result, libmcp.INVALID_PARAMS, "email_id")
}

func TestHandleListAttachmentsReturnsJSONOnSuccess(t *testing.T) {
	mail := &mockMailStore{attachmentsResult: []domain.Attachment{{ID: "att-1", Name: "report.pdf", Size: 1024, ContentType: "application/pdf"}}}
	handlers := testHandlersWithDeps(mail, &mockCalendarStore{}, &mockPolicyGate{})

	result, err := handlers.HandleListAttachments(context.Background(), toolRequest("list_attachments", map[string]any{"email_id": "mail-1"}))
	if err != nil {
		t.Fatalf("HandleListAttachments() unexpected error = %v", err)
	}
	if result.IsError {
		t.Fatalf("HandleListAttachments() returned error result: %+v", result)
	}
	if !mail.listAttachmentsCalled {
		t.Fatal("ListAttachments store call was not executed")
	}
	if mail.listAttachmentsParams.EmailID != "mail-1" {
		t.Fatalf("ListAttachments params = %+v, want email_id preserved", mail.listAttachmentsParams)
	}
}

func TestHandleGetEventMapsNotFound(t *testing.T) {
	handlers := testHandlersWithDeps(&mockMailStore{}, &mockCalendarStore{getEventErr: domain.ErrNotFound}, &mockPolicyGate{})

	result, err := handlers.HandleGetEvent(context.Background(), toolRequest("get_event", map[string]any{"id": "missing"}))
	if err != nil {
		t.Fatalf("HandleGetEvent() unexpected error = %v", err)
	}

	assertToolError(t, result, libmcp.RESOURCE_NOT_FOUND, "not found")
}

func TestHandleCreateEventReturnsPolicyDeniedBeforeStoreCall(t *testing.T) {
	calendar := &mockCalendarStore{}
	handlers := testHandlersWithDeps(&mockMailStore{}, calendar, &mockPolicyGate{err: errors.New("denied")})

	result, err := handlers.HandleCreateEvent(context.Background(), toolRequest("create_event", map[string]any{
		"title": "Focus time",
		"start": "2026-05-16T10:00:00Z",
		"end":   "2026-05-16T11:00:00Z",
	}))
	if err != nil {
		t.Fatalf("HandleCreateEvent() unexpected error = %v", err)
	}

	assertToolError(t, result, libmcp.INTERNAL_ERROR, "action denied by policy")
	if calendar.createEventCalled {
		t.Fatal("CreateEvent store call must not happen when policy denies")
	}
}

func TestHandleGetEmailReturnsInvalidParamsWhenIDMissing(t *testing.T) {
	handlers := testHandlers()

	result, err := handlers.HandleGetEmail(context.Background(), toolRequest("get_email", map[string]any{}))
	if err != nil {
		t.Fatalf("HandleGetEmail() unexpected error = %v", err)
	}

	assertToolError(t, result, libmcp.INVALID_PARAMS, "id")
}

func TestHandleGetEmailMapsCOMFailure(t *testing.T) {
	handlers := testHandlersWithDeps(&mockMailStore{getEmailErr: domain.ErrCOMFailure}, &mockCalendarStore{}, &mockPolicyGate{})

	result, err := handlers.HandleGetEmail(context.Background(), toolRequest("get_email", map[string]any{"id": "mail-1"}))
	if err != nil {
		t.Fatalf("HandleGetEmail() unexpected error = %v", err)
	}

	assertToolError(t, result, libmcp.INTERNAL_ERROR, "COM automation failure")
	payload := result.StructuredContent.(map[string]any)
	data := payload["data"].(map[string]any)
	if got := data["category"]; got != "com_failure" {
		t.Fatalf("category = %#v, want %q", got, "com_failure")
	}
}

func TestHandleGetEmailMapsNotConnected(t *testing.T) {
	handlers := testHandlersWithDeps(&mockMailStore{getEmailErr: domain.ErrNotConnected}, &mockCalendarStore{}, &mockPolicyGate{})

	result, err := handlers.HandleGetEmail(context.Background(), toolRequest("get_email", map[string]any{"id": "some-id"}))
	if err != nil {
		t.Fatalf("HandleGetEmail() unexpected error = %v", err)
	}

	assertToolError(t, result, libmcp.INTERNAL_ERROR, "Outlook not connected")
	payload := result.StructuredContent.(map[string]any)
	data := payload["data"].(map[string]any)
	if got := data["category"]; got != "com_unavailable" {
		t.Fatalf("category = %#v, want %q", got, "com_unavailable")
	}
	if got := data["retryable"]; got != true {
		t.Fatalf("retryable = %#v, want true", got)
	}
}

func TestHandleListEventsMapsNotConnected(t *testing.T) {
	handlers := testHandlersWithDeps(&mockMailStore{}, &mockCalendarStore{listEventsErr: domain.ErrNotConnected}, &mockPolicyGate{})

	result, err := handlers.HandleListEvents(context.Background(), toolRequest("list_events", map[string]any{
		"start": "2026-01-01T00:00:00Z",
		"end":   "2026-01-31T00:00:00Z",
	}))
	if err != nil {
		t.Fatalf("HandleListEvents() unexpected error = %v", err)
	}

	assertToolError(t, result, libmcp.INTERNAL_ERROR, "Outlook not connected")
	payload := result.StructuredContent.(map[string]any)
	data := payload["data"].(map[string]any)
	if got := data["category"]; got != "com_unavailable" {
		t.Fatalf("category = %#v, want %q", got, "com_unavailable")
	}
	if got := data["retryable"]; got != true {
		t.Fatalf("retryable = %#v, want true", got)
	}
}

func assertToolError(t *testing.T, result *libmcp.CallToolResult, wantCode int, wantMessageSubstring string) {
	t.Helper()
	if result == nil {
		t.Fatal("result = nil")
	}
	if !result.IsError {
		t.Fatalf("result.IsError = false, want true; result = %+v", result)
	}
	payload, ok := result.StructuredContent.(map[string]any)
	if !ok {
		t.Fatalf("StructuredContent type = %T, want map[string]any", result.StructuredContent)
	}
	if got := payload["code"]; got != float64(wantCode) && got != wantCode {
		t.Fatalf("code = %#v, want %d", got, wantCode)
	}
	message, _ := payload["message"].(string)
	if message == "" || !strings.Contains(message, wantMessageSubstring) {
		t.Fatalf("message = %q, want substring %q", message, wantMessageSubstring)
	}
}

func toolRequest(name string, args map[string]any) libmcp.CallToolRequest {
	return libmcp.CallToolRequest{Params: libmcp.CallToolParams{Name: name, Arguments: args}}
}

func testHandlers() *Handlers {
	return testHandlersWithDeps(&mockMailStore{}, &mockCalendarStore{}, &mockPolicyGate{})
}

func testHandlersWithDeps(mail *mockMailStore, calendar *mockCalendarStore, policy *mockPolicyGate) *Handlers {
	return &Handlers{
		Mail:     mail,
		Calendar: calendar,
		Policy:   policy,
		Config:   &config.Config{Limits: config.LimitsConfig{MaxResults: 50}},
		Logger:   slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
}

type mockMailStore struct {
	searchEmailsResult    []domain.Email
	searchEmailsErr       error
	getEmailResult        *domain.Email
	getEmailErr           error
	attachmentsResult     []domain.Attachment
	attachmentsErr        error
	createDraftResult     *domain.Email
	createDraftErr        error
	createDraftCalled     bool
	createDraftParams     domain.CreateDraftParams
	listAttachmentsCalled bool
	listAttachmentsParams domain.ListAttachmentsParams
}

func (m *mockMailStore) SearchEmails(context.Context, domain.SearchEmailsParams) ([]domain.Email, error) {
	return m.searchEmailsResult, m.searchEmailsErr
}

func (m *mockMailStore) GetEmail(context.Context, string) (*domain.Email, error) {
	return m.getEmailResult, m.getEmailErr
}

func (m *mockMailStore) ListAttachments(_ context.Context, params domain.ListAttachmentsParams) ([]domain.Attachment, error) {
	m.listAttachmentsCalled = true
	m.listAttachmentsParams = params
	return m.attachmentsResult, m.attachmentsErr
}

func (m *mockMailStore) CreateDraft(_ context.Context, params domain.CreateDraftParams) (*domain.Email, error) {
	m.createDraftCalled = true
	m.createDraftParams = params
	return m.createDraftResult, m.createDraftErr
}

type mockCalendarStore struct {
	listEventsResult  []domain.CalendarEvent
	listEventsErr     error
	getEventResult    *domain.CalendarEvent
	getEventErr       error
	createEventResult *domain.CalendarEvent
	createEventErr    error
	createEventCalled bool
}

func (m *mockCalendarStore) ListEvents(context.Context, domain.ListEventsParams) ([]domain.CalendarEvent, error) {
	return m.listEventsResult, m.listEventsErr
}

func (m *mockCalendarStore) GetEvent(context.Context, string) (*domain.CalendarEvent, error) {
	return m.getEventResult, m.getEventErr
}

func (m *mockCalendarStore) CreateEvent(context.Context, domain.CreateEventParams) (*domain.CalendarEvent, error) {
	m.createEventCalled = true
	return m.createEventResult, m.createEventErr
}

type mockPolicyGate struct{ err error }

func (m *mockPolicyGate) Check(string) error { return m.err }
