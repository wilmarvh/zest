// Copyright 2026 Wilmar van Heerden
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package music

import (
	"errors"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

// errInvalidID is returned when a persistent ID isn't the hex form the
// iTunesLibrary framework hands us. It's our small bouncer at the door.
var errInvalidID = errors.New("invalid persistent ID")

// Sentinel errors that the TUI can match with errors.Is to show the user a
// message that actually points at the fix, instead of a generic "is it
// installed?".
var (
	// ErrNotAuthorized means macOS hasn't granted this terminal permission to
	// control Music (the Automation TCC prompt was never accepted, or was
	// denied). This is the #1 reason playback silently does nothing.
	ErrNotAuthorized = errors.New("not authorized to control Music")
	// ErrMusicMissing means the Music app itself couldn't be found.
	ErrMusicMissing = errors.New("Music app not found")
	// ErrTrackNotFound means Music is reachable but the track wasn't in its
	// library (e.g. the library hadn't finished loading after a cold launch).
	ErrTrackNotFound = errors.New("track not found in Music library")
)

// isHexID reports whether s looks like the %016llX persistent IDs we mint in
// library.go. We only ever feed osascript values that pass this check, so a
// stray quote or AppleScript fragment can never sneak into the script.
func isHexID(s string) bool {
	if s == "" {
		return false
	}
	for _, c := range s {
		switch {
		case c >= '0' && c <= '9':
		case c >= 'a' && c <= 'f':
		case c >= 'A' && c <= 'F':
		default:
			return false
		}
	}
	return true
}

// PlayerStatus reflects the Music app's current playback state.
type PlayerStatus struct {
	State        string // "playing", "paused", "stopped"
	Track        string
	Artist       string
	Position     float64 // seconds
	Duration     float64 // seconds
	PersistentID string  // uppercase hex, matches Track.PersistentID
}

func osascript(script string) (string, error) {
	cmd := exec.Command("osascript", "-e", script)
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		return strings.TrimSpace(stdout.String()), classifyOsascriptErr(stderr.String(), err)
	}
	return strings.TrimSpace(stdout.String()), nil
}

// classifyOsascriptErr maps osascript's stderr to a sentinel error so the UI
// can give actionable advice. osascript writes the human-readable message and
// the AppleEvent error number (e.g. "(-1743)") to stderr; Output() used to
// throw that away, leaving only "exit status 1".
func classifyOsascriptErr(stderr string, fallback error) error {
	msg := strings.TrimSpace(stderr)
	switch {
	// -1743: the user has not granted Automation permission for this app.
	// -10004 (errAEPrivilegeError) shows up the same way before the prompt.
	case strings.Contains(msg, "-1743"), strings.Contains(msg, "-10004"),
		strings.Contains(strings.ToLower(msg), "not allowed assistive"),
		strings.Contains(strings.ToLower(msg), "not authorized"),
		strings.Contains(strings.ToLower(msg), "doesn't have permission"):
		return fmt.Errorf("%w: %s", ErrNotAuthorized, msg)
	// -1728: can't get the object; -1700: empty `whose` result coerced to a
	// track. Both mean the track isn't in the library (or it never loaded).
	case strings.Contains(msg, "-1728"), strings.Contains(msg, "-1700"),
		strings.Contains(strings.ToLower(msg), "can’t get"),
		strings.Contains(strings.ToLower(msg), "can't get"):
		return fmt.Errorf("%w: %s", ErrTrackNotFound, msg)
	// -1728 on the app object itself, or -600/-10814: app not found/launchable.
	case strings.Contains(msg, "-600"), strings.Contains(msg, "-10814"),
		strings.Contains(strings.ToLower(msg), "application isn’t running"),
		strings.Contains(strings.ToLower(msg), "application can’t be found"):
		return fmt.Errorf("%w: %s", ErrMusicMissing, msg)
	}
	if msg != "" {
		return fmt.Errorf("%s: %w", msg, fallback)
	}
	return fallback
}

// PlayTrack starts playback of the library track with the given persistent ID.
// The ID is validated as hex before it reaches osascript, so even though it's
// interpolated into the script it can't carry an AppleScript payload.
func PlayTrack(persistentID string) error {
	if !isHexID(persistentID) {
		return errInvalidID
	}
	// On a cold launch, Music's library playlist is empty until the library
	// finishes loading, so a `play (... whose persistent ID ...)` would race it
	// and fail with -1728/-1700. We launch Music, then wait *only* while the
	// playlist is still empty (loading). Once it has tracks, a missing ID is a
	// genuine miss (e.g. a cloud track with no scriptable counterpart) and we
	// fail fast instead of spinning for seconds.
	script := fmt.Sprintf(`tell application "Music"
	launch
	-- Wait for the library to load (empty until then on a cold launch).
	repeat 40 times
		if (count of tracks of library playlist 1) > 0 then exit repeat
		delay 0.25
	end repeat
	set theTrack to (first track of library playlist 1 whose persistent ID is "%s")
	-- A freshly-launched player can swallow the first play command, reporting
	-- success while staying stopped. Issue play and confirm it actually
	-- started, then retry a few times before giving up.
	repeat 12 times
		play theTrack
		delay 0.25
		if player state is playing then return
	end repeat
end tell`, persistentID)
	_, err := osascript(script)
	return err
}

func PlayPause() error {
	_, err := osascript(`tell application "Music" to playpause`)
	return err
}

func NextTrack() error {
	_, err := osascript(`tell application "Music" to next track`)
	return err
}

func PreviousTrack() error {
	_, err := osascript(`tell application "Music" to previous track`)
	return err
}

// The running check stays outside the tell block so osascript doesn't launch Music.
const statusScript = `if application "Music" is not running then return "stopped"
tell application "Music"
	if player state is stopped then return "stopped"
	set sep to character id 31
	try
		set t to current track
	on error
		-- Playing/paused but no readable current track (e.g. mid-transition).
		return "stopped"
	end try
	return (player state as text) & sep & (name of t) & sep & (artist of t) & sep & (player position as text) & sep & ((duration of t) as text) & sep & (persistent ID of t)
end tell`

// Status polls the Music app. Returns a stopped status if Music isn't running.
func Status() (PlayerStatus, error) {
	out, err := osascript(statusScript)
	if err != nil {
		return PlayerStatus{State: "stopped"}, err
	}
	if out == "" || out == "stopped" {
		return PlayerStatus{State: "stopped"}, nil
	}
	parts := strings.Split(out, "\x1f")
	if len(parts) != 6 {
		return PlayerStatus{State: "stopped"}, nil
	}
	// AppleScript may use a comma decimal separator depending on locale.
	pos, _ := strconv.ParseFloat(strings.ReplaceAll(parts[3], ",", "."), 64)
	dur, _ := strconv.ParseFloat(strings.ReplaceAll(parts[4], ",", "."), 64)
	return PlayerStatus{
		State:        parts[0],
		Track:        parts[1],
		Artist:       parts[2],
		Position:     pos,
		Duration:     dur,
		PersistentID: parts[5],
	}, nil
}
