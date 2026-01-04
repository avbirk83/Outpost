# Outpost Phase A-D Implementation Spec

## Overview

These four phases work together to create the complete acquisition flow:

```
User wants movie → Quality Target determines what to grab → 
Smart Logic filters garbage → Download client grabs it → 
Auto-Import organizes & renames → Library updated
```

---

# Phase A: Smart Quality System

## A.1 Database Schema

```sql
-- Quality presets (built-in + custom)
CREATE TABLE quality_presets (
    id INTEGER PRIMARY KEY,
    name TEXT NOT NULL,                    -- "Best", "High", "Balanced", "Storage Saver", "Custom"
    is_default BOOLEAN DEFAULT FALSE,      -- User's default preset
    is_built_in BOOLEAN DEFAULT FALSE,     -- System preset, can't delete
    
    -- Video
    resolution TEXT NOT NULL,              -- "4k", "1080p", "720p", "480p"
    source TEXT NOT NULL,                  -- "remux", "bluray", "web", "any"
    hdr_formats TEXT,                      -- JSON: ["dv", "hdr10+", "hdr10"] or null for SDR OK
    codec TEXT DEFAULT 'any',              -- "any", "hevc", "av1"
    
    -- Audio
    audio_formats TEXT,                    -- JSON: ["atmos", "truehd", "dtshd"] or null for any
    
    -- Edition
    preferred_edition TEXT DEFAULT 'any',  -- "any", "theatrical", "directors", "extended", "unrated"
    
    -- Downloads
    min_seeders INTEGER DEFAULT 3,
    
    -- TV
    prefer_season_packs BOOLEAN DEFAULT TRUE,
    
    -- Behavior
    auto_upgrade BOOLEAN DEFAULT TRUE,
    
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Insert built-in presets
INSERT INTO quality_presets (name, is_built_in, resolution, source, hdr_formats, audio_formats) VALUES
('Best', TRUE, '4k', 'remux', '["dv", "hdr10+", "hdr10"]', '["atmos", "truehd", "dtshd"]'),
('High', TRUE, '4k', 'web', '["dv", "hdr10+", "hdr10"]', NULL),
('Balanced', TRUE, '1080p', 'web', NULL, NULL),
('Storage Saver', TRUE, '1080p', 'web', NULL, NULL);

-- Per-item quality override (optional)
CREATE TABLE media_quality_override (
    id INTEGER PRIMARY KEY,
    media_id INTEGER NOT NULL,
    media_type TEXT NOT NULL,              -- "movie", "show"
    preset_id INTEGER REFERENCES quality_presets(id),
    monitored BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Track current vs target quality
CREATE TABLE media_quality_status (
    id INTEGER PRIMARY KEY,
    media_id INTEGER NOT NULL,
    media_type TEXT NOT NULL,              -- "movie", "episode"
    
    -- Current file info
    current_resolution TEXT,
    current_source TEXT,
    current_hdr TEXT,
    current_audio TEXT,
    current_edition TEXT,
    
    -- Status
    target_met BOOLEAN DEFAULT FALSE,
    upgrade_available BOOLEAN DEFAULT FALSE,
    last_search TIMESTAMP,
    
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

## A.2 Release Parsing

Parse release names to extract quality info:

```go
type ParsedRelease struct {
    Title           string
    Year            int
    Season          int      // 0 if movie
    Episode         int      // 0 if movie or season pack
    IsSeasonPack    bool
    IsDailyShow     bool
    AirDate         string   // For daily shows: "2024-01-15"
    
    // Quality
    Resolution      string   // "2160p", "1080p", "720p", "480p"
    Source          string   // "remux", "bluray", "web", "hdtv", "cam"
    HDR             string   // "dv", "hdr10+", "hdr10", "hlg", ""
    Codec           string   // "hevc", "x265", "x264", "av1", "xvid"
    
    // Audio
    AudioFormat     string   // "atmos", "truehd", "dtshd", "dd+", "dd", "aac"
    AudioChannels   string   // "7.1", "5.1", "2.0"
    
    // Other
    Edition         string   // "directors", "extended", "theatrical", "unrated", ""
    ReleaseGroup    string
    IsProper        bool
    IsRepack        bool
    IsReal          bool     // "REAL" tag
    
    // Warnings
    IsHardcodedSubs bool
    IsUpscaled      bool     // Fake 4K
    IsCompressedAudio bool   // MD, LD, LiNE
    IsSample        bool
    Is3D            bool
    
    // Raw
    RawTitle        string
    Size            int64
    Seeders         int
    Indexer         string
}
```

### Resolution Patterns
```
4K/2160p:     /2160p|4k|uhd/i
1080p:        /1080p|1080i/i
720p:         /720p/i
480p:         /480p|dvdrip|sdtv/i
```

### Source Patterns
```
Remux:        /remux/i
Bluray:       /blu-?ray|bdrip|brrip/i (but not remux)
WEB:          /web-?dl|webrip|amzn|nf|dsnp|hulu|atvp|hmax|pcok/i
HDTV:         /hdtv/i
CAM:          /cam|hdcam|ts|telesync|tc|telecine|scr|screener|dvdscr/i
```

### HDR Patterns
```
Dolby Vision: /dv|dolby.?vision|dovi/i
HDR10+:       /hdr10\+|hdr10plus/i
HDR10:        /hdr10|hdr/i (but not hdr10+)
HLG:          /hlg/i
```

### Audio Patterns
```
Atmos:        /atmos/i
TrueHD:       /truehd|true-hd/i
DTS-HD MA:    /dts-?hd|dts-?ma|dts-?hd.?ma/i
DTS-X:        /dts-?x/i
DD+/EAC3:     /dd\+|ddp|eac3|e-ac-3/i
DTS:          /dts(?!-?hd|-?x|-?ma)/i
DD/AC3:       /dd(?!\+|p)|ac3|ac-3/i
AAC:          /aac/i
Compressed:   /md|mic.?dub|line|ld/i (BLOCK)
```

### Bad Patterns (Always Block)
```
Hardcoded:    /hc|hardcoded|hard.?coded|korsub/i
Sample:       /sample/i (in filename)
Password:     /password/i
RAR:          /\.rar$/i
Upscaled:     /upscale|upscaled/i
```

### Edition Patterns
```
Directors:    /directors?.cut|dc/i
Extended:     /extended/i
Theatrical:   /theatrical/i
Unrated:      /unrated/i
```

## A.3 Trusted & Blocked Groups

```go
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

var BlockedGroups = []string{
    "YIFY", "YTS", "RARBG", "TGx", "MeGusta", 
    "STUTTERSHIT", "aXXo", "SPARKS-encoded",
}
```

## A.4 Quality Scoring (Internal)

Used internally to rank releases — user never sees this:

```go
func ScoreRelease(release ParsedRelease, target QualityPreset) int {
    score := 0
    
    // Base score by resolution
    switch release.Resolution {
    case "2160p": score += 100
    case "1080p": score += 75
    case "720p":  score += 50
    case "480p":  score += 25
    }
    
    // Source modifier
    switch release.Source {
    case "remux":  score += 50
    case "bluray": score += 40
    case "web":    score += 30
    case "hdtv":   score += 10
    }
    
    // HDR modifier
    switch release.HDR {
    case "dv":     score += 20
    case "hdr10+": score += 15
    case "hdr10":  score += 10
    }
    
    // Audio modifier
    switch release.AudioFormat {
    case "atmos":  score += 20
    case "truehd": score += 15
    case "dtshd":  score += 15
    case "dtsx":   score += 15
    case "dd+":    score += 5
    }
    
    // Group modifier
    if isTrusted(release.ReleaseGroup) {
        score += 10
    }
    
    // Proper/Repack bonus
    if release.IsProper || release.IsRepack {
        score += 5
    }
    
    // Seeder bonus (prefer more seeders)
    score += min(release.Seeders / 10, 10)
    
    // Penalties (these should already be filtered, but just in case)
    if release.IsUpscaled { score -= 50 }
    if release.IsCompressedAudio { score -= 100 }
    if release.IsHardcodedSubs { score -= 100 }
    
    return score
}
```

## A.5 Quality Matching Logic

```go
func MatchesTarget(release ParsedRelease, target QualityPreset, settings UserSettings) (bool, int) {
    // Hard blocks — never grab
    if release.Source == "cam" { return false, 0 }
    if release.IsHardcodedSubs { return false, 0 }
    if release.IsCompressedAudio { return false, 0 }
    if release.IsSample { return false, 0 }
    if isBlockedGroup(release.ReleaseGroup) { return false, 0 }
    if release.Seeders < target.MinSeeders { return false, 0 }
    
    // Language check
    if !hasLanguage(release, settings.AudioLanguage) {
        return false, 0
    }
    
    // Calculate match score
    score := ScoreRelease(release, target)
    
    // Check if meets minimum quality (fallback floor)
    if !meetsMinimumQuality(release) {
        return false, 0
    }
    
    return true, score
}

func meetsMinimumQuality(release ParsedRelease) bool {
    // Minimum acceptable: 720p from non-garbage source
    if release.Resolution == "480p" {
        return false // Could be configurable
    }
    if release.Source == "cam" || release.Source == "ts" {
        return false
    }
    return true
}
```

## A.6 Best Release Selection

```go
func SelectBestRelease(releases []ParsedRelease, target QualityPreset, settings UserSettings) *ParsedRelease {
    var candidates []struct {
        Release ParsedRelease
        Score   int
        MatchesTarget bool
    }
    
    for _, r := range releases {
        ok, score := MatchesTarget(r, target, settings)
        if ok {
            // Check if this release actually matches target specs
            matchesTarget := checkTargetMatch(r, target)
            candidates = append(candidates, struct{
                Release ParsedRelease
                Score   int
                MatchesTarget bool
            }{r, score, matchesTarget})
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
    
    return &candidates[0].Release
}

func checkTargetMatch(release ParsedRelease, target QualityPreset) bool {
    // Resolution match
    if !resolutionMatches(release.Resolution, target.Resolution) {
        return false
    }
    
    // Source match
    if target.Source != "any" && release.Source != target.Source {
        return false
    }
    
    // HDR match (if required)
    if target.HDRFormats != nil && len(target.HDRFormats) > 0 {
        if !contains(target.HDRFormats, release.HDR) {
            return false
        }
    }
    
    // Audio match (if required)
    if target.AudioFormats != nil && len(target.AudioFormats) > 0 {
        if !contains(target.AudioFormats, release.AudioFormat) {
            return false
        }
    }
    
    return true
}
```

## A.7 API Endpoints

```
GET  /api/quality/presets              — List all presets
POST /api/quality/presets              — Create custom preset
GET  /api/quality/presets/:id          — Get preset details
PUT  /api/quality/presets/:id          — Update preset
DEL  /api/quality/presets/:id          — Delete preset (not built-in)
POST /api/quality/presets/:id/default  — Set as default

GET  /api/movies/:id/quality           — Get movie quality status
PUT  /api/movies/:id/quality           — Set quality override for movie
POST /api/movies/:id/search            — Manual search for movie

GET  /api/shows/:id/quality            — Get show quality status
PUT  /api/shows/:id/quality            — Set quality override for show

GET  /api/search/releases?q=...        — Search indexers, return parsed releases
POST /api/grab                         — Send release to download client
```

## A.8 UI Components

### Quality Preset Selector
```svelte
<script>
  export let value = 'best';
  export let presets = [];
</script>

<div class="preset-selector">
  {#each presets as preset}
    <button 
      class="preset-btn" 
      class:active={value === preset.id}
      on:click={() => value = preset.id}
    >
      <span class="preset-name">{preset.name}</span>
      <span class="preset-desc">{preset.resolution} {preset.source}</span>
    </button>
  {/each}
</div>
```

### Search Results
```svelte
<script>
  export let results = [];
  export let filtered = [];
  export let showFiltered = false;
</script>

<div class="search-results">
  {#if results.length > 0}
    <div class="best-match">
      <span class="label">★ Best Match</span>
      <ReleaseCard release={results[0]} />
    </div>
    
    {#each results.slice(1) as release}
      <ReleaseCard {release} />
    {/each}
  {/if}
  
  {#if filtered.length > 0}
    <button class="show-filtered" on:click={() => showFiltered = !showFiltered}>
      {filtered.length} releases filtered {showFiltered ? '▲' : '▼'}
    </button>
    
    {#if showFiltered}
      {#each filtered as release}
        <ReleaseCard {release} dimmed />
      {/each}
    {/if}
  {/if}
</div>
```

---

# Phase B: Language & Subtitle Preferences

## B.1 Database Schema

```sql
-- Add to settings table or user preferences
-- language_audio TEXT DEFAULT 'en'
-- language_audio_include_original BOOLEAN DEFAULT TRUE
-- language_subtitle TEXT DEFAULT 'en'
-- subtitle_mode TEXT DEFAULT 'prefer' -- 'off', 'prefer', 'always'
```

## B.2 Onboarding Flow

Add to initial setup wizard (step 3):

```svelte
<script>
  import { languages } from '$lib/data/languages';
  
  let audioLanguage = 'en';
  let includeOriginal = true;
  let subtitleLanguage = 'en';
  let subtitleMode = 'prefer';
</script>

<div class="onboarding-step">
  <h2>Language Preferences</h2>
  
  <div class="section">
    <label>Audio Language</label>
    <p class="hint">What language do you want to hear?</p>
    <select bind:value={audioLanguage}>
      {#each languages as lang}
        <option value={lang.code}>{lang.name}</option>
      {/each}
    </select>
    
    <label class="checkbox">
      <input type="checkbox" bind:checked={includeOriginal} />
      Include original language for foreign films
    </label>
  </div>
  
  <div class="section">
    <label>Subtitles</label>
    <select bind:value={subtitleLanguage}>
      {#each languages as lang}
        <option value={lang.code}>{lang.name}</option>
      {/each}
    </select>
    
    <div class="radio-group">
      <label>
        <input type="radio" bind:group={subtitleMode} value="off" />
        Off — I don't use subtitles
      </label>
      <label>
        <input type="radio" bind:group={subtitleMode} value="prefer" />
        Prefer — Include if available
      </label>
      <label>
        <input type="radio" bind:group={subtitleMode} value="always" />
        Always — Auto-fetch if missing
      </label>
    </div>
  </div>
</div>
```

## B.3 Language Data

```typescript
// $lib/data/languages.ts
export const languages = [
  { code: 'en', name: 'English' },
  { code: 'es', name: 'Spanish' },
  { code: 'fr', name: 'French' },
  { code: 'de', name: 'German' },
  { code: 'it', name: 'Italian' },
  { code: 'pt', name: 'Portuguese' },
  { code: 'ru', name: 'Russian' },
  { code: 'ja', name: 'Japanese' },
  { code: 'ko', name: 'Korean' },
  { code: 'zh', name: 'Chinese' },
  { code: 'hi', name: 'Hindi' },
  { code: 'ar', name: 'Arabic' },
  // ... more
];
```

## B.4 OpenSubtitles Integration (for "Always" mode)

```go
// When subtitle_mode = "always" and file missing subs

type OpenSubtitlesClient struct {
    APIKey   string
    BaseURL  string // https://api.opensubtitles.com/api/v1
}

func (c *OpenSubtitlesClient) SearchSubtitles(hash string, title string, year int, lang string) ([]Subtitle, error) {
    // Try hash match first (most accurate)
    results, err := c.searchByHash(hash, lang)
    if err == nil && len(results) > 0 {
        return results, nil
    }
    
    // Fall back to title search
    return c.searchByTitle(title, year, lang)
}

func (c *OpenSubtitlesClient) Download(subtitleID string, destPath string) error {
    // Download and save to destPath
}
```

**API Key:** Free tier available, rate limited. User provides their own key in settings.

## B.5 Settings Page

```
┌─────────────────────────────────────────────────────────┐
│  Settings → Languages                                   │
│                                                         │
│  Audio                                                  │
│  Primary language:  [English ▾]                        │
│  ☑ Include original language for foreign films         │
│                                                         │
│  Subtitles                                              │
│  Language:  [English ▾]                                │
│  Mode:      [Prefer - include if available ▾]          │
│                                                         │
│  OpenSubtitles (for auto-fetch)                        │
│  API Key:  [••••••••••••]  [Test]                      │
│  Status:   ✓ Connected                                 │
│                                                         │
└─────────────────────────────────────────────────────────┘
```

---

# Phase C: Storage Management

## C.1 Database Schema

```sql
-- Add to settings
-- storage_threshold_gb INTEGER DEFAULT 100
-- storage_pause_enabled BOOLEAN DEFAULT TRUE
-- upgrade_delete_old BOOLEAN DEFAULT TRUE
```

## C.2 Disk Space Monitoring

```go
type StorageManager struct {
    ThresholdGB int64
    Enabled     bool
    Libraries   []Library
}

func (s *StorageManager) CheckDiskSpace() []StorageAlert {
    var alerts []StorageAlert
    
    for _, lib := range s.Libraries {
        stat, err := disk.Usage(lib.Path)
        if err != nil {
            continue
        }
        
        freeGB := stat.Free / (1024 * 1024 * 1024)
        
        if freeGB < s.ThresholdGB {
            alerts = append(alerts, StorageAlert{
                LibraryID:   lib.ID,
                LibraryPath: lib.Path,
                FreeGB:      freeGB,
                ThresholdGB: s.ThresholdGB,
            })
        }
    }
    
    return alerts
}

func (s *StorageManager) ShouldPauseDownloads() bool {
    if !s.Enabled {
        return false
    }
    
    alerts := s.CheckDiskSpace()
    return len(alerts) > 0
}
```

## C.3 Integration with Download Queue

```go
func (d *DownloadManager) ProcessQueue() {
    // Check storage before grabbing
    if storageManager.ShouldPauseDownloads() {
        d.Status = "paused_storage"
        notifyUser("Downloads paused: Low disk space")
        return
    }
    
    // Continue with normal queue processing
    d.processNextItem()
}
```

## C.4 Upgrade Behavior

```go
func (i *ImportManager) HandleUpgrade(newFile string, existingFile string, deleteOld bool) error {
    // Import new file
    if err := i.importFile(newFile); err != nil {
        return err
    }
    
    // Handle old file
    if deleteOld {
        if err := os.Remove(existingFile); err != nil {
            log.Warn("Failed to delete old file", "file", existingFile, "err", err)
        }
    }
    
    return nil
}
```

## C.5 Settings UI

```svelte
<div class="settings-section">
  <h3>Storage Management</h3>
  
  <label class="checkbox">
    <input type="checkbox" bind:checked={settings.storage_pause_enabled} />
    Pause downloads when disk space below:
  </label>
  
  <select bind:value={settings.storage_threshold_gb} disabled={!settings.storage_pause_enabled}>
    <option value={50}>50 GB</option>
    <option value={100}>100 GB</option>
    <option value={200}>200 GB</option>
    <option value={500}>500 GB</option>
  </select>
  
  <h4>On Upgrade</h4>
  <div class="radio-group">
    <label>
      <input type="radio" bind:group={settings.upgrade_delete_old} value={true} />
      Delete old file after upgrade
    </label>
    <label>
      <input type="radio" bind:group={settings.upgrade_delete_old} value={false} />
      Keep both files
    </label>
  </div>
</div>
```

## C.6 Status Display

Show storage status in UI (dashboard or settings):

```svelte
<div class="storage-status">
  {#each libraries as lib}
    <div class="library-storage">
      <span class="name">{lib.name}</span>
      <div class="bar">
        <div class="used" style="width: {lib.usedPercent}%"></div>
      </div>
      <span class="free">{lib.freeGB} GB free</span>
      {#if lib.freeGB < thresholdGB}
        <span class="warning">⚠️ Low space</span>
      {/if}
    </div>
  {/each}
</div>
```

---

# Phase D: Auto-Import & File Organization

## D.1 Database Schema

```sql
-- Download tracking
CREATE TABLE downloads (
    id INTEGER PRIMARY KEY,
    download_client_id INTEGER REFERENCES download_clients(id),
    external_id TEXT NOT NULL,           -- ID from download client
    
    media_id INTEGER,                    -- Matched media (null if unmatched)
    media_type TEXT,                     -- "movie", "episode"
    
    title TEXT NOT NULL,                 -- Grabbed release name
    size INTEGER,
    
    status TEXT NOT NULL,                -- "downloading", "completed", "importing", "imported", "failed"
    progress REAL DEFAULT 0,
    
    download_path TEXT,
    imported_path TEXT,
    
    error TEXT,
    
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Naming templates
CREATE TABLE naming_templates (
    id INTEGER PRIMARY KEY,
    type TEXT NOT NULL,                  -- "movie", "tv", "daily"
    folder_template TEXT NOT NULL,
    file_template TEXT NOT NULL,
    is_default BOOLEAN DEFAULT FALSE
);

-- Insert defaults
INSERT INTO naming_templates (type, folder_template, file_template, is_default) VALUES
('movie', '{Title} ({Year})', '{Title} ({Year})', TRUE),
('tv', '{Title} ({Year})/Season {Season:00}', '{Title} - S{Season:00}E{Episode:00} - {EpisodeTitle}', TRUE),
('daily', '{Title} ({Year})/Season {Year}', '{Title} - {Air-Date} - {EpisodeTitle}', TRUE);

-- Import history
CREATE TABLE import_history (
    id INTEGER PRIMARY KEY,
    download_id INTEGER REFERENCES downloads(id),
    source_path TEXT NOT NULL,
    dest_path TEXT NOT NULL,
    media_id INTEGER,
    media_type TEXT,
    success BOOLEAN,
    error TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

## D.2 Download Client Polling

```go
type DownloadClientManager struct {
    Clients  []DownloadClient
    Interval time.Duration // 30 seconds default
}

func (m *DownloadClientManager) Start() {
    ticker := time.NewTicker(m.Interval)
    
    for range ticker.C {
        m.checkAllClients()
    }
}

func (m *DownloadClientManager) checkAllClients() {
    for _, client := range m.Clients {
        downloads, err := client.GetDownloads()
        if err != nil {
            log.Error("Failed to get downloads", "client", client.Name(), "err", err)
            continue
        }
        
        for _, dl := range downloads {
            m.processDownload(client, dl)
        }
    }
}

func (m *DownloadClientManager) processDownload(client DownloadClient, dl Download) {
    // Update progress in database
    existing := m.db.GetDownloadByExternalID(client.ID, dl.ID)
    
    if existing == nil {
        // New download — try to match to wanted item
        m.handleNewDownload(client, dl)
        return
    }
    
    // Update progress
    existing.Progress = dl.Progress
    existing.Status = dl.Status
    m.db.UpdateDownload(existing)
    
    // Check if completed
    if dl.Status == "completed" && existing.Status != "imported" {
        m.importManager.QueueImport(existing, dl.Path)
    }
}
```

## D.3 Import Manager

```go
type ImportManager struct {
    db              *Database
    namingTemplates map[string]NamingTemplate
}

func (i *ImportManager) QueueImport(download *Download, sourcePath string) {
    go i.processImport(download, sourcePath)
}

func (i *ImportManager) processImport(download *Download, sourcePath string) {
    // Update status
    download.Status = "importing"
    i.db.UpdateDownload(download)
    
    // Find video files
    files, err := i.findVideoFiles(sourcePath)
    if err != nil {
        i.failImport(download, err)
        return
    }
    
    // Match to media
    media, err := i.matchToMedia(download, files)
    if err != nil {
        i.failImport(download, err)
        return
    }
    
    // Generate destination path
    destPath, err := i.generateDestPath(media)
    if err != nil {
        i.failImport(download, err)
        return
    }
    
    // Create folder structure
    if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
        i.failImport(download, err)
        return
    }
    
    // Move/rename main file
    mainFile := i.selectMainFile(files)
    if err := i.moveFile(mainFile, destPath); err != nil {
        i.failImport(download, err)
        return
    }
    
    // Handle extras
    extras := i.findExtras(files, mainFile)
    if len(extras) > 0 {
        extrasDir := filepath.Join(filepath.Dir(destPath), "Extras")
        os.MkdirAll(extrasDir, 0755)
        for _, extra := range extras {
            i.moveFile(extra, filepath.Join(extrasDir, filepath.Base(extra)))
        }
    }
    
    // Handle subtitles
    subs := i.findSubtitles(sourcePath)
    for _, sub := range subs {
        subDest := i.generateSubtitlePath(destPath, sub)
        i.moveFile(sub, subDest)
    }
    
    // Update database
    download.Status = "imported"
    download.ImportedPath = destPath
    i.db.UpdateDownload(download)
    
    // Update media quality status
    i.updateQualityStatus(media, mainFile)
    
    // Trigger library scan for this file
    i.libraryScanner.ScanFile(destPath)
    
    // Clean up empty folders
    i.cleanupSource(sourcePath)
    
    // Log import
    i.db.CreateImportHistory(&ImportHistory{
        DownloadID: download.ID,
        SourcePath: sourcePath,
        DestPath:   destPath,
        MediaID:    media.ID,
        MediaType:  media.Type,
        Success:    true,
    })
}

func (i *ImportManager) failImport(download *Download, err error) {
    download.Status = "failed"
    download.Error = err.Error()
    i.db.UpdateDownload(download)
    
    // Notify user
    notifyUser("Import failed: " + download.Title, err.Error())
}
```

## D.4 Naming Template Engine

```go
type NamingTemplate struct {
    FolderTemplate string
    FileTemplate   string
}

func (t *NamingTemplate) GeneratePath(media interface{}, libraryPath string) (string, error) {
    var folder, file string
    
    switch m := media.(type) {
    case *Movie:
        folder = t.replacePlaceholders(t.FolderTemplate, map[string]string{
            "{Title}": sanitizeFilename(m.Title),
            "{Year}":  strconv.Itoa(m.Year),
        })
        file = t.replacePlaceholders(t.FileTemplate, map[string]string{
            "{Title}": sanitizeFilename(m.Title),
            "{Year}":  strconv.Itoa(m.Year),
        })
        
    case *Episode:
        folder = t.replacePlaceholders(t.FolderTemplate, map[string]string{
            "{Title}":      sanitizeFilename(m.Show.Title),
            "{Year}":       strconv.Itoa(m.Show.Year),
            "{Season:00}":  fmt.Sprintf("%02d", m.Season),
        })
        file = t.replacePlaceholders(t.FileTemplate, map[string]string{
            "{Title}":        sanitizeFilename(m.Show.Title),
            "{Season:00}":    fmt.Sprintf("%02d", m.Season),
            "{Episode:00}":   fmt.Sprintf("%02d", m.Episode),
            "{EpisodeTitle}": sanitizeFilename(m.EpisodeTitle),
        })
        
    case *DailyEpisode:
        folder = t.replacePlaceholders(t.FolderTemplate, map[string]string{
            "{Title}": sanitizeFilename(m.Show.Title),
            "{Year}":  strconv.Itoa(m.AirDate.Year()),
        })
        file = t.replacePlaceholders(t.FileTemplate, map[string]string{
            "{Title}":        sanitizeFilename(m.Show.Title),
            "{Air-Date}":     m.AirDate.Format("2006-01-02"),
            "{EpisodeTitle}": sanitizeFilename(m.EpisodeTitle),
        })
    }
    
    return filepath.Join(libraryPath, folder, file+".mkv"), nil
}

func sanitizeFilename(s string) string {
    // Remove invalid characters: / \ : * ? " < > |
    invalid := regexp.MustCompile(`[/\\:*?"<>|]`)
    return invalid.ReplaceAllString(s, "")
}
```

## D.5 Extras Detection

```go
var extrasPatterns = []string{
    `(?i)extras?`,
    `(?i)featurettes?`,
    `(?i)bonus`,
    `(?i)deleted.?scenes?`,
    `(?i)behind.?the.?scenes?`,
    `(?i)making.?of`,
    `(?i)interview`,
    `(?i)trailer`,
    `(?i)gag.?reel`,
    `(?i)bloopers?`,
}

func (i *ImportManager) findExtras(files []string, mainFile string) []string {
    var extras []string
    
    for _, f := range files {
        if f == mainFile {
            continue
        }
        
        for _, pattern := range extrasPatterns {
            if regexp.MustCompile(pattern).MatchString(f) {
                extras = append(extras, f)
                break
            }
        }
    }
    
    return extras
}
```

## D.6 Multi-Episode Handling

```go
func (i *ImportManager) parseMultiEpisode(filename string) (int, int, bool) {
    // Match patterns like S01E01E02, S01E01-E02, S01E01-02
    patterns := []string{
        `S(\d+)E(\d+)E(\d+)`,
        `S(\d+)E(\d+)-E?(\d+)`,
    }
    
    for _, p := range patterns {
        re := regexp.MustCompile(p)
        if matches := re.FindStringSubmatch(filename); matches != nil {
            start, _ := strconv.Atoi(matches[2])
            end, _ := strconv.Atoi(matches[3])
            return start, end, true
        }
    }
    
    return 0, 0, false
}
```

## D.7 Failed Match Handling

```go
func (i *ImportManager) handleUnmatched(download *Download, files []string) {
    // Move to "Unmatched" folder for manual review
    unmatchedDir := filepath.Join(i.libraryPath, "_Unmatched")
    os.MkdirAll(unmatchedDir, 0755)
    
    destDir := filepath.Join(unmatchedDir, download.Title)
    os.MkdirAll(destDir, 0755)
    
    for _, f := range files {
        dest := filepath.Join(destDir, filepath.Base(f))
        i.moveFile(f, dest)
    }
    
    download.Status = "unmatched"
    download.ImportedPath = destDir
    i.db.UpdateDownload(download)
    
    // Notify user
    notifyUser("Manual import needed: " + download.Title, "Could not match to library item")
}
```

## D.8 API Endpoints

```
GET  /api/downloads                    — List all downloads
GET  /api/downloads/:id                — Get download details
DEL  /api/downloads/:id                — Remove download record

GET  /api/imports/history              — List import history
GET  /api/imports/unmatched            — List unmatched imports
POST /api/imports/unmatched/:id/match  — Manually match unmatched import

GET  /api/settings/naming              — Get naming templates
PUT  /api/settings/naming              — Update naming templates
```

## D.9 UI: Activity/Downloads Page

```svelte
<script>
  import { downloads } from '$lib/stores/downloads';
</script>

<div class="downloads-page">
  <h1>Activity</h1>
  
  <div class="section">
    <h2>Downloading</h2>
    {#each $downloads.filter(d => d.status === 'downloading') as dl}
      <div class="download-item">
        <div class="info">
          <span class="title">{dl.title}</span>
          <span class="size">{formatBytes(dl.size)}</span>
        </div>
        <div class="progress-bar">
          <div class="fill" style="width: {dl.progress}%"></div>
        </div>
        <span class="percent">{dl.progress.toFixed(1)}%</span>
      </div>
    {/each}
  </div>
  
  <div class="section">
    <h2>Importing</h2>
    {#each $downloads.filter(d => d.status === 'importing') as dl}
      <div class="download-item importing">
        <span class="title">{dl.title}</span>
        <span class="status">Importing...</span>
      </div>
    {/each}
  </div>
  
  <div class="section">
    <h2>Recently Imported</h2>
    {#each $downloads.filter(d => d.status === 'imported').slice(0, 10) as dl}
      <div class="download-item completed">
        <span class="title">{dl.title}</span>
        <span class="path">{dl.imported_path}</span>
        <span class="time">{formatRelative(dl.updated_at)}</span>
      </div>
    {/each}
  </div>
  
  {#if $downloads.filter(d => d.status === 'failed' || d.status === 'unmatched').length > 0}
    <div class="section warning">
      <h2>⚠️ Needs Attention</h2>
      {#each $downloads.filter(d => d.status === 'failed' || d.status === 'unmatched') as dl}
        <div class="download-item error">
          <span class="title">{dl.title}</span>
          <span class="error">{dl.error || 'Could not match'}</span>
          <button on:click={() => retryImport(dl.id)}>Retry</button>
          {#if dl.status === 'unmatched'}
            <button on:click={() => openManualMatch(dl.id)}>Match Manually</button>
          {/if}
        </div>
      {/each}
    </div>
  {/if}
</div>
```

---

# Phase A-D: Task Summary

## Implementation Order

1. **A.1-A.3**: Database schema + release parsing
2. **A.4-A.6**: Scoring + selection logic
3. **B.1-B.3**: Language settings + onboarding
4. **A.7-A.8**: Quality UI + API
5. **C.1-C.4**: Storage monitoring
6. **D.1-D.3**: Download tracking + import flow
7. **D.4-D.7**: Naming + extras + error handling
8. **B.4**: OpenSubtitles integration (if "always" mode)
9. **C.5-C.6, D.8-D.9**: Settings + Activity UI

## Dependencies

```
Quality Presets (A) ←── Language Settings (B)
       ↓
   Search Logic ←── Storage Check (C)
       ↓
   Grab Release
       ↓
Download Client Monitor (D)
       ↓
   Import Manager (D) ←── Naming Templates (D)
       ↓
   Library Updated
```

## Testing Checklist

- [ ] Release parsing handles edge cases
- [ ] Blocked groups never grabbed
- [ ] Language filter works correctly
- [ ] Storage threshold pauses downloads
- [ ] Import creates correct folder structure
- [ ] File renaming handles special characters
- [ ] Extras detected and moved to subfolder
- [ ] Multi-episode files handled
- [ ] Failed imports go to unmatched
- [ ] Subtitles move with video file
- [ ] Quality status updates after import
