package quality

import (
	"sort"
	"strings"

	"github.com/outpost/outpost/internal/parser"
)

// Preset represents a quality preset configuration
type Preset struct {
	ID               int64    `json:"id"`
	Name             string   `json:"name"`
	IsDefault        bool     `json:"isDefault"`
	IsBuiltIn        bool     `json:"isBuiltIn"`
	Resolution       string   `json:"resolution"`       // "4k", "1080p", "720p", "480p"
	Source           string   `json:"source"`           // "remux", "bluray", "web", "any"
	HDRFormats       []string `json:"hdrFormats"`       // ["dv", "hdr10+", "hdr10"] or empty
	Codec            string   `json:"codec"`            // "any", "hevc", "av1"
	AudioFormats     []string `json:"audioFormats"`     // ["atmos", "truehd", "dtshd"] or empty
	PreferredEdition string   `json:"preferredEdition"` // "any", "theatrical", "directors", "extended", "unrated"
	MinSeeders       int      `json:"minSeeders"`
	PreferSeasonPacks bool    `json:"preferSeasonPacks"`
	AutoUpgrade      bool     `json:"autoUpgrade"`
}

// Built-in presets
var BuiltInPresets = []Preset{
	{
		Name:         "Best",
		IsBuiltIn:    true,
		Resolution:   "4k",
		Source:       "remux",
		HDRFormats:   []string{"dv", "hdr10+", "hdr10"},
		AudioFormats: []string{"atmos", "truehd", "dtshd"},
		MinSeeders:   3,
		AutoUpgrade:  true,
	},
	{
		Name:        "High",
		IsBuiltIn:   true,
		Resolution:  "4k",
		Source:      "web",
		HDRFormats:  []string{"dv", "hdr10+", "hdr10"},
		MinSeeders:  3,
		AutoUpgrade: true,
	},
	{
		Name:        "Balanced",
		IsBuiltIn:   true,
		Resolution:  "1080p",
		Source:      "web",
		MinSeeders:  3,
		AutoUpgrade: true,
	},
	{
		Name:        "Storage Saver",
		IsBuiltIn:   true,
		Resolution:  "1080p",
		Source:      "web",
		Codec:       "hevc",
		MinSeeders:  3,
		AutoUpgrade: false,
	},
}

// ScorePresetRelease calculates a quality score for a release against a preset
func ScorePresetRelease(release *parser.ParsedRelease) int {
	score := 0

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
	case "web":
		score += 30
	case "hdtv":
		score += 10
	case "cam":
		score -= 100 // Heavy penalty
	}

	// HDR modifier
	switch release.HDR {
	case "dv":
		score += 20
	case "hdr10+":
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
	case "truehd":
		score += 15
	case "dtshd":
		score += 15
	case "dtsx":
		score += 15
	case "dd+":
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
	case "x264":
		score += 3
	}

	// Group modifier
	if parser.IsTrustedGroup(release.ReleaseGroup, "movies") {
		score += 10
	}

	// Proper/Repack bonus
	if release.IsProper || release.IsRepack {
		score += 5
	}

	// Seeder bonus (prefer more seeders, capped at 10 bonus points)
	seederBonus := release.Seeders / 10
	if seederBonus > 10 {
		seederBonus = 10
	}
	score += seederBonus

	// Penalties
	if release.IsUpscaled {
		score -= 50
	}
	if release.IsCompressedAudio {
		score -= 100
	}
	if release.IsHardcodedSubs {
		score -= 100
	}

	return score
}

// MatchesTarget checks if a release matches the target preset
func MatchesTarget(release *parser.ParsedRelease, preset *Preset) (bool, int) {
	// Hard blocks — never grab
	if release.Source == "cam" {
		return false, 0
	}
	if release.IsHardcodedSubs {
		return false, 0
	}
	if release.IsCompressedAudio {
		return false, 0
	}
	if release.IsSample {
		return false, 0
	}
	if parser.IsBlockedGroup(release.ReleaseGroup) {
		return false, 0
	}
	if release.Seeders < preset.MinSeeders {
		return false, 0
	}

	// Check minimum quality floor
	if !meetsMinimumQuality(release) {
		return false, 0
	}

	score := ScorePresetRelease(release)
	return true, score
}

// meetsMinimumQuality checks if release meets minimum acceptable quality
func meetsMinimumQuality(release *parser.ParsedRelease) bool {
	// Minimum acceptable: 720p from non-garbage source
	if release.Resolution == "480p" {
		return false
	}
	if release.Source == "cam" || release.Source == "" {
		return false
	}
	return true
}

// CheckTargetMatch verifies if a release exactly matches preset requirements
func CheckTargetMatch(release *parser.ParsedRelease, preset *Preset) bool {
	// Resolution match
	if !resolutionMatches(release.Resolution, preset.Resolution) {
		return false
	}

	// Source match
	if preset.Source != "any" && release.Source != preset.Source {
		return false
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

// contains checks if a slice contains a string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if strings.EqualFold(s, item) {
			return true
		}
	}
	return false
}

// PresetScoredRelease pairs a release with its score and match status for preset matching
type PresetScoredRelease struct {
	Release       *parser.ParsedRelease
	Score         int
	MatchesTarget bool
}

// SelectBestRelease finds the best release from a list given a preset
func SelectBestRelease(releases []*parser.ParsedRelease, preset *Preset) *parser.ParsedRelease {
	var candidates []PresetScoredRelease

	for _, r := range releases {
		ok, score := MatchesTarget(r, preset)
		if ok {
			matchesTarget := CheckTargetMatch(r, preset)
			candidates = append(candidates, PresetScoredRelease{
				Release:       r,
				Score:         score,
				MatchesTarget: matchesTarget,
			})
		}
	}

	if len(candidates) == 0 {
		return nil
	}

	// Sort: target matches first, then by score
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].MatchesTarget != candidates[j].MatchesTarget {
			return candidates[i].MatchesTarget
		}
		return candidates[i].Score > candidates[j].Score
	})

	return candidates[0].Release
}

// RankReleases returns all acceptable releases sorted by quality
func RankReleases(releases []*parser.ParsedRelease, preset *Preset) []PresetScoredRelease {
	var acceptable []PresetScoredRelease
	var filtered []PresetScoredRelease

	for _, r := range releases {
		ok, score := MatchesTarget(r, preset)
		if ok {
			matchesTarget := CheckTargetMatch(r, preset)
			acceptable = append(acceptable, PresetScoredRelease{
				Release:       r,
				Score:         score,
				MatchesTarget: matchesTarget,
			})
		} else {
			filtered = append(filtered, PresetScoredRelease{
				Release: r,
				Score:   ScorePresetRelease(r),
			})
		}
	}

	// Sort acceptable: target matches first, then by score
	sort.Slice(acceptable, func(i, j int) bool {
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
				Score:   ScorePresetRelease(r),
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
