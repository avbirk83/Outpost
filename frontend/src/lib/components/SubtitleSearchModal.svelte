<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import {
		searchSubtitles,
		downloadSubtitle,
		COMMON_LANGUAGES,
		formatSubtitleDate,
		formatDownloads,
		type SubtitleResult,
		type SubtitleSearchParams
	} from '$lib/api';
	import Select from './ui/Select.svelte';

	interface MediaInfo {
		type: 'movie' | 'episode';
		mediaId: number;
		title: string;
		year?: number;
		tmdbId?: number;
		imdbId?: string;
		episodeId?: number;
		seasonNumber?: number;
		episodeNumber?: number;
		showTitle?: string;
	}

	interface Props {
		media: MediaInfo;
		onClose: () => void;
		onDownloaded?: (path: string) => void;
	}

	let { media, onClose, onDownloaded }: Props = $props();

	let results: SubtitleResult[] = $state([]);
	let loading = $state(false);
	let downloading = $state<string | null>(null);
	let error = $state<string | null>(null);
	let success = $state<string | null>(null);

	// Search parameters
	let searchQuery = $state(media.type === 'episode' ? (media.showTitle || media.title) : media.title);
	let selectedLanguage = $state('en');

	// Lock body scroll when modal is open
	onMount(() => {
		document.body.style.overflow = 'hidden';
		// Auto-search on mount
		doSearch();
	});

	onDestroy(() => {
		document.body.style.overflow = '';
	});

	async function doSearch() {
		loading = true;
		error = null;
		results = [];

		try {
			const params: SubtitleSearchParams = {
				languages: [selectedLanguage]
			};

			// Prefer TMDB ID for better matching
			if (media.tmdbId) {
				params.tmdbId = media.tmdbId;
			} else if (media.imdbId) {
				params.imdbId = media.imdbId;
			} else {
				params.query = searchQuery;
				if (media.year) params.year = media.year;
			}

			// For episodes, add season/episode info
			if (media.type === 'episode' && media.seasonNumber && media.episodeNumber) {
				params.season = media.seasonNumber;
				params.episode = media.episodeNumber;
			}

			results = await searchSubtitles(params);
		} catch (e) {
			error = e instanceof Error ? e.message : 'Search failed';
		} finally {
			loading = false;
		}
	}

	async function handleDownload(subtitle: SubtitleResult) {
		downloading = subtitle.id;
		error = null;
		success = null;

		try {
			const result = await downloadSubtitle({
				fileId: subtitle.fileId,
				mediaType: media.type,
				mediaId: media.mediaId,
				language: subtitle.language,
				episodeId: media.episodeId
			});

			if (result.success) {
				success = `Downloaded successfully! ${result.remaining} downloads remaining today.`;
				onDownloaded?.(result.path);
			}
		} catch (e) {
			error = e instanceof Error ? e.message : 'Download failed';
		} finally {
			downloading = null;
		}
	}

	function handleBackdropClick(e: MouseEvent) {
		if (e.target === e.currentTarget) {
			onClose();
		}
	}

	function handleKeydown(e: KeyboardEvent) {
		if (e.key === 'Escape') {
			onClose();
		}
	}

	function getLanguageName(code: string): string {
		const lang = COMMON_LANGUAGES.find(l => l.code === code);
		return lang?.name || code.toUpperCase();
	}

	const displayTitle = $derived(
		media.type === 'episode' && media.showTitle
			? `${media.showTitle} - S${String(media.seasonNumber).padStart(2, '0')}E${String(media.episodeNumber).padStart(2, '0')}`
			: media.title
	);
</script>

<svelte:window onkeydown={handleKeydown} />

<!-- svelte-ignore a11y_no_noninteractive_element_interactions -->
<!-- svelte-ignore a11y_click_events_have_key_events -->
<div
	class="modal-overlay"
	onclick={handleBackdropClick}
	role="dialog"
	aria-modal="true"
	aria-labelledby="modal-title"
>
	<div class="modal-container-xl">
		<!-- Header -->
		<div class="p-5 border-b border-border-subtle">
			<div class="flex items-center justify-between">
				<div class="flex items-center gap-3">
					<div class="w-10 h-10 rounded-xl bg-amber-500/20 flex items-center justify-center">
						<svg class="w-5 h-5 text-amber-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M7 8h10M7 12h4m1 8l-4-4H5a2 2 0 01-2-2V6a2 2 0 012-2h14a2 2 0 012 2v8a2 2 0 01-2 2h-3l-4 4z" />
						</svg>
					</div>
					<div>
						<h2 id="modal-title" class="text-lg font-semibold text-text-primary">Search Subtitles</h2>
						<p class="text-sm text-text-secondary truncate max-w-sm">{displayTitle}</p>
					</div>
				</div>
				<button
					onclick={onClose}
					class="w-8 h-8 rounded-full bg-bg-elevated hover:bg-bg-elevated/80 flex items-center justify-center text-text-muted hover:text-text-primary transition-colors"
					aria-label="Close"
				>
					<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
					</svg>
				</button>
			</div>

			<!-- Search Controls -->
			<div class="mt-4 flex gap-3">
				<div class="flex-1">
					<input
						type="text"
						bind:value={searchQuery}
						placeholder="Search query..."
						class="w-full px-4 py-2.5 bg-bg-elevated border border-border-subtle rounded-xl text-text-primary placeholder-text-muted focus:outline-none focus:border-amber-500/50"
						onkeydown={(e) => e.key === 'Enter' && doSearch()}
					/>
				</div>
				<div class="w-40">
					<Select
						bind:value={selectedLanguage}
						options={COMMON_LANGUAGES.map(l => ({ value: l.code, label: l.name }))}
					/>
				</div>
				<button
					onclick={doSearch}
					disabled={loading}
					class="px-5 py-2.5 rounded-xl text-sm font-medium bg-amber-500 text-black hover:bg-amber-400 disabled:opacity-50 disabled:cursor-not-allowed transition-colors flex items-center gap-2"
				>
					{#if loading}
						<svg class="w-4 h-4 animate-spin" fill="none" viewBox="0 0 24 24">
							<circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
							<path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
						</svg>
					{:else}
						<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
						</svg>
					{/if}
					Search
				</button>
			</div>
		</div>

		<!-- Results -->
		<div class="flex-1 overflow-y-auto scrollbar-thin" style="max-height: 50vh;">
			{#if error}
				<div class="m-5 p-4 bg-red-500/10 border border-red-500/30 rounded-xl">
					<p class="text-sm text-red-400">{error}</p>
				</div>
			{/if}

			{#if success}
				<div class="m-5 p-4 bg-green-500/10 border border-green-500/30 rounded-xl">
					<p class="text-sm text-green-400">{success}</p>
				</div>
			{/if}

			{#if loading}
				<div class="flex items-center justify-center py-12">
					<div class="flex flex-col items-center gap-3">
						<svg class="w-8 h-8 animate-spin text-amber-400" fill="none" viewBox="0 0 24 24">
							<circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
							<path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
						</svg>
						<p class="text-sm text-text-muted">Searching OpenSubtitles...</p>
					</div>
				</div>
			{:else if results.length === 0}
				<div class="flex items-center justify-center py-12">
					<div class="text-center">
						<svg class="w-12 h-12 mx-auto text-text-muted mb-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M9.172 16.172a4 4 0 015.656 0M9 10h.01M15 10h.01M12 2a10 10 0 110 20 10 10 0 010-20z" />
						</svg>
						<p class="text-text-muted">No subtitles found</p>
						<p class="text-sm text-text-muted mt-1">Try a different search query or language</p>
					</div>
				</div>
			{:else}
				<div class="divide-y divide-border-subtle">
					{#each results as subtitle}
						<div class="p-4 hover:bg-bg-elevated/50 transition-colors">
							<div class="flex items-start justify-between gap-4">
								<div class="flex-1 min-w-0">
									<div class="flex items-center gap-2 flex-wrap">
										<span class="text-sm font-medium text-text-primary truncate">
											{subtitle.fileName || subtitle.release || 'Unknown'}
										</span>
										{#if subtitle.hearingImpaired}
											<span class="px-1.5 py-0.5 text-xs bg-blue-500/20 text-blue-400 rounded">HI</span>
										{/if}
										{#if subtitle.fromTrusted}
											<span class="px-1.5 py-0.5 text-xs bg-green-500/20 text-green-400 rounded">Trusted</span>
										{/if}
										{#if subtitle.aiTranslated}
											<span class="px-1.5 py-0.5 text-xs bg-yellow-500/20 text-yellow-400 rounded">AI</span>
										{/if}
									</div>
									<div class="flex items-center gap-3 mt-1.5 text-xs text-text-muted">
										<span class="flex items-center gap-1">
											<svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
												<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 5h12M9 3v2m1.048 9.5A18.022 18.022 0 016.412 9m6.088 9h7M11 21l5-10 5 10M12.751 5C11.783 10.77 8.07 15.61 3 18.129" />
											</svg>
											{getLanguageName(subtitle.language)}
										</span>
										<span class="flex items-center gap-1">
											<svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
												<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4" />
											</svg>
											{formatDownloads(subtitle.downloads)}
										</span>
										{#if subtitle.uploadDate}
											<span class="flex items-center gap-1">
												<svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
													<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 7V3m8 4V3m-9 8h10M5 21h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z" />
												</svg>
												{formatSubtitleDate(subtitle.uploadDate)}
											</span>
										{/if}
										{#if subtitle.fps}
											<span>{subtitle.fps} fps</span>
										{/if}
									</div>
								</div>
								<button
									onclick={() => handleDownload(subtitle)}
									disabled={downloading !== null}
									class="px-4 py-2 rounded-lg text-sm font-medium bg-amber-500 text-black hover:bg-amber-400 disabled:opacity-50 disabled:cursor-not-allowed transition-colors flex items-center gap-2 flex-shrink-0"
								>
									{#if downloading === subtitle.id}
										<svg class="w-4 h-4 animate-spin" fill="none" viewBox="0 0 24 24">
											<circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
											<path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
										</svg>
									{:else}
										<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
											<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4" />
										</svg>
									{/if}
									Download
								</button>
							</div>
						</div>
					{/each}
				</div>
			{/if}
		</div>

		<!-- Footer -->
		<div class="p-4 border-t border-border-subtle bg-bg-elevated/30">
			<p class="text-xs text-text-muted text-center">
				Powered by OpenSubtitles.com
			</p>
		</div>
	</div>
</div>
