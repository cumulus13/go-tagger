// go-tagger — Professional MP3 ID3v2 Tag Editor
//
// Author  : Hadi Cahyadi <cumulus13@gmail.com>
// Homepage: https://github.com/cumulus13/go-tagger
// License : MIT
package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	clr "github.com/cumulus13/go-tagger/internal/color"
	"github.com/cumulus13/go-tagger/internal/config"
	"github.com/cumulus13/go-tagger/internal/tagger"
)

// ── CLI flags ─────────────────────────────────────────────────────────────────

var (
	// Tag fields
	fTitle       = flag.String("t", "", "Set Title (or path to .txt with one title per line, or 'file')")
	fTrack       = flag.String("tt", "", "Set Track number (e.g. 3 or 03/12)")
	fLenTracks   = flag.Int("ltt", 0, "Override total track count")
	fChangeTrack = flag.Bool("ct", false, "Update total in existing NN/TT track tag")
	fArtist      = flag.String("a", "", "Set Artist")
	fSoloArtist  = flag.String("sa", "", "Set Solo Artist (TXXX:ARTIST)")
	fAlbum       = flag.String("aa", "", "Set Album")
	fAlbumArtist = flag.String("aaa", "", "Set Album Artist (TPE2)")
	fDisc        = flag.String("d", "", "Set Disc number (e.g. 1 or 01/02)")
	fComposer    = flag.String("c", "", "Set Composer")
	fOrigArtist  = flag.String("o", "", "Set Original Artist")
	fComment     = flag.String("cc", "", "Set Comment")
	fCommentDesc = flag.String("cd", "", "Set Comment description")
	fISRC        = flag.String("i", "", "Set ISRC number")
	fBarcode     = flag.String("b", "", "Set Barcode (TXXX:BARCODE)")
	fGenre       = flag.String("g", "", "Set Genre")
	fDate        = flag.String("dd", "", "Set Date/Year")
	fBPM         = flag.String("bb", "", "Set BPM")
	fCopyright   = flag.String("ccc", "", "Set Copyright")
	fEncodedBy   = flag.String("e", "", "Set Encoded By")
	fKey         = flag.String("k", "", "Set Key")
	fLyric       = flag.String("l", "", "Set Lyric (inline text or path to .txt)")
	fLyricName   = flag.String("ln", "", "Set Lyric Name (TEXT frame)")
	fRemix       = flag.String("r", "", "Set Remix/Interpreted By (TPE4)")
	fSubtitle    = flag.String("s", "", "Set Subtitle (TIT3)")
	fURL         = flag.String("u", "", "Set URL (WXXX)")
	fGroup       = flag.String("gg", "", "Set Content Group (TIT1)")
	fPublisher   = flag.String("p", "", "Set Publisher (TPUB)")
	fLength      = flag.String("ll", "", "Set Length in ms (TLEN)")
	fCover       = flag.String("C", "", "Set Cover art from image path")
	fCoverName   = flag.String("Cn", "Cover Album Front", "Set Cover name label (TXXX:_cover)")

	// Actions
	fGetTitle     = flag.Bool("gt", false, "Print track number and title for each file")
	fInfo         = flag.Bool("I", false, "Print full tag info for each file")
	fExtractCover = flag.Bool("ec", false, "Extract embedded cover art to file")
	fTest         = flag.Bool("T", false, "Dry-run: show changes without saving")
	fRename       = flag.String("R", "", "Rename files: 'title' (from tag) or 'file' (from filename)")
	fRenamePattern = flag.String("RP", "", "Rename separator pattern: '-' for 'NN - Title'")
	fLicface      = flag.Int("A", -1, "Apply preset: licface 0-3, cumulus13 10-13 (see -SA)")
	fHelpLicface  = flag.Bool("SA", false, "Show all preset tables (licface + cumulus13) and exit")
	fRecursive    = flag.Bool("rec", false, "Scan directories recursively")
	fNoColor      = flag.Bool("nc", false, "Disable color output")
	fOutputDir    = flag.String("od", "", "Output directory for extracted covers")
	fVersion      = flag.Bool("v", false, "Print version and exit")
)

func main() {
	printBanner()

	flag.Usage = usage
	flag.Parse()

	if *fNoColor {
		clr.NoColor = true
	}
	if *fVersion {
		fmt.Println(clr.Hex(config.AppInfo(), clr.Palette.Heading))
		return
	}
	if *fHelpLicface {
		printLicFaceTable()
		return
	}

	args := flag.Args()
	if len(args) == 0 {
		usage()
		return
	}

	// Resolve MP3 files
	files, err := resolveFiles(args, *fRecursive)
	if err != nil {
		fatal(err)
	}
	if len(files) == 0 {
		fmt.Printf("%s %s\n", clr.EmojiWarn, clr.Warn("No MP3 files found."))
		return
	}

	// ── Info / read-only actions ──────────────────────────────────────────────

	if *fInfo {
		for _, f := range files {
			if err := tagger.PrintInfo(f); err != nil {
				errLine(f, err)
			}
		}
		return
	}

	if *fGetTitle {
		for _, f := range files {
			tags, err := tagger.GetAll(f)
			if err != nil {
				errLine(f, err)
				continue
			}
			fmt.Printf("  %s %s\n",
				clr.Hex(fmt.Sprintf("%-6s", tags["track"]), clr.Palette.Label),
				clr.Hex(tags["title"], clr.Palette.Value),
			)
		}
		return
	}

	if *fExtractCover {
		outDir := *fOutputDir
		for _, f := range files {
			base := "Cover"
			if outDir != "" {
				base = filepath.Join(outDir, "Cover")
			} else {
				base = filepath.Join(filepath.Dir(f), "Cover")
			}
			out, err := tagger.ExtractCover(f, base)
			if err != nil {
				fmt.Printf("%s %s: %v\n", clr.EmojiError, clr.Err("Extract failed"), err)
			} else {
				fmt.Printf("%s %s %s\n", clr.EmojiExtract, clr.OK("Cover extracted:"), clr.File(out))
			}
		}
		return
	}

	// ── Build title/track lists ───────────────────────────────────────────────

	titles, trackNums, err := buildLists(files)
	if err != nil {
		fatal(err)
	}

	// ── Consistency check ─────────────────────────────────────────────────────

	if !promptConsistency(tagger.CheckConsistency(files)) {
		return
	}

	// ── Per-file tag loop ─────────────────────────────────────────────────────

	year := strconv.Itoa(time.Now().Year())
	total := len(files)
	if *fLenTracks > 0 {
		total = *fLenTracks
	}

	for idx, f := range files {
		printFileBanner(f, idx+1, total)
		ts := buildTagSet(idx, total, titles, trackNums)

		// LicFace preset
		if *fLicface >= 0 {
			if ts.Artist == "" {
				tags, _ := tagger.GetAll(f)
				ts.Artist = tags["artist"]
			}
			tagger.ApplyLicFace(&ts, *fLicface, year)
		}

		if err := tagger.Set(f, ts); err != nil {
			errLine(f, err)
			continue
		}

		if *fRename != "" {
			switch *fRename {
			case "title":
				tagger.RenameByTitle(f, ts.Track, ts.Title, *fTest)
			case "file":
				tagger.RenameByFile(f, *fRenamePattern, *fTest)
			}
		}

		fmt.Println(clr.Sep(clr.TermWidth()))
	}
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func resolveFiles(paths []string, recursive bool) ([]string, error) {
	var out []string
	for _, p := range paths {
		fi, err := os.Stat(p)
		if err != nil {
			fmt.Printf("%s %s: %v\n", clr.EmojiWarn, clr.Warn("Skip"), err)
			continue
		}
		if fi.IsDir() {
			if recursive {
				filepath.Walk(p, func(path string, info os.FileInfo, err error) error {
					if err != nil || info.IsDir() {
						return nil
					}
					if strings.EqualFold(filepath.Ext(path), ".mp3") {
						out = append(out, path)
					}
					return nil
				})
			} else {
				files, err := tagger.ScanDir(p)
				if err != nil {
					return nil, err
				}
				out = append(out, files...)
			}
		} else if strings.EqualFold(filepath.Ext(p), ".mp3") {
			out = append(out, p)
		}
	}
	return out, nil
}

func buildLists(files []string) (titles, trackNums []string, err error) {
	total := len(files)

	// "file" = read titles from existing tags
	if *fTitle == "file" {
		for _, f := range files {
			tags, _ := tagger.GetAll(f)
			titles = append(titles, tags["title"])
		}
		return
	}

	// Title from .txt file
	if *fTitle != "" {
		if _, statErr := os.Stat(*fTitle); statErr == nil {
			lines, readErr := readLines(*fTitle)
			if readErr != nil {
				return nil, nil, readErr
			}
			for _, line := range lines {
				info := tagger.ParseTitleLine(line)
				titles = append(titles, info.Title)
				if info.Track != "" {
					trackNums = append(trackNums, info.Track)
				}
			}
			if len(trackNums) == len(titles) {
				return
			}
			trackNums = nil
			return
		}
		// Inline single title
		titles = []string{*fTitle}
		if total > 1 {
			return nil, nil, fmt.Errorf("single title provided for %d files; use a text file for batch", total)
		}
	}

	// Track from flag
	if *fTrack != "" {
		trackNums = []string{*fTrack}
	}

	// Fallback: read from existing tags
	if len(titles) == 0 {
		for _, f := range files {
			tags, _ := tagger.GetAll(f)
			titles = append(titles, tags["title"])
		}
	}
	return
}

func buildTagSet(idx, total int, titles, trackNums []string) tagger.TagSet {
	ts := tagger.TagSet{
		Artist:      *fArtist,
		SoloArtist:  *fSoloArtist,
		Album:       *fAlbum,
		AlbumArtist: *fAlbumArtist,
		OrigArtist:  *fOrigArtist,
		Composer:    *fComposer,
		Comment:     *fComment,
		CommentDesc: *fCommentDesc,
		ISRC:        *fISRC,
		Barcode:     *fBarcode,
		Genre:       *fGenre,
		Date:        *fDate,
		BPM:         *fBPM,
		Copyright:   *fCopyright,
		EncodedBy:   *fEncodedBy,
		Key:         *fKey,
		LyricName:   *fLyricName,
		Remix:       *fRemix,
		Subtitle:    *fSubtitle,
		URL:         *fURL,
		Group:       *fGroup,
		Publisher:   *fPublisher,
		Length:      *fLength,
		Cover:       *fCover,
		CoverName:   *fCoverName,
		TestMode:    *fTest,
	}

	if idx < len(titles) {
		ts.Title = titles[idx]
	}

	if *fLyric != "" {
		if _, err := os.Stat(*fLyric); err == nil {
			data, _ := os.ReadFile(*fLyric)
			ts.Lyric = string(data)
		} else {
			ts.Lyric = *fLyric
		}
	}

	trackTotal := total
	if *fLenTracks > 0 {
		trackTotal = *fLenTracks
	}

	if idx < len(trackNums) {
		ts.Track = tagger.FormatTrack(trackNums[idx], trackTotal, *fChangeTrack)
	} else {
		ts.Track = tagger.FormatTrack(strconv.Itoa(idx+1), trackTotal, *fChangeTrack)
	}

	if *fDisc != "" {
		ts.Disc = tagger.FormatDisc(*fDisc)
	}
	return ts
}

func promptConsistency(r tagger.ConsistencyReport) bool {
	warned := false
	warnField := func(name string, vals []string) {
		if len(vals) > 1 {
			fmt.Printf("%s %s %s\n",
				clr.EmojiWarn,
				clr.Warn(fmt.Sprintf("Multiple %s detected:", name)),
				clr.OldVal(strings.Join(vals, ", ")),
			)
			warned = true
		}
	}
	warnField("ARTISTS", r.Artists)
	warnField("ALBUMS", r.Albums)
	warnField("ALBUM ARTISTS", r.AlbumArtists)
	warnField("ORIGINAL ARTISTS", r.OrigArtists)
	if !warned {
		return true
	}
	fmt.Printf("\n%s Press %s to continue, %s to quit: ",
		clr.EmojiWarn, clr.OK("Enter"), clr.Err("q/x"),
	)
	sc := bufio.NewScanner(os.Stdin)
	sc.Scan()
	inp := strings.TrimSpace(sc.Text())
	return inp != "q" && inp != "x"
}

func printFileBanner(path string, idx, total int) {
	fmt.Printf("\n%s [%s/%s] %s\n",
		clr.EmojiFile,
		clr.Hex(strconv.Itoa(idx), clr.Palette.OK),
		clr.Hex(strconv.Itoa(total), clr.Palette.Dim),
		clr.File(path),
	)
	fmt.Println(clr.Sep(clr.TermWidth()))
}

func readLines(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var lines []string
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		if line := strings.TrimSpace(sc.Text()); line != "" {
			lines = append(lines, line)
		}
	}
	return lines, sc.Err()
}

func fatal(err error) {
	fmt.Fprintf(os.Stderr, "%s %s\n", clr.EmojiError, clr.Err(err.Error()))
	os.Exit(1)
}

func errLine(path string, err error) {
	fmt.Printf("%s %s %s\n", clr.EmojiError, clr.File(path), clr.Err(err.Error()))
}

// ── Banner & usage ────────────────────────────────────────────────────────────

func printBanner() {
	lines := []struct{ text, from, to string }{
		{"  ╔═══════════════════════════════════════════════╗", "#7dcfff", "#bb9af7"},
		{"  ║  🎵  go-tagger  ·  MP3 ID3v2 Tag Editor       ║", "#bb9af7", "#9ece6a"},
		{"  ║  👤  Hadi Cahyadi  ·  cumulus13@gmail.com     ║", "#9ece6a", "#73daca"},
		{"  ║  🔗  github.com/cumulus13/go-tagger           ║", "#73daca", "#7dcfff"},
		{"  ╚═══════════════════════════════════════════════╝", "#7dcfff", "#bb9af7"},
	}
	for _, l := range lines {
		fmt.Println(clr.GradientText(l.text, l.from, l.to))
	}
	fmt.Printf("  %s\n\n", clr.DimText(config.AppInfo()))
}

func usage() {
	fmt.Printf("%s\n\n", clr.Heading("USAGE"))
	fmt.Printf("  %s\n\n", clr.Hex("go-tagger [PATH...] [FLAGS]", clr.Palette.Value))

	sections := []struct {
		title string
		items [][2]string
	}{
		{"TAG FLAGS", [][2]string{
			{"-t  <title|file.txt|'file'>", "Set title(s) — inline, from .txt file, or from existing tag"},
			{"-tt <track>", "Set track number (e.g. 3 or 03/12)"},
			{"-ltt <n>", "Override total track count"},
			{"-ct", "Update total in existing NN/TT track"},
			{"-a  <artist>", "Set Artist"},
			{"-sa <artist>", "Set Solo Artist (TXXX:ARTIST)"},
			{"-aa <album>", "Set Album"},
			{"-aaa <artist>", "Set Album Artist (TPE2)"},
			{"-d  <disc>", "Set Disc (e.g. 1 or 01/02)"},
			{"-c  <composer>", "Set Composer"},
			{"-o  <artist>", "Set Original Artist"},
			{"-cc <comment>", "Set Comment"},
			{"-cd <desc>", "Set Comment description"},
			{"-i  <isrc>", "Set ISRC number"},
			{"-b  <barcode>", "Set Barcode (TXXX:BARCODE)"},
			{"-g  <genre>", "Set Genre"},
			{"-dd <year>", "Set Date/Year"},
			{"-bb <bpm>", "Set BPM"},
			{"-ccc <text>", "Set Copyright"},
			{"-e  <text>", "Set Encoded By"},
			{"-k  <key>", "Set Key"},
			{"-l  <text|file.txt>", "Set Lyric (inline or from file)"},
			{"-ln <name>", "Set Lyric Name (TEXT frame)"},
			{"-r  <text>", "Set Remix/Interpreted By (TPE4)"},
			{"-s  <text>", "Set Subtitle (TIT3)"},
			{"-u  <url>", "Set URL (WXXX)"},
			{"-gg <group>", "Set Content Group (TIT1)"},
			{"-p  <publisher>", "Set Publisher (TPUB)"},
			{"-ll <ms>", "Set Length in milliseconds (TLEN)"},
			{"-C  <image>", "Set Cover art from image file (JPEG/PNG/GIF/BMP)"},
			{"-Cn <name>", "Set Cover name label (TXXX:_cover)"},
		}},
		{"ACTION FLAGS", [][2]string{
			{"-gt", "Print track number and title for each file"},
			{"-I", "Print full tag info for each file"},
			{"-ec", "Extract embedded cover art to file"},
			{"-T", "Dry-run: show changes without saving"},
			{"-R  <title|file>", "Rename files: 'title' (from tag) or 'file' (from filename)"},
			{"-RP <pattern>", "Rename separator: '-' for 'NN - Title'"},
			{"-A  <0-3>", "Apply LicFace preset (see -SA for details)"},
			{"-SA", "Show LicFace preset table"},
			{"-rec", "Scan directories recursively"},
			{"-nc", "Disable color output (or set NO_COLOR env var)"},
			{"-od <dir>", "Output directory for extracted covers"},
			{"-v", "Print version and exit"},
		}},
		{"EXAMPLES", [][2]string{
			{"go-tagger song.mp3 -a \"Artist\" -aa \"Album\" -g Pop -dd 2024", ""},
			{"go-tagger ./album/ -t titles.txt -a \"Artist\" -aa \"Album\"", ""},
			{"go-tagger ./album/ -A 2", "Apply full LicFace preset"},
			{"go-tagger song.mp3 -I", "Show all tags"},
			{"go-tagger song.mp3 -ec", "Extract cover art"},
			{"go-tagger ./album/ -R title", "Rename files by title tag"},
			{"go-tagger ./album/ -T -a \"NewArtist\"", "Dry-run: preview changes"},
		}},
	}

	for _, sec := range sections {
		fmt.Printf("\n%s\n", clr.Heading(sec.title))
		for _, item := range sec.items {
			if item[1] == "" {
				fmt.Printf("  %s\n", clr.Hex(item[0], clr.Palette.Value))
			} else {
				fmt.Printf("  %-40s %s\n",
					clr.Hex(item[0], clr.Palette.Value),
					clr.DimText(item[1]),
				)
			}
		}
	}
	fmt.Println()
}

// ── Preset table ──────────────────────────────────────────────────────────────

func printLicFaceTable() {
	tw := clr.TermWidth()

	// Shared table data — same structure for both groups
	type tableRow struct{ tag, v0, v1, v2, v3 string }
	rows := []tableRow{
		{"ARTIST",          "AUTO",   "AUTO",   "AUTO",   "AUTO"},
		{"ALBUM_ARTIST",    "NO",     "NO",     "ARTIST", "ARTIST"},
		{"COMMENT",         "YES",    "YES",    "YES",    "YES"},
		{"COMPOSER",        "NO",     "NO",     "ARTIST", "ARTIST"},
		{"COPYRIGHT",       "YES",    "YES",    "YES",    "YES"},
		{"DATE",            "YES",    "YES",    "YES",    "NO"},
		{"DISC",            "AUTO",   "AUTO",   "AUTO",   "AUTO"},
		{"ENCODEDBY",       "YES",    "YES",    "YES",    "YES"},
		{"ORIGINAL_ARTIST", "ARTIST", "NO",     "ARTIST", "ARTIST"},
		{"GROUP",           "ARTIST", "NO",     "ARTIST", "ARTIST"},
		{"PUBLISHER",       "NO",     "YES",    "YES",    "YES"},
		{"URL",             "NO",     "NO",     "YES",    "YES"},
		{"ISRC",            "—",      "—",      "—",      "CLEAR"},
	}

	styleVal := func(v string, w int) string {
		pad := fmt.Sprintf("%-*s", w, v)
		switch v {
		case "YES":
			return clr.OK(pad)
		case "NO", "—":
			return clr.DimText(pad)
		case "AUTO", "ARTIST":
			return clr.Warn(pad)
		case "CLEAR":
			return clr.Err(pad)
		default:
			return clr.Hex(pad, clr.Palette.Value)
		}
	}

	printGroup := func(groupName, accent string, idOffset int) {
		ids := [4]int{idOffset, idOffset + 1, idOffset + 2, idOffset + 3}

		fmt.Printf("\n%s  %s\n",
			clr.Hex("●", accent, clr.AttrBold),
			clr.Heading(strings.ToUpper(groupName)+" Presets"),
		)

		// Contact info for this group
		comment, url, publisher, encodedby := config.ContactForGroup(groupName)
		fmt.Printf("  %s %s\n", clr.DimText("comment  :"), clr.Hex(comment, "#9ece6a"))
		fmt.Printf("  %s %s\n", clr.DimText("url      :"), clr.Hex(url, "#9ece6a"))
		fmt.Printf("  %s %s\n", clr.DimText("publisher:"), clr.Hex(publisher, "#9ece6a"))
		fmt.Printf("  %s %s\n", clr.DimText("encodedby:"), clr.Hex(encodedby, "#9ece6a"))
		fmt.Println()

		const tagW = 18
		const valW = 8

		// Header
		hdr := fmt.Sprintf("  %s  ", clr.PadRight("TAG", tagW))
		for _, id := range ids {
			hdr += clr.Hex(fmt.Sprintf("%-*s  ", valW, fmt.Sprintf("[%d]", id)), accent, clr.AttrBold)
		}
		fmt.Println(hdr)
		fmt.Println("  " + clr.Sep(tw-4))

		// Rows
		for _, r := range rows {
			line := fmt.Sprintf("  %s  ", clr.Label(clr.PadRight(r.tag, tagW)))
			vals := []string{r.v0, r.v1, r.v2, r.v3}
			for _, v := range vals {
				line += styleVal(v, valW) + "  "
			}
			fmt.Println(line)
		}

		// Description legend
		fmt.Println()
		presets := config.PresetsForGroup(groupName)
		for _, p := range presets {
			fmt.Printf("  %s  %s\n",
				clr.Hex(fmt.Sprintf("[%2d]", p.ID), accent, clr.AttrBold),
				clr.DimText(p.Description),
			)
		}
	}

	printGroup("licface",   "#bb9af7", 0)
	printGroup("cumulus13", "#73daca", 10)
	fmt.Println()
}
