package main

import (
	"fmt"
	"os"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
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
	eventForm *huh.Form
	mode int
}

type eventFields struct {
	summary string
	description string
	location string
	start string // to be converted to time.Time per go-anytime
	end string
}

var DATE_HELP string = "Accepts standard YYYY-MM-DD & other formats, or try a phrase: 'two days from now at 2pm'"

func newEventForm() *huh.Form {
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().Title("Summary / Title"),
			huh.NewText().Title("Description"),
			huh.NewInput().Title("Location"),
			huh.NewInput().Title("Start date/time").Placeholder(DATE_HELP),
			huh.NewInput().Title("End date/time").Placeholder(DATE_HELP),
		),
	)

	return form
}

type mainKeyMap struct {
	chooseItem key.Binding
	newEvent key.Binding
}

func newKeyMap() *mainKeyMap {
	return &mainKeyMap{
		chooseItem: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("ENTER", "Details"),
		),
		newEvent: key.NewBinding(
			key.WithKeys("N", "n"),
			key.WithHelp("N/n", "Create new event"),
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

func emptyModel() model {
	keys := newKeyMap()
	list := list.New([]list.Item{}, newItemDelegate(keys), 0, 0)
	list.Title = "Google Calendar"
	return model{list: list, keys: keys, mode: LIST}
}

// viewing modes
const (
	LIST = iota
	NEW_EVENT
)

func (m *model) updateModel(events *calendar.Events) {
	for _, event := range events.Items {
		m.list.InsertItem(len(events.Items), EventWrapper{Event: event})
	}
}

func (m model) Init() tea.Cmd {
	return getEvents
}

func formUpdate(m model, msg tea.Msg) (tea.Model, tea.Cmd) {
	form, cmd := m.eventForm.Update(msg)

	if f, ok := form.(*huh.Form); ok {
        m.eventForm = f
    }

	if m.eventForm.State == huh.StateCompleted || m.eventForm.State == huh.StateAborted {
		m.mode = LIST
	}

    return m, cmd
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.mode == LIST {
		switch msg := msg.(type) {
		case getEventsMsg:
			m.updateModel(msg)

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
			case key.Matches(msg, m.keys.newEvent):
				m.mode = NEW_EVENT
				m.eventForm = newEventForm()
				return m, m.eventForm.Init()
			}

		case tea.WindowSizeMsg:
			xMargin, yMargin := docStyle.GetFrameSize()

			m.contentWidth = msg.Width-xMargin
			m.list.SetSize(msg.Width-xMargin, msg.Height-yMargin)
		}

		var cmd tea.Cmd
		m.list, cmd = m.list.Update(msg)

		return m, cmd
	} else if m.mode == NEW_EVENT {
		// upon finishing event need to create command to create new event in API, change mode back to LIST, maybe update..?
		return formUpdate(m, msg)
	}

	return m, nil
}

func (m model) View() string {
	if len(m.list.Items()) == 0 {
		return docStyle.Render("Obtaining user events...")
	} else if m.selected.Event != nil {
		return docStyle.Render(detailedInfoView(m))
	} else if m.mode == NEW_EVENT {
		return docStyle.Render(m.eventForm.View())
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
	p := tea.NewProgram(emptyModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v", err)
		os.Exit(1)
	}
}

// Combines the main app key bindings to show help on list view
func newItemDelegate(keys *mainKeyMap) list.DefaultDelegate {
	d := list.NewDefaultDelegate()
	
	help := []key.Binding{keys.chooseItem, keys.newEvent}
	
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