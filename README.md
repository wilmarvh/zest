# zest

A fast, keyboard-driven terminal UI for browsing and playing your **Apple Music**
library on macOS. Built with [Bubble Tea](https://github.com/charmbracelet/bubbletea).

`zest` reads your local library directly through Apple's `iTunesLibrary`
framework (no API keys, no entitlements) and controls playback by driving the
**Music.app** application via AppleScript.

```
┌─ zest ───────────────────────────────────────────────────────────────┐
│  Artists                │  Pink Floyd › The Wall                     │
│  ───────────            │  ──────────────────────                    │
│  ♪ Pink Floyd           │  1  In the Flesh?                    3:20  │
│    Radiohead            │  2  The Thin Ice                     2:29  │
│    Miles Davis          │  3  Another Brick…                   3:11  │
│                         │                                            │
│  ♪ Another Brick in the Wall, Pt. 2 — Pink Floyd      ▸ 1:42 / 3:59  │
└──────────────────────────────────────────────────────────────────────┘
```

## Features

- Browse your full library by **artist → album → track**
- **Play, pause, skip** tracks through Music.app
- Live **now-playing bar** with progress
- Live **search** across artists and track names
- Vim-style and arrow-key navigation
- Reads the library locally — fast even with tens of thousands of tracks

## Requirements

- macOS with the **Music** app and a populated library
- [Go 1.25+](https://go.dev/dl/) (uses CGo + the `iTunesLibrary` framework, so a
  recent Xcode Command Line Tools install is required to build)

## Install

```sh
go install github.com/wilmarvh/zest@latest
```

Or build from source:

```sh
git clone https://github.com/wilmarvh/zest.git
cd zest
go build -o zest .
./zest
```

Or run directly:

```sh
go run .
```

## Permissions

On first launch (and the first time you control playback), macOS will prompt you
to allow your terminal to:

1. **Read the Music library** — required to list your tracks.
2. **Control Music.app** (Automation permission, Terminal → Music) — required for
   playback.

Grant both. You can review or revoke these later in
**System Settings → Privacy & Security → Automation / Media & Apple Music**.

> If you accidentally deny the Automation prompt, playback will silently do
> nothing and `zest` will show *"grant permission: System Settings ▸ Privacy &
> Security ▸ Automation"*. Re-enable your terminal under **Automation → Music**
> and try again.

## Keys

| Key | Action |
|-----|--------|
| `↑` `↓` / `j` `k` | Navigate the focused pane |
| `tab` / `→` / `l` | Focus the content pane |
| `shift+tab` / `←` / `h` | Focus the sidebar / go back |
| `enter` | Drill down — on a track, play it |
| `space` | Play / pause |
| `n` / `p` | Next / previous track |
| `esc` | Back one level |
| `/` | Search |
| `q` / `ctrl+c` | Quit |

## How it works

- **Library** (`music/library.go`) — a small Objective-C shim, compiled via CGo,
  enumerates songs through `ITLibrary` and hands them to Go as delimited records.
- **Playback** (`music/player.go`) — issues `osascript` commands to Music.app and
  matches tracks by their persistent ID. Music.app's running state is checked
  *before* polling status so the app is never launched just to read state; on
  play it launches Music and waits for the library to load before issuing the
  command. The only value interpolated into an AppleScript is the persistent ID,
  and it's validated as plain hex first — so no metadata can wander into a
  command. AppleScript errors are classified (permission denied, track not
  playable, Music missing) so failures point at the actual fix.
- **UI** (`tui/`) — a Bubble Tea model renders a two-pane layout and polls the
  player every 2 seconds for the now-playing bar.

MusicKit was intentionally avoided: it requires a signed `.app` with
entitlements, which doesn't fit a `go build` CLI workflow.

## Troubleshooting

**"grant permission…" when I press play.** macOS hasn't authorized your
terminal to control Music. Enable it under **System Settings → Privacy &
Security → Automation → Music**, then try again.

**"can't play this track — it's not in Music's playable library".** Apple's
`iTunesLibrary` framework lists every track it knows about — including
cloud/subscription tracks that aren't downloaded and have no scriptable
counterpart in Music.app. `zest` can list these but cannot start playback on
them; download the track in Music (or play one that's available locally).

**Nothing happens and Music is closed.** `zest` launches Music for you and waits
for its library to finish loading before playing, so the first play after a
cold start can take a couple of seconds.

## Disclaimer

`zest` is **not affiliated with, endorsed by, or sponsored by Apple Inc.**
"Apple Music" and "Music.app" are trademarks of Apple Inc. The app only reads
your local library and controls the Music application through Apple's public
`iTunesLibrary` framework and AppleScript automation.

## Contributing

Issues and pull requests are welcome. See [CONTRIBUTING.md](CONTRIBUTING.md).

## License

Licensed under the [Apache License, Version 2.0](LICENSE).
See [NOTICE](NOTICE) for attribution and third-party acknowledgements.

Copyright © 2026 Wilmar van Heerden.
