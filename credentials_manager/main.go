package main

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type CredentialConfig struct {
	Name        string
	Credentials map[string]Credential
}

type Credential struct {
	Value string
	Show  bool
}

type Entry struct {
	Name    string
	Configs []CredentialConfig
}

type viewState int

const (
	listView viewState = iota
	entryDetailView
	addEntryView
)

type model struct {
	entries      []Entry
	cursor       int
	configCursor int
	viewState    viewState
	width        int
	height       int
}

func initialModel() model {
	entries := []Entry{
		{
			Name: "AWS Account",
			Configs: []CredentialConfig{
				{Name: "dev", Credentials: map[string]Credential{"AWS_ACCESS_KEY": {"dev-key-123", true}, "AWS_SECRET": {"dev-secret", false}}},
				{Name: "prod", Credentials: map[string]Credential{"AWS_ACCESS_KEY": {"prod-key-456", true}, "AWS_SECRET": {"prod-secret", false}}},
			},
		},
		{
			Name: "Database",
			Configs: []CredentialConfig{
				{Name: "dev", Credentials: map[string]Credential{"DB_HOST": {"localhost", true}, "DB_USER": {"dev", false}}},
				{Name: "test", Credentials: map[string]Credential{"DB_HOST": {"test.db.com", true}, "DB_USER": {"test", false}}},
				{Name: "prod", Credentials: map[string]Credential{"DB_HOST": {"prod.db.com", true}, "DB_USER": {"prod", false}}},
			},
		},
	}

	return model{
		entries:   entries,
		viewState: listView,
	}
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		switch m.viewState {
		case listView:
			return m.updateListView(msg)
		case entryDetailView:
			return m.updateEntryDetailView(msg)
		}
	}

	return m, nil
}

func (m model) updateListView(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q":
		return m, tea.Quit

	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}

	case "down", "j":
		if m.cursor < len(m.entries)-1 {
			m.cursor++
		}

	case "enter", "l":
		m.viewState = entryDetailView
		m.configCursor = 0
	}

	return m, nil
}

func (m model) updateEntryDetailView(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q":
		return m, tea.Quit

	case "esc", "h":
		m.viewState = listView

	case "up", "k":
		if m.configCursor > 0 {
			m.configCursor--
		}

	case "down", "j":
		if m.cursor < len(m.entries) && m.configCursor < len(m.entries[m.cursor].Configs)-1 {
			m.configCursor++
		}

	case "enter":
		// Copy selected config to clipboard (placeholder)
		// You'd use a clipboard library here
		// For now, just show which would be copied
		setCredentialInSession(m.entries[m.cursor].Configs[m.configCursor].Credentials)
	}

	return m, nil
}

func (m model) View() string {
	switch m.viewState {
	case listView:
		return m.renderListView()
	case entryDetailView:
		return m.renderEntryDetailView()
	default:
		return "Unknown view"
	}
}

func (m model) renderListView() string {
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("205")).
		Padding(1, 0)

	selectedStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("170")).
		Bold(true).
		PaddingLeft(2)

	normalStyle := lipgloss.NewStyle().
		PaddingLeft(2)

	s := titleStyle.Render("Credential Manager") + "\n\n"

	for i, entry := range m.entries {
		cursor := " "
		if m.cursor == i {
			cursor = ">"
			s += selectedStyle.Render(fmt.Sprintf("%s %s (%d configs)", cursor, entry.Name, len(entry.Configs)))
		} else {
			s += normalStyle.Render(fmt.Sprintf("%s %s (%d configs)", cursor, entry.Name, len(entry.Configs)))
		}
		s += "\n"
	}

	s += "\n\n" + lipgloss.NewStyle().Faint(true).Render("↑/↓ or j/k: navigate • enter/l: select • q: quit")

	return s
}

func (m model) renderEntryDetailView() string {
	if m.cursor >= len(m.entries) {
		return "Invalid entry"
	}

	entry := m.entries[m.cursor]

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("205")).
		Padding(1, 0)

	selectedStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("170")).
		Bold(true).
		PaddingLeft(2)

	normalStyle := lipgloss.NewStyle().
		PaddingLeft(2)

	credStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		PaddingLeft(4)

	s := titleStyle.Render(fmt.Sprintf("%s - Configurations", entry.Name)) + "\n\n"

	for i, config := range entry.Configs {
		cursor := " "
		if m.configCursor == i {
			cursor = ">"
			s += selectedStyle.Render(fmt.Sprintf("%s %s", cursor, config.Name))
		} else {
			s += normalStyle.Render(fmt.Sprintf("%s %s", cursor, config.Name))
		}

		if m.configCursor == i {
			for key, value := range config.Credentials {
				if value.Show {
					s += credStyle.Render(fmt.Sprintf("  %s: %s", key, value.Value))
				}
			}
		}
		s += "\n"
	}

	s += "\n\n" + lipgloss.NewStyle().Faint(true).Render("↑/↓ or j/k: navigate • enter: copy • esc: back • q: quit")

	return s
}

func setCredentialInSession(cred map[string]Credential) error {
	for key, value := range cred {
		cmd := exec.Command(fmt.Sprintf("export %s=%s", key, value.Value))
		err := cmd.Run()
		if err != nil {
			return fmt.Errorf("failed to set credential")
		}
	}
	return nil
}

func main() {
	if runtime.GOOS != "darwin" {
		fmt.Println("only supports macOS")
		os.Exit(1)
	}

	p := tea.NewProgram(initialModel())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v", err)
		os.Exit(1)
	}
}
