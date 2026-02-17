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

package hentry

import (
	"bytes"
	"testing"
)

func TestExtractHEntries(t *testing.T) {
	// Test HTML content with h-entry elements
	htmlContent := `
		<html>
			<body>
				<article class="h-entry">
					<h1 class="p-name">First Post</h1>
					<a class="u-url" href="https://example.com/first">Link</a>
				</article>
				<article class="h-entry">
					<h1 class="p-name">Second Post</h1>
					<a class="u-url" href="https://example.com/second">Link</a>
				</article>
			</body>
		</html>
	`

	entries, err := ExtractHEntries(bytes.NewBufferString(htmlContent))
	if err != nil {
		t.Fatalf("ExtractHEntries failed: %v", err)
	}

	if len(entries) != 2 {
		t.Errorf("Expected 2 entries, got %d", len(entries))
	}

	if entries[0].Name != "First Post" {
		t.Errorf("Expected 'First Post', got '%s'", entries[0].Name)
	}

	if entries[0].URL != "https://example.com/first" {
		t.Errorf("Expected 'https://example.com/first', got '%s'", entries[0].URL)
	}

	if entries[1].Name != "Second Post" {
		t.Errorf("Expected 'Second Post', got '%s'", entries[1].Name)
	}

	if entries[1].URL != "https://example.com/second" {
		t.Errorf("Expected 'https://example.com/second', got '%s'", entries[1].URL)
	}
}

func TestFindNewHEntries(t *testing.T) {
	oldEntries := []HEntry{
		{Name: "Old Post", URL: "https://example.com/old"},
	}

	newEntries := []HEntry{
		{Name: "Old Post", URL: "https://example.com/old"},
		{Name: "New Post", URL: "https://example.com/new"},
	}

	newOnly := FindNewHEntries(oldEntries, newEntries)

	if len(newOnly) != 1 {
		t.Errorf("Expected 1 new entry, got %d", len(newOnly))
	}

	if newOnly[0].Name != "New Post" {
		t.Errorf("Expected 'New Post', got '%s'", newOnly[0].Name)
	}

	if newOnly[0].URL != "https://example.com/new" {
		t.Errorf("Expected 'https://example.com/new', got '%s'", newOnly[0].URL)
	}
}

func TestResolveURL(t *testing.T) {
	tests := []struct {
		name        string
		url         string
		pageURL     string
		expected    string
		expectError bool
	}{
		{
			name:        "absolute URL",
			url:         "https://example.com/path",
			pageURL:     "https://example.com/page",
			expected:    "https://example.com/path",
			expectError: false,
		},
		{
			name:        "root-relative URL",
			url:         "/path",
			pageURL:     "https://example.com/page",
			expected:    "https://example.com/path",
			expectError: false,
		},
		{
			name:        "page-relative URL",
			url:         "path",
			pageURL:     "https://example.com/dir/page",
			expected:    "https://example.com/dir/path",
			expectError: false,
		},
		{
			name:        "page-relative URL with parent directory",
			url:         "../path",
			pageURL:     "https://example.com/dir/page",
			expected:    "https://example.com/path",
			expectError: false,
		},
		{
			name:        "root-relative URL with missing page URL",
			url:         "/path",
			pageURL:     "",
			expected:    "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := resolveURL(tt.url, tt.pageURL)
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestExtractHEntriesWithRelativeURLs(t *testing.T) {
	htmlContent := `
		<html>
			<head>
				<link rel="canonical" href="https://example.com/page/">
			</head>
			<body>
				<article class="h-entry">
					<h1 class="p-name">First Post</h1>
					<a class="u-url" href="/first">Link</a>
				</article>
				<article class="h-entry">
					<h1 class="p-name">Second Post</h1>
					<a class="u-url" href="second">Link</a>
				</article>
			</body>
		</html>
	`

	entries, err := ExtractHEntries(bytes.NewBufferString(htmlContent))
	if err != nil {
		t.Fatalf("ExtractHEntries failed: %v", err)
	}

	if len(entries) != 2 {
		t.Errorf("Expected 2 entries, got %d", len(entries))
	}

	if entries[0].URL != "https://example.com/first" {
		t.Errorf("Expected 'https://example.com/first', got '%s'", entries[0].URL)
	}

	if entries[1].URL != "https://example.com/page/second" {
		t.Errorf("Expected 'https://example.com/page/second', got '%s'", entries[1].URL)
	}
}

func TestExtractHEntriesWithoutCanonical(t *testing.T) {
	htmlContent := `
		<html>
			<body>
				<article class="h-entry">
					<h1 class="p-name">First Post</h1>
					<a class="u-url" href="/first">Link</a>
				</article>
			</body>
		</html>
	`

	entries, err := ExtractHEntries(bytes.NewBufferString(htmlContent))
	if err != nil {
		t.Fatalf("ExtractHEntries failed: %v", err)
	}

	if len(entries) != 1 {
		t.Errorf("Expected 1 entry, got %d", len(entries))
	}

	// When canonical URL is missing, we should keep the original URL
	if entries[0].URL != "/first" {
		t.Errorf("Expected '/first', got '%s'", entries[0].URL)
	}
}
