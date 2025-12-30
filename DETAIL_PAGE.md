# DETAIL_PAGE.md - Movie & TV Detail Pages

> **Goal:** Rich, informative detail pages - everything visible, no scrolling required.

---

## Design Principles

1. **No vertical scrolling** - Everything fits in one viewport
2. **Information dense** - All key info visible immediately
3. **Actionable** - Play, audio, subtitles all accessible without digging
4. **Hero backdrop** - Visual appeal behind content

---

## Movie Detail Page

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                                                                             │
│  ← Back to Library                                      🔔  👤 admin        │
│                                                                             │
│  ┌─────────────┐   MOVIE TITLE                           🍅 93%  Ⓜ 78     │
│  │             │   2024  •  2h 28m  •  PG-13           Action, Sci-Fi      │
│  │             │                                                            │
│  │   POSTER    │   DIRECTED BY    James Cameron                            │
│  │             │   WRITTEN BY     James Cameron, Josh Friedman             │
│  │             │   STUDIO         20th Century Studios                     │
│  │             │                                                            │
│  │             │   VIDEO          [4K Dolby Vision (24GB) ▼]               │
│  │             │   AUDIO          [English (Atmos 7.1) ▼]                  │
│  │             │   SUBTITLES      [English ▼]                              │
│  └─────────────┘                                                            │
│                    "Tagline goes here in italics"                           │
│                                                                             │
│                    Plot overview text goes here. Two to three lines max    │
│                    to keep everything above the fold...                     │
│                                                                             │
│                    [▶ Play]  [+ Add to List]  [▶ Trailer]  [✓ Watched]    │
│                                                                             │
│                    [TMDB] [IMDb] [Trakt]  ← Small icon links               │
│                                                                             │
├─────────────────────────────────────────────────────────────────────────────┤
│  CAST                                                              < >     │
│  (●) (●) (●) (●) (●) (●) (●) (●)                                          │
│  Name Name Name Name Name Name Name Name                                   │
│  Char Char Char Char Char Char Char Char                                   │
├─────────────────────────────────────────────────────────────────────────────┤
│  MORE LIKE THIS                                                    < >     │
│  [Card] [Card] [Card] [Card] [Card] [Card]                                 │
└─────────────────────────────────────────────────────────────────────────────┘

         ↑ BACKDROP IMAGE behind everything with dark gradient overlay
```

### Layout Breakdown

**Top Section (60% of viewport):**
| Left Side | Right Side |
|-----------|------------|
| Poster (fixed height) | Title + year + runtime + rating |
| | Genres + external ratings (RT, Metacritic) |
| | Director, Writer, Studio |
| | Video/Audio/Subtitle dropdowns |
| | Tagline + Overview (2-3 lines) |
| | Action buttons |
| | External links |

**Bottom Section (40% of viewport):**
- Cast row (horizontal scroll)
- More Like This row (horizontal scroll)

### Dropdowns (Inline Selectors)

**Video Version:**
```
VIDEO  [4K Dolby Vision (24GB) ▼]
       ├─ 4K Dolby Vision (24GB) ✓
       ├─ 4K HDR10 (18GB)
       ├─ 1080p (8GB)
       └─ 720p (4GB)
```

**Audio:**
```
AUDIO  [English (Atmos 7.1) ▼]
       ├─ English (Atmos 7.1) ✓
       ├─ English (Commentary)
       ├─ Spanish (5.1)
       └─ French (Stereo)
```

**Subtitles:**
```
SUBTITLES  [English ▼]
           ├─ Off
           ├─ English ✓
           ├─ English (SDH)
           ├─ Spanish
           └─ [+ Download...]
```

---

## TV Show Detail Page

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                                                                             │
│  ← Back                                                 🔔  👤 admin        │
│                                                                             │
│  ┌─────────────┐   SHOW TITLE                            🍅 95%  Ⓜ 82     │
│  │             │   2019-2024  •  4 Seasons  •  TV-MA     Drama, Thriller   │
│  │             │   48 Episodes  •  45min avg  •  Ended                     │
│  │   POSTER    │                                                            │
│  │             │   CREATED BY     Vince Gilligan                           │
│  │             │   NETWORK        AMC, Netflix                              │
│  │             │                                                            │
│  │             │   AUDIO          [English (5.1) ▼]                        │
│  │             │   SUBTITLES      [English ▼]                              │
│  └─────────────┘                                                            │
│                    Overview text here, kept brief...                        │
│                                                                             │
│                    [▶ Play S2 E5]  [+ Add to List]  [▶ Trailer]           │
│                         ↑ Shows next unwatched episode                      │
│                                                                             │
├─────────────────────────────────────────────────────────────────────────────┤
│  EPISODES   [Season 1 ▼]                      12 of 12 watched   [Mark All]│
├─────────────────────────────────────────────────────────────────────────────┤
│  1 │ Pilot                    │ 45m │ ✓  │  2 │ The Train Job    │ 44m │ ✓ │
│  3 │ Bushwhacked              │ 44m │▋60%│  4 │ Safe              │ 43m │   │
│  5 │ Our Mrs. Reynolds        │ 44m │    │  6 │ Shindig           │ 44m │   │
│  7 │ Jaynestown               │ 44m │    │  8 │ Out of Gas        │ 44m │   │
├─────────────────────────────────────────────────────────────────────────────┤
│  CAST                                                              < >     │
│  (●) (●) (●) (●) (●) (●) (●)                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

### TV-Specific Elements

**Episode Grid (instead of list):**
- 2-column grid fits more episodes
- Shows: Number, Title, Runtime, Watch status
- Hover: Play button appears
- Click: Plays episode
- Watch status: ✓ (watched), progress bar (partial), empty (unwatched)

**Season Selector:**
- Dropdown in episodes header
- Shows watch progress: "12 of 12 watched"
- [Mark All] button for bulk watched/unwatched

**Play Button:**
- Shows next episode: "Play S2 E5"
- If nothing in progress, shows "Play S1 E1"

---

## For Content Not in Library

When viewing a movie/show you don't have:

```
┌─────────────────────────────────────────────────────────────────────────────┐
│  ┌─────────────┐   MOVIE TITLE                           🍅 93%  ⭐ 8.2   │
│  │             │   2024  •  2h 28m  •  PG-13           Action, Sci-Fi      │
│  │   POSTER    │                                                            │
│  │             │   DIRECTED BY    Christopher Nolan                        │
│  │             │   WRITTEN BY     Christopher Nolan                        │
│  │             │   STUDIO         Warner Bros                              │
│  │             │                                                            │
│  │             │   (No video/audio/subtitle dropdowns - not in library)    │
│  └─────────────┘                                                            │
│                    Overview text...                                         │
│                                                                             │
│                    [+ Request]  [▶ Trailer]                                │
│                                                                             │
│                    Status: Not in library                                   │
│                    -- or --                                                 │
│                    Status: 🕐 Requested (pending approval)                  │
│                    -- or --                                                 │
│                    Status: ↓ Downloading 45%                                │
│                                                                             │
├─────────────────────────────────────────────────────────────────────────────┤
│  CAST                                                              < >     │
├─────────────────────────────────────────────────────────────────────────────┤
│  MORE LIKE THIS                                                    < >     │
└─────────────────────────────────────────────────────────────────────────────┘
```

**Changes:**
- No video/audio/subtitle selectors
- [+ Request] instead of [▶ Play]
- Shows request/download status

---

## Actor Modal

Keep it simple - modal instead of full page:

```
┌─────────────────────────────────────────────────────────────────────────────┐
│  ACTOR NAME                                                            [X] │
├─────────────────────────────────────────────────────────────────────────────┤
│  ┌────────┐   Born: January 15, 1980 (44 years old)                        │
│  │        │   Los Angeles, California, USA                                 │
│  │ Photo  │                                                                 │
│  │        │   Brief biography text here...                                 │
│  └────────┘                                                                 │
│               [IMDb] [TMDB]                                                 │
├─────────────────────────────────────────────────────────────────────────────┤
│  IN YOUR LIBRARY                                                   < >     │
│  [Movie] [Movie] [Show]  ← Content you have with this actor                │
├─────────────────────────────────────────────────────────────────────────────┤
│  KNOWN FOR                                                         < >     │
│  [Movie] [Movie] [Movie] [Movie]  ← Top credits from TMDB                  │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## Trailer Modal

```
┌─────────────────────────────────────────────────────────────────────────────┐
│  Movie Title - Official Trailer                                        [X] │
├─────────────────────────────────────────────────────────────────────────────┤
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                                                                     │   │
│  │                      YouTube Embed                                  │   │
│  │                      or Local Video Player                          │   │
│  │                                                                     │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## Definition of Done

Detail pages are complete when:

**Layout (No Scroll):**
1. [ ] Everything fits in viewport - no vertical scroll needed
2. [ ] Hero backdrop behind content with dark gradient
3. [ ] Poster + metadata section properly aligned

**Movie Page:**
4. [ ] Title, year, runtime, certification, genres visible
5. [ ] External ratings (RT, Metacritic) if available
6. [ ] Director, Writer, Studio displayed
7. [ ] Video version dropdown (if multiple files)
8. [ ] Audio track dropdown
9. [ ] Subtitle dropdown
10. [ ] Tagline + Overview (truncated if needed)
11. [ ] Action buttons: Play, Add to List, Trailer, Watched
12. [ ] External links: TMDB, IMDb, Trakt
13. [ ] Cast row with photos (horizontal scroll)
14. [ ] More Like This row (horizontal scroll)

**TV Page:**
15. [ ] Season/episode count + show status
16. [ ] Created By + Networks
17. [ ] Episode grid with watch status
18. [ ] Season dropdown with progress ("12 of 12")
19. [ ] Mark All button for bulk watched/unwatched
20. [ ] Play shows next episode ("Play S2 E5")

**Not In Library:**
21. [ ] Request button instead of Play
22. [ ] Shows request/download status
23. [ ] No video/audio/subtitle dropdowns

**Actor Modal:**
24. [ ] Photo, name, birth info
25. [ ] Brief bio
26. [ ] In Your Library row
27. [ ] Known For row

**Trailer Modal:**
28. [ ] YouTube embed or local video
29. [ ] Clean modal with close button

---

## Out of Scope (Future)

- User reviews/ratings
- Comments
- Activity feed
- Full filmography on actor modal
- Collection view (separate page)
