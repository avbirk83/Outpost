<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import {
		getSmartPlaylists,
		deleteSmartPlaylist,
		parseRules,
		type SmartPlaylist
	} from '$lib/api';
	import { auth } from '$lib/stores/auth';

	let playlists: SmartPlaylist[] = $state([]);
	let loading = $state(true);
	let error: string | null = $state(null);
	let deletingId: number | null = $state(null);
	let showDeleteConfirm: SmartPlaylist | null = $state(null);

	const user = $derived($auth);

	onMount(async () => {
		try {
			playlists = await getSmartPlaylists();
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to load playlists';
		} finally {
			loading = false;
		}
	});

	async function handleDelete(playlist: SmartPlaylist) {
		if (playlist.isSystem) return;
		deletingId = playlist.id;
		try {
			await deleteSmartPlaylist(playlist.id);
			playlists = playlists.filter(p => p.id !== playlist.id);
			showDeleteConfirm = null;
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to delete playlist';
		} finally {
			deletingId = null;
		}
	}

	function getPlaylistIcon(playlist: SmartPlaylist): string {
		const rules = parseRules(playlist.rules);
		if (!rules.conditions.length) return '📋';

		const firstField = rules.conditions[0].field;
		switch (firstField) {
			case 'added': return '🕐';
			case 'watched': return '👁️';
			case 'resolution': return '📺';
			case 'rating': return '⭐';
			case 'genre': return '🎭';
			case 'year': return '📅';
			case 'runtime': return '⏱️';
			default: return '📋';
		}
	}

	function getMediaTypeLabel(type: string): string {
		switch (type) {
			case 'movies': return 'Movies';
			case 'shows': return 'TV Shows';
			default: return 'All Media';
		}
	}

	function getRulesPreview(playlist: SmartPlaylist): string {
		const rules = parseRules(playlist.rules);
		if (!rules.conditions.length) return 'No rules defined';

		const previews = rules.conditions.slice(0, 2).map(c => {
			const field = c.field.charAt(0).toUpperCase() + c.field.slice(1);
			return `${field}: ${c.value}`;
		});

		if (rules.conditions.length > 2) {
			previews.push(`+${rules.conditions.length - 2} more`);
		}
		return previews.join(' • ');
	}
</script>

<svelte:head>
	<title>Smart Playlists - Outpost</title>
</svelte:head>

<div class="px-[60px] py-8 space-y-6">
	<!-- Header -->
	<div class="flex items-center justify-between">
		<div>
			<h1 class="text-3xl font-bold text-text-primary">Smart Playlists</h1>
			<p class="text-text-muted mt-1">Dynamic collections that auto-populate based on rules</p>
		</div>
		<button
			onclick={() => goto('/playlists/new')}
			class="px-5 py-2.5 rounded-xl bg-accent-primary text-white font-medium hover:bg-accent-primary/90 transition-colors flex items-center gap-2"
		>
			<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
				<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" />
			</svg>
			Create Playlist
		</button>
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

	{#if loading}
		<div class="flex items-center justify-center py-20">
			<div class="flex items-center gap-3">
				<div class="animate-spin w-8 h-8 border-2 border-accent-primary border-t-transparent rounded-full"></div>
				<span class="text-text-muted">Loading playlists...</span>
			</div>
		</div>
	{:else if playlists.length === 0}
		<div class="text-center py-20">
			<div class="text-6xl mb-4">📋</div>
			<h3 class="text-xl font-medium text-text-primary mb-2">No Smart Playlists Yet</h3>
			<p class="text-text-muted mb-6 max-w-md mx-auto">
				Create dynamic playlists that automatically populate based on rules like genre, year, rating, or watch status.
			</p>
			<button
				onclick={() => goto('/playlists/new')}
				class="px-6 py-3 rounded-xl bg-accent-primary text-white font-medium hover:bg-accent-primary/90 transition-colors"
			>
				Create Your First Playlist
			</button>
		</div>
	{:else}
		<!-- System Playlists -->
		{@const systemPlaylists = playlists.filter(p => p.isSystem)}
		{@const userPlaylists = playlists.filter(p => !p.isSystem)}

		{#if systemPlaylists.length > 0}
			<section class="space-y-4">
				<h2 class="text-lg font-semibold text-text-secondary">Built-in Playlists</h2>
				<div class="grid grid-cols-[repeat(auto-fill,minmax(350px,1fr))] gap-4">
					{#each systemPlaylists as playlist}
						<a
							href="/playlists/{playlist.id}"
							class="group bg-bg-card border border-border-subtle rounded-xl p-5 hover:border-border-hover transition-all hover:shadow-lg"
						>
							<div class="flex items-start gap-4">
								<div class="w-12 h-12 rounded-xl bg-bg-tertiary flex items-center justify-center text-2xl flex-shrink-0">
									{getPlaylistIcon(playlist)}
								</div>
								<div class="flex-1 min-w-0">
									<div class="flex items-center gap-2">
										<h3 class="font-semibold text-text-primary group-hover:text-accent-primary transition-colors truncate">
											{playlist.name}
										</h3>
										<span class="px-2 py-0.5 text-xs rounded-full bg-bg-tertiary text-text-muted">System</span>
									</div>
									{#if playlist.description}
										<p class="text-sm text-text-muted mt-1 line-clamp-1">{playlist.description}</p>
									{/if}
									<div class="flex items-center gap-3 mt-2 text-xs text-text-muted">
										<span class="flex items-center gap-1">
											<svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
												<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M7 4v16M17 4v16M3 8h4m10 0h4M3 12h18M3 16h4m10 0h4" />
											</svg>
											{getMediaTypeLabel(playlist.mediaType)}
										</span>
										{#if playlist.itemCount !== undefined}
											<span>{playlist.itemCount} items</span>
										{/if}
									</div>
								</div>
							</div>
						</a>
					{/each}
				</div>
			</section>
		{/if}

		{#if userPlaylists.length > 0}
			<section class="space-y-4">
				<h2 class="text-lg font-semibold text-text-secondary">Your Playlists</h2>
				<div class="grid grid-cols-[repeat(auto-fill,minmax(350px,1fr))] gap-4">
					{#each userPlaylists as playlist}
						<div class="group bg-bg-card border border-border-subtle rounded-xl p-5 hover:border-border-hover transition-all hover:shadow-lg relative">
							<a href="/playlists/{playlist.id}" class="block">
								<div class="flex items-start gap-4">
									<div class="w-12 h-12 rounded-xl bg-bg-tertiary flex items-center justify-center text-2xl flex-shrink-0">
										{getPlaylistIcon(playlist)}
									</div>
									<div class="flex-1 min-w-0 pr-8">
										<h3 class="font-semibold text-text-primary group-hover:text-accent-primary transition-colors truncate">
											{playlist.name}
										</h3>
										{#if playlist.description}
											<p class="text-sm text-text-muted mt-1 line-clamp-1">{playlist.description}</p>
										{:else}
											<p class="text-sm text-text-muted mt-1 line-clamp-1">{getRulesPreview(playlist)}</p>
										{/if}
										<div class="flex items-center gap-3 mt-2 text-xs text-text-muted">
											<span class="flex items-center gap-1">
												<svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
													<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M7 4v16M17 4v16M3 8h4m10 0h4M3 12h18M3 16h4m10 0h4" />
												</svg>
												{getMediaTypeLabel(playlist.mediaType)}
											</span>
											{#if playlist.itemCount !== undefined}
												<span>{playlist.itemCount} items</span>
											{/if}
										</div>
									</div>
								</div>
							</a>
							<!-- Delete button -->
							<button
								onclick={(e) => { e.preventDefault(); e.stopPropagation(); showDeleteConfirm = playlist; }}
								class="absolute top-4 right-4 p-2 rounded-lg text-text-muted hover:text-red-400 hover:bg-red-500/10 opacity-0 group-hover:opacity-100 transition-all"
								title="Delete playlist"
							>
								<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
								</svg>
							</button>
						</div>
					{/each}
				</div>
			</section>
		{/if}
	{/if}
</div>

<!-- Delete Confirmation Modal -->
{#if showDeleteConfirm}
	<div
		class="fixed inset-0 bg-black/80 z-50 flex items-center justify-center p-4"
		onclick={(e) => { if (e.target === e.currentTarget) showDeleteConfirm = null; }}
		onkeydown={(e) => { if (e.key === 'Escape') showDeleteConfirm = null; }}
		role="dialog"
		tabindex="-1"
	>
		<div class="bg-bg-primary border border-border-subtle rounded-xl w-full max-w-md">
			<div class="p-6">
				<h2 class="text-xl font-semibold text-text-primary mb-2">Delete Playlist?</h2>
				<p class="text-text-muted">
					Are you sure you want to delete "{showDeleteConfirm.name}"? This action cannot be undone.
				</p>
			</div>
			<div class="flex gap-3 p-6 pt-0">
				<button
					onclick={() => showDeleteConfirm = null}
					class="flex-1 px-4 py-2.5 rounded-lg border border-border-subtle text-text-secondary hover:bg-bg-secondary transition-colors"
				>
					Cancel
				</button>
				<button
					onclick={() => showDeleteConfirm && handleDelete(showDeleteConfirm)}
					disabled={deletingId === showDeleteConfirm?.id}
					class="flex-1 px-4 py-2.5 rounded-lg bg-red-500 text-white font-medium hover:bg-red-600 transition-colors disabled:opacity-50"
				>
					{deletingId === showDeleteConfirm?.id ? 'Deleting...' : 'Delete'}
				</button>
			</div>
		</div>
	</div>
{/if}
