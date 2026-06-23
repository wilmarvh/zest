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

import "github.com/charmbracelet/lipgloss"

var (
	// Palette — an ember gradient (the visualizer's voice) over macOS dark grays.
	cRed    = lipgloss.Color("#FA233B") // ember — primary accent, the anchor
	cCoral  = lipgloss.Color("#FF6B5C") // gradient mid — warm transients
	cAmber  = lipgloss.Color("#FFB347") // gradient peak — hot tops & star ratings
	cWhite  = lipgloss.Color("#F5F5F7")
	cText   = lipgloss.Color("#D1D1D6")
	cGray   = lipgloss.Color("#98989D")
	cDim    = lipgloss.Color("#636366")
	cFaint  = lipgloss.Color("#48484A")
	cSelBg  = lipgloss.Color("#FA233B")
	cMuteBg = lipgloss.Color("#3A3A3C")
	cBorder = lipgloss.Color("#3A3A3C")

	// Visualizer gradient, low band → hot top. Indexed by bar height.
	vizGradient = []lipgloss.Color{cRed, cRed, cCoral, cCoral, cAmber}
	starStyle   = lipgloss.NewStyle().Foreground(cAmber)

	// Header bar
	headerBarStyle   = lipgloss.NewStyle().Padding(0, 1)
	logoStyle        = lipgloss.NewStyle().Foreground(cRed).Bold(true)
	headerTitleStyle = lipgloss.NewStyle().Foreground(cWhite).Bold(true)
	headerMetaStyle  = lipgloss.NewStyle().Foreground(cDim)

	// Panes
	paneStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(cBorder).
			Padding(0, 1)
	paneFocusStyle = paneStyle.BorderForeground(cRed)

	// List chrome
	listTitleStyle = lipgloss.NewStyle().Foreground(cGray).Bold(true)
	listTitleBar   = lipgloss.NewStyle().Padding(0, 0, 1, 0)
	paginatorStyle = lipgloss.NewStyle().Foreground(cFaint)

	// Rows
	rowStyle        = lipgloss.NewStyle().Foreground(cText)
	dimStyle        = lipgloss.NewStyle().Foreground(cDim)
	selRowStyle     = lipgloss.NewStyle().Foreground(cWhite).Background(cSelBg).Bold(true)
	mutedSelStyle   = lipgloss.NewStyle().Foreground(cWhite).Background(cMuteBg)
	playingRowStyle = lipgloss.NewStyle().Foreground(cRed)

	// Now-playing stage
	npBarStyle    = lipgloss.NewStyle().Padding(0, 1)
	npTrackStyle  = lipgloss.NewStyle().Foreground(cWhite).Bold(true)
	npArtistStyle = lipgloss.NewStyle().Foreground(cGray)
	npTimeStyle   = lipgloss.NewStyle().Foreground(cDim)
	barFillStyle  = lipgloss.NewStyle().Foreground(cRed)
	barEmptyStyle = lipgloss.NewStyle().Foreground(cFaint)
	vizIdleStyle  = lipgloss.NewStyle().Foreground(cFaint)

	// Bottom bar
	helpStyle         = lipgloss.NewStyle().Foreground(cDim).Padding(0, 1)
	searchBarStyle    = lipgloss.NewStyle().Padding(0, 1)
	searchPromptStyle = lipgloss.NewStyle().Foreground(cRed).Bold(true)

	spinnerStyle = lipgloss.NewStyle().Foreground(cRed)
	errorStyle   = lipgloss.NewStyle().Foreground(cRed).Bold(true)
)
