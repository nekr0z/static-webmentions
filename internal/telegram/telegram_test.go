// Copyright (C) 2026 Evgeny Kuznetsov (evgeny@kuznetsov.md)
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this program. If not, see <https://www.gnu.org/licenses/>.

package telegram

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"evgenykuznetsov.org/go/static-webmentions/internal/hentry"
)

func TestSendTelegramMessage(t *testing.T) {
	// Create a test server to mock the Telegram API
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify the request is correct
		if r.Method != "POST" {
			t.Errorf("Expected POST request, got %s", r.Method)
		}

		// Read the request body
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("Failed to read request body: %v", err)
		}

		// Check that the request contains the expected data
		// Note: JSON field order may vary, so we check for the presence of expected fields
		bodyStr := string(body)
		expectedFields := []string{
			`"chat_id":"123456789"`,
			`"text":"Test Post\nhttps://example.com/test"`,
			`"parse_mode":"HTML"`,
		}

		for _, field := range expectedFields {
			if !strings.Contains(bodyStr, field) {
				t.Errorf("Expected field %s not found in body %s", field, bodyStr)
			}
		}

		// Return a successful response
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte(`{"ok":true,"result":{"message_id":1}}`)); err != nil {
			t.Errorf("Failed to write response: %v", err)
		}
	}))
	defer ts.Close()

	// Test sending a message
	config := Config{
		BotToken: "test_token",
		ChatID:   "123456789",
	}

	// Replace the Telegram API URL with our test server
	telegramAPIURL = ts.URL + "/bot%s/sendMessage"

	entry := hentry.HEntry{
		Name: "Test Post",
		URL:  "https://example.com/test",
	}

	err := SendTelegramMessage(config, entry)
	if err != nil {
		t.Errorf("SendTelegramMessage failed: %v", err)
	}
}

func TestSendTelegramMessageError(t *testing.T) {
	// Create a test server that returns an error
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		if _, err := w.Write([]byte(`{"ok":false,"error_code":400,"description":"Bad Request: chat not found"}`)); err != nil {
			t.Errorf("Failed to write response: %v", err)
		}
	}))
	defer ts.Close()

	// Test sending a message that fails
	config := Config{
		BotToken: "test_token",
		ChatID:   "invalid_chat_id",
	}

	// Replace the Telegram API URL with our test server
	telegramAPIURL = ts.URL + "/bot%s/sendMessage"

	entry := hentry.HEntry{
		Name: "Test Post",
		URL:  "https://example.com/test",
	}

	err := SendTelegramMessage(config, entry)
	if err == nil {
		t.Error("Expected error, got nil")
	}
}

func TestTruncateMessage(t *testing.T) {
	// Test with a very long title that exceeds the limit
	longTitle := strings.Repeat("a", 5000)
	entry := hentry.HEntry{
		Name: longTitle,
		URL:  "https://example.com/short-url",
	}

	// Create a test server to capture the message
	var receivedMessage string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		receivedMessage = string(body)
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte(`{"ok":true,"result":{"message_id":1}}`)); err != nil {
			t.Errorf("Failed to write response: %v", err)
		}
	}))
	defer ts.Close()

	config := Config{
		BotToken: "test_token",
		ChatID:   "123456789",
	}

	// Replace the Telegram API URL with our test server
	telegramAPIURL = ts.URL + "/bot%s/sendMessage"

	err := SendTelegramMessage(config, entry)
	if err != nil {
		t.Errorf("SendTelegramMessage failed: %v", err)
	}

	// Parse the received message to check its length
	var payload map[string]interface{}
	if err := json.Unmarshal([]byte(receivedMessage), &payload); err != nil {
		t.Fatalf("Failed to parse received message: %v", err)
	}

	text, ok := payload["text"].(string)
	if !ok {
		t.Fatalf("Text field not found or not a string")
	}

	// The message should not exceed 4096 characters
	if len([]rune(text)) > 4096 {
		t.Errorf("Message exceeds 4096 characters: %d", len(text))
	}

	// Check that the URL is intact
	if !strings.Contains(text, "https://example.com/short-url") {
		t.Errorf("URL was truncated or missing")
	}

	// Check that the title was truncated (should contain ellipsis)
	if !strings.Contains(text, "…") {
		t.Errorf("Title was not truncated properly")
	}
}

func TestSendTelegramMessageRetry(t *testing.T) {
	attempts := 0
	maxAttempts := 3

	// Create a test server that returns 429 errors for the first few attempts
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts <= maxAttempts {
			// Return 429 with retry_after for the first few attempts
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusTooManyRequests)
			if _, err := w.Write([]byte(`{"ok":false,"error_code":429,"description":"Too Many Requests: retry after 1","retry_after":1}`)); err != nil {
				t.Errorf("Failed to write response: %v", err)
			}
		} else {
			// Return success on the final attempt
			w.WriteHeader(http.StatusOK)
			if _, err := w.Write([]byte(`{"ok":true,"result":{"message_id":1}}`)); err != nil {
				t.Errorf("Failed to write response: %v", err)
			}
		}
	}))
	defer ts.Close()

	// Test sending a message with retries
	config := Config{
		BotToken: "test_token",
		ChatID:   "123456789",
	}

	// Replace the Telegram API URL with our test server
	telegramAPIURL = ts.URL + "/bot%s/sendMessage"

	entry := hentry.HEntry{
		Name: "Test Post",
		URL:  "https://example.com/test",
	}

	err := SendTelegramMessage(config, entry)
	if err != nil {
		t.Errorf("SendTelegramMessage failed after retries: %v", err)
	}

	// Check that we made the expected number of attempts
	if attempts != maxAttempts+1 {
		t.Errorf("Expected %d attempts, got %d", maxAttempts+1, attempts)
	}
}

func TestSendTelegramMessageRetryFailure(t *testing.T) {
	attempts := 0
	maxAttempts := 3

	// Create a test server that always returns 429 errors
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		// Always return 429
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusTooManyRequests)
		if _, err := w.Write([]byte(`{"ok":false,"error_code":429,"description":"Too Many Requests: retry after 1","retry_after":1}`)); err != nil {
			t.Errorf("Failed to write response: %v", err)
		}
	}))
	defer ts.Close()

	// Test sending a message with retries that ultimately fails
	config := Config{
		BotToken: "test_token",
		ChatID:   "123456789",
	}

	// Replace the Telegram API URL with our test server
	telegramAPIURL = ts.URL + "/bot%s/sendMessage"

	entry := hentry.HEntry{
		Name: "Test Post",
		URL:  "https://example.com/test",
	}

	err := SendTelegramMessage(config, entry)
	if err == nil {
		t.Error("Expected error after max retries, got nil")
	}

	// Check that we made the expected number of attempts
	if attempts != maxAttempts+1 {
		t.Errorf("Expected %d attempts, got %d", maxAttempts+1, attempts)
	}
}
