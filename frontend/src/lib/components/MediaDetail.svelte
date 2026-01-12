<script lang="ts">
	import type { Snippet } from 'svelte';
	import { goto } from '$app/navigation';
	import { getTmdbImageUrl, getImageUrl } from '$lib/api';
	import { formatRuntime, getYear, formatMoneyFull, getLanguageName, getCountryName, getCountryFlag, getStatusColor } from '$lib/utils';
	import PersonModal from './PersonModal.svelte';
	import TrailerModal from './TrailerModal.svelte';
	import RatingsRow from './RatingsRow.svelte';
	import ExternalLinks from './ExternalLinks.svelte';
	import PersonCard from './PersonCard.svelte';
	import ScrollableRow from './ScrollableRow.svelte';
	import PosterCard from './PosterCard.svelte';
	import MediaCard from './media/MediaCard.svelte';

	interface CastMember {
		id?: number;
		name: string;
		character: string;
		photo?: string | null;
		profile_path?: string | null;
	}

	interface CrewMember {
		id?: number;
		name: string;
		job: string;
		photo?: string | null;
		profile_path?: string | null;
	}

	interface Recommendation {
		id: number;
		title: string;
		posterPath?: string | null;
		poster_path?: string | null;
		releaseDate?: string;
		release_date?: string;
		rating?: number;
		vote_average?: number;
		runtime?: number;
		contentRating?: string;
	}

	interface Props {
		// Core data
		title: string;
		year?: number | string;
		overview?: string;
		tagline?: string;
		posterPath?: string;
		backdropPath?: string;
		genres?: string[];

		// IDs
		tmdbId?: number;
		imdbId?: string;

		// Type and source
		mediaType: 'movie' | 'tv';
		source: 'library' | 'discover';

		// Movie-specific
		runtime?: number;
		director?: string;
		budget?: number;
		revenue?: number;
		theatricalRelease?: string;
		digitalRelease?: string;
		contentRating?: string;

		// TV-specific
		seasons?: number;
		episodes?: number;
		networks?: string[];
		status?: string;
		firstAirDate?: string;

		// Common
		originalLanguage?: string;
		country?: string;
		productionCountries?: string[];
		studios?: string;
		productionCompanies?: string[];

		// Ratings
		rating?: number;

		// Cast & Crew
		cast?: CastMember[];
		crew?: CrewMember[];

		// Recommendations
		recommendations?: Recommendation[];

		// Library-specific
		addedAt?: string;
		lastWatchedAt?: string;
		playCount?: number;

		// For images - library uses local, discover uses TMDB
		useLocalImages?: boolean;

		// Poster interactivity (for library pages)
		posterClickable?: boolean;
		onPosterClick?: () => void;

		// Trailer
		trailerKey?: string;
		trailersJson?: string;

		// Snippets for customization
		actionButtons?: Snippet;
		centerExtra?: Snippet;
		extraInfoRows?: Snippet;
		extraSections?: Snippet;
		footerContent?: Snippet;
	}

	let {
		title,
		year,
		overview,
		tagline,
		posterPath,
		backdropPath,
		genres = [],
		tmdbId,
		imdbId,
		mediaType,
		source,
		runtime,
		director,
		budget,
		revenue,
		theatricalRelease,
		digitalRelease,
		contentRating,
		seasons,
		episodes,
		networks = [],
		status,
		firstAirDate,
		originalLanguage,
		country,
		productionCountries = [],
		studios,
		productionCompanies = [],
		rating,
		cast = [],
		crew = [],
		recommendations = [],
		addedAt,
		lastWatchedAt,
		playCount,
		useLocalImages = false,
		posterClickable = false,
		onPosterClick,
		trailerKey,
		trailersJson,
		actionButtons,
		centerExtra,
		extraInfoRows,
		extraSections,
		footerContent
	}: Props = $props();

	// Person modal state
	let selectedPersonId = $state<number | null>(null);
	let selectedPersonName = $state<string>('');
	let showTrailerModal = $state(false);

	function handlePersonClick(person: { id?: number; name: string }) {
		if (person.id) {
			selectedPersonId = person.id;
			selectedPersonName = person.name;
		}
	}

	function closePersonModal() {
		selectedPersonId = null;
		selectedPersonName = '';
	}

	function getImageSrc(path: string | undefined, size: string = 'w500'): string {
		if (!path) return '';
		if (useLocalImages) {
			return getImageUrl(path);
		}
		return getTmdbImageUrl(path, size);
	}

	function getBackdropSrc(path: string | undefined): string {
		if (!path) return '';
		if (useLocalImages) {
			return getImageUrl(path);
		}
		return getTmdbImageUrl(path, 'w1280');
	}

	// Get profile path from cast/crew (handles both formats)
	function getProfilePath(member: CastMember | CrewMember): string | undefined {
		return member.photo || member.profile_path || undefined;
	}

	// Normalize recommendations data
	function getRecPosterPath(rec: Recommendation): string | undefined {
		const path = rec.posterPath || rec.poster_path;
		return path ? `https://image.tmdb.org/t/p/w342${path}` : undefined;
	}

	function getRecReleaseYear(rec: Recommendation): string {
		const date = rec.releaseDate || rec.release_date;
		return date?.split('-')[0] || '';
	}

	function getRecRating(rec: Recommendation): number {
		return rec.rating || rec.vote_average || 0;
	}

	// Computed values
	const displayYear = $derived(year || (firstAirDate ? getYear(firstAirDate) : ''));
	const displayStatus = $derived(status || (mediaType === 'movie' ? 'Released' : 'Unknown'));
	const displayCountry = $derived(country || (productionCountries.length > 0 ? productionCountries[0] : ''));
	const displayStudios = $derived(() => {
		if (studios) {
			try {
				return JSON.parse(studios).slice(0, 2).join(', ');
			} catch {
				return studios;
			}
		}
		if (productionCompanies.length > 0) {
			return productionCompanies.slice(0, 2).join(', ');
		}
		return '';
	});

	// Determine recommendation link path
	const recLinkPrefix = $derived(source === 'library' ? '/explore' : '/explore');
	const recMediaPath = $derived(mediaType === 'movie' ? 'movie' : 'show');
</script>

<div class="space-y-6 -mt-22 -mx-6">
	<!-- Hero Section -->
	<section class="relative min-h-[500px]">
		<!-- Backdrop -->
		{#if backdropPath}
			<img
				src={getBackdropSrc(backdropPath)}
				alt=""
				class="absolute inset-0 w-full h-full object-cover pointer-events-none"
				style="object-position: center 25%;"
				draggable="false"
			/>
			<div class="absolute inset-0 bg-gradient-to-r from-bg-primary via-bg-primary/80 to-transparent pointer-events-none"></div>
			<div class="absolute inset-0 bg-gradient-to-t from-bg-primary via-transparent to-bg-primary/30 pointer-events-none"></div>
		{/if}

		<!-- Hero Content: 3 columns -->
		<div class="relative z-10 px-[60px] pt-32 pb-8 flex gap-10">
			<!-- LEFT: Poster Card with Action Buttons -->
			<div class="flex-shrink-0 w-80">
				<div class="liquid-card">
					<!-- svelte-ignore a11y_no_static_element_interactions -->
					<div
						class="relative aspect-[2/3] bg-bg-card overflow-hidden rounded-t-xl {posterClickable ? 'cursor-pointer' : ''}"
						onclick={posterClickable && onPosterClick ? onPosterClick : undefined}
						onkeydown={posterClickable && onPosterClick ? (e) => e.key === 'Enter' && onPosterClick() : undefined}
						role={posterClickable ? 'button' : undefined}
						tabindex={posterClickable ? 0 : undefined}
					>
						{#if posterPath}
							<img
								src={getImageSrc(posterPath)}
								alt={title}
								class="w-full h-full object-cover"
							/>
						{:else}
							<div class="w-full h-full flex items-center justify-center bg-bg-elevated">
								<svg class="w-16 h-16 text-text-muted" fill="none" stroke="currentColor" viewBox="0 0 24 24">
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="1" d="M7 4v16M17 4v16M3 8h4m10 0h4M3 12h18M3 16h4m10 0h4" />
								</svg>
							</div>
						{/if}
						<!-- Play overlay on hover (for library pages) -->
						{#if posterClickable}
							<div class="absolute inset-0 bg-black/50 flex items-center justify-center opacity-0 hover:opacity-100 transition-opacity">
								<div class="w-14 h-14 rounded-full bg-white/20 backdrop-blur flex items-center justify-center">
									<svg class="w-7 h-7 text-white ml-1" fill="currentColor" viewBox="0 0 24 24">
										<path d="M8 5v14l11-7z" />
									</svg>
								</div>
							</div>
						{/if}
					</div>
					<!-- Action Buttons Row -->
					<div class="p-3 flex justify-center gap-1.5 border-t border-border-subtle">
						{#if actionButtons}
							{@render actionButtons()}
						{/if}
					</div>
				</div>
			</div>

			<!-- CENTER: Title, Tags, Overview -->
			<div class="flex-1 min-w-0">
				<!-- Title -->
				<h1 class="text-4xl md:text-5xl font-bold text-text-primary mb-2">
					{title}
					{#if displayYear}
						<span class="text-text-secondary font-normal">({displayYear})</span>
					{/if}
				</h1>

				<!-- Meta line -->
				<div class="flex items-center gap-2 text-text-secondary mb-4">
					{#if contentRating}
						<span class="px-2 py-0.5 border border-white/30 text-xs font-medium">{contentRating}</span>
					{/if}
					{#if runtime}
						<span>{formatRuntime(runtime)}</span>
					{/if}
					{#if mediaType === 'tv' && seasons}
						<span>{seasons} season{seasons !== 1 ? 's' : ''}</span>
					{/if}
					{#if genres.length > 0}
						<span>•</span>
						<span>{genres.join(', ')}</span>
					{/if}
				</div>

				<!-- Tags (clickable pills) -->
				{#if genres.length > 0}
					<div class="flex flex-wrap gap-2 mb-4">
						{#each genres as genre}
							<button onclick={() => goto(`/explore?tab=${mediaType === 'movie' ? 'movies' : 'shows'}&genre=${encodeURIComponent(genre)}`)} class="px-3 py-1.5 text-xs font-medium rounded-full bg-glass border border-border-subtle text-text-primary hover:bg-glass-hover transition-all">
								{genre}
							</button>
						{/each}
					</div>
				{/if}

				<!-- Network (TV only) -->
				{#if mediaType === 'tv' && networks.length > 0}
					<p class="text-text-secondary mb-4">
						<span class="text-text-muted">Network:</span> {networks.join(', ')}
					</p>
				{/if}

				<!-- Tagline -->
				{#if tagline}
					<p class="text-text-secondary italic mb-4">"{tagline}"</p>
				{/if}

				<!-- Overview -->
				{#if overview}
					<p class="text-text-secondary leading-relaxed max-w-2xl mb-5">
						{overview}
					</p>
				{/if}

				<!-- Extra center content (e.g., playback selectors) -->
				{#if centerExtra}
					{@render centerExtra()}
				{/if}
			</div>

			<!-- RIGHT: Info Panel Card -->
			<div class="flex-shrink-0 w-80">
				<div class="liquid-card p-4 space-y-2.5 text-sm">
					<!-- Ratings Row -->
					{#if tmdbId}
						<RatingsRow
							{tmdbId}
							tmdbRating={rating}
							{mediaType}
						/>
						<div class="border-t border-border-subtle my-3"></div>
					{/if}

					<!-- Status -->
					<div class="flex justify-between">
						<span class="text-text-muted">Status</span>
						<span class="{getStatusColor(displayStatus)} font-medium">{displayStatus}</span>
					</div>

					<!-- Release/Air Date -->
					{#if displayYear}
						<div class="flex justify-between">
							<span class="text-text-muted">{mediaType === 'movie' ? 'Released' : 'First Aired'}</span>
							<span>{displayYear}</span>
						</div>
					{/if}

					<!-- Runtime (movies) -->
					{#if mediaType === 'movie' && runtime}
						<div class="flex justify-between">
							<span class="text-text-muted">Runtime</span>
							<span>{formatRuntime(runtime)}</span>
						</div>
					{/if}

					<!-- Seasons/Episodes (TV) -->
					{#if mediaType === 'tv'}
						{#if seasons}
							<div class="flex justify-between">
								<span class="text-text-muted">Seasons</span>
								<span>{seasons}</span>
							</div>
						{/if}
						{#if episodes}
							<div class="flex justify-between">
								<span class="text-text-muted">Episodes</span>
								<span>{episodes}</span>
							</div>
						{/if}
					{/if}

					<!-- Theatrical Release -->
					{#if theatricalRelease}
						<div class="flex justify-between">
							<span class="text-text-muted flex items-center gap-1.5">Theatrical</span>
							<span>{new Date(theatricalRelease).toLocaleDateString('en-US', { month: 'short', day: 'numeric', year: 'numeric' })}</span>
						</div>
					{/if}

					<!-- Digital Release -->
					{#if digitalRelease}
						<div class="flex justify-between">
							<span class="text-text-muted flex items-center gap-1.5">Digital</span>
							<span>{new Date(digitalRelease).toLocaleDateString('en-US', { month: 'short', day: 'numeric', year: 'numeric' })}</span>
						</div>
					{/if}

					<div class="border-t border-border-subtle my-2"></div>

					<!-- Director (movies) -->
					{#if director}
						<div class="flex justify-between">
							<span class="text-text-muted">Director</span>
							<span class="text-right">{director}</span>
						</div>
					{/if}

					<!-- Network (TV) -->
					{#if mediaType === 'tv' && networks.length > 0}
						<div class="flex justify-between">
							<span class="text-text-muted">Network</span>
							<span class="text-right">{networks[0]}</span>
						</div>
					{/if}

					<!-- Budget -->
					{#if budget}
						<div class="flex justify-between">
							<span class="text-text-muted">Budget</span>
							<span>{formatMoneyFull(budget)}</span>
						</div>
					{/if}

					<!-- Revenue -->
					{#if revenue}
						<div class="flex justify-between">
							<span class="text-text-muted">Revenue</span>
							<span class="{revenue > (budget || 0) ? 'text-green-400' : 'text-red-400'}">{formatMoneyFull(revenue)}</span>
						</div>
					{/if}

					{#if budget || revenue}
						<div class="border-t border-border-subtle my-2"></div>
					{/if}

					<!-- Language -->
					{#if originalLanguage}
						<div class="flex justify-between">
							<span class="text-text-muted">Language</span>
							<span>{getLanguageName(originalLanguage)}</span>
						</div>
					{/if}

					<!-- Country -->
					{#if displayCountry}
						<div class="flex justify-between items-center">
							<span class="text-text-muted">Country</span>
							<span class="flex items-center gap-1.5">
								<span class="text-base">{getCountryFlag(displayCountry)}</span>
								<span>{getCountryName(displayCountry)}</span>
							</span>
						</div>
					{/if}

					<!-- Studios -->
					{#if displayStudios()}
						<div class="flex justify-between items-start">
							<span class="text-text-muted">Studios</span>
							<span class="text-right">{displayStudios()}</span>
						</div>
					{/if}

					<div class="border-t border-border-subtle my-2"></div>

					<!-- Content Rating / Parental -->
					{#if contentRating}
						<div class="flex justify-between items-center">
							<span class="text-text-muted">Parental</span>
							<span class="flex items-center gap-2">
								<span class="px-1.5 py-0.5 bg-white/10 rounded text-xs font-medium">{contentRating}</span>
								{#if imdbId}
									<a href="https://www.imdb.com/title/{imdbId}/parentalguide" target="_blank" class="text-sky-400 hover:text-sky-300 text-xs">
										View
									</a>
								{/if}
							</span>
						</div>
					{/if}

					<!-- Extra info rows (library-specific like added date, play count) -->
					{#if extraInfoRows}
						{@render extraInfoRows()}
					{/if}

					<div class="border-t border-border-subtle my-2"></div>

					<!-- External Links -->
					<ExternalLinks
						{tmdbId}
						{imdbId}
						{mediaType}
					/>
				</div>
			</div>
		</div>
	</section>

	<!-- Extra sections (e.g., Files section for library) -->
	{#if extraSections}
		{@render extraSections()}
	{/if}

	<!-- Cast Section -->
	{#if cast.length > 0}
		<ScrollableRow title="Cast" padding="px-[60px]">
			{#each cast as actor}
				<PersonCard
					name={actor.name}
					role={actor.character}
					profilePath={getProfilePath(actor)}
					onclick={() => handlePersonClick({ id: actor.id, name: actor.name })}
				/>
			{/each}
		</ScrollableRow>
	{/if}

	<!-- Crew Section -->
	{#if crew.length > 0}
		<ScrollableRow title="Crew" padding="px-[60px]">
			{#each crew as member}
				<PersonCard
					name={member.name}
					role={member.job}
					profilePath={getProfilePath(member)}
					onclick={() => handlePersonClick({ id: member.id, name: member.name })}
				/>
			{/each}
		</ScrollableRow>
	{/if}

	<!-- More Like This -->
	{#if recommendations.length > 0}
		<ScrollableRow title="More Like This" padding="px-[60px]">
			{#each recommendations as rec}
				<MediaCard
					type="poster"
					href="{recLinkPrefix}/{recMediaPath}/{rec.id}"
					title={rec.title}
					subtitle={getRecReleaseYear(rec)}
					imagePath={getRecPosterPath(rec)}
					mediaType={mediaType}
					runtime={rec.runtime}
					contentRating={rec.contentRating}
				/>
			{/each}
		</ScrollableRow>
	{/if}

	<!-- Footer content (e.g., file info for library) -->
	{#if footerContent}
		{@render footerContent()}
	{/if}
</div>

<!-- Person Modal -->
<PersonModal
	personId={selectedPersonId}
	personName={selectedPersonName}
	onClose={closePersonModal}
/>

<!-- Trailer Modal -->
{#if trailerKey || trailersJson}
	<TrailerModal
		bind:open={showTrailerModal}
		trailers={trailerKey ? [{ key: trailerKey, name: 'Trailer', type: 'Trailer', site: 'YouTube' }] : undefined}
		trailersJson={trailersJson}
		{title}
	/>
{/if}
