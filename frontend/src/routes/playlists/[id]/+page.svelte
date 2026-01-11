<script lang="ts">
	import { page } from '$app/stores';
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import {
		getSmartPlaylist,
		refreshSmartPlaylist,
		deleteSmartPlaylist,
		getImageUrl,
		parseRules,
		type SmartPlaylistDetail,
		type SmartPlaylistItem
	} from '$lib/api';
	import MediaCard from '$lib/components/media/MediaCard.svelte';
	import { auth } from '$lib/stores/auth';

	const id = $derived(Number($page.params.id));

	let playlist: SmartPlaylistDetail | null = $state(null);
	let items: SmartPlaylistItem[] = $state([]);
	let loading = $state(true);
	let refreshing = $state(false);
	let error: string | null = $state(null);
	let showDeleteConfirm = $state(false);
	let deleting = $state(false);

	const user = $derived($auth);
	const canEdit = $derived(user?.role === 'admin' || (playlist?.userId === user?.id));

	onMount(() => {
		loadPlaylist();
	});

	async function loadPlaylist() {
		loading = true;
		error = null;
		try {
			const data = await getSmartPlaylist(id);
			playlist = data;
			items = data.items || [];
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to load playlist';
		} finally {
			loading = false;
		}
	}

	async function handleRefresh() {
		if (refreshing) return;
		refreshing = true;
		try {
			const result = await refreshSmartPlaylist(id);
			items = result.items;
			if (playlist) {
				playlist.itemCount = result.itemCount;
			}
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to refresh playlist';
		} finally {
			refreshing = false;
		}
	}

	async function handleDelete() {
		if (playlist?.isSystem) return;
		deleting = true;
		try {
			await deleteSmartPlaylist(id);
			goto('/playlists');
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to delete playlist';
			deleting = false;
		}
	}

	function getSortLabel(sortBy: string): string {
		switch (sortBy) {
			case 'added': return 'Date Added';
			case 'title': return 'Title';
			case 'year': return 'Year';
			case 'rating': return 'Rating';
			case 'runtime': return 'Runtime';
			default: return sortBy;
		}
	}

	function getRulesDescription(rulesJson: string): string[] {
		const rules = parseRules(rulesJson);
		return rules.conditions.map(c => {
			const field = c.field.charAt(0).toUpperCase() + c.field.slice(1);
			let op = '';
			switch (c.operator) {
				case 'eq': op = 'is'; break;
				case 'gte': op = '>='; break;
				case 'lte': op = '<='; break;
				case 'contains': op = 'contains'; break;
				case 'not_contains': op = 'does not contain'; break;
				case 'within': op = 'within'; break;
				default: op = c.operator;
			}
			let value = String(c.value);
			if (c.field === 'watched') {
				value = c.value ? 'Yes' : 'No';
			} else if (c.field === 'added' && c.operator === 'within') {
				value = c.value + ' (e.g., 7d, 30d)';
			}
			return `${field} ${op} ${value}`;
		});
	}

	function getMediaTypeLabel(type: string): string {
		switch (type) {
			case 'movies': return 'Movies only';
			case 'shows': return 'TV Shows only';
			default: return 'Movies & TV Shows';
		}
	}
</script>

<svelte:head>
	<title>{playlist?.name || 'Playlist'} - Outpost</title>
</svelte:head>

{#if loading}
	<div class="flex items-center justify-center min-h-[60vh]">
		<div class="flex items-center gap-3">
			<div class="animate-spin w-8 h-8 border-2 border-accent-primary border-t-transparent rounded-full"></div>
			<span class="text-text-muted">Loading playlist...</span>
		</div>
	</div>
{:else if error && !playlist}
	<div class="px-[60px] py-8">
		<div class="bg-red-500/10 border border-red-500/30 text-red-400 px-6 py-4 rounded-xl">
			<p class="font-medium mb-2">Error loading playlist</p>
			<p class="text-sm">{error}</p>
			<button onclick={() => goto('/playlists')} class="mt-4 text-sm underline hover:no-underline">
				Back to Playlists
			</button>
		</div>
	</div>
{:else if playlist}
	<div class="px-[60px] py-8 space-y-8">
		<!-- Header -->
		<div class="flex items-start gap-6">
			<div class="w-20 h-20 rounded-2xl bg-bg-tertiary flex items-center justify-center text-4xl flex-shrink-0">
				{#if playlist.isSystem}
					{#if playlist.name.includes('Recently')}
						🕐
					{:else if playlist.name.includes('Unwatched')}
						👁️
					{:else if playlist.name.includes('4K')}
						📺
					{:else if playlist.name.includes('Top Rated')}
						⭐
					{:else if playlist.name.includes('Short')}
						⏱️
					{:else}
						📋
					{/if}
				{:else}
					📋
				{/if}
			</div>
			<div class="flex-1">
				<div class="flex items-center gap-3 mb-2">
					<h1 class="text-3xl font-bold text-text-primary">{playlist.name}</h1>
					{#if playlist.isSystem}
						<span class="px-2.5 py-1 text-xs rounded-full bg-bg-tertiary text-text-muted font-medium">System</span>
					{/if}
				</div>
				{#if playlist.description}
					<p class="text-text-muted text-lg">{playlist.description}</p>
				{/if}
				<div class="flex items-center gap-4 mt-3 text-sm text-text-muted">
					<span class="flex items-center gap-1.5">
						<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M7 4v16M17 4v16M3 8h4m10 0h4M3 12h18M3 16h4m10 0h4" />
						</svg>
						{getMediaTypeLabel(playlist.mediaType)}
					</span>
					<span class="flex items-center gap-1.5">
						<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 4h13M3 8h9m-9 4h6m4 0l4-4m0 0l4 4m-4-4v12" />
						</svg>
						Sort: {getSortLabel(playlist.sortBy)} ({playlist.sortOrder === 'asc' ? 'A-Z' : 'Z-A'})
					</span>
					{#if playlist.limitCount}
						<span class="flex items-center gap-1.5">
							<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
								<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 7h6m0 10v-3m-3 3h.01M9 17h.01M9 14h.01M12 14h.01M15 11h.01M12 11h.01M9 11h.01M7 21h10a2 2 0 002-2V5a2 2 0 00-2-2H7a2 2 0 00-2 2v14a2 2 0 002 2z" />
							</svg>
							Limit: {playlist.limitCount} items
						</span>
					{/if}
					<span class="flex items-center gap-1.5">
						<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10" />
						</svg>
						{items.length} items
					</span>
				</div>
			</div>
			<div class="flex items-center gap-3">
				<button
					onclick={handleRefresh}
					disabled={refreshing}
					class="px-4 py-2.5 rounded-xl border border-border-subtle text-text-secondary hover:bg-bg-secondary transition-colors flex items-center gap-2 disabled:opacity-50"
				>
					<svg class="w-4 h-4 {refreshing ? 'animate-spin' : ''}" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
					</svg>
					{refreshing ? 'Refreshing...' : 'Refresh'}
				</button>
				{#if canEdit && !playlist.isSystem}
					<button
						onclick={() => goto(`/playlists/${id}/edit`)}
						class="px-4 py-2.5 rounded-xl bg-accent-primary text-white font-medium hover:bg-accent-primary/90 transition-colors flex items-center gap-2"
					>
						<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z" />
						</svg>
						Edit
					</button>
					<button
						onclick={() => showDeleteConfirm = true}
						class="p-2.5 rounded-xl border border-border-subtle text-text-muted hover:text-red-400 hover:border-red-400/50 hover:bg-red-500/10 transition-colors"
						title="Delete playlist"
					>
						<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
						</svg>
					</button>
				{/if}
			</div>
		</div>

		{#if error}
			<div class="bg-red-500/10 border border-red-500/30 text-red-400 px-4 py-3 rounded-xl flex items-center justify-between">
				<span>{error}</span>
				<button onclick={() => error = null} class="text-red-400 hover:text-red-300">
					<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
					</svg>
				</button>
			</div>
		{/if}

		<!-- Rules Preview -->
		<div class="bg-bg-card border border-border-subtle rounded-xl p-5">
			<h3 class="text-sm font-medium text-text-secondary uppercase tracking-wide mb-3">Rules</h3>
			<div class="flex flex-wrap gap-2">
				{@const rules = parseRules(playlist.rules)}
				<span class="px-3 py-1.5 rounded-lg bg-bg-tertiary text-text-muted text-sm">
					Match: {rules.match === 'all' ? 'All conditions' : 'Any condition'}
				</span>
				{#each getRulesDescription(playlist.rules) as rule}
					<span class="px-3 py-1.5 rounded-lg bg-accent-primary/20 text-accent-primary text-sm">
						{rule}
					</span>
				{/each}
				{#if rules.conditions.length === 0}
					<span class="text-text-muted text-sm">No rules defined - matches all media</span>
				{/if}
			</div>
		</div>

		<!-- Items Grid -->
		{#if items.length === 0}
			<div class="text-center py-16 bg-bg-card border border-border-subtle rounded-xl">
				<div class="text-5xl mb-4">📭</div>
				<h3 class="text-lg font-medium text-text-primary mb-2">No Items Found</h3>
				<p class="text-text-muted">No media in your library matches this playlist's rules.</p>
			</div>
		{:else}
			<div class="grid grid-cols-[repeat(auto-fill,minmax(200px,1fr))] gap-4">
				{#each items as item}
					<MediaCard
						type="poster"
						fill={true}
						href="/{item.mediaType === 'movie' ? 'movies' : 'tv'}/{item.id}"
						title={item.title}
						subtitle={item.year?.toString() || ''}
						imagePath={item.posterPath}
						isLocal={true}
						runtime={item.runtime}
					/>
				{/each}
			</div>
		{/if}
	</div>
{/if}

<!-- Delete Confirmation Modal -->
{#if showDeleteConfirm && playlist}
	<div
		class="fixed inset-0 bg-black/80 z-50 flex items-center justify-center p-4"
		onclick={(e) => { if (e.target === e.currentTarget) showDeleteConfirm = false; }}
		onkeydown={(e) => { if (e.key === 'Escape') showDeleteConfirm = false; }}
		role="dialog"
		tabindex="-1"
	>
		<div class="bg-bg-primary border border-border-subtle rounded-xl w-full max-w-md">
			<div class="p-6">
				<h2 class="text-xl font-semibold text-text-primary mb-2">Delete Playlist?</h2>
				<p class="text-text-muted">
					Are you sure you want to delete "{playlist.name}"? This action cannot be undone.
				</p>
			</div>
			<div class="flex gap-3 p-6 pt-0">
				<button
					onclick={() => showDeleteConfirm = false}
					class="flex-1 px-4 py-2.5 rounded-lg border border-border-subtle text-text-secondary hover:bg-bg-secondary transition-colors"
				>
					Cancel
				</button>
				<button
					onclick={handleDelete}
					disabled={deleting}
					class="flex-1 px-4 py-2.5 rounded-lg bg-red-500 text-white font-medium hover:bg-red-600 transition-colors disabled:opacity-50"
				>
					{deleting ? 'Deleting...' : 'Delete'}
				</button>
			</div>
		</div>
	</div>
{/if}
