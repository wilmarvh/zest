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

package tui

import (
	"hash/fnv"
	"math"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// The visualizer is the signature element: a spectrum that dances while a
// track plays. We can't read real PCM out of Music.app over osascript, so the
// bars are simulated — but seeded by the track and its position and eased
// frame-to-frame, so the motion reads like a real EQ rather than noise.

// vizGlyphs are the eight block heights, tallest last. A bar's height picks
// an index here; height 0 renders as a baseline dot so the floor stays alive.
var vizGlyphs = []rune{'▁', '▂', '▃', '▄', '▅', '▆', '▇', '█'}

const vizLevels = 8 // == len(vizGlyphs)

// visualizer holds one float per bar, eased toward a fresh target each frame.
type visualizer struct {
	heights []float64 // current animated height per bar, 0..1
	phase   float64   // advances every frame; drives the simulated motion
}

func (v *visualizer) resize(bars int) {
	if bars < 1 {
		bars = 1
	}
	if len(v.heights) == bars {
		return
	}
	next := make([]float64, bars)
	copy(next, v.heights)
	v.heights = next
}

// step advances the animation one frame.
//
//   - playing: each bar chases a target derived from a few summed sines whose
//     frequencies differ per-bar, modulated by an energy envelope seeded from
//     the track + coarse position. Fast attack, slow release — like a real
//     meter catching transients then falling back.
//   - paused: bars settle toward a low shimmer.
//   - stopped: bars fall to the floor.
func (v *visualizer) step(state, seed string, pos float64) {
	v.phase += 0.28
	energy := trackEnergy(seed, pos)

	for i := range v.heights {
		var target float64
		switch state {
		case "playing":
			fi := float64(i)
			// Layered sines at incommensurate rates give a non-repeating,
			// across-the-row ripple; bias gives each bar a stable character.
			s := 0.5 +
				0.30*math.Sin(v.phase*1.7+fi*0.55)+
				0.18*math.Sin(v.phase*0.9+fi*1.30)+
				0.12*math.Sin(v.phase*2.6+fi*0.20)
			bias := 0.6 + 0.4*math.Sin(fi*2.399) // golden-ish, avoids banding
			target = clamp01(s * bias * energy)
		case "paused":
			target = 0.06 + 0.05*math.Sin(v.phase*0.6+float64(i)*0.5)
		default: // stopped
			target = 0
		}

		// Asymmetric easing: snap up to transients, ease down gently.
		cur := v.heights[i]
		if target > cur {
			cur += (target - cur) * 0.6
		} else {
			cur += (target - cur) * 0.25
		}
		v.heights[i] = clamp01(cur)
	}
}

// render draws the bar row. When idle (all bars near the floor) it falls back
// to a quiet hairline so a stopped player doesn't show a dead row of dots.
//
// render takes a value receiver on purpose: View runs on a copy of the model,
// so mutating here (e.g. resizing) would be lost. Sizing is owned by step via
// resize() on the live model; render only reads.
func (v visualizer) render(width int) string {
	if width < 1 || len(v.heights) == 0 {
		return ""
	}

	live := false
	for _, h := range v.heights {
		if h > 0.08 {
			live = true
			break
		}
	}
	if !live {
		return vizIdleStyle.Render(strings.Repeat("─", width))
	}

	var b strings.Builder
	for _, h := range v.heights {
		lvl := int(h*float64(vizLevels-1) + 0.5)
		if lvl < 0 {
			lvl = 0
		}
		if lvl > vizLevels-1 {
			lvl = vizLevels - 1
		}
		// Color-grade by height: ember at the floor, amber at the peaks.
		gi := lvl * (len(vizGradient) - 1) / (vizLevels - 1)
		style := lipgloss.NewStyle().Foreground(vizGradient[gi])
		b.WriteString(style.Render(string(vizGlyphs[lvl])))
	}
	return b.String()
}

// trackEnergy returns a 0..1 "loudness" that's stable for a given track but
// drifts slowly across its duration, so different songs feel different and a
// single song breathes as it plays. Seeded so it's deterministic per moment.
func trackEnergy(seed string, pos float64) float64 {
	h := fnv.New32a()
	_, _ = h.Write([]byte(seed))
	base := 0.55 + 0.35*float64(h.Sum32()%1000)/1000.0 // 0.55..0.90 per track
	drift := 0.12 * math.Sin(pos*0.08)                 // slow swell over time
	return clamp01(base + drift)
}

func clamp01(f float64) float64 {
	if f < 0 {
		return 0
	}
	if f > 1 {
		return 1
	}
	return f
}
