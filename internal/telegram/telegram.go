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

// Package telegram provides functionality for sending messages to Telegram channels
package telegram

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"evgenykuznetsov.org/go/static-webmentions/internal/hentry"
)

// Config holds the Telegram configuration
type Config struct {
	BotToken string   `toml:"botToken"`
	ChatID   string   `toml:"chatID"`
	Feeds    []string `toml:"feeds"`
}

const (
	messageSizeLimit = 4096
)

// telegramAPIURL is the base URL for the Telegram API.
var telegramAPIURL = "https://api.telegram.org/bot%s/sendMessage"

// SendTelegramMessage sends a message to a Telegram chat.
func SendTelegramMessage(config Config, entry hentry.HEntry) error {
	return sendTelegramMessageWithRetry(config, entry, 3, 1*time.Second)
}

// sendTelegramMessageWithRetry sends a message to a Telegram chat with retry logic.
func sendTelegramMessageWithRetry(config Config, entry hentry.HEntry, maxRetries int, initialDelay time.Duration) error {
	if entry.URL == "" {
		return fmt.Errorf("entry URL is empty")
	}

	message, err := formatMessage(entry)
	if err != nil {
		return err
	}

	// Create the request payload
	payload := map[string]string{
		"chat_id":                  config.ChatID,
		"text":                     message,
		"disable_web_page_preview": "false",
	}

	// Convert payload to JSON
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	// Create the request
	url := fmt.Sprintf(telegramAPIURL, config.BotToken)

	// Try to send the message with retries
	delay := initialDelay
	for attempt := 0; attempt <= maxRetries; attempt++ {
		req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonPayload))
		if err != nil {
			return fmt.Errorf("failed to create request: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")

		// Send the request
		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			if attempt < maxRetries {
				time.Sleep(delay)
				delay *= 2 // Exponential backoff
				continue
			}
			return fmt.Errorf("failed to send message: %w", err)
		}

		// Read the response body
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			if closeErr := resp.Body.Close(); closeErr != nil {
				// Log the error but continue with the main error
				fmt.Printf("Failed to close response body: %v\n", closeErr)
			}
			if attempt < maxRetries {
				time.Sleep(delay)
				delay *= 2 // Exponential backoff
				continue
			}
			return fmt.Errorf("failed to read response body: %w", err)
		}
		if closeErr := resp.Body.Close(); closeErr != nil {
			// Log the error but continue with the main error
			fmt.Printf("Failed to close response body: %v\n", closeErr)
		}

		// Handle retriable errors
		if isRetriableError(resp.StatusCode) {
			// Parse the response to check for retry_after
			var result map[string]any
			if err := json.Unmarshal(body, &result); err == nil {
				if retryAfter, ok := result["retry_after"].(float64); ok {
					// Wait for the specified retry time
					time.Sleep(time.Duration(retryAfter) * time.Second)
					continue
				}
			}

			// If no retry_after specified, use exponential backoff
			if attempt < maxRetries {
				time.Sleep(delay)
				delay *= 2 // Exponential backoff
				continue
			}
		}

		// For non-retriable errors or when max retries reached
		if resp.StatusCode != http.StatusOK {
			// If we've reached max retries, return the error
			if attempt >= maxRetries {
				return fmt.Errorf("telegram API returned status %d: %s", resp.StatusCode, string(body))
			}
			// For other errors that aren't retriable, return immediately
			if !isRetriableError(resp.StatusCode) {
				return fmt.Errorf("telegram API returned status %d: %s", resp.StatusCode, string(body))
			}
			// For retriable errors, continue with exponential backoff
			time.Sleep(delay)
			delay *= 2 // Exponential backoff
			continue
		}

		// Parse the response
		var result map[string]interface{}
		if err := json.Unmarshal(body, &result); err != nil {
			if attempt < maxRetries {
				time.Sleep(delay)
				delay *= 2 // Exponential backoff
				continue
			}
			return fmt.Errorf("failed to decode response: %w", err)
		}

		if !result["ok"].(bool) {
			if attempt >= maxRetries {
				return fmt.Errorf("telegram API returned error: %v", result["description"])
			}

			if isRetriableError(resp.StatusCode) {
				time.Sleep(delay)
				delay *= 2 // Exponential backoff
				continue
			}

			return fmt.Errorf("telegram API returned error: %v", result["description"])
		}

		return nil
	}

	return fmt.Errorf("failed to send message after %d attempts", maxRetries+1)
}

func isRetriableError(statusCode int) bool {
	if statusCode == http.StatusTooManyRequests || statusCode >= 500 {
		return true
	}
	return false
}

func formatMessage(entry hentry.HEntry) (string, error) {
	urlLine := entry.URL
	urlLen := len([]rune(urlLine))

	if urlLen > messageSizeLimit {
		return "", fmt.Errorf("URL is too long: %d characters", len(urlLine))
	}

	if messageSizeLimit < urlLen+5 {
		return urlLine, nil
	}

	titleLine := entry.Name
	titleRunes := []rune(titleLine)
	titleLen := len(titleRunes)

	if messageSizeLimit > urlLen+titleLen+1 {
		return fmt.Sprintf("%s\n%s", titleLine, urlLine), nil
	}

	titleRunes = titleRunes[:messageSizeLimit-urlLen-2]
	titleLine = string(titleRunes) + "…"

	return fmt.Sprintf("%s\n%s", titleLine, urlLine), nil
}
