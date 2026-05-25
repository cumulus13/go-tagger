// Package color provides rich terminal color output with true-color hex support,
// gradient text, display-width-aware padding, and emoji constants.
// Zero external dependencies.
package color

import (
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"
	"unicode"
)

// NoColor disables all ANSI output when true.
// Automatically set when NO_COLOR env var is present or TERM=dumb.
var NoColor = os.Getenv("NO_COLOR") != "" || os.Getenv("TERM") == "dumb"

// Attr is an ANSI SGR attribute code.
type Attr int

const (
	AttrBold      Attr = 1
	AttrDim       Attr = 2
	AttrItalic    Attr = 3
	AttrUnderline Attr = 4
	AttrBlink     Attr = 5
	AttrStrike    Attr = 9
)

const esc = "\033["
const reset = "\033[0m"

// ── RGB ──────────────────────────────────────────────────────────────────────

// RGB holds 8-bit color components.
type RGB struct{ R, G, B uint8 }

// ParseHex parses "#rgb", "#rrggbb", "rgb", or "rrggbb" strings.
func ParseHex(h string) (RGB, error) {
	h = strings.TrimSpace(strings.TrimPrefix(h, "#"))
	if len(h) == 3 {
		h = string([]byte{h[0], h[0], h[1], h[1], h[2], h[2]})
	}
	if len(h) != 6 {
		return RGB{}, fmt.Errorf("invalid hex color: %q", h)
	}
	r, e1 := strconv.ParseUint(h[0:2], 16, 8)
	g, e2 := strconv.ParseUint(h[2:4], 16, 8)
	b, e3 := strconv.ParseUint(h[4:6], 16, 8)
	if e1 != nil || e2 != nil || e3 != nil {
		return RGB{}, fmt.Errorf("invalid hex color: %q", h)
	}
	return RGB{uint8(r), uint8(g), uint8(b)}, nil
}

func fgTrue(r, g, b uint8) string { return fmt.Sprintf("\033[38;2;%d;%d;%dm", r, g, b) }
func bgTrue(r, g, b uint8) string { return fmt.Sprintf("\033[48;2;%d;%d;%dm", r, g, b) }

func attrSeq(attrs []Attr) string {
	if len(attrs) == 0 {
		return ""
	}
	parts := make([]string, len(attrs))
	for i, a := range attrs {
		parts[i] = strconv.Itoa(int(a))
	}
	return esc + strings.Join(parts, ";") + "m"
}

// ── Core rendering ────────────────────────────────────────────────────────────

// Hex applies a hex foreground color and optional attributes to text.
func Hex(text, fg string, attrs ...Attr) string {
	if NoColor || text == "" {
		return text
	}
	var sb strings.Builder
	if len(attrs) > 0 {
		sb.WriteString(attrSeq(attrs))
	}
	if rgb, err := ParseHex(fg); err == nil {
		sb.WriteString(fgTrue(rgb.R, rgb.G, rgb.B))
	}
	sb.WriteString(text)
	sb.WriteString(reset)
	return sb.String()
}

// HexBg applies hex fg + bg colors with optional attributes.
func HexBg(text, fg, bg string, attrs ...Attr) string {
	if NoColor || text == "" {
		return text
	}
	var sb strings.Builder
	if len(attrs) > 0 {
		sb.WriteString(attrSeq(attrs))
	}
	if rgb, err := ParseHex(fg); err == nil {
		sb.WriteString(fgTrue(rgb.R, rgb.G, rgb.B))
	}
	if rgb, err := ParseHex(bg); err == nil {
		sb.WriteString(bgTrue(rgb.R, rgb.G, rgb.B))
	}
	sb.WriteString(text)
	sb.WriteString(reset)
	return sb.String()
}

// ── Display-width-aware padding ───────────────────────────────────────────────

// runeDisplayWidth returns the number of terminal columns a rune occupies.
// Returns 0 for zero-width/combining/modifier chars, 2 for wide emoji/CJK, 1 for others.
func runeDisplayWidth(r rune) int {
	switch {
	// Explicit zero-width
	case r == 0:
		return 0
	// Zero Width Joiner, Zero Width Non-Joiner, Zero Width Space
	case r == 0x200B || r == 0x200C || r == 0x200D:
		return 0
	// Variation selectors VS-1..VS-16 (text/emoji presentation)
	case r >= 0xFE00 && r <= 0xFE0F:
		return 0
	// Variation selectors supplement
	case r >= 0xE0100 && r <= 0xE01EF:
		return 0
	// Combining diacritical marks
	case r >= 0x0300 && r <= 0x036F:
		return 0
	// Emoji skin tone modifiers
	case r >= 0x1F3FB && r <= 0x1F3FF:
		return 0
	// Tags block (used in flags etc.)
	case r >= 0xE0000 && r <= 0xE007F:
		return 0
	}
	// Wide characters: CJK, emoji, etc.
	if r >= 0x1100 {
		if r <= 0x115F || // Hangul Jamo
			r == 0x2329 || r == 0x232A || // brackets
			(r >= 0x2E80 && r <= 0x303E) || // CJK Radicals
			(r >= 0x3040 && r <= 0xA4CF) || // Japanese/Korean
			(r >= 0xA960 && r <= 0xA97F) ||
			(r >= 0xAC00 && r <= 0xD7FF) || // Hangul syllables
			(r >= 0xF900 && r <= 0xFAFF) || // CJK Compatibility
			(r >= 0xFE10 && r <= 0xFE1F) ||
			(r >= 0xFE30 && r <= 0xFE6F) ||
			(r >= 0xFF00 && r <= 0xFF60) || // Fullwidth forms
			(r >= 0xFFE0 && r <= 0xFFE6) ||
			(r >= 0x1F004 && r <= 0x1FFFD) { // Emoji & Symbols block
			return 2
		}
	}
	if unicode.IsControl(r) {
		return 0
	}
	return 1
}

// DisplayWidth returns the number of terminal columns s occupies,
// ignoring ANSI escape sequences.
func DisplayWidth(s string) int {
	w := 0
	inEsc := false
	for _, r := range s {
		if r == '\033' {
			inEsc = true
			continue
		}
		if inEsc {
			if r == 'm' {
				inEsc = false
			}
			continue
		}
		w += runeDisplayWidth(r)
	}
	return w
}

// PadRight pads s with spaces on the right to reach width terminal columns.
// It correctly accounts for wide emoji/CJK characters.
// If s is already >= width columns, it is returned unchanged.
func PadRight(s string, width int) string {
	cur := DisplayWidth(s)
	if cur >= width {
		return s
	}
	return s + strings.Repeat(" ", width-cur)
}

// ── Gradient ─────────────────────────────────────────────────────────────────

func hexToLinear(h string) (float64, float64, float64, bool) {
	rgb, err := ParseHex(h)
	if err != nil {
		return 0, 0, 0, false
	}
	toLinear := func(c uint8) float64 {
		v := float64(c) / 255.0
		if v <= 0.04045 {
			return v / 12.92
		}
		return math.Pow((v+0.055)/1.055, 2.4)
	}
	return toLinear(rgb.R), toLinear(rgb.G), toLinear(rgb.B), true
}

func linearToSRGB(v float64) uint8 {
	if v <= 0.0031308 {
		v *= 12.92
	} else {
		v = 1.055*math.Pow(v, 1.0/2.4) - 0.055
	}
	if v < 0 {
		v = 0
	} else if v > 1 {
		v = 1
	}
	return uint8(math.Round(v * 255))
}

// GradientText renders text with a perceptual linear gradient from fromHex to toHex.
func GradientText(text, fromHex, toHex string) string {
	if NoColor || text == "" {
		return text
	}
	r1, g1, b1, ok1 := hexToLinear(fromHex)
	r2, g2, b2, ok2 := hexToLinear(toHex)
	if !ok1 || !ok2 {
		return text
	}
	runes := []rune(text)
	n := len(runes)
	var sb strings.Builder
	for i, ch := range runes {
		t := 0.0
		if n > 1 {
			t = float64(i) / float64(n-1)
		}
		lerp := func(a, b float64) float64 { return a + (b-a)*t }
		sb.WriteString(fgTrue(
			linearToSRGB(lerp(r1, r2)),
			linearToSRGB(lerp(g1, g2)),
			linearToSRGB(lerp(b1, b2)),
		))
		sb.WriteRune(ch)
	}
	sb.WriteString(reset)
	return sb.String()
}

// ── Palette ───────────────────────────────────────────────────────────────────

// Palette is the go-tagger terminal color palette (Tokyo Night inspired).
var Palette = struct {
	Label, Value, Old, New, Arrow, OK, Warn, Error, File, Sep, Dim, Heading string
}{
	Label:   "#7dcfff",
	Value:   "#e0af68",
	Old:     "#f7768e",
	New:     "#9ece6a",
	Arrow:   "#bb9af7",
	OK:      "#73daca",
	Warn:    "#e0af68",
	Error:   "#f7768e",
	File:    "#ffd700",
	Sep:     "#3d59a1",
	Dim:     "#565f89",
	Heading: "#c0caf5",
}

// ── Semantic helpers ──────────────────────────────────────────────────────────

func Label(s string) string   { return Hex(s, Palette.Label, AttrBold) }
func OldVal(s string) string  { return Hex(s, Palette.Old) }
func NewVal(s string) string  { return Hex(s, Palette.New) }
func Arrow() string           { return Hex(" → ", Palette.Arrow, AttrBold) }
func OK(s string) string      { return Hex(s, Palette.OK, AttrBold) }
func Warn(s string) string    { return Hex(s, Palette.Warn, AttrBold) }
func Err(s string) string     { return Hex(s, Palette.Error, AttrBold) }
func File(s string) string    { return Hex(s, Palette.File) }
func Heading(s string) string { return Hex(s, Palette.Heading, AttrBold, AttrUnderline) }
func DimText(s string) string { return Hex(s, Palette.Dim) }

// Sep returns a separator line of exactly width terminal columns.
func Sep(width int) string {
	if width <= 0 {
		width = 80
	}
	return Hex(strings.Repeat("─", width), Palette.Sep)
}

// TagChange formats: LABEL             old → new
// Uses display-width-aware padding so values always align regardless of emoji.
func TagChange(label, old, newVal string) string {
	lbl := Label(PadRight(strings.ToUpper(label)+":", 20))
	return fmt.Sprintf("%s %s%s%s", lbl, OldVal(truncate(old, 45)), Arrow(), NewVal(truncate(newVal, 45)))
}

// TagSame formats: LABEL             value  (no change)
func TagSame(label, val string) string {
	lbl := Label(PadRight(strings.ToUpper(label)+":", 20))
	return fmt.Sprintf("%s %s", lbl, Hex(truncate(val, 70), Palette.OK))
}

func truncate(s string, maxRunes int) string {
	r := []rune(s)
	if len(r) > maxRunes {
		return string(r[:maxRunes]) + "…"
	}
	return s
}

// TermWidth returns the terminal width (fixed 100; override with COLUMNS env).
func TermWidth() int {
	if c := os.Getenv("COLUMNS"); c != "" {
		if n, err := strconv.Atoi(c); err == nil && n > 20 {
			return n
		}
	}
	return 100
}

// ── Emoji constants ───────────────────────────────────────────────────────────

const (
	EmojiMusic    = "🎵"
	EmojiTag      = "🏷"
	EmojiDisk     = "💿"
	EmojiArtist   = "🎤"
	EmojiAlbum    = "📀"
	EmojiFile     = "📄"
	EmojiFolder   = "📂"
	EmojiCover    = "🖼"
	EmojiOK       = "✅"
	EmojiWarn     = "⚠️"
	EmojiError    = "❌"
	EmojiSave     = "💾"
	EmojiExtract  = "📤"
	EmojiRename   = "✏️"
	EmojiInfo     = "ℹ️"
	EmojiBpm      = "🥁"
	EmojiKey      = "🎹"
	EmojiLyric    = "📝"
	EmojiCalendar = "📅"
	EmojiBarcode  = "📊"
	EmojiURL      = "🔗"
	EmojiGroup    = "👥"
	EmojiPublish  = "🏢"
	EmojiSearch   = "🔍"
	EmojiTest     = "🧪"
	EmojiMedia    = "📼"
	EmojiCountry  = "🌍"
	EmojiCatalog  = "🗂"
)
