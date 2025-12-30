# QUALITY.md - Quality Profiles System

---

## Overview

Quality profiles determine which releases to download and when to upgrade. Based on Sonarr/Radarr's proven system.

---

## Quality Definitions

### Video Qualities (Ranked)

| Rank | Quality | Resolution | Source |
|------|---------|------------|--------|
| 1 | WORKPRINT | Any | Pre-release |
| 2 | CAM | 480p | Theater recording |
| 3 | TELESYNC | 480p | Theater audio sync |
| 4 | TELECINE | 480p | Film reel |
| 5 | REGIONAL | 480p | Regional release |
| 6 | DVDSCR | 480p | Screener |
| 7 | SDTV | 480p | TV broadcast |
| 8 | DVD | 480p | DVD rip |
| 9 | WEBDL-480p | 480p | Web download |
| 10 | WEBRip-480p | 480p | Web recording |
| 11 | HDTV-720p | 720p | TV broadcast |
| 12 | WEBDL-720p | 720p | Web download |
| 13 | WEBRip-720p | 720p | Web recording |
| 14 | Bluray-720p | 720p | Blu-ray rip |
| 15 | HDTV-1080p | 1080p | TV broadcast |
| 16 | WEBDL-1080p | 1080p | Web download |
| 17 | WEBRip-1080p | 1080p | Web recording |
| 18 | Bluray-1080p | 1080p | Blu-ray rip |
| 19 | Remux-1080p | 1080p | Blu-ray remux |
| 20 | HDTV-2160p | 2160p | TV broadcast |
| 21 | WEBDL-2160p | 2160p | Web download |
| 22 | WEBRip-2160p | 2160p | Web recording |
| 23 | Bluray-2160p | 2160p | Blu-ray rip |
| 24 | Remux-2160p | 2160p | Blu-ray remux |

---

## Quality Profile Structure

```json
{
  "id": 1,
  "name": "HD-1080p",
  "upgrade_allowed": true,
  "cutoff": "Bluray-1080p",
  "min_format_score": 0,
  "cutoff_format_score": 10000,
  "upgrade_score_increment": 100,
  "items": [
    {
      "quality": "Bluray-1080p",
      "enabled": true
    },
    {
      "quality": "WEBDL-1080p",
      "enabled": true
    },
    {
      "quality": "Remux-1080p",
      "enabled": false
    }
  ]
}
```

### Fields

| Field | Description |
|-------|-------------|
| `name` | Profile display name |
| `upgrade_allowed` | Allow upgrading existing files |
| `cutoff` | Stop upgrading at this quality |
| `min_format_score` | Minimum custom format score to accept |
| `cutoff_format_score` | Stop upgrading at this score |
| `upgrade_score_increment` | Minimum score improvement to upgrade |
| `items` | Quality tiers with enabled/disabled |

---

## Quality Groups

Group multiple qualities to treat them as equal:

```json
{
  "name": "WEB 1080p",
  "qualities": ["WEBDL-1080p", "WEBRip-1080p"]
}
```

When grouped:
- Either quality satisfies the requirement
- No upgrading within the group
- Order within group doesn't matter

---

## Custom Formats

Custom formats score releases based on conditions. Higher score = more preferred.

### Structure

```json
{
  "id": 1,
  "name": "DV HDR10",
  "score": 1500,
  "conditions": [
    {
      "type": "release_title",
      "pattern": "\\bDV\\b|\\bDoVi\\b|\\bDolby.?Vision\\b",
      "required": true
    },
    {
      "type": "release_title",
      "pattern": "\\bHDR10\\b",
      "required": true
    }
  ]
}
```

### Condition Types

| Type | Description | Example Pattern |
|------|-------------|-----------------|
| `release_title` | Regex on release name | `\\b(AMZN|NF|DSNP)\\b` |
| `source` | Source type match | `WEB` |
| `resolution` | Resolution match | `2160p` |
| `language` | Audio language | `en` |

### Condition Logic
- All `required: true` must match
- Any `required: false` can match
- Negate with `negate: true`

---

## Preset Profiles

### Any (Accept Anything)
```
All qualities enabled
Upgrade: No
```

### SD
```
Enabled: SDTV, DVD, WEBDL-480p, WEBRip-480p
Cutoff: DVD
Upgrade: Yes
```

### HD-720p
```
Enabled: HDTV-720p, WEBDL-720p, WEBRip-720p, Bluray-720p
Cutoff: Bluray-720p
Upgrade: Yes
```

### HD-1080p
```
Enabled: HDTV-1080p, WEBDL-1080p, WEBRip-1080p, Bluray-1080p
Cutoff: Bluray-1080p
Upgrade: Yes
```

### Ultra-HD
```
Enabled: WEBDL-2160p, WEBRip-2160p, Bluray-2160p, Remux-2160p
Cutoff: Bluray-2160p
Upgrade: Yes
Custom Formats: +1500 for DV, +1000 for HDR10+, +500 for HDR10
```

---

## Common Custom Formats

### HDR

**Dolby Vision:**
```
Pattern: \b(DV|DoVi|Dolby.?Vision)\b
Score: 1500
```

**HDR10+:**
```
Pattern: \b(HDR10\+|HDR10Plus)\b
Score: 1000
```

**HDR10:**
```
Pattern: \bHDR10?\b
Negate: HDR10\+
Score: 500
```

### Audio

**TrueHD Atmos:**
```
Pattern: \b(TrueHD|True.?HD).?Atmos\b
Score: 2000
```

**DTS-X:**
```
Pattern: \bDTS.?X\b
Score: 1500
```

**Atmos:**
```
Pattern: \bAtmos\b
Score: 1000
```

### Streaming Services

**Preferred Services:**
```
Pattern: \b(AMZN|ATVP|DSNP|NF|HMAX)\b
Score: 100
```

### Release Groups (Anime)

**Preferred:**
```
Pattern: \b(SubsPlease|Erai-raws|EMBER)\b
Score: 500
```

**Avoid:**
```
Pattern: \b(YTS|RARBG)\b
Score: -1000
```

---

## Scoring Example

Release: `Movie.Name.2024.2160p.AMZN.WEB-DL.DDP5.1.Atmos.DV.HDR10.H.265`

| Custom Format | Score |
|---------------|-------|
| Dolby Vision | +1500 |
| HDR10 | +500 |
| Atmos | +1000 |
| AMZN | +100 |
| **Total** | **3100** |

---

## Upgrade Logic

1. New release found for existing media
2. Compare quality tier
3. If new quality > cutoff: Skip
4. If new quality > current: Check score
5. Calculate custom format score
6. If score >= current + upgrade_increment: Upgrade
7. If score < min_format_score: Skip

---

## Anime-Specific Formats

**Dual Audio:**
```
Pattern: \b(Dual.?Audio|JA\+EN)\b
Score: 500
```

**Multi-Sub:**
```
Pattern: \b(Multi.?Sub)\b
Score: 200
```

**Uncensored:**
```
Pattern: \b(Uncensored|Uncut)\b
Score: 300
```

---

## Blocked Extensions

Default blocked (never download):

```
.exe, .bat, .cmd, .com, .cpl, .dll, .js, .jse, .msi, .msp,
.pif, .scr, .vbs, .vbe, .wsf, .wsh, .hta, .reg, .inf,
.ps1, .ps2, .psm1, .psd1, .sh, .apk, .app, .ipa, .jar,
.lnk, .tmp, .html, .php, .torrent
```

Sample files blocked:
```
*sample.mkv, *sample.avi, *sample.mp4
```

---

## Size Limits

Configurable per quality:

| Quality | Min | Max |
|---------|-----|-----|
| SDTV | 100 MB | 2 GB |
| HDTV-720p | 500 MB | 5 GB |
| WEBDL-1080p | 1 GB | 15 GB |
| Bluray-1080p | 2 GB | 30 GB |
| WEBDL-2160p | 5 GB | 50 GB |
| Remux-2160p | 30 GB | 100 GB |

---

## Profile Assignment

- Movies: One profile per movie or default
- TV Shows: One profile per show
- Anime: Separate profile with anime-specific formats
