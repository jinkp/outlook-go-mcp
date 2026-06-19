package mcp

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/jinkp/outlook-go-mcp/internal/config"
	"github.com/jinkp/outlook-go-mcp/internal/domain"
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

// ── ReplyDraft ──────────────────────────────────────────────────────────────

func TestHandleReplyDraftReturnsPolicyDeniedBeforeStoreCall(t *testing.T) {
	mail := &mockMailStore{}
	handlers := testHandlersWithDeps(mail, &mockCalendarStore{}, &mockPolicyGate{err: errors.New("denied")})

	result, err := handlers.HandleReplyDraft(context.Background(), toolRequest("reply_draft", map[string]any{
		"email_id": "mail-1",
		"body":     "Reply body",
	}))
	if err != nil {
		t.Fatalf("HandleReplyDraft() unexpected error = %v", err)
	}

	assertToolError(t, result, libmcp.INTERNAL_ERROR, "action denied by policy")
	if mail.replyDraftCalled {
		t.Fatal("ReplyDraft store call must not happen when policy denies")
	}
}

func TestHandleReplyDraftCallsStoreAndReturnsDraftIDOnSuccess(t *testing.T) {
	mail := &mockMailStore{replyDraftResult: &domain.Email{ID: "reply-draft-1"}}
	handlers := testHandlersWithDeps(mail, &mockCalendarStore{}, &mockPolicyGate{})

	result, err := handlers.HandleReplyDraft(context.Background(), toolRequest("reply_draft", map[string]any{
		"email_id": "mail-1",
		"body":     "Reply body",
	}))
	if err != nil {
		t.Fatalf("HandleReplyDraft() unexpected error = %v", err)
	}
	if result.IsError {
		t.Fatalf("HandleReplyDraft() returned error result: %+v", result)
	}
	if !mail.replyDraftCalled {
		t.Fatal("ReplyDraft store call was not executed")
	}
	if mail.replyDraftParams.EmailID != "mail-1" || mail.replyDraftParams.Body != "Reply body" {
		t.Fatalf("ReplyDraft params = %+v, want email_id=mail-1 body=Reply body", mail.replyDraftParams)
	}
	payload := result.StructuredContent.(map[string]any)
	if got := payload["draft_id"]; got != "reply-draft-1" {
		t.Fatalf("draft_id = %#v, want %q", got, "reply-draft-1")
	}
}

func TestHandleReplyDraftReturnsInvalidParamsWhenEmailIDMissing(t *testing.T) {
	handlers := testHandlers()

	result, err := handlers.HandleReplyDraft(context.Background(), toolRequest("reply_draft", map[string]any{
		"body": "Reply body",
	}))
	if err != nil {
		t.Fatalf("HandleReplyDraft() unexpected error = %v", err)
	}

	assertToolError(t, result, libmcp.INVALID_PARAMS, "email_id")
}

// ── ForwardDraft ─────────────────────────────────────────────────────────────

func TestHandleForwardDraftReturnsPolicyDeniedBeforeStoreCall(t *testing.T) {
	mail := &mockMailStore{}
	handlers := testHandlersWithDeps(mail, &mockCalendarStore{}, &mockPolicyGate{err: errors.New("denied")})

	result, err := handlers.HandleForwardDraft(context.Background(), toolRequest("forward_draft", map[string]any{
		"email_id": "mail-1",
		"to":       []any{"fwd@example.com"},
	}))
	if err != nil {
		t.Fatalf("HandleForwardDraft() unexpected error = %v", err)
	}

	assertToolError(t, result, libmcp.INTERNAL_ERROR, "action denied by policy")
	if mail.forwardDraftCalled {
		t.Fatal("ForwardDraft store call must not happen when policy denies")
	}
}

func TestHandleForwardDraftCallsStoreAndReturnsDraftIDOnSuccess(t *testing.T) {
	mail := &mockMailStore{forwardDraftResult: &domain.Email{ID: "fwd-draft-1"}}
	handlers := testHandlersWithDeps(mail, &mockCalendarStore{}, &mockPolicyGate{})

	result, err := handlers.HandleForwardDraft(context.Background(), toolRequest("forward_draft", map[string]any{
		"email_id": "mail-1",
		"to":       []any{"a@example.com", "b@example.com"},
		"body":     "Fwd body",
	}))
	if err != nil {
		t.Fatalf("HandleForwardDraft() unexpected error = %v", err)
	}
	if result.IsError {
		t.Fatalf("HandleForwardDraft() returned error result: %+v", result)
	}
	if !mail.forwardDraftCalled {
		t.Fatal("ForwardDraft store call was not executed")
	}
	if mail.forwardDraftParams.EmailID != "mail-1" {
		t.Fatalf("ForwardDraft params = %+v, want email_id=mail-1", mail.forwardDraftParams)
	}
	if len(mail.forwardDraftParams.To) != 2 {
		t.Fatalf("ForwardDraft to = %v, want 2 recipients", mail.forwardDraftParams.To)
	}
}

func TestHandleForwardDraftReturnsInvalidParamsWhenToMissing(t *testing.T) {
	handlers := testHandlers()

	result, err := handlers.HandleForwardDraft(context.Background(), toolRequest("forward_draft", map[string]any{
		"email_id": "mail-1",
	}))
	if err != nil {
		t.Fatalf("HandleForwardDraft() unexpected error = %v", err)
	}

	assertToolError(t, result, libmcp.INVALID_PARAMS, "to")
}

// ── MarkRead ─────────────────────────────────────────────────────────────────

func TestHandleMarkReadReturnsPolicyDeniedBeforeStoreCall(t *testing.T) {
	mail := &mockMailStore{}
	handlers := testHandlersWithDeps(mail, &mockCalendarStore{}, &mockPolicyGate{err: errors.New("denied")})

	result, err := handlers.HandleMarkRead(context.Background(), toolRequest("mark_read", map[string]any{
		"email_id": "mail-1",
		"read":     true,
	}))
	if err != nil {
		t.Fatalf("HandleMarkRead() unexpected error = %v", err)
	}

	assertToolError(t, result, libmcp.INTERNAL_ERROR, "action denied by policy")
	if mail.markReadCalled {
		t.Fatal("MarkRead store call must not happen when policy denies")
	}
}

func TestHandleMarkReadCallsStoreOnSuccess(t *testing.T) {
	mail := &mockMailStore{}
	handlers := testHandlersWithDeps(mail, &mockCalendarStore{}, &mockPolicyGate{})

	result, err := handlers.HandleMarkRead(context.Background(), toolRequest("mark_read", map[string]any{
		"email_id": "mail-1",
		"read":     true,
	}))
	if err != nil {
		t.Fatalf("HandleMarkRead() unexpected error = %v", err)
	}
	if result.IsError {
		t.Fatalf("HandleMarkRead() returned error result: %+v", result)
	}
	if !mail.markReadCalled {
		t.Fatal("MarkRead store call was not executed")
	}
	if mail.markReadParams.EmailID != "mail-1" || !mail.markReadParams.Read {
		t.Fatalf("MarkRead params = %+v, want email_id=mail-1 read=true", mail.markReadParams)
	}
}

func TestHandleMarkReadReturnsInvalidParamsWhenEmailIDMissing(t *testing.T) {
	handlers := testHandlers()

	result, err := handlers.HandleMarkRead(context.Background(), toolRequest("mark_read", map[string]any{
		"read": true,
	}))
	if err != nil {
		t.Fatalf("HandleMarkRead() unexpected error = %v", err)
	}

	assertToolError(t, result, libmcp.INVALID_PARAMS, "email_id")
}

// ── FlagEmail ────────────────────────────────────────────────────────────────

func TestHandleFlagEmailReturnsPolicyDeniedBeforeStoreCall(t *testing.T) {
	mail := &mockMailStore{}
	handlers := testHandlersWithDeps(mail, &mockCalendarStore{}, &mockPolicyGate{err: errors.New("denied")})

	result, err := handlers.HandleFlagEmail(context.Background(), toolRequest("flag_email", map[string]any{
		"email_id": "mail-1",
		"flagged":  true,
	}))
	if err != nil {
		t.Fatalf("HandleFlagEmail() unexpected error = %v", err)
	}

	assertToolError(t, result, libmcp.INTERNAL_ERROR, "action denied by policy")
	if mail.flagEmailCalled {
		t.Fatal("FlagEmail store call must not happen when policy denies")
	}
}

func TestHandleFlagEmailCallsStoreOnSuccess(t *testing.T) {
	mail := &mockMailStore{}
	handlers := testHandlersWithDeps(mail, &mockCalendarStore{}, &mockPolicyGate{})

	result, err := handlers.HandleFlagEmail(context.Background(), toolRequest("flag_email", map[string]any{
		"email_id": "mail-1",
		"flagged":  true,
	}))
	if err != nil {
		t.Fatalf("HandleFlagEmail() unexpected error = %v", err)
	}
	if result.IsError {
		t.Fatalf("HandleFlagEmail() returned error result: %+v", result)
	}
	if !mail.flagEmailCalled {
		t.Fatal("FlagEmail store call was not executed")
	}
	if mail.flagEmailParams.EmailID != "mail-1" || !mail.flagEmailParams.Flagged {
		t.Fatalf("FlagEmail params = %+v, want email_id=mail-1 flagged=true", mail.flagEmailParams)
	}
}

func TestHandleFlagEmailReturnsInvalidParamsWhenEmailIDMissing(t *testing.T) {
	handlers := testHandlers()

	result, err := handlers.HandleFlagEmail(context.Background(), toolRequest("flag_email", map[string]any{
		"flagged": true,
	}))
	if err != nil {
		t.Fatalf("HandleFlagEmail() unexpected error = %v", err)
	}

	assertToolError(t, result, libmcp.INVALID_PARAMS, "email_id")
}

// ── MoveEmail ────────────────────────────────────────────────────────────────

func TestHandleMoveEmailReturnsPolicyDeniedBeforeStoreCall(t *testing.T) {
	mail := &mockMailStore{}
	handlers := testHandlersWithDeps(mail, &mockCalendarStore{}, &mockPolicyGate{err: errors.New("denied")})

	result, err := handlers.HandleMoveEmail(context.Background(), toolRequest("move_email", map[string]any{
		"email_id": "mail-1",
		"folder":   "Archive",
	}))
	if err != nil {
		t.Fatalf("HandleMoveEmail() unexpected error = %v", err)
	}

	assertToolError(t, result, libmcp.INTERNAL_ERROR, "action denied by policy")
	if mail.moveEmailCalled {
		t.Fatal("MoveEmail store call must not happen when policy denies")
	}
}

func TestHandleMoveEmailCallsStoreOnSuccess(t *testing.T) {
	mail := &mockMailStore{}
	handlers := testHandlersWithDeps(mail, &mockCalendarStore{}, &mockPolicyGate{})

	result, err := handlers.HandleMoveEmail(context.Background(), toolRequest("move_email", map[string]any{
		"email_id": "mail-1",
		"folder":   "Archive",
	}))
	if err != nil {
		t.Fatalf("HandleMoveEmail() unexpected error = %v", err)
	}
	if result.IsError {
		t.Fatalf("HandleMoveEmail() returned error result: %+v", result)
	}
	if !mail.moveEmailCalled {
		t.Fatal("MoveEmail store call was not executed")
	}
	if mail.moveEmailParams.EmailID != "mail-1" || mail.moveEmailParams.Folder != "Archive" {
		t.Fatalf("MoveEmail params = %+v, want email_id=mail-1 folder=Archive", mail.moveEmailParams)
	}
}

func TestHandleMoveEmailReturnsInvalidParamsWhenFolderMissing(t *testing.T) {
	handlers := testHandlers()

	result, err := handlers.HandleMoveEmail(context.Background(), toolRequest("move_email", map[string]any{
		"email_id": "mail-1",
	}))
	if err != nil {
		t.Fatalf("HandleMoveEmail() unexpected error = %v", err)
	}

	assertToolError(t, result, libmcp.INVALID_PARAMS, "folder")
}

// ── ListFolders ──────────────────────────────────────────────────────────────

func TestHandleListFoldersReturnsJSONOnSuccess(t *testing.T) {
	mail := &mockMailStore{
		listFoldersResult: []domain.MailFolder{
			{Name: "Inbox", EntryID: "eid-1", ParentEntryID: "", FolderType: 0},
			{Name: "Archive", EntryID: "eid-2", ParentEntryID: "eid-1", FolderType: 0},
		},
	}
	handlers := testHandlersWithDeps(mail, &mockCalendarStore{}, &mockPolicyGate{})

	result, err := handlers.HandleListFolders(context.Background(), toolRequest("list_folders", map[string]any{}))
	if err != nil {
		t.Fatalf("HandleListFolders() unexpected error = %v", err)
	}
	if result.IsError {
		t.Fatalf("HandleListFolders() returned error result: %+v", result)
	}
	if !mail.listFoldersCalled {
		t.Fatal("ListFolders store call was not executed")
	}
	payload := result.StructuredContent.(map[string]any)
	if got := payload["count"]; got != float64(2) && got != 2 {
		t.Fatalf("count = %#v, want 2", got)
	}
}

func TestHandleListFoldersReturnsErrNotConnected(t *testing.T) {
	mail := &mockMailStore{listFoldersErr: domain.ErrNotConnected}
	handlers := testHandlersWithDeps(mail, &mockCalendarStore{}, &mockPolicyGate{})

	result, err := handlers.HandleListFolders(context.Background(), toolRequest("list_folders", map[string]any{}))
	if err != nil {
		t.Fatalf("HandleListFolders() unexpected error = %v", err)
	}

	assertToolError(t, result, libmcp.INTERNAL_ERROR, "Outlook not connected")
}

// ── DownloadAttachment ───────────────────────────────────────────────────────

func TestHandleDownloadAttachmentReturnsPolicyDeniedBeforeStoreCall(t *testing.T) {
	mail := &mockMailStore{}
	handlers := testHandlersWithDeps(mail, &mockCalendarStore{}, &mockPolicyGate{err: errors.New("denied")})

	result, err := handlers.HandleDownloadAttachment(context.Background(), toolRequest("download_attachment", map[string]any{
		"email_id":      "mail-1",
		"attachment_id": "1",
		"dest_dir":      "C:\\tmp",
	}))
	if err != nil {
		t.Fatalf("HandleDownloadAttachment() unexpected error = %v", err)
	}

	assertToolError(t, result, libmcp.INTERNAL_ERROR, "action denied by policy")
	if mail.downloadAttachCalled {
		t.Fatal("DownloadAttachment store call must not happen when policy denies")
	}
}

func TestHandleDownloadAttachmentCallsStoreAndReturnsResultOnSuccess(t *testing.T) {
	mail := &mockMailStore{
		downloadAttachResult: &domain.DownloadedAttachment{Name: "report.pdf", Path: "C:\\tmp\\report.pdf", Size: 2048},
	}
	handlers := testHandlersWithDeps(mail, &mockCalendarStore{}, &mockPolicyGate{})

	result, err := handlers.HandleDownloadAttachment(context.Background(), toolRequest("download_attachment", map[string]any{
		"email_id":      "mail-1",
		"attachment_id": "1",
		"dest_dir":      "C:\\tmp",
	}))
	if err != nil {
		t.Fatalf("HandleDownloadAttachment() unexpected error = %v", err)
	}
	if result.IsError {
		t.Fatalf("HandleDownloadAttachment() returned error result: %+v", result)
	}
	if !mail.downloadAttachCalled {
		t.Fatal("DownloadAttachment store call was not executed")
	}
	payload := result.StructuredContent.(map[string]any)
	if got := payload["name"]; got != "report.pdf" {
		t.Fatalf("name = %#v, want %q", got, "report.pdf")
	}
}

func TestHandleDownloadAttachmentReturnsInvalidParamsWhenEmailIDMissing(t *testing.T) {
	handlers := testHandlers()

	result, err := handlers.HandleDownloadAttachment(context.Background(), toolRequest("download_attachment", map[string]any{
		"attachment_id": "1",
		"dest_dir":      "C:\\tmp",
	}))
	if err != nil {
		t.Fatalf("HandleDownloadAttachment() unexpected error = %v", err)
	}

	assertToolError(t, result, libmcp.INVALID_PARAMS, "email_id")
}

// ── DeleteEmail ──────────────────────────────────────────────────────────────

func TestHandleDeleteEmailReturnsPolicyDeniedBeforeStoreCall(t *testing.T) {
	mail := &mockMailStore{}
	handlers := testHandlersWithDeps(mail, &mockCalendarStore{}, &mockPolicyGate{err: errors.New("denied")})

	result, err := handlers.HandleDeleteEmail(context.Background(), toolRequest("delete_email", map[string]any{
		"email_id": "mail-1",
	}))
	if err != nil {
		t.Fatalf("HandleDeleteEmail() unexpected error = %v", err)
	}

	assertToolError(t, result, libmcp.INTERNAL_ERROR, "action denied by policy")
	if mail.deleteEmailCalled {
		t.Fatal("DeleteEmail store call must not happen when policy denies")
	}
}

func TestHandleDeleteEmailCallsStoreOnSuccess(t *testing.T) {
	mail := &mockMailStore{}
	handlers := testHandlersWithDeps(mail, &mockCalendarStore{}, &mockPolicyGate{})

	result, err := handlers.HandleDeleteEmail(context.Background(), toolRequest("delete_email", map[string]any{
		"email_id": "mail-1",
	}))
	if err != nil {
		t.Fatalf("HandleDeleteEmail() unexpected error = %v", err)
	}
	if result.IsError {
		t.Fatalf("HandleDeleteEmail() returned error result: %+v", result)
	}
	if !mail.deleteEmailCalled {
		t.Fatal("DeleteEmail store call was not executed")
	}
	if mail.deleteEmailID != "mail-1" {
		t.Fatalf("DeleteEmail id = %q, want %q", mail.deleteEmailID, "mail-1")
	}
}

func TestHandleDeleteEmailReturnsInvalidParamsWhenEmailIDMissing(t *testing.T) {
	handlers := testHandlers()

	result, err := handlers.HandleDeleteEmail(context.Background(), toolRequest("delete_email", map[string]any{}))
	if err != nil {
		t.Fatalf("HandleDeleteEmail() unexpected error = %v", err)
	}

	assertToolError(t, result, libmcp.INVALID_PARAMS, "email_id")
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

	replyDraftResult     *domain.Email
	replyDraftErr        error
	replyDraftCalled     bool
	replyDraftParams     domain.ReplyDraftParams
	forwardDraftResult   *domain.Email
	forwardDraftErr      error
	forwardDraftCalled   bool
	forwardDraftParams   domain.ForwardDraftParams
	markReadErr          error
	markReadCalled       bool
	markReadParams       domain.MarkReadParams
	flagEmailErr         error
	flagEmailCalled      bool
	flagEmailParams      domain.FlagEmailParams
	moveEmailErr         error
	moveEmailCalled      bool
	moveEmailParams      domain.MoveEmailParams
	listFoldersResult    []domain.MailFolder
	listFoldersErr       error
	listFoldersCalled    bool
	downloadAttachResult *domain.DownloadedAttachment
	downloadAttachErr    error
	downloadAttachCalled bool
	downloadAttachParams domain.DownloadAttachmentParams
	deleteEmailErr       error
	deleteEmailCalled    bool
	deleteEmailID        string
}

func (m *mockMailStore) Ping(context.Context) error { return nil }
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

func (m *mockMailStore) ReplyDraft(_ context.Context, params domain.ReplyDraftParams) (*domain.Email, error) {
	m.replyDraftCalled = true
	m.replyDraftParams = params
	return m.replyDraftResult, m.replyDraftErr
}

func (m *mockMailStore) ForwardDraft(_ context.Context, params domain.ForwardDraftParams) (*domain.Email, error) {
	m.forwardDraftCalled = true
	m.forwardDraftParams = params
	return m.forwardDraftResult, m.forwardDraftErr
}

func (m *mockMailStore) MarkRead(_ context.Context, params domain.MarkReadParams) error {
	m.markReadCalled = true
	m.markReadParams = params
	return m.markReadErr
}

func (m *mockMailStore) FlagEmail(_ context.Context, params domain.FlagEmailParams) error {
	m.flagEmailCalled = true
	m.flagEmailParams = params
	return m.flagEmailErr
}

func (m *mockMailStore) MoveEmail(_ context.Context, params domain.MoveEmailParams) error {
	m.moveEmailCalled = true
	m.moveEmailParams = params
	return m.moveEmailErr
}

func (m *mockMailStore) ListFolders(_ context.Context) ([]domain.MailFolder, error) {
	m.listFoldersCalled = true
	return m.listFoldersResult, m.listFoldersErr
}

func (m *mockMailStore) DownloadAttachment(_ context.Context, params domain.DownloadAttachmentParams) (*domain.DownloadedAttachment, error) {
	m.downloadAttachCalled = true
	m.downloadAttachParams = params
	return m.downloadAttachResult, m.downloadAttachErr
}

func (m *mockMailStore) DeleteEmail(_ context.Context, id string) error {
	m.deleteEmailCalled = true
	m.deleteEmailID = id
	return m.deleteEmailErr
}

func (m *mockMailStore) ListEmailsInRange(_ context.Context, _ domain.ListEmailsInRangeParams) ([]domain.Email, error) {
	return nil, nil
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
