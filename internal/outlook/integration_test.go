//go:build integration && windows

package outlook

import (
	"context"
	"testing"
	"time"
)

func TestIntegration_Connect(t *testing.T) {
	// requires: Outlook Desktop running, MAPI profile configured
	session, executor, cleanup := newIntegrationRuntime(t)
	defer cleanup()

	if session == nil {
		t.Fatal("session = nil")
	}
	if executor == nil {
		t.Fatal("executor = nil")
	}
	if !session.IsConnected() {
		t.Fatal("session.IsConnected() = false, want true")
	}
	if err := executor.Submit(context.Background(), func() error { return nil }); err != nil {
		t.Fatalf("executor.Submit() error = %v", err)
	}
}

func TestIntegration_SearchEmails(t *testing.T) {
	// requires: Outlook Desktop running, MAPI profile configured
	_, executor, cleanup := newIntegrationRuntime(t)
	defer cleanup()

	store := NewMailStore(executor)
	emails, err := store.SearchEmails(context.Background(), SearchEmailsParams{
		Query:      "a",
		Folder:     "Inbox",
		MaxResults: 5,
	})
	if err != nil {
		t.Fatalf("SearchEmails() error = %v", err)
	}
	if len(emails) == 0 {
		t.Fatal("SearchEmails() returned 0 results, want at least 1")
	}
	if emails[0].ID == "" {
		t.Fatal("SearchEmails()[0].ID = empty, want populated id")
	}
	if emails[0].Subject == "" && emails[0].From == "" {
		t.Fatal("SearchEmails()[0] missing both subject and sender")
	}
}

func TestIntegration_GetEmail(t *testing.T) {
	// requires: Outlook Desktop running, MAPI profile configured
	_, executor, cleanup := newIntegrationRuntime(t)
	defer cleanup()

	store := NewMailStore(executor)
	emails, err := store.SearchEmails(context.Background(), SearchEmailsParams{
		Query:      "a",
		Folder:     "Inbox",
		MaxResults: 1,
	})
	if err != nil {
		t.Fatalf("SearchEmails() error = %v", err)
	}
	if len(emails) == 0 {
		t.Fatal("SearchEmails() returned 0 results, cannot validate GetEmail")
	}

	email, err := store.GetEmail(context.Background(), emails[0].ID)
	if err != nil {
		t.Fatalf("GetEmail() error = %v", err)
	}
	if email == nil {
		t.Fatal("GetEmail() = nil, want email")
	}
	if email.ID == "" {
		t.Fatal("GetEmail().ID = empty, want populated id")
	}
	if email.Subject == "" {
		t.Fatal("GetEmail().Subject = empty, want populated subject")
	}
}

func TestIntegration_ListEvents(t *testing.T) {
	// requires: Outlook Desktop running, MAPI profile configured
	_, executor, cleanup := newIntegrationRuntime(t)
	defer cleanup()

	store := NewCalendarStore(executor)
	now := time.Now()
	events, err := store.ListEvents(context.Background(), ListEventsParams{
		Start:      now,
		End:        now.Add(7 * 24 * time.Hour),
		MaxResults: 10,
	})
	if err != nil {
		t.Fatalf("ListEvents() error = %v", err)
	}
	if len(events) == 0 {
		t.Fatal("ListEvents() returned 0 results, want at least 1 upcoming event")
	}
	if events[0].ID == "" {
		t.Fatal("ListEvents()[0].ID = empty, want populated id")
	}
	if !events[0].End.After(events[0].Start) {
		t.Fatal("ListEvents()[0] has non-positive duration")
	}
}

func TestIntegration_CreateAndDeleteDraft(t *testing.T) {
	// requires: Outlook Desktop running, MAPI profile configured
	_, executor, cleanup := newIntegrationRuntime(t)
	defer cleanup()

	store := NewMailStore(executor)
	draft, err := store.CreateDraft(context.Background(), CreateDraftParams{
		To:      []string{"integration@example.com"},
		Subject: "Outlook MCP integration draft",
		Body:    "This draft is created by the integration smoke test.",
	})
	if err != nil {
		t.Fatalf("CreateDraft() error = %v", err)
	}
	if draft == nil {
		t.Fatal("CreateDraft() = nil, want draft")
	}
	if draft.ID == "" {
		t.Fatal("CreateDraft().ID = empty, want populated id")
	}

	loaded, err := store.GetEmail(context.Background(), draft.ID)
	if err != nil {
		t.Fatalf("GetEmail(draft.ID) error = %v", err)
	}
	if loaded == nil {
		t.Fatal("GetEmail(draft.ID) = nil, want draft email")
	}
	if loaded.Subject != draft.Subject {
		t.Fatalf("loaded.Subject = %q, want %q", loaded.Subject, draft.Subject)
	}

	if err := deleteDraftByID(context.Background(), executor, draft.ID); err != nil {
		t.Fatalf("deleteDraftByID() error = %v", err)
	}
	if _, err := store.GetEmail(context.Background(), draft.ID); err == nil {
		t.Fatal("GetEmail(draft.ID) after delete error = nil, want not found")
	}
}
