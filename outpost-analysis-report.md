# Outpost Codebase Analysis Report

## Executive Summary

This analysis covers the Outpost self-hosted media server codebase focusing on cleanup, optimization, and UI consistency. The codebase shows solid architecture but has accumulated some technical debt through rapid development.

---

## 🔴 Critical Issues

### 1. Duplicate Components (DELETE THESE)

| Root Component | Duplicate | Action |
|----------------|-----------|--------|
| `components/TopBar.svelte` (669 lines) | `layout/Topbar.svelte` (711 lines) | **DELETE root TopBar** — layout/Topbar is used in +layout.svelte |
| `components/GlassPanel.svelte` | `ui/GlassPanel.svelte` | **MERGE** — Different implementations, need to unify |
| `components/EpisodeCard.svelte` (108 lines) | `media/EpisodeCard.svelte` (97 lines) | **MERGE** — Similar purpose, different APIs |

**Impact**: 1,400+ lines of duplicate code, import confusion, inconsistent styling.

### 2. Giant Files Needing Decomposition

| File | Lines | Problem | Recommendation |
|------|-------|---------|----------------|
| `VideoPlayer.svelte` | 2,348 | Monolithic player logic | Extract into: `PlayerControls`, `PlayerOverlay`, `SubtitleSelector`, `QualitySelector`, `PlaybackLogic` |
| `settings/+page.svelte` | 2,299 | All settings in one file | Extract into: `LibrarySettings`, `DownloadSettings`, `QualitySettings`, `IntegrationSettings`, `UserSettings` |
| `api.ts` | 2,150 | All API calls in one file | Split by domain: `library.api.ts`, `downloads.api.ts`, `metadata.api.ts`, `auth.api.ts` |
| `library/+page.svelte` | 977 | Complex view logic | Extract filters and grid into shared components |

### 3. Inconsistent Component Usage

The codebase has components that aren't being reused consistently:

**Example — Buttons:**
```
btn-icon-glass-lg (in +page.svelte)
btn-icon-circle (in layout/Topbar.svelte)
liquid-btn-icon (in app.css)
btn-icon-circle-sm (in layout/Topbar.svelte)
```

**Recommendation**: Create a unified Button component with variants.

---

## 🟡 Moderate Issues

### 4. Unused Components

Components that appear to have no imports:
- `components/Sidebar.svelte` (177 lines) — No visible usage in routes
- `components/CastRow.svelte` — Possibly replaced by `CastCard`
- `components/MediaRow.svelte` — May be deprecated

**Action**: Audit usage with `grep -r "import.*ComponentName"` and remove dead code.

### 5. CSS Utility Duplication

`app.css` has multiple similar utilities that could be consolidated:

```css
/* These are essentially the same with minor variations */
@utility liquid-btn { ... }
@utility liquid-btn-sm { ... }
@utility btn-primary { ... }
@utility btn-glass { ... }
```

**Recommendation**: Define a single button utility with size/variant modifiers.

### 6. Inline Styles in Templates

Several components use inline styles instead of CSS classes:
- `style="background: rgba(255, 255, 255, 0.06);"` appears in multiple places
- `style="width: calc(100% - 8px);"` hardcoded values

**Recommendation**: Convert to CSS utilities or component props.

---

## 🟢 Backend Status (Good!)

The backend has improved since the last review:

✅ **Parser consolidation** — `quality/scoring.go` now imports `parser.ParsedRelease` instead of having its own duplicate parser.

✅ **Quality system** — Clean implementation with `ScoreRelease()`, `ComputeQualityTier()`, and custom format matching.

⚠️ **Still need to verify**:
- Scheduler integration with release filters
- Exclusions checking in search flow
- Failed download retry logic

---

## Recommended Component Architecture

### Proposed Structure

```
src/lib/components/
├── ui/                      # Base primitives (no business logic)
│   ├── Button.svelte        # All button variants
│   ├── Card.svelte          # Base card container
│   ├── Badge.svelte         # ✅ EXISTS
│   ├── GlassPanel.svelte    # ✅ EXISTS (consolidate)
│   ├── Select.svelte        # ✅ EXISTS
│   ├── ProgressBar.svelte   # ✅ EXISTS
│   ├── Input.svelte         # NEW - Form input
│   ├── Toggle.svelte        # NEW - Toggle switch
│   └── Spinner.svelte       # NEW - Loading indicator
│
├── layout/                  # App shell components
│   ├── Topbar.svelte        # ✅ EXISTS (keep this one)
│   ├── StatusBar.svelte     # ✅ EXISTS
│   └── PageHeader.svelte    # NEW - Consistent page titles
│
├── media/                   # Media-specific components
│   ├── MediaCard.svelte     # ✅ EXISTS
│   ├── EpisodeCard.svelte   # ✅ EXISTS (consolidate)
│   ├── CastCard.svelte      # ✅ EXISTS
│   ├── FileCard.svelte      # ✅ EXISTS
│   ├── PosterGrid.svelte    # NEW - Grid of posters
│   └── HeroCarousel.svelte  # NEW - Extract from +page.svelte
│
├── player/                  # Video player components (extract from VideoPlayer)
│   ├── VideoPlayer.svelte   # Main player container
│   ├── PlayerControls.svelte
│   ├── SubtitlePicker.svelte
│   ├── QualityPicker.svelte
│   └── ProgressSlider.svelte
│
├── modals/                  # Modal dialogs
│   ├── RequestModal.svelte  # ✅ EXISTS
│   ├── TrailerModal.svelte  # ✅ EXISTS
│   ├── PersonModal.svelte   # ✅ EXISTS
│   ├── ApprovalModal.svelte # ✅ EXISTS
│   └── ConfirmModal.svelte  # NEW - Generic confirm dialog
│
└── containers/              # Layout containers
    └── ScrollSection.svelte # ✅ EXISTS
```

---

## Cleanup Action Plan

### Phase 1: Remove Dead Code (Day 1)

1. **Delete** `/components/TopBar.svelte` — Unused duplicate
2. **Audit and remove** unused components with no imports
3. **Remove** any commented-out code blocks

### Phase 2: Consolidate Duplicates (Day 2-3)

1. **Merge GlassPanel implementations**
   - Keep `ui/GlassPanel.svelte`
   - Add `variant` prop: `'light' | 'medium' | 'heavy' | 'card'`
   - Update all imports

2. **Merge EpisodeCard implementations**
   - Keep `media/EpisodeCard.svelte`
   - Add missing props from root version
   - Update all imports

### Phase 3: Create Shared Components (Day 4-7)

1. **Create `ui/Button.svelte`**
   ```svelte
   <script lang="ts">
     interface Props {
       variant?: 'primary' | 'ghost' | 'icon' | 'danger';
       size?: 'sm' | 'md' | 'lg';
       href?: string;
       disabled?: boolean;
       onclick?: () => void;
       children: Snippet;
     }
   </script>
   ```

2. **Create `ui/Input.svelte`**
3. **Create `layout/PageHeader.svelte`**

### Phase 4: Decompose Large Files (Week 2)

1. **Split VideoPlayer.svelte** into player/ subfolder
2. **Split settings page** into sub-routes or components
3. **Split api.ts** by domain

---

## UI Consistency Checklist

### Current State

| Element | Classes Used | Should Be |
|---------|--------------|-----------|
| Primary Button | `btn-primary`, `liquid-btn`, `btn-glass` | Single `Button variant="primary"` |
| Icon Button | `btn-icon-circle`, `btn-icon-glass-lg`, `liquid-btn-icon` | Single `Button variant="icon"` |
| Cards | `bg-glass`, `glass`, `bg-bg-card/50` | `GlassPanel variant="card"` |
| Inputs | `liquid-input`, `form-input` | Single `Input` component |
| Progress | Custom inline, `ProgressBar` | `ProgressBar` everywhere |

### Design Token Consistency

The design system in `app.css` is well-defined but inconsistently applied:

```css
/* These should be used everywhere */
--color-cream: #F5E6C8;
--color-amber: #E8A849;
--color-text-primary: #F5E6C8;
--color-text-secondary: rgba(245, 230, 200, 0.7);
--color-text-muted: rgba(245, 230, 200, 0.5);
--color-glass: rgba(255, 255, 255, 0.06);
```

**Found inconsistencies:**
- `bg-white/8` instead of `bg-glass`
- `text-white` instead of `text-text-primary`
- `border-white/10` instead of `border-border-subtle`

---

## For Claude Code: CLAUDE.md Recommendations

Add this to your repo root to guide future development:

```markdown
# CLAUDE.md - Development Guidelines

## Component Rules
1. ALWAYS check /src/lib/components/ui/ before creating new base components
2. ALWAYS check /src/lib/components/ before creating any component
3. Use existing shared components: Button, GlassPanel, Select, Badge
4. Never create duplicate button styles - use Button component

## Styling Rules
1. Use CSS variables from app.css theme
2. Use utility classes (glass, liquid-btn, etc.) over inline styles
3. Colors: cream (#F5E6C8), amber (#E8A849), not white
4. Text: text-text-primary, text-text-secondary, text-text-muted

## File Size Limits
- Components: Max 300 lines (decompose if larger)
- Routes: Max 500 lines (extract logic to components)

## Import Paths
- UI primitives: from '$lib/components/ui/Component.svelte'
- Media components: from '$lib/components/media/Component.svelte'
- Modals: from '$lib/components/modals/Component.svelte'
```

---

## Summary Statistics

| Metric | Current | Target |
|--------|---------|--------|
| Total Svelte lines | 17,072 | ~12,000 |
| Duplicate code | ~1,400 lines | 0 |
| Files > 500 lines | 6 | 0 |
| Unused components | ~5 | 0 |
| Button variants | 8+ | 1 component |

**Estimated cleanup effort**: 2-3 weeks for full refactor, or incremental over sprints.
