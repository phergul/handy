package main

import (
	"fmt"
	"log"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const (
	GhosttyConfigDir = ""
	ZellijConfigDir  = ""

	Blue       = "27"
	Turquiose  = "30"
	Green      = "49"
	RoyalBlue  = "63"
	Aquamarine = "79"
	Pink       = "169"
	Grey       = "240"
)

// order (nvim, zellij, ghostty)
type Programs []string

var programs = Programs{"nvim", "zellij", "ghostty"}

type ThemeList struct {
	Nvim    string `json:"nvim"`
	Zellij  string `json:"zellij"`
	Ghostty string `json:"ghostty"`
}

type Entry struct {
	Name   string
	Themes ThemeList
}

type ViewState int

const (
	listView ViewState = iota
	addEntry
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
		case addEntry:
			return m.updateAddEntry(msg)
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
		m.viewState = addEntry
	}

	return m, nil
}

func (m model) updateAddEntry(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q":
		return m, tea.Quit

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
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(Aquamarine)).Padding(1, 0)
	normalStyle := lipgloss.NewStyle().PaddingLeft(2)
	selectedStyle := lipgloss.NewStyle().Bold(true).PaddingLeft(2).Foreground(lipgloss.Color(Turquiose))
	themeStyle := lipgloss.NewStyle().PaddingLeft(4).Foreground(lipgloss.Color(Grey))

	t := titleStyle.Render("Theme Switcher:") + "\n"

	for i, entry := range m.entries {
		cursor := " "
		if m.cursor == i {
			cursor = "+"
			t += selectedStyle.Render(fmt.Sprintf("%s %s", cursor, entry.Name))

			t += "\n" + themeStyle.Render(fmt.Sprintf("  %s", formatThemes(entry.Themes)))
			// theme := reflect.TypeOf(entry.Themes)
			// for i := 0; i < theme.NumField(); i++ {
			// 	field := theme.Field(i)
			// 	t += "\n" + themeStyle.Render(fmt.Sprintf("  %s: %s", field.Name, entry.Themes.Nvim))
			// }
		} else {
			t += normalStyle.Render(fmt.Sprintf("%s %s", cursor, entry.Name))
		}

		t += "\n"
	}

	t += "\n\n" + lipgloss.NewStyle().Faint(true).Render("↑/↓ or j/k: navigate • enter/l: select • q: quit")

	return t
}

func formatThemes(themes ThemeList) string {
	structString := fmt.Sprintf("%+v", themes)

	structString = strings.Replace(structString, "{", "", 1)
	structString = strings.Replace(structString, "}", "", 1)

	structString = strings.Replace(structString, " ", "\n  ", 2)
	structString = strings.Replace(structString, ":", ": ", 3)

	return structString
}

func initialModel() model {
	//test
	e := []Entry{
		{Name: "blue", Themes: ThemeList{Nvim: "tokyonight", Zellij: "nord", Ghostty: "Argonaut"}},
		{Name: "green", Themes: ThemeList{Nvim: "tokyonight", Zellij: "nord", Ghostty: "Argonaut"}},
		{Name: "red", Themes: ThemeList{Nvim: "tokyonight", Zellij: "nord", Ghostty: "Argonaut"}},
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
