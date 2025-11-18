package main

import (
	"fmt"
	"log"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const (
	GhosttyConfigDir = ""
	ZellijConfigDir  = ""

	Blue      = "27"
	Turquiose = "30"
	Green     = "49"
	RoyalBlue = "63"
	Pink      = "169"
	Grey      = "240"
)

// order (nvim, zellij, ghostty)
type Programs []string

var programs = Programs{"nvim", "zellij", "ghostty"}

type Entry struct {
	Name   string
	Themes []string
}

type ViewState int

const (
	listView ViewState = iota
)

type model struct {
	cursor        int
	viewState     ViewState
	width, height int
	entries       []Entry
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
		// TODO
	}

	return m, nil
}

func (m model) View() string {
	switch m.viewState {
	case listView:
		return m.renderListView()
	default:
		return "Unknown view"
	}
}

func (m model) renderListView() string {
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(Blue)).Padding(1, 0)
	normalStyle := lipgloss.NewStyle().PaddingLeft(2)
	selectedStyle := lipgloss.NewStyle().Bold(true).PaddingLeft(2).Foreground(lipgloss.Color(RoyalBlue))
	themeStyle := lipgloss.NewStyle().PaddingLeft(4).Foreground(lipgloss.Color(Grey))

	t := titleStyle.Render("Theme Switcher - Available:") + "\n\n"

	for i, entry := range m.entries {
		cursor := " "
		if m.cursor == i {
			cursor = ">"
			t += selectedStyle.Render(fmt.Sprintf("%s %s", cursor, entry.Name))

			for i, theme := range entry.Themes {
				t += "\n" + themeStyle.Render(fmt.Sprintf("  %s: %s", programs[i], theme))
			}
		} else {
			t += normalStyle.Render(fmt.Sprintf("%s %s", cursor, entry.Name))
		}

		t += "\n"
	}

	t += "\n\n" + lipgloss.NewStyle().Faint(true).Render("↑/↓ or j/k: navigate • enter/l: select • q: quit")

	return t
}

func initialModel() model {
	//test
	e := []Entry{
		{Name: "blue", Themes: []string{"tokyonight", "nord", "Argonaut"}},
		{Name: "green", Themes: []string{"tokyonight", "nord", "Argonaut"}},
		{Name: "red", Themes: []string{"tokyonight", "nord", "Argonaut"}},
	}

	return model{
		entries: e,
	}
}

func main() {
	p := tea.NewProgram(initialModel())
	if _, err := p.Run(); err != nil {
		log.Fatalf("Error: %v", err)
	}
}
