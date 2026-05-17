package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/jinkp/outlook-go-mcp/internal/version"
)

// MenuItem is a single entry in the TUI main menu.
type MenuItem struct {
	Label       string
	Description string
	Command     string // CLI subcommand to dispatch (e.g. "setup opencode")
}

// MenuResult holds the outcome of the TUI menu interaction.
type MenuResult struct {
	Command   string
	Cancelled bool
}

// MenuModel is the Bubbletea model for the main TUI menu.
type MenuModel struct {
	items    []MenuItem
	cursor   int
	result   MenuResult
	quitting bool
}

var menuItems = []MenuItem{
	{Label: "Setup OpenCode", Description: "Register outlook-mcp as MCP server in opencode.json", Command: "setup opencode"},
	{Label: "Setup Claude Code", Description: "Register outlook-mcp as MCP server in Claude Code config", Command: "setup claude"},
	{Label: "Create config", Description: "Generate config.yaml from the built-in example", Command: "setup config"},
}

// NewMenuModel creates a fresh menu model.
func NewMenuModel() MenuModel {
	return MenuModel{
		items:  menuItems,
		cursor: 0,
	}
}

// Result returns the selected menu result.
func (m MenuModel) Result() MenuResult {
	return m.result
}

// Init satisfies tea.Model — no initial command needed.
func (m MenuModel) Init() tea.Cmd {
	return nil
}

// Update handles navigation and selection.
func (m MenuModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q", "esc":
			m.result = MenuResult{Cancelled: true}
			m.quitting = true
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
			return m, nil
		case "down", "j":
			if m.cursor < len(m.items)-1 {
				m.cursor++
			}
			return m, nil
		case "enter":
			m.result = MenuResult{Command: m.items[m.cursor].Command}
			m.quitting = true
			return m, tea.Quit
		}
	}
	return m, nil
}

// View renders the main menu.
func (m MenuModel) View() string {
	if m.quitting {
		return ""
	}

	var b strings.Builder

	b.WriteString(renderMenuBrand())
	b.WriteString("\n\n")

	for i, item := range m.items {
		if i == m.cursor {
			b.WriteString(menuSelectedStyle.Render(fmt.Sprintf(" > %s", item.Label)) + "\n")
			b.WriteString(menuDimStyle.Render(fmt.Sprintf("   %s", item.Description)) + "\n")
		} else {
			b.WriteString(menuNormalStyle.Render(fmt.Sprintf("   %s", item.Label)) + "\n")
		}
	}

	b.WriteString("\n")
	b.WriteString(menuDimStyle.Render("  ↑/↓ navigate  enter select  q quit"))

	return b.String()
}

func renderMenuBrand() string {
	title := fmt.Sprintf(" outlook-mcp  %s ", version.Version)
	subtitle := " Outlook Desktop → AI clients via MCP "

	content := lipgloss.JoinVertical(lipgloss.Left, title, subtitle)

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(menuAccentColor).
		Padding(0, 1).
		Render(content)
}

var (
	menuAccentColor   = lipgloss.Color("6")
	menuSelectedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("6")).Bold(true)
	menuNormalStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("7"))
	menuDimStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
)
