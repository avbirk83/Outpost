<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import Select from '$lib/components/ui/Select.svelte';
	import {
		getLibraries,
		createLibrary,
		deleteLibrary,
		scanLibrary,
		getScanProgress,
		getSettings,
		saveSettings,
		refreshAllMetadata,
		clearLibraryData,
		getDownloadClients,
		createDownloadClient,
		updateDownloadClient,
		deleteDownloadClient,
		testDownloadClient,
		getIndexers,
		createIndexer,
		updateIndexer,
		deleteIndexer,
		testIndexer,
		getQualityPresets,
		createQualityPreset,
		updateQualityPreset,
		deleteQualityPreset,
		setDefaultQualityPreset,
		getStorageStatus,
		type Library,
		type DownloadClient,
		type Indexer,
		type QualityPreset,
		type ScanProgress,
		type StorageStatus
	} from '$lib/api';

	let libraries: Library[] = $state([]);
	let downloadClients: DownloadClient[] = $state([]);
	let indexers: Indexer[] = $state([]);
	let qualityPresets: QualityPreset[] = $state([]);
	let storageStatus: StorageStatus | null = $state(null);
	let loading = $state(true);
	let error: string | null = $state(null);
	let showAddForm = $state(false);
	let showAddClientForm = $state(false);
	let showAddIndexerForm = $state(false);
	let showAddPresetForm = $state(false);
	let editingPreset: QualityPreset | null = $state(null);
	let scanning: Record<number, boolean> = $state({});
	let testing: Record<number, boolean> = $state({});
	let testingIndexer: Record<number, boolean> = $state({});
	let testResults: Record<number, { success: boolean; message: string }> = $state({});
	let indexerTestResults: Record<number, { success: boolean; message: string }> = $state({});
	let scanProgress: ScanProgress | null = $state(null);
	let progressInterval: ReturnType<typeof setInterval> | null = null;

	// Form state
	let name = $state('');
	let path = $state('');
	let type: Library['type'] = $state('movies');

	// Download client form state
	let clientName = $state('');
	let clientType: DownloadClient['type'] = $state('qbittorrent');
	let clientHost = $state('');
	let clientPort = $state(8080);
	let clientUsername = $state('');
	let clientPassword = $state('');
	let clientApiKey = $state('');
	let clientUseTls = $state(false);
	let clientCategory = $state('');

	// Indexer form state
	let indexerName = $state('');
	let indexerType: Indexer['type'] = $state('torznab');
	let indexerUrl = $state('');
	let indexerApiKey = $state('');
	let indexerCategories = $state('');

	// Quality preset form state
	let presetName = $state('');
	let presetResolution = $state('1080p');
	let presetSource = $state('web');
	let presetHdrFormats: string[] = $state([]);
	let presetCodec = $state('any');
	let presetAudioFormats: string[] = $state([]);
	let presetPreferredEdition = $state('any');
	let presetMinSeeders = $state(3);
	let presetPreferSeasonPacks = $state(true);
	let presetAutoUpgrade = $state(true);

	// Preset options
	const resolutionOptions = ['4k', '1080p', '720p', '480p'];
	const sourceOptions = ['remux', 'bluray', 'web', 'any'];
	const hdrOptions = ['dv', 'hdr10+', 'hdr10', 'hlg'];
	const audioOptions = ['atmos', 'truehd', 'dtshd', 'dtsx', 'dd+'];
	const codecOptions = ['any', 'hevc', 'av1'];
	const editionOptions = ['any', 'theatrical', 'directors', 'extended', 'unrated'];

	// Settings state
	let tmdbApiKey = $state('');
	let savingSettings = $state(false);
	let settingsSaved = $state(false);
	let refreshingMetadata = $state(false);
	let refreshResult: { refreshed: number; errors: number; total: number } | null = $state(null);
	let clearingLibrary = $state(false);
	let showClearConfirm = $state(false);

	// Tab navigation
	type SettingsTab = 'general' | 'quality' | 'sources' | 'automation';
	let currentTab: SettingsTab = $state('general');

	const tabs: { id: SettingsTab; label: string; icon: string }[] = [
		{ id: 'general', label: 'General', icon: 'M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z M15 12a3 3 0 11-6 0 3 3 0 016 0z' },
		{ id: 'quality', label: 'Quality', icon: 'M9 12l2 2 4-4M7.835 4.697a3.42 3.42 0 001.946-.806 3.42 3.42 0 014.438 0 3.42 3.42 0 001.946.806 3.42 3.42 0 013.138 3.138 3.42 3.42 0 00.806 1.946 3.42 3.42 0 010 4.438 3.42 3.42 0 00-.806 1.946 3.42 3.42 0 01-3.138 3.138 3.42 3.42 0 00-1.946.806 3.42 3.42 0 01-4.438 0 3.42 3.42 0 00-1.946-.806 3.42 3.42 0 01-3.138-3.138 3.42 3.42 0 00-.806-1.946 3.42 3.42 0 010-4.438 3.42 3.42 0 00.806-1.946 3.42 3.42 0 013.138-3.138z' },
		{ id: 'sources', label: 'Sources', icon: 'M21 12a9 9 0 01-9 9m9-9a9 9 0 00-9-9m9 9H3m9 9a9 9 0 01-9-9m9 9c1.657 0 3-4.03 3-9s-1.343-9-3-9m0 18c-1.657 0-3-4.03-3-9s1.343-9 3-9m-9 9a9 9 0 019-9' },
		{ id: 'automation', label: 'Automation', icon: 'M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15' }
	];

	onMount(async () => {
		await Promise.all([loadLibraries(), loadSettings(), loadDownloadClients(), loadIndexers(), loadQualityPresets(), loadStorageStatus()]);
		// Check if a scan is already running
		checkScanProgress();
	});

	onDestroy(() => {
		if (progressInterval) {
			clearInterval(progressInterval);
			progressInterval = null;
		}
	});

	function startProgressPolling() {
		if (progressInterval) return; // Already polling
		// Check immediately, then poll every second
		checkScanProgress();
		progressInterval = setInterval(async () => {
			await checkScanProgress();
		}, 1000);
	}

	function stopProgressPolling() {
		if (progressInterval) {
			clearInterval(progressInterval);
			progressInterval = null;
		}
	}

	async function checkScanProgress() {
		try {
			const progress = await getScanProgress();
			scanProgress = progress;
			if (progress.scanning) {
				startProgressPolling();
			} else {
				stopProgressPolling();
				// Clear scanning states when done
				for (const id of Object.keys(scanning)) {
					scanning[Number(id)] = false;
				}
			}
		} catch (e) {
			console.error('Failed to get scan progress:', e);
			stopProgressPolling();
		}
	}

	async function loadSettings() {
		try {
			const settings = await getSettings();
			tmdbApiKey = settings['tmdb_api_key'] || '';
		} catch (e) {
			console.error('Failed to load settings:', e);
		}
	}

	async function handleSaveSettings() {
		try {
			savingSettings = true;
			await saveSettings({ tmdb_api_key: tmdbApiKey });
			settingsSaved = true;
			setTimeout(() => (settingsSaved = false), 3000);
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to save settings';
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
			setTimeout(() => (refreshResult = null), 10000);
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to refresh metadata';
		} finally {
			refreshingMetadata = false;
		}
	}

	async function handleClearLibrary() {
		try {
			clearingLibrary = true;
			await clearLibraryData();
			showClearConfirm = false;
			// Reload the page to reflect changes
			window.location.reload();
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to clear library data';
		} finally {
			clearingLibrary = false;
		}
	}

	async function loadLibraries() {
		try {
			loading = true;
			libraries = await getLibraries();
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to load libraries';
		} finally {
			loading = false;
		}
	}

	async function handleAddLibrary() {
		try {
			await createLibrary({ name, path, type, scanInterval: 3600 });
			name = '';
			path = '';
			type = 'movies';
			showAddForm = false;
			await loadLibraries();
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to add library';
		}
	}

	async function handleDelete(id: number) {
		if (!confirm('Are you sure you want to delete this library?')) return;
		try {
			await deleteLibrary(id);
			await loadLibraries();
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to delete library';
		}
	}

	async function handleScan(id: number) {
		try {
			scanning[id] = true;
			await scanLibrary(id);
			// Start polling for progress
			startProgressPolling();
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to start scan';
			scanning[id] = false;
		}
	}

	// Download client functions
	async function loadDownloadClients() {
		try {
			downloadClients = await getDownloadClients();
		} catch (e) {
			console.error('Failed to load download clients:', e);
		}
	}

	async function handleAddClient() {
		try {
			await createDownloadClient({
				name: clientName,
				type: clientType,
				host: clientHost,
				port: clientPort,
				username: clientUsername || undefined,
				password: clientPassword || undefined,
				apiKey: clientApiKey || undefined,
				useTls: clientUseTls,
				category: clientCategory || undefined,
				priority: 0,
				enabled: true
			});
			resetClientForm();
			showAddClientForm = false;
			await loadDownloadClients();
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to add download client';
		}
	}

	function resetClientForm() {
		clientName = '';
		clientType = 'qbittorrent';
		clientHost = '';
		clientPort = 8080;
		clientUsername = '';
		clientPassword = '';
		clientApiKey = '';
		clientUseTls = false;
		clientCategory = '';
	}

	function getDefaultPort(type: string): number {
		switch (type) {
			case 'qbittorrent': return 8080;
			case 'transmission': return 9091;
			case 'sabnzbd': return 8080;
			case 'nzbget': return 6789;
			default: return 8080;
		}
	}

	async function handleTestClient(id: number) {
		try {
			testing[id] = true;
			delete testResults[id];
			const result = await testDownloadClient(id);
			testResults[id] = { success: result.success, message: result.message || result.error || '' };
		} catch (e) {
			testResults[id] = { success: false, message: e instanceof Error ? e.message : 'Test failed' };
		} finally {
			testing[id] = false;
		}
	}

	async function handleDeleteClient(id: number) {
		if (!confirm('Are you sure you want to delete this download client?')) return;
		try {
			await deleteDownloadClient(id);
			await loadDownloadClients();
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to delete download client';
		}
	}

	async function handleToggleClient(client: DownloadClient) {
		try {
			await updateDownloadClient(client.id, { ...client, enabled: !client.enabled });
			await loadDownloadClients();
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to update download client';
		}
	}

	function getClientTypeLabel(type: string): string {
		switch (type) {
			case 'qbittorrent': return 'qBittorrent';
			case 'transmission': return 'Transmission';
			case 'sabnzbd': return 'SABnzbd';
			case 'nzbget': return 'NZBGet';
			default: return type;
		}
	}

	function getClientTypeBadge(type: string): string {
		switch (type) {
			case 'qbittorrent':
			case 'transmission':
				return 'Torrent';
			case 'sabnzbd':
			case 'nzbget':
				return 'Usenet';
			default:
				return '';
		}
	}

	// Indexer functions
	async function loadIndexers() {
		try {
			indexers = await getIndexers();
		} catch (e) {
			console.error('Failed to load indexers:', e);
		}
	}

	async function handleAddIndexer() {
		try {
			await createIndexer({
				name: indexerName,
				type: indexerType,
				url: indexerUrl,
				apiKey: indexerApiKey || undefined,
				categories: indexerCategories || undefined,
				priority: 0,
				enabled: true
			});
			resetIndexerForm();
			showAddIndexerForm = false;
			await loadIndexers();
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to add indexer';
		}
	}

	function resetIndexerForm() {
		indexerName = '';
		indexerType = 'torznab';
		indexerUrl = '';
		indexerApiKey = '';
		indexerCategories = '';
	}

	async function handleTestIndexer(id: number) {
		try {
			testingIndexer[id] = true;
			delete indexerTestResults[id];
			const result = await testIndexer(id);
			indexerTestResults[id] = { success: result.success, message: result.message || result.error || '' };
		} catch (e) {
			indexerTestResults[id] = { success: false, message: e instanceof Error ? e.message : 'Test failed' };
		} finally {
			testingIndexer[id] = false;
		}
	}

	async function handleDeleteIndexer(id: number) {
		if (!confirm('Are you sure you want to delete this indexer?')) return;
		try {
			await deleteIndexer(id);
			await loadIndexers();
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to delete indexer';
		}
	}

	async function handleToggleIndexer(idx: Indexer) {
		try {
			await updateIndexer(idx.id, { ...idx, enabled: !idx.enabled });
			await loadIndexers();
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to update indexer';
		}
	}

	function getIndexerTypeLabel(type: string): string {
		switch (type) {
			case 'torznab': return 'Torznab';
			case 'newznab': return 'Newznab';
			case 'prowlarr': return 'Prowlarr';
			default: return type;
		}
	}

	function getIndexerTypeBadge(type: string): string {
		switch (type) {
			case 'torznab': return 'Torrent';
			case 'newznab': return 'Usenet';
			case 'prowlarr': return 'Proxy';
			default: return '';
		}
	}

	// Quality preset functions
	async function loadQualityPresets() {
		try {
			qualityPresets = (await getQualityPresets()) || [];
		} catch (e) {
			console.error('Failed to load quality presets:', e);
			qualityPresets = [];
		}
	}

	function resetPresetForm() {
		presetName = '';
		presetResolution = '1080p';
		presetSource = 'web';
		presetHdrFormats = [];
		presetCodec = 'any';
		presetAudioFormats = [];
		presetPreferredEdition = 'any';
		presetMinSeeders = 3;
		presetPreferSeasonPacks = true;
		presetAutoUpgrade = true;
		editingPreset = null;
	}

	async function handleAddPreset() {
		try {
			await createQualityPreset({
				name: presetName,
				resolution: presetResolution,
				source: presetSource,
				hdrFormats: presetHdrFormats,
				codec: presetCodec,
				audioFormats: presetAudioFormats,
				preferredEdition: presetPreferredEdition,
				minSeeders: presetMinSeeders,
				preferSeasonPacks: presetPreferSeasonPacks,
				autoUpgrade: presetAutoUpgrade
			});
			resetPresetForm();
			showAddPresetForm = false;
			await loadQualityPresets();
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to add quality preset';
		}
	}

	async function handleUpdatePreset() {
		if (!editingPreset) return;
		try {
			await updateQualityPreset(editingPreset.id, {
				name: presetName,
				resolution: presetResolution,
				source: presetSource,
				hdrFormats: presetHdrFormats,
				codec: presetCodec,
				audioFormats: presetAudioFormats,
				preferredEdition: presetPreferredEdition,
				minSeeders: presetMinSeeders,
				preferSeasonPacks: presetPreferSeasonPacks,
				autoUpgrade: presetAutoUpgrade
			});
			resetPresetForm();
			await loadQualityPresets();
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to update quality preset';
		}
	}

	function startEditPreset(preset: QualityPreset) {
		editingPreset = preset;
		presetName = preset.name;
		presetResolution = preset.resolution;
		presetSource = preset.source;
		presetHdrFormats = preset.hdrFormats || [];
		presetCodec = preset.codec || 'any';
		presetAudioFormats = preset.audioFormats || [];
		presetPreferredEdition = preset.preferredEdition || 'any';
		presetMinSeeders = preset.minSeeders;
		presetPreferSeasonPacks = preset.preferSeasonPacks;
		presetAutoUpgrade = preset.autoUpgrade;
		showAddPresetForm = false;
	}

	async function handleDeletePreset(id: number) {
		if (!confirm('Are you sure you want to delete this quality preset?')) return;
		try {
			await deleteQualityPreset(id);
			await loadQualityPresets();
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to delete quality preset';
		}
	}

	async function handleSetDefaultPreset(id: number) {
		try {
			await setDefaultQualityPreset(id);
			await loadQualityPresets();
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to set default preset';
		}
	}

	function toggleHdrFormat(format: string) {
		if (presetHdrFormats.includes(format)) {
			presetHdrFormats = presetHdrFormats.filter(f => f !== format);
		} else {
			presetHdrFormats = [...presetHdrFormats, format];
		}
	}

	function toggleAudioFormat(format: string) {
		if (presetAudioFormats.includes(format)) {
			presetAudioFormats = presetAudioFormats.filter(f => f !== format);
		} else {
			presetAudioFormats = [...presetAudioFormats, format];
		}
	}

	function getPresetDescription(preset: QualityPreset): string {
		const parts: string[] = [];
		parts.push(preset.resolution.toUpperCase());
		if (preset.source !== 'any') parts.push(preset.source.toUpperCase());
		if (preset.hdrFormats && preset.hdrFormats.length > 0) {
			parts.push(preset.hdrFormats.map(h => h.toUpperCase()).join('/'));
		}
		return parts.join(' · ');
	}

	function getResolutionColor(resolution: string): string {
		switch (resolution) {
			case '4k': return 'bg-purple-600';
			case '1080p': return 'bg-blue-600';
			case '720p': return 'bg-green-600';
			default: return 'bg-bg-elevated';
		}
	}

	// Storage management functions
	async function loadStorageStatus() {
		try {
			storageStatus = await getStorageStatus();
		} catch (e) {
			console.error('Failed to load storage status:', e);
			storageStatus = { thresholdGb: 50, pauseEnabled: false, upgradeDeleteOld: true, moviesSize: 0, tvSize: 0, musicSize: 0, booksSize: 0 };
		}
	}

	function getStorageBarColor(usedPercent: number, freeGb: number, thresholdGb: number): string {
		if (freeGb < thresholdGb) return 'bg-red-500';
		if (usedPercent > 80) return 'bg-yellow-500';
		return 'bg-green-500';
	}

	// Input class for consistency
	const inputClass = "liquid-input w-full px-4 py-2.5";
	const selectClass = "liquid-select w-full px-4 py-2.5";
	const labelClass = "block text-sm text-text-secondary mb-1.5";
</script>

<svelte:head>
	<title>Settings - Outpost</title>
</svelte:head>

<div class="space-y-8 max-w-4xl mx-auto">
	<div>
		<h1 class="text-3xl font-bold text-text-primary">Settings</h1>
		<p class="text-text-secondary mt-1">Configure your media server</p>
	</div>

	{#if error}
		<div class="bg-white/5 border border-white/10 text-text-secondary px-4 py-3 rounded-xl flex items-center justify-between">
			<span>{error}</span>
			<button class="text-text-muted hover:text-text-secondary" onclick={() => (error = null)}>
				<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
				</svg>
			</button>
		</div>
	{/if}

	<!-- Tab Navigation -->
	<div class="flex gap-1 p-1 bg-bg-card rounded-xl border border-white/5">
		{#each tabs as tab}
			<button
				class="flex items-center gap-2 px-4 py-2.5 rounded-lg text-sm font-medium transition-all flex-1 justify-center
					{currentTab === tab.id ? 'bg-white/10 text-text-primary' : 'text-text-muted hover:text-text-secondary hover:bg-white/5'}"
				onclick={() => currentTab = tab.id}
			>
				<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d={tab.icon} />
				</svg>
				<span class="hidden sm:inline">{tab.label}</span>
			</button>
		{/each}
	</div>

	<!-- ============================================ -->
	<!-- GENERAL TAB -->
	<!-- ============================================ -->
	{#if currentTab === 'general'}

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
					bind:value={tmdbApiKey}
					class={inputClass}
					class:flex-1={true}
					placeholder="Enter your TMDB API key"
				/>
				<button
					class="liquid-btn disabled:opacity-50"
					onclick={handleSaveSettings}
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
				onclick={() => (showAddForm = !showAddForm)}
			>
				{showAddForm ? 'Cancel' : 'Add Library'}
			</button>
		</div>

		{#if showAddForm}
			<form
				class="p-4 bg-bg-elevated/50 rounded-xl space-y-4 border border-white/5"
				onsubmit={(e) => {
					e.preventDefault();
					handleAddLibrary();
				}}
			>
				<div class="grid sm:grid-cols-2 gap-4">
					<div>
						<label for="lib-name" class={labelClass}>Name</label>
						<input
							type="text"
							id="lib-name"
							bind:value={name}
							required
							class={inputClass}
							placeholder="Movies"
						/>
					</div>
					<div>
						<label for="lib-type" class={labelClass}>Type</label>
						<Select
							id="lib-type"
							bind:value={type}
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
					<input
						type="text"
						id="lib-path"
						bind:value={path}
						required
						class={inputClass}
						placeholder="/media/movies"
					/>
				</div>
				<button type="submit" class="liquid-btn">
					Add Library
				</button>
			</form>
		{/if}

		{#if loading}
			<div class="flex items-center gap-3 py-4">
				<div class="w-5 h-5 border-2 border-white-400 border-t-transparent rounded-full animate-spin"></div>
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
								onclick={() => handleScan(lib.id)}
								disabled={scanning[lib.id] || scanProgress?.scanning}
							>
								{scanning[lib.id] || (scanProgress?.scanning && scanProgress.library === lib.name) ? 'Scanning...' : 'Scan'}
							</button>
							<button
								class="liquid-btn-sm !bg-white/5 !border-t-white/10 text-text-secondary hover:text-white"
								onclick={() => handleDelete(lib.id)}
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
						<div class="w-5 h-5 border-2 border-blue-400 border-t-transparent rounded-full animate-spin"></div>
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

	{/if}

	<!-- ============================================ -->
	<!-- SOURCES TAB -->
	<!-- ============================================ -->
	{#if currentTab === 'sources'}

	<!-- Download Clients -->
	<section class="glass-card p-6 space-y-4">
		<div class="flex items-center justify-between">
			<div class="flex items-center gap-3">
				<div class="w-10 h-10 rounded-xl bg-green-600/20 flex items-center justify-center">
					<svg class="w-5 h-5 text-green-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4" />
					</svg>
				</div>
				<div>
					<h2 class="text-lg font-semibold text-text-primary">Download Clients</h2>
					<p class="text-sm text-text-secondary">Configure torrent and usenet downloaders</p>
				</div>
			</div>
			<button
				class="liquid-btn-sm"
				onclick={() => { showAddClientForm = !showAddClientForm; resetClientForm(); }}
			>
				{showAddClientForm ? 'Cancel' : 'Add Client'}
			</button>
		</div>

		{#if showAddClientForm}
			<form
				class="p-4 bg-bg-elevated/50 rounded-xl space-y-4 border border-white/5"
				onsubmit={(e) => { e.preventDefault(); handleAddClient(); }}
			>
				<div class="grid sm:grid-cols-2 gap-4">
					<div>
						<label for="client-name" class={labelClass}>Name</label>
						<input type="text" id="client-name" bind:value={clientName} required class={inputClass} placeholder="My qBittorrent" />
					</div>
					<div>
						<label for="client-type" class={labelClass}>Type</label>
						<Select
							id="client-type"
							bind:value={clientType}
							onchange={() => { clientPort = getDefaultPort(clientType); }}
							options={[
								{ value: 'qbittorrent', label: 'qBittorrent' },
								{ value: 'transmission', label: 'Transmission' },
								{ value: 'sabnzbd', label: 'SABnzbd' },
								{ value: 'nzbget', label: 'NZBGet' }
							]}
						/>
					</div>
				</div>
				<div class="grid sm:grid-cols-3 gap-4">
					<div class="sm:col-span-2">
						<label for="client-host" class={labelClass}>Host</label>
						<input type="text" id="client-host" bind:value={clientHost} required class={inputClass} placeholder="localhost or 192.168.1.100" />
					</div>
					<div>
						<label for="client-port" class={labelClass}>Port</label>
						<input type="number" id="client-port" bind:value={clientPort} required class={inputClass} />
					</div>
				</div>
				<div class="grid sm:grid-cols-2 gap-4">
					<div>
						<label for="client-username" class={labelClass}>Username</label>
						<input type="text" id="client-username" bind:value={clientUsername} class={inputClass} placeholder="Optional" />
					</div>
					<div>
						<label for="client-password" class={labelClass}>Password</label>
						<input type="password" id="client-password" bind:value={clientPassword} class={inputClass} placeholder="Optional" />
					</div>
				</div>
				{#if clientType === 'sabnzbd' || clientType === 'nzbget'}
					<div>
						<label for="client-apikey" class={labelClass}>API Key</label>
						<input type="password" id="client-apikey" bind:value={clientApiKey} class={inputClass} placeholder="API Key from client settings" />
					</div>
				{/if}
				<div class="grid sm:grid-cols-2 gap-4">
					<div>
						<label for="client-category" class={labelClass}>Category/Label</label>
						<input type="text" id="client-category" bind:value={clientCategory} class={inputClass} placeholder="outpost (optional)" />
					</div>
					<div class="flex items-center pt-7">
						<label class="flex items-center gap-2 cursor-pointer">
							<input type="checkbox" bind:checked={clientUseTls} class="w-4 h-4 rounded bg-bg-elevated border-white/20 text-white-400 focus:ring-white-400" />
							<span class="text-sm text-text-secondary">Use HTTPS</span>
						</label>
					</div>
				</div>
				<button type="submit" class="liquid-btn">
					Add Client
				</button>
			</form>
		{/if}

		{#if downloadClients.length === 0}
			<p class="text-text-muted py-4">No download clients configured. Add one to enable downloading.</p>
		{:else}
			<div class="space-y-2">
				{#each downloadClients as client}
					<div class="p-4 bg-bg-elevated/50 rounded-xl border border-white/5">
						<div class="flex items-center justify-between">
							<div class="flex items-center gap-3">
								<button
									class="w-10 h-6 rounded-full relative transition-colors {client.enabled ? 'bg-green-600' : 'bg-bg-card'}"
									onclick={() => handleToggleClient(client)}
									aria-label={client.enabled ? 'Disable client' : 'Enable client'}
								>
									<span class="absolute top-1 w-4 h-4 rounded-full bg-white transition-transform {client.enabled ? 'left-5' : 'left-1'}"></span>
								</button>
								<div>
									<div class="flex items-center gap-2 flex-wrap">
										<h3 class="font-medium text-text-primary">{client.name}</h3>
										<span class="px-2 py-0.5 text-xs rounded-lg bg-bg-card text-text-muted">
											{getClientTypeLabel(client.type)}
										</span>
										<span class="px-2 py-0.5 text-xs rounded-lg {client.type === 'qbittorrent' || client.type === 'transmission' ? 'bg-blue-900/50 text-blue-300' : 'bg-purple-900/50 text-purple-300'}">
											{getClientTypeBadge(client.type)}
										</span>
									</div>
									<p class="text-sm text-text-secondary">
										{client.useTls ? 'https' : 'http'}://{client.host}:{client.port}
										{#if client.category}
											<span class="ml-2 text-text-muted">Category: {client.category}</span>
										{/if}
									</p>
								</div>
							</div>
							<div class="flex gap-2">
								<button
									class="liquid-btn-sm disabled:opacity-50"
									onclick={() => handleTestClient(client.id)}
									disabled={testing[client.id]}
								>
									{testing[client.id] ? 'Testing...' : 'Test'}
								</button>
								<button
									class="liquid-btn-sm !bg-white/5 !border-t-white/10 text-text-secondary hover:text-white"
									onclick={() => handleDeleteClient(client.id)}
								>
									Delete
								</button>
							</div>
						</div>
						{#if testResults[client.id]}
							<div class="mt-3 text-sm flex items-center gap-2 {testResults[client.id].success ? 'text-green-400' : 'text-text-secondary'}">
								<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
									{#if testResults[client.id].success}
										<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
									{:else}
										<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
									{/if}
								</svg>
								{testResults[client.id].success ? 'Connection successful!' : testResults[client.id].message}
							</div>
						{/if}
					</div>
				{/each}
			</div>
		{/if}
	</section>

	<!-- Indexers -->
	<section class="glass-card p-6 space-y-4">
		<div class="flex items-center justify-between">
			<div class="flex items-center gap-3">
				<div class="w-10 h-10 rounded-xl bg-purple-600/20 flex items-center justify-center">
					<svg class="w-5 h-5 text-purple-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
					</svg>
				</div>
				<div>
					<h2 class="text-lg font-semibold text-text-primary">Indexers</h2>
					<p class="text-sm text-text-secondary">Configure release search sources</p>
				</div>
			</div>
			<button
				class="liquid-btn-sm"
				onclick={() => { showAddIndexerForm = !showAddIndexerForm; resetIndexerForm(); }}
			>
				{showAddIndexerForm ? 'Cancel' : 'Add Indexer'}
			</button>
		</div>

		{#if showAddIndexerForm}
			<form
				class="p-4 bg-bg-elevated/50 rounded-xl space-y-4 border border-white/5"
				onsubmit={(e) => { e.preventDefault(); handleAddIndexer(); }}
			>
				<div class="grid sm:grid-cols-2 gap-4">
					<div>
						<label for="indexer-name" class={labelClass}>Name</label>
						<input type="text" id="indexer-name" bind:value={indexerName} required class={inputClass} placeholder="My Indexer" />
					</div>
					<div>
						<label for="indexer-type" class={labelClass}>Type</label>
						<Select
							id="indexer-type"
							bind:value={indexerType}
							options={[
								{ value: 'torznab', label: 'Torznab (Torrent)' },
								{ value: 'newznab', label: 'Newznab (Usenet)' },
								{ value: 'prowlarr', label: 'Prowlarr' }
							]}
						/>
					</div>
				</div>
				<div>
					<label for="indexer-url" class={labelClass}>URL</label>
					<input type="url" id="indexer-url" bind:value={indexerUrl} required class={inputClass} placeholder={indexerType === 'prowlarr' ? 'http://localhost:9696' : 'http://indexer.example.com/api'} />
				</div>
				<div>
					<label for="indexer-apikey" class={labelClass}>API Key</label>
					<input type="password" id="indexer-apikey" bind:value={indexerApiKey} class={inputClass} placeholder="API Key from indexer" />
				</div>
				{#if indexerType !== 'prowlarr'}
					<div>
						<label for="indexer-categories" class={labelClass}>Categories (optional)</label>
						<input type="text" id="indexer-categories" bind:value={indexerCategories} class={inputClass} placeholder="2000,5000 (comma-separated IDs)" />
						<p class="text-xs text-text-muted mt-1">Common: 2000=Movies, 5000=TV</p>
					</div>
				{/if}
				<button type="submit" class="liquid-btn">
					Add Indexer
				</button>
			</form>
		{/if}

		{#if indexers.length === 0}
			<p class="text-text-muted py-4">No indexers configured. Add one to enable searching for releases.</p>
		{:else}
			<div class="space-y-2">
				{#each indexers as idx}
					<div class="p-4 bg-bg-elevated/50 rounded-xl border border-white/5">
						<div class="flex items-center justify-between">
							<div class="flex items-center gap-3">
								<button
									class="w-10 h-6 rounded-full relative transition-colors {idx.enabled ? 'bg-green-600' : 'bg-bg-card'}"
									onclick={() => handleToggleIndexer(idx)}
									aria-label={idx.enabled ? 'Disable indexer' : 'Enable indexer'}
								>
									<span class="absolute top-1 w-4 h-4 rounded-full bg-white transition-transform {idx.enabled ? 'left-5' : 'left-1'}"></span>
								</button>
								<div>
									<div class="flex items-center gap-2 flex-wrap">
										<h3 class="font-medium text-text-primary">{idx.name}</h3>
										<span class="px-2 py-0.5 text-xs rounded-lg bg-bg-card text-text-muted">
											{getIndexerTypeLabel(idx.type)}
										</span>
										<span class="px-2 py-0.5 text-xs rounded-lg {idx.type === 'torznab' ? 'bg-blue-900/50 text-blue-300' : idx.type === 'newznab' ? 'bg-purple-900/50 text-purple-300' : 'bg-green-900/50 text-green-300'}">
											{getIndexerTypeBadge(idx.type)}
										</span>
									</div>
									<p class="text-sm text-text-secondary truncate max-w-sm">
										{idx.url}
									</p>
								</div>
							</div>
							<div class="flex gap-2">
								<button
									class="liquid-btn-sm disabled:opacity-50"
									onclick={() => handleTestIndexer(idx.id)}
									disabled={testingIndexer[idx.id]}
								>
									{testingIndexer[idx.id] ? 'Testing...' : 'Test'}
								</button>
								<button
									class="liquid-btn-sm !bg-white/5 !border-t-white/10 text-text-secondary hover:text-white"
									onclick={() => handleDeleteIndexer(idx.id)}
								>
									Delete
								</button>
							</div>
						</div>
						{#if indexerTestResults[idx.id]}
							<div class="mt-3 text-sm flex items-center gap-2 {indexerTestResults[idx.id].success ? 'text-green-400' : 'text-text-secondary'}">
								<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
									{#if indexerTestResults[idx.id].success}
										<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
									{:else}
										<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
									{/if}
								</svg>
								{indexerTestResults[idx.id].success ? 'Connection successful!' : indexerTestResults[idx.id].message}
							</div>
						{/if}
					</div>
				{/each}
			</div>
		{/if}
	</section>

	{/if}

	<!-- ============================================ -->
	<!-- QUALITY TAB -->
	<!-- ============================================ -->
	{#if currentTab === 'quality'}

	<!-- Quality Presets -->
	<section class="glass-card p-6 space-y-4">
		<div class="flex items-center justify-between">
			<div class="flex items-center gap-3">
				<div class="w-10 h-10 rounded-xl bg-cyan-600/20 flex items-center justify-center">
					<svg class="w-5 h-5 text-cyan-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4M7.835 4.697a3.42 3.42 0 001.946-.806 3.42 3.42 0 014.438 0 3.42 3.42 0 001.946.806 3.42 3.42 0 013.138 3.138 3.42 3.42 0 00.806 1.946 3.42 3.42 0 010 4.438 3.42 3.42 0 00-.806 1.946 3.42 3.42 0 01-3.138 3.138 3.42 3.42 0 00-1.946.806 3.42 3.42 0 01-4.438 0 3.42 3.42 0 00-1.946-.806 3.42 3.42 0 01-3.138-3.138 3.42 3.42 0 00-.806-1.946 3.42 3.42 0 010-4.438 3.42 3.42 0 00.806-1.946 3.42 3.42 0 013.138-3.138z" />
					</svg>
				</div>
				<div>
					<h2 class="text-lg font-semibold text-text-primary">Quality Presets</h2>
					<p class="text-sm text-text-secondary">Define target quality for automatic downloads</p>
				</div>
			</div>
			<button
				class="liquid-btn-sm"
				onclick={() => { showAddPresetForm = !showAddPresetForm; resetPresetForm(); }}
			>
				{showAddPresetForm ? 'Cancel' : 'Add Preset'}
			</button>
		</div>

		{#if showAddPresetForm || editingPreset}
			<form
				class="p-4 bg-bg-elevated/50 rounded-xl space-y-4 border border-white/5"
				onsubmit={(e) => { e.preventDefault(); editingPreset ? handleUpdatePreset() : handleAddPreset(); }}
			>
				<div>
					<label for="preset-name" class={labelClass}>Preset Name</label>
					<input type="text" id="preset-name" bind:value={presetName} required class={inputClass} placeholder="My Custom Preset" disabled={editingPreset?.isBuiltIn} />
				</div>

				<div class="grid sm:grid-cols-2 gap-4">
					<div>
						<label for="preset-resolution" class={labelClass}>Resolution</label>
						<Select
							id="preset-resolution"
							bind:value={presetResolution}
							options={resolutionOptions.map(res => ({ value: res, label: res.toUpperCase() }))}
						/>
					</div>
					<div>
						<label for="preset-source" class={labelClass}>Source</label>
						<Select
							id="preset-source"
							bind:value={presetSource}
							options={sourceOptions.map(src => ({ value: src, label: src.charAt(0).toUpperCase() + src.slice(1) }))}
						/>
					</div>
				</div>

				<div>
					<label class="block text-sm text-text-secondary mb-2">HDR Formats (optional)</label>
					<p class="text-xs text-text-muted mb-3">Select preferred HDR formats. Empty = SDR is acceptable.</p>
					<div class="flex flex-wrap gap-2">
						{#each hdrOptions as hdr}
							<button
								type="button"
								class="px-3 py-1.5 text-xs rounded-lg transition-colors font-medium {presetHdrFormats.includes(hdr) ? 'bg-purple-600 text-white' : 'bg-bg-card text-text-muted hover:bg-bg-elevated'}"
								onclick={() => toggleHdrFormat(hdr)}
							>
								{hdr.toUpperCase()}
							</button>
						{/each}
					</div>
				</div>

				<div>
					<label class="block text-sm text-text-secondary mb-2">Audio Formats (optional)</label>
					<p class="text-xs text-text-muted mb-3">Select preferred audio formats. Empty = any audio.</p>
					<div class="flex flex-wrap gap-2">
						{#each audioOptions as audio}
							<button
								type="button"
								class="px-3 py-1.5 text-xs rounded-lg transition-colors font-medium {presetAudioFormats.includes(audio) ? 'bg-blue-600 text-white' : 'bg-bg-card text-text-muted hover:bg-bg-elevated'}"
								onclick={() => toggleAudioFormat(audio)}
							>
								{audio.toUpperCase()}
							</button>
						{/each}
					</div>
				</div>

				<div class="grid sm:grid-cols-3 gap-4">
					<div>
						<label for="preset-codec" class={labelClass}>Codec</label>
						<Select
							id="preset-codec"
							bind:value={presetCodec}
							options={codecOptions.map(codec => ({ value: codec, label: codec.toUpperCase() }))}
						/>
					</div>
					<div>
						<label for="preset-edition" class={labelClass}>Preferred Edition</label>
						<Select
							id="preset-edition"
							bind:value={presetPreferredEdition}
							options={editionOptions.map(edition => ({ value: edition, label: edition.charAt(0).toUpperCase() + edition.slice(1) }))}
						/>
					</div>
					<div>
						<label for="preset-seeders" class={labelClass}>Min Seeders</label>
						<input type="number" id="preset-seeders" bind:value={presetMinSeeders} min="0" class={inputClass} />
					</div>
				</div>

				<div class="flex flex-wrap gap-6">
					<label class="flex items-center gap-2 cursor-pointer">
						<input type="checkbox" bind:checked={presetPreferSeasonPacks} class="w-4 h-4 rounded bg-bg-elevated border-white/20 text-cyan-400 focus:ring-cyan-400" />
						<span class="text-sm text-text-secondary">Prefer season packs for TV</span>
					</label>
					<label class="flex items-center gap-2 cursor-pointer">
						<input type="checkbox" bind:checked={presetAutoUpgrade} class="w-4 h-4 rounded bg-bg-elevated border-white/20 text-cyan-400 focus:ring-cyan-400" />
						<span class="text-sm text-text-secondary">Auto-upgrade when better quality found</span>
					</label>
				</div>

				<div class="flex gap-2">
					<button type="submit" class="liquid-btn">
						{editingPreset ? 'Update Preset' : 'Add Preset'}
					</button>
					{#if editingPreset}
						<button type="button" class="liquid-btn !bg-white/5 !border-t-white/10 text-text-secondary hover:text-white" onclick={resetPresetForm}>
							Cancel
						</button>
					{/if}
				</div>
			</form>
		{/if}

		{#if qualityPresets.length === 0}
			<p class="text-text-muted py-4">No quality presets configured. Built-in presets will be added on first use.</p>
		{:else}
			<div class="space-y-2">
				{#each qualityPresets as preset}
					<div class="p-4 bg-bg-elevated/50 rounded-xl border border-white/5 {preset.isDefault ? 'ring-1 ring-cyan-500/50' : ''}">
						<div class="flex items-center justify-between">
							<div>
								<div class="flex items-center gap-2 flex-wrap">
									<h3 class="font-medium text-text-primary">{preset.name}</h3>
									<span class="px-2 py-0.5 text-xs rounded-lg {getResolutionColor(preset.resolution)} text-white">
										{preset.resolution.toUpperCase()}
									</span>
									{#if preset.isBuiltIn}
										<span class="px-2 py-0.5 text-xs rounded-lg bg-bg-card text-text-muted">Built-in</span>
									{/if}
									{#if preset.isDefault}
										<span class="px-2 py-0.5 text-xs rounded-lg bg-cyan-900/50 text-cyan-300">Default</span>
									{/if}
									{#if preset.autoUpgrade}
										<span class="px-2 py-0.5 text-xs rounded-lg bg-green-900/50 text-green-300">Auto-upgrade</span>
									{/if}
								</div>
								<p class="text-sm text-text-secondary mt-1">
									{getPresetDescription(preset)}
								</p>
								{#if preset.audioFormats && preset.audioFormats.length > 0}
									<p class="text-xs text-text-muted mt-1">
										Audio: {preset.audioFormats.map(a => a.toUpperCase()).join(', ')}
									</p>
								{/if}
							</div>
							<div class="flex gap-2">
								{#if !preset.isDefault}
									<button
										class="liquid-btn-sm"
										onclick={() => handleSetDefaultPreset(preset.id)}
									>
										Set Default
									</button>
								{/if}
								{#if !preset.isBuiltIn}
									<button
										class="liquid-btn-sm"
										onclick={() => startEditPreset(preset)}
									>
										Edit
									</button>
									<button
										class="liquid-btn-sm !bg-white/5 !border-t-white/10 text-text-secondary hover:text-white"
										onclick={() => handleDeletePreset(preset.id)}
									>
										Delete
									</button>
								{/if}
							</div>
						</div>
					</div>
				{/each}
			</div>
		{/if}
	</section>

	{/if}

	<!-- ============================================ -->
	<!-- AUTOMATION TAB -->
	<!-- ============================================ -->
	{#if currentTab === 'automation'}

	<!-- Storage Management -->
	<section class="glass-card p-6 space-y-4">
		<div class="flex items-center gap-3">
			<div class="w-10 h-10 rounded-xl bg-orange-600/20 flex items-center justify-center">
				<svg class="w-5 h-5 text-orange-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 7v10c0 2.21 3.582 4 8 4s8-1.79 8-4V7M4 7c0 2.21 3.582 4 8 4s8-1.79 8-4M4 7c0-2.21 3.582-4 8-4s8 1.79 8 4m0 5c0 2.21-3.582 4-8 4s-8-1.79-8-4" />
				</svg>
			</div>
			<div>
				<h2 class="text-lg font-semibold text-text-primary">Storage</h2>
				<p class="text-sm text-text-secondary">Server disk space and media usage</p>
			</div>
		</div>

		{#if storageStatus}
			{@const formatSize = (bytes: number) => {
				if (bytes === 0) return '0 B';
				const gb = bytes / (1024 * 1024 * 1024);
				if (gb >= 1000) return (gb / 1024).toFixed(1) + ' TB';
				if (gb >= 1) return gb.toFixed(1) + ' GB';
				const mb = bytes / (1024 * 1024);
				return mb.toFixed(0) + ' MB';
			}}
			{@const totalMedia = storageStatus.moviesSize + storageStatus.tvSize + storageStatus.musicSize + storageStatus.booksSize}

			<!-- Media Usage Breakdown -->
			<div class="grid grid-cols-2 gap-3">
				{#if storageStatus.moviesSize > 0}
					<div class="p-4 bg-bg-elevated/50 rounded-xl border border-white/5">
						<div class="flex items-center gap-3">
							<div class="w-8 h-8 rounded-lg bg-blue-500/20 flex items-center justify-center">
								<svg class="w-4 h-4 text-blue-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M7 4v16M17 4v16M3 8h4m10 0h4M3 12h18M3 16h4m10 0h4M4 20h16a1 1 0 001-1V5a1 1 0 00-1-1H4a1 1 0 00-1 1v14a1 1 0 001 1z" />
								</svg>
							</div>
							<div>
								<p class="text-xs text-text-muted uppercase tracking-wide">Movies</p>
								<p class="text-lg font-semibold text-text-primary">{formatSize(storageStatus.moviesSize)}</p>
							</div>
						</div>
					</div>
				{/if}
				{#if storageStatus.tvSize > 0}
					<div class="p-4 bg-bg-elevated/50 rounded-xl border border-white/5">
						<div class="flex items-center gap-3">
							<div class="w-8 h-8 rounded-lg bg-purple-500/20 flex items-center justify-center">
								<svg class="w-4 h-4 text-purple-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9.75 17L9 20l-1 1h8l-1-1-.75-3M3 13h18M5 17h14a2 2 0 002-2V5a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z" />
								</svg>
							</div>
							<div>
								<p class="text-xs text-text-muted uppercase tracking-wide">TV Shows</p>
								<p class="text-lg font-semibold text-text-primary">{formatSize(storageStatus.tvSize)}</p>
							</div>
						</div>
					</div>
				{/if}
				{#if storageStatus.musicSize > 0}
					<div class="p-4 bg-bg-elevated/50 rounded-xl border border-white/5">
						<div class="flex items-center gap-3">
							<div class="w-8 h-8 rounded-lg bg-green-500/20 flex items-center justify-center">
								<svg class="w-4 h-4 text-green-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 19V6l12-3v13M9 19c0 1.105-1.343 2-3 2s-3-.895-3-2 1.343-2 3-2 3 .895 3 2zm12-3c0 1.105-1.343 2-3 2s-3-.895-3-2 1.343-2 3-2 3 .895 3 2zM9 10l12-3" />
								</svg>
							</div>
							<div>
								<p class="text-xs text-text-muted uppercase tracking-wide">Music</p>
								<p class="text-lg font-semibold text-text-primary">{formatSize(storageStatus.musicSize)}</p>
							</div>
						</div>
					</div>
				{/if}
				{#if storageStatus.booksSize > 0}
					<div class="p-4 bg-bg-elevated/50 rounded-xl border border-white/5">
						<div class="flex items-center gap-3">
							<div class="w-8 h-8 rounded-lg bg-amber-500/20 flex items-center justify-center">
								<svg class="w-4 h-4 text-amber-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 6.253v13m0-13C10.832 5.477 9.246 5 7.5 5S4.168 5.477 3 6.253v13C4.168 18.477 5.754 18 7.5 18s3.332.477 4.5 1.253m0-13C13.168 5.477 14.754 5 16.5 5c1.747 0 3.332.477 4.5 1.253v13C19.832 18.477 18.247 18 16.5 18c-1.746 0-3.332.477-4.5 1.253" />
								</svg>
							</div>
							<div>
								<p class="text-xs text-text-muted uppercase tracking-wide">Books</p>
								<p class="text-lg font-semibold text-text-primary">{formatSize(storageStatus.booksSize)}</p>
							</div>
						</div>
					</div>
				{/if}
			</div>

			<!-- Total Media -->
			{#if totalMedia > 0}
				<div class="text-sm text-text-secondary">
					Total media: <span class="text-text-primary font-medium">{formatSize(totalMedia)}</span>
				</div>
			{/if}

			<!-- Disk Space -->
			{#if storageStatus.diskUsage}
				{@const disk = storageStatus.diskUsage}
				{@const totalGb = Math.round(disk.total / (1024 * 1024 * 1024))}
				{@const freeGb = Math.round(disk.free / (1024 * 1024 * 1024))}
				{@const usedGb = Math.round(disk.used / (1024 * 1024 * 1024))}
				<div class="p-4 bg-bg-elevated/50 rounded-xl border border-white/5">
					<div class="flex items-center justify-between mb-3">
						<span class="font-medium text-text-primary">Disk Space</span>
						<span class="text-sm {freeGb < storageStatus.thresholdGb ? 'text-red-400' : 'text-text-secondary'}">
							{freeGb} GB free of {totalGb} GB
						</span>
					</div>
					<div class="w-full bg-bg-card rounded-full h-3 overflow-hidden">
						<div
							class="{getStorageBarColor(disk.usedPercent, freeGb, storageStatus.thresholdGb)} h-full transition-all duration-300"
							style="width: {disk.usedPercent}%"
						></div>
					</div>
					<div class="flex justify-between mt-2 text-sm text-text-muted">
						<span>{usedGb} GB used</span>
						<span>{disk.usedPercent.toFixed(1)}%</span>
					</div>
					{#if freeGb < storageStatus.thresholdGb}
						<div class="mt-3 flex items-center gap-2 text-sm text-red-400">
							<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
								<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
							</svg>
							Below threshold ({storageStatus.thresholdGb} GB) - downloads may pause
						</div>
					{/if}
				</div>
			{:else}
				<p class="text-text-muted text-sm">Disk usage information not available.</p>
			{/if}
		{:else}
			<div class="flex items-center gap-3 py-4">
				<div class="w-5 h-5 border-2 border-orange-400 border-t-transparent rounded-full animate-spin"></div>
				<span class="text-text-secondary">Loading storage status...</span>
			</div>
		{/if}
	</section>

	{/if}
</div>
