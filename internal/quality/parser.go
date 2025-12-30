package quality

import (
	"regexp"
	"strconv"
	"strings"
)

// ParsedRelease contains the parsed attributes of a release name
type ParsedRelease struct {
	Title       string   `json:"title"`
	Year        int      `json:"year,omitempty"`
	Resolution  string   `json:"resolution,omitempty"`  // 2160p, 1080p, 720p, 480p
	Source      string   `json:"source,omitempty"`      // remux, bluray, webdl, webrip, hdtv, dvd
	Codec       string   `json:"codec,omitempty"`       // x265, x264, hevc, av1
	AudioCodec  string   `json:"audioCodec,omitempty"`  // truehd, dtshd, dts, dd, aac
	AudioFeature string  `json:"audioFeature,omitempty"` // atmos, 7.1, 5.1
	HDR         []string `json:"hdr,omitempty"`          // hdr, hdr10, hdr10plus, dv
	ReleaseGroup string  `json:"releaseGroup,omitempty"`
	Proper      bool     `json:"proper,omitempty"`
	Repack      bool     `json:"repack,omitempty"`
	Edition     string   `json:"edition,omitempty"`      // extended, directors.cut, etc.
	Season      int      `json:"season,omitempty"`
	Episode     int      `json:"episode,omitempty"`
	Quality     string   `json:"quality,omitempty"`      // Computed quality tier
}

var (
	// Resolution patterns
	resolutionPatterns = map[string]*regexp.Regexp{
		"2160p": regexp.MustCompile(`(?i)(2160p|4k|uhd)`),
		"1080p": regexp.MustCompile(`(?i)1080[pi]`),
		"720p":  regexp.MustCompile(`(?i)720p`),
		"480p":  regexp.MustCompile(`(?i)(480p|sd)`),
	}

	// Source patterns (order matters for priority)
	sourcePatterns = []struct {
		name    string
		pattern *regexp.Regexp
	}{
		{"remux", regexp.MustCompile(`(?i)\b(remux)\b`)},
		{"bluray", regexp.MustCompile(`(?i)\b(blu[\-\.]?ray|bdrip|bd\b)`)},
		{"webdl", regexp.MustCompile(`(?i)\b(web[\-\.]?dl|webdl)\b`)},
		{"webrip", regexp.MustCompile(`(?i)\b(web[\-\.]?rip|webrip)\b`)},
		{"hdtv", regexp.MustCompile(`(?i)\b(hdtv)\b`)},
		{"dvd", regexp.MustCompile(`(?i)\b(dvd|dvdrip)\b`)},
		{"sdtv", regexp.MustCompile(`(?i)\b(sdtv|pdtv)\b`)},
	}

	// Codec patterns
	codecPatterns = map[string]*regexp.Regexp{
		"x265":  regexp.MustCompile(`(?i)\b(x265|h\.?265|hevc)\b`),
		"x264":  regexp.MustCompile(`(?i)\b(x264|h\.?264|avc)\b`),
		"av1":   regexp.MustCompile(`(?i)\b(av1)\b`),
		"xvid":  regexp.MustCompile(`(?i)\b(xvid|divx)\b`),
		"mpeg2": regexp.MustCompile(`(?i)\b(mpeg[\-\.]?2)\b`),
	}

	// Audio codec patterns
	audioCodecPatterns = []struct {
		name    string
		pattern *regexp.Regexp
	}{
		{"truehd", regexp.MustCompile(`(?i)\b(true[\-\.]?hd)\b`)},
		{"dtshd", regexp.MustCompile(`(?i)\b(dts[\-\.]?hd[\-\.]?(ma)?|dts[\-\.]?x)\b`)},
		{"dts", regexp.MustCompile(`(?i)\b(dts)\b`)},
		{"ddplus", regexp.MustCompile(`(?i)\b(dd[\+p]|ddp|e[\-\.]?ac[\-\.]?3|eac3)\b`)},
		{"dd", regexp.MustCompile(`(?i)\b(dd|ac[\-\.]?3|ac3)\b`)},
		{"flac", regexp.MustCompile(`(?i)\b(flac)\b`)},
		{"aac", regexp.MustCompile(`(?i)\b(aac)\b`)},
		{"mp3", regexp.MustCompile(`(?i)\b(mp3)\b`)},
	}

	// Audio feature patterns
	audioFeaturePatterns = map[string]*regexp.Regexp{
		"atmos": regexp.MustCompile(`(?i)\b(atmos)\b`),
		"7.1":   regexp.MustCompile(`(?i)\b(7[\.\s]?1)\b`),
		"5.1":   regexp.MustCompile(`(?i)\b(5[\.\s]?1)\b`),
	}

	// HDR patterns
	hdrPatterns = map[string]*regexp.Regexp{
		"dv":        regexp.MustCompile(`(?i)\b(dv|dovi|dolby[\.\s]?vision)\b`),
		"hdr10plus": regexp.MustCompile(`(?i)\b(hdr10[\+p]|hdr10plus)\b`),
		"hdr10":     regexp.MustCompile(`(?i)\b(hdr10)\b`),
		"hdr":       regexp.MustCompile(`(?i)\b(hdr)\b`),
		"hlg":       regexp.MustCompile(`(?i)\b(hlg)\b`),
	}

	// Special edition patterns
	editionPatterns = map[string]*regexp.Regexp{
		"extended":      regexp.MustCompile(`(?i)\b(extended)\b`),
		"directors.cut": regexp.MustCompile(`(?i)\b(director'?s?[\.\s]?cut)\b`),
		"unrated":       regexp.MustCompile(`(?i)\b(unrated)\b`),
		"theatrical":    regexp.MustCompile(`(?i)\b(theatrical)\b`),
		"remastered":    regexp.MustCompile(`(?i)\b(remaster(ed)?)\b`),
		"imax":          regexp.MustCompile(`(?i)\b(imax)\b`),
	}

	// Proper/Repack patterns
	properPattern = regexp.MustCompile(`(?i)\b(proper)\b`)
	repackPattern = regexp.MustCompile(`(?i)\b(repack|rerip)\b`)

	// Release group pattern (at end of name)
	releaseGroupPattern = regexp.MustCompile(`-([A-Za-z0-9]+)(?:\.[a-z]{2,4})?$`)

	// Year pattern
	yearPattern = regexp.MustCompile(`\b(19|20)\d{2}\b`)

	// Season/Episode patterns
	seasonEpisodePattern = regexp.MustCompile(`(?i)S(\d{1,2})E(\d{1,2})`)
	seasonOnlyPattern    = regexp.MustCompile(`(?i)(?:Season|S)[\.\s]?(\d{1,2})`)
)

// ParseReleaseName parses a release name and extracts its attributes
func ParseReleaseName(name string) *ParsedRelease {
	release := &ParsedRelease{}

	// Clean up the name
	cleanName := strings.ReplaceAll(name, "_", " ")
	cleanName = strings.ReplaceAll(cleanName, ".", " ")

	// Extract resolution
	for res, pattern := range resolutionPatterns {
		if pattern.MatchString(name) {
			release.Resolution = res
			break
		}
	}

	// Extract source
	for _, sp := range sourcePatterns {
		if sp.pattern.MatchString(name) {
			release.Source = sp.name
			break
		}
	}

	// Extract codec
	for codec, pattern := range codecPatterns {
		if pattern.MatchString(name) {
			release.Codec = codec
			break
		}
	}

	// Extract audio codec
	for _, ap := range audioCodecPatterns {
		if ap.pattern.MatchString(name) {
			release.AudioCodec = ap.name
			break
		}
	}

	// Extract audio feature
	for feature, pattern := range audioFeaturePatterns {
		if pattern.MatchString(name) {
			release.AudioFeature = feature
			break
		}
	}

	// Extract HDR features (can have multiple)
	for hdr, pattern := range hdrPatterns {
		if pattern.MatchString(name) {
			release.HDR = append(release.HDR, hdr)
		}
	}

	// Extract edition
	for edition, pattern := range editionPatterns {
		if pattern.MatchString(name) {
			release.Edition = edition
			break
		}
	}

	// Check proper/repack
	release.Proper = properPattern.MatchString(name)
	release.Repack = repackPattern.MatchString(name)

	// Extract release group
	if matches := releaseGroupPattern.FindStringSubmatch(name); len(matches) > 1 {
		release.ReleaseGroup = matches[1]
	}

	// Extract year
	if matches := yearPattern.FindStringSubmatch(cleanName); len(matches) > 0 {
		var year int
		if err := parseIntFromString(matches[0], &year); err == nil {
			release.Year = year
		}
	}

	// Extract season/episode
	if matches := seasonEpisodePattern.FindStringSubmatch(name); len(matches) > 2 {
		parseIntFromString(matches[1], &release.Season)
		parseIntFromString(matches[2], &release.Episode)
	} else if matches := seasonOnlyPattern.FindStringSubmatch(name); len(matches) > 1 {
		parseIntFromString(matches[1], &release.Season)
	}

	// Extract title (everything before year or resolution or season info)
	release.Title = extractTitle(cleanName, release)

	// Compute quality tier
	release.Quality = computeQualityTier(release)

	return release
}

func parseIntFromString(s string, target *int) error {
	var n int
	for _, c := range s {
		if c >= '0' && c <= '9' {
			n = n*10 + int(c-'0')
		}
	}
	*target = n
	return nil
}

func extractTitle(cleanName string, release *ParsedRelease) string {
	title := cleanName

	// Remove everything after common markers
	markers := []string{
		" " + release.Resolution,
		" S0", " S1", " S2", " S3", " Season",
		" 19", " 20", // Years
	}

	for _, marker := range markers {
		if marker == " " || marker == " 0" {
			continue
		}
		if idx := strings.Index(title, marker); idx > 0 {
			title = title[:idx]
		}
	}

	// Also try year
	if release.Year > 0 {
		yearStr := strconv.Itoa(release.Year)
		if idx := strings.Index(title, yearStr); idx > 0 {
			title = title[:idx]
		}
	}

	return strings.TrimSpace(title)
}

func computeQualityTier(release *ParsedRelease) string {
	res := release.Resolution
	source := release.Source

	if res == "" {
		res = "unknown"
	}
	if source == "" {
		source = "unknown"
	}

	// Map to quality tiers based on resolution and source
	switch {
	case res == "2160p" && source == "remux":
		return "Remux-2160p"
	case res == "2160p" && source == "bluray":
		return "Bluray-2160p"
	case res == "2160p" && source == "webdl":
		return "WEBDL-2160p"
	case res == "2160p" && source == "webrip":
		return "WEBRip-2160p"
	case res == "2160p" && source == "hdtv":
		return "HDTV-2160p"
	case res == "2160p":
		return "WEBDL-2160p" // Default 4K

	case res == "1080p" && source == "remux":
		return "Remux-1080p"
	case res == "1080p" && source == "bluray":
		return "Bluray-1080p"
	case res == "1080p" && source == "webdl":
		return "WEBDL-1080p"
	case res == "1080p" && source == "webrip":
		return "WEBRip-1080p"
	case res == "1080p" && source == "hdtv":
		return "HDTV-1080p"
	case res == "1080p":
		return "WEBDL-1080p" // Default 1080p

	case res == "720p" && source == "bluray":
		return "Bluray-720p"
	case res == "720p" && source == "webdl":
		return "WEBDL-720p"
	case res == "720p" && source == "webrip":
		return "WEBRip-720p"
	case res == "720p" && source == "hdtv":
		return "HDTV-720p"
	case res == "720p":
		return "WEBDL-720p" // Default 720p

	case source == "dvd":
		return "DVD"
	case source == "sdtv":
		return "SDTV"
	case res == "480p":
		return "SDTV"

	default:
		return "Unknown"
	}
}
