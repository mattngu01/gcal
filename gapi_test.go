package main

import (
	"encoding/json"
	"os"
	"testing"
	"time"

	"golang.org/x/oauth2"
)

// this is a test to learn more about go testing,
// I think the time it would take to test the auth flow
// would be better spent testing the actual behavior of the app
func TestTokenFromFile(t *testing.T) {
	// Create a temporary file to hold the token JSON
	tmpfile, err := os.CreateTemp("", "example")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name()) // Clean up

	// Create a sample token to write to the temporary file
	token := &oauth2.Token{
		AccessToken:  "access_token",
		TokenType:    "Bearer",
		RefreshToken: "refresh_token",
		Expiry:       time.Now(),
	}

	// Marshal the token to JSON and write it to the temporary file
	data, err := json.Marshal(token)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := tmpfile.Write(data); err != nil {
		t.Fatal(err)
	}
	if err := tmpfile.Close(); err != nil {
		t.Fatal(err)
	}

	// Call the function under test
	got, err := tokenFromFile(tmpfile.Name())
	if err != nil {
		t.Fatalf("tokenFromFile(%q) returned error: %v", tmpfile.Name(), err)
	}

	// Check if the token retrieved matches the original token
	if got.AccessToken != token.AccessToken || got.RefreshToken != token.RefreshToken {
		t.Errorf("tokenFromFile(%q) = %v; want %v", tmpfile.Name(), got, token)
	}	
}

// ChatGPT seems to suggest mocking out all the responses from
// all the dependent funcs, such as NewService, getClient,
// but I don't think it's worth the time atm to write the test
func TestAuthorize(t *testing.T) {

}