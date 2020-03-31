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
	data := url.Values{}
	data.Set("hub.mode", "publish")
	feedsString := uniteFeeds(feeds)
	data.Set("hub.url", feedsString)

	u, _ := url.ParseRequestURI(hub)
	urlStr := u.String()

	client := &http.Client{}
	r, _ := http.NewRequest("POST", urlStr, strings.NewReader(data.Encode())) // URL-encoded payload
	r.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	fmt.Printf("pinging %s\n for %s\n", hub, feedsString)
	resp, _ := client.Do(r)
	fmt.Println(resp.Status)
}

func uniteFeeds(feeds []string) string {
	var urls []string
	for _, feed := range feeds {
		urls = append(urls, "<"+feed+">")
	}
	s := strings.Join(urls, ",")
	return s
}

func findFeeds(conf config) []string {
	var files []string
	_ = filepath.Walk(conf.newDir,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil
			}

			relPath := strings.TrimPrefix(path, strings.TrimSuffix(conf.newDir, "/")+"/")

			if !strings.HasSuffix(relPath, "index.xml") {
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
	equal, err := cmp.CompareFile("file1", "file2")
	if err != nil {
		return true
	}
	return !equal
}
