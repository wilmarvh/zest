# Changelog

All notable changes to this project are documented here. The format is based on
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and this project
adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.1.0] - 2026-06-30

First tagged release.

### Added

- **Play album**: a `▶ Play album` row at the top of every album queues the whole
  album and plays it through in order, with shuffle forced off, instead of
  falling back to Music.app's own queue after one track.
- **Shuffle and repeat controls**: toggle shuffle with `s` and cycle repeat
  (off → all → one) with `r`. Both states are shown in the now-playing bar.
- **Album-name search**: search now matches artist, album, *and* track names
  (previously only artist and track names).
- **Search navigation**: use the arrow keys to move through results while the
  search box is still active; `enter` commits the search and selects the
  highlighted result in one press.

### Changed

- Pressing `enter` on a single track plays just that track and lets Music
  continue with whatever it would play next; queueing a whole album is now an
  explicit action via the `▶ Play album` row.
- Removed the now-playing visualizer graph in favour of a single, compact
  transport line.

### Fixed

- Album playback no longer aborts when the library lists a persistent ID that
  Music.app can't resolve (e.g. a duplicate cloud entry). Such tracks are
  skipped, and the rest of the album still queues; if no tracks resolve, a clear
  error is surfaced instead of silently doing nothing.

[0.1.0]: https://github.com/wilmarvh/zest/releases/tag/v0.1.0
