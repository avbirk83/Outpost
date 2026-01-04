package parser

import (
	"regexp"
	"strconv"
	"strings"
)

// ParsedRelease contains all extracted info from a release name
type ParsedRelease struct {
	// Identification
	Title string
	Year  int

	// TV specific
	Season       int  // 0 if movie
	Episode      int  // 0 if movie or season pack
	EpisodeEnd   int  // For multi-episode (S01E01E02)
	IsSeasonPack bool
	IsDailyShow  bool
	AirDate      string // For daily shows: "2024-01-15"
	Volume       int    // Netflix-style volumes
	Part         int    // Part releases

	// Anime specific
	IsAnime           bool
	IsAbsoluteEpisode bool // episode 45 vs S02E05
	Version           int  // v1, v2, v3 — default 1
	IsBatch           bool
	IsOVA             bool
	IsONA             bool
	IsOAD             bool
	HasDualAudio      bool
	HasSoftSubs       bool
	IsFansub          bool

	// Video quality
	Resolution string // "2160p", "1080p", "720p", "480p"
	Source     string // "remux", "bluray", "webdl", "webrip", "hdtv", "dvd", "satellite", "uhdtv", "ppv"
	Codec      string // "hevc", "avc", "av1", "vp9", "xvid", "mpeg2", "vc1"
	BitDepth   int    // 8 or 10
	HDR        string // "dv", "hdr10plus", "hdr10", "hlg", "sdr", ""

	// 3D
	Is3D     bool
	Format3D string // "sbs", "hsbs", "ou", "hou", "mvc"

	// Audio
	AudioFormat   string // "atmos", "truehd", "dtshd", "dtsx", "flac", "pcm", "ddplus", "dts", "dd", "aac", "opus"
	AudioChannels string // "7.1", "5.1", "2.0"

	// Language
	Languages     []string // ISO codes
	HasMultiAudio bool
	IsDubbed      bool

	// Subtitles
	HasSubtitles      bool
	SubtitleLanguages []string
	HasHardcodedSubs  bool
	HasMultipleSubs   bool

	// Edition
	Edition string // "extended", "directors", "theatrical", "unrated", "remastered", "imax", "criterion", "ultimate", "collectors", "anniversary", "special", "openmatte"

	// Aspect ratio
	IsFullscreen bool // cropped — avoid

	// Metadata
	ReleaseGroup     string
	StreamingService string // "AMZN", "NF", "ATVP", "DSNP", "HMAX", "HULU", "PCOK", "PMTP", "iT"
	IsRepack         bool
	IsProper         bool
	IsReal           bool // "REAL" tag
	IsInternal       bool
	IsLimited        bool
	IsRemastered     bool

	// Warnings (reasons to block)
	IsBlockedSource   bool // CAM, TS, etc.
	IsBlockedAudio    bool // MD, LD, etc.
	IsUpscaled        bool // Fake 4K
	IsSample          bool
	IsBlockedGroup    bool
	IsNuked           bool
	IsWorkprint       bool
	IsFastsub         bool
	IsCompressedAudio bool // MD, LD, LiNE

	// Scene fix tags
	IsRerip   bool // New rip of source
	IsDirfix  bool // Fixed directory naming
	IsSyncfix bool // Audio sync fixed
	IsDS4K    bool // Downscaled from 4K (good quality indicator)

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
		"SubsPlease", "Erai-raws", "EMBER", "Judas", "ASW",
		"Tsundere", "Commie", "GJM", "Kametsu",
	},
}

// Blocked release groups
var BlockedGroups = []string{
	"YIFY", "YTS", "RARBG", "TGx", "MeGusta",
	"STUTTERSHIT", "aXXo", "SPARKS-encoded",
}

// Streaming services
var streamingServices = map[string]string{
	"AMZN": "Amazon Prime",
	"NF":   "Netflix",
	"ATVP": "Apple TV+",
	"DSNP": "Disney+",
	"HMAX": "HBO Max",
	"HULU": "Hulu",
	"PCOK": "Peacock",
	"PMTP": "Paramount+",
	"iT":   "iTunes",
	"ZEE5": "ZEE5",
	"ANGL": "Angel Studios",
}

// Regex patterns for parsing
var (
	// Title patterns
	yearPattern       = regexp.MustCompile(`\b(19[0-9]{2}|20[0-9]{2})\b`)
	tvShowPattern     = regexp.MustCompile(`(?i)S(\d{1,2})E(\d{1,3})(?:[-]?E?(\d{1,3}))?`)
	seasonPackPattern = regexp.MustCompile(`(?i)S(\d{1,2})(?:[-.\s]?complete|$)`)
	dailyShowPattern  = regexp.MustCompile(`(\d{4})[\.-](\d{2})[\.-](\d{2})`)
	volumePattern     = regexp.MustCompile(`(?i)Vol(?:ume)?[\.\s-]?(\d+)`)
	partPattern       = regexp.MustCompile(`(?i)Part[\.\s-]?(\d+)|P(\d{2})`)

	// Anime patterns
	animeEpisodePattern = regexp.MustCompile(`(?i)\[([^\]]+)\]\s*([^-\[]+)\s*-\s*(\d+)(?:v(\d+))?`)
	animeBatchPattern   = regexp.MustCompile(`(?i)(\d+)[-](\d+)\s*\[.*\].*\[?Batch\]?`)
	animeVersionPattern = regexp.MustCompile(`(?i)v(\d+)`)
	absoluteEpPattern   = regexp.MustCompile(`(?i)-\s*(\d{2,4})(?:\s*v\d+)?\s*[\[\(]`)

	// Resolution patterns
	res4KPattern   = regexp.MustCompile(`(?i)2160p|4k|uhd`)
	res1080Pattern = regexp.MustCompile(`(?i)1080[pi]`)
	res720Pattern  = regexp.MustCompile(`(?i)720p`)
	res480Pattern  = regexp.MustCompile(`(?i)480p|dvdrip|sdtv`)

	// Source patterns
	remuxPattern     = regexp.MustCompile(`(?i)remux`)
	blurayPattern    = regexp.MustCompile(`(?i)blu-?ray|bdrip|brrip|\bBD\b`)
	webdlPattern     = regexp.MustCompile(`(?i)web-?dl|webdl`)
	webripPattern    = regexp.MustCompile(`(?i)webrip`)
	hdtvPattern      = regexp.MustCompile(`(?i)hdtv`)
	pdtvPattern      = regexp.MustCompile(`(?i)pdtv`)
	satellitePattern = regexp.MustCompile(`(?i)\b(dsr|satrip)\b`)
	uhdtvPattern     = regexp.MustCompile(`(?i)uhdtv`)
	ppvPattern       = regexp.MustCompile(`(?i)\bppv\b`)
	dvdPattern       = regexp.MustCompile(`(?i)dvdrip|\bdvd\b`)
	retailPattern    = regexp.MustCompile(`(?i)\bretail\b`)
	hybridPattern    = regexp.MustCompile(`(?i)\bhybrid\b`)

	// Blocked sources
	camPattern       = regexp.MustCompile(`(?i)\b(cam|hdcam)\b`)
	telesyncPattern  = regexp.MustCompile(`(?i)\b(ts|hdts|telesync)\b`)
	telecinePattern  = regexp.MustCompile(`(?i)\b(tc|telecine)\b`)
	screenerPattern  = regexp.MustCompile(`(?i)\b(scr|screener|dvdscr)\b`)
	r5Pattern        = regexp.MustCompile(`(?i)\br5\b`)
	workprintPattern = regexp.MustCompile(`(?i)\bworkprint\b`)

	// HDR patterns
	dvPattern     = regexp.MustCompile(`(?i)\b(dv|dolby[\.\s-]?vision|dovi)\b`)
	hdr10pPattern = regexp.MustCompile(`(?i)hdr10\+|hdr10plus`)
	hdr10Pattern  = regexp.MustCompile(`(?i)\bhdr10?\b`)
	hlgPattern    = regexp.MustCompile(`(?i)\bhlg\b`)
	sdrPattern    = regexp.MustCompile(`(?i)\bsdr\b`)

	// Audio patterns
	atmosPattern     = regexp.MustCompile(`(?i)\batmos\b`)
	truehdPattern    = regexp.MustCompile(`(?i)truehd|true-hd`)
	dtshdPattern     = regexp.MustCompile(`(?i)dts-?hd|dts-?ma|dts-?hd\.?ma`)
	dtsxPattern      = regexp.MustCompile(`(?i)dts-?x|dts:x`)
	flacPattern      = regexp.MustCompile(`(?i)\bflac\b`)
	pcmPattern       = regexp.MustCompile(`(?i)\b(lpcm|pcm)\b`)
	ddplusPattern    = regexp.MustCompile(`(?i)dd\+|ddp|ddpa|eac3|e-ac-3`)
	dtsPattern       = regexp.MustCompile(`(?i)\bdts\b`)
	ddPattern        = regexp.MustCompile(`(?i)(?:\bdd\s*5[\.\s]?1)|(?:ac-?3)`)
	aacPattern       = regexp.MustCompile(`(?i)\baac\b`)
	opusPattern      = regexp.MustCompile(`(?i)\bopus\b`)
	compAudioPattern = regexp.MustCompile(`(?i)\b(md|mic[\.\s-]?dub|line|ld)\b`)
	dubbedPattern    = regexp.MustCompile(`(?i)\bdubbed\b`)

	// Audio channels
	channels71Pattern = regexp.MustCompile(`(?i)7[\.\s]1|8ch`)
	channels51Pattern = regexp.MustCompile(`(?i)5[\.\s]1|6ch`)
	channels20Pattern = regexp.MustCompile(`(?i)2[\.\s]0|2ch|stereo`)

	// Codec patterns
	hevcPattern  = regexp.MustCompile(`(?i)\b(hevc|x265|h\.?265)\b`)
	avcPattern   = regexp.MustCompile(`(?i)\b(x264|h\.?264|avc)\b`)
	av1Pattern   = regexp.MustCompile(`(?i)\bav1\b`)
	vp9Pattern   = regexp.MustCompile(`(?i)\bvp9\b`)
	xvidPattern  = regexp.MustCompile(`(?i)\bxvid\b`)
	mpeg2Pattern = regexp.MustCompile(`(?i)\b(mpeg-?2)\b`)
	vc1Pattern   = regexp.MustCompile(`(?i)\bvc-?1\b`)

	// Bit depth
	bit10Pattern = regexp.MustCompile(`(?i)10[\.\s-]?bits?`)
	bit8Pattern  = regexp.MustCompile(`(?i)8[\.\s-]?bits?`)

	// Edition patterns
	directorsPattern   = regexp.MustCompile(`(?i)director'?s?[\.\s-]?cut|\bdc\b`)
	extendedPattern    = regexp.MustCompile(`(?i)\bextended\b`)
	theatricalPattern  = regexp.MustCompile(`(?i)\btheatrical\b`)
	unratedPattern     = regexp.MustCompile(`(?i)\bunrated\b`)
	remasteredPattern  = regexp.MustCompile(`(?i)\bremastered\b`)
	imaxPattern        = regexp.MustCompile(`(?i)\bimax\b`)
	criterionPattern   = regexp.MustCompile(`(?i)\b(criterion|cc)\b`)
	ultimatePattern    = regexp.MustCompile(`(?i)\bultimate\b`)
	collectorsPattern  = regexp.MustCompile(`(?i)\bcollector'?s?\b`)
	anniversaryPattern = regexp.MustCompile(`(?i)\banniversary\b`)
	specialEdPattern   = regexp.MustCompile(`(?i)special[\.\s-]?edition`)
	openMattePattern   = regexp.MustCompile(`(?i)open[\.\s-]?matte`)

	// 3D patterns
	threeDPattern = regexp.MustCompile(`(?i)\b3d\b`)
	sbsPattern    = regexp.MustCompile(`(?i)\bsbs\b`)
	hsbsPattern   = regexp.MustCompile(`(?i)\bhsbs\b`)
	ouPattern     = regexp.MustCompile(`(?i)\b(ou|tab)\b`)
	houPattern    = regexp.MustCompile(`(?i)\bhou\b`)
	mvcPattern    = regexp.MustCompile(`(?i)\bmvc\b`)

	// Aspect patterns
	fullscreenPattern = regexp.MustCompile(`(?i)\b(fs|fullscreen)\b`)

	// Language patterns
	dualPattern  = regexp.MustCompile(`(?i)\bdual\b`)
	multiPattern = regexp.MustCompile(`(?i)\bmulti\b`)

	// Subtitle patterns
	hardcodedPattern = regexp.MustCompile(`(?i)\b(hc|hardcoded|hard[\.\s-]?coded|korsub)\b`)
	subbedPattern    = regexp.MustCompile(`(?i)\b(subbed|esub|esubs|msub|msubs)\b`)
	softsubPattern   = regexp.MustCompile(`(?i)\bsoftsub\b`)
	hardsubPattern   = regexp.MustCompile(`(?i)\bhardsub\b`)
	fansubPattern    = regexp.MustCompile(`(?i)\bfansub\b`)
	fastsubPattern   = regexp.MustCompile(`(?i)\bfastsub\b`)

	// Scene tags
	properPattern   = regexp.MustCompile(`(?i)\bproper\b`)
	repackPattern   = regexp.MustCompile(`(?i)\brepack\b`)
	realPattern     = regexp.MustCompile(`(?i)\breal\b`)
	reripPattern    = regexp.MustCompile(`(?i)\brerip\b`)
	dirfixPattern   = regexp.MustCompile(`(?i)\bdirfix\b`)
	syncfixPattern  = regexp.MustCompile(`(?i)\bsyncfix\b`)
	internalPattern = regexp.MustCompile(`(?i)\binternal\b`)
	limitedPattern  = regexp.MustCompile(`(?i)\blimited\b`)
	nukedPattern    = regexp.MustCompile(`(?i)\bnuked\b`)
	ds4kPattern     = regexp.MustCompile(`(?i)\bds4k\b`)

	// Bad patterns
	samplePattern   = regexp.MustCompile(`(?i)\bsample\b`)
	upscaledPattern = regexp.MustCompile(`(?i)\b(upscale|upscaled)\b`)

	// Anime types
	ovaPattern = regexp.MustCompile(`(?i)\bova\b`)
	onaPattern = regexp.MustCompile(`(?i)\bona\b`)
	oadPattern = regexp.MustCompile(`(?i)\boad\b`)

	// Release group patterns
	groupPattern        = regexp.MustCompile(`-([a-zA-Z0-9]+)(?:\.[a-z]+)?$`)
	animeGroupPattern   = regexp.MustCompile(`^\[([^\]]+)\]`)
	streamServicePattern = regexp.MustCompile(`(?i)\b(AMZN|NF|ATVP|DSNP|HMAX|HULU|PCOK|PMTP|iT|ZEE5|ANGL)\b`)
)

// Language code mappings
var languageCodes = map[string]string{
	"ENG": "en", "ENGLISH": "en", "EN": "en",
	"ITA": "it", "ITALIAN": "it",
	"SPA": "es", "SPANISH": "es", "ESP": "es", "LATIN": "es", "LAT": "es", "CASTELLANO": "es", "CAST": "es",
	"FRA": "fr", "FRE": "fr", "FRENCH": "fr", "VFF": "fr", "VFQ": "fr", "VF": "fr",
	"DEU": "de", "GER": "de", "GERMAN": "de",
	"JPN": "ja", "JAP": "ja", "JAPANESE": "ja",
	"KOR": "ko", "KOREAN": "ko",
	"HIN": "hi", "HINDI": "hi",
	"RUS": "ru", "RUSSIAN": "ru",
	"POR": "pt", "PORTUGUESE": "pt",
	"POL": "pl", "POLISH": "pl",
	"NLD": "nl", "DUTCH": "nl",
	"SWE": "sv", "SWEDISH": "sv",
	"FIN": "fi", "FINNISH": "fi",
	"CZE": "cs", "CZECH": "cs",
	"HUN": "hu", "HUNGARIAN": "hu",
	"THA": "th", "THAI": "th",
	"VIE": "vi", "VIETNAMESE": "vi",
	"IND": "id", "INDONESIAN": "id",
	"ARA": "ar", "ARABIC": "ar",
	"HEB": "he", "HEBREW": "he",
	"TUR": "tr", "TURKISH": "tr",
	"GRE": "el", "GREEK": "el",
	"ROM": "ro", "ROMANIAN": "ro",
	"UKR": "uk", "UKRAINIAN": "uk",
	"DANISH": "da", "NORWEGIAN": "no", "TAGALOG": "tl",
}

// Parse extracts information from a release name
func Parse(name string) *ParsedRelease {
	r := &ParsedRelease{
		RawTitle: name,
		Version:  1, // Default version
		BitDepth: 8, // Default bit depth
	}

	// Clean name for parsing
	cleanName := strings.ReplaceAll(name, ".", " ")
	cleanName = strings.ReplaceAll(cleanName, "_", " ")

	// Check if anime (starts with [Group])
	if animeGroupPattern.MatchString(name) {
		r.IsAnime = true
		if matches := animeGroupPattern.FindStringSubmatch(name); matches != nil {
			r.ReleaseGroup = matches[1]
		}
	}

	// Extract TV show info
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

	// Check for anime absolute episode numbering
	if r.IsAnime && r.Season == 0 && r.Episode == 0 {
		if matches := absoluteEpPattern.FindStringSubmatch(name); matches != nil {
			r.Episode, _ = strconv.Atoi(matches[1])
			r.IsAbsoluteEpisode = true
		}
	}

	// Check for anime batch
	if animeBatchPattern.MatchString(name) || strings.Contains(strings.ToLower(name), "batch") {
		r.IsBatch = true
	}

	// Extract anime version
	if r.IsAnime {
		if matches := animeVersionPattern.FindStringSubmatch(name); matches != nil {
			r.Version, _ = strconv.Atoi(matches[1])
		}
	}

	// Extract volume
	if matches := volumePattern.FindStringSubmatch(name); matches != nil {
		r.Volume, _ = strconv.Atoi(matches[1])
	}

	// Extract part
	if matches := partPattern.FindStringSubmatch(name); matches != nil {
		if matches[1] != "" {
			r.Part, _ = strconv.Atoi(matches[1])
		} else if matches[2] != "" {
			r.Part, _ = strconv.Atoi(matches[2])
		}
	}

	// Extract year
	if matches := yearPattern.FindAllString(cleanName, -1); len(matches) > 0 {
		r.Year, _ = strconv.Atoi(matches[0])
	}

	// Extract title
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

	// Source - check blocked sources first
	switch {
	case camPattern.MatchString(name):
		r.Source = "cam"
		r.IsBlockedSource = true
	case telesyncPattern.MatchString(name):
		r.Source = "ts"
		r.IsBlockedSource = true
	case telecinePattern.MatchString(name):
		r.Source = "tc"
		r.IsBlockedSource = true
	case screenerPattern.MatchString(name):
		r.Source = "screener"
		r.IsBlockedSource = true
	case r5Pattern.MatchString(name):
		r.Source = "r5"
		r.IsBlockedSource = true
	case workprintPattern.MatchString(name):
		r.Source = "workprint"
		r.IsBlockedSource = true
		r.IsWorkprint = true
	case remuxPattern.MatchString(name):
		r.Source = "remux"
	case blurayPattern.MatchString(name) && !remuxPattern.MatchString(name):
		r.Source = "bluray"
	case webdlPattern.MatchString(name):
		r.Source = "webdl"
	case webripPattern.MatchString(name):
		r.Source = "webrip"
	case uhdtvPattern.MatchString(name):
		r.Source = "uhdtv"
	case hdtvPattern.MatchString(name):
		r.Source = "hdtv"
	case pdtvPattern.MatchString(name):
		r.Source = "pdtv"
	case satellitePattern.MatchString(name):
		r.Source = "satellite"
	case ppvPattern.MatchString(name):
		r.Source = "ppv"
	case dvdPattern.MatchString(name):
		r.Source = "dvd"
	case retailPattern.MatchString(name):
		r.Source = "retail"
	case hybridPattern.MatchString(name):
		r.Source = "hybrid"
	}

	// HDR
	switch {
	case dvPattern.MatchString(name):
		r.HDR = "dv"
	case hdr10pPattern.MatchString(name):
		r.HDR = "hdr10plus"
	case hdr10Pattern.MatchString(name) && !hdr10pPattern.MatchString(name):
		r.HDR = "hdr10"
	case hlgPattern.MatchString(name):
		r.HDR = "hlg"
	case sdrPattern.MatchString(name):
		r.HDR = "sdr"
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
	case flacPattern.MatchString(name):
		r.AudioFormat = "flac"
	case pcmPattern.MatchString(name):
		r.AudioFormat = "pcm"
	case ddplusPattern.MatchString(name):
		r.AudioFormat = "ddplus"
	case dtsPattern.MatchString(name) && !dtshdPattern.MatchString(name) && !dtsxPattern.MatchString(name):
		r.AudioFormat = "dts"
	case ddPattern.MatchString(name) && !ddplusPattern.MatchString(name):
		r.AudioFormat = "dd"
	case aacPattern.MatchString(name):
		r.AudioFormat = "aac"
	case opusPattern.MatchString(name):
		r.AudioFormat = "opus"
	}

	// Blocked audio
	if compAudioPattern.MatchString(name) {
		r.IsCompressedAudio = true
		r.IsBlockedAudio = true
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
	case vp9Pattern.MatchString(name):
		r.Codec = "vp9"
	case avcPattern.MatchString(name):
		r.Codec = "avc"
	case xvidPattern.MatchString(name):
		r.Codec = "xvid"
	case mpeg2Pattern.MatchString(name):
		r.Codec = "mpeg2"
	case vc1Pattern.MatchString(name):
		r.Codec = "vc1"
	}

	// Bit depth
	if bit10Pattern.MatchString(name) {
		r.BitDepth = 10
	} else if bit8Pattern.MatchString(name) {
		r.BitDepth = 8
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
	case remasteredPattern.MatchString(name):
		r.Edition = "remastered"
		r.IsRemastered = true
	case imaxPattern.MatchString(name):
		r.Edition = "imax"
	case criterionPattern.MatchString(name):
		r.Edition = "criterion"
	case ultimatePattern.MatchString(name):
		r.Edition = "ultimate"
	case collectorsPattern.MatchString(name):
		r.Edition = "collectors"
	case anniversaryPattern.MatchString(name):
		r.Edition = "anniversary"
	case specialEdPattern.MatchString(name):
		r.Edition = "special"
	case openMattePattern.MatchString(name):
		r.Edition = "openmatte"
	}

	// 3D
	if threeDPattern.MatchString(name) {
		r.Is3D = true
		switch {
		case hsbsPattern.MatchString(name):
			r.Format3D = "hsbs"
		case sbsPattern.MatchString(name):
			r.Format3D = "sbs"
		case houPattern.MatchString(name):
			r.Format3D = "hou"
		case ouPattern.MatchString(name):
			r.Format3D = "ou"
		case mvcPattern.MatchString(name):
			r.Format3D = "mvc"
		default:
			r.Format3D = "3d"
		}
	}

	// Streaming service
	if matches := streamServicePattern.FindStringSubmatch(name); matches != nil {
		r.StreamingService = strings.ToUpper(matches[1])
	}

	// Release group (if not already set by anime pattern)
	if r.ReleaseGroup == "" {
		if matches := groupPattern.FindStringSubmatch(name); matches != nil {
			r.ReleaseGroup = matches[1]
		}
	}

	// Scene tags
	r.IsProper = properPattern.MatchString(name)
	r.IsRepack = repackPattern.MatchString(name)
	r.IsReal = realPattern.MatchString(name)
	r.IsRerip = reripPattern.MatchString(name)
	r.IsDirfix = dirfixPattern.MatchString(name)
	r.IsSyncfix = syncfixPattern.MatchString(name)
	r.IsInternal = internalPattern.MatchString(name)
	r.IsLimited = limitedPattern.MatchString(name)
	r.IsNuked = nukedPattern.MatchString(name)
	r.IsDS4K = ds4kPattern.MatchString(name)

	// Warnings
	r.HasHardcodedSubs = hardcodedPattern.MatchString(name) || hardsubPattern.MatchString(name)
	r.IsSample = samplePattern.MatchString(name)
	r.IsUpscaled = upscaledPattern.MatchString(name)

	// Fullscreen (avoid)
	r.IsFullscreen = fullscreenPattern.MatchString(name)

	// Language
	r.HasMultiAudio = dualPattern.MatchString(name) || multiPattern.MatchString(name)
	r.IsDubbed = dubbedPattern.MatchString(name)
	r.Languages = extractLanguages(name)

	// Subtitles
	r.HasSubtitles = subbedPattern.MatchString(name)
	r.HasMultipleSubs = strings.Contains(strings.ToLower(name), "msub") || strings.Contains(strings.ToLower(name), "multiple subtitle")
	r.HasSoftSubs = softsubPattern.MatchString(name)
	r.IsFansub = fansubPattern.MatchString(name)
	r.IsFastsub = fastsubPattern.MatchString(name)

	// Anime types
	r.IsOVA = ovaPattern.MatchString(name)
	r.IsONA = onaPattern.MatchString(name)
	r.IsOAD = oadPattern.MatchString(name)

	// Dual audio for anime
	if strings.Contains(strings.ToLower(name), "dual-audio") || strings.Contains(strings.ToLower(name), "dual audio") {
		r.HasDualAudio = true
	}

	// Check blocked group
	r.IsBlockedGroup = IsBlockedGroup(r.ReleaseGroup)

	return r
}

// extractTitle tries to extract the clean title from a release name
func extractTitle(cleanName string, r *ParsedRelease) string {
	// For anime, extract differently
	if r.IsAnime {
		// Remove brackets at start
		title := regexp.MustCompile(`^\[[^\]]+\]\s*`).ReplaceAllString(cleanName, "")
		// Stop at episode number
		if idx := strings.Index(title, " - "); idx > 0 {
			title = title[:idx]
		}
		return strings.TrimSpace(title)
	}

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
	qualityPatterns := []string{"2160p", "1080p", "720p", "480p", "HDTV", "BluRay", "WEBRip", "WEB-DL", "REMUX", "AMZN", "NF", "DSNP", "HMAX", "ATVP", "HULU", "PCOK"}
	for _, q := range qualityPatterns {
		qPattern := regexp.MustCompile(`(?i)\b` + q + `\b`)
		if loc := qPattern.FindStringIndex(cleanName); loc != nil && loc[0] < titleEnd {
			titleEnd = loc[0]
		}
	}

	title := strings.TrimSpace(cleanName[:titleEnd])
	return title
}

// extractLanguages extracts language codes from a release name
func extractLanguages(name string) []string {
	var langs []string
	seen := make(map[string]bool)

	nameLower := strings.ToLower(name)

	for tag, code := range languageCodes {
		pattern := regexp.MustCompile(`(?i)\b` + tag + `\b`)
		if pattern.MatchString(name) {
			if !seen[code] {
				langs = append(langs, code)
				seen[code] = true
			}
		}
	}

	// Check for multi-language patterns like ITA.ENG
	multiLangPattern := regexp.MustCompile(`(?i)([A-Z]{2,3})\.([A-Z]{2,3})`)
	if matches := multiLangPattern.FindAllStringSubmatch(name, -1); matches != nil {
		for _, match := range matches {
			for _, tag := range match[1:] {
				if code, ok := languageCodes[strings.ToUpper(tag)]; ok {
					if !seen[code] {
						langs = append(langs, code)
						seen[code] = true
					}
				}
			}
		}
	}

	// If no language detected and it's not dubbed, assume English
	if len(langs) == 0 && !strings.Contains(nameLower, "dubbed") {
		langs = append(langs, "en")
	}

	return langs
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
	if group == "" {
		return false
	}
	groupLower := strings.ToLower(group)
	for _, g := range BlockedGroups {
		if strings.ToLower(g) == groupLower {
			return true
		}
	}
	return false
}

// ShouldBlock returns true if the release should be blocked
func (r *ParsedRelease) ShouldBlock() bool {
	return r.IsBlockedSource || r.IsBlockedAudio || r.IsBlockedGroup ||
		r.HasHardcodedSubs || r.IsUpscaled || r.IsSample || r.IsNuked || r.IsWorkprint
}

// BlockReason returns the reason why a release should be blocked
func (r *ParsedRelease) BlockReason() string {
	switch {
	case r.IsBlockedSource:
		return "blocked_source"
	case r.IsBlockedAudio:
		return "blocked_audio"
	case r.IsBlockedGroup:
		return "blocked_group"
	case r.HasHardcodedSubs:
		return "hardcoded_subs"
	case r.IsUpscaled:
		return "upscaled"
	case r.IsSample:
		return "sample"
	case r.IsNuked:
		return "nuked"
	case r.IsWorkprint:
		return "workprint"
	default:
		return ""
	}
}
