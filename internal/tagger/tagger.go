// Package tagger implements all MP3 tag set/get/rename/cover operations.
package tagger

import (
	"bytes"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	clr "github.com/cumulus13/go-tagger/internal/color"
	"github.com/cumulus13/go-tagger/internal/config"
	"github.com/cumulus13/go-tagger/internal/id3"
	"github.com/cumulus13/go-tagger/pkg/mimelist"
)

// ── TagSet ────────────────────────────────────────────────────────────────────

// TagSet holds all values to write. Empty string = no change. "clear" = delete.
type TagSet struct {
	Title, Track, Disc, Album, Artist, AlbumArtist, OrigArtist string
	Composer, Comment, CommentDesc, ISRC, Barcode, Genre       string
	Date, BPM, Copyright, EncodedBy, Key, Lyric, LyricName    string
	Remix, Subtitle, URL, Group, Publisher, Length             string
	CoverName, Cover, SoloArtist                               string
	TestMode                                                   bool
}

// ── TrackInfo ─────────────────────────────────────────────────────────────────

// TrackInfo holds a parsed track number + title from a text line.
type TrackInfo struct{ Track, Title string }

var reTrackLine = regexp.MustCompile(`^(\d+)\.\s+(.+)$`)
var reMp3 = regexp.MustCompile(`(?i)\.mp3$`)

// ParseTitleLine parses "NN. Title" or plain "Title" lines.
func ParseTitleLine(line string) TrackInfo {
	line = strings.TrimSpace(line)
	if m := reTrackLine.FindStringSubmatch(line); len(m) == 3 {
		title := strings.TrimSpace(reMp3.ReplaceAllString(m[2], ""))
		return TrackInfo{Track: m[1], Title: titleCase(title)}
	}
	title := strings.TrimSpace(reMp3.ReplaceAllString(line, ""))
	return TrackInfo{Title: titleCase(title)}
}

func titleCase(s string) string {
	words := strings.Fields(s)
	for i, w := range words {
		if len(w) > 0 {
			words[i] = strings.ToUpper(w[:1]) + strings.ToLower(w[1:])
		}
	}
	return strings.Join(words, " ")
}

// ── Track/disc formatting ─────────────────────────────────────────────────────

// FormatTrack returns a zero-padded "NN/TT" track string.
func FormatTrack(track string, total int, change bool) string {
	pad := func(s string) string {
		if len(s) == 1 {
			return "0" + s
		}
		return s
	}
	totalStr := strconv.Itoa(total)
	if total < 10 {
		totalStr = "0" + totalStr
	}
	if !strings.Contains(track, "/") {
		return pad(track) + "/" + totalStr
	}
	parts := strings.SplitN(track, "/", 2)
	fr := pad(strings.TrimSpace(parts[0]))
	to := strings.TrimSpace(parts[1])
	if change {
		return fr + "/" + totalStr
	}
	return fr + "/" + pad(to)
}

// FormatDisc returns a zero-padded "NN/MM" disc string.
func FormatDisc(disc string) string {
	if disc == "" {
		return config.DefaultDisc
	}
	pad := func(s string) string {
		if len(s) == 1 {
			return "0" + s
		}
		return s
	}
	if !strings.Contains(disc, "/") {
		return pad(disc) + "/01"
	}
	parts := strings.SplitN(disc, "/", 2)
	return pad(strings.TrimSpace(parts[0])) + "/" + pad(strings.TrimSpace(parts[1]))
}

// ── ScanDir ───────────────────────────────────────────────────────────────────

// ScanDir returns all .mp3 files in dir (non-recursive, sorted).
func ScanDir(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read dir %q: %w", dir, err)
	}
	var files []string
	for _, e := range entries {
		if !e.IsDir() && strings.EqualFold(filepath.Ext(e.Name()), ".mp3") {
			files = append(files, filepath.Join(dir, e.Name()))
		}
	}
	sort.Strings(files)
	return files, nil
}

// ── GetAll ────────────────────────────────────────────────────────────────────

// GetAll opens a file and returns all its tag fields.
func GetAll(path string) (map[string]string, error) {
	t, err := id3.Open(path)
	if err != nil {
		return nil, err
	}
	return t.AllFields(), nil
}

// ── Set ───────────────────────────────────────────────────────────────────────

// Set writes a TagSet to an MP3 file.
func Set(path string, ts TagSet) error {
	if !strings.EqualFold(filepath.Ext(path), ".mp3") {
		return fmt.Errorf("not an MP3 file: %q", path)
	}

	t, err := id3.Open(path)
	if err != nil {
		return fmt.Errorf("open %q: %w", path, err)
	}

	cur := t.AllFields()

	// applyText sets a simple text frame, printing a diff line.
	applyText := func(label, frameID, newVal string) {
		if newVal == "" {
			return
		}
		old := cur[strings.ToLower(strings.ReplaceAll(label, " ", "_"))]
		if old == "" {
			old = t.GetText(frameID)
		}
		actual := newVal
		if actual == "clear" {
			actual = ""
		}
		if old != actual {
			fmt.Println(clr.TagChange(label, old, actual))
		} else {
			fmt.Println(clr.TagSame(label, actual))
		}
		t.SetText(frameID, actual)
	}

	applyText("title", "TIT2", ts.Title)
	applyText("artist", "TPE1", ts.Artist)
	applyText("album", "TALB", ts.Album)
	applyText("genre", "TCON", ts.Genre)
	applyText("date", "TDRC", ts.Date)
	applyText("track", "TRCK", ts.Track)
	applyText("disc", "TPOS", ts.Disc)
	applyText("album_artist", "TPE2", ts.AlbumArtist)
	applyText("original_artist", "TOPE", ts.OrigArtist)
	applyText("composer", "TCOM", ts.Composer)
	applyText("isrc", "TSRC", ts.ISRC)
	applyText("bpm", "TBPM", ts.BPM)
	applyText("copyright", "TCOP", ts.Copyright)
	applyText("encodedby", "TENC", ts.EncodedBy)
	applyText("key", "TKEY", ts.Key)
	applyText("length", "TLEN", ts.Length)
	applyText("remix", "TPE4", ts.Remix)
	applyText("subtitle", "TIT3", ts.Subtitle)
	applyText("group", "TIT1", ts.Group)
	applyText("publisher", "TPUB", ts.Publisher)
	applyText("lyric_name", "TEXT", ts.LyricName)

	// TXXX frames
	applyTXXX := func(label, desc, newVal string) {
		if newVal == "" {
			return
		}
		old := cur[strings.ToLower(strings.ReplaceAll(label, " ", "_"))]
		actual := newVal
		if actual == "clear" {
			actual = ""
		}
		if old != actual {
			fmt.Println(clr.TagChange(label, old, actual))
		} else {
			fmt.Println(clr.TagSame(label, actual))
		}
		t.SetTXXX(desc, actual)
	}
	applyTXXX("barcode", "BARCODE", ts.Barcode)
	applyTXXX("solo_artist", "ARTIST", ts.SoloArtist)
	applyTXXX("cover_name", "_cover", ts.CoverName)

	// COMM
	if ts.Comment != "" {
		actual := ts.Comment
		if actual == "clear" {
			actual = ""
		}
		_, old := t.GetComment()
		if old != actual {
			fmt.Println(clr.TagChange("comment", old, actual))
		} else {
			fmt.Println(clr.TagSame("comment", actual))
		}
		t.SetComment("eng", ts.CommentDesc, actual)
	}

	// USLT
	if ts.Lyric != "" {
		actual := ts.Lyric
		if actual == "clear" {
			actual = ""
		}
		old := t.GetLyric()
		preview := func(s string) string {
			r := []rune(s)
			if len(r) > 40 {
				return string(r[:40]) + "…"
			}
			return s
		}
		if old != actual {
			fmt.Println(clr.TagChange("lyric", preview(old), preview(actual)))
		} else {
			fmt.Println(clr.TagSame("lyric", preview(actual)))
		}
		t.SetLyric("xxx", "", actual)
	}

	// WXXX
	if ts.URL != "" {
		actual := ts.URL
		if actual == "clear" {
			actual = ""
		}
		old := t.GetURL()
		if old != actual {
			fmt.Println(clr.TagChange("url", old, actual))
		} else {
			fmt.Println(clr.TagSame("url", actual))
		}
		t.SetURL("", actual)
	}

	// Cover art
	if ts.Cover != "" && ts.Cover != "clear" {
		if err := applyCover(t, ts.Cover); err != nil {
			fmt.Printf("%s %s: %v\n", clr.EmojiError, clr.Err("Cover"), err)
		} else {
			fmt.Printf("%s %s\n", clr.EmojiCover, clr.OK("Cover updated"))
		}
	}

	if ts.TestMode {
		fmt.Printf("\n%s %s\n", clr.EmojiTest, clr.Warn("TEST MODE — no changes written to disk"))
		return nil
	}

	if err := t.Save(); err != nil {
		return err
	}
	fmt.Printf("%s %s\n", clr.EmojiSave, clr.OK("Saved"))
	return nil
}

func applyCover(t *id3.Tag, imgPath string) error {
	data, err := os.ReadFile(imgPath)
	if err != nil {
		return err
	}
	_, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("decode image: %w", err)
	}
	t.SetCover(mimelist.ImageMIME(format), 3, "Cover Album Front", data)
	return nil
}

// ── Info display ──────────────────────────────────────────────────────────────

// displayField defines one row in the -I output.
type displayField struct{ key, label, emoji string }

// infoGroup is a named group of display fields shown together with a header.
type infoGroup struct {
	header string // empty = no header line
	fields []displayField
}

// infoGroups defines the display layout: groups with headers and field order.
var infoGroups = []infoGroup{
	{
		header: "TRACK",
		fields: []displayField{
			{"title", "Title", clr.EmojiMusic},
			{"artist", "Artist", clr.EmojiArtist},
			{"album_artist", "Album Artist", clr.EmojiArtist},
			{"album", "Album", clr.EmojiAlbum},
			{"track", "Track", clr.EmojiDisk},
			{"disc", "Disc", clr.EmojiDisk},
			{"date", "Date", clr.EmojiCalendar},
			{"genre", "Genre", clr.EmojiTag},
		},
	},
	{
		header: "CREDITS",
		fields: []displayField{
			{"original_artist", "Original Artist", clr.EmojiArtist},
			{"composer", "Composer", clr.EmojiArtist},
			{"solo_artist", "Solo Artist", clr.EmojiArtist},
			{"remix", "Remix By", clr.EmojiMusic},
		},
	},
	{
		header: "DETAILS",
		fields: []displayField{
			{"bpm", "BPM", clr.EmojiBpm},
			{"key", "Key", clr.EmojiKey},
			{"subtitle", "Subtitle", clr.EmojiMusic},
			{"group", "Group", clr.EmojiGroup},
			{"length", "Length", clr.EmojiInfo},
			{"media", "Media", clr.EmojiMedia},
		},
	},
	{
		header: "RELEASE",
		fields: []displayField{
			{"publisher", "Publisher", clr.EmojiPublish},
			{"copyright", "Copyright", "©"},
			{"isrc", "ISRC", clr.EmojiBarcode},
			{"barcode", "Barcode", clr.EmojiBarcode},
			{"catalognumber", "Catalog Number", clr.EmojiCatalog},
			{"country", "Country", clr.EmojiCountry},
			{"url", "URL", clr.EmojiURL},
		},
	},
	{
		header: "ENCODING",
		fields: []displayField{
			{"encodedby", "Encoded By", clr.EmojiInfo},
			{"cover_name", "Cover Name", clr.EmojiCover},
		},
	},
	{
		header: "TEXT",
		fields: []displayField{
			{"lyric_name", "Lyric Name", clr.EmojiLyric},
			{"comment", "Comment", clr.EmojiTag},
			{"lyric", "Lyric Preview", clr.EmojiLyric},
		},
	},
}

// PrintInfo prints all tag fields for a file in a clean, aligned, grouped layout.
func PrintInfo(path string) error {
	tw := clr.TermWidth()

	fmt.Printf("\n%s %s\n", clr.EmojiFile, clr.File(path))
	fmt.Println(clr.Sep(tw))

	t, err := id3.Open(path)
	if err != nil {
		return err
	}
	fields := t.AllFields()

	// ── Compute max label display-width across ALL non-empty fields ───────────
	// This ensures the value column starts at the same column for every row.
	maxLabelW := 0
	for _, g := range infoGroups {
		for _, f := range g.fields {
			if fields[f.key] == "" {
				continue
			}
			w := clr.DisplayWidth(f.label + ":")
			if w > maxLabelW {
				maxLabelW = w
			}
		}
	}
	// Also account for "Cover:" which is rendered separately
	coverData, coverMIME := t.GetCover()
	if len(coverData) > 0 {
		w := clr.DisplayWidth("Cover:")
		if w > maxLabelW {
			maxLabelW = w
		}
	}
	// Add 1 space of breathing room after the colon
	labelColW := maxLabelW + 1

	// ── Emoji column: always 2 terminal-cell wide + 1 space gap ──────────────
	const emojiColW = 2 // all our emojis are full-width (2 cells)

	// ── Print each group ──────────────────────────────────────────────────────
	anyPrinted := false
	for _, g := range infoGroups {
		// Collect rows for this group that have non-empty values
		type row struct{ emoji, label, value string }
		var rows []row
		for _, f := range g.fields {
			v := fields[f.key]
			if v == "" {
				continue
			}
			if f.key == "lyric" {
				r := []rune(v)
				if len(r) > 100 {
					v = string(r[:100]) + "…"
				}
			}
			rows = append(rows, row{f.emoji, f.label, v})
		}
		if len(rows) == 0 {
			continue
		}

		// Group header
		if anyPrinted {
			fmt.Println()
		}
		if g.header != "" {
			fmt.Printf("  %s\n", clr.DimText("── "+g.header+" "+strings.Repeat("─", tw-6-len(g.header))))
		}

		for _, r := range rows {
			// Emoji: pad to emojiColW display columns
			emojiCell := clr.PadRight(r.emoji, emojiColW)
			// Label: pad to labelColW display columns, then style
			labelText := r.label + ":"
			labelCell := clr.Label(clr.PadRight(labelText, labelColW))
			// Value
			valueCell := clr.Hex(r.value, clr.Palette.Value)

			fmt.Printf("  %s  %s  %s\n", emojiCell, labelCell, valueCell)
		}
		anyPrinted = true
	}

	// ── Cover art row (special: value is computed, not from fields map) ───────
	if len(coverData) > 0 {
		info := coverMIME
		cfg, format, imgErr := image.DecodeConfig(bytes.NewReader(coverData))
		if imgErr == nil {
			info = fmt.Sprintf("%s  %d × %d px  %s",
				strings.ToUpper(format),
				cfg.Width, cfg.Height,
				formatBytes(len(coverData)),
			)
		}
		if anyPrinted {
			fmt.Println()
			fmt.Printf("  %s\n", clr.DimText("── ARTWORK "+strings.Repeat("─", tw-12)))
		}
		emojiCell := clr.PadRight(clr.EmojiCover, emojiColW)
		labelCell := clr.Label(clr.PadRight("Cover:", labelColW))
		fmt.Printf("  %s  %s  %s\n", emojiCell, labelCell, clr.Hex(info, clr.Palette.Value))
	}

	fmt.Println()
	fmt.Println(clr.Sep(tw))
	return nil
}

// formatBytes formats a byte count as a human-readable string.
func formatBytes(n int) string {
	switch {
	case n >= 1024*1024:
		return fmt.Sprintf("%.1f MB", float64(n)/(1024*1024))
	case n >= 1024:
		return fmt.Sprintf("%.1f KB", float64(n)/1024)
	default:
		return fmt.Sprintf("%d B", n)
	}
}

// ── Cover extraction ──────────────────────────────────────────────────────────

// ExtractCover saves embedded cover art to outputBase.<ext> and returns the path.
func ExtractCover(path, outputBase string) (string, error) {
	t, err := id3.Open(path)
	if err != nil {
		return "", err
	}
	data, mime := t.GetCover()
	if len(data) == 0 {
		return "", fmt.Errorf("no cover art in %q", path)
	}
	ext := mimelist.ExtFromMIME(mime)
	out := outputBase + "." + ext
	if err := os.WriteFile(out, data, 0o644); err != nil {
		return "", err
	}
	return out, nil
}

// ── Consistency check ─────────────────────────────────────────────────────────

// ConsistencyReport holds unique values per field across a file set.
type ConsistencyReport struct {
	Artists, Albums, AlbumArtists, OrigArtists []string
}

// CheckConsistency reads artist/album fields across all files and reports
// which fields have more than one distinct value (indicating a mixed batch).
func CheckConsistency(files []string) ConsistencyReport {
	sets := [4]map[string]bool{{}, {}, {}, {}}
	keys := [4]string{"artist", "album", "album_artist", "original_artist"}
	for _, f := range files {
		tags, err := GetAll(f)
		if err != nil {
			continue
		}
		for i, key := range keys {
			if v := tags[key]; v != "" {
				sets[i][v] = true
			}
		}
	}
	uniq := func(m map[string]bool) []string {
		out := make([]string, 0, len(m))
		for k := range m {
			out = append(out, k)
		}
		sort.Strings(out)
		return out
	}
	return ConsistencyReport{
		Artists:     uniq(sets[0]),
		Albums:      uniq(sets[1]),
		AlbumArtists: uniq(sets[2]),
		OrigArtists:  uniq(sets[3]),
	}
}

// ── LicFace presets ───────────────────────────────────────────────────────────

// ApplyLicFace fills ts with values from the named LicFace preset.
// ts.Artist must be set before calling.
func ApplyLicFace(ts *TagSet, presetID int, year string) {
	p := config.Get(presetID)
	if p == nil {
		return
	}
	ts.Comment = config.Comment
	ts.EncodedBy = config.EncodedBy
	ts.Copyright = year
	if p.SetDate {
		ts.Date = year
	}
	if p.SetPublisher {
		ts.Publisher = config.Publisher
	}
	if p.SetURL {
		ts.URL = config.URL
	}
	if p.SetOriginalArtist {
		ts.OrigArtist = ts.Artist
	}
	if p.SetAlbumArtist {
		ts.AlbumArtist = ts.Artist
	}
	if p.SetGroup {
		ts.Group = ts.Artist
	}
	if p.SetComposer {
		ts.Composer = ts.Artist
	}
	if p.ClearISRC {
		ts.ISRC = "clear"
	}
	if ts.Disc == "" {
		ts.Disc = config.DefaultDisc
	}
}

// ── Rename helpers ────────────────────────────────────────────────────────────

// RenameByTitle renames path to "NN. Title.mp3" in the same directory.
func RenameByTitle(path, track, title string, test bool) (string, error) {
	dir := filepath.Dir(path)
	num := strings.Split(track, "/")[0]
	newName := fmt.Sprintf("%s. %s.mp3", num, title)
	newPath := filepath.Join(dir, newName)
	fmt.Printf("%s %s%s%s\n", clr.EmojiRename, clr.OldVal(filepath.Base(path)), clr.Arrow(), clr.NewVal(newName))
	if test {
		return newPath, nil
	}
	return newPath, os.Rename(path, newPath)
}

// RenameByFile extracts track+title from the filename and renames it.
func RenameByFile(path, pattern string, test bool) (string, error) {
	base := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	var num, title string
	if pattern == "-" {
		re := regexp.MustCompile(`^(\d+)\s*-\s*(.+)$`)
		if m := re.FindStringSubmatch(base); len(m) == 3 {
			num, title = m[1], strings.TrimSpace(m[2])
		}
	} else {
		re := regexp.MustCompile(`^(\d+)\.\s*(.+)$`)
		if m := re.FindStringSubmatch(base); len(m) == 3 {
			num, title = m[1], strings.TrimSpace(m[2])
		}
	}
	if num == "" {
		return path, fmt.Errorf("cannot parse track/title from filename %q", base)
	}
	if len(num) == 1 {
		num = "0" + num
	}
	newName := fmt.Sprintf("%s. %s.mp3", num, title)
	newPath := filepath.Join(filepath.Dir(path), newName)
	fmt.Printf("%s %s%s%s\n", clr.EmojiRename, clr.OldVal(filepath.Base(path)), clr.Arrow(), clr.NewVal(newName))
	if test {
		return newPath, nil
	}
	return newPath, os.Rename(path, newPath)
}
