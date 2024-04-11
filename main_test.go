package main

import (
	"testing"

	"errors"

	"github.com/google/go-cmp/cmp"
	"google.golang.org/api/calendar/v3"
)

func generateSampleModel() model {
	m := emptyModel()
	event := sampleEvent()
	m.list.InsertItem(0, EventWrapper{&event})

	return m
}

func sampleEvent() calendar.Event {
	return calendar.Event{
		Summary: "Sample Event", 
		Description: "Sample", 
		Start: &calendar.EventDateTime{DateTime: "2024-04-03T00:00:00-07:00"}, 
		End: &calendar.EventDateTime{DateTime: "2024-04-04T00:00:00-07:00"},
	}
}

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
	sampleEvent := sampleEvent()
	eventsList := &calendar.Events{
		Items: []*calendar.Event{&sampleEvent},
	}

	newModel, _ := m.Update(getEventsMsg(eventsList))
	wrapper := EventWrapper{Event: &sampleEvent}

	if newModel.(model).list.Items()[0].(EventWrapper) != wrapper {
		t.Errorf("%v != %v", newModel.(model).list.Items()[0].(EventWrapper), wrapper)	
	}
}

func TestFilledForm(t *testing.T) {
	event := sampleEvent()
	f := filledEventForm(EventWrapper{&event})

	// huh forms do not set the field value until processing nextFieldMsg 
	for i := 0; i < 50; i++ {
		f.Update(f.NextField())
	}

	if f.GetString("summary") != event.Summary {
		t.Errorf("Summary: %v != %v", f.Get("summary"), event.Summary)
	}

	if f.GetString("description") != event.Description {
		t.Errorf("Description: %v != %v", f.Get("description"), event.Summary)
	}

	if f.GetString("start") != event.Start.DateTime {
		t.Errorf("Start: %v != %v", f.Get("start"), event.Start.DateTime)
	}

	if f.GetString("end") != event.End.DateTime {
		t.Errorf("End: %v != %v", f.Get("end"), event.End.DateTime)
	}

	if f.GetString("location") != event.Location {
		t.Errorf("Location: %v != %v", f.GetString("location"), event.Location)
	}
}


func TestConvertFormToEvent(t *testing.T) {
	event := sampleEvent()
	f := filledEventForm(EventWrapper{&event})

	// huh forms do not set the field value until processing nextFieldMsg
	for i := 0; i < 50; i++ {
		f.Update(f.NextField())
	}

	convertedEvent, err := formToEvent(f)

	if err != nil {
		t.Errorf("error %v occurred converting form to event", err)
	}

	if !cmp.Equal(*convertedEvent, event) {
		t.Errorf("Converted event is not equal, converted\n %+v \n!=\n %+v", *convertedEvent, event)
	}
}

func TestDefaultEndTimeOneHourAfterStart(t *testing.T) {
	event := sampleEvent()
	event.End.DateTime = ""

	f := filledEventForm(EventWrapper{&event})

	// huh forms do not set the field value until processing nextFieldMsg
	for i := 0; i < 50; i++ {
		f.Update(f.NextField())
	}

	convertedEvent, err := formToEvent(f)

	if err != nil {
		t.Errorf("error %v occurred converting form to event", err)
	}

	expectedEnd := "2024-04-03T01:00:00-07:00"
	if convertedEvent.End.DateTime != expectedEnd {
		t.Errorf("Expected %v != Actual %v", expectedEnd, convertedEvent.End.DateTime)
	}
}
