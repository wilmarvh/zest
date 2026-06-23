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
	"testing"
)

func TestClassifyOsascriptErr(t *testing.T) {
	fallback := errors.New("exit status 1")
	cases := []struct {
		name   string
		stderr string
		want   error
	}{
		{"perm by number", "execution error: Not authorized to send Apple events to Music. (-1743)", ErrNotAuthorized},
		{"perm privilege", "execution error: A privilege violation occurred. (-10004)", ErrNotAuthorized},
		{"perm text", "Music got an error: doesn't have permission", ErrNotAuthorized},
		{"track empty whose", "execution error: Music got an error: Can’t make some data into the expected type. (-1700)", ErrTrackNotFound},
		{"track cant get", "execution error: Music got an error: Can’t get track. (-1728)", ErrTrackNotFound},
		{"app missing", "execution error: Application can’t be found. (-10814)", ErrMusicMissing},
		{"app proc", "execution error: An error of type -600 has occurred. (-600)", ErrMusicMissing},
		{"unknown keeps fallback", "execution error: something weird (-9999)", fallback},
		{"empty keeps fallback", "", fallback},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := classifyOsascriptErr(c.stderr, fallback)
			if !errors.Is(got, c.want) {
				t.Fatalf("classifyOsascriptErr(%q) = %v, want errors.Is(..%v)", c.stderr, got, c.want)
			}
		})
	}
}

func TestIsHexID(t *testing.T) {
	for _, s := range []string{"A1B2C3D4E5F60718", "0123456789abcdef"} {
		if !isHexID(s) {
			t.Errorf("isHexID(%q) = false, want true", s)
		}
	}
	for _, s := range []string{"", "ghij", `"; tell application "System Events"`, "12 34"} {
		if isHexID(s) {
			t.Errorf("isHexID(%q) = true, want false", s)
		}
	}
}
