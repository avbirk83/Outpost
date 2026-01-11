<script lang="ts">
	import { page } from '$app/stores';
	import { goto } from '$app/navigation';
	import { onMount } from 'svelte';
	import {
		getCollection,
		updateCollection,
		deleteCollection,
		removeCollectionItem,
		reorderCollectionItems,
		getImageUrl,
		type CollectionDetail,
		type CollectionItem
	} from '$lib/api';
	import MediaCard from '$lib/components/media/MediaCard.svelte';
	import { auth } from '$lib/stores/auth';

	let collection: CollectionDetail | null = $state(null);
	let loading = $state(true);
	let error: string | null = $state(null);

	// Edit modal state
	let showEditModal = $state(false);
	let editName = $state('');
	let editDescription = $state('');
	let saving = $state(false);

	// Delete confirmation
	let confirmingDelete = $state(false);
	let deleting = $state(false);

	// Drag and drop state
	let draggingItemId: number | null = $state(null);
	let dragOverItemId: number | null = $state(null);

	const user = $derived($auth);
	const isAdmin = $derived(user?.role === 'admin');

	// Sort items based on collection's sortOrder
	const sortedItems = $derived(() => {
		if (!collection?.items) return [];
		const items = [...collection.items];

		switch (collection.sortOrder) {
			case 'release':
				return items.sort((a, b) => (a.year || 0) - (b.year || 0));
			case 'title':
				return items.sort((a, b) => a.title.localeCompare(b.title));
			case 'added':
				return items.sort((a, b) => new Date(b.addedAt).getTime() - new Date(a.addedAt).getTime());
			case 'custom':
			default:
				return items.sort((a, b) => a.sortOrder - b.sortOrder);
		}
	});

	onMount(async () => {
		const id = parseInt($page.params.id || '0');
		try {
			collection = await getCollection(id);
			editName = collection.name;
			editDescription = collection.description || '';
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to load collection';
		} finally {
			loading = false;
		}
	});

	async function handleSaveEdit(e: Event) {
		e.preventDefault();
		if (!collection || saving) return;

		saving = true;
		try {
			await updateCollection(collection.id, {
				name: editName,
				description: editDescription || undefined
			});
			collection.name = editName;
			collection.description = editDescription || undefined;
			showEditModal = false;
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to update collection';
		} finally {
			saving = false;
		}
	}

	async function handleDelete() {
		if (!collection || deleting) return;

		deleting = true;
		try {
			await deleteCollection(collection.id);
			goto('/library?tab=collections');
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to delete collection';
			deleting = false;
		}
	}

	async function handleRemoveItem(item: CollectionItem) {
		if (!collection) return;
		try {
			await removeCollectionItem(collection.id, item.tmdbId, item.mediaType);
			collection.items = collection.items.filter(i => i.id !== item.id);
			collection.itemCount--;
			if (item.inLibrary) collection.ownedCount--;
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to remove item';
		}
	}

	async function handleSortChange(newSort: string) {
		if (!collection) return;
		try {
			await updateCollection(collection.id, { sortOrder: newSort });
			collection.sortOrder = newSort as CollectionDetail['sortOrder'];
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to update sort order';
		}
	}

	// Drag and drop handlers
	function handleDragStart(e: DragEvent, itemId: number) {
		draggingItemId = itemId;
		if (e.dataTransfer) {
			e.dataTransfer.effectAllowed = 'move';
			e.dataTransfer.setData('text/plain', String(itemId));
		}
	}

	function handleDragOver(e: DragEvent, itemId: number) {
		e.preventDefault();
		if (e.dataTransfer) {
			e.dataTransfer.dropEffect = 'move';
		}
		if (draggingItemId !== itemId) {
			dragOverItemId = itemId;
		}
	}

	function handleDragLeave() {
		dragOverItemId = null;
	}

	function handleDragEnd() {
		draggingItemId = null;
		dragOverItemId = null;
	}

	async function handleDrop(e: DragEvent, targetItemId: number) {
		e.preventDefault();
		if (!collection || draggingItemId === null || draggingItemId === targetItemId) {
			handleDragEnd();
			return;
		}

		const items = sortedItems();
		const draggedIndex = items.findIndex(i => i.id === draggingItemId);
		const targetIndex = items.findIndex(i => i.id === targetItemId);

		if (draggedIndex === -1 || targetIndex === -1) {
			handleDragEnd();
			return;
		}

		// Create new order
		const newItems = [...items];
		const [removed] = newItems.splice(draggedIndex, 1);
		newItems.splice(targetIndex, 0, removed);

		// Update sort order
		const newOrder = newItems.map(i => i.id);

		try {
			await reorderCollectionItems(collection.id, newOrder);
			// Update local state
			collection.items = newItems.map((item, index) => ({ ...item, sortOrder: index }));
			collection.sortOrder = 'custom';
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to reorder items';
		}

		handleDragEnd();
	}
</script>

<svelte:head>
	<title>{collection?.name || 'Collection'} - Outpost</title>
</svelte:head>

{#if loading}
	<div class="flex items-center justify-center min-h-screen">
		<div class="animate-spin w-8 h-8 border-2 border-accent-primary border-t-transparent rounded-full"></div>
	</div>
{:else if error}
	<div class="flex items-center justify-center min-h-screen">
		<div class="text-center">
			<p class="text-text-muted mb-4">{error}</p>
			<a href="/library?tab=collections" class="text-accent-primary hover:underline">Back to Collections</a>
		</div>
	</div>
{:else if collection}
	<!-- Hero Section -->
	<div class="relative">
		{#if collection.backdropPath}
			<div class="absolute inset-0 h-[500px]">
				<img
					src={getImageUrl(collection.backdropPath)}
					alt=""
					class="w-full h-full object-cover"
				/>
				<div class="absolute inset-0 bg-gradient-to-t from-bg-primary via-bg-primary/80 to-transparent"></div>
			</div>
		{:else}
			<div class="absolute inset-0 h-[500px] bg-gradient-to-b from-bg-secondary to-bg-primary"></div>
		{/if}

		<div class="relative px-[60px] pt-32 pb-8">
			<div class="flex gap-8">
				<!-- Poster -->
				<div class="flex-shrink-0 w-64">
					{#if collection.posterPath}
						<img
							src={getImageUrl(collection.posterPath)}
							alt={collection.name}
							class="w-full rounded-xl shadow-2xl"
						/>
					{:else}
						<div class="w-full aspect-[2/3] rounded-xl bg-bg-card border border-border-subtle flex items-center justify-center text-6xl">
							📚
						</div>
					{/if}
				</div>

				<!-- Info -->
				<div class="flex-1 pt-8">
					<div class="flex items-start justify-between gap-4">
						<div>
							{#if collection.isAuto}
								<span class="inline-block px-2 py-1 rounded bg-accent-primary/20 text-accent-primary text-xs font-medium mb-3">
									TMDB Collection
								</span>
							{/if}
							<h1 class="text-4xl font-bold text-text-primary mb-2">{collection.name}</h1>
							<p class="text-lg text-text-muted">
								{collection.ownedCount} of {collection.itemCount} in library
							</p>
						</div>

						{#if isAdmin}
							<div class="flex gap-2">
								<button
									onclick={() => showEditModal = true}
									class="px-4 py-2 rounded-lg bg-bg-card border border-border-subtle text-text-secondary hover:bg-bg-secondary transition-colors"
								>
									Edit
								</button>
								<button
									onclick={() => confirmingDelete = true}
									class="px-4 py-2 rounded-lg bg-red-500/10 border border-red-500/30 text-red-400 hover:bg-red-500/20 transition-colors"
								>
									Delete
								</button>
							</div>
						{/if}
					</div>

					{#if collection.description}
						<p class="text-text-secondary mt-4 max-w-2xl">{collection.description}</p>
					{/if}
				</div>
			</div>
		</div>
	</div>

	<!-- Content -->
	<div class="px-[60px] py-8 space-y-6">
		<!-- Controls -->
		<div class="flex items-center justify-between">
			<h2 class="text-xl font-semibold text-text-primary">Items</h2>
			<div class="flex items-center gap-3">
				<span class="text-sm text-text-muted">Sort by:</span>
				<select
					value={collection.sortOrder}
					onchange={(e) => handleSortChange(e.currentTarget.value)}
					class="px-3 py-1.5 rounded-lg bg-bg-card border border-border-subtle text-text-primary text-sm focus:outline-none focus:border-accent-primary"
				>
					<option value="release">Release Date</option>
					<option value="title">Title</option>
					<option value="added">Added</option>
					<option value="custom">Custom</option>
				</select>
			</div>
		</div>

		<!-- Items Grid -->
		<div class="grid grid-cols-[repeat(auto-fill,minmax(180px,1fr))] gap-4">
			{#each sortedItems() as item (item.id)}
				<div
					class="group relative"
					draggable={collection.sortOrder === 'custom' && isAdmin}
					ondragstart={(e) => handleDragStart(e, item.id)}
					ondragover={(e) => handleDragOver(e, item.id)}
					ondragleave={handleDragLeave}
					ondragend={handleDragEnd}
					ondrop={(e) => handleDrop(e, item.id)}
					class:opacity-50={draggingItemId === item.id}
					class:ring-2={dragOverItemId === item.id}
					class:ring-accent-primary={dragOverItemId === item.id}
				>
					{#if item.inLibrary && item.mediaId}
						<a href="/{item.mediaType === 'movie' ? 'movies' : 'tv'}/{item.mediaId}">
							<MediaCard
								type="poster"
								fill={true}
								title={item.title}
								subtitle={item.year?.toString() || ''}
								imagePath={item.posterPath}
								isLocal={true}
							/>
						</a>
					{:else}
						<div class="relative">
							<MediaCard
								type="poster"
								fill={true}
								title={item.title}
								subtitle={item.year?.toString() || ''}
								imagePath={item.posterPath}
								isLocal={false}
							/>
							<!-- Not in library overlay -->
							<div class="absolute inset-0 bg-black/60 flex items-center justify-center rounded-xl">
								<span class="px-3 py-1.5 rounded-lg bg-bg-card text-text-muted text-sm">
									Not in Library
								</span>
							</div>
						</div>
					{/if}

					<!-- Remove button (admin only) -->
					{#if isAdmin}
						<button
							onclick={() => handleRemoveItem(item)}
							class="absolute top-2 right-2 w-8 h-8 rounded-full bg-black/70 text-white opacity-0 group-hover:opacity-100 transition-opacity flex items-center justify-center hover:bg-red-500"
							title="Remove from collection"
						>
							<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
								<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
							</svg>
						</button>
					{/if}
				</div>
			{/each}
		</div>

		{#if collection.items.length === 0}
			<div class="text-center py-20">
				<div class="text-6xl mb-4">🎬</div>
				<h3 class="text-xl font-medium text-text-primary mb-2">No Items Yet</h3>
				<p class="text-text-muted">Add movies or shows to this collection from their detail pages.</p>
			</div>
		{/if}
	</div>
{/if}

<!-- Edit Modal -->
{#if showEditModal && collection}
	<div
		class="fixed inset-0 bg-black/80 z-50 flex items-center justify-center p-4"
		onclick={(e) => { if (e.target === e.currentTarget) showEditModal = false; }}
		onkeydown={(e) => { if (e.key === 'Escape') showEditModal = false; }}
		role="dialog"
		tabindex="-1"
	>
		<div class="bg-bg-primary border border-border-subtle rounded-xl w-full max-w-md">
			<div class="p-6 border-b border-border-subtle">
				<h2 class="text-xl font-semibold text-text-primary">Edit Collection</h2>
			</div>
			<form onsubmit={handleSaveEdit} class="p-6 space-y-4">
				<div>
					<label for="editName" class="block text-sm font-medium text-text-secondary mb-2">Name</label>
					<input
						id="editName"
						type="text"
						bind:value={editName}
						required
						class="w-full px-4 py-2.5 rounded-lg bg-bg-secondary border border-border-subtle text-text-primary focus:outline-none focus:border-accent-primary"
					/>
				</div>
				<div>
					<label for="editDesc" class="block text-sm font-medium text-text-secondary mb-2">Description</label>
					<textarea
						id="editDesc"
						bind:value={editDescription}
						rows="3"
						class="w-full px-4 py-2.5 rounded-lg bg-bg-secondary border border-border-subtle text-text-primary focus:outline-none focus:border-accent-primary resize-none"
					></textarea>
				</div>
				<div class="flex gap-3 pt-2">
					<button
						type="button"
						onclick={() => showEditModal = false}
						class="flex-1 px-4 py-2.5 rounded-lg border border-border-subtle text-text-secondary hover:bg-bg-secondary transition-colors"
					>
						Cancel
					</button>
					<button
						type="submit"
						disabled={!editName.trim() || saving}
						class="flex-1 px-4 py-2.5 rounded-lg bg-accent-primary text-white font-medium hover:bg-accent-primary/90 transition-colors disabled:opacity-50"
					>
						{saving ? 'Saving...' : 'Save'}
					</button>
				</div>
			</form>
		</div>
	</div>
{/if}

<!-- Delete Confirmation -->
{#if confirmingDelete}
	<div
		class="fixed inset-0 bg-black/80 z-50 flex items-center justify-center p-4"
		onclick={(e) => { if (e.target === e.currentTarget) confirmingDelete = false; }}
		onkeydown={(e) => { if (e.key === 'Escape') confirmingDelete = false; }}
		role="dialog"
		tabindex="-1"
	>
		<div class="bg-bg-primary border border-border-subtle rounded-xl w-full max-w-sm p-6">
			<h2 class="text-xl font-semibold text-text-primary mb-2">Delete Collection?</h2>
			<p class="text-text-muted mb-6">This will permanently delete "{collection?.name}". Items in your library will not be affected.</p>
			<div class="flex gap-3">
				<button
					onclick={() => confirmingDelete = false}
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
