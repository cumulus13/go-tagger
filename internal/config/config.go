// Package config provides LicFace presets and build-time version info.
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

// LicFacePreset describes one of the four built-in tagging presets.
type LicFacePreset struct {
	ID                int
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

// Presets is the ordered list of LicFace presets.
var Presets = []LicFacePreset{
	{0, "Minimal: original-artist, group, comment, copyright, date, encodedby, disc",
		true, false, false, true, false, false, true, false},
	{1, "Standard: adds publisher",
		false, false, false, false, false, true, true, false},
	{2, "Full: publisher, album-artist, original-artist, group, composer, URL",
		true, true, true, true, true, true, true, false},
	{3, "Full (no date, clear ISRC)",
		true, true, true, true, true, true, false, true},
}

// Get returns the preset for the given ID or nil.
func Get(id int) *LicFacePreset {
	for i := range Presets {
		if Presets[i].ID == id {
			return &Presets[i]
		}
	}
	return nil
}

// Standard LicFace values.
const (
	Comment    = "LICFACE (licface@yahoo.com)"
	URL        = "licface@yahoo.com"
	Publisher  = "LICFACE"
	EncodedBy  = "BLACKID"
	DefaultDisc = "01/01"
)
