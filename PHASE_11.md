# PHASE_11.md - Integrated Experience

> **THIS IS NOT A UI REFRESH. This is building the core Netflix-like experience.**

---

## Goal

Users should be able to:
1. Open Outpost
2. See what they were watching → Continue immediately
3. Browse content → See if it's available, requested, or downloadable
4. One click to play OR request
5. See download progress without leaving the page
6. Never feel like they're using "separate apps glued together"

---

## Priority Order

**Build in this order. Do not skip ahead.**

### Priority 1: Continue Watching

This is THE most important feature. Users open the app and immediately see what they were watching.

**Data Model:**
```sql
-- Already exists as 'progress' table, but needs:
-- - Query for "in progress" items (position > 0, not completed)
-- - Sorted by updated_at DESC
-- - Include media details (title, poster, episode info)
```

**API Endpoint:**
```
GET /api/continue-watching
Returns: Array of in-progress items with:
  - media_type (movie/episode)
  - media_id
  - title
  - poster_path
  - backdrop_path (for continue watching cards)
  - progress_percent
  - position (seconds)
  - duration (seconds)
  - episode_info (if TV): { show_title, season, episode, episode_title }
```

**UI Requirements:**
- First row on home page, above everything else
- Horizontal scroll
- Card shows: Backdrop (16:9), title, episode info if TV, progress bar
- One click → Opens player at saved position
- X button to remove from continue watching (marks as not in progress)

**Card Component:**
```
┌─────────────────────────────────────┐
│                                     │
│         Backdrop Image              │
│                                     │
│  ▶ Play                             │  ← Centered play button
│                                     │
├─────────────────────────────────────┤
│ Show Title                          │
│ S1 E4 · Episode Name                │
│ ████████░░░░░░░░ 45%                │  ← Progress bar
└─────────────────────────────────────┘
```

---

### Priority 2: Card Status System

Every card must show its status at a glance.

**Status Types:**
| Status | Badge | Position | Color |
|--------|-------|----------|-------|
| In Library | ✓ checkmark | Top right | Green |
| Requested (pending) | Clock icon | Top right | Yellow/Amber |
| Approved | Clock + check | Top right | Blue |
| Downloading | ↓ Arrow + % | Top right | Blue |
| Not in library | (nothing) | - | - |

**Type Badge:**
| Type | Badge | Position | Color |
|------|-------|----------|-------|
| Movie | "MOVIE" | Top left | Amber/Yellow |
| Series | "SERIES" | Top left | Blue |
| Anime | "ANIME" | Top left | Purple |

**Rating Badge:**
- Position: Top right, below status badge
- Format: ★ 7.4
- Background: black/60 blur

**Implementation:**
```javascript
// Every card needs this data:
{
  id: 123,
  title: "Movie Name",
  year: 2024,
  poster_path: "/path.jpg",
  media_type: "movie", // movie, series, anime
  rating: 7.4,
  
  // Status - ONE of these:
  library_status: "available" | "not_in_library",
  request_status: null | "pending" | "approved" | "denied",
  download_status: null | { progress: 45, speed: "2.5 MB/s" }
}
```

**API Endpoint:**
```
GET /api/discover/movies (or /tv, /anime)
Each item must include:
  - library_status: Check if tmdb_id exists in our library
  - request_status: Check requests table
  - download_status: Check active downloads matching this tmdb_id
```

---

### Priority 3: Smart Card Actions

The hover/click action depends on status.

**Logic:**
```
IF library_status === "available":
  Show: ▶ Play button
  Click: Open player (or detail page with play button)

ELSE IF request_status === "pending":
  Show: Clock icon, "Requested" text
  Click: Open detail page (can't do anything)

ELSE IF request_status === "approved" OR download_status:
  Show: Download progress or "Downloading"
  Click: Open detail page

ELSE (not in library, not requested):
  Show: + Request button
  Click: Trigger request (inline, don't navigate)
```

**Hover State:**
```
┌─────────────────┐
│ MOVIE       ✓   │  ← Type + Status always visible
│                 │
│   [▶ Play]      │  ← Action button on hover (centered)
│                 │
│           ★ 7.4 │  ← Rating
└─────────────────┘
  Title
  2024
```

---

### Priority 4: Inline Request Flow

Users should request content without leaving the browse page.

**Flow:**
1. User hovers card without ✓ (not in library)
2. Shows "+ Request" button
3. User clicks → Button changes to "Requesting..."
4. API call completes → Badge changes to 🕐 (pending)
5. Toast notification: "Requested! We'll notify you when it's ready."

**NO modal. NO navigation. Inline.**

**API:**
```
POST /api/requests
Body: { media_type: "movie", tmdb_id: 12345 }
Response: { status: "pending", request_id: 456 }
```

---

### Priority 5: Download Status on Cards

When something is downloading, show it on the card.

**Requirements:**
- Poll download status every 10-30 seconds
- Match downloads to tmdb_id (need to store tmdb_id when sending to download client)
- Show progress % on card badge
- Optional: Show ETA or speed

**Badge when downloading:**
```
↓ 45%
```

**Card overlay when downloading:**
```
┌─────────────────┐
│ MOVIE      ↓45% │
│                 │
│  Downloading... │  ← Instead of play button
│  2.5 MB/s       │
│                 │
└─────────────────┘
```

---

### Priority 6: Home Page Structure

**Order matters. This is the hierarchy:**

```
┌─────────────────────────────────────────────────────────────┐
│  [Hero - Optional, YOUR content or currently downloading]   │
├─────────────────────────────────────────────────────────────┤
│  Continue Watching  ←  FIRST ROW, ALWAYS                    │
│  [Card] [Card] [Card] [Card] →                              │
├─────────────────────────────────────────────────────────────┤
│  Recently Added  ←  What's new in YOUR library              │
│  [Card] [Card] [Card] [Card] →                              │
├─────────────────────────────────────────────────────────────┤
│  Trending Movies  ←  Discovery (TMDB), shows status badges  │
│  [Card] [Card] [Card] [Card] →                              │
├─────────────────────────────────────────────────────────────┤
│  Trending TV Shows                                          │
│  [Card] [Card] [Card] [Card] →                              │
└─────────────────────────────────────────────────────────────┘
```

**Key Points:**
- Continue Watching is ALWAYS first (if user has in-progress items)
- Recently Added shows YOUR library, not TMDB
- Trending rows show TMDB content WITH library/request status

---

### Priority 7: Unified Search

Search shows both library AND requestable content.

**Results Layout:**
```
Search: "avatar"

In Your Library (2)
┌──────┐ ┌──────┐
│ ✓    │ │ ✓    │
│Avatar│ │Avatar│
│ 2009 │ │ 2022 │
└──────┘ └──────┘

Available to Request (3)
┌──────┐ ┌──────┐ ┌──────┐
│      │ │      │ │ 🕐   │
│Avatar│ │Avatar│ │Avatar│
│ 3    │ │ 4    │ │ Game │
└──────┘ └──────┘ └──────┘
         (requested)
```

**Each result has same card status system.**

---

### Priority 8: Visual Polish (LAST)

Only after 1-7 work:

- Glass blur effects
- Sidebar styling (dark zinc-900, collapsible)
- Hero carousel
- Dropdown filters
- Smooth transitions (300ms)
- Hover animations (scale 1.05)

---

## DO and DON'T

### DO:
- Check library status on EVERY card
- Show continue watching FIRST
- Make request work inline (no navigation)
- Update card status after request without refresh
- Show download progress on cards
- Make Play button obvious for library items

### DON'T:
- Show Play button for items not in library
- Navigate away for simple actions (request)
- Hide status information
- Make users guess if they own something
- Prioritize visual polish over functionality
- Build trending/discover before continue watching works

---

## Definition of Done

Phase 11 is complete when:

1. [ ] User opens app → Sees continue watching row with their in-progress items
2. [ ] User clicks continue watching card → Player opens at saved position
3. [ ] Every card shows: Type badge, status badge, rating
4. [ ] Library items show ✓ and Play button on hover
5. [ ] Non-library items show + Request on hover
6. [ ] Clicking Request → Card updates to show 🕐 without page refresh
7. [ ] Downloading items show progress on card
8. [ ] Search shows library and requestable items clearly separated
9. [ ] Home page order: Continue Watching → Recently Added → Trending
10. [ ] Card click → Opens detail page
11. [ ] Play button → Plays directly (next unwatched for TV)
12. [ ] Mark as watched works (without playing)
13. [ ] Mark as unwatched works (resets progress)
14. [ ] Bulk mark season/series as watched/unwatched
15. [ ] Watch states visible: Unwatched (clean), Partial (progress bar), Watched (✓ or dimmed)
16. [ ] TV shows indicate episode progress (4 of 12 watched)
17. [ ] THEN: Visual polish applied

---

## Technical Notes

### State Management
Cards need reactive state. When a request is made:
- Optimistically update the card's request_status
- If API fails, revert and show error

### Polling
Download status should poll:
- Every 10 seconds when downloads are active
- Stop polling when no active downloads
- Use WebSocket if available (future)

### Caching
- Library status can be cached (invalidate on library scan)
- Request status should be fresh (or use WebSocket)
- Download status must be real-time

---

## Watch State Indicators

Every library item needs to show watch state at a glance.

**States:**
| State | Indicator | Notes |
|-------|-----------|-------|
| Unwatched | (none) | Clean card, no indicator |
| Partially watched | Progress bar | Shows % complete at bottom of card |
| Watched | ✓ Checkmark overlay or dimmed | Clearly "done" |

**On Poster Cards:**
```
Unwatched:          Partial:            Watched:
┌──────────┐       ┌──────────┐        ┌──────────┐
│          │       │          │        │    ✓     │  ← Small checkmark
│  Poster  │       │  Poster  │        │  Poster  │     OR dimmed/faded
│          │       │          │        │ (dimmed) │
│          │       ├──────────┤        │          │
└──────────┘       │███░░░ 45%│        └──────────┘
                   └──────────┘
```

**On Episode Lists:**
```
┌──────────────────────────────────────────────────────────────┐
│ 1  │  Pilot                              45m         ✓      │  ← Watched
├──────────────────────────────────────────────────────────────┤
│ 2  │  The Train Job                      44m    ███░░ 60%   │  ← Partial
├──────────────────────────────────────────────────────────────┤
│ 3  │  Bushwhacked                        44m               │  ← Unwatched
└──────────────────────────────────────────────────────────────┘
```

**On Show Cards (aggregate):**
- Show overall progress: "4 of 12 episodes watched"
- Or progress bar representing series completion
- Or badge: "NEW EPISODES" if unwatched episodes exist

**API must return:**
```javascript
{
  // For movies:
  watch_state: "unwatched" | "partial" | "watched",
  progress_percent: 45, // if partial
  
  // For shows:
  watch_state: "unwatched" | "partial" | "watched",
  episodes_watched: 4,
  episodes_total: 12,
  next_episode: { season: 1, episode: 5, title: "..." }
}
```

**Visual Hierarchy:**
1. Unwatched = Full brightness, inviting to watch
2. Partial = Progress bar draws attention, "continue this"
3. Watched = Slightly dimmed or checkmark, "you've seen this"

---

1. **Card click → Detail page** | **Play button → Plays directly**
2. **TV Play button → Plays next unwatched episode**
3. **Same card actions for everyone** - Permissions (delete, etc.) handled in admin panel settings, not different card UI

---

## Additional Feature: Mark as Watched / Unwatched

Users need to toggle watch state without playing content.

**Actions needed:**
| Action | What it does |
|--------|--------------|
| Mark as Watched | Sets to 100% complete, removes from Continue Watching |
| Mark as Unwatched | Resets progress to 0%, removes ✓ indicator |

**Where:**
- On episode cards (checkmark icon toggles)
- On detail page (per episode or "Mark season watched/unwatched")
- On movie cards (context menu or icon)
- In continue watching row (mark complete or remove)
- Bulk action: "Mark all as watched" for seasons/series

**UI:**
```
Episode Card:
┌──────────────────────────────────────────────────────────────┐
│ 4  │  Episode Title                      45m    [✓] [▶]     │
│    │  Episode description...                                │
└──────────────────────────────────────────────────────────────┘
                                                 ↑     ↑
                                          Toggle watched  Play

Context Menu (right-click or ⋮ button):
┌─────────────────────┐
│ ▶ Play              │
│ ✓ Mark as Watched   │  ← Or "Mark as Unwatched" if already watched
│ + Add to List       │
│ ↓ Download          │
└─────────────────────┘
```

**States:**
- Unwatched: Empty checkbox → Click = Mark Watched
- Watched: Filled checkmark (✓) → Click = Mark Unwatched
- In progress: Shows progress → Click = Mark Watched (completes it)

**API:**
```
PUT /api/progress/:type/:id
Body: { completed: true }   // Mark as watched

PUT /api/progress/:type/:id  
Body: { completed: false, position: 0 }  // Mark as unwatched (reset)

DELETE /api/progress/:type/:id  // Alternative: remove all progress

// Bulk operations:
PUT /api/shows/:id/seasons/:season/watched
Body: { watched: true }  // Mark entire season

PUT /api/shows/:id/watched
Body: { watched: true }  // Mark entire series
```
