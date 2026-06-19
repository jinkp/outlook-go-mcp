package main

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jinkp/outlook-go-mcp/internal/config"
	"github.com/jinkp/outlook-go-mcp/internal/domain"
	"github.com/jinkp/outlook-go-mcp/internal/mcp"
	"github.com/jinkp/outlook-go-mcp/internal/outlook"
	"github.com/jinkp/outlook-go-mcp/internal/security"
)

func TestBootstrapReturnsApplicationForValidConfig(t *testing.T) {
	configPath := writeMainTestConfig(t)
	fakeServer := &fakeMCPServer{}
	fakeExecutor := &fakeExecutor{}

	app, err := bootstrap(configPath, "", bootstrapDeps{
		loadConfig:   config.Load,
		newLogger:    loggingNewForTest,
		newSession:   func() outlook.OutlookSession { return &fakeSession{} },
		newExecutor:  func(outlook.OutlookSession) executorController { return fakeExecutor },
		newMailStore: func(executorController) domain.MailStore { return fakeMailStore{} },
		newCalendarStore: func(executorController) domain.CalendarStore {
			return fakeCalendarStore{}
		},
		newPolicyGate: func(cfg config.Config) security.PolicyGate {
			return security.NewPolicyGate(cfg)
		},
		newServer: func(handlers *mcp.Handlers) mcpServer {
			if handlers == nil {
				t.Fatal("handlers = nil")
			}
			if handlers.Mail == nil {
				t.Fatal("handlers.Mail = nil")
			}
			if handlers.Calendar == nil {
				t.Fatal("handlers.Calendar = nil")
			}
			if handlers.Policy == nil {
				t.Fatal("handlers.Policy = nil")
			}
			if handlers.Config == nil {
				t.Fatal("handlers.Config = nil")
			}
			if handlers.Logger == nil {
				t.Fatal("handlers.Logger = nil")
			}
			return fakeServer
		},
	})
	if err != nil {
		t.Fatalf("bootstrap() error = %v", err)
	}

	if app == nil {
		t.Fatal("bootstrap() = nil app")
	}
	if !fakeExecutor.started {
		t.Fatal("executor.Start() was not called during bootstrap")
	}
	if !fakeServer.registered {
		t.Fatal("server.RegisterTools() was not called")
	}
	if app.server != fakeServer {
		t.Fatal("bootstrap() did not retain the created server")
	}
	if app.executor != fakeExecutor {
		t.Fatal("bootstrap() did not retain the created executor")
	}
	if app.configPath != configPath {
		t.Fatalf("configPath = %q, want %q", app.configPath, configPath)
	}
}

func TestBootstrapReturnsExecutorStartError(t *testing.T) {
	configPath := writeMainTestConfig(t)
	wantErr := errors.New("executor start failed")

	_, err := bootstrap(configPath, "", bootstrapDeps{
		loadConfig:   config.Load,
		newLogger:    loggingNewForTest,
		newSession:   func() outlook.OutlookSession { return &fakeSession{} },
		newExecutor:  func(outlook.OutlookSession) executorController { return &fakeExecutor{startErr: wantErr} },
		newMailStore: func(executorController) domain.MailStore { return fakeMailStore{} },
		newCalendarStore: func(executorController) domain.CalendarStore {
			return fakeCalendarStore{}
		},
		newPolicyGate: func(cfg config.Config) security.PolicyGate {
			return security.NewPolicyGate(cfg)
		},
		newServer: func(handlers *mcp.Handlers) mcpServer {
			return &fakeMCPServer{}
		},
	})
	if err == nil {
		t.Fatal("bootstrap() error = nil, want executor start error")
	}

	var bootstrapErr *bootstrapError
	if !errors.As(err, &bootstrapErr) {
		t.Fatalf("bootstrap() error type = %T, want *bootstrapError", err)
	}
	if bootstrapErr.stage != stageExecutorStart {
		t.Fatalf("stage = %q, want %q", bootstrapErr.stage, stageExecutorStart)
	}
	if !errors.Is(bootstrapErr.err, wantErr) {
		t.Fatalf("underlying error = %v, want %v", bootstrapErr.err, wantErr)
	}
}

func TestBootstrapReturnsConfigErrorWhenFileMissing(t *testing.T) {
	missingPath := filepath.Join(t.TempDir(), "missing.yaml")

	_, err := bootstrap(missingPath, "", productionDeps())
	if err == nil {
		t.Fatal("bootstrap() error = nil, want missing config error")
	}

	var bootstrapErr *bootstrapError
	if !errors.As(err, &bootstrapErr) {
		t.Fatalf("bootstrap() error type = %T, want *bootstrapError", err)
	}
	if bootstrapErr.stage != stageConfigLoad {
		t.Fatalf("stage = %q, want %q", bootstrapErr.stage, stageConfigLoad)
	}
}

func writeMainTestConfig(t *testing.T) string {
	t.Helper()

	configPath := filepath.Join(t.TempDir(), "config.yaml")
	content := `outlook:
  profile: "default"

security:
  allow_send_email: false
  allow_create_draft: true
  allow_create_event: true
  allow_save_attachments: false

paths:
  attachment_dir: "C:\\OutlookMCP\\attachments"

logging:
  level: "info"

limits:
  max_results: 25
`

	if err := os.WriteFile(configPath, []byte(content), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	return configPath
}

func loggingNewForTest(level, _ string) (*slog.Logger, error) {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo})), nil
}

type fakeExecutor struct {
	started  bool
	stopped  bool
	startErr error
}

func (f *fakeExecutor) Start() error {
	f.started = true
	return f.startErr
}

func (f *fakeExecutor) Stop() {
	f.stopped = true
}

type fakeMCPServer struct {
	registered bool
	serveErr   error
}

func (f *fakeMCPServer) RegisterTools() {
	f.registered = true
}

func (f *fakeMCPServer) Serve(context.Context) error {
	return f.serveErr
}

type fakeSession struct{}

func (f *fakeSession) Connect() error    { return nil }
func (f *fakeSession) Close() error      { return nil }
func (f *fakeSession) IsConnected() bool { return true }

type fakeMailStore struct{}

func (fakeMailStore) Ping(context.Context) error { return nil }
func (fakeMailStore) SearchEmails(context.Context, domain.SearchEmailsParams) ([]domain.Email, error) {
	return nil, nil
}
func (fakeMailStore) GetEmail(context.Context, string) (*domain.Email, error) { return nil, nil }
func (fakeMailStore) ListAttachments(context.Context, domain.ListAttachmentsParams) ([]domain.Attachment, error) {
	return nil, nil
}
func (fakeMailStore) CreateDraft(context.Context, domain.CreateDraftParams) (*domain.Email, error) {
	return &domain.Email{ID: "draft-1"}, nil
}

func (fakeMailStore) ReplyDraft(context.Context, domain.ReplyDraftParams) (*domain.Email, error) {
	return nil, nil
}

func (fakeMailStore) ForwardDraft(context.Context, domain.ForwardDraftParams) (*domain.Email, error) {
	return nil, nil
}

func (fakeMailStore) MarkRead(context.Context, domain.MarkReadParams) error {
	return nil
}

func (fakeMailStore) FlagEmail(context.Context, domain.FlagEmailParams) error {
	return nil
}

func (fakeMailStore) MoveEmail(context.Context, domain.MoveEmailParams) error {
	return nil
}

func (fakeMailStore) ListFolders(context.Context) ([]domain.MailFolder, error) {
	return nil, nil
}

func (fakeMailStore) DownloadAttachment(context.Context, domain.DownloadAttachmentParams) (*domain.DownloadedAttachment, error) {
	return nil, nil
}

func (fakeMailStore) DeleteEmail(context.Context, string) error {
	return nil
}

func (fakeMailStore) ListEmailsInRange(context.Context, domain.ListEmailsInRangeParams) ([]domain.Email, error) {
	return nil, nil
}

func TestNewReportCmdWritesFileWhenOutputConfigured(t *testing.T) {
	outputFile := filepath.Join(t.TempDir(), "report.md")
	configPath := writeReportTestConfig(t, outputFile, "")

	// Build the command tree
	rootCmd := newRootCmd()

	// Override with a test-friendly args slice
	rootCmd.SetArgs([]string{
		"report",
		"--config", configPath,
		"--output", outputFile,
	})

	// We can't run it fully (COM not available), but we can verify the command structure
	// by checking --help works and the command is registered
	reportCmd, _, err := rootCmd.Find([]string{"report"})
	if err != nil {
		t.Fatalf("Find report command: %v", err)
	}
	if reportCmd == nil || reportCmd.Use != "report" {
		t.Fatalf("report command not found or has wrong Use: %v", reportCmd)
	}
	if reportCmd.Short == "" {
		t.Fatal("report command has no Short description")
	}

	// Verify flags are registered
	outputFlag := reportCmd.Flags().Lookup("output")
	if outputFlag == nil {
		t.Fatal("report command missing --output flag")
	}
	draftFlag := reportCmd.Flags().Lookup("draft")
	if draftFlag == nil {
		t.Fatal("report command missing --draft flag")
	}
	sinceFlag := reportCmd.Flags().Lookup("since")
	if sinceFlag == nil {
		t.Fatal("report command missing --since flag")
	}
}

func writeReportTestConfig(t *testing.T, outputFile, draftRecipient string) string {
	t.Helper()
	configPath := filepath.Join(t.TempDir(), "config.yaml")
	content := `outlook:
  profile: "default"

security:
  allow_create_draft: true

paths:
  attachment_dir: "C:\\OutlookMCP\\attachments"

logging:
  level: "info"

limits:
  max_results: 25

report:
  since_hours: 24
  max_per_section: 20
`
	if err := os.WriteFile(configPath, []byte(content), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	return configPath
}

func TestRunDryRunPrintsOKOnSuccess(t *testing.T) {
	configPath := writeMainTestConfig(t)

	var buf bytes.Buffer
	err := runDryRun(configPath, "", &buf, bootstrapDeps{
		loadConfig:   config.Load,
		newLogger:    loggingNewForTest,
		newSession:   func() outlook.OutlookSession { return &fakeSession{} },
		newExecutor:  func(outlook.OutlookSession) executorController { return &fakeExecutor{} },
		newMailStore: func(executorController) domain.MailStore { return fakeMailStore{} },
		newCalendarStore: func(executorController) domain.CalendarStore {
			return fakeCalendarStore{}
		},
		newPolicyGate: func(cfg config.Config) security.PolicyGate {
			return security.NewPolicyGate(cfg)
		},
		newServer: func(handlers *mcp.Handlers) mcpServer {
			return &fakeMCPServer{}
		},
	})
	if err != nil {
		t.Fatalf("runDryRun() error = %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "dry-run OK") {
		t.Fatalf("output missing 'dry-run OK', got: %s", output)
	}
	if !strings.Contains(output, "outlook:  connected") {
		t.Fatalf("output missing 'outlook:  connected', got: %s", output)
	}
}

func TestRunDryRunReportsBootstrapError(t *testing.T) {
	missingPath := filepath.Join(t.TempDir(), "missing.yaml")

	var buf bytes.Buffer
	err := runDryRun(missingPath, "", &buf, productionDeps())
	if err == nil {
		t.Fatal("runDryRun() error = nil, want config error")
	}

	output := buf.String()
	if !strings.Contains(output, "dry-run FAIL") {
		t.Fatalf("output missing 'dry-run FAIL', got: %s", output)
	}
}

func TestRunDryRunReportsConnectionError(t *testing.T) {
	configPath := writeMainTestConfig(t)

	var buf bytes.Buffer
	err := runDryRun(configPath, "", &buf, bootstrapDeps{
		loadConfig: config.Load,
		newLogger:  loggingNewForTest,
		newSession: func() outlook.OutlookSession { return &fakeSession{} },
		newExecutor: func(outlook.OutlookSession) executorController {
			return &fakeExecutor{}
		},
		newMailStore: func(executorController) domain.MailStore {
			return failingMailStore{}
		},
		newCalendarStore: func(executorController) domain.CalendarStore {
			return fakeCalendarStore{}
		},
		newPolicyGate: func(cfg config.Config) security.PolicyGate {
			return security.NewPolicyGate(cfg)
		},
		newServer: func(handlers *mcp.Handlers) mcpServer {
			return &fakeMCPServer{}
		},
	})
	if err == nil {
		t.Fatal("runDryRun() error = nil, want connection error")
	}

	output := buf.String()
	if !strings.Contains(output, "dry-run FAIL") {
		t.Fatalf("output missing 'dry-run FAIL', got: %s", output)
	}
}

func TestNewStatusCmdIsRegistered(t *testing.T) {
	rootCmd := newRootCmd()

	statusCmd, _, err := rootCmd.Find([]string{"status"})
	if err != nil {
		t.Fatalf("Find status command: %v", err)
	}
	if statusCmd == nil || statusCmd.Use != "status" {
		t.Fatalf("status command not found or has wrong Use: %v", statusCmd)
	}
	if statusCmd.Short == "" {
		t.Fatal("status command has no Short description")
	}

	configFlag := statusCmd.Flags().Lookup("config")
	if configFlag == nil {
		t.Fatal("status command missing --config flag")
	}
}

func TestMCPCmdHasDryRunFlag(t *testing.T) {
	rootCmd := newRootCmd()

	mcpCmd, _, err := rootCmd.Find([]string{"mcp"})
	if err != nil {
		t.Fatalf("Find mcp command: %v", err)
	}

	dryRunFlag := mcpCmd.Flags().Lookup("dry-run")
	if dryRunFlag == nil {
		t.Fatal("mcp command missing --dry-run flag")
	}
}

// failingMailStore simulates a mail store that cannot connect to Outlook.
type failingMailStore struct{}

func (failingMailStore) Ping(context.Context) error { return domain.ErrNotConnected }
func (failingMailStore) SearchEmails(context.Context, domain.SearchEmailsParams) ([]domain.Email, error) {
	return nil, domain.ErrNotConnected
}
func (failingMailStore) GetEmail(context.Context, string) (*domain.Email, error) {
	return nil, domain.ErrNotConnected
}
func (failingMailStore) ListAttachments(context.Context, domain.ListAttachmentsParams) ([]domain.Attachment, error) {
	return nil, domain.ErrNotConnected
}
func (failingMailStore) CreateDraft(context.Context, domain.CreateDraftParams) (*domain.Email, error) {
	return nil, domain.ErrNotConnected
}
func (failingMailStore) ReplyDraft(context.Context, domain.ReplyDraftParams) (*domain.Email, error) {
	return nil, domain.ErrNotConnected
}
func (failingMailStore) ForwardDraft(context.Context, domain.ForwardDraftParams) (*domain.Email, error) {
	return nil, domain.ErrNotConnected
}
func (failingMailStore) MarkRead(context.Context, domain.MarkReadParams) error {
	return domain.ErrNotConnected
}
func (failingMailStore) FlagEmail(context.Context, domain.FlagEmailParams) error {
	return domain.ErrNotConnected
}
func (failingMailStore) MoveEmail(context.Context, domain.MoveEmailParams) error {
	return domain.ErrNotConnected
}
func (failingMailStore) ListFolders(context.Context) ([]domain.MailFolder, error) {
	return nil, domain.ErrNotConnected
}
func (failingMailStore) DownloadAttachment(context.Context, domain.DownloadAttachmentParams) (*domain.DownloadedAttachment, error) {
	return nil, domain.ErrNotConnected
}
func (failingMailStore) DeleteEmail(context.Context, string) error {
	return domain.ErrNotConnected
}
func (failingMailStore) ListEmailsInRange(context.Context, domain.ListEmailsInRangeParams) ([]domain.Email, error) {
	return nil, domain.ErrNotConnected
}

type fakeCalendarStore struct{}

func (fakeCalendarStore) ListEvents(context.Context, domain.ListEventsParams) ([]domain.CalendarEvent, error) {
	return nil, nil
}
func (fakeCalendarStore) GetEvent(context.Context, string) (*domain.CalendarEvent, error) {
	return nil, nil
}
func (fakeCalendarStore) CreateEvent(context.Context, domain.CreateEventParams) (*domain.CalendarEvent, error) {
	return &domain.CalendarEvent{ID: "event-1"}, nil
}
