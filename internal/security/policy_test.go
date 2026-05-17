package security

import (
	"testing"

	"github.com/isai/outlook-mcp/internal/config"
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
