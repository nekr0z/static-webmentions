# Changelog
All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Pre-release]

## [0.5.0] - 2021-10-04
### Added
- an option to specify CSS selectors that would be included in changes discovery along with the existing `h-entry`
- an option to specify CSS selectors that would be excluded from gathering targets when sending webmentions

## [0.4.0] - 2021-10-03
### Changed
- bumped Go to 1.15

### Fixed
- webmentions not re-sent for the links that were removed from a page

## [0.3.2] - 2020-10-30
- maintenance release, updated dependencies

## [0.3.1] - 2020-08-09
### Fixed
- hopefully fixed occasional panic by updating dependencies

## [0.3.0] - 2020-06-03
### Breaking
- require full path for exclusions in config
- don't bother with .htaccess, rely on tombstones instead for detecting gone pages

### Fixed
- actually re-send webmentions for gone pages
- detect un-gone pages

## [0.2.1] - 2020-05-11
### Added
- support non-standart feed filenames

### Fixed
- websub pinger would panic on response error

## [0.2.0] - 2020-05-06
### Added
- support for multiple WebSub hubs

### Fixed
- trying to send webmentions when endpoint not found
- sending websub ping for multiple feeds at once

## [0.1.3] - 2020-04-02
### Fixed
- links with fragments were not excluded

## [0.1.2] - 2020-04-01
### Added
- finding XML feeds and pinging WebSub hub on changes

## [0.1.1] - 2020-03-29
### Fixed
- errors if symlinks to directories are present

## [0.1.0] - 2020-03-29
*initial release*

[Pre-release]: https://github.com/nekr0z/static-webmentions/releases/tag/latest
[0.5.0]: https://github.com/nekr0z/static-webmentions/releases/tag/v0.5.0
[0.4.0]: https://github.com/nekr0z/static-webmentions/releases/tag/v0.4.0
[0.3.2]: https://github.com/nekr0z/static-webmentions/releases/tag/v0.3.2
[0.3.1]: https://github.com/nekr0z/static-webmentions/releases/tag/v0.3.1
[0.3.0]: https://github.com/nekr0z/static-webmentions/releases/tag/v0.3.0
[0.2.1]: https://github.com/nekr0z/static-webmentions/releases/tag/v0.2.1
[0.2.0]: https://github.com/nekr0z/static-webmentions/releases/tag/v0.2.0
[0.1.3]: https://github.com/nekr0z/static-webmentions/releases/tag/v0.1.3
[0.1.2]: https://github.com/nekr0z/static-webmentions/releases/tag/v0.1.2
[0.1.1]: https://github.com/nekr0z/static-webmentions/releases/tag/v0.1.1
[0.1.0]: https://github.com/nekr0z/static-webmentions/releases/tag/v0.1.0
