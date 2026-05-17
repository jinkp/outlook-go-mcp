package security

import (
	"fmt"

	"github.com/isai/outlook-mcp/internal/config"
)

type PolicyGate interface {
	Check(action string) error
}

type policyGate struct {
	allowCreateDraft bool
	allowCreateEvent bool
}

func NewPolicyGate(cfg config.Config) PolicyGate {
	return policyGate{
		allowCreateDraft: cfg.Security.AllowCreateDraft,
		allowCreateEvent: cfg.Security.AllowCreateEvent,
	}
}

func (g policyGate) Check(action string) error {
	switch action {
	case "create_draft":
		if !g.allowCreateDraft {
			return fmt.Errorf("action '%s' is disabled by security policy", action)
		}
	case "create_event":
		if !g.allowCreateEvent {
			return fmt.Errorf("action '%s' is disabled by security policy", action)
		}
	}

	return nil
}
