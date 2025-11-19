package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const (
	StorageFile = ".config/theme_switcher/theme_entries.json"

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

type ThemeList struct {
	Nvim    string `json:"nvim"`
	Zellij  string `json:"zellij"`
	Ghostty string `json:"ghostty"`
}

type Entry struct {
	Name   string    `json:"name"`
	Themes ThemeList `json:"theme_list"`
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

	case "enter":
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

	case "enter":
		m.saveEntry()
		m.viewState = listView

	case "esc":
		for i := 0; i < len(m.inputs); i++ {
			m.inputs[i].Reset()
		}
		m.viewState = listView

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

	t += "\n\n" + lipgloss.NewStyle().Faint(true).Render("j/k: navigate • enter: select • a: add entry • q: quit")

	return t
}

func (m model) renderAddEntry() string {
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(Celadon)).Padding(1, 0)
	borderStyle := lipgloss.NewStyle().Bold(true).Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color(Teal))
	inputTitleStyle := lipgloss.NewStyle().Bold(true).PaddingLeft(2).Foreground(lipgloss.Color(Purple))
	inputTextStyle := lipgloss.NewStyle()

	t := titleStyle.Render("Add Theme Entry:") + "\n"

	formLines := []string{
		inputTitleStyle.Render(" Entry Name: ") + inputTextStyle.Render(m.inputs[0].View()) + "\n",
		inputTitleStyle.Render("Neovim Theme: ") + inputTextStyle.Render(m.inputs[1].View()) + "\n",
		inputTitleStyle.Render("Zellij Theme: ") + inputTextStyle.Render(m.inputs[2].View()) + "\n",
		inputTitleStyle.Render("Ghostty Theme: ") + inputTextStyle.Render(m.inputs[3].View()),
	}

	t += borderStyle.Render(formLines...)
	// 	t += inputTitleStyle.Render(fmt.Sprintf(`Entry Name: %s
	//
	// Neovim Theme: %s
	//
	// Zellij Theme: %s
	//
	// Ghostty Theme: %s`, m.inputs[0].View(), m.inputs[1].View(), m.inputs[2].View(), m.inputs[3].View()))

	t += "\n\n" + lipgloss.NewStyle().Faint(true).Render("tab/shift-tab: navigate • enter: select • q: quit")

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

func (m *model) saveEntry() error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get user home dir: %w", err)
	}
	entriesFile := filepath.Join(homeDir, StorageFile)

	dir := filepath.Dir(entriesFile)
	if err = os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create dir: %w", err)
	}

	var existingEntries []Entry

	if _, err := os.Stat(entriesFile); err == nil {
		data, err := os.ReadFile(entriesFile)
		if err != nil {
			return fmt.Errorf("failed to read entries file: %w", err)
		}
		if len(data) > 0 {
			if err := json.Unmarshal(data, &existingEntries); err != nil {
				return fmt.Errorf("failed to parse entries file: %w", err)
			}
		}
	}

	newEntry := Entry{
		Name: m.inputs[0].Value(),
		Themes: ThemeList{
			Nvim:    m.inputs[1].Value(),
			Zellij:  m.inputs[2].Value(),
			Ghostty: m.inputs[3].Value(),
		},
	}

	existingEntries = append(existingEntries, newEntry)

	data, err := json.MarshalIndent(existingEntries, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal entries: %w", err)
	}

	if err := os.WriteFile(entriesFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write entries file: %w", err)
	}

	m.entries = existingEntries

	for i := range m.inputs {
		m.inputs[i].Reset()
	}

	return nil
}

func loadEntries() []Entry {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatalf("failed to get user home dir: %v", err)
	}
	entriesFile := filepath.Join(homeDir, StorageFile)

	var entries []Entry
	data, err := os.ReadFile(entriesFile)
	if err != nil {
		if !os.IsNotExist(err) {
			return entries
		}
		log.Fatalf("failed to read entries file: %v", err)
	}

	if len(data) > 0 {
		if err := json.Unmarshal(data, &entries); err != nil {
			log.Fatalf("failed to unmarshal entries: %v", err)
		}
	}

	return entries
}

func initialModel() model {
	//test
	// e := []Entry{
	// {Name: "blue", Themes: ThemeList{Nvim: "tokyonight", Zellij: "nord", Ghostty: "Argonaut"}},
	// {Name: "green", Themes: ThemeList{Nvim: "tokyonight", Zellij: "nord", Ghostty: "Argonaut"}},
	// {Name: "red", Themes: ThemeList{Nvim: "tokyonight", Zellij: "nord", Ghostty: "Argonaut"}},
	// }

	var inputs []textinput.Model = make([]textinput.Model, 4)
	inputs[0] = textinput.New()
	inputs[0].Width = 25
	inputs[0].Prompt = ""
	inputs[0].Focus()

	inputs[1] = textinput.New()
	inputs[1].Prompt = ""
	inputs[1].Width = 25

	inputs[2] = textinput.New()
	inputs[2].Prompt = ""
	inputs[2].Width = 25

	inputs[3] = textinput.New()
	inputs[3].Prompt = ""
	inputs[3].Width = 25

	return model{
		entries: loadEntries(),
		inputs:  inputs,
	}
}

func main() {
	p := tea.NewProgram(initialModel())
	if _, err := p.Run(); err != nil {
		log.Fatalf("Error: %v", err)
	}
}
