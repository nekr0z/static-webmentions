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
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func TestFindWork(t *testing.T) {
	conf, err := readConfig("config.toml")
	if err != nil {
		t.Fatal(err)
	}

	mm, err := findWork(conf)
	if err != nil {
		t.Fatal(err)
	}

	var got []string
	for _, m := range mm {
		got = append(got, m.Dest)
	}

	want := []string{
		"http://resend.me",
		"http://kuznetsov.md",
		"https://my-awesome.site/testdata/page/",
		"http://some.site/post/title",
		"https://my-awesome.site/other/",
		"https://my-awesome.site/testdata/page/",
		"http://some.site/post/title",
		"https://my-awesome.site/other/",
		"http://example.site/post",
	}

	if !stringSlicesEqual(got, want) {
		t.Fatalf("want: %v\ngot: %v", want, got)
	}
}

func TestGetSources(t *testing.T) {
	path := filepath.Join("testdata", "page", "index.html")
	base := "https://my-awesome.site"
	got, err := getSources(path, base, []string{base, "mailto:", "/tags"}, []string{}, "")
	if err != nil {
		t.Fatal(err)
	}

	want := []string{
		"http://some.site/post/title",
		"https://my-awesome.site/other/",
	}

	if !stringSlicesEqual(got, want) {
		t.Fatalf("want: %v\ngot: %v", want, got)
	}
}

func TestGetSourcesError(t *testing.T) {
	path := filepath.Join("testdata", "page", "page")
	gotL, gotE := getSources(path, "", []string{}, []string{}, "")
	var wantL []string
	var wantE error
	wantL = nil
	wantE = nil
	if !stringSlicesEqual(gotL, wantL) {
		t.Fatalf("want: %s\n got: %s", wantL, gotL)
	}
	if gotE != wantE {
		t.Fatalf("want: %s\n got: %s", wantE, gotE)
	}
}

func TestCompareDirs(t *testing.T) {
	conf, err := readConfig("config.toml")
	if err != nil {
		t.Fatal(err)
	}

	got, err := compareDirs(conf)
	if err != nil {
		t.Fatal(err)
	}

	want := []string{
		"extra_tag.html",
		"ignored_css.html",
		"other.xml",
		"posts/1/index.html",
		"posts/2/index.html",
		"posts/3/index.html",
		"posts/4/index.html",
	}

	if !stringSlicesEqual(got, want) {
		t.Fatalf("want: %v\ngot: %v", want, got)
	}
}

func TestExLink(t *testing.T) {
	tests := map[string]struct {
		source string
		ex     string
		want   bool
	}{
		"russian": {source: "https://site.org/%D0%BD%D0%B0%D1%89%D0%B0%D0%BB%D1%8C%D0%BD%D0%B8%D0%BA%D0%B0", ex: "https://site.org/нащальника", want: true},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := exLink(tc.source, tc.ex)
			if got != tc.want {
				t.Fatalf("want: %v, got: %v", tc.want, got)
			}
		})
	}
}

func TestThisPage(t *testing.T) {
	tests := map[string]struct {
		path string
		dir  string
		base string
		want string
	}{
		"reaction": {
			path: "public/reactions/test/index.html",
			dir:  "public",
			base: "https://beta.evgenykuznetsov.org/",
			want: "https://beta.evgenykuznetsov.org/reactions/test/",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := thisPage(tc.path, tc.dir, tc.base)
			if got != tc.want {
				t.Fatalf("want: %v, got: %v", tc.want, got)
			}
		})
	}
}

func TestSend(t *testing.T) {
	src := "source"

	okay := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))
	defer okay.Close()
	fail := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(400)
	}))
	defer fail.Close()

	empty := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `nothing here!`)
	}))
	defer empty.Close()
	bad := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `<link rel="webmention" href="%s" />`, fail.URL)
	}))
	defer bad.Close()
	good := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `<link rel="webmention" href="%s" />`, okay.URL)
	}))
	defer good.Close()
	bridgy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Location", "https://silo.org/me/status/42")
		w.WriteHeader(201)
	}))
	defer bridgy.Close()
	creator := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `<link rel="webmention" href="%s" />`, bridgy.URL)
	}))
	defer creator.Close()

	tests := map[string]struct {
		url  string
		want string
	}{
		"good":        {good.URL, "webmention for " + good.URL + " sent"},
		"bridgy":      {creator.URL, "webmention for " + creator.URL + " sent\ncreated for source is https://silo.org/me/status/42"},
		"failed send": {bad.URL, "could not send webmention for " + bad.URL + ": response error: 400"},
		"bad page":    {"destination", "could not discover endpoint for destination: Get \"destination\": unsupported protocol scheme \"\""},
		"no endpoint": {empty.URL, "could not discover endpoint for " + empty.URL + ": no webmention rel found"},
	}

	rescueStdout := os.Stdout
	defer func() { os.Stdout = rescueStdout }()
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			r, w, _ := os.Pipe()
			os.Stdout = w
			var wg sync.WaitGroup
			wg.Add(1)
			sc := make(map[string]chan struct{})
			send(src, tc.url, &wg, sc, 15)
			w.Close()
			out, _ := ioutil.ReadAll(r)
			got := strings.SplitN(strings.TrimRight(string(out), "\n"), "\n", 2)[1]
			if tc.want != got {
				t.Fatalf("\nwant %q,\ngot: %q", tc.want, got)
			}
		})
	}
}

func stringSlicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i, v := range a {
		if v != b[i] {
			return false
		}
	}
	return true
}
