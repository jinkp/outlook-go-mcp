package tui

import (
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
)

// ConfigWizardScreen represents the current screen in the config wizard.
type ConfigWizardScreen int

const (
	ConfigScreenPath     ConfigWizardScreen = iota // Step 1: where to write config.yaml
	ConfigScreenAttDir                             // Step 2: attachment_dir absolute path
	ConfigScreenConfirm                            // Step 3: confirm + preview
	ConfigScreenDone                               // Done
	ConfigScreenError                              // Error
)

// ConfigWizardModel is the Bubbletea model for the config generation wizard.
type ConfigWizardModel struct {
	screen    ConfigWizardScreen
	pathInput textinput.Model
	attInput  textinput.Model
	err       error
	cancelled bool
}

// NewConfigWizardModel creates a fresh config wizard model.
func NewConfigWizardModel() ConfigWizardModel {
	pathInput := textinput.New()
	pathInput.Placeholder = "config.yaml"
	pathInput.SetValue("config.yaml")
	pathInput.Prompt = "> "
	pathInput.CharLimit = 260
	pathInput.Width = 48
	pathInput.PromptStyle = promptStyle
	pathInput.TextStyle = inputStyle
	pathInput.PlaceholderStyle = dimStyle
	pathInput.Cursor.Style = inputStyle
	pathInput.Focus()

	attInput := textinput.New()
	attInput.Placeholder = `C:\OutlookMCP\attachments`
	attInput.SetValue(`C:\OutlookMCP\attachments`)
	attInput.Prompt = "> "
	attInput.CharLimit = 260
	attInput.Width = 48
	attInput.PromptStyle = promptStyle
	attInput.TextStyle = inputStyle
	attInput.PlaceholderStyle = dimStyle
	attInput.Cursor.Style = inputStyle

	return ConfigWizardModel{
		screen:    ConfigScreenPath,
		pathInput: pathInput,
		attInput:  attInput,
	}
}

// Cancelled returns true if the user cancelled the wizard.
func (m ConfigWizardModel) Cancelled() bool { return m.cancelled }

// Done returns true if config was written successfully.
func (m ConfigWizardModel) Done() bool { return m.screen == ConfigScreenDone }

// Error returns true if the wizard encountered an error.
func (m ConfigWizardModel) Error() bool { return m.screen == ConfigScreenError }

// ErrorMessage returns the error string.
func (m ConfigWizardModel) ErrorMessage() string {
	if m.err != nil {
		return m.err.Error()
	}
	return ""
}

// ConfigPath returns the resolved output path.
func (m ConfigWizardModel) ConfigPath() string {
	return strings.TrimSpace(m.pathInput.Value())
}

// Init satisfies tea.Model.
func (m ConfigWizardModel) Init() tea.Cmd {
	return textinput.Blink
}

// Update handles key events and screen transitions.
func (m ConfigWizardModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			m.cancelled = true
			return m, tea.Quit
		}

		switch m.screen {
		case ConfigScreenPath:
			return m.updatePath(msg)
		case ConfigScreenAttDir:
			return m.updateAttDir(msg)
		case ConfigScreenConfirm:
			return m.updateConfirm(msg)
		case ConfigScreenDone, ConfigScreenError:
			if msg.String() == "enter" || msg.String() == "q" {
				return m, tea.Quit
			}
		}
	}

	return m, nil
}

func (m ConfigWizardModel) updatePath(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter", "tab":
		p := strings.TrimSpace(m.pathInput.Value())
		if p == "" {
			p = "config.yaml"
			m.pathInput.SetValue(p)
		}
		m.pathInput.Blur()
		m.attInput.Focus()
		m.screen = ConfigScreenAttDir
		return m, nil
	}

	var cmd tea.Cmd
	m.pathInput, cmd = m.pathInput.Update(msg)
	return m, cmd
}

func (m ConfigWizardModel) updateAttDir(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter", "tab":
		dir := strings.TrimSpace(m.attInput.Value())
		if dir == "" {
			dir = `C:\OutlookMCP\attachments`
			m.attInput.SetValue(dir)
		}
		m.attInput.Blur()
		m.screen = ConfigScreenConfirm
		return m, nil
	}

	var cmd tea.Cmd
	m.attInput, cmd = m.attInput.Update(msg)
	return m, cmd
}

func (m ConfigWizardModel) updateConfirm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "enter":
		if err := writeConfig(m.ConfigPath(), strings.TrimSpace(m.attInput.Value())); err != nil {
			m.err = err
			m.screen = ConfigScreenError
		} else {
			m.screen = ConfigScreenDone
		}
	case "n", "q":
		m.cancelled = true
		return m, tea.Quit
	}
	return m, nil
}

// View renders the current screen.
func (m ConfigWizardModel) View() string {
	if m.cancelled {
		return ""
	}

	switch m.screen {
	case ConfigScreenPath:
		return renderWizardScreen(
			"outlook-mcp setup config",
			"Step 1 of 2 — Where should config.yaml be written?",
			m.pathInput.View(),
			"",
			"Enter to continue • Ctrl+C to cancel",
		)
	case ConfigScreenAttDir:
		return renderWizardScreen(
			"outlook-mcp setup config",
			"Step 2 of 2 — Absolute path for attachment downloads.",
			m.attInput.View(),
			"",
			"Enter to continue • Ctrl+C to cancel",
		)
	case ConfigScreenConfirm:
		preview := buildConfigPreview(strings.TrimSpace(m.attInput.Value()))
		body := strings.Join([]string{
			dimStyle.Render("Target: ") + inputStyle.Render(m.ConfigPath()),
			"",
			dimStyle.Render("Will write:"),
			preview,
		}, "\n")
		return renderWizardScreen(
			"outlook-mcp setup config",
			"Confirm — review and write config.yaml.",
			body,
			"",
			"y/Enter to write • n to cancel",
		)
	case ConfigScreenDone:
		body := successStyle.Render("config.yaml written to:\n  ") +
			inputStyle.Render(m.ConfigPath()) +
			"\n\n" +
			dimStyle.Render("Edit the file to enable write operations (create_draft, create_event).\nThen run:  outlook-mcp mcp --config "+m.ConfigPath())
		return renderWizardScreen("outlook-mcp setup config", "Done.", body, "", "Press Enter to exit")
	case ConfigScreenError:
		return renderWizardScreen(
			"outlook-mcp setup config",
			"Setup failed.",
			errorStyle.Render(m.ErrorMessage()),
			"",
			"Press Enter to exit",
		)
	default:
		return ""
	}
}

// buildConfigPreview returns a compact YAML preview shown before writing.
func buildConfigPreview(attDir string) string {
	return lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Render(fmt.Sprintf(`outlook:
  profile: "default"
security:
  allow_create_draft: false
  allow_create_event: false
paths:
  attachment_dir: "%s"
logging:
  level: "info"
limits:
  max_results: 50`, attDir))
}

// writeConfig writes a ready-to-use config.yaml to the given path.
func writeConfig(path, attDir string) error {
	content := fmt.Sprintf(`# outlook-mcp configuration
# Documentation: https://github.com/jinkp/outlook-go-mcp

# Outlook profile selection.
outlook:
  profile: "default"

# Security switches for write-capable operations.
# All write actions are denied unless explicitly enabled.
security:
  allow_send_email: false
  allow_create_draft: false
  allow_create_event: false
  allow_save_attachments: false

# Storage paths used by the server.
paths:
  # Absolute Windows path for attachment export.
  attachment_dir: %q

# Structured logging configuration.
logging:
  level: "info"

# Runtime safety limits.
limits:
  max_results: 50
`, attDir)

	return os.WriteFile(path, []byte(content), 0o644)
}
