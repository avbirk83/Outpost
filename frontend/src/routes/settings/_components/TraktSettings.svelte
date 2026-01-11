<script lang="ts">
	import { onMount } from 'svelte';
	import { getTraktConfig, updateTraktConfig, disconnectTrakt, syncTrakt, getTraktAuthURL, formatLastSynced, type TraktConfig } from '$lib/api';

	// Trakt settings state
	let traktConfig = $state<TraktConfig | null>(null);
	let savingTrakt = $state(false);
	let traktSaved = $state(false);
	let syncingTrakt = $state(false);
	let traktSyncResult = $state<{ success: boolean; message: string } | null>(null);
	let connectingTrakt = $state(false);

	onMount(async () => {
		await loadTraktConfig();
	});

	async function loadTraktConfig() {
		try {
			traktConfig = await getTraktConfig();
		} catch (e) {
			console.error('Failed to load Trakt config:', e);
		}
	}

	async function handleConnectTrakt() {
		connectingTrakt = true;
		try {
			const redirectUri = `${window.location.origin}/settings/trakt-callback`;
			const result = await getTraktAuthURL(redirectUri);
			sessionStorage.setItem('trakt_return_url', window.location.href);
			window.location.href = result.url;
		} catch (e) {
			console.error('Failed to get Trakt auth URL:', e);
			connectingTrakt = false;
		}
	}

	async function handleDisconnectTrakt() {
		if (!confirm('Are you sure you want to disconnect your Trakt account?')) return;
		try {
			await disconnectTrakt();
			traktConfig = { connected: false, syncEnabled: false, syncWatched: false, syncRatings: false, syncWatchlist: false };
		} catch (e) {
			console.error('Failed to disconnect Trakt:', e);
		}
	}

	async function handleSaveTraktSettings() {
		if (!traktConfig) return;
		savingTrakt = true;
		try {
			await updateTraktConfig({
				syncEnabled: traktConfig.syncEnabled,
				syncWatched: traktConfig.syncWatched,
				syncRatings: traktConfig.syncRatings,
				syncWatchlist: traktConfig.syncWatchlist
			});
			traktSaved = true;
			setTimeout(() => traktSaved = false, 3000);
		} catch (e) {
			console.error('Failed to save Trakt settings:', e);
		} finally {
			savingTrakt = false;
		}
	}

	async function handleSyncTrakt() {
		syncingTrakt = true;
		traktSyncResult = null;
		try {
			const result = await syncTrakt();
			const pulled = result.pulled.movies + result.pulled.shows;
			const pushed = result.pushed.movies + result.pushed.episodes;
			traktSyncResult = {
				success: result.success,
				message: result.success
					? `Synced! Pulled ${pulled} items, pushed ${pushed} items`
					: result.errors.join(', ')
			};
			await loadTraktConfig();
		} catch (e) {
			traktSyncResult = {
				success: false,
				message: e instanceof Error ? e.message : 'Sync failed'
			};
		} finally {
			syncingTrakt = false;
		}
	}
</script>

<section class="glass-card p-6 space-y-4">
	<div class="flex items-center gap-3">
		<div class="w-10 h-10 rounded-xl flex items-center justify-center overflow-hidden">
			<img src="/icons/trakt.svg" alt="Trakt" class="w-full h-full object-cover" />
		</div>
		<div>
			<h2 class="text-lg font-semibold text-text-primary">Trakt.tv</h2>
			<p class="text-sm text-text-secondary">Sync your watch history and ratings with Trakt</p>
		</div>
	</div>

	{#if traktConfig?.connected}
		<div class="space-y-4">
			<div class="flex items-center justify-between p-3 rounded-lg bg-green-500/10 border border-green-500/30">
				<div class="flex items-center gap-3">
					<svg class="w-5 h-5 text-green-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
					</svg>
					<div>
						<p class="text-sm text-green-300">Connected as <span class="font-semibold">{traktConfig.username}</span></p>
						<p class="text-xs text-text-muted">Last synced: {formatLastSynced(traktConfig.lastSyncedAt)}</p>
					</div>
				</div>
				<button
					onclick={handleDisconnectTrakt}
					class="text-xs text-red-400 hover:text-red-300 transition-colors"
				>
					Disconnect
				</button>
			</div>

			<div class="space-y-3">
				<label class="flex items-center gap-3 cursor-pointer">
					<input
						type="checkbox"
						bind:checked={traktConfig.syncEnabled}
						class="w-4 h-4 accent-red-500"
					/>
					<div>
						<span class="text-text-primary">Enable Sync</span>
						<p class="text-xs text-text-muted">Automatically sync with Trakt</p>
					</div>
				</label>

				{#if traktConfig.syncEnabled}
					<div class="ml-7 space-y-2">
						<label class="flex items-center gap-2 cursor-pointer">
							<input type="checkbox" bind:checked={traktConfig.syncWatched} class="w-4 h-4 accent-red-500" />
							<span class="text-sm text-text-secondary">Sync watch history</span>
						</label>
						<label class="flex items-center gap-2 cursor-pointer">
							<input type="checkbox" bind:checked={traktConfig.syncRatings} class="w-4 h-4 accent-red-500" />
							<span class="text-sm text-text-secondary">Sync ratings</span>
						</label>
						<label class="flex items-center gap-2 cursor-pointer">
							<input type="checkbox" bind:checked={traktConfig.syncWatchlist} class="w-4 h-4 accent-red-500" />
							<span class="text-sm text-text-secondary">Sync watchlist</span>
						</label>
					</div>
				{/if}
			</div>

			<div class="flex items-center gap-3 pt-2">
				<button class="liquid-btn" onclick={handleSaveTraktSettings} disabled={savingTrakt}>
					{savingTrakt ? 'Saving...' : 'Save Settings'}
				</button>
				<button
					onclick={handleSyncTrakt}
					disabled={syncingTrakt || !traktConfig.syncEnabled}
					class="px-4 py-2 text-sm rounded-lg bg-bg-elevated hover:bg-bg-card border border-border-subtle text-text-secondary hover:text-text-primary disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
				>
					{syncingTrakt ? 'Syncing...' : 'Sync Now'}
				</button>
				{#if traktSaved}
					<span class="text-sm text-green-400 flex items-center gap-1">
						<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
						</svg>
						Saved!
					</span>
				{/if}
			</div>
			{#if traktSyncResult}
				<p class="text-xs {traktSyncResult.success ? 'text-green-400' : 'text-red-400'}">
					{traktSyncResult.message}
				</p>
			{/if}
		</div>
	{:else}
		<div class="text-center py-6">
			<p class="text-text-secondary mb-4">Connect your Trakt account to sync your watch history across devices</p>
			<button
				onclick={handleConnectTrakt}
				disabled={connectingTrakt}
				class="liquid-btn"
			>
				{connectingTrakt ? 'Connecting...' : 'Connect to Trakt'}
			</button>
		</div>
	{/if}
</section>
