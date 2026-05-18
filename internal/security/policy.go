package security

import (
	"fmt"

	"github.com/jinkp/outlook-go-mcp/internal/config"
)

type PolicyGate interface {
	Check(action string) error
}

type policyGate struct {
	allowCreateDraft    bool
	allowCreateEvent    bool
	allowReplyDraft     bool
	allowForwardDraft   bool
	allowMarkRead       bool
	allowFlagEmail      bool
	allowMoveEmail      bool
	allowDeleteEmail    bool
	allowSaveAttachment bool
}

func NewPolicyGate(cfg config.Config) PolicyGate {
	return policyGate{
		allowCreateDraft:    cfg.Security.AllowCreateDraft,
		allowCreateEvent:    cfg.Security.AllowCreateEvent,
		allowReplyDraft:     cfg.Security.AllowReplyDraft,
		allowForwardDraft:   cfg.Security.AllowForwardDraft,
		allowMarkRead:       cfg.Security.AllowMarkRead,
		allowFlagEmail:      cfg.Security.AllowFlagEmail,
		allowMoveEmail:      cfg.Security.AllowMoveEmail,
		allowDeleteEmail:    cfg.Security.AllowDeleteEmail,
		allowSaveAttachment: cfg.Security.AllowSaveAttachment,
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
	case "reply_draft":
		if !g.allowReplyDraft {
			return fmt.Errorf("action '%s' is disabled by security policy", action)
		}
	case "forward_draft":
		if !g.allowForwardDraft {
			return fmt.Errorf("action '%s' is disabled by security policy", action)
		}
	case "mark_read":
		if !g.allowMarkRead {
			return fmt.Errorf("action '%s' is disabled by security policy", action)
		}
	case "flag_email":
		if !g.allowFlagEmail {
			return fmt.Errorf("action '%s' is disabled by security policy", action)
		}
	case "move_email":
		if !g.allowMoveEmail {
			return fmt.Errorf("action '%s' is disabled by security policy", action)
		}
	case "delete_email":
		if !g.allowDeleteEmail {
			return fmt.Errorf("action '%s' is disabled by security policy", action)
		}
	case "download_attachment":
		if !g.allowSaveAttachment {
			return fmt.Errorf("action '%s' is disabled by security policy", action)
		}
	}

	return nil
}
