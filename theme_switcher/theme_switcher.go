package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/google/go-cmp/cmp"
)

const (
	StorageDir  = ".config/theme_switcher"
	StorageFile = ".config/theme_switcher/theme_entries.json"

	GhosttyConfigFile = "Library/Application\\ Support/com.mitchellh.ghostty/config"
	ZellijConfigFile  = ".config/zellij/config.kdl"

	Blue    = "#9AC2C9"
	Pink    = "#FED4E7"
	Orange  = "#E26D5C"
	Teal    = "#006C67"
	Celadon = "#B9D8C2"
	Purple  = "#745C97"
	Indigo  = "#39375B"
)

var (
	homeDir     = os.Getenv("HOME")
	entriesFile = filepath.Join(homeDir, StorageFile)
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
		errs := applyThemes(m.entries[m.cursor])
		// TODO: show the errors in the model view
		for _, err := range errs {
			if err != nil {
				log.Println(err)
			}
		}

	case "a":
		m.viewState = addEntry

	case "d":
		if err := deleteEntry(m.entries[m.cursor]); err != nil {
			// TODO: show error in model view
			log.Println(err)
		}
		m.entries = loadEntries()
		if m.cursor > 0 {
			m.cursor--
		}

	case "e":
		// TODO: add entry edit function
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
		} else {
			t += normalStyle.Render(fmt.Sprintf("%s %s", cursor, entry.Name))
		}

		t += "\n"
	}

	t += "\n\n" + lipgloss.NewStyle().Faint(true).Render("j/k: navigate • enter: select • a: add entry • d: delete entry • q: quit")

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
	dir := filepath.Dir(entriesFile)
	if err := os.MkdirAll(dir, 0755); err != nil {
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

	if err := saveEntriesToFile(existingEntries); err != nil {
		return err
	}

	m.entries = existingEntries

	for i := range m.inputs {
		m.inputs[i].Reset()
	}

	return nil
}

func saveEntriesToFile(entries []Entry) error {
	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal entries: %w", err)
	}

	if err := os.WriteFile(entriesFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write entries file: %w", err)
	}

	return nil
}

func loadEntries() []Entry {
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

func deleteEntry(entry Entry) error {
	existingEntries := loadEntries()

	newEntries := make([]Entry, 0, len(existingEntries))
	for _, existingEntry := range existingEntries {
		if cmp.Equal(existingEntry, entry) {
			log.Println("Deleting entry:", existingEntry.Name)
			continue
		}
		newEntries = append(newEntries, existingEntry)
	}

	if len(existingEntries) == len(newEntries) {
		return fmt.Errorf("Entry (%s) not found for deletion", entry.Name)
	}

	if err := saveEntriesToFile(newEntries); err != nil {
		return err
	}

	return nil
}

func applyThemes(entry Entry) []error {
	var errs []error

	errs = append(errs, applyNvim(entry.Themes.Nvim))
	errs = append(errs, applyZellij(entry.Themes.Zellij))
	errs = append(errs, applyGhostty(entry.Themes.Ghostty))

	return errs
}

func applyNvim(theme string) error {
	if !strings.Contains(theme, ",") {
		return fmt.Errorf("[Nvim] theme could not be validated.")
	}

	themeFile := filepath.Join(homeDir, ".config/nvim/theme.conf")

	themeParts := strings.Split(theme, ",")
	colourschemeID := strings.TrimSpace(themeParts[0])
	themeName := strings.TrimSpace(themeParts[1])

	if colourschemeID == "" {
		colourschemeID = "nil"
	}

	content := colourschemeID + "\n" + themeName

	if err := os.WriteFile(themeFile, []byte(content), 0644); err != nil {
		return fmt.Errorf("[Nvim] failed to write to 'theme.conf': %w", err)
	}

	// neovim-remote is required
	/*
		TODO: add another way to communicate with neovim that requires no dependancies?
		make this the fallback eventually
	*/
	cmd := exec.Command("nvr", "--remote-send", ":ReloadTheme<CR>")
	cmd.Run() //ignore error if nvim isn’t running

	return nil
}

func applyZellij(theme string) error {
	configPath := filepath.Join(homeDir, ZellijConfigFile)
	data, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("[Zellij] failed to read zellij config file: %w", err)
	}

	content := string(data)

	replacement := fmt.Sprintf("theme \"%s\"", theme)
	themeRe := regexp.MustCompile(`(?m)^\s*theme\s+"[^"]+"\s*$`)

	if themeRe.MatchString(content) {
		content = themeRe.ReplaceAllString(content, replacement)
	} else {
		content = replacement + "\n" + content
	}

	if err = os.WriteFile(configPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("[Zellij] failed to write updated config: %w", err)
	}

	return nil
}

func applyGhostty(theme string) error {
	configPath := filepath.Join(homeDir, GhosttyConfigFile)
	data, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("[Ghostty] failed to read ghostty config file: %w", err)
	}

	content := string(data)

	replacement := fmt.Sprintf("theme = %s", theme)
	themeRe := regexp.MustCompile(`(?m)^\s*theme\s*=\s*\S+`)

	if themeRe.MatchString(content) {
		content = themeRe.ReplaceAllString(content, replacement)
	} else {
		content = replacement + "\n" + content
	}

	if err = os.WriteFile(configPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("[Ghostty] failed to write updated config: %w", err)
	}

	return nil
}

func initialModel() model {
	var inputs []textinput.Model = make([]textinput.Model, 4)
	inputs[0] = textinput.New()
	inputs[0].Width = 25
	inputs[0].Prompt = ""
	inputs[0].Placeholder = "entry-name"
	inputs[0].Focus()

	inputs[1] = textinput.New()
	inputs[1].Prompt = ""
	inputs[1].Width = 25
	inputs[1].Placeholder = "theme-source,theme-name"

	inputs[2] = textinput.New()
	inputs[2].Prompt = ""
	inputs[2].Width = 25
	inputs[2].Placeholder = "theme-name"

	inputs[3] = textinput.New()
	inputs[3].Prompt = ""
	inputs[3].Width = 25
	inputs[3].Placeholder = "theme-name"

	return model{
		entries: loadEntries(),
		inputs:  inputs,
	}
}

func main() {
	if runtime.GOOS != "darwin" {
		log.Fatalf("only available on macOS")
	}

	logLoc := filepath.Join(homeDir, StorageDir, "theme_switcher.log")
	os.Remove(logLoc)
	logFile, err := os.OpenFile(logLoc, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("Failed to open log file: %v", err)
	}
	defer logFile.Close()

	log.SetOutput(logFile)
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	log.Println("Application started")

	p := tea.NewProgram(initialModel())
	if _, err := p.Run(); err != nil {
		log.Fatalf("Error: %v", err)
	}
}
