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

// Package hentry provides functionality for working with H-Feeds and H-Entries.
package hentry

import (
	"fmt"
	"io"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// HEntry represents a microformat h-entry.
type HEntry struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

// ExtractHEntries extracts h-entry elements from HTML content.
func ExtractHEntries(r io.Reader) ([]HEntry, error) {
	doc, err := goquery.NewDocumentFromReader(r)
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %w", err)
	}

	var entries []HEntry

	doc.Find(".h-entry").Each(func(i int, s *goquery.Selection) {
		var entry HEntry

		// Extract the name (p-name)
		if name := s.Find(".p-name").First(); name.Length() > 0 {
			entry.Name = strings.TrimSpace(name.Text())
		}

		// Extract the URL (u-url)
		if url := s.Find(".u-url").First(); url.Length() > 0 {
			if href, exists := url.Attr("href"); exists {
				entry.URL = href
			}
		}

		entries = append(entries, entry)
	})

	return entries, nil
}

// FindNewHEntries compares old and new entries and returns only the new ones.
func FindNewHEntries(oldEntries, newEntries []HEntry) []HEntry {
	oldURLs := make(map[string]bool)
	for _, entry := range oldEntries {
		oldURLs[entry.URL] = true
	}

	var newOnly []HEntry
	for _, entry := range newEntries {
		if !oldURLs[entry.URL] {
			newOnly = append(newOnly, entry)
		}
	}

	return newOnly
}
