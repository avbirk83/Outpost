package parser

import (
	"regexp"
	"strconv"
	"strings"
)

// ParsedRelease contains all extracted info from a release name
type ParsedRelease struct {
	Title        string
	Year         int
	Season       int  // 0 if movie
	Episode      int  // 0 if movie or season pack
	EpisodeEnd   int  // For multi-episode (S01E01E02)
	IsSeasonPack bool
	IsDailyShow  bool
	AirDate      string // For daily shows: "2024-01-15"

	// Quality
	Resolution string // "2160p", "1080p", "720p", "480p"
	Source     string // "remux", "bluray", "web", "hdtv", "cam"
	HDR        string // "dv", "hdr10+", "hdr10", "hlg", ""
	Codec      string // "hevc", "x265", "x264", "av1", "xvid"

	// Audio
	AudioFormat   string // "atmos", "truehd", "dtshd", "dtsx", "dd+", "dd", "aac"
	AudioChannels string // "7.1", "5.1", "2.0"

	// Other
	Edition      string // "directors", "extended", "theatrical", "unrated", ""
	ReleaseGroup string
	IsProper     bool
	IsRepack     bool
	IsReal       bool // "REAL" tag

	// Warnings
	IsHardcodedSubs   bool
	IsUpscaled        bool // Fake 4K
	IsCompressedAudio bool // MD, LD, LiNE
	IsSample          bool
	Is3D              bool

	// Raw
	RawTitle string
	Size     int64
	Seeders  int
	Indexer  string
}

// Trusted release groups by category
var TrustedGroups = map[string][]string{
	"movies": {
		"FraMeSToR", "SPARKS", "FLUX", "TERMINAL", "SMURF",
		"CtrlHD", "EVO", "HULU", "ATVP", "NTb", "CMRG",
		"PlayWEB", "DSNP", "PCOK", "HMAX", "AMZN",
	},
	"tv": {
		"NTb", "FLUX", "ATVP", "HULU", "AMZN", "DSNP",
		"PCOK", "HMAX", "CMRG", "PlayWEB",
	},
	"anime": {
		"SubsPlease", "Erai-raws", "EMBER", "Judas",
	},
}

// Blocked release groups
var BlockedGroups = []string{
	"YIFY", "YTS", "RARBG", "TGx", "MeGusta",
	"STUTTERSHIT", "aXXo", "SPARKS-encoded",
}

// Regex patterns for parsing
var (
	// Title patterns
	yearPattern      = regexp.MustCompile(`\b(19[0-9]{2}|20[0-9]{2})\b`)
	tvShowPattern    = regexp.MustCompile(`(?i)S(\d{1,2})E(\d{1,3})(?:E(\d{1,3}))?`)
	seasonPackPattern = regexp.MustCompile(`(?i)S(\d{1,2})(?:[-.]?complete|$)`)
	dailyShowPattern = regexp.MustCompile(`(\d{4})[\.-](\d{2})[\.-](\d{2})`)

	// Resolution patterns
	res4KPattern    = regexp.MustCompile(`(?i)2160p|4k|uhd`)
	res1080Pattern  = regexp.MustCompile(`(?i)1080[pi]`)
	res720Pattern   = regexp.MustCompile(`(?i)720p`)
	res480Pattern   = regexp.MustCompile(`(?i)480p|dvdrip|sdtv`)

	// Source patterns
	remuxPattern  = regexp.MustCompile(`(?i)remux`)
	blurayPattern = regexp.MustCompile(`(?i)blu-?ray|bdrip|brrip`)
	webPattern    = regexp.MustCompile(`(?i)web-?dl|webrip|amzn|nf|dsnp|hulu|atvp|hmax|pcok|aptv`)
	hdtvPattern   = regexp.MustCompile(`(?i)hdtv`)
	camPattern    = regexp.MustCompile(`(?i)\b(cam|hdcam|ts|telesync|tc|telecine|scr|screener|dvdscr)\b`)

	// HDR patterns
	dvPattern     = regexp.MustCompile(`(?i)\b(dv|dolby[\.\s-]?vision|dovi)\b`)
	hdr10pPattern = regexp.MustCompile(`(?i)hdr10\+|hdr10plus`)
	hdr10Pattern  = regexp.MustCompile(`(?i)\bhdr10?\b`)
	hlgPattern    = regexp.MustCompile(`(?i)\bhlg\b`)

	// Audio patterns
	atmosPattern   = regexp.MustCompile(`(?i)\batmos\b`)
	truehdPattern  = regexp.MustCompile(`(?i)truehd|true-hd`)
	dtshdPattern   = regexp.MustCompile(`(?i)dts-?hd|dts-?ma|dts-?hd\.?ma`)
	dtsxPattern    = regexp.MustCompile(`(?i)dts-?x`)
	ddplusPattern  = regexp.MustCompile(`(?i)dd\+|ddp|eac3|e-ac-3`)
	dtsPattern     = regexp.MustCompile(`(?i)\bdts\b`)
	ddPattern      = regexp.MustCompile(`(?i)(?:\bdd\s*5[\.\s]?1)|(?:ac-?3)`)
	aacPattern     = regexp.MustCompile(`(?i)\baac\b`)
	compAudioPattern = regexp.MustCompile(`(?i)\b(md|mic[\.\s-]?dub|line|ld)\b`)

	// Audio channels
	channels71Pattern = regexp.MustCompile(`(?i)7[\.\s]1`)
	channels51Pattern = regexp.MustCompile(`(?i)5[\.\s]1`)
	channels20Pattern = regexp.MustCompile(`(?i)2[\.\s]0|stereo`)

	// Codec patterns
	hevcPattern = regexp.MustCompile(`(?i)\b(hevc|x265|h\.?265)\b`)
	av1Pattern  = regexp.MustCompile(`(?i)\bav1\b`)
	x264Pattern = regexp.MustCompile(`(?i)\b(x264|h\.?264|avc)\b`)
	xvidPattern = regexp.MustCompile(`(?i)\bxvid\b`)

	// Edition patterns
	directorsPattern  = regexp.MustCompile(`(?i)director'?s?[\.\s-]?cut|dc\b`)
	extendedPattern   = regexp.MustCompile(`(?i)\bextended\b`)
	theatricalPattern = regexp.MustCompile(`(?i)\btheatrical\b`)
	unratedPattern    = regexp.MustCompile(`(?i)\bunrated\b`)

	// Bad patterns
	hardcodedPattern = regexp.MustCompile(`(?i)\b(hc|hardcoded|hard[\.\s-]?coded|korsub)\b`)
	samplePattern    = regexp.MustCompile(`(?i)\bsample\b`)
	upscaledPattern  = regexp.MustCompile(`(?i)\b(upscale|upscaled)\b`)
	threeDPattern    = regexp.MustCompile(`(?i)\b3d\b`)

	// Other patterns
	properPattern = regexp.MustCompile(`(?i)\bproper\b`)
	repackPattern = regexp.MustCompile(`(?i)\brepack\b`)
	realPattern   = regexp.MustCompile(`(?i)\breal\b`)
	groupPattern  = regexp.MustCompile(`-([a-zA-Z0-9]+)(?:\.[a-z]+)?$`)
)

// Parse extracts information from a release name
func Parse(name string) *ParsedRelease {
	r := &ParsedRelease{
		RawTitle: name,
	}

	// Clean name for parsing
	cleanName := strings.ReplaceAll(name, ".", " ")
	cleanName = strings.ReplaceAll(cleanName, "_", " ")

	// Extract TV show info first
	if matches := tvShowPattern.FindStringSubmatch(name); matches != nil {
		r.Season, _ = strconv.Atoi(matches[1])
		r.Episode, _ = strconv.Atoi(matches[2])
		if len(matches) > 3 && matches[3] != "" {
			r.EpisodeEnd, _ = strconv.Atoi(matches[3])
		}
	} else if matches := seasonPackPattern.FindStringSubmatch(name); matches != nil {
		r.Season, _ = strconv.Atoi(matches[1])
		r.IsSeasonPack = true
	} else if matches := dailyShowPattern.FindStringSubmatch(name); matches != nil {
		r.IsDailyShow = true
		r.AirDate = matches[1] + "-" + matches[2] + "-" + matches[3]
	}

	// Extract year
	if matches := yearPattern.FindAllString(cleanName, -1); len(matches) > 0 {
		// Use first year found (usually the release year)
		r.Year, _ = strconv.Atoi(matches[0])
	}

	// Extract title (everything before quality info)
	r.Title = extractTitle(cleanName, r)

	// Resolution
	switch {
	case res4KPattern.MatchString(name):
		r.Resolution = "2160p"
	case res1080Pattern.MatchString(name):
		r.Resolution = "1080p"
	case res720Pattern.MatchString(name):
		r.Resolution = "720p"
	case res480Pattern.MatchString(name):
		r.Resolution = "480p"
	}

	// Source
	switch {
	case remuxPattern.MatchString(name):
		r.Source = "remux"
	case camPattern.MatchString(name):
		r.Source = "cam"
	case blurayPattern.MatchString(name) && !remuxPattern.MatchString(name):
		r.Source = "bluray"
	case webPattern.MatchString(name):
		r.Source = "web"
	case hdtvPattern.MatchString(name):
		r.Source = "hdtv"
	}

	// HDR
	switch {
	case dvPattern.MatchString(name):
		r.HDR = "dv"
	case hdr10pPattern.MatchString(name):
		r.HDR = "hdr10+"
	case hdr10Pattern.MatchString(name) && !hdr10pPattern.MatchString(name):
		r.HDR = "hdr10"
	case hlgPattern.MatchString(name):
		r.HDR = "hlg"
	}

	// Audio format
	switch {
	case atmosPattern.MatchString(name):
		r.AudioFormat = "atmos"
	case truehdPattern.MatchString(name):
		r.AudioFormat = "truehd"
	case dtsxPattern.MatchString(name):
		r.AudioFormat = "dtsx"
	case dtshdPattern.MatchString(name):
		r.AudioFormat = "dtshd"
	case ddplusPattern.MatchString(name):
		r.AudioFormat = "dd+"
	case dtsPattern.MatchString(name):
		r.AudioFormat = "dts"
	case ddPattern.MatchString(name):
		r.AudioFormat = "dd"
	case aacPattern.MatchString(name):
		r.AudioFormat = "aac"
	}

	// Audio channels
	switch {
	case channels71Pattern.MatchString(name):
		r.AudioChannels = "7.1"
	case channels51Pattern.MatchString(name):
		r.AudioChannels = "5.1"
	case channels20Pattern.MatchString(name):
		r.AudioChannels = "2.0"
	}

	// Codec
	switch {
	case hevcPattern.MatchString(name):
		r.Codec = "hevc"
	case av1Pattern.MatchString(name):
		r.Codec = "av1"
	case x264Pattern.MatchString(name):
		r.Codec = "x264"
	case xvidPattern.MatchString(name):
		r.Codec = "xvid"
	}

	// Edition
	switch {
	case directorsPattern.MatchString(name):
		r.Edition = "directors"
	case extendedPattern.MatchString(name):
		r.Edition = "extended"
	case theatricalPattern.MatchString(name):
		r.Edition = "theatrical"
	case unratedPattern.MatchString(name):
		r.Edition = "unrated"
	}

	// Release group
	if matches := groupPattern.FindStringSubmatch(name); matches != nil {
		r.ReleaseGroup = matches[1]
	}

	// Flags
	r.IsProper = properPattern.MatchString(name)
	r.IsRepack = repackPattern.MatchString(name)
	r.IsReal = realPattern.MatchString(name)

	// Warnings
	r.IsHardcodedSubs = hardcodedPattern.MatchString(name)
	r.IsSample = samplePattern.MatchString(name)
	r.IsUpscaled = upscaledPattern.MatchString(name)
	r.Is3D = threeDPattern.MatchString(name)
	r.IsCompressedAudio = compAudioPattern.MatchString(name)

	return r
}

// extractTitle tries to extract the clean title from a release name
func extractTitle(cleanName string, r *ParsedRelease) string {
	// Find where the title ends (usually at year or quality info)
	titleEnd := len(cleanName)

	// Stop at year if present
	if r.Year > 0 {
		yearStr := strconv.Itoa(r.Year)
		if idx := strings.Index(cleanName, yearStr); idx > 0 {
			titleEnd = idx
		}
	}

	// Stop at S01E01 pattern
	if r.Season > 0 {
		seasonPattern := regexp.MustCompile(`(?i)S\d{1,2}`)
		if loc := seasonPattern.FindStringIndex(cleanName); loc != nil && loc[0] < titleEnd {
			titleEnd = loc[0]
		}
	}

	// Stop at quality indicators
	qualityPatterns := []string{"2160p", "1080p", "720p", "480p", "HDTV", "BluRay", "WEBRip", "WEB-DL", "REMUX"}
	for _, q := range qualityPatterns {
		qPattern := regexp.MustCompile(`(?i)\b` + q + `\b`)
		if loc := qPattern.FindStringIndex(cleanName); loc != nil && loc[0] < titleEnd {
			titleEnd = loc[0]
		}
	}

	title := strings.TrimSpace(cleanName[:titleEnd])
	return title
}

// IsTrustedGroup checks if a release group is trusted
func IsTrustedGroup(group, category string) bool {
	groups, ok := TrustedGroups[category]
	if !ok {
		groups = TrustedGroups["movies"] // Default
	}
	groupLower := strings.ToLower(group)
	for _, g := range groups {
		if strings.ToLower(g) == groupLower {
			return true
		}
	}
	return false
}

// IsBlockedGroup checks if a release group is blocked
func IsBlockedGroup(group string) bool {
	groupLower := strings.ToLower(group)
	for _, g := range BlockedGroups {
		if strings.ToLower(g) == groupLower {
			return true
		}
	}
	return false
}
