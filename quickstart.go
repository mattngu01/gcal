package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/option"
)

/* Google Calendar API */

// Retrieve a token, saves the token, then returns the generated client.
func getClient(config *oauth2.Config) *http.Client {
	// The file token.json stores the user's access and refresh tokens, and is
	// created automatically when the authorization flow completes for the first
	// time.
	tokFile := "token.json"
	tok, err := tokenFromFile(tokFile)
	if err != nil {
		tok = getTokenFromWeb(config)
		saveToken(tokFile, tok)
	}
	return config.Client(context.Background(), tok)
}

// Request a token from the web, then returns the retrieved token.
func getTokenFromWeb(config *oauth2.Config) *oauth2.Token {
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Printf("Go to the following link in your browser then type the "+
		"authorization code: \n%v\n", authURL)

	var authCode string
	if _, err := fmt.Scan(&authCode); err != nil {
		log.Fatalf("Unable to read authorization code: %v", err)
	}

	tok, err := config.Exchange(context.TODO(), authCode)
	if err != nil {
		log.Fatalf("Unable to retrieve token from web: %v", err)
	}
	return tok
}

// Retrieves a token from a local file.
func tokenFromFile(file string) (*oauth2.Token, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	tok := &oauth2.Token{}
	err = json.NewDecoder(f).Decode(tok)
	return tok, err
}

// Saves a token to a file path.
func saveToken(path string, token *oauth2.Token) {
	fmt.Printf("Saving credential file to: %s\n", path)
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		log.Fatalf("Unable to cache oauth token: %v", err)
	}
	defer f.Close()
	json.NewEncoder(f).Encode(token)
}

func authorize() *calendar.Service {
	ctx := context.Background()
	b, err := os.ReadFile("credentials.json")
	if err != nil {
		log.Fatalf("Unable to read client secret file: %v", err)
	}

	// If modifying these scopes, delete your previously saved token.json.
	config, err := google.ConfigFromJSON(b, calendar.CalendarReadonlyScope)
	if err != nil {
		log.Fatalf("Unable to parse client secret file to config: %v", err)
	}
	client := getClient(config)

	srv, err := calendar.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		log.Fatalf("Unable to retrieve Calendar client: %v", err)
	}

	return srv
}

/* BubbleTea Things */

var docStyle = lipgloss.NewStyle().Margin(1, 2)

type model struct {
	list list.Model
	// not sure if better to have it as list.Item instead
	selected EventWrapper
	keys     *mainKeyMap
}

type mainKeyMap struct {
	chooseItem key.Binding
}

func newKeyMap() *mainKeyMap {
	return &mainKeyMap{
		chooseItem: key.NewBinding(
			key.WithKeys("enter"),
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

	m := model{list: list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0), keys: newKeyMap()}
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
		h, v := docStyle.GetFrameSize()
		m.list.SetSize(msg.Width-h, msg.Height-v)
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

	return msg
}

func main() {
	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v", err)
		os.Exit(1)
	}
}
