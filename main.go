package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/reflow/wordwrap"
	"google.golang.org/api/calendar/v3"
)

/* BubbleTea Things */

var MAX_STR_LENGTH int = 150

var docStyle = lipgloss.NewStyle().Margin(1, 2)

type model struct {
	list list.Model
	// not sure if better to have it as list.Item instead
	selected EventWrapper
	keys     *mainKeyMap
	contentWidth int
}

type mainKeyMap struct {
	chooseItem key.Binding
}

func newKeyMap() *mainKeyMap {
	return &mainKeyMap{
		chooseItem: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("ENTER", "Details"),
		),
	}
}

// https://stackoverflow.com/questions/28800672/how-to-add-new-methods-to-an-existing-type-in-go
// conflicting Description
type EventWrapper struct {
	*calendar.Event
}

func (e EventWrapper) Date() string {
	date := ""

	if e.Start.DateTime != "" {
		date = e.Start.DateTime + " - " + e.End.DateTime
	} else {
		date = e.Start.Date + " - " + e.End.Date
	}

	return date
}

// implement list.Item interface
func (e EventWrapper) FilterValue() string {
	return e.Event.Summary
}

func (e EventWrapper) Title() string {
	return e.Event.Summary
}

func (e EventWrapper) Description() string {
	return e.Date()
}

func initialModel() model {
	srv := authorize()
	t := time.Now().Format(time.RFC3339)
	events, err := srv.Events.List("primary").ShowDeleted(false).
		SingleEvents(true).TimeMin(t).MaxResults(10).OrderBy("startTime").Do()
	if err != nil {
		log.Fatalf("Unable to retrieve next ten of the user's events: %v", err)
	}

	keys := newKeyMap()
	m := model{list: list.New([]list.Item{}, newItemDelegate(keys), 0, 0), keys: keys}
	m.list.Title = "Events"

	for _, event := range events.Items {
		m.list.InsertItem(len(events.Items), EventWrapper{Event: event})
	}

	return m
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			m.selected.Event = nil
			return m, nil

		case "ctrl+c", "q":
			return m, tea.Quit
		}

		switch {
		case key.Matches(msg, m.keys.chooseItem):
			event := m.list.SelectedItem().(EventWrapper)
			m.selected = event
			return m, m.list.NewStatusMessage("You chose " + event.Summary)
		}

	case tea.WindowSizeMsg:
		xMargin, yMargin := docStyle.GetFrameSize()

		m.contentWidth = msg.Width-xMargin
		m.list.SetSize(msg.Width-xMargin, msg.Height-yMargin)
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m model) View() string {
	if m.selected.Event != nil {
		return docStyle.Render(detailedInfoView(m))
	} else {
		return docStyle.Render(m.list.View())
	}
}

func detailedInfoView(m model) string {
	eventWrapper := m.selected

	msg := eventWrapper.Summary + "\n" + eventWrapper.Date() + "\n"

	if eventWrapper.Location != "" {
		msg += "Location: " + eventWrapper.Location + "\n"
	}

	msg += "\n\n" + eventWrapper.Event.Description

	return wordwrap.String(msg, min(m.contentWidth, MAX_STR_LENGTH))
}

func main() {
	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v", err)
		os.Exit(1)
	}
}

// Combines the main app key bindings to show help on list view
func newItemDelegate(keys *mainKeyMap) list.DefaultDelegate {
	d := list.NewDefaultDelegate()
	
	help := []key.Binding{keys.chooseItem}
	
	d.ShortHelpFunc = func() []key.Binding {
		return help
	}

	d.FullHelpFunc = func() [][]key.Binding {
		return [][]key.Binding{help}
	}

	return d
}

// satiesfies help.KeyMap interface
func (m mainKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{
		m.chooseItem,
	}
}

// satiesfies help.KeyMap interface
// each row in the first array corresponds with the columns showed in help
func (m mainKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{
			m.chooseItem,
		},
	}
}