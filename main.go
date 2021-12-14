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
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/BurntSushi/toml"
	"github.com/PuerkitoBio/goquery"
	"willnorris.com/go/webmention"
)

type config struct {
	baseURL             string
	newDir              string
	oldDir              string
	alsoWatch           []string
	excludeSources      []string
	excludeDestinations []string
	excludeSelectors    []string
	storage             string
	websubHub           []string
	feedFiles           []string
	concurFiles         int
	concurReqs          int
}

type mention struct {
	Source string
	Dest   string
}

var version string = "custom"

var errPageDeleted = errors.New("410")

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
		sendMentions(mentions, cfg.concurReqs)
		fmt.Println("all sent")
	default:
		mentions, err := findWork(cfg)
		if err != nil {
			fmt.Printf("%v\n", err)
			os.Exit(1)
		}
		if len(cfg.websubHub) != 0 {
			feeds := findFeeds(cfg)
			for _, hub := range cfg.websubHub {
				ping(hub, feeds)
			}
		}
		sendMentions(mentions, cfg.concurReqs)
		fmt.Println("all sent")
	}
}

func sendMentions(mentions []mention, smax int) {
	sc := make(map[string]chan struct{})
	var wg sync.WaitGroup
	for _, m := range mentions {
		wg.Add(1)
		go send(m.Source, m.Dest, &wg, sc, smax)
	}
	wg.Wait()
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
	var mentions struct {
		mu sync.Mutex
		mm []mention
	}

	cc := make(chan struct{}, cfg.concurFiles)
	wgDone := make(chan bool)
	errors := make(chan error)
	var wg sync.WaitGroup

	for _, file := range files {
		wg.Add(1)
		cc <- struct{}{}

		go func(file string) {
			defer wg.Done()
			defer func() { <-cc }()
			path := filepath.Join(cfg.oldDir, file)
			oldtargets, _ := getSources(path, cfg.baseURL, cfg.excludeDestinations, cfg.excludeSelectors, cfg.oldDir)
			path = filepath.Join(cfg.newDir, file)
			targets, err := getSources(path, cfg.baseURL, cfg.excludeDestinations, cfg.excludeSelectors, cfg.newDir)
			if err != nil {
				if err == errPageDeleted {
					targets, err = getSources(filepath.Join(cfg.oldDir, file), cfg.baseURL, cfg.excludeDestinations, cfg.excludeSelectors, cfg.oldDir)
					if err != nil {
						return
					}
				} else {
					errors <- err
					return
				}
			}
			targets = appendDedupe(targets, oldtargets...)
			for _, target := range targets {
				m := mention{base + strings.TrimSuffix(file, "index.html"), target}
				mentions.mu.Lock()
				mentions.mm = append(mentions.mm, m)
				mentions.mu.Unlock()
			}
		}(file)
	}
	go func() {
		wg.Wait()
		close(wgDone)
	}()
	select {
	case <-wgDone:
		break
	case err := <-errors:
		return nil, err
	}
	return mentions.mm, nil
}

func readConfig(path string) (config, error) {
	type webm struct {
		NewDir              string
		OldDir              string
		AlsoWatch           []string
		ExcludeSources      []string
		ExcludeDestinations []string
		ExcludeSelectors    []string
		WebmentionsFile     string
		ConcurrentFiles     int
		ConcurrentRequests  int
	}
	type params struct {
		WebsubHub []string
		FeedFiles []string
	}
	type configuration struct {
		BaseURL     string
		Webmentions webm
		Params      params
	}
	var cfg configuration
	_, err := toml.DecodeFile(path, &cfg)

	var conf config
	conf.baseURL = cfg.BaseURL
	conf.newDir = cfg.Webmentions.NewDir
	conf.oldDir = cfg.Webmentions.OldDir
	conf.alsoWatch = cfg.Webmentions.AlsoWatch
	conf.excludeSources = cfg.Webmentions.ExcludeSources
	conf.excludeDestinations = cfg.Webmentions.ExcludeDestinations
	conf.excludeSelectors = cfg.Webmentions.ExcludeSelectors
	conf.storage = cfg.Webmentions.WebmentionsFile
	conf.websubHub = cfg.Params.WebsubHub
	conf.concurFiles = cfg.Webmentions.ConcurrentFiles - 1
	if conf.concurFiles < 0 {
		conf.concurFiles = 0
	}
	conf.concurReqs = cfg.Webmentions.ConcurrentRequests - 1
	if conf.concurReqs < 0 {
		conf.concurReqs = 0
	}
	if len(cfg.Params.FeedFiles) == 0 {
		conf.feedFiles = []string{"index.xml"}
	} else {
		conf.feedFiles = cfg.Params.FeedFiles
	}
	return conf, err
}

func send(source, target string, wg *sync.WaitGroup, sc map[string]chan struct{}, smax int) {
	defer wg.Done()
	u, err := url.Parse(target)
	if err != nil {
		fmt.Printf("  %v doesn't look like a parsable URL\n", target)
		return
	}
	if _, ok := sc[u.Host]; !ok {
		sc[u.Host] = make(chan struct{}, smax)
	}
	sc[u.Host] <- struct{}{}

	fmt.Printf("processing webmention for %v ...\n", target)
	client := webmention.New(nil)

	endpoint, err := client.DiscoverEndpoint(target)
	<-sc[u.Host]
	if err != nil {
		fmt.Printf("could not discover endpoint for %v: %v\n", target, err)
		return
	}
	u, err = url.Parse(endpoint)
	if err != nil {
		fmt.Printf("%v: discovered enpoint (%v) doesn't look like a parsable URL\n", target, endpoint)
		return
	}
	if _, ok := sc[u.Host]; !ok {
		sc[u.Host] = make(chan struct{}, smax)
	}
	sc[u.Host] <- struct{}{}
	defer func() { <-sc[u.Host] }()

	r, err := client.SendWebmention(endpoint, source, target)
	if err != nil {
		fmt.Printf("could not send webmention for %v: %v\n", target, err)
		return
	}
	fmt.Printf("webmention for %v sent\n", target)

	if r.StatusCode == 201 {
		fmt.Printf("created for %v is %s", source, r.Header.Get("location"))
	}
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

			if pathIsExcluded(relPath, conf.excludeSources) {
				return nil
			}

			if info.IsDir() {
				return nil
			}

			if fileNotChanged(path, filepath.Join(conf.oldDir, relPath), conf.alsoWatch) {
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

func fileNotChanged(oldPath, newPath string, addSel []string) bool {
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

	o, _ := extractEntryAndSel(of, addSel)
	n, _ := extractEntryAndSel(nf, addSel)

	return o == n
}

// extractEntryAndSel returns the HTML representation of the first h-entry found in `r`,
// along with anything matching the additional CSS selectors `as`
func extractEntryAndSel(r io.Reader, as []string) (string, error) {
	doc, err := goquery.NewDocumentFromReader(r)
	if err != nil {
		return "", err
	}

	out, err := doc.Find(".h-entry").Html()
	if err != nil {
		return "", err
	}

	s := strings.Join(as, ", ")
	doc.Find(s).Each(func(_ int, sel *goquery.Selection) {
		h, err := sel.Html()
		if err != nil {
			return
		}
		out = fmt.Sprintf("%s\n%s", out, h)
	})

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
		path = "/" + path
		ex = strings.TrimSuffix(ex, "index.html")
		ex = strings.TrimSuffix(ex, "/") + "/index.html"
		return path == ex
	}
}

func getSources(path string, base string, exclude []string, excludeCSS []string, relPath string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	defer f.Close()

	if isDeleted(f) {
		return nil, errPageDeleted
	}
	_, err = f.Seek(0, 0)
	if err != nil {
		return nil, err
	}

	d, err := goquery.NewDocumentFromReader(f)
	if err != nil {
		return nil, nil
	}

	s := d.Find(".h-entry")

	h, err := s.Html()
	if err != nil {
		return nil, err
	}

	// this is ugly, but I can't figure out how to do it with goquery
	for _, ex := range excludeCSS {
		exc, err := s.Find(ex).Html()
		if err != nil {
			return nil, err
		}
		if exc == "" {
			continue
		}
		h = strings.ReplaceAll(h, exc, "\n")
	}

	links, err := webmention.DiscoverLinksFromReader(strings.NewReader(h), postSlash(base), "")
	if err != nil {
		return nil, nil
	}
	exclude = append(exclude, thisPage(path, relPath, base))

	links = cleanupSources(links, exclude)

	return links, nil
}

func isDeleted(r io.Reader) bool {
	doc, err := goquery.NewDocumentFromReader(r)
	if err != nil {
		return false
	}

	gone := false

	doc.Find("meta").Each(func(_ int, s *goquery.Selection) {
		if _, ok := s.Attr("http-equiv"); ok {
			if v, ok := s.Attr("content"); ok {
				if strings.HasPrefix(v, "410") {
					gone = true
				}
			}
		}
	})

	return gone
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
	// first check for fragments and recurse if any
	sURL, err := url.Parse(source)
	if err == nil {
		noFrag := *sURL
		noFrag.Fragment = ""
		if *sURL != noFrag {
			return exLink(noFrag.String(), ex)
		}
	}

	source = strings.TrimSuffix(source, "index.html")
	source = postSlash(source)
	ex = postSlash(ex)

	if source == ex {
		return true
	}

	if eqUnescaped(source, ex) {
		return true
	}

	sURL, err = url.Parse(source)
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

func postSlash(s string) string {
	return strings.TrimSuffix(s, "/") + "/"
}

func appendDedupe(a []string, b ...string) (out []string) {
	a = append(a, b...)
lp:
	for _, s := range a {
		for _, str := range out {
			if eqUnescaped(s, str) {
				continue lp
			}
		}
		out = append(out, s)
	}
	return
}
