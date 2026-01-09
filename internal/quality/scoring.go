package quality

import (
	"encoding/json"
	"strings"

	"github.com/outpost/outpost/internal/database"
	"github.com/outpost/outpost/internal/parser"
)

// BaseQualityScores maps quality tiers to their base scores
var BaseQualityScores = map[string]int{
	"Remux-2160p":  100000,
	"Bluray-2160p": 90000,
	"WEBDL-2160p":  80000,
	"WEBRip-2160p": 75000,
	"HDTV-2160p":   70000,
	"Remux-1080p":  60000,
	"Bluray-1080p": 50000,
	"WEBDL-1080p":  40000,
	"WEBRip-1080p": 35000,
	"HDTV-1080p":   30000,
	"Bluray-720p":  25000,
	"WEBDL-720p":   20000,
	"WEBRip-720p":  18000,
	"HDTV-720p":    15000,
	"DVD":          10000,
	"SDTV":         5000,
	"Unknown":      1000,
}

// Condition represents a single condition for a custom format
type Condition struct {
	Type     string `json:"type"`     // resolution, source, codec, audioCodec, audioFeature, keyword, notKeyword, releaseGroup
	Value    string `json:"value"`    // The value to match
	Required bool   `json:"required"` // If true, must match or release rejected
	Negate   bool   `json:"negate"`   // If true, condition is inverted
}

// FormatScore represents a custom format with its score
type FormatScore struct {
	FormatID int64 `json:"formatId"`
	Score    int   `json:"score"`
}

// ScoredRelease contains a parsed release with its scores
type ScoredRelease struct {
	*parser.ParsedRelease
	Quality          string            `json:"quality"`
	BaseScore        int               `json:"baseScore"`
	CustomFormatHits []CustomFormatHit `json:"customFormatHits"`
	TotalScore       int               `json:"totalScore"`
	Rejected         bool              `json:"rejected"`
	RejectionReason  string            `json:"rejectionReason,omitempty"`
}

// CustomFormatHit represents a matched custom format
type CustomFormatHit struct {
	Name  string `json:"name"`
	Score int    `json:"score"`
}

// Profile represents a quality profile for scoring
type Profile struct {
	ID                 int64                  `json:"id"`
	Name               string                 `json:"name"`
	UpgradeAllowed     bool                   `json:"upgradeAllowed"`
	UpgradeUntilScore  int                    `json:"upgradeUntilScore"`
	MinFormatScore     int                    `json:"minFormatScore"`
	CutoffFormatScore  int                    `json:"cutoffFormatScore"`
	Qualities          []string               `json:"qualities"`         // Enabled quality tiers
	CustomFormatScores map[int64]int          `json:"customFormatScores"` // format_id -> score
}

// CustomFormatDef represents a custom format definition
type CustomFormatDef struct {
	ID         int64       `json:"id"`
	Name       string      `json:"name"`
	Conditions []Condition `json:"conditions"`
}

// ScoreRelease scores a release against a profile
func ScoreRelease(release *parser.ParsedRelease, profile *Profile, customFormats []CustomFormatDef) *ScoredRelease {
	quality := ComputeQualityTier(release)
	scored := &ScoredRelease{
		ParsedRelease:    release,
		Quality:          quality,
		CustomFormatHits: []CustomFormatHit{},
	}

	// Check if quality is enabled
	qualityEnabled := false
	for _, q := range profile.Qualities {
		if q == quality {
			qualityEnabled = true
			break
		}
	}

	if !qualityEnabled && len(profile.Qualities) > 0 {
		scored.Rejected = true
		scored.RejectionReason = "Quality not enabled in profile"
		return scored
	}

	// Get base score for quality
	scored.BaseScore = BaseQualityScores[quality]

	// Apply custom format scores
	for _, format := range customFormats {
		if matchesCustomFormat(release, format.Conditions) {
			score := profile.CustomFormatScores[format.ID]
			scored.CustomFormatHits = append(scored.CustomFormatHits, CustomFormatHit{
				Name:  format.Name,
				Score: score,
			})
			scored.TotalScore += score
		}
	}

	// Add base score to total
	scored.TotalScore += scored.BaseScore

	// Check minimum format score
	if scored.TotalScore < profile.MinFormatScore {
		scored.Rejected = true
		scored.RejectionReason = "Below minimum format score"
	}

	return scored
}

// matchesCustomFormat checks if a release matches all conditions of a custom format
func matchesCustomFormat(release *parser.ParsedRelease, conditions []Condition) bool {
	if len(conditions) == 0 {
		return false
	}

	for _, cond := range conditions {
		matches := conditionMatches(release, cond)
		if cond.Negate {
			matches = !matches
		}

		if cond.Required && !matches {
			return false
		}
	}

	return true
}

func conditionMatches(release *parser.ParsedRelease, cond Condition) bool {
	value := strings.ToLower(cond.Value)

	switch cond.Type {
	case "resolution":
		return strings.ToLower(release.Resolution) == value

	case "source":
		return strings.ToLower(release.Source) == value

	case "codec":
		return strings.ToLower(release.Codec) == value

	case "audioCodec":
		return strings.ToLower(release.AudioFormat) == value

	case "audioFeature":
		return strings.ToLower(release.AudioChannels) == value

	case "hdr":
		return strings.ToLower(release.HDR) == value

	case "releaseGroup":
		return strings.EqualFold(release.ReleaseGroup, cond.Value)

	case "keyword":
		// Check if keyword exists in the original release name
		return strings.Contains(strings.ToLower(release.Title), value)

	case "edition":
		return strings.ToLower(release.Edition) == value

	case "proper":
		return release.IsProper

	case "repack":
		return release.IsRepack

	case "quality":
		return strings.EqualFold(ComputeQualityTier(release), cond.Value)
	}

	return false
}

// ParseConditions parses conditions JSON string into slice
func ParseConditions(conditionsJSON string) ([]Condition, error) {
	if conditionsJSON == "" || conditionsJSON == "[]" {
		return []Condition{}, nil
	}

	var conditions []Condition
	if err := json.Unmarshal([]byte(conditionsJSON), &conditions); err != nil {
		return nil, err
	}
	return conditions, nil
}

// ParseQualities parses qualities JSON string into slice
func ParseQualities(qualitiesJSON string) ([]string, error) {
	if qualitiesJSON == "" || qualitiesJSON == "[]" {
		return []string{}, nil
	}

	var qualities []string
	if err := json.Unmarshal([]byte(qualitiesJSON), &qualities); err != nil {
		return nil, err
	}
	return qualities, nil
}

// ParseCustomFormatScores parses custom format scores JSON string into map
func ParseCustomFormatScores(scoresJSON string) (map[int64]int, error) {
	if scoresJSON == "" || scoresJSON == "{}" {
		return map[int64]int{}, nil
	}

	var scores map[int64]int
	if err := json.Unmarshal([]byte(scoresJSON), &scores); err != nil {
		return nil, err
	}
	return scores, nil
}

// DefaultProfiles returns the default quality profiles
func DefaultProfiles() []Profile {
	return []Profile{
		{
			Name:              "4K Enthusiast",
			UpgradeAllowed:    true,
			UpgradeUntilScore: 200000,
			MinFormatScore:    0,
			CutoffFormatScore: 150000,
			Qualities: []string{
				"Remux-2160p", "Bluray-2160p", "WEBDL-2160p", "WEBRip-2160p",
				"Remux-1080p", "Bluray-1080p", "WEBDL-1080p",
			},
			CustomFormatScores: map[int64]int{},
		},
		{
			Name:              "1080p Standard",
			UpgradeAllowed:    true,
			UpgradeUntilScore: 100000,
			MinFormatScore:    0,
			CutoffFormatScore: 60000,
			Qualities: []string{
				"Remux-1080p", "Bluray-1080p", "WEBDL-1080p", "WEBRip-1080p", "HDTV-1080p",
				"Bluray-720p", "WEBDL-720p",
			},
			CustomFormatScores: map[int64]int{},
		},
		{
			Name:              "Bandwidth Saver",
			UpgradeAllowed:    false,
			UpgradeUntilScore: 0,
			MinFormatScore:    0,
			CutoffFormatScore: 0,
			Qualities: []string{
				"WEBDL-1080p", "WEBDL-720p", "WEBRip-720p",
			},
			CustomFormatScores: map[int64]int{},
		},
		{
			Name:              "Any",
			UpgradeAllowed:    false,
			UpgradeUntilScore: 0,
			MinFormatScore:    0,
			CutoffFormatScore: 0,
			Qualities:         []string{}, // Empty means all qualities enabled
			CustomFormatScores: map[int64]int{},
		},
	}
}

// DefaultCustomFormats returns some default custom formats
func DefaultCustomFormats() []CustomFormatDef {
	return []CustomFormatDef{
		{
			Name: "2160p Remux",
			Conditions: []Condition{
				{Type: "resolution", Value: "2160p", Required: true},
				{Type: "source", Value: "remux", Required: true},
			},
		},
		{
			Name: "1080p Remux",
			Conditions: []Condition{
				{Type: "resolution", Value: "1080p", Required: true},
				{Type: "source", Value: "remux", Required: true},
			},
		},
		{
			Name: "TrueHD/DTS-HD MA",
			Conditions: []Condition{
				{Type: "audioCodec", Value: "truehd", Required: false},
				{Type: "audioCodec", Value: "dtshd", Required: false},
			},
		},
		{
			Name: "Dolby Atmos",
			Conditions: []Condition{
				{Type: "audioFeature", Value: "atmos", Required: true},
			},
		},
		{
			Name: "HDR/DV",
			Conditions: []Condition{
				{Type: "hdr", Value: "hdr", Required: false},
				{Type: "hdr", Value: "dv", Required: false},
			},
		},
		{
			Name: "x265/HEVC",
			Conditions: []Condition{
				{Type: "codec", Value: "x265", Required: true},
			},
		},
		{
			Name: "PROPER/REPACK",
			Conditions: []Condition{
				{Type: "proper", Value: "true", Required: false},
				{Type: "repack", Value: "true", Required: false},
			},
		},
	}
}

// AllQualities returns all available quality tiers
func AllQualities() []string {
	return []string{
		"Remux-2160p",
		"Bluray-2160p",
		"WEBDL-2160p",
		"WEBRip-2160p",
		"HDTV-2160p",
		"Remux-1080p",
		"Bluray-1080p",
		"WEBDL-1080p",
		"WEBRip-1080p",
		"HDTV-1080p",
		"Bluray-720p",
		"WEBDL-720p",
		"WEBRip-720p",
		"HDTV-720p",
		"DVD",
		"SDTV",
	}
}

// ComputeQualityTier computes the quality tier string from a parsed release
func ComputeQualityTier(release *parser.ParsedRelease) string {
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

// FormatRejection represents a format validation failure
type FormatRejection struct {
	Reason    string
	Permanent bool // If true, should be blocklisted
}

// ValidateFormat checks if a release's format is acceptable
func ValidateFormat(release *parser.ParsedRelease, settings *database.FormatSettings) *FormatRejection {
	if settings == nil {
		return nil // No settings means accept all
	}

	// Check for disc releases
	if settings.RejectDiscs && release.IsDisc {
		return &FormatRejection{
			Reason:    "Disc release not accepted (BDMV/VIDEO_TS/full disc)",
			Permanent: true,
		}
	}

	// Check for archive releases
	if settings.RejectArchives && release.IsArchive {
		return &FormatRejection{
			Reason:    "Archive release not accepted (RAR/ZIP)",
			Permanent: true,
		}
	}

	// Check container if detected
	if release.Container != "" && len(settings.AcceptedContainers) > 0 {
		containerLower := strings.ToLower(release.Container)
		
		// Check if it's an unacceptable container (ISO, RAR, etc.)
		if containerLower == "iso" || containerLower == "rar" || containerLower == "zip" || containerLower == "7z" {
			return &FormatRejection{
				Reason:    "Container type not accepted: " + release.Container,
				Permanent: true,
			}
		}
		
		// Check if container is in accepted list
		found := false
		for _, accepted := range settings.AcceptedContainers {
			if strings.ToLower(accepted) == containerLower {
				found = true
				break
			}
		}
		if !found {
			return &FormatRejection{
				Reason:    "Container type not in accepted list: " + release.Container,
				Permanent: true,
			}
		}
	}

	return nil // Accepted
}

// IsAcceptableFormat is a convenience function for quick checks
func IsAcceptableFormat(release *parser.ParsedRelease, settings *database.FormatSettings) bool {
	return ValidateFormat(release, settings) == nil
}
