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

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework iTunesLibrary -framework Foundation
#import <iTunesLibrary/iTunesLibrary.h>

#include <stdlib.h>

// fetchTracks returns all songs as newline-delimited records.
// Fields separated by ASCII 0x1F (unit separator) to avoid collisions with
// any character that might appear in metadata:
//   title \x1f artist \x1f album \x1f genre \x1f duration_ms \x1f play_count \x1f rating \x1f persistent_id \x1f track_number
const char* fetchTracks(void) {
    NSError *error = nil;
    ITLibrary *lib = [ITLibrary libraryWithAPIVersion:@"1.0" error:&error];
    if (!lib) {
        return strdup("");
    }

    NSMutableString *out = [NSMutableString stringWithCapacity:1024 * 1024];
    NSString *sep = @"\x1f";

    for (ITLibMediaItem *item in lib.allMediaItems) {
        if (item.mediaKind != ITLibMediaItemMediaKindSong) continue;

        NSString *title     = item.title      ?: @"";
        NSString *artist    = item.artist.name ?: @"";
        NSString *album     = item.album.title ?: @"";
        NSString *genre     = item.genre       ?: @"";
        long      duration  = (long)item.totalTime;  // milliseconds
        long      playCount = (long)item.playCount;
        long      rating    = (long)item.rating;     // 0-100
        long      trackNum  = (long)item.trackNumber;
        // %016llX matches the persistent ID format AppleScript exposes.
        unsigned long long pid = item.persistentID.unsignedLongLongValue;

        [out appendFormat:@"%@%@%@%@%@%@%@%@%ld%@%ld%@%ld%@%016llX%@%ld\n",
            title,    sep,
            artist,   sep,
            album,    sep,
            genre,    sep,
            duration, sep,
            playCount, sep,
            rating,   sep,
            pid,      sep,
            trackNum];
    }

    // Return a C string the Go side must free.
    return strdup([out UTF8String]);
}
*/
import "C"

import (
	"sort"
	"strconv"
	"strings"
	"unsafe"
)

// Track holds metadata for a single song.
type Track struct {
	Name         string
	Artist       string
	Album        string
	Genre        string
	Duration     int // milliseconds
	PlayCount    int
	Rating       int // 0-100
	PersistentID string
	TrackNumber  int
}

// DurationString formats Duration as "m:ss".
func (t Track) DurationString() string {
	if t.Duration <= 0 {
		return ""
	}
	total := t.Duration / 1000
	m := total / 60
	s := total % 60
	return strings.TrimSpace(strconv.Itoa(m) + ":" + func() string {
		if s < 10 {
			return "0" + strconv.Itoa(s)
		}
		return strconv.Itoa(s)
	}())
}

type Album struct {
	Name   string
	Tracks []Track
}

type Artist struct {
	Name   string
	Albums []Album
}

type Library struct {
	Artists []Artist
	Tracks  []Track // flat list for search
}

// Load fetches the library via the iTunesLibrary framework (CGo).
func Load() (*Library, error) {
	cstr := C.fetchTracks()
	defer C.free(unsafe.Pointer(cstr))
	return parse(C.GoString(cstr)), nil
}

func parse(raw string) *Library {
	lines := strings.Split(strings.TrimSpace(raw), "\n")

	artistMap := map[string]map[string][]Track{}
	var allTracks []Track

	for _, line := range lines {
		if line == "" {
			continue
		}
		parts := strings.Split(line, "\x1f")
		if len(parts) != 9 {
			continue
		}

		duration, _ := strconv.Atoi(parts[4])
		playCount, _ := strconv.Atoi(parts[5])
		rating, _ := strconv.Atoi(parts[6])
		trackNum, _ := strconv.Atoi(parts[8])

		t := Track{
			Name:         parts[0],
			Artist:       parts[1],
			Album:        parts[2],
			Genre:        parts[3],
			Duration:     duration,
			PlayCount:    playCount,
			Rating:       rating,
			PersistentID: parts[7],
			TrackNumber:  trackNum,
		}
		allTracks = append(allTracks, t)

		artist := t.Artist
		if artist == "" {
			artist = "Unknown Artist"
		}
		album := t.Album
		if album == "" {
			album = "Unknown Album"
		}

		if _, ok := artistMap[artist]; !ok {
			artistMap[artist] = map[string][]Track{}
		}
		artistMap[artist][album] = append(artistMap[artist][album], t)
	}

	artistNames := make([]string, 0, len(artistMap))
	for name := range artistMap {
		artistNames = append(artistNames, name)
	}
	sort.Strings(artistNames)

	artists := make([]Artist, 0, len(artistNames))
	for _, artistName := range artistNames {
		albumMap := artistMap[artistName]
		albumNames := make([]string, 0, len(albumMap))
		for name := range albumMap {
			albumNames = append(albumNames, name)
		}
		sort.Strings(albumNames)

		albums := make([]Album, 0, len(albumNames))
		for _, albumName := range albumNames {
			tracks := albumMap[albumName]
			sort.SliceStable(tracks, func(i, j int) bool {
				return tracks[i].TrackNumber < tracks[j].TrackNumber
			})
			albums = append(albums, Album{Name: albumName, Tracks: tracks})
		}
		artists = append(artists, Artist{Name: artistName, Albums: albums})
	}

	return &Library{
		Artists: artists,
		Tracks:  allTracks,
	}
}
