# Outpost UI Specification

Reference mockup: `outpost-mockup-v4.html`

## Design System

### Colors
```css
--bg-primary: #0a0a0a
--text-primary: #F5E6C8 (cream)
--text-secondary: rgba(245, 230, 200, 0.7)
--text-muted: rgba(245, 230, 200, 0.5)
--accent: #E8A849 (amber)
--success: #4ade80 (green)
--bg-glass: rgba(255, 255, 255, 0.06)
--bg-glass-hover: rgba(255, 255, 255, 0.1)
--border-subtle: rgba(255, 255, 255, 0.1)
```

### Spacing
- Page padding: 60px horizontal
- Section gap: 30px
- Card gap in rows: 16px
- Border radius: 8px (sm), 12px (md), 16px (lg), 9999px (full/pill)

### Touch Targets
- Minimum 44px for interactive elements
- Focus states with 2px accent ring

---

## Layout Structure

### Top Navigation Bar
- Fixed position, glass background with blur
- Height: 64px
- Contents:
  - Logo banner (outpost-banner-transparent.png, 40px height)
  - Nav tabs: Home, Library, Explore
  - Right side: Search input, Requests dropdown, User avatar

### Pages
1. **Home** - Continue watching, your list, coming soon, recommendations
2. **Library** - Filter tabs (All/Movies/TV/4K/etc), grid of owned content
3. **Explore** - Discover new content, trending, popular by category
4. **Movie Detail** - Full metadata, play controls, cast/crew
5. **TV Detail** - Same as movie + season selector, episode slider

---

## Components

### Media Card (Poster)
- Aspect ratio: 2:3
- Hover: slight scale, shadow
- Badge position: top-right (4K, HD, etc)
- Progress bar: bottom, amber fill

### Media Card (Landscape)
- Aspect ratio: 16:9
- Used for: Continue watching, episodes
- Shows progress bar when applicable

### Episode Card (Horizontal Slider)
- Width: 240px
- 16:9 thumbnail
- Episode number badge (top-left, circular)
- Duration badge (bottom-right)
- Checkmark badge for watched (top-right, green)
- Amber ring for in-progress
- Progress bar at bottom
- Title + date below thumbnail

### Coming Soon Card
- Landscape thumbnail
- Date badge (top-left)
- Title, subtitle, meta below

### Cast/Crew Card
- Circular photo (64px)
- Name + role below
- Width: 85px

### Scroll Row
- Horizontal scroll with snap
- Padding: 60px left (aligns with page)
- Gap: 16px between items
- Scroll arrows in section header

### Section Header
- Title left, controls right
- Controls: "See All" link + scroll arrows (‹ ›)

---

## Detail Page Layout

### Movie Detail
```
┌─────────────────────────────────────────────────────┐
│ [Backdrop image with gradient fade]                 │
├──────────┬────────────────────────┬─────────────────┤
│ POSTER   │ CENTER                 │ INFO PANEL      │
│ 220px    │ (flex)                 │ 280px           │
│          │                        │                 │
│ [Image]  │ Genre tags (pills)     │ Critic scores   │
│ Title    │ Tagline (italic)       │ Status          │
│ Year•Run │ Description            │ Released        │
│ •Rating  │                        │ Runtime         │
│          │                        │ Theatrical date │
│ Actions: │                        │ Digital date    │
│ ✓ 👁 ▶ 🎬 ⋮│                        │ Budget/Revenue  │
│          │                        │ Language        │
│ Selectors│                        │ Country (flag)  │
│ 🎬 4K ▾   │                        │ Studios         │
│ 🔊 ENG ▾  │                        │ Parental        │
│ 💬 Off ▾  │                        │ Last watched    │
│          │                        │ Play count      │
│          │                        │ Added date      │
│          │                        │ External links  │
└──────────┴────────────────────────┴─────────────────┘
│ Files section                                       │
│ Cast (horizontal scroll)                            │
│ Crew (horizontal scroll)                            │
│ More Like This (horizontal scroll)                  │
└─────────────────────────────────────────────────────┘
```

### Action Buttons
- ✓ In Library (green when active)
- 👁 Watched
- ▶ Play (amber, larger 48px)
- 🎬 Trailer (red, opens YouTube)
- ⋮ More options

### Playback Selectors (inline, single row)
```
🎬 4K HEVC ▾ | 🔊 ENG Atmos ▾ | 💬 Off ▾
```

### TV Detail Differences
- "Play Next Episode" instead of "Play"
- Watch progress bar: "5 of 17 episodes watched"
- Season selector tabs below hero
- Episodes as horizontal slider (not list)
- Info panel shows: Status (Continuing), Network, Seasons, Episodes, Next Episode date

---

## Interactions

### Play Triggers
1. Click poster (hover shows ▶ overlay)
2. Click Play button (amber)
3. Click file card
4. Click episode card

### Navigation
- Logo → Home
- Nav tabs → respective pages
- Media cards → Detail page (movie or TV)
- Back behavior: browser history

### Hover States
- Cards: scale(1.02), elevated shadow
- Buttons: background lighten
- Posters: play overlay appears

### Focus States (TV/Keyboard)
- 2px accent ring
- All interactive elements focusable
- Tab order: left-to-right, top-to-bottom

---

## Files Section (Movie Detail)
- Shows actual media files
- Card with thumbnail, codec badge
- Filename (truncated)
- Meta: codec • audio • file size
- Click to play that specific file

---

## Assets Required
- `outpost-banner-transparent.png` - Nav logo
- `logo-cream-512.png` - Icon only (for favicon, etc)

---

## Responsive Notes (for future)
- Columns collapse on smaller screens
- Scroll rows become swipeable
- Info panel moves below content on mobile
- Touch-friendly tap targets throughout
