package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/jinkp/outlook-go-mcp/internal/claude"
)

// ClaudeWizardScope selects which Claude Code config file to target.
type ClaudeWizardScope int

const (
	ClaudeScopeGlobal ClaudeWizardScope = iota // ~/.claude.json
	ClaudeScopeLocal                           // ./.claude/settings.json
)

// ClaudeWizardScreen represents the current wizard screen.
type ClaudeWizardScreen int

const (
	ClaudeScreenScope   ClaudeWizardScreen = iota // Screen 0: choose global or local scope
	ClaudeScreenConfirm                           // Screen 1: confirm before writing
	ClaudeScreenDone                              // Screen 2: success
	ClaudeScreenError                             // Screen 3: error
)

// ClaudeWizardModel is the Bubbletea model for the outlook-mcp setup claude wizard.
type ClaudeWizardModel struct {
	screen    ClaudeWizardScreen
	scope     ClaudeWizardScope
	cursor    int // 0 = global, 1 = local
	err       error
	cancelled bool
}

// NewClaudeWizardModel creates a fresh wizard model with default state.
func NewClaudeWizardModel() ClaudeWizardModel {
	return ClaudeWizardModel{
		screen: ClaudeScreenScope,
		scope:  ClaudeScopeGlobal,
		cursor: 0,
	}
}

// Cancelled returns true if the user cancelled the wizard.
func (m ClaudeWizardModel) Cancelled() bool {
	return m.cancelled
}

// Done returns true if the wizard completed successfully.
func (m ClaudeWizardModel) Done() bool {
	return m.screen == ClaudeScreenDone
}

// Error returns true if the wizard encountered an error.
func (m ClaudeWizardModel) Error() bool {
	return m.screen == ClaudeScreenError
}

// ErrorMessage returns the error message, if any.
func (m ClaudeWizardModel) ErrorMessage() string {
	if m.err != nil {
		return m.err.Error()
	}
	return ""
}

// Init satisfies the tea.Model interface — no initial command needed.
func (m ClaudeWizardModel) Init() tea.Cmd {
	return nil
}

// Update handles key events and state transitions.
func (m ClaudeWizardModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			m.cancelled = true
			return m, tea.Quit
		}

		switch m.screen {
		case ClaudeScreenScope:
			return m.updateScope(msg)
		case ClaudeScreenConfirm:
			return m.updateConfirm(msg)
		case ClaudeScreenDone, ClaudeScreenError:
			switch msg.String() {
			case "enter", "q":
				return m, tea.Quit
			}
		}
	}
	return m, nil
}

func (m ClaudeWizardModel) updateScope(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < 1 {
			m.cursor++
		}
	case "enter", "tab":
		if m.cursor == 0 {
			m.scope = ClaudeScopeGlobal
		} else {
			m.scope = ClaudeScopeLocal
		}
		m.screen = ClaudeScreenConfirm
	}
	return m, nil
}

func (m ClaudeWizardModel) updateConfirm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "enter":
		path, err := m.targetPath()
		if err != nil {
			m.err = err
			m.screen = ClaudeScreenError
			return m, nil
		}
		if err := claude.Save(path); err != nil {
			m.err = err
			m.screen = ClaudeScreenError
		} else {
			m.screen = ClaudeScreenDone
		}
	case "n", "q":
		m.cancelled = true
		return m, tea.Quit
	}
	return m, nil
}

// targetPath returns the config file path for the selected scope.
func (m ClaudeWizardModel) targetPath() (string, error) {
	switch m.scope {
	case ClaudeScopeGlobal:
		return claude.GlobalPath()
	case ClaudeScopeLocal:
		return claude.LocalPath()
	default:
		return "", fmt.Errorf("unknown scope")
	}
}

// View renders the current wizard screen.
func (m ClaudeWizardModel) View() string {
	if m.cancelled {
		return ""
	}

	switch m.screen {
	case ClaudeScreenScope:
		return m.viewScope()
	case ClaudeScreenConfirm:
		return m.viewConfirm()
	case ClaudeScreenDone:
		return m.viewDone()
	case ClaudeScreenError:
		return m.viewError()
	default:
		return ""
	}
}

func (m ClaudeWizardModel) viewScope() string {
	scopes := []string{"Global (~/.claude.json)", "Local (.claude/settings.json)"}
	lines := make([]string, len(scopes))
	for i, s := range scopes {
		if i == m.cursor {
			lines[i] = promptStyle.Render("› ") + inputStyle.Render(s)
		} else {
			lines[i] = "  " + dimStyle.Render(s)
		}
	}

	body := strings.Join(lines, "\n")
	hint := "↑/↓ navigate • Enter to select • Ctrl+C to cancel"

	return renderWizardScreen(
		"outlook-mcp setup claude",
		"Step 1 of 2 — Choose where to write the MCP configuration.",
		body,
		"",
		hint,
	)
}

func (m ClaudeWizardModel) viewConfirm() string {
	targetPath, err := m.targetPath()
	if err != nil {
		targetPath = "(error resolving path)"
	}

	preview := fmt.Sprintf(`{
  "mcpServers": {
    "outlook-mcp": {
      "command": "outlook-mcp",
      "args": ["mcp"]
    }
  }
}`)

	body := strings.Join([]string{
		dimStyle.Render("Target: ") + inputStyle.Render(targetPath),
		"",
		dimStyle.Render("Will merge into file:"),
		preview,
	}, "\n")

	return renderWizardScreen(
		"outlook-mcp setup claude",
		"Step 2 of 2 — Confirm changes.",
		body,
		"",
		"y/Enter to confirm • n to cancel",
	)
}

func (m ClaudeWizardModel) viewDone() string {
	targetPath, err := m.targetPath()
	if err != nil {
		targetPath = "(unknown)"
	}

	body := successStyle.Render("outlook-mcp MCP server written to:\n  ") + inputStyle.Render(targetPath)
	footer := dimStyle.Render("Claude Code configured. Restart Claude Code to apply.")

	content := strings.Join([]string{body, "", footer}, "\n")

	return renderWizardScreen(
		"outlook-mcp setup claude",
		"Done.",
		content,
		"",
		"Press Enter to exit",
	)
}

func (m ClaudeWizardModel) viewError() string {
	return renderWizardScreen(
		"outlook-mcp setup claude",
		"Setup failed.",
		errorStyle.Render(m.ErrorMessage()),
		"",
		"Press Enter to exit",
	)
}
