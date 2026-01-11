<script lang="ts">
	import { onMount } from 'svelte';
	import {
		getCollections,
		getMediaCollections,
		addCollectionItem,
		removeCollectionItem,
		createCollection,
		type Collection
	} from '$lib/api';
	import IconButton from './IconButton.svelte';

	interface Props {
		tmdbId: number;
		mediaType: 'movie' | 'show';
		title: string;
		year?: number;
		posterPath?: string;
	}

	let { tmdbId, mediaType, title, year, posterPath }: Props = $props();

	let showMenu = $state(false);
	let allCollections: Collection[] = $state([]);
	let inCollections: Set<number> = $state(new Set());
	let loading = $state(true);
	let updating: number | null = $state(null);

	// Create new collection state
	let showCreateForm = $state(false);
	let newCollectionName = $state('');
	let creating = $state(false);

	onMount(async () => {
		await loadCollections();
	});

	async function loadCollections() {
		loading = true;
		try {
			const [all, current] = await Promise.all([
				getCollections(),
				getMediaCollections(tmdbId, mediaType)
			]);
			allCollections = all;
			inCollections = new Set(current.map(c => c.id));
		} catch (e) {
			console.error('Failed to load collections:', e);
		} finally {
			loading = false;
		}
	}

	async function toggleCollection(collection: Collection) {
		if (updating !== null) return;
		updating = collection.id;

		try {
			if (inCollections.has(collection.id)) {
				await removeCollectionItem(collection.id, tmdbId, mediaType);
				inCollections.delete(collection.id);
				inCollections = new Set(inCollections);
			} else {
				await addCollectionItem(collection.id, {
					mediaType,
					tmdbId,
					title,
					year,
					posterPath
				});
				inCollections.add(collection.id);
				inCollections = new Set(inCollections);
			}
		} catch (e) {
			console.error('Failed to update collection:', e);
		} finally {
			updating = null;
		}
	}

	async function handleCreateCollection(e: Event) {
		e.preventDefault();
		if (!newCollectionName.trim() || creating) return;

		creating = true;
		try {
			const newColl = await createCollection({ name: newCollectionName.trim() });
			allCollections = [...allCollections, newColl];
			// Automatically add the item to the new collection
			await addCollectionItem(newColl.id, {
				mediaType,
				tmdbId,
				title,
				year,
				posterPath
			});
			inCollections.add(newColl.id);
			inCollections = new Set(inCollections);
			newCollectionName = '';
			showCreateForm = false;
		} catch (e) {
			console.error('Failed to create collection:', e);
		} finally {
			creating = false;
		}
	}

	function handleOpenMenu() {
		showMenu = !showMenu;
		if (showMenu) {
			loadCollections();
		}
	}

	function closeMenu() {
		showMenu = false;
		showCreateForm = false;
		newCollectionName = '';
	}
</script>

<div class="relative">
	<IconButton
		onclick={handleOpenMenu}
		title="Add to Collection"
		active={inCollections.size > 0}
	>
		<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
			<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10" />
		</svg>
	</IconButton>

	{#if showMenu}
		<!-- svelte-ignore a11y_no_static_element_interactions -->
		<!-- svelte-ignore a11y_click_events_have_key_events -->
		<div
			class="fixed inset-0 z-[55]"
			onclick={closeMenu}
		></div>
		<!-- svelte-ignore a11y_no_static_element_interactions -->
		<!-- svelte-ignore a11y_click_events_have_key_events -->
		<div
			class="absolute left-1/2 -translate-x-1/2 mt-2 w-64 z-[60] bg-bg-dropdown border border-white/10 rounded-2xl shadow-xl overflow-hidden"
			onclick={(e: MouseEvent) => e.stopPropagation()}
		>
			<div class="px-4 py-3 border-b border-border-subtle">
				<h3 class="text-sm font-medium text-text-primary">Add to Collection</h3>
			</div>

			<div class="max-h-64 overflow-y-auto">
				{#if loading}
					<div class="flex items-center justify-center py-8">
						<div class="animate-spin w-5 h-5 border-2 border-accent-primary border-t-transparent rounded-full"></div>
					</div>
				{:else if allCollections.length === 0 && !showCreateForm}
					<div class="px-4 py-6 text-center">
						<p class="text-text-muted text-sm mb-3">No collections yet</p>
						<button
							onclick={() => showCreateForm = true}
							class="text-accent-primary text-sm hover:underline"
						>
							Create your first collection
						</button>
					</div>
				{:else}
					{#each allCollections as collection (collection.id)}
						<button
							onclick={() => toggleCollection(collection)}
							disabled={updating === collection.id}
							class="w-full flex items-center gap-3 px-4 py-2.5 text-left hover:bg-white/5 transition-colors disabled:opacity-50"
						>
							<div class="w-5 h-5 rounded border flex items-center justify-center flex-shrink-0 {inCollections.has(collection.id) ? 'bg-accent-primary border-accent-primary' : 'border-border-subtle'}">
								{#if updating === collection.id}
									<div class="animate-spin w-3 h-3 border border-white border-t-transparent rounded-full"></div>
								{:else if inCollections.has(collection.id)}
									<svg class="w-3 h-3 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
										<path stroke-linecap="round" stroke-linejoin="round" stroke-width="3" d="M5 13l4 4L19 7" />
									</svg>
								{/if}
							</div>
							<div class="flex-1 min-w-0">
								<p class="text-sm text-text-primary truncate">{collection.name}</p>
								<p class="text-xs text-text-muted">
									{collection.itemCount} {collection.itemCount === 1 ? 'item' : 'items'}
									{#if collection.isAuto}
										<span class="text-accent-primary ml-1">TMDB</span>
									{/if}
								</p>
							</div>
						</button>
					{/each}
				{/if}
			</div>

			{#if !showCreateForm && !loading}
				<div class="border-t border-border-subtle">
					<button
						onclick={() => showCreateForm = true}
						class="w-full flex items-center gap-2 px-4 py-2.5 text-sm text-accent-primary hover:bg-white/5 transition-colors"
					>
						<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" />
						</svg>
						Create New Collection
					</button>
				</div>
			{/if}

			{#if showCreateForm}
				<form onsubmit={handleCreateCollection} class="border-t border-border-subtle p-3">
					<input
						type="text"
						bind:value={newCollectionName}
						placeholder="Collection name"
						class="w-full px-3 py-2 rounded-lg bg-bg-secondary border border-border-subtle text-text-primary text-sm focus:outline-none focus:border-accent-primary mb-2"
						autofocus
					/>
					<div class="flex gap-2">
						<button
							type="button"
							onclick={() => { showCreateForm = false; newCollectionName = ''; }}
							class="flex-1 px-3 py-1.5 rounded-lg text-sm text-text-secondary hover:bg-white/5 transition-colors"
						>
							Cancel
						</button>
						<button
							type="submit"
							disabled={!newCollectionName.trim() || creating}
							class="flex-1 px-3 py-1.5 rounded-lg text-sm bg-accent-primary text-white font-medium hover:bg-accent-primary/90 transition-colors disabled:opacity-50"
						>
							{creating ? 'Creating...' : 'Create'}
						</button>
					</div>
				</form>
			{/if}
		</div>
	{/if}
</div>
