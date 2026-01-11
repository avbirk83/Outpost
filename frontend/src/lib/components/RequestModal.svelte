<script lang="ts">
	import { getQualityPresets, getTmdbImageUrl, getDiscoverShowDetail, type QualityPreset } from '$lib/api';
	import { onMount, onDestroy } from 'svelte';
	import TypeBadge from './TypeBadge.svelte';
	import Select from './ui/Select.svelte';

	interface SeasonInfo {
		seasonNumber: number;
		name: string;
		episodeCount: number;
		airDate?: string;
	}

	interface MediaItem {
		title: string;
		year?: number;
		type: 'movie' | 'show';
		posterPath?: string | null;
		backdropPath?: string | null;
		overview?: string;
		tmdbId?: number;
	}

	interface Props {
		item: MediaItem;
		mode: 'request' | 'approve';
		onConfirm: (qualityPresetId: number, selectedSeasons?: number[]) => void;
		onCancel: () => void;
	}

	let { item, mode, onConfirm, onCancel }: Props = $props();

	let presets: QualityPreset[] = $state([]);
	let selectedPresetId: number = $state(0);
	let loading = $state(true);
	let loadingSeasons = $state(false);

	// Season selection
	let seasons: SeasonInfo[] = $state([]);
	let selectedSeasons: Set<number> = $state(new Set());
	let allSeasonsSelected = $derived(seasons.length > 0 && selectedSeasons.size === seasons.length);

	// Lock body scroll when modal is open
	onMount(() => {
		document.body.style.overflow = 'hidden';
	});

	onDestroy(() => {
		document.body.style.overflow = '';
	});

	onMount(async () => {
		try {
			// Load quality presets filtered by media type
			const allPresets = await getQualityPresets();
			const targetMediaType = item.type === 'movie' ? 'movie' : 'tv';
			presets = allPresets.filter(p => p.enabled && p.mediaType === targetMediaType);
			const defaultPreset = presets.find(p => p.isDefault);
			if (defaultPreset) {
				selectedPresetId = defaultPreset.id;
			} else if (presets.length > 0) {
				selectedPresetId = presets[0].id;
			}

			// Load seasons for TV shows
			if (item.type === 'show' && item.tmdbId) {
				loadingSeasons = true;
				try {
					const showDetail = await getDiscoverShowDetail(item.tmdbId);
					if (showDetail.seasonDetails) {
						seasons = showDetail.seasonDetails
							.filter((s: any) => s.season_number > 0) // Exclude specials
							.map((s: any) => ({
								seasonNumber: s.season_number,
								name: s.name || `Season ${s.season_number}`,
								episodeCount: s.episode_count || 0,
								airDate: s.air_date
							}));
						// Select all seasons by default
						selectedSeasons = new Set(seasons.map(s => s.seasonNumber));
					}
				} catch (e) {
					console.error('Failed to load season details:', e);
				} finally {
					loadingSeasons = false;
				}
			}
		} catch (e) {
			console.error('Failed to load quality presets:', e);
		} finally {
			loading = false;
		}
	});

	function handleConfirm() {
		const seasonsArray = item.type === 'show' ? Array.from(selectedSeasons) : undefined;
		onConfirm(selectedPresetId, seasonsArray);
	}

	function handleBackdropClick(e: MouseEvent) {
		if (e.target === e.currentTarget) {
			onCancel();
		}
	}

	function handleKeydown(e: KeyboardEvent) {
		if (e.key === 'Escape') {
			onCancel();
		}
	}

	function getPresetDescription(preset: QualityPreset): string {
		const parts: string[] = [];
		if (preset.resolution && preset.resolution !== 'any') parts.push(preset.resolution.toUpperCase());
		if (preset.source && preset.source !== 'any') parts.push(preset.source);
		if (preset.codec) parts.push(preset.codec.toUpperCase());
		if (preset.hdrFormats) {
			try {
				const formats = JSON.parse(preset.hdrFormats);
				if (formats.length > 0) parts.push('HDR');
			} catch {}
		}
		return parts.join(' · ') || 'Any quality';
	}

	function toggleSeason(seasonNumber: number) {
		if (selectedSeasons.has(seasonNumber)) {
			selectedSeasons.delete(seasonNumber);
		} else {
			selectedSeasons.add(seasonNumber);
		}
		selectedSeasons = new Set(selectedSeasons);
	}

	function toggleAllSeasons() {
		if (allSeasonsSelected) {
			selectedSeasons = new Set();
		} else {
			selectedSeasons = new Set(seasons.map(s => s.seasonNumber));
		}
	}

	const buttonText = $derived(mode === 'request' ? 'Request' : 'Approve & Search');
	const descriptionText = $derived(mode === 'request'
		? 'Your request will be sent for approval. Once approved, the item will be searched for on configured indexers.'
		: 'The item will be added to the wanted list and searched for on your indexers. If found, it will be sent to your download client automatically.');

	const canSubmit = $derived(() => {
		if (loading) return false;
		if (presets.length > 0 && !selectedPresetId) return false;
		if (item.type === 'show' && seasons.length > 0 && selectedSeasons.size === 0) return false;
		return true;
	});
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
		<!-- Header with backdrop -->
		<div class="relative h-40 bg-gradient-to-b from-white/10 to-transparent flex-shrink-0">
			{#if item.backdropPath}
				<img
					src={getTmdbImageUrl(item.backdropPath, 'w780')}
					alt=""
					class="absolute inset-0 w-full h-full object-cover opacity-40"
				/>
			{/if}
			<div class="absolute inset-0 bg-gradient-to-t from-bg-base via-bg-base/60 to-transparent"></div>

			<!-- Close button -->
			<button
				onclick={onCancel}
				class="absolute top-4 right-4 w-8 h-8 rounded-full bg-black/50 hover:bg-black/70 flex items-center justify-center text-white/70 hover:text-white transition-colors z-10"
				aria-label="Close"
			>
				<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
				</svg>
			</button>

			<div class="absolute bottom-0 left-0 right-0 p-5 flex items-end gap-4">
				{#if item.posterPath}
					<img
						src={getTmdbImageUrl(item.posterPath, 'w185')}
						alt={item.title}
						class="w-20 h-28 object-cover rounded-xl shadow-lg flex-shrink-0 -mb-10 border border-white/10"
					/>
				{:else}
					<div class="w-20 h-28 bg-bg-elevated rounded-xl flex items-center justify-center flex-shrink-0 -mb-10 border border-white/10">
						<svg class="w-8 h-8 text-text-muted" fill="none" stroke="currentColor" viewBox="0 0 24 24">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z" />
						</svg>
					</div>
				{/if}

				<div class="flex-1 min-w-0 pb-1">
					<h2 id="modal-title" class="text-xl font-bold text-text-primary truncate">
						{item.title}
					</h2>
					<div class="flex items-center gap-2 mt-1.5">
						<TypeBadge type={item.type} />
						{#if item.year}
							<span class="text-sm text-text-secondary">{item.year}</span>
						{/if}
					</div>
				</div>
			</div>
		</div>

		<!-- Content -->
		<div class="p-5 pt-14 space-y-5 overflow-y-auto flex-1 scrollbar-thin">
			<!-- Quality Preset -->
			<div>
				<label for="quality-preset" class="block text-sm font-medium text-text-primary mb-2">
					Quality Preset
				</label>
				{#if loading}
					<div class="h-11 bg-bg-elevated rounded-xl animate-pulse"></div>
				{:else if presets.length === 0}
					<p class="text-sm text-text-muted">No quality presets found. Default settings will be used.</p>
				{:else}
					<Select
						bind:value={selectedPresetId}
						options={presets.map(p => ({
							value: p.id,
							label: p.name + (p.isDefault ? ' (Default)' : ''),
						}))}
					/>
					{#if selectedPresetId}
						{@const selectedPreset = presets.find(p => p.id === selectedPresetId)}
						{#if selectedPreset}
							<p class="text-xs text-text-muted mt-2">
								{getPresetDescription(selectedPreset)}
							</p>
						{/if}
					{/if}
				{/if}
			</div>

			<!-- Season Selection (TV Shows only) -->
			{#if item.type === 'show'}
				<div>
					<div class="flex items-center justify-between mb-3">
						<label class="block text-sm font-medium text-text-primary">
							Seasons
						</label>
						{#if seasons.length > 0}
							<button
								onclick={toggleAllSeasons}
								class="text-xs text-amber-400 hover:text-amber-300 transition-colors"
							>
								{allSeasonsSelected ? 'Deselect All' : 'Select All'}
							</button>
						{/if}
					</div>

					{#if loadingSeasons}
						<div class="grid grid-cols-2 sm:grid-cols-3 gap-2">
							{#each [1, 2, 3] as _}
								<div class="h-16 bg-bg-elevated rounded-xl animate-pulse"></div>
							{/each}
						</div>
					{:else if seasons.length === 0}
						<p class="text-sm text-text-muted">All seasons will be monitored.</p>
					{:else}
						<div class="grid grid-cols-2 sm:grid-cols-3 gap-2 max-h-48 overflow-y-auto pr-1 scrollbar-thin">
							{#each seasons as season}
								<button
									onclick={() => toggleSeason(season.seasonNumber)}
									class="p-3 rounded-xl border text-left transition-all {selectedSeasons.has(season.seasonNumber)
										? 'bg-amber-500/20 border-amber-500/50 text-text-primary'
										: 'bg-bg-elevated border-border-subtle text-text-secondary hover:border-border-hover hover:text-text-primary'}"
								>
									<div class="font-medium text-sm">Season {season.seasonNumber}</div>
									<div class="text-xs text-text-muted mt-0.5">{season.episodeCount} episodes</div>
								</button>
							{/each}
						</div>
						{#if selectedSeasons.size === 0}
							<p class="text-xs text-amber-400 mt-2">Please select at least one season</p>
						{/if}
					{/if}
				</div>
			{/if}

			<!-- Info box -->
			<div class="bg-bg-elevated/50 rounded-xl p-4 border border-border-subtle">
				<p class="text-sm text-text-secondary">
					{descriptionText}
				</p>
			</div>
		</div>

		<!-- Footer -->
		<div class="flex items-center justify-end gap-3 p-5 border-t border-border-subtle flex-shrink-0">
			<button
				onclick={onCancel}
				class="px-5 py-2.5 rounded-xl text-sm font-medium bg-bg-elevated text-text-secondary hover:bg-bg-elevated/80 hover:text-text-primary transition-colors"
			>
				Cancel
			</button>
			<button
				onclick={handleConfirm}
				disabled={!canSubmit()}
				class="px-5 py-2.5 rounded-xl text-sm font-medium bg-amber-500 text-black hover:bg-amber-400 disabled:opacity-50 disabled:cursor-not-allowed transition-colors flex items-center gap-2"
			>
				{#if mode === 'request'}
					<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4" />
					</svg>
				{:else}
					<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
					</svg>
				{/if}
				{buttonText}
			</button>
		</div>
	</div>
</div>
