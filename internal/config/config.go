// Package config provides tagging presets and build-time version info.
package config

import "fmt"

// Build-time variables injected via -ldflags.
var (
	Version   = "dev"
	BuildDate = "unknown"
	GitCommit = "none"
)

// AppInfo returns a one-line version string.
func AppInfo() string {
	return fmt.Sprintf("go-tagger %s (commit:%s built:%s)", Version, GitCommit, BuildDate)
}

// ── Preset definition ─────────────────────────────────────────────────────────

// Preset describes one named tagging preset.
type Preset struct {
	ID                int
	Group             string // "licface" or "cumulus13"
	Description       string
	SetOriginalArtist bool
	SetAlbumArtist    bool
	SetComposer       bool
	SetGroup          bool
	SetURL            bool
	SetPublisher      bool
	SetDate           bool
	ClearISRC         bool
}

// AllPresets is the complete ordered list of built-in presets.
// IDs 0-3: licface  |  IDs 10-13: cumulus13
var AllPresets = []Preset{
	// ── licface presets (original) ────────────────────────────────────────────
	{
		ID: 0, Group: "licface",
		Description:       "Minimal — original-artist, group, comment, copyright, date, encodedby, disc",
		SetOriginalArtist: true,
		SetGroup:          true,
		SetDate:           true,
	},
	{
		ID: 1, Group: "licface",
		Description:  "Standard — adds publisher",
		SetPublisher: true,
		SetDate:      true,
	},
	{
		ID: 2, Group: "licface",
		Description:       "Full — publisher, album-artist, original-artist, group, composer, URL",
		SetOriginalArtist: true,
		SetAlbumArtist:    true,
		SetComposer:       true,
		SetGroup:          true,
		SetURL:            true,
		SetPublisher:      true,
		SetDate:           true,
	},
	{
		ID: 3, Group: "licface",
		Description:       "Full (no date, clear ISRC)",
		SetOriginalArtist: true,
		SetAlbumArtist:    true,
		SetComposer:       true,
		SetGroup:          true,
		SetURL:            true,
		SetPublisher:      true,
		ClearISRC:         true,
	},

	// ── cumulus13 presets ─────────────────────────────────────────────────────
	// Same structural logic as licface but with cumulus13 contact info.
	{
		ID: 10, Group: "cumulus13",
		Description:       "Minimal — original-artist, group, comment, copyright, date, encodedby, disc",
		SetOriginalArtist: true,
		SetGroup:          true,
		SetDate:           true,
	},
	{
		ID: 11, Group: "cumulus13",
		Description:  "Standard — adds publisher",
		SetPublisher: true,
		SetDate:      true,
	},
	{
		ID: 12, Group: "cumulus13",
		Description:       "Full — publisher, album-artist, original-artist, group, composer, URL",
		SetOriginalArtist: true,
		SetAlbumArtist:    true,
		SetComposer:       true,
		SetGroup:          true,
		SetURL:            true,
		SetPublisher:      true,
		SetDate:           true,
	},
	{
		ID: 13, Group: "cumulus13",
		Description:       "Full (no date, clear ISRC)",
		SetOriginalArtist: true,
		SetAlbumArtist:    true,
		SetComposer:       true,
		SetGroup:          true,
		SetURL:            true,
		SetPublisher:      true,
		ClearISRC:         true,
	},
}

// Get returns the Preset for the given ID, or nil if not found.
func Get(id int) *Preset {
	for i := range AllPresets {
		if AllPresets[i].ID == id {
			return &AllPresets[i]
		}
	}
	return nil
}

// PresetsForGroup returns all presets belonging to group ("licface" or "cumulus13").
func PresetsForGroup(group string) []Preset {
	var out []Preset
	for _, p := range AllPresets {
		if p.Group == group {
			out = append(out, p)
		}
	}
	return out
}

// ── licface contact values ────────────────────────────────────────────────────

const (
	LicfaceComment   = "LICFACE (licface@yahoo.com)"
	LicfaceURL       = "licface@yahoo.com"
	LicfacePublisher = "LICFACE"
	LicfaceEncodedBy = "BLACKID"
)

// ── cumulus13 contact values ──────────────────────────────────────────────────

const (
	Cumulus13Comment   = "cumulus13 (cumulus13@gmail.com)"
	Cumulus13URL       = "cumulus13@gmail.com"
	Cumulus13Publisher = "cumulus13"
	Cumulus13EncodedBy = "cumulus13"
)

// ── Shared defaults ───────────────────────────────────────────────────────────

const DefaultDisc = "01/01"

// ContactForGroup returns the comment, url, publisher, and encodedby strings
// for the given preset group.
func ContactForGroup(group string) (comment, url, publisher, encodedby string) {
	switch group {
	case "cumulus13":
		return Cumulus13Comment, Cumulus13URL, Cumulus13Publisher, Cumulus13EncodedBy
	default: // "licface"
		return LicfaceComment, LicfaceURL, LicfacePublisher, LicfaceEncodedBy
	}
}

// Legacy aliases so existing tagger.go code compiles unchanged.
// (tagger.go calls config.Comment, config.URL, etc. for licface presets)
const (
	Comment   = LicfaceComment
	URL       = LicfaceURL
	Publisher = LicfacePublisher
	EncodedBy = LicfaceEncodedBy
)
