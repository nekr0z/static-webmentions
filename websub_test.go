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
