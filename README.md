# static-webmentions
a tool to send webmentions from a static website

[![Build Status](https://github.com/nekr0z/static-webmentions/actions/workflows/pre-release.yml/badge.svg)](https://travis-ci.org/nekr0z/static-webmentions) [![codecov](https://codecov.io/gh/nekr0z/static-webmentions/branch/master/graph/badge.svg)](https://codecov.io/gh/nekr0z/static-webmentions) [![Go Report Card](https://goreportcard.com/badge/github.com/nekr0z/static-webmentions)](https://goreportcard.com/report/github.com/nekr0z/static-webmentions)

##### Table of Contents
* [Why](#why)
* [How to use](#how)
  * [Configuration](#configuration)
  * [Automation](#automating-your-webmentions)
* [Development](#development)
* [Credits](#credits)

#### Help `static-webmentions` get better!
Join the [development](#development) (or just [buy me a coffee](https://www.buymeacoffee.com/nekr0z), that helps, too).

## Why
Sending [webmentions](https://en.wikipedia.org/wiki/Webmention) is relatively easy if you use a CMS. If, however, your site is built using a static site generator like [Hugo](https://gohugo.io/), sending webmentions is not as straightforward.

This tool allows you to do just that, and with some effort you can even automate your outgoing webmentions.

## How
The way `static-webmentions` works is by comparing two versions of a static site (residing in two separate directories), figuring out what new webmentions are to be sent, and, eventually, sending them. Thus, you need to make sure that at some point of your publishing process you have two directories: one with the old state of your site, and one with the new state. Given that you have `static-webmentions` [configured](#configuration) correctly, at this point you can: 
```
static-webmentions find
```
to find out which webmentions are to be sent and save them to a temporary file, and then, when the new content is online and alive,
```
static-webmentions send
```
to read the pending webmentions from the temporary file and send them out.

If your publishing process is set up so that at the moment when you have the two versions the new version is already published, you can do both steps in one go:
```
static-webmentions
```

### Configuration
`static-webmentions` will look for a config file (`config.toml` in current directory by default). An example config file with all the supported options is [included](config.toml), all the options are fairly self-explanatory. [Hugo](https://gohugo.in) users may want to use the same `config.toml` that they store their website configuration in.

Several command-line flags can be used to override the options specified in the config file:
```
-c [filename]
```
the config file to read config from. Default is `config.toml` in current working directory.

```
-b [url]
```
base URL of the site (i.e. `http://my-awesome-site.web`).

```
-f [filename]
```
the file to store pending webmentions to, or to read them from if `send` command is given. If no file is specified, `static-webmentions find` will just print out the list of webmentions it found.

```
-o [directory] -n [directory]
```
the directories where old and new versions of the site are located. 

### Automation
Depending on your publishing process, automating your outgoing webmentions may require different steps. Here's an example how I do it for my own website.

The website is built using [Hugo](https://hugo.io), the source is stored in a Git repository. I put both `hugo` and `static-webmentions` binaries on the hosting server, and can publish a new version of the site with a simple script:
```
#!/bin/sh
cd /home/my-user/hugo                       # this is the folder with my website source
git pull
rsync -am --del public old                  # the current public directory is copied to old/public
hugo --gc --cleanDestinationDir -d public   # hugo puts new version in public directory
static-webmentions                          # compares public to old/public and sends webmentions
```
The actual script I use is more complex and the actual process is more error-proof, the script above is a minimal working example.

## Development
Pull requests are always welcome!

## Credits
This software includes the following software or parts thereof:
* [willnorris/webmention](https://willnorris.com/go/webmention) Copyright 2014, Google Inc. All rights reserved.
* [goquery](https://github.com/PuerkitoBio/goquery) Copyright © 2012-2016, Martin Angers & Contributors
* [BurntSushi/toml](https://github.com/BurntSushi/toml) Copyright © 2013 TOML authors
* [udhos/equalfile](https://github.com/udhos/equalfile) Copyright © 2016 udhos
* [go-cmp](https://github.com/google/go-cmp) Copyright © 2017 The Go Authors. All rights reserved.
* [cascadia](https://github.com/andybalholm/cascadia) Copyright © 2011 Andy Balholm. All rights reserved.
* [wsxiaoys/terminal](https://github.com/wsxiaoys/terminal) Copyright © 2013 Meng Zhang. All rights reserved.
* [The Go Programming Language](https://golang.org) Copyright © 2009 The Go Authors
