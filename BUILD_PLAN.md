# BUILD_PLAN.md - Implementation Phases

---

## Overview

Phases 1-10 are **COMPLETE**. Current work begins at Phase 11.

---

## ✅ COMPLETED PHASES (1-10)

### Phase 1: Project Scaffolding ✅
- Go backend structure
- SvelteKit frontend structure
- Docker configuration
- Basic project files

### Phase 2: Database Schema ✅
- SQLite with WAL mode
- 19 tables defined
- Migration system

### Phase 3: Library Management ✅
- Library CRUD
- Folder scanning
- File detection by type

### Phase 4: Metadata Integration ✅
- TMDB client
- Movie metadata fetching
- TV show metadata fetching
- Poster/backdrop downloads

### Phase 5: Streaming Backend ✅
- Direct play endpoint
- FFmpeg transcoding
- Audio transcoding to AAC
- Range request support

### Phase 6: User System ✅
- User CRUD
- Role system (admin/user/kid)
- Session-based auth
- bcrypt password hashing

### Phase 7: Frontend Core ✅
- Basic layout
- Library browsing
- Detail pages
- Video player

### Phase 8: Download Automation ✅
- Download client integration (qBit, Transmission, SABnzbd, NZBGet)
- Indexer integration (Torznab/Newznab)
- Quality profiles
- Custom formats
- Wanted list
- Auto-search scheduler

### Phase 9: Request System ✅
- Request CRUD
- Approval workflow
- Status tracking

### Phase 10: Polish & Fixes ✅
- Progress tracking
- Resume playback
- Search functionality
- Settings pages
- Bug fixes

---

## 🔄 CURRENT PHASE

### Phase 11: UI Refresh

**Goal:** Update frontend to match new glass design system

**Design Files:**
- `mockups/outpost-v2.jsx` - Desktop app
- `mockups/outpost-tv.jsx` - TV app
- `docs/UI.md` - Component specifications

**Tasks:**

#### 11.1 Design System Setup
- [ ] Update Tailwind config with new color palette
- [ ] Add glass panel utility classes
- [ ] Add animation utilities (transitions, hover states)
- [ ] Create base component styles

#### 11.2 Layout Components
- [ ] New Sidebar (slim, 64px, icon-only)
- [ ] New TopBar (centered search, notifications, profile)
- [ ] Search overlay with live suggestions
- [ ] Glass panel component

#### 11.3 Card Components
- [ ] PosterCard (2:3 ratio, hover effects, play overlay)
- [ ] ContinueCard (16:9, progress bar, direct play button)
- [ ] StatCard (for settings dashboard)
- [ ] EpisodeCard (for TV detail pages)

#### 11.4 Page Updates
- [ ] Home page (hero + continue watching + rows)
- [ ] Detail page (full page, not modal)
- [ ] Library pages (grid layouts)
- [ ] Settings (tabbed admin dashboard with stat cards)

#### 11.5 Player Enhancements
- [ ] Auto-hiding controls
- [ ] Stats panel (codec, bitrate, etc.)
- [ ] Audio sync panel (-500ms to +500ms)
- [ ] Cast & crew panel
- [ ] Audio track selection
- [ ] Subtitle selection
- [ ] Skip intro button (at 5-15% progress)
- [ ] Up next card (at >90% progress)

**Definition of Done:**
- All pages match design mockups
- Smooth 300ms transitions
- Responsive down to 768px (tablet)
- All existing functionality preserved

---

## 📋 UPCOMING PHASES

### Phase 12: Subtitles

**Goal:** Full subtitle support in player

**Tasks:**
- [ ] Extract embedded subtitles from MKV (FFmpeg)
- [ ] Convert to WebVTT for browser
- [ ] Detect external .srt, .ass, .vtt files
- [ ] Associate subtitles with videos
- [ ] Subtitle track selection in player
- [ ] Basic subtitle styling options
- [ ] Subtitle sync adjustment

**Definition of Done:**
- Embedded subs display in player
- External subs detected and selectable
- User can adjust timing

---

### Phase 13: Anime Support (AniList)

**Goal:** Proper anime metadata via AniList

**Tasks:**
- [ ] AniList GraphQL API client
- [ ] Search anime
- [ ] Get anime details
- [ ] Episode metadata
- [ ] Store AniList IDs
- [ ] Anime-specific filename parsing
- [ ] Japanese/Romaji/English title handling
- [ ] Airing status for ongoing shows

**Definition of Done:**
- Anime libraries scan with proper metadata
- Episode titles from AniList
- Better matching than TMDB for anime

---

### Phase 14: Audiobook Support (Audnexus)

**Goal:** Audiobook metadata and player

**Tasks:**
- [ ] Audnexus API client
- [ ] Search audiobooks
- [ ] Get audiobook details
- [ ] Chapter information
- [ ] Parse m4b/mp3 chapter markers
- [ ] Audiobook player UI
- [ ] Chapter navigation
- [ ] Sleep timer
- [ ] Playback speed control (0.5x - 3x)

**Definition of Done:**
- Audiobooks scan with metadata
- Chapter markers work
- Player has audiobook-specific controls

---

### Phase 15: Hardware Transcoding

**Goal:** GPU-accelerated transcoding

**Tasks:**
- [ ] Detect available GPU (NVIDIA, Intel QSV, AMD)
- [ ] NVIDIA NVENC encoding
- [ ] Intel Quick Sync encoding
- [ ] Graceful fallback to CPU
- [ ] Settings UI for hardware acceleration
- [ ] Quality presets

**Definition of Done:**
- Transcoding uses GPU when available
- Significant performance improvement
- Clean fallback if GPU unavailable

---

### Phase 16: Remote Relay (Premium)

**Goal:** Access Outpost from outside home network

**Tasks:**
- [ ] Relay server architecture
- [ ] Client-server handshake
- [ ] NAT traversal
- [ ] Bandwidth management
- [ ] License/premium check
- [ ] Payment integration (Stripe)
- [ ] Premium UI gating

**Definition of Done:**
- Users can access from outside network
- Works through NAT/firewalls
- Premium users only

---

### Phase 17: TV App

**Goal:** Dedicated lean-back experience

**Tasks:**
- [ ] Decide platform (Web PWA vs native)
- [ ] 10-foot UI with large touch targets
- [ ] D-pad/remote navigation
- [ ] Focus management
- [ ] Horizontal scroll rows
- [ ] Light admin (approve requests, mark watched)
- [ ] Premium gating

**Definition of Done:**
- Usable with TV remote
- Core playback works
- Basic admin actions

---

### Phase 18: Mobile Apps

**Goal:** Native iOS and Android apps

**Tasks:**
- [ ] Framework decision (React Native vs Flutter)
- [ ] Library browsing
- [ ] Video/audio playback
- [ ] Downloads for offline
- [ ] Push notifications
- [ ] App store submission
- [ ] Premium gating

**Definition of Done:**
- Apps in App Store and Play Store
- Full playback functionality
- Offline downloads work

---

## 🗄️ BACKLOG (Unscheduled)

- Collections / Playlists
- Watch party (sync playback)
- Parental controls (content filtering)
- Intro/outro detection (auto-skip)
- Watch history / statistics
- Trakt.tv integration
- Smart playlists
- Notification system (Discord, email, webhooks)
- OpenSubtitles integration
- Fanart.tv integration
- Multi-server support

---

## ❌ NOT PLANNED

- Live TV / DVR
- Content hosting
- Built-in download clients
- VPN functionality
- Retro game emulation
