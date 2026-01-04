# Outpost Component Extraction Spec

## Overview

The movie and TV detail pages have ~800 lines each with massive duplication. This document specifies components to extract that will:
1. Reduce each page to ~200 lines
2. Enable consistent styling
3. Make mobile/TV apps easier to build

---

## Priority 1: ScrollableRow (eliminates 6 duplicate implementations)

Both pages have 3 identical scroll implementations (cast, crew, recommendations).

```svelte
<!-- ScrollableRow.svelte -->
<script lang="ts">
  import type { Snippet } from 'svelte';

  interface Props {
    title: string;
    children: Snippet;
    class?: string;
  }

  let { title, children, class: className = '' }: Props = $props();

  let container: HTMLElement;
  let canScrollLeft = $state(false);
  let canScrollRight = $state(true);

  function updateScrollState() {
    if (!container) return;
    canScrollLeft = container.scrollLeft > 0;
    canScrollRight = container.scrollLeft < container.scrollWidth - container.clientWidth - 10;
  }

  function scroll(direction: 'left' | 'right') {
    if (!container) return;
    container.scrollBy({
      left: direction === 'left' ? -300 : 300,
      behavior: 'smooth'
    });
    setTimeout(updateScrollState, 350);
  }
</script>

<section class="px-6 {className}">
  <div class="flex items-center justify-between mb-3">
    <h2 class="text-lg font-semibold text-text-primary">{title}</h2>
    <div class="flex gap-1">
      <button
        onclick={() => scroll('left')}
        disabled={!canScrollLeft}
        class="p-1.5 rounded-full bg-white/10 hover:bg-white/20 text-white transition-colors disabled:opacity-30"
      >
        <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 19l-7-7 7-7" />
        </svg>
      </button>
      <button
        onclick={() => scroll('right')}
        disabled={!canScrollRight}
        class="p-1.5 rounded-full bg-white/10 hover:bg-white/20 text-white transition-colors disabled:opacity-30"
      >
        <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7" />
        </svg>
      </button>
    </div>
  </div>
  <div
    bind:this={container}
    onscroll={updateScrollState}
    class="flex gap-4 overflow-x-auto pb-2 scrollbar-thin"
  >
    {@render children()}
  </div>
</section>
```

**Usage:**
```svelte
<ScrollableRow title="Cast">
  {#each cast as actor}
    <PersonCard person={actor} onclick={() => handlePersonClick(actor)} />
  {/each}
</ScrollableRow>
```

---

## Priority 2: PersonCard (used in cast AND crew rows)

```svelte
<!-- PersonCard.svelte -->
<script lang="ts">
  import { getTmdbImageUrl } from '$lib/api';

  interface Props {
    name: string;
    role: string;  // character for cast, job for crew
    profilePath?: string | null;
    onclick?: () => void;
  }

  let { name, role, profilePath, onclick }: Props = $props();
</script>

<button
  {onclick}
  class="flex-shrink-0 w-28 text-center cursor-pointer group"
>
  <div class="w-28 h-28 rounded-full bg-bg-elevated overflow-hidden mx-auto ring-2 ring-white/10 group-hover:ring-white/30 transition-all">
    {#if profilePath}
      <img
        src={getTmdbImageUrl(profilePath, 'w185')}
        alt={name}
        class="w-full h-full object-cover"
      />
    {:else}
      <div class="w-full h-full flex items-center justify-center text-3xl text-text-muted bg-gradient-to-br from-bg-card to-bg-elevated">
        {name.charAt(0)}
      </div>
    {/if}
  </div>
  <p class="mt-2 text-sm font-medium text-text-primary truncate group-hover:text-white transition-colors">{name}</p>
  <p class="text-xs text-text-muted truncate">{role}</p>
</button>
```

---

## Priority 3: InfoPanel (the right sidebar)

```svelte
<!-- InfoPanel.svelte -->
<script lang="ts">
  import ExternalLinks from './ExternalLinks.svelte';
  import { formatRuntime, getLanguageName, getCountryName, getCountryFlag, formatMoney, getStatusColor } from '$lib/utils';

  interface Props {
    // Common fields
    status?: string;
    year?: number;
    runtime?: number;  // minutes for movies, total for shows
    language?: string;
    country?: string;
    contentRating?: string;
    addedAt?: string;
    // Movie-specific
    budget?: number;
    revenue?: number;
    // Show-specific
    network?: string;
    seasonCount?: number;
    episodeCount?: number;
    nextAirDate?: string;
    creators?: string[];
    // Links
    tmdbId?: number;
    imdbId?: string;
    mediaType: 'movie' | 'tv';
  }

  let props: Props = $props();
</script>

<div class="liquid-card p-4 space-y-2.5 text-sm">
  <!-- Status -->
  <div class="flex justify-between">
    <span class="text-text-muted">Status</span>
    <span class="{getStatusColor(props.status)} font-medium">{props.status || 'Unknown'}</span>
  </div>

  <!-- Show-specific: Network -->
  {#if props.network}
    <div class="flex justify-between">
      <span class="text-text-muted">Network</span>
      <span>{props.network}</span>
    </div>
  {/if}

  <!-- Show-specific: Creators -->
  {#if props.creators?.length}
    <div class="flex justify-between">
      <span class="text-text-muted">Created By</span>
      <span class="text-right max-w-[180px] truncate">{props.creators.join(', ')}</span>
    </div>
  {/if}

  <!-- Year -->
  {#if props.year}
    <div class="flex justify-between">
      <span class="text-text-muted">{props.mediaType === 'movie' ? 'Released' : 'Year'}</span>
      <span>{props.year}</span>
    </div>
  {/if}

  <!-- Runtime -->
  {#if props.runtime}
    <div class="flex justify-between">
      <span class="text-text-muted">Runtime</span>
      <span>{formatRuntime(props.runtime)}</span>
    </div>
  {/if}

  <!-- Show-specific: Seasons/Episodes -->
  {#if props.seasonCount !== undefined}
    <div class="flex justify-between">
      <span class="text-text-muted">Seasons</span>
      <span>{props.seasonCount}</span>
    </div>
  {/if}
  {#if props.episodeCount !== undefined}
    <div class="flex justify-between">
      <span class="text-text-muted">Episodes</span>
      <span>{props.episodeCount}</span>
    </div>
  {/if}

  <div class="border-t border-white/10 my-2"></div>

  <!-- Movie-specific: Budget/Revenue -->
  {#if props.budget}
    <div class="flex justify-between">
      <span class="text-text-muted">Budget</span>
      <span>{formatMoney(props.budget)}</span>
    </div>
  {/if}
  {#if props.revenue}
    <div class="flex justify-between">
      <span class="text-text-muted">Revenue</span>
      <span class="{props.revenue > (props.budget || 0) ? 'text-green-400' : 'text-red-400'}">{formatMoney(props.revenue)}</span>
    </div>
  {/if}

  <!-- Language -->
  {#if props.language}
    <div class="flex justify-between">
      <span class="text-text-muted">Language</span>
      <span>{getLanguageName(props.language)}</span>
    </div>
  {/if}

  <!-- Country -->
  {#if props.country}
    <div class="flex justify-between">
      <span class="text-text-muted">Country</span>
      <span>{getCountryFlag(props.country)} {getCountryName(props.country)}</span>
    </div>
  {/if}

  <div class="border-t border-white/10 my-2"></div>

  <!-- Content Rating -->
  {#if props.contentRating}
    <div class="flex justify-between items-center">
      <span class="text-text-muted">Parental</span>
      <span class="flex items-center gap-2">
        <span class="px-1.5 py-0.5 bg-white/10 rounded text-xs font-medium">{props.contentRating}</span>
        {#if props.imdbId}
          <a href="https://www.imdb.com/title/{props.imdbId}/parentalguide" target="_blank" class="text-sky-400 hover:text-sky-300 text-xs">
            View ↗
          </a>
        {/if}
      </span>
    </div>
  {/if}

  <!-- Added Date -->
  {#if props.addedAt}
    <div class="flex justify-between">
      <span class="text-text-muted">Added</span>
      <span>{new Date(props.addedAt).toLocaleDateString('en-US', { month: 'short', day: 'numeric', year: 'numeric' })}</span>
    </div>
  {/if}

  <div class="border-t border-white/10 my-2"></div>

  <!-- External Links -->
  <ExternalLinks
    tmdbId={props.tmdbId}
    imdbId={props.imdbId}
    mediaType={props.mediaType}
  />
</div>
```

---

## Priority 4: RatingsRow

```svelte
<!-- RatingsRow.svelte -->
<script lang="ts">
  interface Props {
    tmdbId?: number;
    tmdbRating?: number;
    rtRating?: number;   // future
    metacriticRating?: number;  // future
    mediaType: 'movie' | 'tv';
  }

  let { tmdbId, tmdbRating, rtRating, metacriticRating, mediaType }: Props = $props();
</script>

<div class="p-3 flex justify-around items-center border-t border-white/10">
  {#if tmdbRating}
    <a href="https://www.themoviedb.org/{mediaType}/{tmdbId}" target="_blank" class="flex items-center gap-1.5 hover:opacity-80 transition-opacity" title="TMDB Rating">
      <img src="/icons/tmdb.svg" alt="TMDB" class="w-6 h-6 rounded" />
      <span class="text-base font-bold text-white">{tmdbRating.toFixed(1)}</span>
    </a>
  {/if}
  <div class="flex items-center gap-1.5 opacity-40" title="Rotten Tomatoes (coming soon)">
    <img src="/icons/rottentomatoes.svg" alt="Rotten Tomatoes" class="w-6 h-6" />
    <span class="text-base font-bold">{rtRating?.toString() || '--'}</span>
  </div>
  <div class="flex items-center gap-1.5 opacity-40" title="Metacritic (coming soon)">
    <img src="/icons/metacritic.svg" alt="Metacritic" class="w-6 h-6 rounded" />
    <span class="text-base font-bold">{metacriticRating?.toString() || '--'}</span>
  </div>
</div>
```

---

## Priority 5: ExternalLinks

```svelte
<!-- ExternalLinks.svelte -->
<script lang="ts">
  interface Props {
    tmdbId?: number;
    imdbId?: string;
    mediaType: 'movie' | 'tv';
  }

  let { tmdbId, imdbId, mediaType }: Props = $props();
</script>

<div class="flex justify-center gap-3">
  {#if tmdbId}
    <a href="https://www.themoviedb.org/{mediaType}/{tmdbId}" target="_blank"
       class="w-9 h-9 rounded-lg bg-white/10 hover:bg-white/20 flex items-center justify-center transition-colors overflow-hidden" title="View on TMDB">
      <img src="/icons/tmdb.svg" alt="TMDB" class="w-7 h-7" />
    </a>
  {/if}
  {#if imdbId}
    <a href="https://www.imdb.com/title/{imdbId}" target="_blank"
       class="w-9 h-9 rounded-lg bg-white/10 hover:bg-white/20 flex items-center justify-center transition-colors overflow-hidden" title="View on IMDb">
      <img src="/icons/imdb.svg" alt="IMDb" class="w-7 h-7" />
    </a>
  {/if}
  <a href="https://trakt.tv/search/{imdbId ? 'imdb/' + imdbId : 'tmdb/' + tmdbId + '?id_type=' + (mediaType === 'movie' ? 'movie' : 'show')}" target="_blank"
     class="w-9 h-9 rounded-lg bg-white/10 hover:bg-white/20 flex items-center justify-center transition-colors overflow-hidden" title="View on Trakt">
    <img src="/icons/trakt.svg" alt="Trakt" class="w-7 h-7" />
  </a>
  {#if mediaType === 'movie'}
    <a href="https://letterboxd.com/tmdb/{tmdbId}" target="_blank"
       class="w-9 h-9 rounded-lg bg-white/10 hover:bg-white/20 flex items-center justify-center transition-colors overflow-hidden" title="View on Letterboxd">
      <img src="/icons/letterboxd.svg" alt="Letterboxd" class="w-7 h-7" />
    </a>
  {/if}
</div>
```

---

## Priority 6: Utility Functions to Centralize

Add to `$lib/utils/formatters.ts`:

```typescript
// Already exists: formatRuntime

export function parseGenres(g?: string): string[] {
  if (!g) return [];
  try { return JSON.parse(g); } catch { return []; }
}

export function parseCast(c?: string): Array<{ name: string; character: string; profile_path?: string; id?: number }> {
  if (!c) return [];
  try { return JSON.parse(c); } catch { return []; }
}

export function parseCrew(c?: string): Array<{ name: string; job: string; department: string; profile_path?: string; id?: number }> {
  if (!c) return [];
  try { return JSON.parse(c); } catch { return []; }
}

export function formatMoney(amount?: number): string {
  if (!amount || amount === 0) return '-';
  return new Intl.NumberFormat('en-US', { style: 'currency', currency: 'USD', maximumFractionDigits: 0 }).format(amount);
}

export function getLanguageName(code?: string): string {
  if (!code || code === 'und') return 'Unknown';
  try {
    return new Intl.DisplayNames(['en'], { type: 'language' }).of(code) || code;
  } catch { return code; }
}

export function getCountryName(code?: string): string {
  if (!code) return '';
  try {
    return new Intl.DisplayNames(['en'], { type: 'region' }).of(code.toUpperCase()) || code;
  } catch { return code; }
}

export function getCountryFlag(code?: string): string {
  if (!code || code.length !== 2) return '';
  return code.toUpperCase().split('').map(c => String.fromCodePoint(127397 + c.charCodeAt(0))).join('');
}

export function getStatusColor(status?: string): string {
  switch (status?.toLowerCase()) {
    case 'released':
    case 'returning series':
    case 'in production':
      return 'text-green-400';
    case 'ended':
    case 'post production':
      return 'text-yellow-400';
    case 'canceled':
      return 'text-red-400';
    default:
      return 'text-text-secondary';
  }
}

export function formatResolution(width?: number, height?: number): string {
  if (!width && !height) return '';
  const w = width || 0;
  const h = height || 0;
  if (w >= 3840 || h >= 2160) return '4K';
  if (w >= 1920 || h >= 1080) return '1080p';
  if (w >= 1280 || h >= 720) return '720p';
  if (h > 0) return `${h}p`;
  return '';
}

export function formatAudioChannels(channels?: number): string {
  if (!channels) return '';
  if (channels >= 8) return '7.1';
  if (channels >= 6) return '5.1';
  if (channels === 2) return 'Stereo';
  return `${channels}ch`;
}
```

---

## Refactoring Order

1. **Add utility functions** to `$lib/utils/formatters.ts` and export from `$lib/utils/index.ts`
2. **Create ScrollableRow** — highest impact, eliminates most duplication
3. **Create PersonCard** — used by both cast and crew
4. **Create ExternalLinks** — simple extraction
5. **Create RatingsRow** — simple extraction
6. **Create InfoPanel** — combines movie/TV info with conditional fields
7. **Update CastRow** to use ScrollableRow + PersonCard internally
8. **Refactor movies/[id]** to use new components
9. **Refactor tv/[id]** to use new components
10. **Delete unused DetailHero** or update pages to use it

---

## After Refactoring: movies/[id]+page.svelte

Should look approximately like this (~200 lines vs current ~800):

```svelte
<script lang="ts">
  import { onMount } from 'svelte';
  import { page } from '$app/stores';
  import { getMovie, getMediaInfo, getMovieSuggestions, /* etc */ } from '$lib/api';
  import { parseGenres, parseCast, parseCrew } from '$lib/utils';
  import ScrollableRow from '$lib/components/ScrollableRow.svelte';
  import PersonCard from '$lib/components/PersonCard.svelte';
  import InfoPanel from '$lib/components/InfoPanel.svelte';
  import RatingsRow from '$lib/components/RatingsRow.svelte';
  import PosterCard from '$lib/components/PosterCard.svelte';
  import IconButton from '$lib/components/IconButton.svelte';
  import PersonModal from '$lib/components/PersonModal.svelte';
  import TrailerModal from '$lib/components/TrailerModal.svelte';

  // ... state and handlers (much shorter now) ...
</script>

{#if movie}
  <div class="space-y-6 -mt-22 -mx-6">
    <!-- Hero -->
    <section class="relative min-h-[500px]">
      <!-- backdrop + gradients -->
      <div class="relative z-10 px-6 pt-32 pb-8 flex gap-6">
        <!-- Poster with ratings -->
        <div class="flex-shrink-0 w-64 mt-8">
          <div class="liquid-card overflow-hidden">
            <PosterCard ... />
            <RatingsRow tmdbId={movie.tmdbId} tmdbRating={movie.rating} mediaType="movie" />
          </div>
        </div>

        <!-- Center content -->
        <div class="flex-1 min-w-0 py-4">
          <h1>...</h1>
          <p>...</p>
          <div class="flex items-center gap-2">
            <IconButton ... />
          </div>
        </div>

        <!-- Info Panel -->
        <InfoPanel
          status={movie.status}
          year={movie.year}
          runtime={movie.runtime}
          budget={movie.budget}
          revenue={movie.revenue}
          language={movie.originalLanguage}
          country={movie.country}
          contentRating={movie.contentRating}
          addedAt={movie.addedAt}
          tmdbId={movie.tmdbId}
          imdbId={movie.imdbId}
          mediaType="movie"
        />
      </div>
    </section>

    <!-- Cast -->
    <ScrollableRow title="Cast">
      {#each parseCast(movie.cast) as actor}
        <PersonCard
          name={actor.name}
          role={actor.character}
          profilePath={actor.profile_path}
          onclick={() => handlePersonClick(actor)}
        />
      {/each}
    </ScrollableRow>

    <!-- Crew -->
    <ScrollableRow title="Crew">
      {#each parseCrew(movie.crew) as member}
        <PersonCard
          name={member.name}
          role={member.job}
          profilePath={member.profile_path}
          onclick={() => handlePersonClick(member)}
        />
      {/each}
    </ScrollableRow>

    <!-- Recommendations -->
    <ScrollableRow title="More Like This">
      {#each recommendations as rec}
        <a href="/discover/movie/{rec.id}" class="flex-shrink-0 w-32 group">
          ...
        </a>
      {/each}
    </ScrollableRow>
  </div>
{/if}
```
