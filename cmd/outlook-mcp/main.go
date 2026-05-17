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
	"strings"
	"syscall"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/jinkp/outlook-go-mcp/internal/config"
	"github.com/jinkp/outlook-go-mcp/internal/domain"
	"github.com/jinkp/outlook-go-mcp/internal/logging"
	"github.com/jinkp/outlook-go-mcp/internal/mcp"
	"github.com/jinkp/outlook-go-mcp/internal/outlook"
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
	newLogger        func(string) (*slog.Logger, error)
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

	return root
}

// newMCPCmd returns the `outlook-mcp mcp` subcommand.
//
// CRITICAL: This command MUST NOT write ANYTHING to stdout before the server starts.
// The MCP stdio transport owns stdout entirely. All diagnostics go to stderr.
func newMCPCmd() *cobra.Command {
	var configPath string

	cmd := &cobra.Command{
		Use:          "mcp",
		Short:        "Start the Outlook MCP stdio server (for use with opencode, Claude, etc.)",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Redirect default logger to stderr — belt-and-suspenders guard.
			log.SetOutput(os.Stderr)

			code := run(configPath, os.Stderr, productionDeps())
			if code != 0 {
				os.Exit(code)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&configPath, "config", "config.yaml", "Path to YAML configuration file")
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

func run(configPath string, stderr io.Writer, deps bootstrapDeps) int {
	app, err := bootstrap(configPath, deps)
	if err != nil {
		reportBootstrapError(stderr, err)
		return 1
	}

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
	defer app.executor.Stop()

	app.logger.Info("Outlook MCP server starting",
		slog.String("config_path", app.configPath),
		slog.Int("tool_count", len(mcp.ToolDefinitions())),
	)

	if err := app.server.Serve(ctx); err != nil && !errors.Is(err, context.Canceled) {
		app.logger.Error("Outlook MCP server stopped with error", slog.Any("error", err))
		return 1
	}

	return 0
}

func bootstrap(configPath string, deps bootstrapDeps) (*application, error) {
	cfg, err := deps.loadConfig(configPath)
	if err != nil {
		return nil, &bootstrapError{stage: stageConfigLoad, err: err}
	}

	logger, err := deps.newLogger(cfg.Logging.Level)
	if err != nil {
		return nil, &bootstrapError{stage: stageLoggerInit, err: err}
	}

	session := deps.newSession()
	executor := deps.newExecutor(session)
	if err := executor.Start(); err != nil {
		return nil, &bootstrapError{stage: stageExecutorStart, err: err, logger: logger}
	}

	handlers := mcp.Handlers{
		Mail:     deps.newMailStore(executor),
		Calendar: deps.newCalendarStore(executor),
		Policy:   deps.newPolicyGate(*cfg),
		Config:   cfg,
		Logger:   logger,
	}

	server := deps.newServer(&handlers)
	server.RegisterTools()

	return &application{
		configPath: configPath,
		config:     cfg,
		logger:     logger,
		executor:   executor,
		server:     server,
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

func productionDeps() bootstrapDeps {
	return bootstrapDeps{
		loadConfig: config.Load,
		newLogger:  logging.New,
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
