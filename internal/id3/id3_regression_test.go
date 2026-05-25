package id3_test

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"

	"github.com/cumulus13/go-tagger/internal/id3"
)

// ── Helpers ───────────────────────────────────────────────────────────────────

// makeID3v23 builds a minimal valid ID3v2.3 file with the given raw frames.
func makeID3v23(t *testing.T, frames []byte) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "test.mp3")

	const padding = 512
	bodySize := len(frames) + padding

	hdr := make([]byte, 10)
	copy(hdr[0:3], "ID3")
	hdr[3] = 3 // version 2.3
	hdr[6] = byte((bodySize >> 21) & 0x7F)
	hdr[7] = byte((bodySize >> 14) & 0x7F)
	hdr[8] = byte((bodySize >> 7) & 0x7F)
	hdr[9] = byte(bodySize & 0x7F)

	var data []byte
	data = append(data, hdr...)
	data = append(data, frames...)
	data = append(data, make([]byte, padding)...)
	// append fake audio data so we can verify it survives a save
	data = append(data, 0xFF, 0xFB, 0x90, 0x00)

	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("write test file: %v", err)
	}
	return path
}

// mkFrameHdr builds a 10-byte ID3v2.3 frame header.
func mkFrameHdr(fid string, payloadSize int) []byte {
	h := make([]byte, 10)
	copy(h[0:4], fid)
	binary.BigEndian.PutUint32(h[4:8], uint32(payloadSize))
	// flags = 0x00 0x00
	return h
}

// latin1Frame builds a Latin-1 (enc=0) text frame, no trailing null.
func latin1Frame(fid, text string) []byte {
	payload := append([]byte{0x00}, []byte(text)...)
	return append(mkFrameHdr(fid, len(payload)), payload...)
}

// latin1FrameNull builds a Latin-1 text frame WITH trailing null terminator
// (some taggers include it, some don't — we must handle both).
func latin1FrameNull(fid, text string) []byte {
	payload := append([]byte{0x00}, append([]byte(text), 0x00)...)
	return append(mkFrameHdr(fid, len(payload)), payload...)
}

// utf8Frame builds a UTF-8 (enc=3) text frame, no trailing null.
func utf8Frame(fid, text string) []byte {
	payload := append([]byte{0x03}, []byte(text)...)
	return append(mkFrameHdr(fid, len(payload)), payload...)
}

// utf8FrameNull builds a UTF-8 text frame WITH trailing null.
func utf8FrameNull(fid, text string) []byte {
	payload := append([]byte{0x03}, append([]byte(text), 0x00)...)
	return append(mkFrameHdr(fid, len(payload)), payload...)
}

// utf16LEFrame builds a UTF-16LE + BOM (enc=1) text frame WITH null terminator.
// This is what iTunes, Windows Media Player, and many rippers write.
func utf16LEFrame(fid, text string) []byte {
	var utf16le []byte
	utf16le = append(utf16le, 0xFF, 0xFE) // BOM
	for _, r := range text {
		utf16le = append(utf16le, byte(r&0xFF), byte(r>>8))
	}
	utf16le = append(utf16le, 0x00, 0x00) // UTF-16 null terminator
	payload := append([]byte{0x01}, utf16le...)
	return append(mkFrameHdr(fid, len(payload)), payload...)
}

// utf16LEFrameNoNull builds UTF-16LE + BOM WITHOUT null terminator.
func utf16LEFrameNoNull(fid, text string) []byte {
	var utf16le []byte
	utf16le = append(utf16le, 0xFF, 0xFE) // BOM
	for _, r := range text {
		utf16le = append(utf16le, byte(r&0xFF), byte(r>>8))
	}
	payload := append([]byte{0x01}, utf16le...)
	return append(mkFrameHdr(fid, len(payload)), payload...)
}

// commFrame builds a COMM frame: enc(1) + lang(3) + desc + \x00 + text
func commFrame(enc byte, lang, desc, text string) []byte {
	var payload []byte
	payload = append(payload, enc)
	if len(lang) >= 3 {
		payload = append(payload, lang[:3]...)
	} else {
		payload = append(payload, []byte("eng")...)
	}
	payload = append(payload, []byte(desc)...)
	payload = append(payload, 0x00) // desc null terminator
	payload = append(payload, []byte(text)...)
	return append(mkFrameHdr("COMM", len(payload)), payload...)
}

// wxxxFrame builds a WXXX frame: enc(1) + desc + \x00 + url
func wxxxFrame(enc byte, desc, url string) []byte {
	var payload []byte
	payload = append(payload, enc)
	payload = append(payload, []byte(desc)...)
	payload = append(payload, 0x00)
	payload = append(payload, []byte(url)...)
	return append(mkFrameHdr("WXXX", len(payload)), payload...)
}

// ── Latin-1 text frame tests ──────────────────────────────────────────────────

func TestLatin1NoNullTerminator(t *testing.T) {
	// Most common case: mutagen writes Latin-1 without trailing null
	tests := []struct{ fid, text string }{
		{"TIT2", "Buzz"},
		{"TPE1", "Niki"},
		{"TALB", "Buzz"},
		{"TRCK", "01/13"},
		{"TCON", "Alt-Pop/Indie Pop/Indie Rock"},
		{"TPE2", "Niki"},
		{"TIT1", "Rock/Pop"},
		{"TPUB", "88rising"},
		{"TDRC", "2024"},
	}

	var frames []byte
	for _, tt := range tests {
		frames = append(frames, latin1Frame(tt.fid, tt.text)...)
	}

	path := makeID3v23(t, frames)
	tag, err := id3.Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}

	fields := tag.AllFields()
	want := map[string]string{
		"title":     "Buzz",
		"artist":    "Niki",
		"album":     "Buzz",
		"track":     "01/13",
		"genre":     "Alt-Pop/Indie Pop/Indie Rock",
		"album_artist": "Niki",
		"group":     "Rock/Pop",
		"publisher": "88rising",
		"date":      "2024",
	}
	for k, wantVal := range want {
		if got := fields[k]; got != wantVal {
			t.Errorf("Latin-1 no-null: fields[%q] = %q, want %q", k, got, wantVal)
		}
	}
}

func TestLatin1WithNullTerminator(t *testing.T) {
	// Some taggers DO include trailing null — we must not eat the last character
	var frames []byte
	frames = append(frames, latin1FrameNull("TIT2", "Buzz")...)
	frames = append(frames, latin1FrameNull("TRCK", "01/13")...)
	frames = append(frames, latin1FrameNull("TCON", "Alt-Pop/Indie Pop/Indie Rock")...)

	path := makeID3v23(t, frames)
	tag, _ := id3.Open(path)
	fields := tag.AllFields()

	if got := fields["title"]; got != "Buzz" {
		t.Errorf("Latin-1 null-term title = %q, want %q", got, "Buzz")
	}
	if got := fields["track"]; got != "01/13" {
		t.Errorf("Latin-1 null-term track = %q, want %q", got, "01/13")
	}
	if got := fields["genre"]; got != "Alt-Pop/Indie Pop/Indie Rock" {
		t.Errorf("Latin-1 null-term genre = %q, want %q", got, "Alt-Pop/Indie Pop/Indie Rock")
	}
}

// ── UTF-8 text frame tests ────────────────────────────────────────────────────

func TestUTF8NoNullTerminator(t *testing.T) {
	var frames []byte
	frames = append(frames, utf8Frame("TIT2", "Buzz")...)
	frames = append(frames, utf8Frame("TPE1", "Niki")...)
	frames = append(frames, utf8Frame("TRCK", "01/13")...)

	path := makeID3v23(t, frames)
	tag, _ := id3.Open(path)
	fields := tag.AllFields()

	if got := fields["title"]; got != "Buzz" {
		t.Errorf("UTF-8 no-null title = %q, want %q", got, "Buzz")
	}
	if got := fields["track"]; got != "01/13" {
		t.Errorf("UTF-8 no-null track = %q, want %q", got, "01/13")
	}
}

func TestUTF8WithNullTerminator(t *testing.T) {
	var frames []byte
	frames = append(frames, utf8FrameNull("TIT2", "Buzz")...)
	frames = append(frames, utf8FrameNull("TRCK", "01/13")...)
	frames = append(frames, utf8FrameNull("TCON", "Alt-Pop/Indie Pop/Indie Rock")...)

	path := makeID3v23(t, frames)
	tag, _ := id3.Open(path)
	fields := tag.AllFields()

	if got := fields["title"]; got != "Buzz" {
		t.Errorf("UTF-8 null-term title = %q, want %q", got, "Buzz")
	}
	if got := fields["genre"]; got != "Alt-Pop/Indie Pop/Indie Rock" {
		t.Errorf("UTF-8 null-term genre = %q, want %q", got, "Alt-Pop/Indie Pop/Indie Rock")
	}
}

// ── UTF-16 text frame tests (THE CRITICAL TRUNCATION BUG) ────────────────────

func TestUTF16LEBOMWithNullTerm(t *testing.T) {
	// This is the EXACT bug reported: iTunes/Windows encode UTF-16 with BOM +
	// null terminator. The old code did bytes.TrimRight("\x00") which stripped
	// individual bytes, corrupting the last UTF-16 character.
	tests := []struct{ fid, text string }{
		{"TIT2", "Buzz"},
		{"TPE1", "Niki"},
		{"TALB", "Buzz"},
		{"TRCK", "01/13"},
		{"TCON", "Alt-Pop/Indie Pop/Indie Rock"},
		{"TPE2", "Niki"},
		{"TIT1", "Rock/Pop"},
		{"TPUB", "88rising"},
	}

	var frames []byte
	for _, tt := range tests {
		frames = append(frames, utf16LEFrame(tt.fid, tt.text)...)
	}

	path := makeID3v23(t, frames)
	tag, err := id3.Open(path)
	if err != nil {
		t.Fatalf("Open UTF-16 LE BOM file: %v", err)
	}

	fields := tag.AllFields()
	want := map[string]string{
		"title":        "Buzz",
		"artist":       "Niki",
		"album":        "Buzz",
		"track":        "01/13",
		"genre":        "Alt-Pop/Indie Pop/Indie Rock",
		"album_artist": "Niki",
		"group":        "Rock/Pop",
		"publisher":    "88rising",
	}
	for k, wantVal := range want {
		got := fields[k]
		if got != wantVal {
			t.Errorf("UTF-16LE+BOM fields[%q] = %q, want %q (TRUNCATION BUG)", k, got, wantVal)
		}
	}
}

func TestUTF16LEBOMNoNullTerm(t *testing.T) {
	// UTF-16 with BOM but no null terminator at end
	var frames []byte
	frames = append(frames, utf16LEFrameNoNull("TIT2", "Buzz")...)
	frames = append(frames, utf16LEFrameNoNull("TRCK", "01/13")...)

	path := makeID3v23(t, frames)
	tag, _ := id3.Open(path)
	fields := tag.AllFields()

	if got := fields["title"]; got != "Buzz" {
		t.Errorf("UTF-16 no-null title = %q, want %q", got, "Buzz")
	}
	if got := fields["track"]; got != "01/13" {
		t.Errorf("UTF-16 no-null track = %q, want %q", got, "01/13")
	}
}

func TestUTF16MultipleNullTerms(t *testing.T) {
	// Pathological: some encoders write multiple \x00\x00 pairs after text
	text := "Buzz"
	var utf16le []byte
	utf16le = append(utf16le, 0xFF, 0xFE) // BOM
	for _, r := range text {
		utf16le = append(utf16le, byte(r), 0x00)
	}
	// Double null terminator
	utf16le = append(utf16le, 0x00, 0x00, 0x00, 0x00)
	payload := append([]byte{0x01}, utf16le...)

	var frames []byte
	frames = append(frames, append(mkFrameHdr("TIT2", len(payload)), payload...)...)

	path := makeID3v23(t, frames)
	tag, _ := id3.Open(path)
	if got := tag.Title(); got != "Buzz" {
		t.Errorf("UTF-16 double-null title = %q, want %q", got, "Buzz")
	}
}

func TestUTF16LongString(t *testing.T) {
	// Ensure full long genre string is not truncated
	genre := "Alt-Pop/Indie Pop/Indie Rock"
	var frames []byte
	frames = append(frames, utf16LEFrame("TCON", genre)...)

	path := makeID3v23(t, frames)
	tag, _ := id3.Open(path)
	if got := tag.Genre(); got != genre {
		t.Errorf("UTF-16 long genre = %q (%d chars), want %q (%d chars)",
			got, len([]rune(got)), genre, len([]rune(genre)))
	}
}

// ── COMM frame tests ──────────────────────────────────────────────────────────

func TestCOMMEmptyDesc(t *testing.T) {
	// Standard comment: COMM::eng with empty description — most common
	var frames []byte
	frames = append(frames, commFrame(0x00, "eng", "", "(myzuka)")...)

	path := makeID3v23(t, frames)
	tag, _ := id3.Open(path)
	desc, text := tag.GetComment()
	if text != "(myzuka)" {
		t.Errorf("COMM empty-desc text = %q, want %q", text, "(myzuka)")
	}
	if desc != "" {
		t.Errorf("COMM empty-desc desc = %q, want empty", desc)
	}
}

func TestCOMMID3v1Style(t *testing.T) {
	// COMM:ID3v1 Comment:eng — written by old rippers alongside regular COMM
	var frames []byte
	frames = append(frames, commFrame(0x00, "eng", "ID3v1 Comment", "(myzuka)")...)

	path := makeID3v23(t, frames)
	tag, _ := id3.Open(path)
	desc, text := tag.GetComment()
	if text != "(myzuka)" {
		t.Errorf("COMM ID3v1-style text = %q, want %q", text, "(myzuka)")
	}
	_ = desc
}

func TestCOMMPreferEmptyDesc(t *testing.T) {
	// When BOTH a "ID3v1 Comment" and empty-desc COMM exist,
	// GetComment must return the empty-desc one (the "real" comment).
	var frames []byte
	// Put ID3v1 Comment first — GetComment should still return the empty-desc
	frames = append(frames, commFrame(0x00, "eng", "ID3v1 Comment", "(myzuka)")...)
	frames = append(frames, commFrame(0x00, "eng", "", "Real Comment")...)

	path := makeID3v23(t, frames)
	tag, _ := id3.Open(path)
	desc, text := tag.GetComment()
	if desc != "" {
		t.Errorf("GetComment should prefer empty-desc COMM, got desc=%q", desc)
	}
	if text != "Real Comment" {
		t.Errorf("GetComment preferred text = %q, want %q", text, "Real Comment")
	}
}

func TestCOMMNullTerminatedText(t *testing.T) {
	// Some encoders put a null at the end of the text field
	payload := []byte{0x00, 'e', 'n', 'g', 0x00, '(', 'm', 'y', 'z', 'u', 'k', 'a', ')', 0x00}
	var frames []byte
	frames = append(frames, append(mkFrameHdr("COMM", len(payload)), payload...)...)

	path := makeID3v23(t, frames)
	tag, _ := id3.Open(path)
	_, text := tag.GetComment()
	if text != "(myzuka)" {
		t.Errorf("COMM null-term text = %q, want %q", text, "(myzuka)")
	}
}

// ── WXXX / URL frame tests ────────────────────────────────────────────────────

func TestWXXXEmptyDesc(t *testing.T) {
	// Standard WXXX with empty description (most common)
	url := "https://www.discogs.com/release/31467473-Niki-31-Buzz"
	var frames []byte
	frames = append(frames, wxxxFrame(0x00, "", url)...)

	path := makeID3v23(t, frames)
	tag, _ := id3.Open(path)
	got := tag.GetURL()
	if got != url {
		t.Errorf("WXXX empty-desc url = %q, want %q", got, url)
	}
}

func TestWXXXWithDesc(t *testing.T) {
	url := "https://example.com/artist"
	var frames []byte
	frames = append(frames, wxxxFrame(0x00, "Homepage", url)...)

	path := makeID3v23(t, frames)
	tag, _ := id3.Open(path)
	got := tag.GetURL()
	if got != url {
		t.Errorf("WXXX with-desc url = %q, want %q", got, url)
	}
}

// ── TXXX frame tests ──────────────────────────────────────────────────────────

func TestTXXXCountryAndCatalog(t *testing.T) {
	// Real-world TXXX frames seen in the bug report
	buildTXXX := func(desc, val string) []byte {
		payload := append([]byte{0x00}, append([]byte(desc), append([]byte{0x00}, []byte(val)...)...)...)
		return append(mkFrameHdr("TXXX", len(payload)), payload...)
	}

	var frames []byte
	frames = append(frames, buildTXXX("COUNTRY", "Canada")...)
	frames = append(frames, buildTXXX("CATALOGNUMBER", "none")...)
	frames = append(frames, buildTXXX("BARCODE", "1234567890")...)

	path := makeID3v23(t, frames)
	tag, _ := id3.Open(path)
	fields := tag.AllFields()

	if got := fields["country"]; got != "Canada" {
		t.Errorf("TXXX:COUNTRY = %q, want %q", got, "Canada")
	}
	if got := fields["catalognumber"]; got != "none" {
		t.Errorf("TXXX:CATALOGNUMBER = %q, want %q", got, "none")
	}
	if got := fields["barcode"]; got != "1234567890" {
		t.Errorf("TXXX:BARCODE = %q, want %q", got, "1234567890")
	}
}

// ── Full real-world simulation (matching the bug report) ─────────────────────

func TestRealWorldNikiBuzz(t *testing.T) {
	// Simulate exactly the tag layout from the bug report:
	// Latin-1 encoded frames from what appears to be a mutagen-written file
	// that showed truncated values in go-tagger -I output.
	var frames []byte

	// Text frames (Latin-1, no null terminator — mutagen default)
	for fid, text := range map[string]string{
		"TALB": "Buzz",
		"TPE1": "Niki",
		"TIT2": "Buzz",
		"TCON": "Alt-Pop/Indie Pop/Indie Rock",
		"TRCK": "01/13",
		"TSSE": "Lavf56.15.102",
		"TPE2": "Niki",
		"TIT1": "Rock/Pop",
		"TPUB": "88rising",
		"TMED": "CD",
		"TDRC": "2024",
	} {
		frames = append(frames, latin1Frame(fid, text)...)
	}

	// COMM::eng = "(myzuka)" (empty description)
	frames = append(frames, commFrame(0x00, "eng", "", "(myzuka)")...)

	// COMM:ID3v1 Comment:eng = "(myzuka)"
	frames = append(frames, commFrame(0x00, "eng", "ID3v1 Comment", "(myzuka)")...)

	// WXXX: empty desc, discogs URL
	frames = append(frames, wxxxFrame(0x00, "", "https://www.discogs.com/release/31467473-Niki-31-Buzz")...)

	// TXXX:COUNTRY = Canada
	buildTXXX := func(desc, val string) []byte {
		payload := append([]byte{0x00}, append([]byte(desc), append([]byte{0x00}, []byte(val)...)...)...)
		return append(mkFrameHdr("TXXX", len(payload)), payload...)
	}
	frames = append(frames, buildTXXX("COUNTRY", "Canada")...)
	frames = append(frames, buildTXXX("CATALOGNUMBER", "none")...)

	path := makeID3v23(t, frames)
	tag, err := id3.Open(path)
	if err != nil {
		t.Fatalf("Open real-world sim: %v", err)
	}

	fields := tag.AllFields()

	// These are the exact values from the bug report that were truncated
	exact := map[string]string{
		"title":         "Buzz",
		"artist":        "Niki",
		"album":         "Buzz",
		"album_artist":  "Niki",
		"track":         "01/13",
		"genre":         "Alt-Pop/Indie Pop/Indie Rock",
		"group":         "Rock/Pop",
		"publisher":     "88rising",
		"date":          "2024",
		"comment":       "(myzuka)",
		"url":           "https://www.discogs.com/release/31467473-Niki-31-Buzz",
		"country":       "Canada",
		"catalognumber": "none",
	}

	for k, wantVal := range exact {
		got := fields[k]
		if got != wantVal {
			t.Errorf("real-world fields[%q]:\n  got  %q (%d chars)\n  want %q (%d chars)",
				k, got, len([]rune(got)), wantVal, len([]rune(wantVal)))
		}
	}
}
