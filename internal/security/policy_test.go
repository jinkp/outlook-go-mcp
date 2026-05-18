package security

import (
	"testing"

	"github.com/jinkp/outlook-go-mcp/internal/config"
)

func TestPolicyGateCheckCreateDraftDeniedByDefault(t *testing.T) {
	gate := NewPolicyGate(config.Config{})

	err := gate.Check("create_draft")
	if err == nil {
		t.Fatal("Check(create_draft) error = nil, want denied")
	}

	if got, want := err.Error(), "action 'create_draft' is disabled by security policy"; got != want {
		t.Fatalf("error = %q, want %q", got, want)
	}
}

func TestPolicyGateCheckCreateDraftAllowedWhenConfigured(t *testing.T) {
	gate := NewPolicyGate(config.Config{
		Security: config.SecurityConfig{AllowCreateDraft: true},
	})

	if err := gate.Check("create_draft"); err != nil {
		t.Fatalf("Check(create_draft) error = %v, want nil", err)
	}
}

func TestPolicyGateCheckCreateEventDeniedByDefault(t *testing.T) {
	gate := NewPolicyGate(config.Config{})

	err := gate.Check("create_event")
	if err == nil {
		t.Fatal("Check(create_event) error = nil, want denied")
	}

	if got, want := err.Error(), "action 'create_event' is disabled by security policy"; got != want {
		t.Fatalf("error = %q, want %q", got, want)
	}
}

func TestPolicyGateCheckCreateEventAllowedWhenConfigured(t *testing.T) {
	gate := NewPolicyGate(config.Config{
		Security: config.SecurityConfig{AllowCreateEvent: true},
	})

	if err := gate.Check("create_event"); err != nil {
		t.Fatalf("Check(create_event) error = %v, want nil", err)
	}
}

func TestPolicyGateCheckUnknownActionAllowed(t *testing.T) {
	gate := NewPolicyGate(config.Config{})

	if err := gate.Check("search_emails"); err != nil {
		t.Fatalf("Check(search_emails) error = %v, want nil", err)
	}
}

func TestPolicyGateCheckReplyDraftDeniedByDefault(t *testing.T) {
	gate := NewPolicyGate(config.Config{})

	err := gate.Check("reply_draft")
	if err == nil {
		t.Fatal("Check(reply_draft) error = nil, want denied")
	}
	if got, want := err.Error(), "action 'reply_draft' is disabled by security policy"; got != want {
		t.Fatalf("error = %q, want %q", got, want)
	}
}

func TestPolicyGateCheckReplyDraftAllowedWhenConfigured(t *testing.T) {
	gate := NewPolicyGate(config.Config{
		Security: config.SecurityConfig{AllowReplyDraft: true},
	})

	if err := gate.Check("reply_draft"); err != nil {
		t.Fatalf("Check(reply_draft) error = %v, want nil", err)
	}
}

func TestPolicyGateCheckForwardDraftDeniedByDefault(t *testing.T) {
	gate := NewPolicyGate(config.Config{})

	err := gate.Check("forward_draft")
	if err == nil {
		t.Fatal("Check(forward_draft) error = nil, want denied")
	}
	if got, want := err.Error(), "action 'forward_draft' is disabled by security policy"; got != want {
		t.Fatalf("error = %q, want %q", got, want)
	}
}

func TestPolicyGateCheckForwardDraftAllowedWhenConfigured(t *testing.T) {
	gate := NewPolicyGate(config.Config{
		Security: config.SecurityConfig{AllowForwardDraft: true},
	})

	if err := gate.Check("forward_draft"); err != nil {
		t.Fatalf("Check(forward_draft) error = %v, want nil", err)
	}
}

func TestPolicyGateCheckMarkReadDeniedByDefault(t *testing.T) {
	gate := NewPolicyGate(config.Config{})

	err := gate.Check("mark_read")
	if err == nil {
		t.Fatal("Check(mark_read) error = nil, want denied")
	}
	if got, want := err.Error(), "action 'mark_read' is disabled by security policy"; got != want {
		t.Fatalf("error = %q, want %q", got, want)
	}
}

func TestPolicyGateCheckMarkReadAllowedWhenConfigured(t *testing.T) {
	gate := NewPolicyGate(config.Config{
		Security: config.SecurityConfig{AllowMarkRead: true},
	})

	if err := gate.Check("mark_read"); err != nil {
		t.Fatalf("Check(mark_read) error = %v, want nil", err)
	}
}

func TestPolicyGateCheckFlagEmailDeniedByDefault(t *testing.T) {
	gate := NewPolicyGate(config.Config{})

	err := gate.Check("flag_email")
	if err == nil {
		t.Fatal("Check(flag_email) error = nil, want denied")
	}
	if got, want := err.Error(), "action 'flag_email' is disabled by security policy"; got != want {
		t.Fatalf("error = %q, want %q", got, want)
	}
}

func TestPolicyGateCheckFlagEmailAllowedWhenConfigured(t *testing.T) {
	gate := NewPolicyGate(config.Config{
		Security: config.SecurityConfig{AllowFlagEmail: true},
	})

	if err := gate.Check("flag_email"); err != nil {
		t.Fatalf("Check(flag_email) error = %v, want nil", err)
	}
}

func TestPolicyGateCheckMoveEmailDeniedByDefault(t *testing.T) {
	gate := NewPolicyGate(config.Config{})

	err := gate.Check("move_email")
	if err == nil {
		t.Fatal("Check(move_email) error = nil, want denied")
	}
	if got, want := err.Error(), "action 'move_email' is disabled by security policy"; got != want {
		t.Fatalf("error = %q, want %q", got, want)
	}
}

func TestPolicyGateCheckMoveEmailAllowedWhenConfigured(t *testing.T) {
	gate := NewPolicyGate(config.Config{
		Security: config.SecurityConfig{AllowMoveEmail: true},
	})

	if err := gate.Check("move_email"); err != nil {
		t.Fatalf("Check(move_email) error = %v, want nil", err)
	}
}

func TestPolicyGateCheckDeleteEmailDeniedByDefault(t *testing.T) {
	gate := NewPolicyGate(config.Config{})

	err := gate.Check("delete_email")
	if err == nil {
		t.Fatal("Check(delete_email) error = nil, want denied")
	}
	if got, want := err.Error(), "action 'delete_email' is disabled by security policy"; got != want {
		t.Fatalf("error = %q, want %q", got, want)
	}
}

func TestPolicyGateCheckDeleteEmailAllowedWhenConfigured(t *testing.T) {
	gate := NewPolicyGate(config.Config{
		Security: config.SecurityConfig{AllowDeleteEmail: true},
	})

	if err := gate.Check("delete_email"); err != nil {
		t.Fatalf("Check(delete_email) error = %v, want nil", err)
	}
}

func TestPolicyGateCheckDownloadAttachmentDeniedByDefault(t *testing.T) {
	gate := NewPolicyGate(config.Config{})

	err := gate.Check("download_attachment")
	if err == nil {
		t.Fatal("Check(download_attachment) error = nil, want denied")
	}
	if got, want := err.Error(), "action 'download_attachment' is disabled by security policy"; got != want {
		t.Fatalf("error = %q, want %q", got, want)
	}
}

func TestPolicyGateCheckDownloadAttachmentAllowedWhenConfigured(t *testing.T) {
	gate := NewPolicyGate(config.Config{
		Security: config.SecurityConfig{AllowSaveAttachment: true},
	})

	if err := gate.Check("download_attachment"); err != nil {
		t.Fatalf("Check(download_attachment) error = %v, want nil", err)
	}
}
