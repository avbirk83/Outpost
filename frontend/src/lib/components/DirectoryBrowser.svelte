<script lang="ts">
	interface Props {
		open: boolean;
		currentPath: string;
		onSelect: (path: string) => void;
	}

	let { open = $bindable(), currentPath = $bindable(), onSelect }: Props = $props();

	interface DirEntry {
		name: string;
		path: string;
		isDir: boolean;
	}

	interface BrowseResponse {
		current: string;
		parent: string;
		dirs: DirEntry[];
	}

	let loading = $state(false);
	let error: string | null = $state(null);
	let browsePath = $state('/');
	let dirs: DirEntry[] = $state([]);
	let parent = $state('');

	// Load directory when modal opens
	$effect(() => {
		if (open) {
			const startPath = currentPath || '/';
			browsePath = startPath;
			loadDirectory(startPath);
		}
	});

	async function loadDirectory(path: string) {
		loading = true;
		error = null;

		try {
			const response = await fetch(`/api/filesystem/browse?path=${encodeURIComponent(path)}`);
			if (!response.ok) {
				throw new Error(await response.text());
			}
			const data: BrowseResponse = await response.json();
			browsePath = data.current;
			parent = data.parent;
			dirs = data.dirs || [];
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to load directory';
			dirs = [];
		} finally {
			loading = false;
		}
	}

	function handleNavigate(path: string) {
		loadDirectory(path);
	}

	function handleGoUp() {
		if (parent) {
			loadDirectory(parent);
		}
	}

	function handleSelectCurrent() {
		onSelect(browsePath);
		open = false;
	}

	function handleClose() {
		open = false;
	}

	function handleBackdropClick(e: MouseEvent) {
		if (e.target === e.currentTarget) {
			open = false;
		}
	}

	function handleKeydown(e: KeyboardEvent) {
		if (open && e.key === 'Escape') {
			open = false;
		}
	}

	function stopPropagation(e: MouseEvent) {
		e.stopPropagation();
	}
</script>

<svelte:window onkeydown={handleKeydown} />

{#if open}
	<!-- svelte-ignore a11y_click_events_have_key_events -->
	<!-- svelte-ignore a11y_no_static_element_interactions -->
	<div
		class="modal-overlay z-[600]"
		onclick={handleBackdropClick}
	>
		<!-- svelte-ignore a11y_no_static_element_interactions -->
		<div
			class="modal-container max-w-xl"
			onclick={stopPropagation}
		>
			<!-- Header -->
			<div class="flex items-center justify-between p-4 border-b border-white/10">
				<h2 class="text-lg font-semibold text-text-primary">Select Directory</h2>
				<button
					type="button"
					onclick={handleClose}
					class="p-2 rounded-lg hover:bg-white/10 text-text-muted hover:text-text-primary transition-colors"
					aria-label="Close"
				>
					<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
					</svg>
				</button>
			</div>

			<!-- Current path bar -->
			<div class="px-4 py-3 bg-bg-base/50 border-b border-white/10">
				<div class="flex items-center gap-2">
					<button
						type="button"
						onclick={handleGoUp}
						disabled={!parent}
						class="p-1.5 rounded-lg hover:bg-white/10 text-text-muted hover:text-text-primary transition-colors disabled:opacity-30 disabled:cursor-not-allowed"
						aria-label="Go up"
					>
						<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 10l7-7m0 0l7 7m-7-7v18" />
						</svg>
					</button>
					<div class="flex-1 px-3 py-2 bg-bg-base rounded-lg border border-white/10 text-sm text-text-primary font-mono truncate">
						{browsePath}
					</div>
				</div>
			</div>

			<!-- Directory list -->
			<div class="max-h-80 overflow-y-auto scrollbar-thin scrollbar-track-transparent scrollbar-thumb-white/20 hover:scrollbar-thumb-white/30">
				{#if loading}
					<div class="flex items-center justify-center py-12">
						<div class="flex items-center gap-3 text-text-muted">
							<div class="spinner-md text-text-muted"></div>
							<span>Loading...</span>
						</div>
					</div>
				{:else if error}
					<div class="flex items-center justify-center py-12">
						<div class="text-center text-text-muted">
							<svg class="w-10 h-10 mx-auto mb-2 text-red-500/50" fill="none" stroke="currentColor" viewBox="0 0 24 24">
								<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
							</svg>
							<p class="text-sm">{error}</p>
						</div>
					</div>
				{:else if dirs.length === 0}
					<div class="flex items-center justify-center py-12">
						<div class="text-center text-text-muted">
							<svg class="w-10 h-10 mx-auto mb-2 text-text-muted/50" fill="none" stroke="currentColor" viewBox="0 0 24 24">
								<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 7v10a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-6l-2-2H5a2 2 0 00-2 2z" />
							</svg>
							<p class="text-sm">No subdirectories</p>
						</div>
					</div>
				{:else}
					<div class="p-2">
						{#each dirs as dir (dir.path)}
							<button
								type="button"
								onclick={() => handleNavigate(dir.path)}
								class="w-full flex items-center gap-3 px-3 py-2.5 rounded-lg hover:bg-white/10 transition-colors text-left group"
							>
								<svg class="w-5 h-5 text-amber-500 flex-shrink-0" fill="currentColor" viewBox="0 0 24 24">
									<path d="M10 4H4c-1.11 0-2 .89-2 2v12c0 1.11.89 2 2 2h16c1.11 0 2-.89 2-2V8c0-1.11-.89-2-2-2h-8l-2-2z" />
								</svg>
								<span class="text-sm text-text-primary group-hover:text-white truncate">{dir.name}</span>
							</button>
						{/each}
					</div>
				{/if}
			</div>

			<!-- Footer -->
			<div class="flex items-center justify-end gap-3 p-4 border-t border-white/10 bg-black/30">
				<button
					type="button"
					onclick={handleClose}
					class="liquid-btn-sm !bg-white/5 text-text-secondary hover:text-white"
				>
					Cancel
				</button>
				<button
					type="button"
					onclick={handleSelectCurrent}
					class="liquid-btn-sm"
				>
					Select This Folder
				</button>
			</div>
		</div>
	</div>
{/if}

<style>
	/* Custom scrollbar styling */
	.scrollbar-thin::-webkit-scrollbar {
		width: 8px;
	}
	.scrollbar-thin::-webkit-scrollbar-track {
		background: transparent;
	}
	.scrollbar-thin::-webkit-scrollbar-thumb {
		background: rgba(255, 255, 255, 0.2);
		border-radius: 4px;
	}
	.scrollbar-thin::-webkit-scrollbar-thumb:hover {
		background: rgba(255, 255, 255, 0.3);
	}
</style>
