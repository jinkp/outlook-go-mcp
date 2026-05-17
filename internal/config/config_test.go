package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadValidConfig(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")

	configYAML := `outlook:
  profile: "default"

security:
  allow_send_email: false
  allow_create_draft: true
  allow_create_event: true
  allow_save_attachments: false

paths:
  attachment_dir: "C:\\OutlookMCP\\attachments"

logging:
  level: "debug"

limits:
  max_results: 42
`

	if err := os.WriteFile(configPath, []byte(configYAML), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Outlook.Profile != "default" {
		t.Fatalf("Profile = %q, want %q", cfg.Outlook.Profile, "default")
	}
	if !cfg.Security.AllowCreateDraft {
		t.Fatal("AllowCreateDraft = false, want true")
	}
	if !cfg.Security.AllowCreateEvent {
		t.Fatal("AllowCreateEvent = false, want true")
	}
	if cfg.Logging.Level != "debug" {
		t.Fatalf("Level = %q, want %q", cfg.Logging.Level, "debug")
	}
	if cfg.Limits.MaxResults != 42 {
		t.Fatalf("MaxResults = %d, want %d", cfg.Limits.MaxResults, 42)
	}
	if cfg.Paths.AttachmentDir != "C:\\OutlookMCP\\attachments" {
		t.Fatalf("AttachmentDir = %q, want %q", cfg.Paths.AttachmentDir, "C:\\OutlookMCP\\attachments")
	}
}

func TestLoadAppliesDefaultsForMissingOptionalFields(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")

	configYAML := `paths:
  attachment_dir: "C:\\OutlookMCP\\attachments"
`

	if err := os.WriteFile(configPath, []byte(configYAML), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Logging.Level != "info" {
		t.Fatalf("Level = %q, want %q", cfg.Logging.Level, "info")
	}
	if cfg.Limits.MaxResults != 50 {
		t.Fatalf("MaxResults = %d, want %d", cfg.Limits.MaxResults, 50)
	}
	if cfg.Security.AllowSendEmail {
		t.Fatal("AllowSendEmail = true, want false")
	}
	if cfg.Security.AllowCreateDraft {
		t.Fatal("AllowCreateDraft = true, want false")
	}
	if cfg.Security.AllowCreateEvent {
		t.Fatal("AllowCreateEvent = true, want false")
	}
}

func TestLoadRejectsInvalidMaxResults(t *testing.T) {
	tests := []struct {
		name       string
		maxResults int
	}{
		{name: "zero", maxResults: 0},
		{name: "too large", maxResults: 501},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()
			configPath := filepath.Join(tempDir, "config.yaml")
			configYAML := fmt.Sprintf("paths:\n  attachment_dir: \"C:\\\\OutlookMCP\\\\attachments\"\nlimits:\n  max_results: %d\n", tt.maxResults)

			if err := os.WriteFile(configPath, []byte(configYAML), 0o600); err != nil {
				t.Fatalf("write config: %v", err)
			}

			_, err := Load(configPath)
			if err == nil {
				t.Fatal("Load() error = nil, want max_results validation error")
			}
		})
	}
}

func TestLoadRejectsInvalidLogLevel(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")
	configYAML := `paths:
  attachment_dir: "C:\\OutlookMCP\\attachments"

logging:
  level: "trace"
`

	if err := os.WriteFile(configPath, []byte(configYAML), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	_, err := Load(configPath)
	if err == nil {
		t.Fatal("Load() error = nil, want invalid log level error")
	}
}

func TestLoadReturnsErrorWhenFileDoesNotExist(t *testing.T) {
	missingPath := filepath.Join(t.TempDir(), "missing.yaml")

	_, err := Load(missingPath)
	if err == nil {
		t.Fatal("Load() error = nil, want file not found error")
	}
}

func TestLoadRejectsMissingAttachmentDir(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")

	configYAML := `paths:
  attachment_dir: "   "
`

	if err := os.WriteFile(configPath, []byte(configYAML), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	_, err := Load(configPath)
	if err == nil {
		t.Fatal("Load() error = nil, want missing attachment_dir error")
	}
	if got := err.Error(); got != "paths.attachment_dir is required" {
		t.Fatalf("Load() error = %q, want %q", got, "paths.attachment_dir is required")
	}
}

func TestLoadRejectsRelativeAttachmentDir(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")

	configYAML := `paths:
  attachment_dir: "relative/path"
`

	if err := os.WriteFile(configPath, []byte(configYAML), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	_, err := Load(configPath)
	if err == nil {
		t.Fatal("Load() error = nil, want error for relative attachment_dir")
	}
	if !strings.Contains(err.Error(), "absolute") {
		t.Fatalf("Load() error = %q, want it to contain %q", err.Error(), "absolute")
	}
}

func TestLoadRejectsUnknownField(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")

	configYAML := `paths:
  attachment_dir: "C:\\OutlookMCP\\attachments"
unknown_top_level_key: "this should be rejected"
`

	if err := os.WriteFile(configPath, []byte(configYAML), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	_, err := Load(configPath)
	if err == nil {
		t.Fatal("Load() error = nil, want error for unknown field")
	}
	if !strings.Contains(err.Error(), "unknown_top_level_key") {
		t.Fatalf("Load() error = %q, want it to contain field name %q", err.Error(), "unknown_top_level_key")
	}
}

func TestLoadAcceptsMaxResultsAtUpperBound(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")

	configYAML := `paths:
  attachment_dir: "C:\\OutlookMCP\\attachments"

limits:
  max_results: 500
`

	if err := os.WriteFile(configPath, []byte(configYAML), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Limits.MaxResults != 500 {
		t.Fatalf("MaxResults = %d, want 500", cfg.Limits.MaxResults)
	}
}
