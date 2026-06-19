package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"log/slog"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/jinkp/outlook-go-mcp/internal/config"
	"github.com/jinkp/outlook-go-mcp/internal/domain"
	"github.com/jinkp/outlook-go-mcp/internal/logging"
	"github.com/jinkp/outlook-go-mcp/internal/mcp"
	"github.com/jinkp/outlook-go-mcp/internal/outlook"
	"github.com/jinkp/outlook-go-mcp/internal/report"
	"github.com/jinkp/outlook-go-mcp/internal/security"
	"github.com/jinkp/outlook-go-mcp/internal/tui"
	"github.com/jinkp/outlook-go-mcp/internal/version"
	"github.com/spf13/cobra"
)

const (
	stageConfigLoad    = "config_load"
	stageLoggerInit    = "logger_init"
	stageExecutorStart = "executor_start"
)

type executorController interface {
	Start() error
	Stop()
}

type mcpServer interface {
	RegisterTools()
	Serve(context.Context) error
}

type bootstrapDeps struct {
	loadConfig       func(string) (*config.Config, error)
	newLogger        func(level string, logFile string) (*slog.Logger, error)
	newSession       func() outlook.OutlookSession
	newExecutor      func(outlook.OutlookSession) executorController
	newMailStore     func(executorController) domain.MailStore
	newCalendarStore func(executorController) domain.CalendarStore
	newPolicyGate    func(config.Config) security.PolicyGate
	newServer        func(*mcp.Handlers) mcpServer
}

type bootstrapError struct {
	stage  string
	err    error
	logger *slog.Logger
}

func (e *bootstrapError) Error() string {
	if e == nil || e.err == nil {
		return "bootstrap failed"
	}
	return e.err.Error()
}

func (e *bootstrapError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.err
}

type application struct {
	configPath string
	config     *config.Config
	logger     *slog.Logger
	executor   executorController
	server     mcpServer
	Mail       domain.MailStore
	Calendar   domain.CalendarStore
}

func main() {
	root := newRootCmd()
	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}

// newRootCmd builds the Cobra command tree:
//
//	outlook-mcp mcp            — start the MCP stdio server
//	outlook-mcp tui            — interactive TUI menu
//	outlook-mcp setup opencode — register in opencode.json via TUI wizard
//	outlook-mcp setup claude   — register in Claude Code config via TUI wizard
//	outlook-mcp setup config   — generate config.yaml via TUI wizard
func newRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:           "outlook-mcp",
		Short:         "Outlook MCP server — exposes Outlook mail and calendar to AI clients",
		Version:       version.Version,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	root.AddCommand(newMCPCmd())
	root.AddCommand(newTUICmd())
	root.AddCommand(newSetupCmd())
	root.AddCommand(newReportCmd())
	root.AddCommand(newStatusCmd())

	return root
}

// newMCPCmd returns the `outlook-mcp mcp` subcommand.
//
// CRITICAL: This command MUST NOT write ANYTHING to stdout before the server starts.
// The MCP stdio transport owns stdout entirely. All diagnostics go to stderr (and log file).
func newMCPCmd() *cobra.Command {
	var configPath string
	var logFile string
	var dryRun bool

	cmd := &cobra.Command{
		Use:          "mcp",
		Short:        "Start the Outlook MCP stdio server (for use with opencode, Claude, etc.)",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Redirect default logger to stderr — belt-and-suspenders guard.
			log.SetOutput(os.Stderr)

			// Pre-bootstrap logger: writes to stderr (and log file if set) BEFORE config loads.
			// Captures failures that happen before the real slog logger is ready.
			pre := logging.NewPreBootstrap(logFile)
			pre.Debug("outlook-mcp starting",
				slog.String("version", version.Version),
				slog.String("config_path", configPath),
				slog.String("log_file", logFile),
				slog.Int("pid", os.Getpid()),
				slog.String("exe", func() string { p, _ := os.Executable(); return p }()),
			)

			if dryRun {
				return runDryRun(configPath, logFile, cmd.OutOrStdout(), productionDeps())
			}

			code := run(configPath, logFile, os.Stderr, productionDeps())
			if code != 0 {
				os.Exit(code)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&configPath, "config", "config.yaml", "Path to YAML configuration file")
	cmd.Flags().StringVar(&logFile, "log-file", "", "Append structured JSON logs to this file (useful for debugging MCP startup)")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Validate bootstrap (config, logger, executor) and exit without starting the server")
	return cmd
}

// newTUICmd returns the `outlook-mcp tui` subcommand — interactive menu.
func newTUICmd() *cobra.Command {
	return &cobra.Command{
		Use:   "tui",
		Short: "Interactive TUI menu for outlook-mcp setup",
		RunE: func(cmd *cobra.Command, args []string) error {
			model := tui.NewMenuModel()
			p := tea.NewProgram(model, tea.WithAltScreen())

			finalModel, err := p.Run()
			if err != nil {
				return err
			}

			result := finalModel.(tui.MenuModel).Result()
			if result.Cancelled {
				return nil
			}

			return runSubcommand(result.Command)
		},
	}
}

// runSubcommand re-invokes the current binary with the given subcommand args.
// This mirrors bbkit's pattern: the TUI dispatches to CLI subcommands.
func runSubcommand(command string) error {
	parts := strings.Fields(command)
	if len(parts) == 0 {
		return nil
	}

	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("cannot find outlook-mcp executable: %w", err)
	}

	c := exec.Command(exe, parts...)
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr

	return c.Run()
}

// newSetupCmd returns the `outlook-mcp setup` command with opencode, claude and config subcommands.
func newSetupCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "setup",
		Short: "Register outlook-mcp as an MCP server in an AI client",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	cmd.AddCommand(newSetupOpenCodeCmd())
	cmd.AddCommand(newSetupClaudeCmd())
	cmd.AddCommand(newSetupConfigCmd())
	return cmd
}

func newSetupOpenCodeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "opencode",
		Short: "Wire outlook-mcp as an MCP server in opencode.json",
		Long:  "Launches a TUI wizard to write the outlook-mcp MCP server entry into the global or local opencode.json.",
		RunE: func(cmd *cobra.Command, args []string) error {
			wizard := tui.NewOpenCodeWizardModel()
			program := tea.NewProgram(wizard)

			finalModel, err := program.Run()
			if err != nil {
				return err
			}

			m, ok := finalModel.(tui.OpenCodeWizardModel)
			if !ok {
				return fmt.Errorf("setup opencode did not complete correctly")
			}

			if m.Cancelled() {
				fmt.Fprintln(cmd.OutOrStdout(), "Setup opencode cancelled.")
				return nil
			}

			if m.Error() {
				return fmt.Errorf("%s", m.ErrorMessage())
			}

			if m.Done() {
				fmt.Fprintln(cmd.OutOrStdout(), "opencode.json updated successfully.")
			}

			return nil
		},
	}
}

func newSetupClaudeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "claude",
		Short: "Wire outlook-mcp as an MCP server in Claude Code config",
		Long:  "Launches a TUI wizard to write the outlook-mcp MCP server entry into the global (~/.claude.json) or local (.claude/settings.json) Claude Code config.",
		RunE: func(cmd *cobra.Command, args []string) error {
			wizard := tui.NewClaudeWizardModel()
			program := tea.NewProgram(wizard)

			finalModel, err := program.Run()
			if err != nil {
				return err
			}

			m, ok := finalModel.(tui.ClaudeWizardModel)
			if !ok {
				return fmt.Errorf("setup claude did not complete correctly")
			}

			if m.Cancelled() {
				fmt.Fprintln(cmd.OutOrStdout(), "Setup claude cancelled.")
				return nil
			}

			if m.Error() {
				return fmt.Errorf("%s", m.ErrorMessage())
			}

			if m.Done() {
				fmt.Fprintln(cmd.OutOrStdout(), "Claude Code config updated successfully.")
			}

			return nil
		},
	}
}

func newSetupConfigCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "config",
		Short: "Generate config.yaml from the built-in template",
		Long:  "Launches a TUI wizard to create a config.yaml with your attachment path.",
		RunE: func(cmd *cobra.Command, args []string) error {
			wizard := tui.NewConfigWizardModel()
			program := tea.NewProgram(wizard)

			finalModel, err := program.Run()
			if err != nil {
				return err
			}

			m, ok := finalModel.(tui.ConfigWizardModel)
			if !ok {
				return fmt.Errorf("setup config did not complete correctly")
			}

			if m.Cancelled() {
				fmt.Fprintln(cmd.OutOrStdout(), "Setup config cancelled.")
				return nil
			}

			if m.Error() {
				return fmt.Errorf("%s", m.ErrorMessage())
			}

			if m.Done() {
				fmt.Fprintln(cmd.OutOrStdout(), "config.yaml written to: "+m.ConfigPath())
			}

			return nil
		},
	}
}

func run(configPath string, logFile string, stderr io.Writer, deps bootstrapDeps) int {
	app, err := bootstrap(configPath, logFile, deps)
	if err != nil {
		reportBootstrapError(stderr, err)
		return 1
	}
	defer app.executor.Stop()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	shutdownComplete := make(chan struct{})
	go func() {
		select {
		case <-ctx.Done():
			app.executor.Stop()
		case <-shutdownComplete:
		}
	}()
	defer close(shutdownComplete)

	app.logger.Info("Outlook MCP server ready",
		slog.String("version", version.Version),
		slog.String("config_path", app.configPath),
		slog.Int("tool_count", len(mcp.ToolDefinitions())),
		slog.String("outlook_profile", app.config.Outlook.Profile),
		slog.String("log_level", app.config.Logging.Level),
	)

	if err := app.server.Serve(ctx); err != nil && !errors.Is(err, context.Canceled) {
		app.logger.Error("Outlook MCP server stopped with error", slog.Any("error", err))
		return 1
	}

	app.logger.Info("Outlook MCP server stopped cleanly")
	return 0
}

func bootstrap(configPath string, logFile string, deps bootstrapDeps) (*application, error) {
	cfg, err := deps.loadConfig(configPath)
	if err != nil {
		return nil, &bootstrapError{stage: stageConfigLoad, err: err}
	}

	// --log-file flag takes priority over logging.log_file in config.
	resolvedLogFile := logFile
	if resolvedLogFile == "" {
		resolvedLogFile = cfg.Logging.LogFile
	}

	logger, err := deps.newLogger(cfg.Logging.Level, resolvedLogFile)
	if err != nil {
		return nil, &bootstrapError{stage: stageLoggerInit, err: err}
	}

	logger.Debug("config loaded",
		slog.String("outlook_profile", cfg.Outlook.Profile),
		slog.String("attachment_dir", cfg.Paths.AttachmentDir),
		slog.String("log_level", cfg.Logging.Level),
		slog.Int("max_results", cfg.Limits.MaxResults),
	)

	// Lazy connect: Start() launches the worker goroutine but does NOT connect
	// to Outlook. The COM connection is established on the first Submit() call.
	// This allows the MCP server to start and report "connected" to the AI client
	// even when Outlook is not yet open.
	session := deps.newSession()
	executor := deps.newExecutor(session)

	if err := executor.Start(); err != nil {
		return nil, &bootstrapError{stage: stageExecutorStart, err: err, logger: logger}
	}

	logger.Debug("executor started — Outlook connection deferred to first tool call")

	mailStore := deps.newMailStore(executor)
	calendarStore := deps.newCalendarStore(executor)

	handlers := mcp.Handlers{
		Mail:     mailStore,
		Calendar: calendarStore,
		Policy:   deps.newPolicyGate(*cfg),
		Config:   cfg,
		Logger:   logger,
	}

	server := deps.newServer(&handlers)
	server.RegisterTools()

	logger.Debug("MCP server ready", slog.Int("tool_count", len(mcp.ToolDefinitions())))

	return &application{
		configPath: configPath,
		config:     cfg,
		logger:     logger,
		executor:   executor,
		server:     server,
		Mail:       mailStore,
		Calendar:   calendarStore,
	}, nil
}

func reportBootstrapError(stderr io.Writer, err error) {
	var bootstrapErr *bootstrapError
	if !errors.As(err, &bootstrapErr) {
		fmt.Fprintf(stderr, "startup failed: %v\n", err)
		return
	}

	switch bootstrapErr.stage {
	case stageConfigLoad:
		fmt.Fprintf(stderr, "config load failed: %v\n", bootstrapErr.err)
	case stageLoggerInit:
		fmt.Fprintf(stderr, "logger init failed: %v\n", bootstrapErr.err)
	case stageExecutorStart:
		if bootstrapErr.logger != nil {
			bootstrapErr.logger.Error("Outlook is not running or not accessible", slog.Any("error", bootstrapErr.err))
			return
		}
		fmt.Fprintln(stderr, "Outlook is not running or not accessible")
	default:
		fmt.Fprintf(stderr, "startup failed: %v\n", bootstrapErr.err)
	}
}

// newReportCmd returns the `outlook-mcp report` subcommand.
func newReportCmd() *cobra.Command {
	var (
		configPath     string
		outputOverride string
		draftOverride  string
		sinceOverride  int
	)

	cmd := &cobra.Command{
		Use:          "report",
		Short:        "Generate daily email intelligence report (markdown file and/or Outlook draft)",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			log.SetOutput(os.Stderr)

			app, err := bootstrap(configPath, "", productionDeps())
			if err != nil {
				reportBootstrapError(os.Stderr, err)
				return err
			}
			defer app.executor.Stop()

			cfg := app.config.Report
			if outputOverride != "" {
				cfg.OutputFile = outputOverride
			}
			if draftOverride != "" {
				cfg.DraftRecipient = draftOverride
			}
			if sinceOverride > 0 {
				cfg.SinceHours = sinceOverride
			}

			// Apply defaults for zero values
			if cfg.SinceHours == 0 {
				cfg.SinceHours = 24
			}
			if cfg.MaxPerSection == 0 {
				cfg.MaxPerSection = 20
			}

			// Validate: at least one output must be configured
			if cfg.OutputFile == "" && cfg.DraftRecipient == "" {
				return fmt.Errorf("report: set output_file and/or draft_recipient in config (or use --output / --draft flags)")
			}

			ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
			defer stop()

			engine := report.NewEngine(app.Mail, app.Calendar, cfg, app.logger, nil)
			rpt, err := engine.Run(ctx)
			if err != nil {
				return fmt.Errorf("report failed: %w", err)
			}

			content := report.RenderMarkdown(rpt)

			if cfg.OutputFile != "" {
				if err := os.MkdirAll(filepath.Dir(cfg.OutputFile), 0o755); err != nil {
					return fmt.Errorf("create output directory: %w", err)
				}
				if err := os.WriteFile(cfg.OutputFile, []byte(content), 0o644); err != nil {
					return fmt.Errorf("write report file: %w", err)
				}
				app.logger.Info("Report written", slog.String("path", cfg.OutputFile))
			}

			if cfg.DraftRecipient != "" {
				if !app.config.Security.AllowCreateDraft {
					return fmt.Errorf("report: draft output requires security.allow_create_draft = true in config")
				}
				_, err := app.Mail.CreateDraft(ctx, domain.CreateDraftParams{
					To:      []string{cfg.DraftRecipient},
					Subject: fmt.Sprintf("Daily Report — %s", time.Now().Format("2006-01-02")),
					Body:    content,
				})
				if err != nil {
					return fmt.Errorf("create draft: %w", err)
				}
				app.logger.Info("Report draft created", slog.String("recipient", cfg.DraftRecipient))
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&configPath, "config", "config.yaml", "Path to config.yaml")
	cmd.Flags().StringVar(&outputOverride, "output", "", "Write report to this file path (overrides config)")
	cmd.Flags().StringVar(&draftOverride, "draft", "", "Create Outlook draft to this email (overrides config)")
	cmd.Flags().IntVar(&sinceOverride, "since", 0, "Lookback window in hours (overrides config)")
	return cmd
}

// runDryRun performs the full bootstrap (config, logger, executor) and validates
// connectivity by submitting a no-op job to the COM executor. It prints the
// result and exits without starting the MCP stdio server.
func runDryRun(configPath, logFile string, w io.Writer, deps bootstrapDeps) error {
	app, err := bootstrap(configPath, logFile, deps)
	if err != nil {
		fmt.Fprintf(w, "dry-run FAIL: bootstrap error: %v\n", err)
		return err
	}
	defer app.executor.Stop()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Ping forces the lazy COM connection and validates GetDefaultFolder(Inbox).
	if err := app.Mail.Ping(ctx); err != nil {
		fmt.Fprintf(w, "dry-run FAIL: Outlook connection error: %v\n", err)
		return err
	}

	fmt.Fprintf(w, "dry-run OK\n")
	fmt.Fprintf(w, "  config:   %s\n", app.configPath)
	fmt.Fprintf(w, "  version:  %s\n", version.Version)
	fmt.Fprintf(w, "  profile:  %s\n", app.config.Outlook.Profile)
	fmt.Fprintf(w, "  tools:    %d\n", len(mcp.ToolDefinitions()))
	fmt.Fprintf(w, "  outlook:  connected\n")
	return nil
}

// newStatusCmd returns the `outlook-mcp status` subcommand — a quick health
// check that validates config, starts the COM executor, connects to Outlook,
// and prints a diagnostic summary.
func newStatusCmd() *cobra.Command {
	var configPath string

	cmd := &cobra.Command{
		Use:          "status",
		Short:        "Check Outlook connectivity and print diagnostic info",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			w := cmd.OutOrStdout()

			fmt.Fprintln(w, "outlook-mcp status")
			fmt.Fprintln(w, strings.Repeat("-", 40))

			// Step 1: config
			fmt.Fprintf(w, "  config:     ")
			cfg, err := config.Load(configPath)
			if err != nil {
				fmt.Fprintf(w, "FAIL (%v)\n", err)
				return err
			}
			fmt.Fprintf(w, "OK (%s)\n", configPath)

			// Step 2: version
			fmt.Fprintf(w, "  version:    %s\n", version.Version)
			fmt.Fprintf(w, "  profile:    %s\n", cfg.Outlook.Profile)
			fmt.Fprintf(w, "  tools:      %d\n", len(mcp.ToolDefinitions()))

			// Step 3: COM executor + Outlook connection
			fmt.Fprintf(w, "  outlook:    ")

			session := outlook.NewOutlookSession()
			executor := outlook.NewCOMExecutor(session)

			if err := executor.Start(); err != nil {
				fmt.Fprintf(w, "FAIL (executor: %v)\n", err)
				return err
			}
			defer executor.Stop()

			mailStore := outlook.NewMailStore(executor)

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			// Ping forces lazy COM connect + validates GetDefaultFolder(Inbox).Name.
			// With retry, this will attempt up to 3 reconnections if Exchange is slow.
			if err := mailStore.Ping(ctx); err != nil {
				fmt.Fprintf(w, "FAIL (%v)\n", err)
				return err
			}
			fmt.Fprintln(w, "connected")

			// Step 4: security summary
			fmt.Fprintln(w)
			fmt.Fprintln(w, "  security policy:")
			fmt.Fprintf(w, "    send_email:      %v\n", cfg.Security.AllowSendEmail)
			fmt.Fprintf(w, "    create_draft:    %v\n", cfg.Security.AllowCreateDraft)
			fmt.Fprintf(w, "    create_event:    %v\n", cfg.Security.AllowCreateEvent)
			fmt.Fprintf(w, "    save_attachment: %v\n", cfg.Security.AllowSaveAttachment)
			fmt.Fprintf(w, "    reply_draft:     %v\n", cfg.Security.AllowReplyDraft)
			fmt.Fprintf(w, "    forward_draft:   %v\n", cfg.Security.AllowForwardDraft)
			fmt.Fprintf(w, "    mark_read:       %v\n", cfg.Security.AllowMarkRead)
			fmt.Fprintf(w, "    flag_email:      %v\n", cfg.Security.AllowFlagEmail)
			fmt.Fprintf(w, "    move_email:      %v\n", cfg.Security.AllowMoveEmail)
			fmt.Fprintf(w, "    delete_email:    %v\n", cfg.Security.AllowDeleteEmail)

			fmt.Fprintln(w)
			fmt.Fprintln(w, "All checks passed.")
			return nil
		},
	}

	cmd.Flags().StringVar(&configPath, "config", "config.yaml", "Path to config.yaml")
	return cmd
}

func productionDeps() bootstrapDeps {
	return bootstrapDeps{
		loadConfig: config.Load,
		newLogger:  func(level, logFile string) (*slog.Logger, error) { return logging.New(level, logFile) },
		newSession: outlook.NewOutlookSession,
		newExecutor: func(session outlook.OutlookSession) executorController {
			return outlook.NewCOMExecutor(session)
		},
		newMailStore: func(executor executorController) domain.MailStore {
			return outlook.NewMailStore(executor.(*outlook.COMExecutor))
		},
		newCalendarStore: func(executor executorController) domain.CalendarStore {
			return outlook.NewCalendarStore(executor.(*outlook.COMExecutor))
		},
		newPolicyGate: func(cfg config.Config) security.PolicyGate {
			return security.NewPolicyGate(cfg)
		},
		newServer: func(handlers *mcp.Handlers) mcpServer {
			return mcp.NewServer(handlers)
		},
	}
}
