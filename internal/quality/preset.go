package quality

import (
	"sort"
	"strings"

	"github.com/outpost/outpost/internal/parser"
)

// Preset represents a quality preset configuration
type Preset struct {
	ID                 int64    `json:"id"`
	Name               string   `json:"name"`
	IsDefault          bool     `json:"isDefault"`
	IsBuiltIn          bool     `json:"isBuiltIn"`
	Resolution         string   `json:"resolution"`         // "4k", "1080p", "720p", "480p"
	MinResolution      string   `json:"minResolution"`      // Minimum acceptable resolution
	Sources            []string `json:"sources"`            // ["remux", "bluray", "webdl", "webrip"]
	Source             string   `json:"source"`             // Deprecated: single source, use Sources
	HDRFormats         []string `json:"hdrFormats"`         // ["dv", "hdr10plus", "hdr10"] or empty
	Codec              string   `json:"codec"`              // "any", "hevc", "av1"
	AudioFormats       []string `json:"audioFormats"`       // ["atmos", "truehd", "dtshd"] or empty
	AudioChannels      []string `json:"audioChannels"`      // ["7.1", "5.1", "2.0"]
	PreferredEdition   string   `json:"preferredEdition"`   // "any", "theatrical", "directors", "extended", "unrated"
	MinSeeders         int      `json:"minSeeders"`
	PreferSeasonPacks  bool     `json:"preferSeasonPacks"`
	AutoUpgrade        bool     `json:"autoUpgrade"`

	// Cutoff settings
	CutoffResolution   string   `json:"cutoffResolution"`   // Stop upgrading after reaching this
	CutoffSource       string   `json:"cutoffSource"`       // Stop upgrading after reaching this source
	CutoffMetBehavior  string   `json:"cutoffMetBehavior"`  // "stop" or "continue"

	// Anime-specific
	PreferDualAudio    bool     `json:"preferDualAudio"`
	PreferSoftSubs     bool     `json:"preferSoftSubs"`
	PreferHigherVersion bool    `json:"preferHigherVersion"` // v2 over v1

	// Size preferences
	PreferSmallerSize  bool     `json:"preferSmallerSize"`
}

// Built-in presets
var BuiltInPresets = []Preset{
	{
		Name:              "Best Quality",
		IsBuiltIn:         true,
		Resolution:        "4k",
		MinResolution:     "1080p",
		Sources:           []string{"remux", "bluray"},
		Source:            "remux",
		HDRFormats:        []string{"dv", "hdr10plus", "hdr10"},
		AudioFormats:      []string{"atmos", "truehd", "dtshd", "dtsx"},
		AudioChannels:     []string{"7.1", "5.1"},
		MinSeeders:        3,
		AutoUpgrade:       true,
		CutoffResolution:  "2160p",
		CutoffSource:      "remux",
		CutoffMetBehavior: "stop",
	},
	{
		Name:             "High Quality",
		IsBuiltIn:        true,
		Resolution:       "4k",
		MinResolution:    "1080p",
		Sources:          []string{"webdl", "webrip", "bluray"},
		Source:           "web",
		HDRFormats:       []string{"dv", "hdr10plus", "hdr10"},
		MinSeeders:       3,
		AutoUpgrade:      true,
		CutoffResolution: "2160p",
		CutoffSource:     "webdl",
	},
	{
		Name:          "Balanced",
		IsBuiltIn:     true,
		Resolution:    "1080p",
		MinResolution: "720p",
		Sources:       []string{"webdl", "webrip"},
		Source:        "web",
		MinSeeders:    3,
		AutoUpgrade:   true,
	},
	{
		Name:              "Storage Saver",
		IsBuiltIn:         true,
		Resolution:        "1080p",
		MinResolution:     "720p",
		Sources:           []string{"webdl", "webrip"},
		Source:            "web",
		Codec:             "hevc",
		AudioFormats:      []string{"ddplus", "aac"},
		MinSeeders:        3,
		AutoUpgrade:       false,
		PreferSmallerSize: true,
	},
	{
		Name:                "Anime",
		IsBuiltIn:           true,
		Resolution:          "1080p",
		MinResolution:       "720p",
		Sources:             []string{"bluray", "webdl", "webrip"},
		Source:              "bluray",
		AudioFormats:        []string{"flac", "aac", "opus"},
		AudioChannels:       []string{"2.0", "5.1"},
		MinSeeders:          2,
		AutoUpgrade:         true,
		PreferDualAudio:     true,
		PreferSoftSubs:      true,
		PreferHigherVersion: true,
	},
}

// ScoreWithPreset calculates a quality score for a release using preset rules
// Based on the scoring guidelines from the parsing guide
func ScoreWithPreset(release *parser.ParsedRelease, preset *Preset) int {
	score := 0

	// Instant rejection (score = -1000)
	if release.ShouldBlock() {
		return -1000
	}
	if release.Seeders < preset.MinSeeders && preset.MinSeeders > 0 {
		return -1000
	}

	// Base score by resolution
	switch release.Resolution {
	case "2160p":
		score += 100
	case "1080p":
		score += 75
	case "720p":
		score += 50
	case "480p":
		score += 25
	}

	// Source modifier
	switch release.Source {
	case "remux":
		score += 50
	case "bluray":
		score += 40
	case "webdl":
		score += 30
	case "webrip":
		score += 20
	case "hdtv":
		score += 10
	case "satellite", "pdtv":
		score += 5
	case "dvd":
		score += 5
	}

	// HDR modifier
	switch release.HDR {
	case "dv":
		score += 20
	case "hdr10plus":
		score += 15
	case "hdr10":
		score += 10
	case "hlg":
		score += 5
	}

	// Audio modifier
	switch release.AudioFormat {
	case "atmos":
		score += 20
	case "truehd", "dtshd", "dtsx":
		score += 15
	case "flac", "pcm":
		score += 10
	case "ddplus":
		score += 5
	case "dts":
		score += 3
	case "dd":
		score += 2
	}

	// Codec modifier
	switch release.Codec {
	case "hevc", "av1":
		score += 5 // Efficient codecs
	case "avc":
		score += 3
	}

	// Bit depth modifier
	if release.BitDepth == 10 {
		score += 5
	}

	// Group modifier
	if parser.IsTrustedGroup(release.ReleaseGroup, "movies") ||
		parser.IsTrustedGroup(release.ReleaseGroup, "tv") ||
		parser.IsTrustedGroup(release.ReleaseGroup, "anime") {
		score += 5
	}

	// Proper/Repack/Rerip/Syncfix bonus
	if release.IsProper || release.IsRepack {
		score += 5
	}
	if release.IsRerip || release.IsSyncfix {
		score += 5
	}
	// DS4K (downscaled from 4K) is a good quality indicator
	if release.IsDS4K {
		score += 3
	}

	// Version bonus for anime (v2, v3)
	if release.IsAnime && release.Version > 1 {
		score += 3 * (release.Version - 1)
	}

	// Dual-audio bonus for anime
	if release.IsAnime && release.HasDualAudio && preset.PreferDualAudio {
		score += 10
	}

	// Soft subs bonus
	if release.HasSoftSubs && preset.PreferSoftSubs {
		score += 5
	}

	// Seeder bonus (prefer more seeders, capped at 10 bonus points)
	seederBonus := release.Seeders / 10
	if seederBonus > 10 {
		seederBonus = 10
	}
	score += seederBonus

	// Negative modifiers
	if release.IsFullscreen {
		score -= 20
	}
	if release.IsDubbed {
		score -= 10
	}
	if release.IsFansub {
		score -= 5
	}

	return score
}

// MatchesTarget checks if a release matches the target preset
func MatchesTarget(release *parser.ParsedRelease, preset *Preset) (bool, int) {
	// Hard blocks — never grab
	if release.ShouldBlock() {
		return false, 0
	}
	if preset.MinSeeders > 0 && release.Seeders < preset.MinSeeders {
		return false, 0
	}

	// Check minimum resolution floor
	if !meetsMinimumResolution(release.Resolution, preset.MinResolution) {
		return false, 0
	}

	score := ScoreWithPreset(release, preset)
	return true, score
}

// meetsMinimumResolution checks if release meets minimum resolution
func meetsMinimumResolution(releaseRes, minRes string) bool {
	if minRes == "" {
		minRes = "720p" // Default minimum
	}

	order := map[string]int{
		"2160p": 4,
		"4k":    4,
		"1080p": 3,
		"720p":  2,
		"480p":  1,
	}

	releaseOrder := order[releaseRes]
	minOrder := order[minRes]

	return releaseOrder >= minOrder
}

// CheckTargetMatch verifies if a release exactly matches preset requirements
func CheckTargetMatch(release *parser.ParsedRelease, preset *Preset) bool {
	// Resolution match
	if !resolutionMatches(release.Resolution, preset.Resolution) {
		return false
	}

	// Source match - check against Sources list first, fall back to single Source
	if len(preset.Sources) > 0 {
		if !containsSource(preset.Sources, release.Source) {
			return false
		}
	} else if preset.Source != "" && preset.Source != "any" {
		// Legacy single source check
		if !sourceMatches(release.Source, preset.Source) {
			return false
		}
	}

	// HDR match (if required)
	if len(preset.HDRFormats) > 0 {
		if !contains(preset.HDRFormats, release.HDR) {
			return false
		}
	}

	// Audio match (if required)
	if len(preset.AudioFormats) > 0 {
		if !contains(preset.AudioFormats, release.AudioFormat) {
			return false
		}
	}

	// Codec match (if required)
	if preset.Codec != "" && preset.Codec != "any" {
		if release.Codec != preset.Codec && release.Codec != "" {
			return false
		}
	}

	return true
}

// resolutionMatches checks if release resolution meets or exceeds target
func resolutionMatches(releaseRes, targetRes string) bool {
	order := map[string]int{
		"2160p": 4,
		"4k":    4,
		"1080p": 3,
		"720p":  2,
		"480p":  1,
	}

	releaseOrder := order[releaseRes]
	targetOrder := order[targetRes]

	return releaseOrder >= targetOrder
}

// sourceMatches checks if a release source matches a target source
func sourceMatches(releaseSource, targetSource string) bool {
	if targetSource == "any" || targetSource == "" {
		return true
	}

	// "web" matches both webdl and webrip
	if targetSource == "web" {
		return releaseSource == "webdl" || releaseSource == "webrip"
	}

	return releaseSource == targetSource
}

// containsSource checks if a source is in the allowed list
func containsSource(sources []string, source string) bool {
	for _, s := range sources {
		if sourceMatches(source, s) {
			return true
		}
	}
	return false
}

// contains checks if a slice contains a string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if strings.EqualFold(s, item) {
			return true
		}
	}
	return false
}

// MeetsCutoff checks if a release meets the cutoff criteria
func MeetsCutoff(release *parser.ParsedRelease, preset *Preset) bool {
	if preset.CutoffResolution == "" && preset.CutoffSource == "" {
		return false // No cutoff defined
	}

	// Check resolution cutoff
	if preset.CutoffResolution != "" {
		if !resolutionMatches(release.Resolution, preset.CutoffResolution) {
			return false
		}
	}

	// Check source cutoff
	if preset.CutoffSource != "" {
		sourceOrder := map[string]int{
			"remux":  5,
			"bluray": 4,
			"webdl":  3,
			"webrip": 2,
			"hdtv":   1,
		}

		releaseOrder := sourceOrder[release.Source]
		cutoffOrder := sourceOrder[preset.CutoffSource]

		if releaseOrder < cutoffOrder {
			return false
		}
	}

	return true
}

// PresetScoredRelease pairs a release with its score and match status for preset matching
type PresetScoredRelease struct {
	Release       *parser.ParsedRelease
	Score         int
	MatchesTarget bool
	MeetsCutoff   bool
}

// SelectBestRelease finds the best release from a list given a preset
func SelectBestRelease(releases []*parser.ParsedRelease, preset *Preset) *parser.ParsedRelease {
	var candidates []PresetScoredRelease

	for _, r := range releases {
		ok, score := MatchesTarget(r, preset)
		if ok {
			matchesTarget := CheckTargetMatch(r, preset)
			meetsCutoff := MeetsCutoff(r, preset)
			candidates = append(candidates, PresetScoredRelease{
				Release:       r,
				Score:         score,
				MatchesTarget: matchesTarget,
				MeetsCutoff:   meetsCutoff,
			})
		}
	}

	if len(candidates) == 0 {
		return nil
	}

	// Sort: cutoff met first, then target matches, then by score
	sort.Slice(candidates, func(i, j int) bool {
		// Prefer releases that meet cutoff
		if candidates[i].MeetsCutoff != candidates[j].MeetsCutoff {
			return candidates[i].MeetsCutoff
		}
		// Then prefer target matches
		if candidates[i].MatchesTarget != candidates[j].MatchesTarget {
			return candidates[i].MatchesTarget
		}
		// Then by score
		return candidates[i].Score > candidates[j].Score
	})

	return candidates[0].Release
}

// RankReleases returns all acceptable releases sorted by quality
func RankReleases(releases []*parser.ParsedRelease, preset *Preset) []PresetScoredRelease {
	var acceptable []PresetScoredRelease

	for _, r := range releases {
		ok, score := MatchesTarget(r, preset)
		if ok {
			matchesTarget := CheckTargetMatch(r, preset)
			meetsCutoff := MeetsCutoff(r, preset)
			acceptable = append(acceptable, PresetScoredRelease{
				Release:       r,
				Score:         score,
				MatchesTarget: matchesTarget,
				MeetsCutoff:   meetsCutoff,
			})
		}
	}

	// Sort: cutoff met first, then target matches, then by score
	sort.Slice(acceptable, func(i, j int) bool {
		if acceptable[i].MeetsCutoff != acceptable[j].MeetsCutoff {
			return acceptable[i].MeetsCutoff
		}
		if acceptable[i].MatchesTarget != acceptable[j].MatchesTarget {
			return acceptable[i].MatchesTarget
		}
		return acceptable[i].Score > acceptable[j].Score
	})

	return acceptable
}

// GetFilteredReleases returns releases that were filtered out
func GetFilteredReleases(releases []*parser.ParsedRelease, preset *Preset) []PresetScoredRelease {
	var filtered []PresetScoredRelease

	for _, r := range releases {
		ok, _ := MatchesTarget(r, preset)
		if !ok {
			filtered = append(filtered, PresetScoredRelease{
				Release: r,
				Score:   ScoreWithPreset(r, preset),
			})
		}
	}

	return filtered
}

// FormatQualityBadge creates a display string for quality
func FormatQualityBadge(release *parser.ParsedRelease) string {
	parts := []string{}

	if release.Resolution != "" {
		parts = append(parts, release.Resolution)
	}
	if release.Source != "" {
		parts = append(parts, strings.ToUpper(release.Source))
	}
	if release.HDR != "" {
		parts = append(parts, strings.ToUpper(release.HDR))
	}
	if release.AudioFormat != "" {
		parts = append(parts, strings.ToUpper(release.AudioFormat))
	}

	return strings.Join(parts, " ")
}

// IsUpgrade checks if a new release is an upgrade over existing quality
func IsUpgrade(newRelease *parser.ParsedRelease, currentResolution, currentSource, currentHDR, currentAudio string, preset *Preset) bool {
	// Create a mock current release for comparison
	current := &parser.ParsedRelease{
		Resolution:  currentResolution,
		Source:      currentSource,
		HDR:         currentHDR,
		AudioFormat: currentAudio,
	}

	newScore := ScoreWithPreset(newRelease, preset)
	currentScore := ScoreWithPreset(current, preset)

	return newScore > currentScore
}
