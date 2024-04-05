package main

import (
	"testing"

	"errors"

	"google.golang.org/api/calendar/v3"
)

func generateSampleModel() model {
	m := emptyModel()
	event := &calendar.Event{
		Summary: "Sample Event", 
		Description: "Sample", 
		Start: &calendar.EventDateTime{DateTime: "2024-04-03T00:00:00-07:00"}, 
		End: &calendar.EventDateTime{DateTime: "2024-04-04T00:00:00-07:00"},
	}
	m.list.InsertItem(0, EventWrapper{event})

	return m
}

/*
let's test the following:
- new event pipeline
- selecting a event should show its details
- correctly showing errorView
- getting events
*/

func TestShowEventDetails(t *testing.T) {
	m := generateSampleModel()
	m.selected = m.list.Items()[0].(EventWrapper)

	// .View() seems to add some extra padding that I couldn't capture
	// in expected var
	detailedInfo := m.detailedInfoView()

	expectedResult := "Sample Event\n2024-04-03T00:00:00-07:00 - 2024-04-04T00:00:00-07:00\n\n\nSample" 

	if detailedInfo != expectedResult {
		t.Errorf("Actual:\n%s\nExpected:\n%s", detailedInfo, expectedResult)
	}
}

func TestShowError(t *testing.T) {
	m := generateSampleModel()
	expected := "Sample error"
	m.err = errors.New(expected)

	errOutput := m.View()
	if errOutput != expected {
		t.Errorf("%s != %s", m.errView(), "Sample error")
	}
}

func TestUpdateGetEvents(t *testing.T) {
	m := generateSampleModel()
	sampleEvent := calendar.Event{
		Summary: "Different Sample Event", 
		Description: "Sample", 
		Start: &calendar.EventDateTime{DateTime: "2024-04-03T00:00:00-07:00"}, 
		End: &calendar.EventDateTime{DateTime: "2024-04-04T00:00:00-07:00"},
	}
	eventsList := &calendar.Events{
		Items: []*calendar.Event{&sampleEvent},
	}

	newModel, _ := m.Update(getEventsMsg(eventsList))
	wrapper := EventWrapper{Event: &sampleEvent}

	if newModel.(model).list.Items()[0].(EventWrapper) != wrapper {
		t.Errorf("%v != %v", newModel.(model).list.Items()[0].(EventWrapper), wrapper)	
	}
}