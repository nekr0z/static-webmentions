// Copyright (C) 2020 Evgeny Kuznetsov (evgeny@kuznetsov.md)
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

package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
)

func TestFindFeeds(t *testing.T) {
	var cfg config
	cfg.baseURL = "https://my.site/"
	cfg.oldDir = filepath.Join("testdata", "prod")
	cfg.newDir = filepath.Join("testdata", "staging")

	want := []string{
		"https://my.site/index.xml",
		"https://my.site/posts/index.xml",
		"https://my.site/tags/b/index.xml",
	}

	got := findFeeds(cfg)

	if !stringSlicesEqual(want, got) {
		t.Fatalf("want:\n%s\ngot:\n%s\n", want, got)
	}
}

func TestPing(t *testing.T) {
	buf := new(bytes.Buffer)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		buf.ReadFrom(r.Body)
	}))
	defer ts.Close()

	tests := map[string]struct {
		feeds  []string
		result string
	}{
		"empty":     {[]string{}, ""},
		"non-empty": {[]string{"one", "two"}, "hub.mode=publish&hub.url%5B%5D=one&hub.url%5B%5D=two"},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			ping(ts.URL, tc.feeds)
			got := buf.String()
			if tc.result != got {
				t.Fatalf("want %v, got: %v", tc.result, got)
			}
		})
	}
}
