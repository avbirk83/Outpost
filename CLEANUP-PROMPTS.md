# Outpost Cleanup Guide - Prompts for Claude Code

Copy the CLAUDE.md file to your repo root first, then work through these prompts in order.

---

## PHASE 1: DELETE DEAD CODE

### 1.1 Remove Duplicate TopBar
```
Delete /frontend/src/lib/components/TopBar.svelte

This is an unused duplicate. The app uses /frontend/src/lib/components/layout/Topbar.svelte (check +layout.svelte to confirm). 

First run: grep -r "TopBar" frontend/src --include="*.svelte" to verify no imports, then delete the file.
```

### 1.2 Audit Unused Components
```
Check if these components have any imports. If not, delete them:

1. /frontend/src/lib/components/Sidebar.svelte
2. /frontend/src/lib/components/CastRow.svelte  
3. /frontend/src/lib/components/MediaRow.svelte

Run for each: grep -r "import.*ComponentName" frontend/src

Delete any with zero imports.
```

### 1.3 Remove Commented Code
```
Search the entire frontend/src folder for large blocks of commented-out code (10+ consecutive lines starting with //) and remove them. Dead code should live in git history, not the codebase.
```

---

## PHASE 2: CONSOLIDATE DUPLICATES

### 2.1 Merge GlassPanel Components
```
Merge the two GlassPanel implementations:

SOURCE: /frontend/src/lib/components/GlassPanel.svelte (has variant prop)
TARGET: /frontend/src/lib/components/ui/GlassPanel.svelte (has sticky prop)

1. Update ui/GlassPanel.svelte to include both variant AND sticky props
2. Map variants to appropriate styles:
   - light: bg-glass
   - medium: bg-glass-hover  
   - heavy: bg-glass-focus
   - card: liquid-card styles
3. Find all imports of the root GlassPanel and update to ui/GlassPanel
4. Delete /frontend/src/lib/components/GlassPanel.svelte
```

### 2.2 Merge EpisodeCard Components
```
Merge the two EpisodeCard implementations:

SOURCE: /frontend/src/lib/components/EpisodeCard.svelte
TARGET: /frontend/src/lib/components/media/EpisodeCard.svelte

1. Add missing props from SOURCE to TARGET:
   - id (for linking)
   - episodeNumber (SOURCE uses this, TARGET uses "number")
   - overview
   - stillPath (SOURCE) vs imagePath (TARGET) - unify naming
   - runtime with formatRuntime
   - airDate with date formatting

2. Unify the image URL handling (one uses getImageUrl, other uses getTmdbImageUrl)

3. Update all imports to use media/EpisodeCard.svelte

4. Delete /frontend/src/lib/components/EpisodeCard.svelte
```

---

## PHASE 3: FIX COLOR/STYLE INCONSISTENCIES

### 3.1 Replace text-white
```
Search all .svelte files in frontend/src for "text-white" and replace with "text-text-primary" (for primary text) or "text-cream" (for accent text).

Exception: Keep text-white only inside SVG elements or where it's genuinely meant to be pure white against a colored background.
```

### 3.2 Replace bg-white opacity classes
```
Search all .svelte files for bg-white with opacity and replace:

- bg-white/5 → bg-glass or remove if redundant
- bg-white/6 → bg-glass
- bg-white/8 → bg-glass
- bg-white/10 → bg-glass-hover
- bg-white/12 → bg-glass-hover
- bg-white/15 → bg-glass-focus

These are defined in app.css as CSS variables.
```

### 3.3 Replace border-white opacity classes
```
Search all .svelte files for border-white with opacity and replace:

- border-white/5 → border-border-subtle
- border-white/8 → border-border-subtle
- border-white/10 → border-border-subtle
- border-white/12 → border-border-subtle
- border-white/15 → border-border-focus
- border-white/20 → border-border-focus
- border-white/25 → border-border-focus
```

### 3.4 Remove Inline rgba Styles
```
Search for inline style attributes containing "rgba(255" and replace with appropriate CSS classes:

- style="background: rgba(255, 255, 255, 0.06)" → class="bg-glass"
- style="background: rgba(255, 255, 255, 0.1)" → class="bg-glass-hover"

Remove the style attribute entirely and add the class.
```

---

## PHASE 4: COMPONENT EXTRACTION

### 4.1 Extract VideoPlayer Components
```
VideoPlayer.svelte is 2,348 lines. Split into manageable pieces:

Create /frontend/src/lib/components/player/ folder with:

1. PlayerControls.svelte (~150 lines)
   - Play/pause button
   - Volume slider and mute
   - Fullscreen toggle
   - Picture-in-picture toggle

2. ProgressSlider.svelte (~100 lines)
   - Seek bar
   - Time display (current / duration)
   - Buffered indicator
   - Chapter markers if applicable

3. TrackPicker.svelte (~150 lines)
   - Audio track selection dropdown
   - Subtitle track selection dropdown
   - Quality selection dropdown
   - Use the existing ui/Select component pattern

4. PlayerOverlay.svelte (~100 lines)
   - Title display
   - Back button
   - Skip intro button
   - Next episode button

Keep VideoPlayer.svelte as the main container that:
- Manages the video element
- Handles keyboard shortcuts
- Composes the above components
- Should be under 500 lines after extraction
```

### 4.2 Extract Settings Sections
```
settings/+page.svelte is 2,299 lines. Split into sections:

Option A - Sub-routes:
Create /frontend/src/routes/settings/[section]/+page.svelte for each:
- libraries
- downloads  
- quality
- integrations
- notifications
- system

Option B - Tab Components:
Create /frontend/src/lib/components/settings/ folder with:
- LibrarySettings.svelte
- DownloadSettings.svelte
- QualitySettings.svelte
- IntegrationSettings.svelte
- NotificationSettings.svelte
- SystemSettings.svelte

Keep settings/+page.svelte as navigation/tabs that lazy-loads each section.
```

---

## PHASE 5: API CLEANUP

### 5.1 Split api.ts
```
Split /frontend/src/lib/api.ts (2,150 lines) into domain modules:

Create /frontend/src/lib/api/ folder:

1. index.ts - Re-exports everything, keeps backward compatibility
2. client.ts - apiFetch helper, error handling, base types
3. auth.ts - login, logout, getMe, user management
4. library.ts - movies, shows, episodes, seasons, libraries
5. requests.ts - createRequest, getRequests, approveRequest
6. downloads.ts - download clients, queue, history
7. metadata.ts - TMDB search, images, trailers
8. system.ts - health, status, settings, tasks

Update all imports across the codebase. The index.ts re-export means most imports won't need to change if they import from '$lib/api'.
```

---

## PHASE 6: DATABASE CLEANUP

### 6.1 Audit Unused Tables
```
Check the database schema in /internal/database/database.go for tables that may not be fully wired up:

1. release_filters - Is this checked in scheduler.go before grabbing releases?
2. exclusions - Is IsMediaExcluded() called in the search flow?
3. delay_profiles - Is this checked before immediate grabs?
4. blocklist - Is IsReleaseBlocklisted() called in scheduler?

For each table, trace the flow from API → service → scheduler to ensure it's actually used. If not wired up, either wire it up or document it as TODO.
```

### 6.2 Add Missing Indexes
```
Review the database schema for missing indexes on frequently queried columns:

Likely candidates:
- movies: tmdb_id, imdb_id, library_id
- shows: tmdb_id, tvdb_id, library_id
- episodes: show_id, season_number, episode_number
- downloads: status, media_type, media_id
- requests: status, user_id, tmdb_id

Add CREATE INDEX statements to the schema initialization.
```

### 6.3 Fix Type Mismatches
```
Check for type inconsistencies between database and Go structs:

In database.go, Download.Size is int64
In acquisition/service.go, Size is *int64 (pointer)

Unify these - prefer non-pointer for required fields, pointer for nullable fields.
```

---

## PHASE 7: BACKEND INTEGRATION FIXES

### 7.1 Wire Up Release Filters
```
In /internal/scheduler/scheduler.go, before grabbing a release, check release filters:

1. Load active release_filters from database
2. For each search result, check if it matches any filter
3. Skip releases that match a "block" filter
4. Prefer releases that match a "prefer" filter

The filter checking logic may already exist - trace from the API endpoint to see if it's used.
```

### 7.2 Wire Up Exclusions
```
In the search flow, before searching for media, check exclusions:

1. Check IsMediaExcluded(mediaID, mediaType) 
2. Check IsIndexerExcludedForLibrary(indexerID, libraryID)

If excluded, skip the search entirely.
```

### 7.3 Fix searchAlternative Stub
```
In /internal/acquisition/service.go around line 433, searchAlternative_() is a stub that just logs.

Implement it to:
1. Get the failed download's media info
2. Search indexers for the same media
3. Filter out the failed release (by name or hash)
4. Grab the next best result based on quality profile
```

---

## PHASE 8: FINAL POLISH

### 8.1 Standardize Button Usage
```
Audit all button elements in the codebase and ensure they use consistent classes:

Primary actions: liquid-btn
Secondary actions: liquid-btn-sm or btn-ghost
Icon buttons in topbar: btn-icon-circle
Icon buttons in content: btn-icon-circle-sm
Hero/large icon buttons: btn-icon-glass-lg

Remove any one-off button classes that aren't in app.css.
```

### 8.2 Add Loading States
```
Ensure all async operations show loading states:

1. API calls should set a loading state
2. Buttons should show spinner when loading
3. Pages should show skeleton or spinner while loading initial data

Use the existing spinner-sm, spinner-md, spinner-lg classes from app.css.
```

### 8.3 Error Handling
```
Add consistent error handling:

1. API errors should show toast notifications
2. Failed loads should show error states with retry buttons
3. Form validation errors should show inline

Use the existing toast store: import { toast } from '$lib/stores/toast'
```

---

## Verification Checklist

After cleanup, verify:

- [ ] `npm run build` succeeds with no errors
- [ ] `npm run check` passes TypeScript/Svelte checks
- [ ] `go build ./...` compiles backend
- [ ] Home page loads and displays content
- [ ] Library page shows movies/shows
- [ ] Detail pages load with full info
- [ ] Settings page renders all sections
- [ ] Search works in topbar
- [ ] Video player plays content
- [ ] Requests can be created and approved
