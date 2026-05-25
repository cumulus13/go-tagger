package tagger_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/cumulus13/go-tagger/internal/tagger"
)

// ── Helpers ───────────────────────────────────────────────────────────────────

// buildMinimalMP3 writes a minimal ID3v2.3 tagged file.
func buildMinimalMP3(t *testing.T, title, artist, album string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "test.mp3")

	mkFrame := func(id, text string) []byte {
		payload := append([]byte{0x03}, []byte(text)...)
		payload = append(payload, 0x00)
		b := make([]byte, 10+len(payload))
		copy(b[0:4], id)
		b[4] = 0; b[5] = 0
		b[6] = byte(len(payload) >> 8)
		b[7] = byte(len(payload))
		copy(b[10:], payload)
		return b
	}

	var frames []byte
	if title != "" {
		frames = append(frames, mkFrame("TIT2", title)...)
	}
	if artist != "" {
		frames = append(frames, mkFrame("TPE1", artist)...)
	}
	if album != "" {
		frames = append(frames, mkFrame("TALB", album)...)
	}
	frames = append(frames, mkFrame("TRCK", "01/10")...)

	const padding = 512
	bodySize := len(frames) + padding
	hdr := make([]byte, 10)
	copy(hdr[0:3], "ID3")
	hdr[3] = 3
	hdr[6] = byte((bodySize >> 21) & 0x7F)
	hdr[7] = byte((bodySize >> 14) & 0x7F)
	hdr[8] = byte((bodySize >> 7) & 0x7F)
	hdr[9] = byte(bodySize & 0x7F)

	var data []byte
	data = append(data, hdr...)
	data = append(data, frames...)
	data = append(data, make([]byte, padding)...)

	os.WriteFile(path, data, 0o644)
	return path
}

// ── TrackInfo parsing ─────────────────────────────────────────────────────────

func TestParseTitleLine(t *testing.T) {
	tests := []struct {
		input      string
		wantTrack  string
		wantTitle  string
	}{
		{"01. Song Title", "01", "Song Title"},
		{"1. song title", "1", "Song Title"},
		{"12. Another Song.mp3", "12", "Another Song"},
		{"Plain Title", "", "Plain Title"},
		{"plain title.mp3", "", "Plain Title"},
		{"  03. Spaced Title  ", "03", "Spaced Title"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := tagger.ParseTitleLine(tt.input)
			if got.Track != tt.wantTrack {
				t.Errorf("ParseTitleLine(%q).Track = %q, want %q", tt.input, got.Track, tt.wantTrack)
			}
			if got.Title != tt.wantTitle {
				t.Errorf("ParseTitleLine(%q).Title = %q, want %q", tt.input, got.Title, tt.wantTitle)
			}
		})
	}
}

// ── Track formatting ──────────────────────────────────────────────────────────

func TestFormatTrack(t *testing.T) {
	tests := []struct {
		track  string
		total  int
		change bool
		want   string
	}{
		{"1", 10, false, "01/10"},
		{"3", 12, false, "03/12"},
		{"01/10", 10, false, "01/10"},
		{"1/10", 10, false, "01/10"},
		{"01/10", 15, true, "01/15"},
		{"5", 100, false, "05/100"},
		{"1", 1, false, "01/01"},
	}

	for _, tt := range tests {
		t.Run(tt.track, func(t *testing.T) {
			got := tagger.FormatTrack(tt.track, tt.total, tt.change)
			if got != tt.want {
				t.Errorf("FormatTrack(%q,%d,%v) = %q, want %q",
					tt.track, tt.total, tt.change, got, tt.want)
			}
		})
	}
}

func TestFormatDisc(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"", "01/01"},
		{"1", "01/01"},
		{"2", "02/01"},
		{"01/02", "01/02"},
		{"2/3", "02/03"},
		{"1/1", "01/01"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := tagger.FormatDisc(tt.input)
			if got != tt.want {
				t.Errorf("FormatDisc(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// ── ScanDir ───────────────────────────────────────────────────────────────────

func TestScanDir(t *testing.T) {
	dir := t.TempDir()
	// Create test files
	for _, name := range []string{"01. First.mp3", "02. Second.MP3", "cover.jpg", "notes.txt"} {
		os.WriteFile(filepath.Join(dir, name), []byte{}, 0o644)
	}

	files, err := tagger.ScanDir(dir)
	if err != nil {
		t.Fatalf("ScanDir: %v", err)
	}
	if len(files) != 2 {
		t.Errorf("ScanDir found %d MP3 files, want 2; got: %v", len(files), files)
	}
}

func TestScanDirNotExist(t *testing.T) {
	_, err := tagger.ScanDir("/nonexistent/path/xyz")
	if err == nil {
		t.Error("ScanDir on non-existent path should return error")
	}
}

// ── GetAll ────────────────────────────────────────────────────────────────────

func TestGetAll(t *testing.T) {
	path := buildMinimalMP3(t, "My Song", "My Artist", "My Album")
	tags, err := tagger.GetAll(path)
	if err != nil {
		t.Fatalf("GetAll: %v", err)
	}
	if tags["title"] != "My Song" {
		t.Errorf("title = %q", tags["title"])
	}
	if tags["artist"] != "My Artist" {
		t.Errorf("artist = %q", tags["artist"])
	}
	if tags["album"] != "My Album" {
		t.Errorf("album = %q", tags["album"])
	}
}

// ── Set ───────────────────────────────────────────────────────────────────────

func TestSetBasicFields(t *testing.T) {
	path := buildMinimalMP3(t, "Original", "Artist", "Album")

	ts := tagger.TagSet{
		Title:  "Updated Title",
		Artist: "Updated Artist",
		Album:  "Updated Album",
		Genre:  "Jazz",
		Date:   "2024",
		Track:  "03/10",
	}

	if err := tagger.Set(path, ts); err != nil {
		t.Fatalf("Set: %v", err)
	}

	tags, err := tagger.GetAll(path)
	if err != nil {
		t.Fatalf("GetAll after Set: %v", err)
	}

	checks := map[string]string{
		"title":  "Updated Title",
		"artist": "Updated Artist",
		"album":  "Updated Album",
		"genre":  "Jazz",
		"date":   "2024",
		"track":  "03/10",
	}
	for field, want := range checks {
		if tags[field] != want {
			t.Errorf("%s = %q, want %q", field, tags[field], want)
		}
	}
}

func TestSetTestMode(t *testing.T) {
	path := buildMinimalMP3(t, "Original", "Artist", "Album")

	// Record file mod time before
	fi, _ := os.Stat(path)
	beforeMod := fi.ModTime()

	ts := tagger.TagSet{
		Title:    "Should Not Save",
		TestMode: true,
	}
	tagger.Set(path, ts)

	// File should not have been modified
	fi2, _ := os.Stat(path)
	if !fi2.ModTime().Equal(beforeMod) {
		t.Error("TestMode should not modify the file")
	}

	// Title should still be original
	tags, _ := tagger.GetAll(path)
	if tags["title"] != "Original" {
		t.Errorf("TestMode changed title to %q", tags["title"])
	}
}

func TestSetClear(t *testing.T) {
	path := buildMinimalMP3(t, "To Clear", "Artist", "Album")

	ts := tagger.TagSet{Title: "clear"}
	tagger.Set(path, ts)

	tags, _ := tagger.GetAll(path)
	if tags["title"] != "" {
		t.Errorf("After clear, title = %q, want empty", tags["title"])
	}
}

func TestSetExtendedFields(t *testing.T) {
	path := buildMinimalMP3(t, "Song", "Artist", "Album")

	ts := tagger.TagSet{
		ISRC:        "USRC17607839",
		BPM:         "128",
		Copyright:   "2024 Test Co",
		EncodedBy:   "TestEncoder",
		Barcode:     "9781234567897",
		Comment:     "Test comment",
		URL:         "https://example.com",
		Publisher:   "Test Publisher",
		Group:       "Test Group",
		AlbumArtist: "Test Album Artist",
		OrigArtist:  "Original Artist",
		Composer:    "A Composer",
		Subtitle:    "A Subtitle",
		Remix:       "DJ Remix",
	}

	if err := tagger.Set(path, ts); err != nil {
		t.Fatalf("Set extended: %v", err)
	}

	tags, _ := tagger.GetAll(path)

	checkField := func(field, want string) {
		t.Helper()
		if tags[field] != want {
			t.Errorf("%s = %q, want %q", field, tags[field], want)
		}
	}

	checkField("isrc", "USRC17607839")
	checkField("bpm", "128")
	checkField("copyright", "2024 Test Co")
	checkField("encodedby", "TestEncoder")
	checkField("barcode", "9781234567897")
	checkField("comment", "Test comment")
	checkField("url", "https://example.com")
	checkField("publisher", "Test Publisher")
	checkField("group", "Test Group")
	checkField("album_artist", "Test Album Artist")
	checkField("original_artist", "Original Artist")
	checkField("composer", "A Composer")
	checkField("subtitle", "A Subtitle")
	checkField("remix", "DJ Remix")
}

// ── LicFace presets ───────────────────────────────────────────────────────────

func TestApplyLicFacePreset0(t *testing.T) {
	ts := &tagger.TagSet{Artist: "Test Artist"}
	tagger.ApplyLicFace(ts, 0, "2024")

	if ts.Comment == "" {
		t.Error("Preset 0 should set comment")
	}
	if ts.EncodedBy == "" {
		t.Error("Preset 0 should set encodedby")
	}
	if ts.OrigArtist != "Test Artist" {
		t.Errorf("Preset 0 should set original_artist = artist, got %q", ts.OrigArtist)
	}
	if ts.Group != "Test Artist" {
		t.Errorf("Preset 0 should set group = artist, got %q", ts.Group)
	}
	if ts.Publisher != "" {
		t.Errorf("Preset 0 should NOT set publisher, got %q", ts.Publisher)
	}
	if ts.Date != "2024" {
		t.Errorf("Preset 0 should set date, got %q", ts.Date)
	}
}

func TestApplyLicFacePreset2(t *testing.T) {
	ts := &tagger.TagSet{Artist: "Test Artist"}
	tagger.ApplyLicFace(ts, 2, "2024")

	if ts.Publisher == "" {
		t.Error("Preset 2 should set publisher")
	}
	if ts.AlbumArtist != "Test Artist" {
		t.Errorf("Preset 2 should set album_artist = artist, got %q", ts.AlbumArtist)
	}
	if ts.Composer != "Test Artist" {
		t.Errorf("Preset 2 should set composer = artist, got %q", ts.Composer)
	}
	if ts.URL == "" {
		t.Error("Preset 2 should set URL")
	}
}

func TestApplyLicFacePreset3(t *testing.T) {
	ts := &tagger.TagSet{Artist: "Test Artist"}
	tagger.ApplyLicFace(ts, 3, "2024")

	if ts.ISRC != "clear" {
		t.Errorf("Preset 3 should clear ISRC, got %q", ts.ISRC)
	}
	if ts.Date != "" {
		t.Errorf("Preset 3 should NOT set date, got %q", ts.Date)
	}
}

// ── Consistency check ─────────────────────────────────────────────────────────

func TestCheckConsistencySingle(t *testing.T) {
	p1 := buildMinimalMP3(t, "Song A", "Same Artist", "Same Album")
	p2 := buildMinimalMP3(t, "Song B", "Same Artist", "Same Album")
	// Move both to same dir
	dir := filepath.Dir(p1)
	np2 := filepath.Join(dir, "02.mp3")
	os.Rename(p2, np2)

	report := tagger.CheckConsistency([]string{p1, np2})
	if len(report.Artists) != 1 {
		t.Errorf("Expected 1 unique artist, got %d: %v", len(report.Artists), report.Artists)
	}
}

func TestCheckConsistencyMultiple(t *testing.T) {
	p1 := buildMinimalMP3(t, "Song A", "Artist One", "Album")
	p2 := buildMinimalMP3(t, "Song B", "Artist Two", "Album")
	dir := filepath.Dir(p1)
	np2 := filepath.Join(dir, "02.mp3")
	os.Rename(p2, np2)

	report := tagger.CheckConsistency([]string{p1, np2})
	if len(report.Artists) != 2 {
		t.Errorf("Expected 2 unique artists, got %d: %v", len(report.Artists), report.Artists)
	}
}

// ── Rename ────────────────────────────────────────────────────────────────────

func TestRenameByTitleTestMode(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "old_name.mp3")
	os.WriteFile(path, []byte{}, 0o644)

	newPath, err := tagger.RenameByTitle(path, "03/12", "New Song Title", true)
	if err != nil {
		t.Fatalf("RenameByTitle test mode: %v", err)
	}
	// File should still exist at old path
	if _, err := os.Stat(path); err != nil {
		t.Error("TestMode should not rename the file")
	}
	if filepath.Base(newPath) != "03. New Song Title.mp3" {
		t.Errorf("newPath = %q, want %q", filepath.Base(newPath), "03. New Song Title.mp3")
	}
}

func TestRenameByTitle(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "old_name.mp3")
	os.WriteFile(path, []byte{0xFF, 0xFB}, 0o644)

	newPath, err := tagger.RenameByTitle(path, "05/12", "My Renamed Song", false)
	if err != nil {
		t.Fatalf("RenameByTitle: %v", err)
	}
	if _, err := os.Stat(newPath); err != nil {
		t.Errorf("Renamed file not found at %q: %v", newPath, err)
	}
	if _, err := os.Stat(path); err == nil {
		t.Error("Old file should no longer exist")
	}
}

func TestRenameByFilePattern(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "03 - Cool Track.mp3")
	os.WriteFile(path, []byte{}, 0o644)

	newPath, err := tagger.RenameByFile(path, "-", true) // test mode
	if err != nil {
		t.Fatalf("RenameByFile: %v", err)
	}
	if filepath.Base(newPath) != "03. Cool Track.mp3" {
		t.Errorf("newPath = %q, want %q", filepath.Base(newPath), "03. Cool Track.mp3")
	}
}

func TestRenameByFileDefault(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "07. Great Song.mp3")
	os.WriteFile(path, []byte{}, 0o644)

	newPath, err := tagger.RenameByFile(path, "", true)
	if err != nil {
		t.Fatalf("RenameByFile default: %v", err)
	}
	if filepath.Base(newPath) != "07. Great Song.mp3" {
		t.Errorf("newPath base = %q", filepath.Base(newPath))
	}
}
