# DECISIONS.md - Locked Decisions

> **These decisions are FINAL. Do not change without explicit user approval.**

---

## Product Identity

| Decision | Value |
|----------|-------|
| **Name** | Outpost |
| **Tagline** | "Stream in Sanctuary" |
| **Type** | Self-hosted media and entertainment platform |
| **License** | Proprietary (closed source) |
| **Business Model** | Freemium (free core + premium features) |

### Brand Aesthetic
- **Logo:** Cabin in the woods with moon, pine trees, chimney smoke, glowing windows
- **UI Style:** Monochrome glass, content-first (posters provide color)
- **Accent Color:** Warm amber (#fbbf24) - used only for logo windows and active states

---

## Tech Stack

| Layer | Technology | Reason |
|-------|------------|--------|
| **Backend** | Go 1.24 | Fast, single binary, great concurrency |
| **Frontend** | SvelteKit | Performance, small bundles, good DX |
| **Database** | SQLite (WAL mode) | Simple, portable, single file backup |
| **Styling** | Tailwind CSS | Utility-first, consistent |
| **Streaming** | FFmpeg | Industry standard transcoding |
| **Container** | Docker (Alpine) | Standard deployment |

---

## Pricing Model

### Always Free
- Server application
- Desktop web app
- All transcoding (CPU + GPU)
- Local streaming
- Library management (all media types)
- All metadata providers
- Download automation
- Unlimited local users
- Request system

### Premium ($5/mo or $40/yr)
- Remote relay (NAT punch-through)
- Cloud sync (settings, progress)
- TV app access
- Mobile apps (future)
- Invite/sharing links

**Philosophy:** Never charge for transcoding. Never host user content. Premium is for infrastructure costs.

---

## Media Types Supported

All in v1:

- Movies
- TV Shows
- Anime (separate from TV, with language/subtitle preferences)
- Music
- Books (epub, pdf, mobi, cbz/cbr)
- Audiobooks

### NOT in v1:
- Comics/Manga (use Books with cbz/cbr)
- Retro Games (explicitly out of scope)
- Live TV / DVR (explicitly out of scope)

---

## Core Features

### Library Management
- Import existing libraries
- Diagnose mismatches
- Manual match override
- Metadata from multiple sources
- User-editable metadata

### Automation/Monitoring
- Watch for new releases
- Send to external download client (NOT built-in)
- Support both torrent and usenet
- Quality profiles with scoring system

### Discovery (Jellyseerr-style)
- Recently Added
- Trending / Popular / Upcoming
- Request system
- Status tracking (Requested → Approved → Downloading → Available)

### Player
- Direct play (default)
- Transcode only if needed
- Subtitle selection
- Audio track selection
- Audio sync adjustment
- Skip intro (when detected)
- Skip credits / auto-next
- Resume playback
- Keyboard shortcuts

### Users & Permissions

| Role | Browse | Request | Auto-Approve | Download Details | Settings |
|------|--------|---------|--------------|------------------|----------|
| Admin | ✅ | ✅ | ✅ | Full | ✅ |
| User | ✅ | ✅ | Admin decides | Status only | ❌ |
| Kid | ✅ (filtered) | ✅ (requires approval) | ❌ | Status only | ❌ |

---

## Download Client Integration

**Supported torrent clients:**
- qBittorrent ✅
- Transmission ✅

**Supported usenet clients:**
- SABnzbd ✅
- NZBGet ✅

**Future:**
- Deluge
- rTorrent

**NOT building our own download client.**

---

## Indexer Integration

**Approach:** Prowlarr-compatible Torznab/Newznab
- Configure indexers manually (URL + API key)
- OR connect to Prowlarr and pull indexer list
- User choice, both work

**NOT maintaining our own indexer definitions.**

---

## Quality Profile System

Based on Sonarr's model:

- Profile name
- Upgrades allowed (yes/no)
- Upgrade until (ceiling quality)
- Minimum custom format score (floor)
- Upgrade until custom format score (ceiling)
- Minimum score increment for upgrades
- Custom formats with scores
- Quality tiers (checkboxes + drag to reorder)

---

## Metadata Sources

| Media Type | Primary Source | Status |
|------------|---------------|--------|
| Movies | TMDB | ✅ Working |
| TV Shows | TMDB | ✅ Working |
| Anime | AniList | 🔜 Planned |
| Music | File tags (MusicBrainz future) | ✅ Working |
| Books | File metadata | ✅ Working |
| Audiobooks | Audnexus | 🔜 Planned |

---

## Remote Access

**Premium feature:** Built-in relay service for NAT traversal

**Also supports:**
- Self-managed VPN
- Reverse proxy (nginx, Caddy, Traefik)
- Cloudflare Tunnel

---

## Mobile/TV Apps

**TV App:**
- Lean-back 10-foot UI
- D-pad/remote navigation
- Light admin (approve requests, mark watched)
- Premium required

**Mobile Apps (future):**
- iOS and Android
- Stream + offline download
- Premium required

---

## NOT Building

- Live TV / DVR
- Built-in download client
- Our own indexer definitions
- Public server sharing
- Retro game emulation
- Ad-supported tier

---

## Terminology

| Concept | Outpost Name |
|---------|--------------|
| TV Shows | TV |
| Movies | Movies |
| Anime | Anime |
| Music | Music |
| Books/Audiobooks | Books |
| Wanted/Wishlist | Wanted |
| Queue | Downloads |
| Discovery | Discover |
