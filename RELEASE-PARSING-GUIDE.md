# Release Parsing Tag Reference

## Overview

This document defines all tags and patterns Outpost should recognize when parsing release names from indexers. Use this as the source of truth for building the release parser.

---

## Resolution

| Tag | Normalized Value |
|-----|------------------|
| 4K, 2160p, UHD | 2160p |
| 1080p, 1080i | 1080p |
| 720p | 720p |
| 480p, SD, DVDRip, SDTV | 480p |

---

## Source

| Tag | Normalized Value | Notes |
|-----|------------------|-------|
| Remux | remux | Highest quality |
| BluRay, Blu-Ray, BDRip, BRRip | bluray | |
| WEB-DL, WEBDL, WEB | webdl | Direct download |
| WEBRip | webrip | Screen capture |
| HDTV | hdtv | |
| PDTV | pdtv | Digital TV capture |
| DSR, SATRip | satellite | Satellite rip |
| UHDTV | uhdtv | 4K broadcast |
| PPV | ppv | Pay-per-view |
| DVDRip | dvd | |
| BD | bluray | Blu-ray (anime term) |
| RETAIL | retail | Official retail release |
| HYBRID | hybrid | Mixed sources |

### Blocked Sources (Never Grab)

| Tag | Why |
|-----|-----|
| CAM, HDCAM | Theater recording |
| TS, HDTS, Telesync | Theater recording |
| TC, Telecine | Theater recording |
| SCR, Screener, DVDScr | Pre-release leak |
| R5 | Low quality Russian release |
| WORKPRINT | Unfinished version |

---

## Streaming Services

| Tag | Service |
|-----|---------|
| AMZN | Amazon Prime |
| NF | Netflix |
| ATVP | Apple TV+ |
| DSNP | Disney+ |
| HMAX | HBO Max |
| HULU | Hulu |
| PCOK | Peacock |
| PMTP | Paramount+ |
| iT | iTunes |
| ZEE5 | ZEE5 |
| ANGL | Angel Studios |

---

## Video Codec

| Tag | Normalized Value |
|-----|------------------|
| x265, H.265, H265, HEVC | hevc |
| x264, H.264, H264, AVC | avc |
| AV1 | av1 |
| VP9 | vp9 |
| XviD | xvid |
| MPEG2, MPEG-2 | mpeg2 |
| VC-1 | vc1 |

---

## Bit Depth

| Tag | Value |
|-----|-------|
| 10bit, 10bits, 10Bit, 10-bit | 10 |
| 8bit | 8 |

---

## HDR Format

| Tag | Normalized Value | Priority |
|-----|------------------|----------|
| DV, DoVi, Dolby Vision | dv | 1 (best) |
| HDR10+, HDR10Plus | hdr10plus | 2 |
| HDR10, HDR | hdr10 | 3 |
| HLG | hlg | 4 |
| SDR | sdr | 5 |

---

## Audio Format

| Tag | Normalized Value | Priority |
|-----|------------------|----------|
| Atmos | atmos | 1 (best) |
| TrueHD, True-HD | truehd | 2 |
| DTS-HD, DTS-HD MA, DTS-MA | dtshd | 2 |
| DTS-X, DTS:X | dtsx | 2 |
| FLAC | flac | 3 (lossless) |
| LPCM, PCM | pcm | 3 (uncompressed) |
| DDP, DD+, DDPA, EAC3, E-AC-3 | ddplus | 4 |
| DTS | dts | 5 |
| DD, AC3, AC-3 | dd | 6 |
| AAC | aac | 7 |
| Opus | opus | 7 |

### Blocked Audio (Never Grab)

| Tag | Why |
|-----|-----|
| MD, Mic Dub | Microphone recorded in theater |
| LD, Line Dub | Poor line-in recording |
| LiNE | Poor quality audio source |
| DUBBED | Audio replaced (unless user wants dubbed) |

---

## Audio Channels

| Tag | Normalized Value |
|-----|------------------|
| 7.1, 8CH | 7.1 |
| 5.1, 6CH | 5.1 |
| 2.0, 2CH, Stereo | 2.0 |

---

## Edition

| Tag | Normalized Value |
|-----|------------------|
| Extended | extended |
| Director's Cut, DC | directors |
| Theatrical | theatrical |
| Unrated | unrated |
| Remastered | remastered |
| IMAX | imax |
| Criterion, CC | criterion |
| Ultimate | ultimate |
| Collector's, Collectors | collectors |
| Anniversary | anniversary |
| Special Edition | special |
| Open Matte | openmatte |

---

## Aspect Ratio / Frame

| Tag | Normalized Value | Notes |
|-----|------------------|-------|
| WS, Widescreen | widescreen | Normal |
| FS, Fullscreen | fullscreen | AVOID - cropped |
| Open Matte | openmatte | Full frame version |

---

## TV Episode Patterns

### Single Episode
```
S01E01
S1E1
```

### Multi-Episode
```
S01E01E02       → Episodes 1-2
S01E01-E02      → Episodes 1-2
S01E01-02       → Episodes 1-2
S01E05-E07      → Episodes 5-7
```

### Season Pack
```
S01.Complete
S01 Complete
S01 Season 1
Season 1 COMPLETE
S01 (with no episode number)
```

### Volume/Part (Netflix Style)
```
Vol.1, Vol.2, Volume 1
Part 1, Part 2, P01, P02
Parte.01, Parte.02 (Italian)
S05.Vol.2.E05-E07
```

---

## Language Tags

| Tag | Language Code |
|-----|---------------|
| ENG, English, Eng | en |
| ITA, Italian | it |
| SPA, Spanish, ESP | es |
| FRA, FRE, French | fr |
| DEU, GER, German | de |
| JPN, JAP, Japanese | ja |
| KOR, Korean | ko |
| HIN, Hindi | hi |
| Tagalog | tl |
| Danish | da |
| Norwegian | no |
| RUS, Russian | ru |
| POR, Portuguese | pt |
| POL, Polish | pl |
| NLD, Dutch | nl |
| SWE, Swedish | sv |
| FIN, Finnish | fi |
| CZE, Czech | cs |
| HUN, Hungarian | hu |
| THA, Thai | th |
| VIE, Vietnamese | vi |
| IND, Indonesian | id |
| ARA, Arabic | ar |
| HEB, Hebrew | he |
| TUR, Turkish | tr |
| GRE, Greek | el |
| ROM, Romanian | ro |
| UKR, Ukrainian | uk |

### Multi-Language Indicators
```
DUAL        → Two audio tracks
Multi       → Multiple audio tracks
ITA.ENG     → Italian and English
```

### Foreign Language Tags (Specialized)

| Tag | Meaning |
|-----|---------|
| VFF | French audio from France |
| VFQ | French audio from Quebec |
| VOSTFR | Original audio + French subs |
| VO | Original version |
| VF | French version |
| LATIN, LAT | Latin American Spanish |
| CASTELLANO, CAST | Castilian Spanish |
| NORDIC | Nordic languages package |
| SUBBED | Has subtitles |

---

## Scene Tags

| Tag | Meaning | Action |
|-----|---------|--------|
| INTERNAL | Scene internal release | Neutral |
| LIMITED | Limited theater release | Neutral |
| PROPER | Fixes previous release | Prefer |
| REPACK | Fixes previous release | Prefer |
| REAL | Authentic release | Prefer |
| RERIP | New rip of source | Prefer |
| DIRFIX | Fixed directory naming | Neutral |
| SYNCFIX | Audio sync fixed | Prefer |
| READNFO | Has NFO file | Ignore |
| NUKED | Scene rejected release | BLOCK |

---

## 3D Formats

| Tag | Normalized Value | Notes |
|-----|------------------|-------|
| 3D | 3d | Generic 3D |
| SBS | sbs | Side by Side |
| HSBS | hsbs | Half Side by Side |
| OU, TAB | ou | Over-Under / Top and Bottom |
| HOU | hou | Half Over-Under |
| MVC | mvc | Multi-view codec |

Note: Block 3D by default unless user has 3D enabled in preferences.

---

## Anime-Specific Tags

### Source (Anime)

| Tag | Meaning |
|-----|---------|
| BD | Blu-ray Disc |
| BDMV | Blu-ray disc structure |
| BDREMUX | Blu-ray remux |
| DVD | DVD source |
| TV | Television broadcast |
| WEB | Web release |

### Type (Anime)

| Tag | Normalized Value | Notes |
|-----|------------------|-------|
| OVA | ova | Original Video Animation |
| ONA | ona | Original Net Animation |
| OAD | oad | Original Animation DVD |
| Movie | movie | Theatrical film |
| Special | special | Special episode |
| Batch | batch | Multiple episodes in one release |

### Version (Anime)

| Tag | Meaning |
|-----|---------|
| v0 | Initial release (may have issues) |
| v1 | First version (normal) |
| v2, v3, v4 | Fixed versions (prefer higher) |

### Subtitle Type (Anime)

| Tag | Meaning | Action |
|-----|---------|--------|
| FANSUB | Fan subtitled | Neutral |
| HARDSUB | Hardcoded subs | BLOCK |
| SOFTSUB | Soft subtitles | Prefer |
| FASTSUB | Quick/speed sub | Avoid (lower quality) |
| DUAL-AUDIO | Japanese + English audio | Prefer |

### Trusted Anime Groups

```
SubsPlease      → Fast, reliable simulcast rips
Erai-raws       → Multi-language subs
EMBER           → Quality encodes
Judas           → Good dual-audio
ASW             → Quality mini-encodes
Tsundere        → Good quality
Commie          → Meme subs but accurate
GJM             → High quality
Kametsu         → Dual-audio, quality
```

### Anime Episode Patterns

```
[Group] Title - 01 [1080p]              → Episode 1
[Group] Title - 01v2 [1080p]            → Episode 1, version 2
[Group] Title - 01-12 [1080p] [Batch]   → Episodes 1-12 batch
[Group] Title - S01E01 [1080p]          → Season 1 Episode 1 (western style)
Title - 001 [1080p]                     → Absolute episode numbering
```

---

## Subtitle Tags

| Tag | Meaning |
|-----|---------|
| ESub, ESubs, E-Sub | English subtitles included |
| MSub, MSubs | Multiple subtitle languages |
| Subbed | Has subtitles |

### Blocked Subtitles

| Tag | Why |
|-----|-----|
| HC, Hardcoded | Burned into video, can't disable |
| Hard-Coded | Same |
| KorSub | Usually hardcoded Korean |

---

## Quality/Fix Tags

| Tag | Meaning | Action |
|-----|---------|--------|
| REPACK | Fixed release | Prefer |
| PROPER | Correct release | Prefer |
| REAL | Authentic release | Prefer |
| RERIP | New rip | Prefer |
| Remastered | Updated source | Prefer |
| DS4K | Downscaled from 4K | Good quality |
| v2, v3 | Fixed version | Prefer higher |

### Blocked Quality Tags

| Tag | Why |
|-----|-----|
| Upscaled, Upscale | Fake resolution increase |
| 3D | Unless user specifically wants |
| Sample | Not full file |
| NUKED | Scene rejected |
| FASTSUB | Rush job, errors |
| FS, Fullscreen | Cropped |
| DVDSCR | Pre-release screener |
| WORKPRINT | Unfinished |

---

## Blocked File Types

If release contains these, block and optionally auto-block the group:

```
.rar
.iso
.exe
.zip (unless contains mkv/mp4)
password protected
```

---

## Release Group

Location in filename:
```
-GroupName      (at end, after hyphen)
[GroupName]     (in brackets)
```

Examples:
```
Movie.2024.1080p.WEB-DL-FLUX         → FLUX
Movie.2024.1080p.WEB-DL.x265-NeoNoir → NeoNoir
[QxR]                                 → QxR
```

### Default Blocked Groups

```
YIFY
YTS
aXXo
```

User can add additional blocked groups in settings.

System should auto-block groups that upload RAR/ISO files.

---

## Quality Presets

Presets are starting points. Users can edit any preset to customize preferences.

### Built-in Presets

All presets are editable. Users can modify any field or create custom presets.

#### Best Quality
```
Name: Best Quality
Resolution: 2160p
Min Resolution: 1080p (will accept lower if nothing else)
Source: remux, bluray
Codec: hevc, av1
HDR: dv, hdr10plus, hdr10 (prefer in this order)
Audio: atmos, truehd, dtshd, dtsx (prefer in this order)
Audio Channels: 7.1, 5.1
Edition: any
Min Seeders: 3
Auto-upgrade: true
```

#### High Quality
```
Name: High Quality
Resolution: 2160p
Min Resolution: 1080p
Source: webdl, webrip, bluray
Codec: hevc, avc, av1
HDR: any (accept SDR)
Audio: any
Audio Channels: 5.1, 7.1, 2.0
Edition: any
Min Seeders: 3
Auto-upgrade: true
```

#### Balanced
```
Name: Balanced
Resolution: 1080p
Min Resolution: 720p
Source: webdl, webrip
Codec: hevc, avc
HDR: any
Audio: any
Audio Channels: any
Edition: any
Min Seeders: 3
Auto-upgrade: false
```

#### Storage Saver
```
Name: Storage Saver
Resolution: 1080p
Min Resolution: 720p
Source: webdl, webrip
Codec: hevc, av1 (prefer smaller files)
HDR: sdr (avoid large HDR files)
Audio: ddplus, aac (avoid lossless)
Audio Channels: 5.1, 2.0
Edition: any
Min Seeders: 3
Prefer Smaller Size: true
Auto-upgrade: false
```

#### Anime
```
Name: Anime
Resolution: 1080p
Min Resolution: 720p
Source: bluray, webdl, webrip
Codec: hevc, avc
HDR: any
Audio: flac, aac, opus (anime often uses these)
Audio Channels: 2.0, 5.1
Languages: ja (Japanese required)
Subtitles: en (English required)
Prefer Dual-Audio: true
Prefer Soft Subs: true
Prefer Higher Version: true (v2 over v1)
Min Seeders: 2 (anime has fewer seeders)
Auto-upgrade: true
```

### Preset Settings UI

```
┌─────────────────────────────────────────────────────────┐
│  Quality Presets                                        │
│                                                         │
│  ┌─────────────────────────────────────────────────────┐
│  │ ● Best Quality                            [Edit]    │
│  │   4K Remux · Atmos · DV                             │
│  └─────────────────────────────────────────────────────┘
│  ┌─────────────────────────────────────────────────────┐
│  │ ○ High Quality                            [Edit]    │
│  │   4K WEB-DL · Any HDR · Any Audio                   │
│  └─────────────────────────────────────────────────────┘
│  ┌─────────────────────────────────────────────────────┐
│  │ ○ Balanced                                [Edit]    │
│  │   1080p WEB-DL                                      │
│  └─────────────────────────────────────────────────────┘
│  ┌─────────────────────────────────────────────────────┐
│  │ ○ Storage Saver                           [Edit]    │
│  │   1080p HEVC · Small files                          │
│  └─────────────────────────────────────────────────────┘
│  ┌─────────────────────────────────────────────────────┐
│  │ ○ Anime                                   [Edit]    │
│  │   1080p · Dual Audio · Soft Subs                    │
│  └─────────────────────────────────────────────────────┘
│                                                         │
│  [+ Create Custom Preset]                               │
│                                                         │
└─────────────────────────────────────────────────────────┘
```

### Preset Edit UI

```
┌─────────────────────────────────────────────────────────┐
│  Edit Preset: Best Quality                              │
│                                                         │
│  Name: [Best Quality                              ]     │
│                                                         │
│  ─── Video ───────────────────────────────────────────  │
│                                                         │
│  Preferred Resolution:  [2160p ▾]                       │
│  Minimum Resolution:    [1080p ▾]                       │
│                                                         │
│  Source (select all acceptable):                        │
│  ☑ Remux  ☑ Bluray  ☐ WEB-DL  ☐ WEBRip  ☐ HDTV       │
│                                                         │
│  Codec:                                                 │
│  ☑ HEVC  ☐ AVC  ☑ AV1                                  │
│                                                         │
│  HDR (prefer order, drag to reorder):                   │
│  [1] Dolby Vision                                       │
│  [2] HDR10+                                             │
│  [3] HDR10                                              │
│  ☐ Accept SDR if no HDR available                      │
│                                                         │
│  ─── Audio ───────────────────────────────────────────  │
│                                                         │
│  Format (prefer order, drag to reorder):                │
│  [1] Atmos                                              │
│  [2] TrueHD                                             │
│  [3] DTS-HD MA                                          │
│  [4] DTS-X                                              │
│  ☐ Accept lossy if no lossless available               │
│                                                         │
│  Channels:                                              │
│  ☑ 7.1  ☑ 5.1  ☐ 2.0                                   │
│                                                         │
│  ─── Downloads ───────────────────────────────────────  │
│                                                         │
│  Minimum seeders:  [3 ▾]                               │
│  ☑ Auto-upgrade when better quality available          │
│                                                         │
│  ─── Edition ─────────────────────────────────────────  │
│                                                         │
│  Preferred edition:  [Any ▾]                           │
│                                                         │
│                          [Cancel]  [Save]               │
└─────────────────────────────────────────────────────────┘
```

### Anime Preset Edit UI (Additional Fields)

```
│  ─── Anime-Specific ──────────────────────────────────  │
│                                                         │
│  ☑ Prefer dual-audio releases                          │
│  ☑ Prefer soft subtitles                               │
│  ☑ Prefer higher version (v2 > v1)                     │
│  ☐ Accept fansubs                                       │
│  ☐ Accept hardsubs (not recommended)                   │
│                                                         │
│  Episode format:                                        │
│  ○ Absolute numbering (001, 002, 003)                  │
│  ● Season numbering (S01E01, S01E02)                   │
```

### Per-Library Preset Assignment

Users should be able to assign different presets to different libraries:

```
┌─────────────────────────────────────────────────────────┐
│  Libraries                                              │
│                                                         │
│  Movies          Preset: [Best Quality ▾]              │
│  TV Shows        Preset: [High Quality ▾]              │
│  Anime           Preset: [Anime ▾]                     │
│  Kids            Preset: [Balanced ▾]                  │
│                                                         │
└─────────────────────────────────────────────────────────┘
```

### Per-Item Override

On any movie/show detail page, user can override the preset:

```
┌─────────────────────────────────────────────────────────┐
│  Quality                                                │
│                                                         │
│  Using: Library default (Best Quality)                 │
│                                                         │
│  ○ Use library default                                  │
│  ● Override for this item: [Storage Saver ▾]           │
│                                                         │
└─────────────────────────────────────────────────────────┘
```

---

## Parsed Release Object

The parser should output an object with these fields:

```
ParsedRelease {
    // Identification
    title: string
    year: number
    
    // TV specific
    season: number (0 if movie)
    episodes: number[] (empty if movie)
    isSeasonPack: boolean
    volume: number (0 if none)
    part: number (0 if none)
    
    // Anime specific
    isAnime: boolean
    isAbsoluteEpisode: boolean (episode 45 vs S02E05)
    version: number (v1, v2, v3 — default 1)
    isBatch: boolean
    isOVA: boolean
    isONA: boolean
    hasDualAudio: boolean
    hasSoftSubs: boolean
    isFansub: boolean
    
    // Video quality
    resolution: "2160p" | "1080p" | "720p" | "480p"
    source: "remux" | "bluray" | "webdl" | "webrip" | "hdtv" | "dvd" | "satellite"
    codec: "hevc" | "avc" | "av1" | "vp9" | "xvid" | "mpeg2"
    bitDepth: 8 | 10
    hdr: "dv" | "hdr10plus" | "hdr10" | "hlg" | "sdr" | null
    
    // 3D
    is3D: boolean
    format3D: "sbs" | "hsbs" | "ou" | "hou" | null
    
    // Audio
    audioFormat: "atmos" | "truehd" | "dtshd" | "dtsx" | "flac" | "ddplus" | "dts" | "dd" | "aac" | "opus"
    audioChannels: "7.1" | "5.1" | "2.0"
    
    // Language
    languages: string[] (ISO codes)
    hasMultiAudio: boolean
    isDubbed: boolean
    
    // Subtitles
    hasSubtitles: boolean
    subtitleLanguages: string[]
    hasHardcodedSubs: boolean
    hasMultipleSubs: boolean
    
    // Edition
    edition: "extended" | "directors" | "theatrical" | "unrated" | "remastered" | "imax" | "criterion" | "ultimate" | "collectors" | "anniversary" | "special" | "openmatte" | null
    
    // Aspect ratio
    isFullscreen: boolean (cropped — avoid)
    
    // Metadata
    releaseGroup: string
    streamingService: string (AMZN, NF, etc.)
    isRepack: boolean
    isProper: boolean
    isReal: boolean
    isInternal: boolean
    isLimited: boolean
    isRemastered: boolean
    
    // Warnings (reasons to block)
    isBlockedSource: boolean (CAM, TS, etc.)
    isBlockedAudio: boolean (MD, LD, etc.)
    isBlockedSubs: boolean (hardcoded)
    isUpscaled: boolean
    isSample: boolean
    isBlockedGroup: boolean
    isNuked: boolean
    isWorkprint: boolean
    isFastsub: boolean
    
    // Raw
    rawTitle: string
    size: number (bytes)
    seeders: number
    indexer: string
}
```

---

## Scoring Guidelines

Quality should be scored internally (user never sees numbers) to rank releases:

### Base Score by Resolution
- 2160p: 100
- 1080p: 75
- 720p: 50
- 480p: 25

### Source Modifier
- remux: +50
- bluray: +40
- webdl: +30
- webrip: +20
- hdtv: +10
- satellite: +5

### HDR Modifier
- dv: +20
- hdr10plus: +15
- hdr10: +10
- hlg: +5

### Audio Modifier
- atmos: +20
- truehd/dtshd/dtsx: +15
- flac: +10
- ddplus: +5

### Other Modifiers
- 10bit: +5
- REPACK/PROPER: +5
- Higher version (v2, v3): +3 per version
- Dual-audio (anime): +10
- Soft subs: +5
- More seeders: +1 per 10 seeders (cap at +10)
- Trusted group: +5

### Negative Modifiers
- Fullscreen/cropped: -20
- Dubbed (unless preferred): -10
- Fansub: -5
- Old version when newer exists: -10 per version behind

### Instant Rejection (score = -1000)
- Blocked source (CAM, TS, etc.)
- Blocked audio (MD, LD, etc.)
- Hardcoded subs
- Upscaled
- Sample
- Blocked group
- Below minimum seeders
- Missing required language
- NUKED
- WORKPRINT
- FASTSUB (for anime)

---

## Example Parses

**Input:** `Dune.Part.Two.2024.2160p.AMZN.WEB-DL.DDP5.1.Atmos.DV.H.265-FLUX`
```
title: "Dune Part Two"
year: 2024
resolution: "2160p"
source: "webdl"
codec: "hevc"
hdr: "dv"
audioFormat: "atmos"
audioChannels: "5.1"
streamingService: "AMZN"
releaseGroup: "FLUX"
```

**Input:** `Stranger.Things.S05E08.1080p.NF.WEB-DL.DDP5.1.Atmos.H.264-FLUX`
```
title: "Stranger Things"
season: 5
episodes: [8]
resolution: "1080p"
source: "webdl"
codec: "avc"
audioFormat: "atmos"
audioChannels: "5.1"
streamingService: "NF"
releaseGroup: "FLUX"
```

**Input:** `IT.Welcome.to.Derry.S01.1080p.HMAX.WEBRip.AAC5.1.10bits.x265-Rapta`
```
title: "IT Welcome to Derry"
season: 1
isSeasonPack: true
resolution: "1080p"
source: "webrip"
codec: "hevc"
bitDepth: 10
audioFormat: "aac"
audioChannels: "5.1"
streamingService: "HMAX"
releaseGroup: "Rapta"
```

**Input:** `Avatar.Fire.and.Ash.2025.1080p.TS.EN-RGB`
```
title: "Avatar Fire and Ash"
year: 2025
resolution: "1080p"
source: "ts"
isBlockedSource: true  ← BLOCK THIS
releaseGroup: "RGB"
```

**Input:** `[SubsPlease] Jujutsu Kaisen - 45 (1080p) [ABCD1234].mkv`
```
title: "Jujutsu Kaisen"
episodes: [45]
isAbsoluteEpisode: true
resolution: "1080p"
releaseGroup: "SubsPlease"
isAnime: true
```

**Input:** `[Judas] Chainsaw Man - S01E12v2 (BD 1080p HEVC AAC) [Dual-Audio].mkv`
```
title: "Chainsaw Man"
season: 1
episodes: [12]
version: 2
resolution: "1080p"
source: "bluray"
codec: "hevc"
audioFormat: "aac"
hasDualAudio: true
releaseGroup: "Judas"
isAnime: true
```

**Input:** `[Erai-raws] Demon Slayer - 01-26 [1080p][Batch][Multiple Subtitle].mkv`
```
title: "Demon Slayer"
episodes: [1,2,3...26]
isBatch: true
resolution: "1080p"
hasMultipleSubs: true
releaseGroup: "Erai-raws"
isAnime: true
```

**Input:** `Blade.Runner.1982.Final.Cut.2160p.UHD.BluRay.REMUX.DV.HDR.TrueHD.Atmos.7.1-FraMeSToR`
```
title: "Blade Runner"
year: 1982
edition: "final cut"
resolution: "2160p"
source: "remux"
hdr: "dv"
audioFormat: "atmos"
audioChannels: "7.1"
releaseGroup: "FraMeSToR"
```

**Input:** `The.Matrix.1999.REMASTERED.1080p.BluRay.x265.HEVC.10bit.AAC.5.1-Tigole`
```
title: "The Matrix"
year: 1999
edition: "remastered"
resolution: "1080p"
source: "bluray"
codec: "hevc"
bitDepth: 10
audioFormat: "aac"
audioChannels: "5.1"
releaseGroup: "Tigole"
isRemastered: true
```
