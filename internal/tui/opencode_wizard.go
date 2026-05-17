package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/jinkp/outlook-go-mcp/internal/opencode"
)

// OpenCodeWizardScreen represents the current wizard screen.
type OpenCodeWizardScreen int

const (
	WizardScreenScope   OpenCodeWizardScreen = iota // Screen 0: choose global or local scope
	WizardScreenConfirm                             // Screen 1: confirm before writing
	WizardScreenDone                                // Screen 2: success
	WizardScreenError                               // Screen 3: error
)

// OpenCodeWizardModel is the Bubbletea model for the outlook-mcp setup opencode wizard.
type OpenCodeWizardModel struct {
	screen    OpenCodeWizardScreen
	scope     opencode.Scope
	cursor    int // 0 = global, 1 = local
	err       error
	cancelled bool
}

// NewOpenCodeWizardModel creates a fresh wizard model with default state.
func NewOpenCodeWizardModel() OpenCodeWizardModel {
	return OpenCodeWizardModel{
		screen: WizardScreenScope,
		scope:  opencode.ScopeGlobal,
		cursor: 0,
	}
}

// Cancelled returns true if the user cancelled the wizard.
func (m OpenCodeWizardModel) Cancelled() bool {
	return m.cancelled
}

// Done returns true if the wizard completed successfully.
func (m OpenCodeWizardModel) Done() bool {
	return m.screen == WizardScreenDone
}

// Error returns true if the wizard encountered an error.
func (m OpenCodeWizardModel) Error() bool {
	return m.screen == WizardScreenError
}

// ErrorMessage returns the error message, if any.
func (m OpenCodeWizardModel) ErrorMessage() string {
	if m.err != nil {
		return m.err.Error()
	}
	return ""
}

// Init satisfies the tea.Model interface — no initial command needed.
func (m OpenCodeWizardModel) Init() tea.Cmd {
	return nil
}

// Update handles key events and state transitions.
func (m OpenCodeWizardModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			m.cancelled = true
			return m, tea.Quit
		}

		switch m.screen {
		case WizardScreenScope:
			return m.updateScope(msg)
		case WizardScreenConfirm:
			return m.updateConfirm(msg)
		case WizardScreenDone, WizardScreenError:
			switch msg.String() {
			case "enter", "q":
				return m, tea.Quit
			}
		}
	}
	return m, nil
}

func (m OpenCodeWizardModel) updateScope(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
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
			m.scope = opencode.ScopeGlobal
		} else {
			m.scope = opencode.ScopeLocal
		}
		m.screen = WizardScreenConfirm
	}
	return m, nil
}

func (m OpenCodeWizardModel) updateConfirm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "enter":
		if err := opencode.Save(m.scope); err != nil {
			m.err = err
			m.screen = WizardScreenError
		} else {
			m.screen = WizardScreenDone
		}
	case "n", "q":
		m.cancelled = true
		return m, tea.Quit
	}
	return m, nil
}

// View renders the current wizard screen.
func (m OpenCodeWizardModel) View() string {
	if m.cancelled {
		return ""
	}

	switch m.screen {
	case WizardScreenScope:
		return m.viewScope()
	case WizardScreenConfirm:
		return m.viewConfirm()
	case WizardScreenDone:
		return m.viewDone()
	case WizardScreenError:
		return m.viewError()
	default:
		return ""
	}
}

func (m OpenCodeWizardModel) viewScope() string {
	scopes := []string{"Global (~/.config/opencode/opencode.json)", "Local (./opencode.json)"}
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
		"outlook-mcp setup opencode",
		"Step 1 of 2 — Choose where to write the MCP configuration.",
		body,
		"",
		hint,
	)
}

func (m OpenCodeWizardModel) viewConfirm() string {
	var targetPath string
	switch m.scope {
	case opencode.ScopeGlobal:
		p, _ := opencode.GlobalPath()
		targetPath = p
	case opencode.ScopeLocal:
		targetPath = opencode.LocalPath()
	}

	preview := fmt.Sprintf(`{
  "mcp": {
    "outlook-mcp": {
      "type": "local",
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
		"outlook-mcp setup opencode",
		"Step 2 of 2 — Confirm changes.",
		body,
		"",
		"y/Enter to confirm • n to cancel",
	)
}

func (m OpenCodeWizardModel) viewDone() string {
	var targetPath string
	switch m.scope {
	case opencode.ScopeGlobal:
		p, _ := opencode.GlobalPath()
		targetPath = p
	case opencode.ScopeLocal:
		targetPath = opencode.LocalPath()
	}

	body := successStyle.Render("outlook-mcp MCP server written to:\n  ") + inputStyle.Render(targetPath)
	return renderWizardScreen(
		"outlook-mcp setup opencode",
		"Done.",
		body,
		"",
		"Press Enter to exit",
	)
}

func (m OpenCodeWizardModel) viewError() string {
	return renderWizardScreen(
		"outlook-mcp setup opencode",
		"Setup failed.",
		errorStyle.Render(m.ErrorMessage()),
		"",
		"Press Enter to exit",
	)
}

func renderWizardScreen(title, subtitle, body, errText, hint string) string {
	parts := []string{
		titleStyle.Render(title),
		dimStyle.Render(subtitle),
		"",
		body,
	}

	if strings.TrimSpace(errText) != "" {
		parts = append(parts, "", errorStyle.Render(errText))
	}

	if strings.TrimSpace(hint) != "" {
		parts = append(parts, "", dimStyle.Render(hint))
	}

	return lipgloss.NewStyle().Padding(1, 2).Render(strings.Join(parts, "\n"))
}
