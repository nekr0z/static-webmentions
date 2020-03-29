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

//go:generate go run version_generate.go

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/PuerkitoBio/goquery"
	"willnorris.com/go/webmention"
)

type config struct {
	baseURL             string
	newDir              string
	oldDir              string
	excludeSources      []string
	excludeDestinations []string
	storage             string
}

type mention struct {
	Source string
	Dest   string
}

var version string = "custom"

func main() {
	var configFile string
	flag.StringVar(&configFile, "c", "config.toml", "config file")
	fn := flag.String("n", "", "new site version")
	fo := flag.String("o", "", "old site version")
	fb := flag.String("b", "", "base URL")
	fd := flag.String("f", "", "file to store pending webmentions")

	flag.Parse()

	fmt.Printf("static-webmentions version %s\n", version)
	cfg, err := readConfig(configFile)
	if err != nil {
		fmt.Printf("could not read config file %s: %s", configFile, err)
		os.Exit(1)
	}

	if *fn != "" {
		cfg.newDir = *fn
	}

	if *fo != "" {
		cfg.oldDir = *fo
	}

	if *fb != "" {
		cfg.baseURL = *fb
	}

	if *fd != "" {
		cfg.storage = *fd
	}

	if len(flag.Args()) > 1 {
		fmt.Println("too many arguments")
		os.Exit(1)
	}

	switch flag.Arg(0) {
	case "find":
		mentions, err := findWork(cfg)
		if err != nil {
			fmt.Printf("%v\n", err)
			os.Exit(1)
		}
		if err := dump(mentions, cfg.storage); err != nil {
			fmt.Printf("%v\n", err)
			os.Exit(1)
		}
	case "send":
		mentions, err := loadMentionsFromJSON(cfg.storage)
		if err != nil {
			fmt.Printf("%v\n", err)
			os.Exit(1)
		}
		sendMentions(mentions)
		fmt.Println("all sent")
	default:
		mentions, err := findWork(cfg)
		if err != nil {
			fmt.Printf("%v\n", err)
			os.Exit(1)
		}
		sendMentions(mentions)
		fmt.Println("all sent")
	}
}

func sendMentions(mentions []mention) {
	for _, m := range mentions {
		send(m.Source, m.Dest)
	}
}

func dump(mentions []mention, file string) error {
	switch file {
	case "":
		printMentions(mentions)
		return nil
	default:
		err := saveMentionsToJSON(mentions, file)
		return err
	}
}

func saveMentionsToJSON(mentions []mention, file string) error {
	bs, err := json.MarshalIndent(mentions, "", " ")
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(file, bs, 0644)
	return err
}

func loadMentionsFromJSON(file string) (mentions []mention, err error) {
	data, err := ioutil.ReadFile(file)
	if err != nil {
		return
	}
	err = json.Unmarshal(data, &mentions)
	return
}

func printMentions(mentions []mention) {
	for _, m := range mentions {
		fmt.Printf("%v -> %v\n", m.Source, m.Dest)
	}
}

func findWork(cfg config) ([]mention, error) {
	files, err := compareDirs(cfg)
	if err != nil {
		return nil, err
	}

	base := postSlash(cfg.baseURL)
	var mentions []mention

	for _, file := range files {
		path := filepath.Join(cfg.newDir, file)
		targets, err := getSources(path, cfg.baseURL, cfg.excludeDestinations, cfg.newDir)
		if err != nil {
			return nil, err
		}
		for _, target := range targets {
			m := mention{base + strings.TrimSuffix(file, "index.html"), target}
			mentions = append(mentions, m)
		}
	}
	return mentions, nil
}

func readConfig(path string) (config, error) {
	type webm struct {
		NewDir              string
		OldDir              string
		ExcludeSources      []string
		ExcludeDestinations []string
		WebmentionsFile     string
	}
	type configuration struct {
		BaseURL     string
		Webmentions webm
	}
	var cfg configuration
	_, err := toml.DecodeFile(path, &cfg)

	var conf config
	conf.baseURL = cfg.BaseURL
	conf.newDir = cfg.Webmentions.NewDir
	conf.oldDir = cfg.Webmentions.OldDir
	conf.excludeSources = cfg.Webmentions.ExcludeSources
	conf.excludeDestinations = cfg.Webmentions.ExcludeDestinations
	conf.storage = cfg.Webmentions.WebmentionsFile
	return conf, err
}

func send(source, target string) {
	client := webmention.New(nil)

	fmt.Printf("  %v ... ", target)
	endpoint, err := client.DiscoverEndpoint(target)
	if err != nil {
		fmt.Printf("%v", err)
	} else if endpoint == "" {
		fmt.Println("no webmention support")
	}

	_, err = client.SendWebmention(endpoint, source, target)
	if err != nil {
		fmt.Printf("%v", err)
	}
	fmt.Println("sent")
}

func compareDirs(conf config) ([]string, error) {
	var changedFiles []string

	err := filepath.Walk(conf.newDir,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return fmt.Errorf("%s: %w", path, err)
			}

			if path == conf.newDir {
				return nil
			}

			relPath := strings.TrimPrefix(path, strings.TrimSuffix(conf.newDir, "/")+"/")

			if strings.HasSuffix(relPath, ".htaccess") {
				gone := findGone(path, filepath.Join(conf.oldDir, relPath), conf.excludeSources, conf.newDir)
				changedFiles = append(changedFiles, gone...)
				return nil
			}

			if pathIsExcluded(relPath, conf.excludeSources) {
				return nil
			}

			if info.IsDir() {
				return nil
			}

			if fileNotChanged(path, filepath.Join(conf.oldDir, relPath)) {
				return nil
			}

			changedFiles = append(changedFiles, relPath)
			return nil
		})
	if err != nil {
		return nil, err
	}

	return changedFiles, err
}

func findGone(newPath, oldPath string, exclude []string, root string) []string {
	n, err := ioutil.ReadFile(newPath)
	if err != nil {
		return nil
	}

	gone := findGonePages(n, newPath, root)

	o, err := ioutil.ReadFile(oldPath)
	if err != nil {
		return gone
	}

	old := findGonePages(o, newPath, root)

	gone = subtractSlices(gone, old)

	return gone
}

func findGonePages(b []byte, path, root string) []string {
	var out []string
	relPath := strings.TrimPrefix(path, root)
	relPath = preSlash(relPath)
	lines := strings.Split(string(b), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		values := strings.Split(line, " ")
		if len(values) != 3 {
			continue
		}
		if values[0] != "Redirect" {
			continue
		}
		if values[1] != "gone" && values[1] != "410" {
			continue
		}
		value := strings.TrimSuffix(strings.TrimPrefix(values[2], "\""), "\"")
		if strings.HasPrefix(relPath, value) {
			value = strings.TrimSuffix(relPath, ".htaccess")
		}
		value = strings.TrimSuffix(strings.TrimPrefix(value, "/"), "/") + "/index.html"
		out = append(out, value)
	}
	return out
}

func subtractSlices(full, part []string) []string {
	var out []string

	for _, vf := range full {
		if inSlice(vf, part) {
			continue
		}
		out = append(out, vf)
	}

	return out
}

func inSlice(value string, slice []string) bool {
	for _, val := range slice {
		if value == val {
			return true
		}
	}
	return false
}

func fileNotChanged(oldPath, newPath string) bool {
	of, err := os.Open(oldPath)
	if err != nil {
		return true
	}
	defer of.Close()

	nf, err := os.Open(newPath)
	if err != nil {
		return false
	}
	defer nf.Close()

	o, _ := extractEntry(of)
	n, _ := extractEntry(nf)

	return o == n
}

func extractEntry(r io.Reader) (string, error) {
	doc, err := goquery.NewDocumentFromReader(r)
	if err != nil {
		return "", err
	}

	out, err := doc.Find(".h-entry").Html()
	if err != nil {
		return "", err
	}

	return out, nil
}

func pathIsExcluded(path string, exclude []string) bool {
	for _, ex := range exclude {
		if pathExcluded(path, ex) {
			return true
		}
	}
	return false
}

func pathExcluded(path, ex string) bool {
	switch strings.HasSuffix(ex, "*") {
	case true:
		return strings.HasPrefix(path, strings.TrimSuffix(strings.TrimPrefix(ex, "/"), "*"))
	default:
		ex = strings.TrimSuffix(ex, "index.html")
		ex = strings.TrimSuffix(ex, "/") + "/index.html"
		return path == ex
	}
}

func getSources(path string, base string, exclude []string, relPath string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	defer f.Close()

	links, err := webmention.DiscoverLinksFromReader(f, base, ".h-entry")
	if err != nil {
		return nil, nil
	}
	exclude = append(exclude, thisPage(path, relPath, base))

	links = cleanupSources(links, exclude)

	return links, nil
}

func thisPage(path, directory, base string) string {
	path = strings.TrimPrefix(strings.TrimPrefix(path, "/"), directory)
	path = strings.TrimPrefix(path, "/")
	this := postSlash(base) + strings.TrimSuffix(path, "index.html")
	return this
}

func cleanupSources(links, exclude []string) []string {
	var out []string

	for _, link := range links {
		if sourceMatch(link, exclude) {
			continue
		}
		out = append(out, link)
	}

	return out
}

func sourceMatch(link string, exclude []string) bool {
	for _, ex := range exclude {
		if exLink(link, ex) {
			return true
		}
	}
	return false
}

func exLink(source, ex string) bool {
	source = strings.TrimSuffix(source, "index.html")
	source = postSlash(source)
	ex = postSlash(ex)

	if source == ex {
		return true
	}

	if eqUnescaped(source, ex) {
		return true
	}

	sURL, err := url.Parse(source)
	if err != nil {
		return false
	}

	if sURL.Scheme == strings.TrimSuffix(ex, ":/") {
		return true
	}

	eURL, err := url.Parse(ex)
	if err != nil {
		return false
	}

	if eURL.IsAbs() {
		return false
	}

	s := strings.TrimPrefix(sURL.EscapedPath(), "/")
	e := strings.TrimPrefix(eURL.EscapedPath(), "/")

	return strings.HasPrefix(s, e)
}

func eqUnescaped(source, ex string) bool {
	us, err := url.PathUnescape(source)
	if err != nil {
		return false
	}
	ue, err := url.PathUnescape(ex)
	if err != nil {
		return false
	}
	return us == ue
}

func preSlash(s string) string {
	return "/" + strings.TrimPrefix(s, "/")
}

func postSlash(s string) string {
	return strings.TrimSuffix(s, "/") + "/"
}
