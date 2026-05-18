package config

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	defaultMaxResults = 50
	defaultLogLevel   = "info"
)

type Config struct {
	Outlook  OutlookConfig  `yaml:"outlook"`
	Security SecurityConfig `yaml:"security"`
	Paths    PathsConfig    `yaml:"paths"`
	Logging  LoggingConfig  `yaml:"logging"`
	Limits   LimitsConfig   `yaml:"limits"`
}

type OutlookConfig struct {
	Profile string `yaml:"profile"`
}

type SecurityConfig struct {
	AllowSendEmail      bool `yaml:"allow_send_email"`
	AllowCreateDraft    bool `yaml:"allow_create_draft"`
	AllowCreateEvent    bool `yaml:"allow_create_event"`
	AllowSaveAttachment bool `yaml:"allow_save_attachments"`
	AllowReplyDraft     bool `yaml:"allow_reply_draft"`
	AllowForwardDraft   bool `yaml:"allow_forward_draft"`
	AllowMarkRead       bool `yaml:"allow_mark_read"`
	AllowFlagEmail      bool `yaml:"allow_flag_email"`
	AllowMoveEmail      bool `yaml:"allow_move_email"`
	AllowDeleteEmail    bool `yaml:"allow_delete_email"`
}

type PathsConfig struct {
	AttachmentDir string `yaml:"attachment_dir"`
}

type LoggingConfig struct {
	Level string `yaml:"level"`
}

type LimitsConfig struct {
	MaxResults int `yaml:"max_results"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config %q: %w", path, err)
	}

	cfg := defaultConfig()
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	if err := dec.Decode(&cfg); err != nil {
		return nil, fmt.Errorf("parse config %q: %w", path, err)
	}

	cfg.Logging.Level = normalizeLevel(cfg.Logging.Level)

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func (c Config) Validate() error {
	if strings.TrimSpace(c.Paths.AttachmentDir) == "" {
		return fmt.Errorf("paths.attachment_dir is required")
	}
	if !filepath.IsAbs(c.Paths.AttachmentDir) {
		return fmt.Errorf("paths.attachment_dir must be an absolute path")
	}

	if c.Limits.MaxResults < 1 || c.Limits.MaxResults > 500 {
		return fmt.Errorf("limits.max_results must be between 1 and 500")
	}

	switch c.Logging.Level {
	case "debug", "info", "warn", "error":
		return nil
	default:
		return fmt.Errorf("logging.level must be one of: debug, info, warn, error")
	}
}

func defaultConfig() Config {
	return Config{
		Security: SecurityConfig{
			AllowSendEmail:   false,
			AllowCreateDraft: false,
			AllowCreateEvent: false,
		},
		Logging: LoggingConfig{Level: defaultLogLevel},
		Limits:  LimitsConfig{MaxResults: defaultMaxResults},
	}
}

func normalizeLevel(level string) string {
	trimmed := strings.TrimSpace(level)
	if trimmed == "" {
		return defaultLogLevel
	}

	return strings.ToLower(trimmed)
}
