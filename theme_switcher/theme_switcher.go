package main

import (
	"log"

	tea "github.com/charmbracelet/bubbletea"
)

const (
	GhosttyConfigDir = ""
	ZellijConfigDir  = ""
)

type Entry struct {
	Name string
}

type ViewState int

const (
	listView ViewState = iota
)

type model struct {
	cursor        int
	width, height int
	entries       []Entry
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	return nil, nil
}

func (m model) View() string {
	return ""
}

func initialModel() model {
	e := []Entry{
		{Name: "blue"},
		{Name: "green"},
		{Name: "red"},
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
