# QUALITY.md - Quality Profile System

## Overview

Quality profiles determine which releases to download and when to upgrade. Based on Sonarr/Radarr's proven system.

---

## Quality Definitions

### Video Qualities (ordered best to worst)

| Quality | Resolution | Source | Typical Size (2hr movie) |
|---------|------------|--------|--------------------------|
| Remux-2160p | 4K | Blu-ray Remux | 50-80 GB |
| Bluray-2160p | 4K | Blu-ray Encode | 15-40 GB |
| WEBDL-2160p | 4K | Streaming | 10-25 GB |
| WEBRip-2160p | 4K | Screen capture | 10-25 GB |
| HDTV-2160p | 4K | TV Broadcast | 8-20 GB |
| Remux-1080p | 1080p | Blu-ray Remux | 20-40 GB |
| Bluray-1080p | 1080p | Blu-ray Encode | 8-15 GB |
| WEBDL-1080p | 1080p | Streaming | 3-8 GB |
| WEBRip-1080p | 1080p | Screen capture | 3-8 GB |
| HDTV-1080p | 1080p | TV Broadcast | 2-6 GB |
| Bluray-720p | 720p | Blu-ray Encode | 4-8 GB |
| WEBDL-720p | 720p | Streaming | 1.5-4 GB |
| WEBRip-720p | 720p | Screen capture | 1.5-4 GB |
| HDTV-720p | 720p | TV Broadcast | 1-3 GB |
| DVD | 480p | DVD | 1-4 GB |
| SDTV | 480p | TV Broadcast | 0.5-1.5 GB |

### Audio Qualities

| Codec | Quality | Notes |
|-------|---------|-------|
| TrueHD Atmos | Highest | Dolby Atmos |
| DTS-HD MA | Highest | Lossless |
| TrueHD | High | Lossless |
| FLAC | High | Lossless |
| DTS-HD HRA | High | Near-lossless |
| DTS | Medium | Lossy |
| DD+ (E-AC-3) | Medium | Streaming standard |
| DD (AC-3) | Medium | DVD standard |
| AAC | Low | Compressed |
| MP3 | Lowest | Compressed |

---

## Quality Profile Structure

```json
{
  "id": 1,
  "name": "4K Enthusiast",
  "upgradeAllowed": true,
  "upgradeUntilQuality": "Remux-2160p",
  "minFormatScore": 20000,
  "upgradeUntilScore": 400000,
  "minScoreIncrement": 1,
  "qualities": [
    { "quality": "Remux-2160p", "enabled": true },
    { "quality": "Bluray-2160p", "enabled": true },
    { "quality": "WEBDL-2160p", "enabled": true },
    { "quality": "Remux-1080p", "enabled": true },
    { "quality": "Bluray-1080p", "enabled": true },
    { "quality": "WEBDL-1080p", "enabled": true },
    { "quality": "WEBDL-720p", "enabled": false },
    { "quality": "SDTV", "enabled": false }
  ],
  "qualityGroups": [
    {
      "name": "WEB 2160p",
      "qualities": ["WEBDL-2160p", "WEBRip-2160p"]
    }
  ],
  "customFormats": [
    { "name": "2160p Remux", "score": 160000 },
    { "name": "2160p WEB-DL", "score": 140000 },
    { "name": "1080p Remux", "score": 120000 },
    { "name": "1080p WEB-DL", "score": 100000 },
    { "name": "DTS-HD MA", "score": 5000 },
    { "name": "TrueHD", "score": 5000 },
    { "name": "Atmos", "score": 3000 },
    { "name": "Bad Release Group", "score": -100000 }
  ]
}
```

---

## Custom Formats

Custom formats apply score modifiers based on release attributes.

### Format Definition

```json
{
  "name": "2160p Remux",
  "score": 160000,
  "conditions": [
    {
      "type": "resolution",
      "value": "2160p",
      "required": true
    },
    {
      "type": "source",
      "value": "remux",
      "required": true
    }
  ]
}
```

### Condition Types

| Type | Description | Values |
|------|-------------|--------|
| `resolution` | Video resolution | 2160p, 1080p, 720p, 480p |
| `source` | Release source | remux, bluray, webdl, webrip, hdtv, dvd |
| `codec` | Video codec | x265, x264, xvid, h265, h264, av1 |
| `audioCodec` | Audio codec | truehd, dtshd, dts, dd, aac |
| `audioFeature` | Audio feature | atmos, 7.1, 5.1 |
| `releaseGroup` | Release group | SPARKS, FLUX, etc. |
| `keyword` | Contains keyword | proper, repack, hdr, dv |
| `notKeyword` | Does not contain | cam, ts, sample |
| `size` | File size range | min/max in bytes |
| `language` | Audio language | english, japanese, etc. |
| `subtitle` | Has subtitles | english, any, none |

### Condition Matching

- `required: true` - Must match or release rejected
- `required: false` - Adds/subtracts score if matched

---

## Default Profiles

### 4K Enthusiast

For users with 4K displays and storage space.

- Upgrades allowed: Yes
- Upgrade until: Remux-2160p
- Min score: 20000
- Enabled qualities: All 2160p and 1080p
- Prefers: Remux > Bluray > WEB-DL

### 1080p Standard

For most users with 1080p displays.

- Upgrades allowed: Yes
- Upgrade until: Remux-1080p
- Min score: 10000
- Enabled qualities: All 1080p and 720p
- Prefers: Bluray > WEB-DL

### Bandwidth Saver

For users with limited storage/bandwidth.

- Upgrades allowed: No
- Enabled qualities: WEBDL-1080p, WEBDL-720p
- Prefers: Smaller files

### Any Quality

For users who just want content fast.

- Upgrades allowed: No
- Enabled qualities: All
- Grabs first available

---

## Scoring Algorithm

```
total_score = base_quality_score + sum(custom_format_scores)

if total_score < min_format_score:
    reject release

if has_existing_file:
    if not upgrade_allowed:
        reject release
    if existing_score >= upgrade_until_score:
        reject release
    if total_score - existing_score < min_score_increment:
        reject release
```

### Base Quality Scores

Higher quality = higher base score

| Quality | Base Score |
|---------|------------|
| Remux-2160p | 10000 |
| Bluray-2160p | 9000 |
| WEBDL-2160p | 8000 |
| Remux-1080p | 7000 |
| Bluray-1080p | 6000 |
| WEBDL-1080p | 5000 |
| WEBDL-720p | 3000 |
| SDTV | 1000 |

---

## Release Parsing

Parse release names to extract attributes:

```
Inception.2010.2160p.UHD.BluRay.REMUX.HDR.DV.TrueHD.Atmos.7.1-FGT

Title: Inception
Year: 2010
Resolution: 2160p
Source: BluRay REMUX
HDR: Yes (HDR + DV)
Audio: TrueHD Atmos 7.1
Group: FGT
```

### Parsing Patterns

```regex
Resolution: (2160p|1080p|720p|480p|4K|UHD)
Source: (REMUX|BluRay|Blu-Ray|BDRip|WEB-DL|WEBDL|WEBRip|HDTV|DVDRip|DVD)
Codec: (x265|x264|HEVC|H\.265|H\.264|AV1|XviD)
Audio: (TrueHD|DTS-HD\.?MA|DTS-HD|DTS|DD\+|DDP|DD|AC3|AAC|FLAC)
Features: (Atmos|7\.1|5\.1|HDR|HDR10|HDR10\+|DV|DoVi|Dolby\.Vision)
Group: -([A-Za-z0-9]+)$
```

---

## Anime-Specific

Anime profiles include additional conditions:

### Language Preferences

```json
{
  "preferredAudio": "japanese",
  "preferredSubtitles": "english",
  "acceptDualAudio": true,
  "acceptDubOnly": false
}
```

### Release Group Preferences

```json
{
  "preferredGroups": ["SubsPlease", "Erai-raws", "ASW"],
  "avoidGroups": ["HorribleSubs-Reupload"]
}
```

### Custom Formats for Anime

- Dual Audio: +5000
- Japanese Audio: +3000
- English Subs: +2000
- Preferred Group: +1000
- Dub Only: -5000 (if user prefers sub)

---

## UI Components

### Profile Editor

- Name field
- Upgrade toggle
- Upgrade until dropdown
- Score thresholds
- Quality checklist (drag to reorder)
- Quality grouping interface
- Custom format list with scores

### Release Scoring Display

When viewing search results:

```
Inception.2010.2160p.UHD.BluRay.REMUX

Quality: Remux-2160p     +10000
Custom Formats:
  2160p Remux            +160000
  TrueHD Atmos           +8000
  ─────────────────────────────
  Total Score:           178000 ✓
```

---

## API Endpoints

```
GET  /api/settings/quality-profiles
POST /api/settings/quality-profiles
GET  /api/settings/quality-profiles/:id
PUT  /api/settings/quality-profiles/:id
DEL  /api/settings/quality-profiles/:id

GET  /api/settings/custom-formats
POST /api/settings/custom-formats
...

POST /api/releases/parse
  Body: { "name": "Release.Name.2024" }
  Returns: Parsed attributes and score
```
