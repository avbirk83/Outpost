<script lang="ts">
	import { page } from '$app/stores';
	import { onMount } from 'svelte';
	import { getBook, type Book } from '$lib/api';

	let book: Book | null = $state(null);
	let loading = $state(true);
	let error: string | null = $state(null);

	onMount(async () => {
		const id = parseInt($page.params.id);
		try {
			book = await getBook(id);
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to load book';
		} finally {
			loading = false;
		}
	});

	function formatSize(bytes: number): string {
		if (bytes < 1024) return `${bytes} B`;
		if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
		return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
	}

	function formatDate(dateStr: string): string {
		try {
			const date = new Date(dateStr);
			return date.toLocaleDateString();
		} catch {
			return dateStr;
		}
	}

	function getFormatColor(format: string): string {
		switch (format.toLowerCase()) {
			case 'epub':
				return 'bg-green-900 text-green-300';
			case 'pdf':
				return 'bg-white/20 text-white';
			case 'mobi':
			case 'azw':
			case 'azw3':
				return 'bg-yellow-900 text-yellow-300';
			case 'cbz':
			case 'cbr':
				return 'bg-purple-900 text-purple-300';
			default:
				return 'bg-gray-700 text-gray-300';
		}
	}
</script>

<svelte:head>
	<title>{book?.title || 'Book'} - Outpost</title>
</svelte:head>

{#if loading}
	<p class="text-gray-400">Loading book...</p>
{:else if error}
	<div class="bg-glass border border-border-subtle text-text-secondary px-4 py-3 rounded">
		{error}
	</div>
{:else if book}
	<div class="flex flex-col md:flex-row gap-8">
		<div class="flex-shrink-0">
			<div class="w-64 aspect-[2/3] bg-gray-800 rounded-lg overflow-hidden">
				{#if book.coverPath}
					<img
						src="/images/{book.coverPath}"
						alt={book.title}
						class="w-full h-full object-cover"
					/>
				{:else}
					<div class="w-full h-full flex items-center justify-center text-gray-600">
						<div class="text-center">
							<svg class="w-16 h-16 mx-auto mb-2" fill="currentColor" viewBox="0 0 24 24">
								<path d="M18 2H6c-1.1 0-2 .9-2 2v16c0 1.1.9 2 2 2h12c1.1 0 2-.9 2-2V4c0-1.1-.9-2-2-2zM6 4h5v8l-2.5-1.5L6 12V4z"/>
							</svg>
							<span>{book.format.toUpperCase()}</span>
						</div>
					</div>
				{/if}
			</div>
			<div class="mt-4 space-y-2">
				<a
					href="/api/stream/book/{book.id}"
					download
					class="block w-full text-center bg-blue-600 hover:bg-blue-700 text-white py-2 px-4 rounded"
				>
					Download
				</a>
				{#if book.format === 'pdf'}
					<a
						href="/api/stream/book/{book.id}"
						target="_blank"
						class="block w-full text-center bg-gray-700 hover:bg-gray-600 text-white py-2 px-4 rounded"
					>
						Open in Browser
					</a>
				{/if}
			</div>
		</div>

		<div class="flex-1 space-y-6">
			<div>
				<span class="px-2 py-0.5 text-xs rounded {getFormatColor(book.format)}">
					{book.format.toUpperCase()}
				</span>
				<h1 class="text-3xl font-bold mt-2">{book.title}</h1>
				{#if book.author}
					<p class="text-xl text-gray-400 mt-1">{book.author}</p>
				{/if}
			</div>

			<div class="grid grid-cols-2 gap-4 text-sm">
				{#if book.year}
					<div>
						<span class="text-gray-500">Year</span>
						<p class="text-gray-200">{book.year}</p>
					</div>
				{/if}
				{#if book.publisher}
					<div>
						<span class="text-gray-500">Publisher</span>
						<p class="text-gray-200">{book.publisher}</p>
					</div>
				{/if}
				{#if book.isbn}
					<div>
						<span class="text-gray-500">ISBN</span>
						<p class="text-gray-200">{book.isbn}</p>
					</div>
				{/if}
				<div>
					<span class="text-gray-500">Size</span>
					<p class="text-gray-200">{formatSize(book.size)}</p>
				</div>
				<div>
					<span class="text-gray-500">Added</span>
					<p class="text-gray-200">{formatDate(book.addedAt)}</p>
				</div>
			</div>

			{#if book.description}
				<div>
					<h2 class="text-lg font-semibold mb-2">Description</h2>
					<p class="text-gray-300">{book.description}</p>
				</div>
			{/if}
		</div>
	</div>
{/if}
