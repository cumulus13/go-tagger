// Package id3 implements a production-quality ID3v2.3/v2.4 tag reader and writer
// for MP3 files with zero external dependencies.
package id3

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"strings"
	"unicode/utf16"
)

// ── Encoding constants (ID3v2 spec) ────────────────────────────────────────────

const (
	EncLatin1  byte = 0 // ISO-8859-1
	EncUTF16   byte = 1 // UTF-16 with BOM
	EncUTF16BE byte = 2 // UTF-16BE, no BOM
	EncUTF8    byte = 3 // UTF-8
)

const id3Magic = "ID3"

// ── Tag ───────────────────────────────────────────────────────────────────────

// Tag holds all ID3v2 frames read from an MP3 file.
// frames maps frameID → slice of raw frame payloads (everything after the
// 10-byte frame header, i.e. including the encoding byte for text frames).
type Tag struct {
	path        string
	version     byte // major version: 3 or 4
	flags       byte
	frames      map[string][][]byte
	origTagSize int // bytes consumed by original tag (header + body)
}

// Open reads and parses the ID3v2 tag from path.
// If no tag is present a valid empty Tag is returned (ready for writing).
func Open(path string) (*Tag, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	t := &Tag{path: path, frames: make(map[string][][]byte)}
	if err := t.parse(f); err != nil && err != errNoTag {
		return nil, err
	}
	return t, nil
}

var errNoTag = fmt.Errorf("no ID3v2 tag")

// parse reads and validates the 10-byte ID3 header, then parses all frames.
func (t *Tag) parse(r io.Reader) error {
	var hdr [10]byte
	if _, err := io.ReadFull(r, hdr[:]); err != nil {
		return errNoTag
	}
	if string(hdr[0:3]) != id3Magic {
		return errNoTag
	}
	t.version = hdr[3]
	t.flags = hdr[5]

	tagSize := decodeSyncsafe(hdr[6:10])
	t.origTagSize = 10 + tagSize

	buf := make([]byte, tagSize)
	if _, err := io.ReadFull(r, buf); err != nil {
		return fmt.Errorf("reading tag body: %w", err)
	}

	t.parseFrames(buf)
	return nil
}

// parseFrames walks the raw frame buffer and populates t.frames.
func (t *Tag) parseFrames(buf []byte) {
	const frameHdrLen = 10
	pos := 0
	for pos+frameHdrLen <= len(buf) {
		// Padding / end of frames
		if buf[pos] == 0 {
			break
		}
		fid := string(buf[pos : pos+4])
		if !isValidFrameID(fid) {
			break
		}

		var size int
		if t.version == 4 {
			// ID3v2.4: frame sizes are syncsafe
			size = decodeSyncsafe(buf[pos+4 : pos+8])
		} else {
			// ID3v2.3: frame sizes are plain big-endian uint32
			size = int(binary.BigEndian.Uint32(buf[pos+4 : pos+8]))
		}
		// flags at pos+8, pos+9 — we store but mostly ignore them for now
		pos += frameHdrLen

		if size <= 0 || pos+size > len(buf) {
			break
		}

		payload := make([]byte, size)
		copy(payload, buf[pos:pos+size])
		t.frames[fid] = append(t.frames[fid], payload)
		pos += size
	}
}

func isValidFrameID(s string) bool {
	for _, c := range s {
		if !((c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9')) {
			return false
		}
	}
	return true
}

// ── Syncsafe integers ─────────────────────────────────────────────────────────

func decodeSyncsafe(b []byte) int {
	return int(b[0])<<21 | int(b[1])<<14 | int(b[2])<<7 | int(b[3])
}

func encodeSyncsafe(n int) [4]byte {
	return [4]byte{
		byte((n >> 21) & 0x7F),
		byte((n >> 14) & 0x7F),
		byte((n >> 7) & 0x7F),
		byte(n & 0x7F),
	}
}

// ── Text decoding ─────────────────────────────────────────────────────────────

// decodeText decodes a full frame payload (first byte = encoding, rest = text).
// Handles all four ID3v2 encodings and strips null terminators correctly.
func decodeText(data []byte) string {
	if len(data) == 0 {
		return ""
	}
	enc := data[0]
	payload := data[1:]

	switch enc {
	case EncUTF16, EncUTF16BE:
		return decodeUTF16text(payload, enc)
	default:
		// Latin-1 or UTF-8: strip at most one trailing null byte
		return trimNull1(payload)
	}
}

// decodeUTF16text decodes a UTF-16 payload, handling BOM and null terminators.
// The critical fix: for UTF-16 we must strip \x00\x00 pairs at the end, not
// individual \x00 bytes — bytes.TrimRight("\x00") destroys the last character.
func decodeUTF16text(b []byte, enc byte) string {
	if len(b) == 0 {
		return ""
	}

	// Strip trailing UTF-16 null terminator (\x00\x00) if present
	for len(b) >= 2 && b[len(b)-2] == 0 && b[len(b)-1] == 0 {
		b = b[:len(b)-2]
	}

	if len(b) == 0 {
		return ""
	}

	var order binary.ByteOrder = binary.BigEndian
	start := 0

	// Detect BOM
	if len(b) >= 2 && b[0] == 0xFF && b[1] == 0xFE {
		order = binary.LittleEndian
		start = 2
	} else if len(b) >= 2 && b[0] == 0xFE && b[1] == 0xFF {
		order = binary.BigEndian
		start = 2
	} else if enc == EncUTF16 {
		// No BOM present, default to little-endian per common practice
		order = binary.LittleEndian
	}

	b = b[start:]
	// Ensure even length (drop a stray byte)
	if len(b)%2 != 0 {
		b = b[:len(b)-1]
	}
	if len(b) == 0 {
		return ""
	}

	u16 := make([]uint16, len(b)/2)
	for i := range u16 {
		if order == binary.LittleEndian {
			u16[i] = binary.LittleEndian.Uint16(b[i*2:])
		} else {
			u16[i] = binary.BigEndian.Uint16(b[i*2:])
		}
	}
	// Remove UTF-16 null codepoints at the end
	for len(u16) > 0 && u16[len(u16)-1] == 0 {
		u16 = u16[:len(u16)-1]
	}
	return string(utf16.Decode(u16))
}

// trimNull1 strips exactly one trailing null byte from a Latin-1/UTF-8 payload.
func trimNull1(b []byte) string {
	if len(b) > 0 && b[len(b)-1] == 0 {
		return string(b[:len(b)-1])
	}
	return string(b)
}

// encodeText encodes a string as a frame payload: enc_byte(UTF-8) + text.
// We always write UTF-8; no trailing null (the frame size is authoritative).
func encodeText(s string) []byte {
	out := make([]byte, 1+len(s))
	out[0] = EncUTF8
	copy(out[1:], s)
	return out
}

// ── Null helpers ──────────────────────────────────────────────────────────────

// nullLen returns the number of bytes in a null terminator for the encoding.
func nullLen(enc byte) int {
	if enc == EncUTF16 || enc == EncUTF16BE {
		return 2
	}
	return 1
}

// findNull returns the byte index of the first null terminator in b,
// respecting the encoding's null width. Returns -1 if not found.
func findNull(b []byte, enc byte) int {
	if enc == EncUTF16 || enc == EncUTF16BE {
		// Must be aligned to 2-byte boundary
		for i := 0; i+1 < len(b); i += 2 {
			if b[i] == 0 && b[i+1] == 0 {
				return i
			}
		}
		return -1
	}
	return bytes.IndexByte(b, 0)
}

// ── Getters ───────────────────────────────────────────────────────────────────

func (t *Tag) getText(frameID string) string {
	if frames := t.frames[frameID]; len(frames) > 0 {
		return decodeText(frames[0])
	}
	return ""
}

// GetText returns the decoded text of the first instance of frameID.
func (t *Tag) GetText(frameID string) string { return t.getText(frameID) }

func (t *Tag) Title()  string { return t.getText("TIT2") }
func (t *Tag) Artist() string { return t.getText("TPE1") }
func (t *Tag) Album()  string { return t.getText("TALB") }
func (t *Tag) Genre()  string { return t.getText("TCON") }
func (t *Tag) Year()   string { return t.getText("TDRC") }
func (t *Tag) Track()  string { return t.getText("TRCK") }

// GetTXXX returns the value of the first TXXX frame whose description matches
// desc (case-insensitive). Returns "" if not found.
func (t *Tag) GetTXXX(desc string) string {
	for _, data := range t.frames["TXXX"] {
		if len(data) < 2 {
			continue
		}
		enc := data[0]
		rest := data[1:]
		sep := findNull(rest, enc)
		if sep < 0 {
			continue
		}
		frameDesc := decodeText(append([]byte{enc}, rest[:sep]...))
		if strings.EqualFold(frameDesc, desc) {
			valStart := sep + nullLen(enc)
			if valStart > len(rest) {
				return ""
			}
			return decodeText(append([]byte{enc}, rest[valStart:]...))
		}
	}
	return ""
}

// GetComment returns the description and text of the first COMM frame.
// Frames with non-empty descriptions (e.g. "ID3v1 Comment") are tried last
// so that the "main" comment (empty description) is returned preferentially.
func (t *Tag) GetComment() (desc, text string) {
	var fallback []byte
	for _, data := range t.frames["COMM"] {
		if len(data) < 5 {
			// minimum: enc(1) + lang(3) + null(1)
			continue
		}
		enc := data[0]
		// data[1:4] = 3-char language code
		rest := data[4:] // everything after language

		sep := findNull(rest, enc)
		var d, v string
		if sep < 0 {
			// No null separator → treat entire rest as text, empty description
			d = ""
			v = decodeText(append([]byte{enc}, rest...))
		} else {
			d = decodeText(append([]byte{enc}, rest[:sep]...))
			valStart := sep + nullLen(enc)
			if valStart <= len(rest) {
				v = decodeText(append([]byte{enc}, rest[valStart:]...))
			}
		}

		if d == "" {
			// Prefer frames with empty description (the "real" comment)
			return d, v
		}
		if fallback == nil {
			fallback = data
		}
	}

	// No empty-desc frame found — return first non-empty-desc frame
	if fallback != nil {
		enc := fallback[0]
		rest := fallback[4:]
		sep := findNull(rest, enc)
		if sep < 0 {
			return "", decodeText(append([]byte{enc}, rest...))
		}
		d := decodeText(append([]byte{enc}, rest[:sep]...))
		valStart := sep + nullLen(enc)
		v := ""
		if valStart <= len(rest) {
			v = decodeText(append([]byte{enc}, rest[valStart:]...))
		}
		return d, v
	}
	return "", ""
}

// GetLyric returns the lyrics text of the first USLT frame.
func (t *Tag) GetLyric() string {
	for _, data := range t.frames["USLT"] {
		if len(data) < 5 {
			continue
		}
		enc := data[0]
		// data[1:4] = language, skip it
		rest := data[4:]
		sep := findNull(rest, enc)
		if sep < 0 {
			return decodeText(append([]byte{enc}, rest...))
		}
		valStart := sep + nullLen(enc)
		if valStart > len(rest) {
			return ""
		}
		return decodeText(append([]byte{enc}, rest[valStart:]...))
	}
	return ""
}

// GetURL returns the URL from the first WXXX frame.
// Per spec, the URL field is always Latin-1; only the description is encoded.
func (t *Tag) GetURL() string {
	for _, data := range t.frames["WXXX"] {
		if len(data) < 2 {
			continue
		}
		enc := data[0]
		rest := data[1:]
		sep := findNull(rest, enc)
		if sep < 0 {
			// No description terminator — treat entire rest as URL
			return string(rest)
		}
		urlStart := sep + nullLen(enc)
		if urlStart > len(rest) {
			return ""
		}
		// URL is always raw bytes / Latin-1 regardless of encoding byte
		url := string(rest[urlStart:])
		// Strip trailing null if any
		url = strings.TrimRight(url, "\x00")
		return url
	}
	return ""
}

// GetCover returns the raw image data and MIME type of the first APIC frame.
func (t *Tag) GetCover() (data []byte, mime string) {
	for _, raw := range t.frames["APIC"] {
		if len(raw) < 5 {
			continue
		}
		enc := raw[0]
		rest := raw[1:]

		// MIME type: null-terminated Latin-1 (always, regardless of enc byte)
		mimeEnd := bytes.IndexByte(rest, 0)
		if mimeEnd < 0 {
			continue
		}
		mime = string(rest[:mimeEnd])
		rest = rest[mimeEnd+1:]

		if len(rest) < 2 {
			continue
		}
		// Picture type byte
		rest = rest[1:]

		// Description: null-terminated, encoding-aware
		sep := findNull(rest, enc)
		var imgStart int
		if sep < 0 {
			// No description — image starts immediately
			imgStart = 0
		} else {
			imgStart = sep + nullLen(enc)
		}

		if imgStart > len(rest) {
			continue
		}
		data = rest[imgStart:]
		return
	}
	return nil, ""
}

// ── Setters ───────────────────────────────────────────────────────────────────

// SetText replaces (or deletes if value=="") the first instance of frameID.
func (t *Tag) SetText(frameID, value string) {
	if value == "" {
		delete(t.frames, frameID)
		return
	}
	t.frames[frameID] = [][]byte{encodeText(value)}
}

func (t *Tag) SetTitle(v string)  { t.SetText("TIT2", v) }
func (t *Tag) SetArtist(v string) { t.SetText("TPE1", v) }
func (t *Tag) SetAlbum(v string)  { t.SetText("TALB", v) }
func (t *Tag) SetGenre(v string)  { t.SetText("TCON", v) }
func (t *Tag) SetYear(v string)   { t.SetText("TDRC", v) }
func (t *Tag) SetTrack(v string)  { t.SetText("TRCK", v) }

// SetTXXX sets (or replaces) a TXXX frame matching desc (case-insensitive).
// Pass value=="" to delete the frame.
func (t *Tag) SetTXXX(desc, value string) {
	// Filter out existing frames with the same description
	var kept [][]byte
	for _, data := range t.frames["TXXX"] {
		if d := getTXXXDesc(data); !strings.EqualFold(d, desc) {
			kept = append(kept, data)
		}
	}
	if value == "" {
		t.frames["TXXX"] = kept
		return
	}
	// Build: enc(UTF-8) + desc + \x00 + value
	var buf bytes.Buffer
	buf.WriteByte(EncUTF8)
	buf.WriteString(desc)
	buf.WriteByte(0)
	buf.WriteString(value)
	t.frames["TXXX"] = append(kept, buf.Bytes())
}

func getTXXXDesc(data []byte) string {
	if len(data) < 2 {
		return ""
	}
	enc := data[0]
	rest := data[1:]
	sep := findNull(rest, enc)
	if sep < 0 {
		return string(rest)
	}
	return decodeText(append([]byte{enc}, rest[:sep]...))
}

// SetComment sets the COMM frame, replacing all existing COMM frames.
func (t *Tag) SetComment(lang, desc, text string) {
	if lang == "" || len(lang) < 3 {
		lang = "eng"
	}
	var buf bytes.Buffer
	buf.WriteByte(EncUTF8)
	buf.WriteString(lang[:3])
	buf.WriteString(desc)
	buf.WriteByte(0) // desc null terminator
	buf.WriteString(text)
	t.frames["COMM"] = [][]byte{buf.Bytes()}
}

// SetLyric sets the USLT (unsynchronised lyrics) frame.
func (t *Tag) SetLyric(lang, desc, text string) {
	if lang == "" {
		lang = "xxx"
	}
	var buf bytes.Buffer
	buf.WriteByte(EncUTF8)
	buf.WriteString(lang[:3])
	buf.WriteString(desc)
	buf.WriteByte(0) // desc null terminator
	buf.WriteString(text)
	t.frames["USLT"] = [][]byte{buf.Bytes()}
}

// SetURL sets the WXXX (user-defined URL) frame.
// The URL itself is stored as raw Latin-1 bytes per spec.
func (t *Tag) SetURL(desc, url string) {
	var buf bytes.Buffer
	buf.WriteByte(EncLatin1)
	buf.WriteString(desc)
	buf.WriteByte(0) // desc null terminator
	buf.WriteString(url)
	t.frames["WXXX"] = [][]byte{buf.Bytes()}
}

// SetCover sets the APIC (attached picture) frame.
// pictureType 3 = front cover (most common).
func (t *Tag) SetCover(mimeType string, pictureType byte, desc string, imgData []byte) {
	var buf bytes.Buffer
	buf.WriteByte(EncLatin1)
	buf.WriteString(mimeType)
	buf.WriteByte(0) // MIME null terminator
	buf.WriteByte(pictureType)
	buf.WriteString(desc)
	buf.WriteByte(0) // desc null terminator
	buf.Write(imgData)
	t.frames["APIC"] = [][]byte{buf.Bytes()}
}

// DeleteFrame removes all instances of the given frame ID.
func (t *Tag) DeleteFrame(frameID string) {
	delete(t.frames, frameID)
}

// ── Save ──────────────────────────────────────────────────────────────────────

// Save writes the updated tag to disk atomically (temp-file + rename).
// Audio data is preserved exactly.
func (t *Tag) Save() error {
	orig, err := os.ReadFile(t.path)
	if err != nil {
		return fmt.Errorf("read %q: %w", t.path, err)
	}

	// Locate the start of audio data (after the original ID3 tag, if any)
	audioStart := 0
	if len(orig) >= 10 && string(orig[0:3]) == id3Magic {
		audioStart = 10 + decodeSyncsafe(orig[6:10])
		if audioStart > len(orig) {
			audioStart = len(orig)
		}
	}
	audio := orig[audioStart:]

	tagBuf, err := t.marshal()
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}

	// Atomic write: write to temp, rename over original
	tmp := t.path + ".gotagger.tmp"
	f, err := os.Create(tmp)
	if err != nil {
		return fmt.Errorf("create temp: %w", err)
	}
	if _, err := f.Write(tagBuf); err != nil {
		f.Close(); os.Remove(tmp)
		return fmt.Errorf("write tag: %w", err)
	}
	if _, err := f.Write(audio); err != nil {
		f.Close(); os.Remove(tmp)
		return fmt.Errorf("write audio: %w", err)
	}
	if err := f.Close(); err != nil {
		os.Remove(tmp)
		return fmt.Errorf("close temp: %w", err)
	}
	if err := os.Rename(tmp, t.path); err != nil {
		os.Remove(tmp)
		return fmt.Errorf("rename: %w", err)
	}
	return nil
}

// marshal serialises all frames into a complete ID3v2.3 tag block.
func (t *Tag) marshal() ([]byte, error) {
	var framesBuf bytes.Buffer
	for fid, blobs := range t.frames {
		if len(fid) != 4 {
			continue
		}
		for _, data := range blobs {
			if len(data) == 0 {
				continue
			}
			// 4-byte frame ID
			framesBuf.WriteString(fid)
			// 4-byte size: plain big-endian uint32 (ID3v2.3 format)
			var sz [4]byte
			binary.BigEndian.PutUint32(sz[:], uint32(len(data)))
			framesBuf.Write(sz[:])
			// 2-byte flags (both zero)
			framesBuf.WriteByte(0)
			framesBuf.WriteByte(0)
			// payload
			framesBuf.Write(data)
		}
	}

	const padding = 2048 // generous padding for future in-place edits
	framesBytes := framesBuf.Bytes()
	bodySize := len(framesBytes) + padding

	// ID3v2.3 header
	var hdr [10]byte
	copy(hdr[0:3], id3Magic)
	hdr[3] = 3 // version 2.3
	hdr[4] = 0 // revision 0
	hdr[5] = 0 // flags
	ss := encodeSyncsafe(bodySize)
	copy(hdr[6:10], ss[:])

	var out bytes.Buffer
	out.Write(hdr[:])
	out.Write(framesBytes)
	out.Write(make([]byte, padding))
	return out.Bytes(), nil
}

// ── AllFields ─────────────────────────────────────────────────────────────────

// AllFields returns every well-known field as a string map.
func (t *Tag) AllFields() map[string]string {
	_, comment := t.GetComment()
	return map[string]string{
		"title":           t.Title(),
		"artist":          t.Artist(),
		"album":           t.Album(),
		"album_artist":    t.GetText("TPE2"),
		"original_artist": t.GetText("TOPE"),
		"composer":        t.GetText("TCOM"),
		"genre":           t.Genre(),
		"date":            t.Year(),
		"track":           t.Track(),
		"disc":            t.GetText("TPOS"),
		"isrc":            t.GetText("TSRC"),
		"bpm":             t.GetText("TBPM"),
		"copyright":       t.GetText("TCOP"),
		"encodedby":       t.GetText("TENC") + t.GetText("TSSE"), // some taggers use TSSE
		"key":             t.GetText("TKEY"),
		"length":          t.GetText("TLEN"),
		"remix":           t.GetText("TPE4"),
		"subtitle":        t.GetText("TIT3"),
		"group":           t.GetText("TIT1"),
		"publisher":       t.GetText("TPUB"),
		"lyric_name":      t.GetText("TEXT"),
		"media":           t.GetText("TMED"),
		"barcode":         t.GetTXXX("BARCODE"),
		"catalognumber":   t.GetTXXX("CATALOGNUMBER"),
		"country":         t.GetTXXX("COUNTRY"),
		"solo_artist":     t.GetTXXX("ARTIST"),
		"cover_name":      t.GetTXXX("_cover"),
		"lyric":           t.GetLyric(),
		"url":             t.GetURL(),
		"comment":         comment,
	}
}
