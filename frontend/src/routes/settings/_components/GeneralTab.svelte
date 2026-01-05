<script lang="ts">
	import Select from '$lib/components/ui/Select.svelte';
	import type { Library, ScanProgress } from '$lib/api';

	interface Props {
		// Settings state
		tmdbApiKey: string;
		savingSettings: boolean;
		settingsSaved: boolean;
		refreshingMetadata: boolean;
		refreshResult: { refreshed: number; errors: number; total: number } | null;
		clearingLibrary: boolean;
		showClearConfirm: boolean;
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
		onTmdbKeyChange: (value: string) => void;
		onSaveSettings: () => void;
		onRefreshMetadata: () => void;
		onClearLibrary: () => void;
		onShowClearConfirm: (show: boolean) => void;
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
		tmdbApiKey,
		savingSettings,
		settingsSaved,
		refreshingMetadata,
		refreshResult,
		clearingLibrary,
		showClearConfirm,
		libraries,
		loading,
		showAddForm,
		name,
		path,
		type,
		scanning,
		scanProgress,
		onTmdbKeyChange,
		onSaveSettings,
		onRefreshMetadata,
		onClearLibrary,
		onShowClearConfirm,
		onShowAddForm,
		onNameChange,
		onPathChange,
		onTypeChange,
		onAddLibrary,
		onDeleteLibrary,
		onScanLibrary,
		onBrowse
	}: Props = $props();

	const inputClass = "liquid-input w-full px-4 py-2.5";
	const labelClass = "block text-sm text-text-secondary mb-1.5";
</script>

<!-- TMDB Settings -->
<section class="glass-card p-6 space-y-4">
	<div class="flex items-center gap-3">
		<div class="w-10 h-10 rounded-xl bg-white-600/20 flex items-center justify-center">
			<svg class="w-5 h-5 text-white-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
				<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M7 4v16M17 4v16M3 8h4m10 0h4M3 12h18M3 16h4m10 0h4M4 20h16a1 1 0 001-1V5a1 1 0 00-1-1H4a1 1 0 00-1 1v14a1 1 0 001 1z" />
			</svg>
		</div>
		<div>
			<h2 class="text-lg font-semibold text-text-primary">Metadata</h2>
			<p class="text-sm text-text-secondary">Configure metadata sources for your library</p>
		</div>
	</div>

	<div class="pt-2">
		<label for="tmdb-api-key" class={labelClass}>TMDB API Key</label>
		<p class="text-xs text-text-muted mb-2">
			Get a free API key from <a href="https://www.themoviedb.org/settings/api" target="_blank" rel="noopener" class="text-white-400 hover:underline">themoviedb.org</a>
		</p>
		<div class="flex gap-3">
			<input
				type="password"
				id="tmdb-api-key"
				value={tmdbApiKey}
				oninput={(e) => onTmdbKeyChange((e.target as HTMLInputElement).value)}
				class="{inputClass} flex-1"
				placeholder="Enter your TMDB API key"
			/>
			<button
				class="liquid-btn disabled:opacity-50"
				onclick={onSaveSettings}
				disabled={savingSettings}
			>
				{savingSettings ? 'Saving...' : 'Save'}
			</button>
		</div>
		{#if settingsSaved}
			<p class="text-green-400 text-sm mt-2 flex items-center gap-2">
				<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
				</svg>
				Settings saved! Metadata will be fetched during next scan.
			</p>
		{/if}
	</div>

	<div class="pt-4 border-t border-white/10">
		<label class={labelClass}>Refresh All Metadata</label>
		<p class="text-xs text-text-muted mb-2">
			Re-fetch metadata from TMDB for all movies and TV shows in your library
		</p>
		<button
			class="liquid-btn disabled:opacity-50"
			onclick={onRefreshMetadata}
			disabled={refreshingMetadata}
		>
			{#if refreshingMetadata}
				<span class="flex items-center gap-2">
					<svg class="w-4 h-4 animate-spin" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
					</svg>
					Refreshing...
				</span>
			{:else}
				Refresh All Metadata
			{/if}
		</button>
		{#if refreshResult}
			<p class="text-sm mt-2 flex items-center gap-2 {refreshResult.errors > 0 ? 'text-yellow-400' : 'text-green-400'}">
				<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
				</svg>
				Refreshed {refreshResult.refreshed} of {refreshResult.total} items
				{#if refreshResult.errors > 0}
					({refreshResult.errors} errors)
				{/if}
			</p>
		{/if}
	</div>

	<div class="pt-4 border-t border-white/10">
		<label class={labelClass}>Clear Library Data</label>
		<p class="text-xs text-text-muted mb-2">
			Remove all movies, TV shows, and watch progress. Library folders will be kept but all scanned media will be deleted.
		</p>
		{#if showClearConfirm}
			<div class="flex items-center gap-3">
				<span class="text-sm text-yellow-400">Are you sure? This cannot be undone.</span>
				<button
					class="liquid-btn !bg-red-600 hover:!bg-red-700 disabled:opacity-50"
					onclick={onClearLibrary}
					disabled={clearingLibrary}
				>
					{clearingLibrary ? 'Clearing...' : 'Yes, Clear All'}
				</button>
				<button
					class="liquid-btn"
					onclick={() => onShowClearConfirm(false)}
					disabled={clearingLibrary}
				>
					Cancel
				</button>
			</div>
		{:else}
			<button
				class="liquid-btn !bg-red-600/20 !text-red-400 hover:!bg-red-600/30"
				onclick={() => onShowClearConfirm(true)}
			>
				Clear All Library Data
			</button>
		{/if}
	</div>
</section>

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
							class="liquid-btn-sm !bg-white/5 !border-t-white/10 text-text-secondary hover:text-white"
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
