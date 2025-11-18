package main

import (
	"fmt"
	"log"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const (
	GhosttyConfigDir = ""
	ZellijConfigDir  = ""

	Blue    = "#9AC2C9"
	Pink    = "#FED4E7"
	Orange  = "#E26D5C"
	Teal    = "#006C67"
	Celadon = "#B9D8C2"
	Purple  = "#745C97"
	Indigo  = "#39375B"
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
	inputs        []textinput.Model
	focusedInput  int
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
		// TODO

	case "a":
		m.viewState = addEntry
	}

	return m, nil
}

func (m model) updateAddEntry(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd = make([]tea.Cmd, len(m.inputs))

	switch msg.String() {
	case "ctrl+c", "q":
		return m, tea.Quit

	case "enter", "l":
		// TODO

	case "tab":
		m.focusedInput = (m.focusedInput + 1) % len(m.inputs)

	case "shift+tab":
		m.focusedInput--
		if m.focusedInput < 0 {
			m.focusedInput = len(m.inputs) - 1
		}
	}

	for i := range m.inputs {
		m.inputs[i].Blur()
	}
	m.inputs[m.focusedInput].Focus()

	for i := range m.inputs {
		m.inputs[i], cmds[i] = m.inputs[i].Update(msg)
	}
	return m, tea.Batch(cmds...)
}

func (m model) View() string {
	switch m.viewState {
	case listView:
		return m.renderListView()
	case addEntry:
		return m.renderAddEntry()
	default:
		return "Unknown view"
	}
}

func (m model) renderListView() string {
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(Celadon)).Padding(1, 0)
	normalStyle := lipgloss.NewStyle().PaddingLeft(2)
	selectedStyle := lipgloss.NewStyle().Bold(true).PaddingLeft(2).Foreground(lipgloss.Color(Purple))
	themeStyle := lipgloss.NewStyle().PaddingLeft(4).Foreground(lipgloss.Color(Indigo))

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

func (m model) renderAddEntry() string {
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(Celadon)).Padding(1, 0)
	inputStyle := lipgloss.NewStyle().Bold(true).PaddingLeft(2).Border(lipgloss.RoundedBorder())

	t := titleStyle.Render("Add Theme Entry:") + "\n"

	t += inputStyle.Render(fmt.Sprintf(`Entry Name: %s

Neovim Theme: %s

Zellij Theme: %s

Ghostty Theme: %s`, m.inputs[0].View(), m.inputs[1].View(), m.inputs[2].View(), m.inputs[3].View()))

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

	var inputs []textinput.Model = make([]textinput.Model, 3)
	inputs[0] = textinput.New()
	inputs[0].Width = 25
	inputs[0].Prompt = ""
	inputs[0].Focus()

	inputs[1] = textinput.New()
	inputs[1].Width = 25
	inputs[1].Prompt = ""

	inputs[2] = textinput.New()
	inputs[2].Width = 25
	inputs[2].Prompt = ""

	inputs[3] = textinput.New()
	inputs[3].Width = 25
	inputs[3].Prompt = ""

	return model{
		entries: e,
		inputs:  inputs,
	}
}

func main() {
	p := tea.NewProgram(initialModel())
	if _, err := p.Run(); err != nil {
		log.Fatalf("Error: %v", err)
	}
}
