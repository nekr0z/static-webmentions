# Changelog
All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]
### Added
- support non-standart feed filenames

### Fixed
- websub pinger would panic on response error

## [0.2.0] - 2019-05-06
### Added
- support for multiple WebSub hubs

### Fixed
- trying to send webmentions when endpoint not found
- sending websub ping for multiple feeds at once

## [0.1.3] - 2019-04-02
### Fixed
- links with fragments were not excluded

## [0.1.2] - 2019-04-01
### Added
- finding XML feeds and pinging WebSub hub on changes

## [0.1.1] - 2019-03-29
### Fixed
- errors if symlinks to directories are present

## 0.1.0 - 2020-03-29
*initial release*

[Unreleased]: https://github.com/nekr0z/static-webmentions/compare/v0.2.0...HEAD
[0.2.0]: https://github.com/nekr0z/static-webmentions/compare/v0.1.3...v0.2.0
[0.1.3]: https://github.com/nekr0z/static-webmentions/compare/v0.1.2...v0.1.3
[0.1.2]: https://github.com/nekr0z/static-webmentions/compare/v0.1.1...v0.1.2
[0.1.1]: https://github.com/nekr0z/static-webmentions/compare/v0.1.0...v0.1.1
