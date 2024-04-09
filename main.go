package main

import (
	"fmt"
	"os"
	"time"

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
	err error
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
			huh.NewInput().Title("Summary / Title").Key("summary"),
			huh.NewText().Title("Description").Key("description"),
			huh.NewInput().Title("Location").Key("location"),
			huh.NewInput().Title("Start date/time").Key("start").Description(DATE_HELP).Validate(func(str string) error {
				_, err := convertStrToDateTime(str)
				return err
			}),
			huh.NewInput().Title("End date/time").Key("end").Description(DATE_HELP).Validate(func(str string) error {
				_, err := convertStrToDateTime(str)
				return err
			}),
		),
	)

	return form
}

func filledEventForm(e EventWrapper) *huh.Form {
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().Title("Summary / Title").Key("summary").Value(&e.Summary),
			huh.NewText().Title("Description").Key("description").Value(&e.Event.Description),
			huh.NewInput().Title("Location").Key("location").Value(&e.Location),
			huh.NewInput().Title("Start date/time").Key("start").Description(DATE_HELP).Value(&e.Start.DateTime).Validate(func(str string) error {
				_, err := convertStrToDateTime(str)
				return err
			}),
			huh.NewInput().Title("End date/time").Key("end").Description(DATE_HELP).Value(&e.Start.DateTime).Validate(func(str string) error {
				_, err := convertStrToDateTime(str)
				return err
			}),
		),
	)

	return form
}

type mainKeyMap struct {
	chooseItem key.Binding
	newEvent key.Binding
	refreshEvents key.Binding
	deleteItem key.Binding
	editItem key.Binding
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
		refreshEvents: key.NewBinding(
			key.WithKeys("R", "r"),
			key.WithHelp("R/r", "Refresh events list"),
		),
		deleteItem: key.NewBinding(
			key.WithKeys("D"),
			key.WithHelp("D", "Delete event"),
		),
		editItem: key.NewBinding(
			key.WithKeys("E", "e"),
			key.WithHelp("E/e", "Edit event"),
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
	EDIT_EVENT
	SELECT_EVENT
)

func (m *model) updateModelEvents(events *calendar.Events) {
	// on refresh, want to make sure we start on a clean slate
	m.list.SetItems([]list.Item{})
	for _, event := range events.Items {
		m.list.InsertItem(len(events.Items), EventWrapper{Event: event})
	}
}

func (m model) Init() tea.Cmd {
	return getEvents
}

// assumes that form has required fields
// for calendar event
func formToEvent(f *huh.Form) (*calendar.Event, error) {
	event, err := updateEventWithFormFields(f, &calendar.Event{})
	return event, err
}

func updateEventWithFormFields(f *huh.Form, event *calendar.Event) (*calendar.Event, error) {
	start, err := convertStrToDateTime(f.GetString("start"))

	if err != nil {
		return &calendar.Event{}, nil
	}

	end, err := convertStrToDateTime(f.GetString("end"))

	if err != nil {
		return &calendar.Event{}, nil
	}

	event.Summary = f.GetString("summary")
	event.Location = f.GetString("location")
	event.Description = f.GetString("description")
	event.Start = &calendar.EventDateTime{DateTime: start.Format(time.RFC3339)}
	event.End = &calendar.EventDateTime{DateTime: end.Format(time.RFC3339)}


	return event, nil 
}

func formUpdate(m model, msg tea.Msg) (tea.Model, tea.Cmd) {
	form, cmd := m.eventForm.Update(msg)

	if f, ok := form.(*huh.Form); ok {
        m.eventForm = f
    }

	if m.eventForm.State == huh.StateCompleted {
		var cmd tea.Cmd
		if m.mode == NEW_EVENT {
			cmd = createEvent(eventFields{
				summary: m.eventForm.GetString("summary"),
				description: m.eventForm.GetString("description"),
				location: m.eventForm.GetString("location"),
				start: m.eventForm.GetString("start"),
				end: m.eventForm.GetString("end"),
			})
		}

		if m.mode == EDIT_EVENT {
			event, _ := updateEventWithFormFields(m.eventForm, m.selected.Event)
			cmd = updateEvent(*event)
		}

		m.mode = LIST
		return m, cmd
		}

	if m.eventForm.State == huh.StateAborted {
		m.mode = LIST
	}

    return m, cmd
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.mode == LIST {
		switch msg := msg.(type) {
		case getEventsMsg:
			m.updateModelEvents(msg)
			return m, m.list.NewStatusMessage("Updated event list")

		case errMsg:
			m.err = msg
			return m, tea.Quit

		case tea.KeyMsg:
			// do not match any keys below if we're filtering
			if m.list.FilterState() == list.Filtering {
				break
			}

			switch msg.String() {
			case "esc":
				m.selected.Event = nil
				return m, nil

			case "ctrl+c", "q":
				return m, tea.Quit
			}

			switch {
			case key.Matches(msg, m.keys.chooseItem):
				m.mode = SELECT_EVENT
				event := m.list.SelectedItem().(EventWrapper)
				m.selected = event
				return m, m.list.NewStatusMessage("You chose " + event.Summary)
			case key.Matches(msg, m.keys.newEvent):
				m.mode = NEW_EVENT
				m.eventForm = newEventForm()
				return m, m.eventForm.Init()
			case key.Matches(msg, m.keys.refreshEvents):
				return m, getEvents
			case key.Matches(msg, m.keys.editItem):
				m.mode = EDIT_EVENT
				event := m.list.SelectedItem().(EventWrapper)
				m.selected = event
				m.eventForm = filledEventForm(event)
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
	} else if m.mode == NEW_EVENT || m.mode == EDIT_EVENT {
		return formUpdate(m, msg)
	}

	return m, nil
}

func (m model) View() string {
	if m.err != nil {
		return m.errView()
	} else if len(m.list.Items()) == 0 {
		return docStyle.Render("Obtaining user events...")
	} else if m.mode == SELECT_EVENT && m.selected.Event != nil {
		return docStyle.Render(m.detailedInfoView())
	} else if m.mode == NEW_EVENT || m.mode == EDIT_EVENT {
		return docStyle.Render(m.eventForm.View())
	} else {
		return docStyle.Render(m.list.View())
	}
}

func (m model) errView() string {
	return m.err.Error()
}

func (m model) detailedInfoView() string {
	eventWrapper := m.selected

	msg := eventWrapper.Summary + "\n" + eventWrapper.Date() + "\n"

	if eventWrapper.Location != "" {
		msg += "Location: " + eventWrapper.Location + "\n"
	}

	msg += "\n\n" + eventWrapper.Event.Description

	return wordwrap.String(msg, min(m.contentWidth, MAX_STR_LENGTH))
}

func main() {
	// hack since bubbletea eats the input for token
	authorize()
	p := tea.NewProgram(emptyModel())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v", err)
		os.Exit(1)
	}
}

// Combines the main app key bindings to show help on list view
func newItemDelegate(keys *mainKeyMap) list.DefaultDelegate {
	d := list.NewDefaultDelegate()

	d.UpdateFunc = func(msg tea.Msg, m *list.Model) tea.Cmd {
		var event EventWrapper
		if e, ok := m.SelectedItem().(EventWrapper); ok {
			event = e
		} else {
			return nil
		}

		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch {
			case key.Matches(msg, keys.deleteItem):
				return tea.Batch(m.NewStatusMessage("Deleted event " + event.Summary), deleteEvent(event))
			}
			
		}

		return nil
	}
	
	d.ShortHelpFunc = keys.ShortHelp

	d.FullHelpFunc = keys.FullHelp

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
			m.refreshEvents,
			m.deleteItem,
			m.editItem,
		},
	}
}