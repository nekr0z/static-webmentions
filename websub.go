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
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/udhos/equalfile"
)

func ping(hub string, feeds []string) {
	if len(feeds) == 0 {
		return
	}

	u, _ := url.ParseRequestURI(hub)
	urlStr := u.String()

	for _, feed := range feeds {
		data := url.Values{}
		data.Set("hub.mode", "publish")
		data.Set("hub.url", feed)

		client := &http.Client{}
		r, _ := http.NewRequest("POST", urlStr, strings.NewReader(data.Encode())) // URL-encoded payload
		r.Header.Add("Content-Type", "application/x-www-form-urlencoded")

		fmt.Printf("pinging %s for: %s ... ", hub, feed)
		resp, _ := client.Do(r)
		fmt.Println(resp.Status)
	}
}

func findFeeds(conf config) []string {
	var files []string
	_ = filepath.Walk(conf.newDir,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil
			}

			relPath := strings.TrimPrefix(path, strings.TrimSuffix(conf.newDir, "/")+"/")

			if !suffixInArray(relPath, conf.feedFiles) {
				return nil
			}

			if info.IsDir() {
				return nil
			}

			if !feedChanged(path, filepath.Join(conf.oldDir, relPath)) {
				return nil
			}

			files = append(files, relPath)
			return nil
		})
	var feeds []string
	for _, file := range files {
		feed := postSlash(conf.baseURL) + strings.TrimPrefix(file, "/")
		feeds = append(feeds, feed)
	}
	return feeds
}

func feedChanged(newFile, oldFile string) bool {
	cmp := equalfile.New(nil, equalfile.Options{}) // compare using single mode
	r1, err := os.Open(newFile)
	if err != nil {
		return false
	}
	defer r1.Close()
	r2, err := os.Open(oldFile)
	if err != nil {
		return true
	}
	defer r2.Close()

	equal, err := cmp.CompareReader(r1, r2)
	if err != nil {
		return true
	}
	return !equal
}

func suffixInArray(s string, a []string) bool {
	for _, e := range a {
		if strings.HasSuffix(s, e) {
			return true
		}
	}
	return false
}
