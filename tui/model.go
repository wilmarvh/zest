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
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/paginator"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/reflow/truncate"

	"github.com/wilmarvh/zest/music"
)

// ---------------------------------------------------------------------------
// Pane focus / content level
// ---------------------------------------------------------------------------

type Pane int

const (
	SidebarPane Pane = iota
	ContentPane
)

type ContentLevel int

const (
	LevelAlbums ContentLevel = iota
	LevelTracks
)

// ---------------------------------------------------------------------------
// List item types
// ---------------------------------------------------------------------------

type artistItem struct{ artist music.Artist }

func (i artistItem) FilterValue() string { return i.artist.Name }

type albumItem struct{ album music.Album }

func (i albumItem) FilterValue() string { return i.album.Name }

type trackItem struct{ track music.Track }

func (i trackItem) FilterValue() string { return i.track.Name }

// playAlbumItem is the "▶ Play album" action row that sits at the top of the
// track view. Selecting it queues the whole album (shuffle off, in order).
type playAlbumItem struct{ ids []string }

func (i playAlbumItem) FilterValue() string { return "" }

// ---------------------------------------------------------------------------
// Row delegate
// ---------------------------------------------------------------------------

type rowDelegate struct {
	focused   bool
	playingID string
}

func (d rowDelegate) Height() int                               { return 1 }
func (d rowDelegate) Spacing() int                              { return 0 }
func (d rowDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd { return nil }

func (d rowDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	width := m.Width()
	if width <= 0 {
		return
	}

	var left, right, stars string
	playing := false
	switch v := listItem.(type) {
	case playAlbumItem:
		left = "▶ Play album"
		right = fmt.Sprintf("%d ♪", len(v.ids))
	case artistItem:
		left = v.artist.Name
		n := 0
		for _, al := range v.artist.Albums {
			n += len(al.Tracks)
		}
		right = fmt.Sprintf("%d", n)
	case albumItem:
		left = v.album.Name
		right = fmt.Sprintf("%d ♪", len(v.album.Tracks))
	case trackItem:
		if v.track.TrackNumber > 0 {
			left = fmt.Sprintf("%2d  %s", v.track.TrackNumber, v.track.Name)
		} else {
			left = v.track.Name
		}
		stars = ratingStars(v.track.Rating)
		right = v.track.DurationString()
		if stars != "" {
			right = stars + "  " + right // stars sit just left of the duration
		}
		playing = d.playingID != "" && d.playingID == v.track.PersistentID
	default:
		return
	}

	prefix := "  "
	if playing {
		prefix = "♪ "
	}

	rightW := lipgloss.Width(right)
	maxLeft := width - 2 - rightW - 2
	if maxLeft < 4 {
		right = ""
		rightW = 0
		maxLeft = width - 2
	}
	if maxLeft < 1 {
		maxLeft = 1
	}
	left = truncate.StringWithTail(left, uint(maxLeft), "…")

	gap := width - 2 - lipgloss.Width(left) - rightW
	if gap < 0 {
		gap = 0
	}
	pad := strings.Repeat(" ", gap)

	isSel := index == m.Index()
	switch {
	case isSel && d.focused:
		fmt.Fprint(w, selRowStyle.Render(prefix+left+pad+right))
	case isSel:
		fmt.Fprint(w, mutedSelStyle.Render(prefix+left+pad+right))
	case playing:
		fmt.Fprint(w, playingRowStyle.Render(prefix+left+pad+right))
	default:
		// Stars (if any) glow amber; the duration stays dim.
		rightStyled := dimStyle.Render(right)
		if stars != "" {
			dur := strings.TrimPrefix(right, stars)
			rightStyled = starStyle.Render(stars) + dimStyle.Render(dur)
		}
		fmt.Fprint(w, rowStyle.Render(prefix+left)+pad+rightStyled)
	}
}

// ratingStars renders a 0–100 Music rating as filled amber stars (1 star per
// 20 points). Unrated tracks get nothing — the row stays clean.
func ratingStars(rating int) string {
	n := rating / 20
	if n < 1 {
		return ""
	}
	if n > 5 {
		n = 5
	}
	return strings.Repeat("★", n)
}

// ---------------------------------------------------------------------------
// Messages
// ---------------------------------------------------------------------------

type LibraryLoadedMsg struct{ Library *music.Library }
type LibraryErrorMsg struct{ Err error }
type statusMsg struct{ status music.PlayerStatus }
type playbackErrMsg struct{ err error }
type tickMsg time.Time

// ---------------------------------------------------------------------------
// Model
// ---------------------------------------------------------------------------

type Model struct {
	library   *music.Library
	sidebar   list.Model
	content   list.Model
	focus     Pane
	level     ContentLevel
	search    textinput.Model
	searching bool
	width     int
	height    int
	loading   bool
	spin      spinner.Model
	err       error

	player    music.PlayerStatus
	playingID string
	playErr   error // last playback hiccup, shown in the now-playing bar

	selectedArtist string
	selectedAlbum  string
}

func newList(title string) list.Model {
	l := list.New(nil, rowDelegate{}, 0, 0)
	l.Title = title
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.SetShowHelp(false)
	l.Paginator.Type = paginator.Arabic
	l.Styles.Title = listTitleStyle
	l.Styles.TitleBar = listTitleBar
	l.Styles.PaginationStyle = paginatorStyle
	l.Styles.NoItems = dimStyle
	return l
}

func New() Model {
	si := textinput.New()
	si.Placeholder = "type to filter artists & songs"
	si.Prompt = "/ "
	si.PromptStyle = searchPromptStyle
	si.PlaceholderStyle = dimStyle
	si.CharLimit = 100

	sp := spinner.New(spinner.WithSpinner(spinner.Dot), spinner.WithStyle(spinnerStyle))

	sidebar := newList("Artists")
	content := newList("Albums")

	m := Model{
		sidebar: sidebar,
		content: content,
		focus:   SidebarPane,
		search:  si,
		spin:    sp,
		loading: true,
	}
	m.updateDelegates()
	return m
}

// ---------------------------------------------------------------------------
// Init / commands
// ---------------------------------------------------------------------------

func (m Model) Init() tea.Cmd {
	return tea.Batch(loadLibrary, fetchStatus, tickCmd(), m.spin.Tick)
}

func loadLibrary() tea.Msg {
	lib, err := music.Load()
	if err != nil {
		return LibraryErrorMsg{Err: err}
	}
	return LibraryLoadedMsg{Library: lib}
}

func fetchStatus() tea.Msg {
	s, _ := music.Status()
	return statusMsg{status: s}
}

func tickCmd() tea.Cmd {
	return tea.Tick(2*time.Second, func(t time.Time) tea.Msg { return tickMsg(t) })
}

// playbackCmd runs a player action and reports any error, then refreshes
// status so the now-playing bar reflects what actually happened.
func playbackCmd(action func() error) tea.Cmd {
	return tea.Batch(
		func() tea.Msg { return playbackErrMsg{err: action()} },
		fetchStatus,
	)
}

func playTrackCmd(persistentID string) tea.Cmd {
	return playbackCmd(func() error { return music.PlayTrack(persistentID) })
}

func playAlbumCmd(albumIDs []string, startID string) tea.Cmd {
	return playbackCmd(func() error { return music.PlayAlbumFromTrack(albumIDs, startID) })
}

// ---------------------------------------------------------------------------
// Update
// ---------------------------------------------------------------------------

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.search.Width = max(m.width-8, 10)
		m.updatePaneSizes()
		return m, nil

	case LibraryLoadedMsg:
		m.library = msg.Library
		m.loading = false
		m.populateSidebar(m.library.Artists)
		return m, nil

	case LibraryErrorMsg:
		m.loading = false
		m.err = msg.Err
		return m, nil

	case spinner.TickMsg:
		if !m.loading {
			return m, nil
		}
		var cmd tea.Cmd
		m.spin, cmd = m.spin.Update(msg)
		return m, cmd

	case tickMsg:
		return m, tea.Batch(fetchStatus, tickCmd())

	case playbackErrMsg:
		m.playErr = msg.err
		return m, nil

	case statusMsg:
		if msg.status.State == "playing" || msg.status.State == "paused" {
			m.playErr = nil // music is flowing again, clear any stale hiccup
		}
		changed := m.playingID != msg.status.PersistentID
		m.player = msg.status
		if msg.status.State == "stopped" {
			m.playingID = ""
		} else {
			m.playingID = msg.status.PersistentID
		}
		if changed {
			m.updateDelegates()
		}
		return m, nil

	case tea.KeyMsg:
		if m.searching {
			return m.handleSearchKey(msg)
		}
		return m.handleKey(msg)
	}

	var cmd tea.Cmd
	if m.focus == SidebarPane {
		m.sidebar, cmd = m.sidebar.Update(msg)
	} else {
		m.content, cmd = m.content.Update(msg)
	}
	return m, cmd
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit

	case "/":
		m.searching = true
		m.search.SetValue("")
		return m, m.search.Focus()

	case " ":
		return m, playbackCmd(music.PlayPause)

	case "n":
		return m, playbackCmd(music.NextTrack)

	case "p":
		return m, playbackCmd(music.PreviousTrack)

	case "s":
		return m, playbackCmd(func() error { _, err := music.ToggleShuffle(); return err })

	case "r":
		return m, playbackCmd(func() error { _, err := music.CycleRepeat(); return err })

	case "tab", "right", "l":
		if m.focus == SidebarPane {
			m.focus = ContentPane
			m.updateDelegates()
		}
		return m, nil

	case "shift+tab", "left", "h":
		if m.focus == ContentPane {
			m.focus = SidebarPane
			m.updateDelegates()
		} else if m.level == LevelTracks {
			m.level = LevelAlbums
			m.populateAlbums(m.selectedArtist)
		}
		return m, nil

	case "esc":
		if m.level == LevelTracks {
			m.level = LevelAlbums
			m.populateAlbums(m.selectedArtist)
		} else if m.focus == ContentPane {
			m.focus = SidebarPane
			m.updateDelegates()
		}
		return m, nil

	case "enter":
		if m.focus == SidebarPane {
			sel := m.sidebar.SelectedItem()
			if ai, ok := sel.(artistItem); ok {
				m.selectedArtist = ai.artist.Name
				m.level = LevelAlbums
				m.populateAlbums(ai.artist.Name)
				m.focus = ContentPane
				m.updateDelegates()
			}
			return m, nil
		}
		switch it := m.content.SelectedItem().(type) {
		case albumItem:
			m.selectedAlbum = it.album.Name
			m.level = LevelTracks
			m.populateTracks(m.selectedArtist, it.album.Name)
		case playAlbumItem:
			// Queue the whole album in order (shuffle off).
			if len(it.ids) == 0 {
				return m, nil
			}
			m.playingID = it.ids[0]
			m.updateDelegates()
			return m, playAlbumCmd(it.ids, it.ids[0])
		case trackItem:
			// Single track: play just this one and let Music continue with
			// whatever it would play next.
			m.playingID = it.track.PersistentID
			m.updateDelegates()
			return m, playTrackCmd(it.track.PersistentID)
		}
		return m, nil
	}

	var cmd tea.Cmd
	if m.focus == SidebarPane {
		m.sidebar, cmd = m.sidebar.Update(msg)
	} else {
		m.content, cmd = m.content.Update(msg)
	}
	return m, cmd
}

func (m Model) handleSearchKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.searching = false
		m.search.Blur()
		if m.library != nil {
			m.populateSidebar(m.library.Artists)
		}
		return m, nil

	case "enter":
		// Commit the search and select the highlighted result in one press.
		m.searching = false
		m.search.Blur()
		return m.handleKey(msg)

	case "up", "down":
		// Let the arrows drive the focused list mid-search, so you can filter
		// and navigate to a result without leaving the search box first.
		var cmd tea.Cmd
		if m.focus == SidebarPane {
			m.sidebar, cmd = m.sidebar.Update(msg)
		} else {
			m.content, cmd = m.content.Update(msg)
		}
		return m, cmd
	}

	var cmd tea.Cmd
	m.search, cmd = m.search.Update(msg)
	if m.library != nil {
		query := strings.ToLower(m.search.Value())
		if query == "" {
			m.populateSidebar(m.library.Artists)
		} else {
			m.populateSidebar(filterArtists(m.library.Artists, query))
		}
	}
	return m, cmd
}

// filterArtists keeps artists whose name, any album name, or any track name
// contains the (already lower-cased) query.
func filterArtists(artists []music.Artist, query string) []music.Artist {
	var filtered []music.Artist
	for _, a := range artists {
		if artistMatches(a, query) {
			filtered = append(filtered, a)
		}
	}
	return filtered
}

func artistMatches(a music.Artist, query string) bool {
	if strings.Contains(strings.ToLower(a.Name), query) {
		return true
	}
	for _, al := range a.Albums {
		if strings.Contains(strings.ToLower(al.Name), query) {
			return true
		}
		for _, t := range al.Tracks {
			if strings.Contains(strings.ToLower(t.Name), query) {
				return true
			}
		}
	}
	return false
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func (m Model) paneWidths() (int, int) {
	sw := m.width / 3
	if sw < 24 {
		sw = 24
	}
	if sw > 38 {
		sw = 38
	}
	cw := m.width - sw - 4
	if cw < 20 {
		cw = 20
	}
	return sw, cw
}

func (m *Model) updatePaneSizes() {
	sw, cw := m.paneWidths()
	// header(1) + now-playing(1) + bottom bar(1) + pane borders(2)
	listH := m.height - 6
	if listH < 3 {
		listH = 3
	}
	m.sidebar.SetSize(sw-2, listH)
	m.content.SetSize(cw-2, listH)
}

func (m *Model) updateDelegates() {
	m.sidebar.SetDelegate(rowDelegate{focused: m.focus == SidebarPane, playingID: m.playingID})
	m.content.SetDelegate(rowDelegate{focused: m.focus == ContentPane, playingID: m.playingID})
}

func (m *Model) populateSidebar(artists []music.Artist) {
	items := make([]list.Item, len(artists))
	for i, a := range artists {
		items[i] = artistItem{artist: a}
	}
	m.sidebar.SetItems(items)
}

func (m *Model) populateAlbums(artistName string) {
	if m.library == nil {
		return
	}
	for _, a := range m.library.Artists {
		if a.Name == artistName {
			items := make([]list.Item, len(a.Albums))
			for i, al := range a.Albums {
				items[i] = albumItem{album: al}
			}
			m.content.SetItems(items)
			m.content.Title = artistName
			m.content.ResetSelected()
			return
		}
	}
}

func (m *Model) populateTracks(artistName, albumName string) {
	if m.library == nil {
		return
	}
	for _, a := range m.library.Artists {
		if a.Name != artistName {
			continue
		}
		for _, al := range a.Albums {
			if al.Name == albumName {
				// "▶ Play album" action sits first, above the tracks.
				items := make([]list.Item, 0, len(al.Tracks)+1)
				items = append(items, playAlbumItem{ids: albumTrackIDs(al.Tracks)})
				for _, t := range al.Tracks {
					items = append(items, trackItem{track: t})
				}
				m.content.SetItems(items)
				m.content.Title = artistName + " › " + albumName
				m.content.ResetSelected()
				return
			}
		}
	}
}

// albumTrackIDs pulls the persistent IDs from an album's tracks, in order.
func albumTrackIDs(tracks []music.Track) []string {
	ids := make([]string, len(tracks))
	for i, t := range tracks {
		ids[i] = t.PersistentID
	}
	return ids
}

func fmtTime(sec float64) string {
	if sec < 0 {
		sec = 0
	}
	s := int(sec)
	return fmt.Sprintf("%d:%02d", s/60, s%60)
}

// ---------------------------------------------------------------------------
// View
// ---------------------------------------------------------------------------

func (m Model) View() string {
	if m.width == 0 {
		return ""
	}

	if m.loading {
		msg := m.spin.View() + " " + headerTitleStyle.Render("Loading your library…") +
			"\n\n" + dimStyle.Render("this can take a few seconds")
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, msg)
	}

	if m.err != nil {
		msg := errorStyle.Render("Error: "+m.err.Error()) + "\n\n" + dimStyle.Render("press q to quit")
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, msg)
	}

	sw, cw := m.paneWidths()
	var sPane, cPane string
	if m.focus == SidebarPane {
		sPane = paneFocusStyle.Width(sw).Render(m.sidebar.View())
		cPane = paneStyle.Width(cw).Render(m.content.View())
	} else {
		sPane = paneStyle.Width(sw).Render(m.sidebar.View())
		cPane = paneFocusStyle.Width(cw).Render(m.content.View())
	}
	main := lipgloss.JoinHorizontal(lipgloss.Top, sPane, cPane)

	return strings.Join([]string{m.headerView(), main, m.nowPlayingView(), m.bottomView()}, "\n")
}

func (m Model) headerView() string {
	// Wordmark: a small equalizer glyph + spaced letters for presence.
	left := logoStyle.Render("▰▰▱") + " " + headerTitleStyle.Render("Z E S T")
	right := ""
	if m.library != nil {
		right = headerMetaStyle.Render(fmt.Sprintf("%d songs · %d artists", len(m.library.Tracks), len(m.library.Artists)))
	}
	gap := m.width - 2 - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 1 {
		gap = 1
	}
	return headerBarStyle.Render(left + strings.Repeat(" ", gap) + right)
}

// playErrMessage turns a playback error into a one-line hint that points at
// the actual fix. The permission case is the common one — the macOS Automation
// prompt has to be accepted once before zest can drive Music at all.
func playErrMessage(err error) string {
	switch {
	case errors.Is(err, music.ErrNotAuthorized):
		return "grant permission: System Settings ▸ Privacy & Security ▸ Automation ▸ allow this terminal to control Music"
	case errors.Is(err, music.ErrMusicMissing):
		return "couldn't launch Music.app — is it installed?"
	case errors.Is(err, music.ErrTrackNotFound):
		return "can't play this track — it's not in Music's playable library (cloud-only track?)"
	default:
		return "playback failed — couldn't reach Music.app"
	}
}

// nowPlayingView renders the transport line:
// icon · track — artist … shuffle repeat position bar duration.
func (m Model) nowPlayingView() string {
	inner := m.width - 2
	if inner < 1 {
		inner = 1
	}
	st := m.player

	if st.State != "playing" && st.State != "paused" {
		var line string
		if m.playErr != nil {
			line = errorStyle.Render("◼ " + playErrMessage(m.playErr))
		} else {
			line = dimStyle.Render("◼ nothing playing — press ↵ on a track")
		}
		return npBarStyle.Render(line)
	}

	icon := barFillStyle.Render("▶")
	if st.State == "paused" {
		icon = dimStyle.Render("❚❚")
	}

	cur, tot := fmtTime(st.Position), fmtTime(st.Duration)
	timesW := lipgloss.Width(cur) + lipgloss.Width(tot) + 2

	barW := inner / 4
	if barW < 10 {
		barW = 10
	}
	if barW > 40 {
		barW = 40
	}

	metaAvail := inner - 2 - timesW - barW - 2
	if metaAvail < 8 {
		metaAvail = 8
	}
	track := truncate.StringWithTail(st.Track, uint(metaAvail), "…")
	meta := npTrackStyle.Render(track)
	rem := metaAvail - lipgloss.Width(track)
	if st.Artist != "" && rem > 8 {
		meta += npArtistStyle.Render(" — " + truncate.StringWithTail(st.Artist, uint(rem-3), "…"))
	}

	modes := modeIndicators(st.Shuffle, st.Repeat)
	right := modes + npTimeStyle.Render(cur) + " " + renderBar(st.Position, st.Duration, barW) + " " + npTimeStyle.Render(tot)
	gap := inner - 2 - lipgloss.Width(meta) - lipgloss.Width(right)
	if gap < 1 {
		gap = 1
	}
	return npBarStyle.Render(icon + " " + meta + strings.Repeat(" ", gap) + right)
}

// modeIndicators renders shuffle/repeat glyphs, bright when on and dim when
// off, so their state is always visible in the now-playing bar. Returns a
// trailing space when anything is shown so it sits flush before the timer.
func modeIndicators(shuffle bool, repeat string) string {
	var b strings.Builder
	if shuffle {
		b.WriteString(barFillStyle.Render("⇄"))
	} else {
		b.WriteString(dimStyle.Render("⇄"))
	}
	b.WriteString(" ")
	switch repeat {
	case "one":
		b.WriteString(barFillStyle.Render("↻¹"))
	case "all":
		b.WriteString(barFillStyle.Render("↻"))
	default:
		b.WriteString(dimStyle.Render("↻"))
	}
	b.WriteString("  ")
	return b.String()
}

func renderBar(pos, dur float64, w int) string {
	if w < 1 {
		return ""
	}
	frac := 0.0
	if dur > 0 {
		frac = pos / dur
	}
	if frac < 0 {
		frac = 0
	}
	if frac > 1 {
		frac = 1
	}
	filled := int(frac*float64(w) + 0.5)
	if filled > w {
		filled = w
	}
	return barFillStyle.Render(strings.Repeat("━", filled)) + barEmptyStyle.Render(strings.Repeat("─", w-filled))
}

func (m Model) bottomView() string {
	if m.searching {
		return searchBarStyle.Render(m.search.View())
	}
	help := "↑↓/jk move · ⇥ pane · ↵ select · space pause · n/p skip · s shuffle · r repeat · / search · esc back · q quit"
	return helpStyle.Render(truncate.String(help, uint(max(m.width-2, 10))))
}
