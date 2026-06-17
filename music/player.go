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
	out, err := exec.Command("osascript", "-e", script).Output()
	return strings.TrimSpace(string(out)), err
}

// PlayTrack starts playback of the library track with the given persistent ID.
// The ID is validated as hex before it reaches osascript, so even though it's
// interpolated into the script it can't carry an AppleScript payload.
func PlayTrack(persistentID string) error {
	if !isHexID(persistentID) {
		return errInvalidID
	}
	script := fmt.Sprintf(`tell application "Music" to play (first track of library playlist 1 whose persistent ID is "%s")`, persistentID)
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
	set t to current track
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
