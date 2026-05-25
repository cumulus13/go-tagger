package color_test

import (
	"strings"
	"testing"

	clr "github.com/cumulus13/go-tagger/internal/color"
)

func TestParseHex(t *testing.T) {
	tests := []struct {
		input   string
		wantR   uint8
		wantG   uint8
		wantB   uint8
		wantErr bool
	}{
		{"#ff5733", 255, 87, 51, false},
		{"ff5733", 255, 87, 51, false},
		{"#FF5733", 255, 87, 51, false},
		{"#abc", 170, 187, 204, false},   // shorthand
		{"abc", 170, 187, 204, false},    // shorthand no hash
		{"#000000", 0, 0, 0, false},
		{"#ffffff", 255, 255, 255, false},
		{"invalid", 0, 0, 0, true},
		{"#xyz123", 0, 0, 0, true},
		{"#12345", 0, 0, 0, true}, // wrong length
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			rgb, err := clr.ParseHex(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Errorf("ParseHex(%q) expected error, got nil", tt.input)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseHex(%q) unexpected error: %v", tt.input, err)
			}
			if rgb.R != tt.wantR || rgb.G != tt.wantG || rgb.B != tt.wantB {
				t.Errorf("ParseHex(%q) = RGB{%d,%d,%d}, want {%d,%d,%d}",
					tt.input, rgb.R, rgb.G, rgb.B, tt.wantR, tt.wantG, tt.wantB)
			}
		})
	}
}

func TestHexNoColor(t *testing.T) {
	orig := clr.NoColor
	clr.NoColor = true
	defer func() { clr.NoColor = orig }()

	text := "hello"
	got := clr.Hex(text, "#ff0000")
	if got != text {
		t.Errorf("Hex() with NoColor=true should return plain text, got %q", got)
	}
}

func TestHexContainsESC(t *testing.T) {
	orig := clr.NoColor
	clr.NoColor = false
	defer func() { clr.NoColor = orig }()

	got := clr.Hex("hello", "#ff5733")
	if !strings.Contains(got, "\033[") {
		t.Errorf("Hex() should contain ANSI escape, got %q", got)
	}
	if !strings.Contains(got, "hello") {
		t.Errorf("Hex() should contain original text, got %q", got)
	}
}

func TestGradientText(t *testing.T) {
	orig := clr.NoColor
	clr.NoColor = false
	defer func() { clr.NoColor = orig }()

	got := clr.GradientText("gradient", "#ff0000", "#0000ff")
	if !strings.Contains(got, "g") {
		t.Errorf("GradientText should contain original chars")
	}
	if !strings.Contains(got, "\033[") {
		t.Errorf("GradientText should contain ANSI codes")
	}
}

func TestGradientTextNoColor(t *testing.T) {
	orig := clr.NoColor
	clr.NoColor = true
	defer func() { clr.NoColor = orig }()

	text := "gradient"
	got := clr.GradientText(text, "#ff0000", "#0000ff")
	if got != text {
		t.Errorf("GradientText with NoColor should return plain text")
	}
}

func TestTagChange(t *testing.T) {
	orig := clr.NoColor
	clr.NoColor = true
	defer func() { clr.NoColor = orig }()

	got := clr.TagChange("title", "Old Title", "New Title")
	if !strings.Contains(got, "TITLE") {
		t.Errorf("TagChange should contain uppercased label")
	}
	if !strings.Contains(got, "Old Title") {
		t.Errorf("TagChange should contain old value")
	}
	if !strings.Contains(got, "New Title") {
		t.Errorf("TagChange should contain new value")
	}
}

func TestTagSame(t *testing.T) {
	orig := clr.NoColor
	clr.NoColor = true
	defer func() { clr.NoColor = orig }()

	got := clr.TagSame("artist", "Artist Name")
	if !strings.Contains(got, "ARTIST") {
		t.Errorf("TagSame should contain uppercased label")
	}
	if !strings.Contains(got, "Artist Name") {
		t.Errorf("TagSame should contain value")
	}
}

func TestAttrs(t *testing.T) {
	orig := clr.NoColor
	clr.NoColor = false
	defer func() { clr.NoColor = orig }()

	got := clr.Hex("bold", "#ffffff", clr.AttrBold)
	if !strings.Contains(got, "\033[1m") {
		t.Errorf("AttrBold should produce \\033[1m, got %q", got)
	}
}

func TestSep(t *testing.T) {
	got := clr.Sep(10)
	// Should contain 10 instances of the separator rune
	plain := strings.ReplaceAll(got, "\033[0m", "")
	// Strip all ANSI codes for counting
	stripped := ""
	inEsc := false
	for _, r := range plain {
		if r == '\033' {
			inEsc = true
		} else if inEsc && r == 'm' {
			inEsc = false
		} else if !inEsc {
			stripped += string(r)
		}
	}
	if len([]rune(stripped)) != 10 {
		t.Errorf("Sep(10) should have 10 chars, got %d: %q", len([]rune(stripped)), stripped)
	}
}

func TestEmojiConstants(t *testing.T) {
	// Just ensure they're non-empty valid strings
	emojis := []string{
		clr.EmojiMusic, clr.EmojiTag, clr.EmojiDisk,
		clr.EmojiArtist, clr.EmojiAlbum, clr.EmojiFile,
		clr.EmojiOK, clr.EmojiWarn, clr.EmojiError,
	}
	for _, e := range emojis {
		if e == "" {
			t.Errorf("Emoji constant should not be empty")
		}
	}
}
