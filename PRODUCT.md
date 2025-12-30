# PRODUCT.md - Product Vision

---

## What is Outpost?

Outpost is a unified self-hosted media platform that combines:

- **Media Server** (like Plex/Jellyfin) - organize and stream your library
- **Content Automation** (like Sonarr/Radarr/Lidarr) - monitor and download
- **Request Management** (like Jellyseerr) - discover and request content

**One app. All your media. Zero hassle.**

---

## Why Outpost?

### The Problem

The current self-hosted media stack requires 6+ separate applications:
- Plex or Jellyfin (media server)
- Sonarr (TV automation)
- Radarr (movie automation)
- Lidarr (music automation)
- Readarr (book automation)
- Prowlarr (indexer management)
- Jellyseerr or Overseerr (requests)

Each has its own:
- Web UI to manage
- Database to backup
- Updates to track
- Configuration to maintain
- Docker container to run

### The Solution

Outpost consolidates everything into a single application:
- One UI for all media types
- One database to backup
- One container to run
- One place to configure
- Seamless experience across library, discovery, and automation

---

## Target Users

### Primary: The Home Server Enthusiast
- Already runs or wants to run a media server
- Technical enough to set up Docker
- Wants the power of the *arr stack without the complexity
- Values ownership and privacy

### Secondary: The Streaming Refugee
- Tired of juggling 5+ streaming subscriptions
- Wants to own their media library
- Looking for a Plex alternative that "just works"

---

## Core Value Propositions

1. **Unified Experience** - Stop context-switching between apps
2. **Simpler Setup** - One Docker container, one config
3. **Beautiful UI** - Modern, content-first design
4. **All Media Types** - Movies, TV, anime, music, books, audiobooks
5. **Self-Hosted** - Your data stays yours

---

## What Outpost Is NOT

- **Not a download client** - Connects to qBittorrent, Transmission, SABnzbd, etc.
- **Not an indexer** - Connects to Prowlarr or manual Torznab/Newznab
- **Not a content provider** - You bring your own media
- **Not a public service** - Self-hosted only, no cloud hosting
- **Not a DVR** - No live TV recording

---

## Competitive Landscape

| Feature | Plex | Jellyfin | Outpost |
|---------|------|----------|---------|
| Media Server | ✅ | ✅ | ✅ |
| Free Transcoding | ❌ (paid) | ✅ | ✅ |
| Content Automation | ❌ | ❌ | ✅ |
| Request System | ❌ | ❌ | ✅ |
| All-in-One | ❌ | ❌ | ✅ |
| Open Source | ❌ | ✅ | ❌ |

---

## Success Metrics

### v1.0 Goals
- Successfully scan and serve all 6 media types
- Direct play works on web, mobile, TV
- Transcoding works when needed
- Download automation functional
- Request system functional
- 100 beta users

### Long-term Goals
- Become the go-to alternative to Plex + *arr stack
- Sustainable premium revenue from relay/mobile
- Active community of contributors and users
