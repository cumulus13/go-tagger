package id3_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/cumulus13/go-tagger/internal/id3"
)

// createMinimalMP3 creates a temporary file with a minimal valid ID3v2.3 header
// but no actual audio frames — sufficient for tag read/write tests.
func createMinimalMP3(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "test.mp3")

	// Build a minimal ID3v2.3 tag with a title frame "Test Song"
	title := []byte{
		0x03,                                    // UTF-8 encoding
		'T', 'e', 's', 't', ' ', 'S', 'o', 'n', 'g', 0x00, // text + null
	}
	frameHdr := func(id string, size int) []byte {
		b := make([]byte, 10)
		copy(b[0:4], id)
		b[4] = byte(size >> 24)
		b[5] = byte(size >> 16)
		b[6] = byte(size >> 8)
		b[7] = byte(size)
		// flags = 0x00 0x00
		return b
	}

	titleFrame := append(frameHdr("TIT2", len(title)), title...)
	bodySize := len(titleFrame) + 256 // + padding

	// ID3v2.3 header
	hdr := make([]byte, 10)
	copy(hdr[0:3], "ID3")
	hdr[3] = 3 // version 2.3
	// syncsafe encode bodySize
	hdr[6] = byte((bodySize >> 21) & 0x7F)
	hdr[7] = byte((bodySize >> 14) & 0x7F)
	hdr[8] = byte((bodySize >> 7) & 0x7F)
	hdr[9] = byte(bodySize & 0x7F)

	var data []byte
	data = append(data, hdr...)
	data = append(data, titleFrame...)
	data = append(data, make([]byte, 256)...) // padding

	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("create test mp3: %v", err)
	}
	return path
}

func TestOpenEmpty(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "empty.mp3")
	// Write a file with no ID3 tag (just some bytes)
	if err := os.WriteFile(path, []byte{0xFF, 0xFB, 0x90, 0x00}, 0o644); err != nil {
		t.Fatal(err)
	}
	tag, err := id3.Open(path)
	if err != nil {
		t.Fatalf("Open empty MP3: %v", err)
	}
	if tag == nil {
		t.Fatal("expected non-nil tag")
	}
	if tag.Title() != "" {
		t.Errorf("expected empty title, got %q", tag.Title())
	}
}

func TestOpenAndReadTitle(t *testing.T) {
	path := createMinimalMP3(t)
	tag, err := id3.Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	title := tag.Title()
	if title != "Test Song" {
		t.Errorf("Title() = %q, want %q", title, "Test Song")
	}
}

func TestSetAndSaveTextFrames(t *testing.T) {
	path := createMinimalMP3(t)

	tag, err := id3.Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}

	tag.SetTitle("New Title")
	tag.SetArtist("Test Artist")
	tag.SetAlbum("Test Album")
	tag.SetGenre("Electronic")
	tag.SetYear("2024")
	tag.SetTrack("05/12")
	tag.SetText("TPOS", "01/01")
	tag.SetText("TCOP", "2024 Test")
	tag.SetText("TPUB", "Test Publisher")

	if err := tag.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// Re-read and verify
	tag2, err := id3.Open(path)
	if err != nil {
		t.Fatalf("Open after save: %v", err)
	}

	checks := map[string]string{
		"title":  tag2.Title(),
		"artist": tag2.Artist(),
		"album":  tag2.Album(),
		"genre":  tag2.Genre(),
		"year":   tag2.Year(),
		"track":  tag2.Track(),
	}
	want := map[string]string{
		"title":  "New Title",
		"artist": "Test Artist",
		"album":  "Test Album",
		"genre":  "Electronic",
		"year":   "2024",
		"track":  "05/12",
	}
	for field, got := range checks {
		if got != want[field] {
			t.Errorf("After save, %s = %q, want %q", field, got, want[field])
		}
	}

	disc := tag2.GetText("TPOS")
	if disc != "01/01" {
		t.Errorf("TPOS = %q, want %q", disc, "01/01")
	}
}

func TestTXXX(t *testing.T) {
	path := createMinimalMP3(t)
	tag, err := id3.Open(path)
	if err != nil {
		t.Fatal(err)
	}

	tag.SetTXXX("BARCODE", "1234567890123")
	tag.SetTXXX("ARTIST", "Solo Artist")
	tag.SetTXXX("_cover", "Cover Album Front")

	if err := tag.Save(); err != nil {
		t.Fatal(err)
	}

	tag2, err := id3.Open(path)
	if err != nil {
		t.Fatal(err)
	}

	if got := tag2.GetTXXX("BARCODE"); got != "1234567890123" {
		t.Errorf("TXXX:BARCODE = %q, want %q", got, "1234567890123")
	}
	if got := tag2.GetTXXX("ARTIST"); got != "Solo Artist" {
		t.Errorf("TXXX:ARTIST = %q, want %q", got, "Solo Artist")
	}
	if got := tag2.GetTXXX("_cover"); got != "Cover Album Front" {
		t.Errorf("TXXX:_cover = %q, want %q", got, "Cover Album Front")
	}
}

func TestTXXXCaseInsensitive(t *testing.T) {
	path := createMinimalMP3(t)
	tag, _ := id3.Open(path)
	tag.SetTXXX("barcode", "999")
	tag.Save()

	tag2, _ := id3.Open(path)
	if got := tag2.GetTXXX("BARCODE"); got != "999" {
		t.Errorf("TXXX lookup should be case-insensitive, got %q", got)
	}
}

func TestComment(t *testing.T) {
	path := createMinimalMP3(t)
	tag, _ := id3.Open(path)
	tag.SetComment("eng", "MyDesc", "This is a comment")
	tag.Save()

	tag2, _ := id3.Open(path)
	desc, text := tag2.GetComment()
	if text != "This is a comment" {
		t.Errorf("GetComment text = %q, want %q", text, "This is a comment")
	}
	if desc != "MyDesc" {
		t.Errorf("GetComment desc = %q, want %q", desc, "MyDesc")
	}
}

func TestLyric(t *testing.T) {
	path := createMinimalMP3(t)
	tag, _ := id3.Open(path)
	tag.SetLyric("xxx", "", "Verse 1\nVerse 2\n")
	tag.Save()

	tag2, _ := id3.Open(path)
	lyric := tag2.GetLyric()
	if lyric != "Verse 1\nVerse 2\n" {
		t.Errorf("GetLyric = %q, want %q", lyric, "Verse 1\nVerse 2\n")
	}
}

func TestURL(t *testing.T) {
	path := createMinimalMP3(t)
	tag, _ := id3.Open(path)
	tag.SetURL("", "https://example.com")
	tag.Save()

	tag2, _ := id3.Open(path)
	url := tag2.GetURL()
	if url != "https://example.com" {
		t.Errorf("GetURL = %q, want %q", url, "https://example.com")
	}
}

func TestDeleteFrame(t *testing.T) {
	path := createMinimalMP3(t)
	tag, _ := id3.Open(path)
	tag.SetTitle("Will Be Deleted")
	tag.Save()

	tag2, _ := id3.Open(path)
	tag2.DeleteFrame("TIT2")
	tag2.Save()

	tag3, _ := id3.Open(path)
	if tag3.Title() != "" {
		t.Errorf("After DeleteFrame, title should be empty, got %q", tag3.Title())
	}
}

func TestAllFields(t *testing.T) {
	path := createMinimalMP3(t)
	tag, _ := id3.Open(path)
	tag.SetTitle("All Fields Test")
	tag.SetArtist("Artist")
	tag.SetAlbum("Album")
	tag.SetGenre("Rock")
	tag.SetYear("2023")
	tag.SetTrack("03/10")
	tag.SetComment("eng", "", "A comment")
	tag.Save()

	tag2, _ := id3.Open(path)
	fields := tag2.AllFields()

	if fields["title"] != "All Fields Test" {
		t.Errorf("AllFields title = %q", fields["title"])
	}
	if fields["artist"] != "Artist" {
		t.Errorf("AllFields artist = %q", fields["artist"])
	}
	if fields["comment"] != "A comment" {
		t.Errorf("AllFields comment = %q", fields["comment"])
	}
}

func TestSavePreservesAudio(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "audio.mp3")

	// Simulate: ID3 tag + fake audio bytes
	const fakeAudio = "FAKEAUDIODATA12345678"
	tag := buildTagBytes("Original Title")
	fullFile := append(tag, []byte(fakeAudio)...)
	os.WriteFile(path, fullFile, 0o644)

	// Open, modify, save
	openedTag, _ := id3.Open(path)
	openedTag.SetTitle("Modified Title")
	openedTag.Save()

	// Read raw file back — fake audio must still be at the end
	saved, _ := os.ReadFile(path)
	if string(saved[len(saved)-len(fakeAudio):]) != fakeAudio {
		t.Errorf("Save() corrupted audio data; tail = %q", saved[len(saved)-20:])
	}

	// And the title must be updated
	tag2, _ := id3.Open(path)
	if tag2.Title() != "Modified Title" {
		t.Errorf("Title after save = %q", tag2.Title())
	}
}

// buildTagBytes builds a minimal ID3v2.3 header+title frame.
func buildTagBytes(title string) []byte {
	payload := append([]byte{0x03}, []byte(title)...)
	payload = append(payload, 0x00) // null terminator

	hdr10 := func(id string, size int) []byte {
		b := make([]byte, 10)
		copy(b[0:4], id)
		b[4] = byte(size >> 24)
		b[5] = byte(size >> 16)
		b[6] = byte(size >> 8)
		b[7] = byte(size)
		return b
	}

	frame := append(hdr10("TIT2", len(payload)), payload...)
	const padding = 256
	bodySize := len(frame) + padding

	id3Hdr := make([]byte, 10)
	copy(id3Hdr[0:3], "ID3")
	id3Hdr[3] = 3
	id3Hdr[6] = byte((bodySize >> 21) & 0x7F)
	id3Hdr[7] = byte((bodySize >> 14) & 0x7F)
	id3Hdr[8] = byte((bodySize >> 7) & 0x7F)
	id3Hdr[9] = byte(bodySize & 0x7F)

	var out []byte
	out = append(out, id3Hdr...)
	out = append(out, frame...)
	out = append(out, make([]byte, padding)...)
	return out
}

