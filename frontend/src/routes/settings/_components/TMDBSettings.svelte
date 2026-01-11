<script lang="ts">
	import { onMount } from 'svelte';
	import { getSettings, saveSettings, refreshAllMetadata } from '$lib/api';
	import { toast } from '$lib/stores/toast';

	// TMDB settings state
	let tmdbApiKey = $state('');
	let savingSettings = $state(false);
	let settingsSaved = $state(false);
	let refreshingMetadata = $state(false);
	let refreshResult = $state<{ refreshed: number; errors: number; total: number } | null>(null);

	onMount(async () => {
		await loadSettings();
	});

	async function loadSettings() {
		try {
			const settings = await getSettings();
			tmdbApiKey = settings['tmdb_api_key'] || '';
		} catch (e) {
			console.error('Failed to load TMDB settings:', e);
		}
	}

	async function handleSaveSettings() {
		try {
			savingSettings = true;
			await saveSettings({ tmdb_api_key: tmdbApiKey });
			settingsSaved = true;
			setTimeout(() => settingsSaved = false, 3000);
			toast.success('TMDB settings saved');
		} catch (e) {
			console.error('Failed to save TMDB settings:', e);
			toast.error('Failed to save settings');
		} finally {
			savingSettings = false;
		}
	}

	async function handleRefreshMetadata() {
		try {
			refreshingMetadata = true;
			refreshResult = null;
			const result = await refreshAllMetadata();
			refreshResult = result;
			setTimeout(() => refreshResult = null, 10000);
			toast.success(`Refreshed ${result.refreshed} items`);
		} catch (e) {
			console.error('Failed to refresh metadata:', e);
			toast.error('Failed to refresh metadata');
		} finally {
			refreshingMetadata = false;
		}
	}
</script>

<section class="glass-card p-6 space-y-4">
	<div class="flex items-center gap-3">
		<div class="w-10 h-10 rounded-xl flex items-center justify-center overflow-hidden bg-[#0d253f] p-1.5">
			<img src="/icons/tmdb.svg" alt="TMDB" class="w-full h-full object-contain" />
		</div>
		<div>
			<h2 class="text-lg font-semibold text-text-primary">TMDB</h2>
			<p class="text-sm text-text-secondary">The Movie Database for metadata and artwork</p>
		</div>
	</div>

	<div class="space-y-4">
		<!-- API Key -->
		<div>
			<label class="block text-sm text-text-secondary mb-1">API Key</label>
			<p class="text-xs text-text-muted mb-2">Get your free API key from <a href="https://www.themoviedb.org/settings/api" target="_blank" rel="noopener noreferrer" class="text-cyan-400 hover:text-cyan-300">themoviedb.org</a></p>
			<div class="flex gap-2">
				<input
					type="password"
					bind:value={tmdbApiKey}
					placeholder="Enter your TMDB API key..."
					class="flex-1 px-3 py-2 text-sm bg-bg-elevated border border-border-subtle rounded-lg text-text-primary placeholder:text-text-muted focus:outline-none focus:border-cream/50"
				/>
			</div>
		</div>

		<!-- Save and Refresh buttons -->
		<div class="flex items-center gap-3 pt-2">
			<button class="liquid-btn" onclick={handleSaveSettings} disabled={savingSettings}>
				{savingSettings ? 'Saving...' : 'Save TMDB Settings'}
			</button>
			<button
				class="px-4 py-2 text-sm rounded-lg bg-bg-elevated hover:bg-bg-card border border-border-subtle text-text-secondary hover:text-text-primary disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
				onclick={handleRefreshMetadata}
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
			{#if settingsSaved}
				<span class="text-sm text-green-400 flex items-center gap-1">
					<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
					</svg>
					Saved!
				</span>
			{/if}
		</div>
		{#if refreshResult}
			<p class="text-sm flex items-center gap-2 {refreshResult.errors > 0 ? 'text-yellow-400' : 'text-green-400'}">
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
</section>
