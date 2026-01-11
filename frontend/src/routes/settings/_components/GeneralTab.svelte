<script lang="ts">
	import Select from '$lib/components/ui/Select.svelte';
	import { clearLibraryData, type Library, type ScanProgress } from '$lib/api';
	import { toast } from '$lib/stores/toast';

	interface Props {
		// Library state
		libraries: Library[];
		loading: boolean;
		showAddForm: boolean;
		name: string;
		path: string;
		type: Library['type'];
		scanning: Record<number, boolean>;
		scanProgress: ScanProgress | null;
		// Callbacks
		onShowAddForm: (show: boolean) => void;
		onNameChange: (value: string) => void;
		onPathChange: (value: string) => void;
		onTypeChange: (value: Library['type']) => void;
		onAddLibrary: () => void;
		onDeleteLibrary: (id: number) => void;
		onScanLibrary: (id: number) => void;
		onBrowse: () => void;
	}

	let {
		libraries,
		loading,
		showAddForm,
		name,
		path,
		type,
		scanning,
		scanProgress,
		onShowAddForm,
		onNameChange,
		onPathChange,
		onTypeChange,
		onAddLibrary,
		onDeleteLibrary,
		onScanLibrary,
		onBrowse
	}: Props = $props();

	// Clear library state
	let clearingLibrary = $state(false);
	let showClearConfirm = $state(false);

	async function handleClearLibrary() {
		try {
			clearingLibrary = true;
			await clearLibraryData();
			showClearConfirm = false;
			toast.success('Library data cleared');
			window.location.reload();
		} catch (e) {
			console.error('Failed to clear library data:', e);
			toast.error('Failed to clear library data');
		} finally {
			clearingLibrary = false;
		}
	}

	const inputClass = "liquid-input w-full px-4 py-2.5";
	const labelClass = "block text-sm text-text-secondary mb-1.5";
</script>

<!-- Libraries -->
<section class="glass-card p-6 space-y-4">
	<div class="flex items-center justify-between">
		<div class="flex items-center gap-3">
			<div class="w-10 h-10 rounded-xl bg-blue-600/20 flex items-center justify-center">
				<svg class="w-5 h-5 text-blue-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 7v10a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-6l-2-2H5a2 2 0 00-2 2z" />
				</svg>
			</div>
			<div>
				<h2 class="text-lg font-semibold text-text-primary">Libraries</h2>
				<p class="text-sm text-text-secondary">Manage your media folders</p>
			</div>
		</div>
		<button
			class="liquid-btn-sm"
			onclick={() => onShowAddForm(!showAddForm)}
		>
			{showAddForm ? 'Cancel' : 'Add Library'}
		</button>
	</div>

	{#if showAddForm}
		<form
			class="p-4 bg-bg-elevated/50 rounded-xl space-y-4 border border-white/5"
			onsubmit={(e) => {
				e.preventDefault();
				onAddLibrary();
			}}
		>
			<div class="grid sm:grid-cols-2 gap-4">
				<div>
					<label for="lib-name" class={labelClass}>Name</label>
					<input
						type="text"
						id="lib-name"
						value={name}
						oninput={(e) => onNameChange((e.target as HTMLInputElement).value)}
						required
						class={inputClass}
						placeholder="Movies"
					/>
				</div>
				<div>
					<label for="lib-type" class={labelClass}>Type</label>
					<Select
						id="lib-type"
						value={type}
						onchange={(val) => onTypeChange(val as Library['type'])}
						options={[
							{ value: 'movies', label: 'Movies' },
							{ value: 'tv', label: 'TV Shows' },
							{ value: 'anime', label: 'Anime' },
							{ value: 'music', label: 'Music' },
							{ value: 'books', label: 'Books' }
						]}
					/>
				</div>
			</div>
			<div>
				<label for="lib-path" class={labelClass}>Path</label>
				<div class="flex gap-2">
					<input
						type="text"
						id="lib-path"
						value={path}
						oninput={(e) => onPathChange((e.target as HTMLInputElement).value)}
						required
						class="{inputClass} flex-1"
						placeholder="/media/movies"
					/>
					<button
						type="button"
						onclick={onBrowse}
						class="px-3 py-2 bg-white/5 hover:bg-white/10 border border-white/10 rounded-lg text-text-secondary hover:text-text-primary transition-colors"
						title="Browse directories"
					>
						<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 7v10a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-6l-2-2H5a2 2 0 00-2 2z" />
						</svg>
					</button>
				</div>
			</div>
			<button type="submit" class="liquid-btn">
				Add Library
			</button>
		</form>
	{/if}

	{#if loading}
		<div class="flex items-center gap-3 py-4">
			<div class="spinner-md text-cream"></div>
			<span class="text-text-secondary">Loading libraries...</span>
		</div>
	{:else if libraries.length === 0}
		<p class="text-text-muted py-4">No libraries configured. Add one to get started.</p>
	{:else}
		<div class="space-y-2">
			{#each libraries as lib}
				<div class="p-4 bg-bg-elevated/50 rounded-xl flex items-center justify-between border border-white/5">
					<div>
						<h3 class="font-medium text-text-primary">{lib.name}</h3>
						<p class="text-sm text-text-secondary">{lib.path}</p>
						<span class="inline-block mt-1 px-2 py-0.5 text-xs rounded-lg bg-bg-card text-text-muted capitalize">
							{lib.type}
						</span>
					</div>
					<div class="flex gap-2">
						<button
							class="liquid-btn-sm disabled:opacity-50"
							onclick={() => onScanLibrary(lib.id)}
							disabled={scanning[lib.id] || scanProgress?.scanning}
						>
							{scanning[lib.id] || (scanProgress?.scanning && scanProgress.library === lib.name) ? 'Scanning...' : 'Scan'}
						</button>
						<button
							class="liquid-btn-sm !bg-white/5 !border-t-white/10 text-text-secondary hover:text-text-primary"
							onclick={() => onDeleteLibrary(lib.id)}
						>
							Delete
						</button>
					</div>
				</div>
			{/each}
		</div>
	{/if}

	<!-- Scan Progress Bar -->
	{#if scanProgress?.scanning}
		<div class="p-4 bg-bg-elevated/50 rounded-xl border border-white/5 space-y-3">
			<div class="flex items-center justify-between">
				<div class="flex items-center gap-3">
					<div class="spinner-md text-blue-400"></div>
					<div>
						<span class="text-text-primary font-medium">
							{scanProgress.phase === 'counting' ? 'Counting files...' :
							 scanProgress.phase === 'extracting' ? 'Extracting subtitles...' :
							 `Scanning ${scanProgress.library}`}
						</span>
						{#if scanProgress.phase !== 'counting' && scanProgress.total > 0}
							<span class="text-text-muted ml-2 text-sm">
								{scanProgress.current} / {scanProgress.total}
							</span>
						{/if}
					</div>
				</div>
				{#if scanProgress.percent > 0}
					<span class="text-text-secondary text-sm font-medium">{scanProgress.percent}%</span>
				{/if}
			</div>
			{#if scanProgress.total > 0}
				<div class="w-full bg-bg-card rounded-full h-2 overflow-hidden">
					<div
						class="bg-blue-500 h-full transition-all duration-300 ease-out"
						style="width: {scanProgress.percent}%"
					></div>
				</div>
			{/if}
		</div>
	{:else if scanProgress?.lastLibrary}
		<!-- Last Scan Result -->
		<div class="p-4 bg-bg-elevated/50 rounded-xl border border-white/5">
			<div class="flex items-center justify-between">
				<div class="flex items-center gap-3">
					<svg class="w-5 h-5 text-green-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
					</svg>
					<div>
						<span class="text-text-primary font-medium">
							Scan complete: {scanProgress.lastLibrary}
						</span>
						<div class="text-sm text-text-secondary mt-0.5">
							<span class="text-green-400">{scanProgress.lastAdded} added</span>
							{#if scanProgress.lastSkipped > 0}
								<span class="mx-1">·</span>
								<span>{scanProgress.lastSkipped} skipped</span>
							{/if}
							{#if scanProgress.lastErrors > 0}
								<span class="mx-1">·</span>
								<span class="text-red-400">{scanProgress.lastErrors} errors</span>
							{/if}
						</div>
					</div>
				</div>
			</div>
		</div>
	{/if}
</section>

<!-- Data Management -->
<section class="glass-card p-6 space-y-4">
	<div class="flex items-center gap-3">
		<div class="w-10 h-10 rounded-xl bg-red-600/20 flex items-center justify-center">
			<svg class="w-5 h-5 text-red-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
				<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
			</svg>
		</div>
		<div>
			<h2 class="text-lg font-semibold text-text-primary">Data Management</h2>
			<p class="text-sm text-text-secondary">Manage your library data</p>
		</div>
	</div>

	<div>
		<label class={labelClass}>Clear Library Data</label>
		<p class="text-xs text-text-muted mb-2">
			Remove all movies, TV shows, and watch progress. Library folders will be kept but all scanned media will be deleted.
		</p>
		{#if showClearConfirm}
			<div class="flex items-center gap-3">
				<span class="text-sm text-yellow-400">Are you sure? This cannot be undone.</span>
				<button
					class="liquid-btn !bg-red-600 hover:!bg-red-700 disabled:opacity-50"
					onclick={handleClearLibrary}
					disabled={clearingLibrary}
				>
					{clearingLibrary ? 'Clearing...' : 'Yes, Clear All'}
				</button>
				<button
					class="liquid-btn"
					onclick={() => showClearConfirm = false}
					disabled={clearingLibrary}
				>
					Cancel
				</button>
			</div>
		{:else}
			<button
				class="liquid-btn !bg-red-600/20 !text-red-400 hover:!bg-red-600/30"
				onclick={() => showClearConfirm = true}
			>
				Clear All Library Data
			</button>
		{/if}
	</div>
</section>
