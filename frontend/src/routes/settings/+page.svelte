<script lang="ts">
	import { onMount } from 'svelte';
	import {
		getLibraries,
		createLibrary,
		deleteLibrary,
		scanLibrary,
		getSettings,
		saveSettings,
		refreshAllMetadata,
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
		getQualityProfiles,
		createQualityProfile,
		updateQualityProfile,
		deleteQualityProfile,
		type Library,
		type DownloadClient,
		type Indexer,
		type QualityProfile
	} from '$lib/api';

	let libraries: Library[] = $state([]);
	let downloadClients: DownloadClient[] = $state([]);
	let indexers: Indexer[] = $state([]);
	let qualityProfiles: QualityProfile[] = $state([]);
	let loading = $state(true);
	let error: string | null = $state(null);
	let showAddForm = $state(false);
	let showAddClientForm = $state(false);
	let showAddIndexerForm = $state(false);
	let showAddProfileForm = $state(false);
	let editingProfile: QualityProfile | null = $state(null);
	let scanning: Record<number, boolean> = $state({});
	let testing: Record<number, boolean> = $state({});
	let testingIndexer: Record<number, boolean> = $state({});
	let testResults: Record<number, { success: boolean; message: string }> = $state({});
	let indexerTestResults: Record<number, { success: boolean; message: string }> = $state({});

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

	// Quality profile form state
	let profileName = $state('');
	let profileUpgradeAllowed = $state(true);
	let profileUpgradeUntilScore = $state(100000);
	let profileMinFormatScore = $state(0);
	let profileCutoffFormatScore = $state(50000);
	let profileQualities: string[] = $state([]);

	// All available qualities
	const allQualities = [
		'Remux-2160p', 'Bluray-2160p', 'WEBDL-2160p', 'WEBRip-2160p', 'HDTV-2160p',
		'Remux-1080p', 'Bluray-1080p', 'WEBDL-1080p', 'WEBRip-1080p', 'HDTV-1080p',
		'Bluray-720p', 'WEBDL-720p', 'WEBRip-720p', 'HDTV-720p',
		'DVD', 'SDTV'
	];

	// Settings state
	let tmdbApiKey = $state('');
	let savingSettings = $state(false);
	let settingsSaved = $state(false);
	let refreshingMetadata = $state(false);
	let refreshResult: { refreshed: number; errors: number; total: number } | null = $state(null);

	onMount(async () => {
		await Promise.all([loadLibraries(), loadSettings(), loadDownloadClients(), loadIndexers(), loadQualityProfiles()]);
	});

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
			setTimeout(() => {
				scanning[id] = false;
			}, 2000);
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

	// Quality profile functions
	async function loadQualityProfiles() {
		try {
			qualityProfiles = await getQualityProfiles();
		} catch (e) {
			console.error('Failed to load quality profiles:', e);
		}
	}

	function resetProfileForm() {
		profileName = '';
		profileUpgradeAllowed = true;
		profileUpgradeUntilScore = 100000;
		profileMinFormatScore = 0;
		profileCutoffFormatScore = 50000;
		profileQualities = [];
		editingProfile = null;
	}

	async function handleAddProfile() {
		try {
			await createQualityProfile({
				name: profileName,
				upgradeAllowed: profileUpgradeAllowed,
				upgradeUntilScore: profileUpgradeUntilScore,
				minFormatScore: profileMinFormatScore,
				cutoffFormatScore: profileCutoffFormatScore,
				qualities: JSON.stringify(profileQualities),
				customFormats: '{}'
			});
			resetProfileForm();
			showAddProfileForm = false;
			await loadQualityProfiles();
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to add quality profile';
		}
	}

	async function handleUpdateProfile() {
		if (!editingProfile) return;
		try {
			await updateQualityProfile(editingProfile.id, {
				name: profileName,
				upgradeAllowed: profileUpgradeAllowed,
				upgradeUntilScore: profileUpgradeUntilScore,
				minFormatScore: profileMinFormatScore,
				cutoffFormatScore: profileCutoffFormatScore,
				qualities: JSON.stringify(profileQualities)
			});
			resetProfileForm();
			await loadQualityProfiles();
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to update quality profile';
		}
	}

	function startEditProfile(profile: QualityProfile) {
		editingProfile = profile;
		profileName = profile.name;
		profileUpgradeAllowed = profile.upgradeAllowed;
		profileUpgradeUntilScore = profile.upgradeUntilScore;
		profileMinFormatScore = profile.minFormatScore;
		profileCutoffFormatScore = profile.cutoffFormatScore;
		try {
			profileQualities = JSON.parse(profile.qualities || '[]');
		} catch {
			profileQualities = [];
		}
		showAddProfileForm = false;
	}

	async function handleDeleteProfile(id: number) {
		if (!confirm('Are you sure you want to delete this quality profile?')) return;
		try {
			await deleteQualityProfile(id);
			await loadQualityProfiles();
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to delete quality profile';
		}
	}

	function toggleQuality(quality: string) {
		if (profileQualities.includes(quality)) {
			profileQualities = profileQualities.filter(q => q !== quality);
		} else {
			profileQualities = [...profileQualities, quality];
		}
	}

	function getQualityColor(quality: string): string {
		if (quality.includes('Remux')) return 'bg-purple-600';
		if (quality.includes('2160p')) return 'bg-blue-600';
		if (quality.includes('1080p')) return 'bg-green-600';
		if (quality.includes('720p')) return 'bg-white-600';
		return 'bg-bg-elevated';
	}

	function formatProfileQualities(qualitiesJson: string): string {
		try {
			const quals = JSON.parse(qualitiesJson || '[]');
			if (quals.length === 0) return 'All qualities';
			if (quals.length <= 3) return quals.join(', ');
			return `${quals.slice(0, 2).join(', ')} +${quals.length - 2} more`;
		} catch {
			return 'All qualities';
		}
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
						<select id="lib-type" bind:value={type} class={selectClass}>
							<option value="movies">Movies</option>
							<option value="tv">TV Shows</option>
							<option value="anime">Anime</option>
							<option value="music">Music</option>
							<option value="books">Books</option>
						</select>
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
								disabled={scanning[lib.id]}
							>
								{scanning[lib.id] ? 'Scanning...' : 'Scan'}
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
	</section>

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
						<select id="client-type" bind:value={clientType} onchange={() => { clientPort = getDefaultPort(clientType); }} class={selectClass}>
							<option value="qbittorrent">qBittorrent</option>
							<option value="transmission">Transmission</option>
							<option value="sabnzbd">SABnzbd</option>
							<option value="nzbget">NZBGet</option>
						</select>
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
						<select id="indexer-type" bind:value={indexerType} class={selectClass}>
							<option value="torznab">Torznab (Torrent)</option>
							<option value="newznab">Newznab (Usenet)</option>
							<option value="prowlarr">Prowlarr</option>
						</select>
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

	<!-- Quality Profiles -->
	<section class="glass-card p-6 space-y-4">
		<div class="flex items-center justify-between">
			<div class="flex items-center gap-3">
				<div class="w-10 h-10 rounded-xl bg-white-600/20 flex items-center justify-center">
					<svg class="w-5 h-5 text-white-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 10V3L4 14h7v7l9-11h-7z" />
					</svg>
				</div>
				<div>
					<h2 class="text-lg font-semibold text-text-primary">Quality Profiles</h2>
					<p class="text-sm text-text-secondary">Control release quality preferences</p>
				</div>
			</div>
			<button
				class="liquid-btn-sm"
				onclick={() => { showAddProfileForm = !showAddProfileForm; resetProfileForm(); }}
			>
				{showAddProfileForm ? 'Cancel' : 'Add Profile'}
			</button>
		</div>

		{#if showAddProfileForm || editingProfile}
			<form
				class="p-4 bg-bg-elevated/50 rounded-xl space-y-4 border border-white/5"
				onsubmit={(e) => { e.preventDefault(); editingProfile ? handleUpdateProfile() : handleAddProfile(); }}
			>
				<div>
					<label for="profile-name" class={labelClass}>Profile Name</label>
					<input type="text" id="profile-name" bind:value={profileName} required class={inputClass} placeholder="4K Enthusiast" />
				</div>

				<div>
					<label class="block text-sm text-text-secondary mb-2">Allowed Qualities</label>
					<p class="text-xs text-text-muted mb-3">Select which quality tiers are acceptable. Empty = all allowed.</p>
					<div class="flex flex-wrap gap-2">
						{#each allQualities as quality}
							<button
								type="button"
								class="px-3 py-1.5 text-xs rounded-lg transition-colors font-medium {profileQualities.includes(quality) ? getQualityColor(quality) + ' text-white' : 'bg-bg-card text-text-muted hover:bg-bg-elevated'}"
								onclick={() => toggleQuality(quality)}
							>
								{quality}
							</button>
						{/each}
					</div>
				</div>

				<div class="grid sm:grid-cols-2 gap-4">
					<div class="flex items-center gap-3">
						<input type="checkbox" id="profile-upgrade" bind:checked={profileUpgradeAllowed} class="w-4 h-4 rounded bg-bg-elevated border-white/20 text-white-400 focus:ring-white-400" />
						<label for="profile-upgrade" class="text-sm text-text-secondary">Allow upgrades</label>
					</div>
					<div>
						<label for="profile-upgrade-until" class={labelClass}>Upgrade until score</label>
						<input type="number" id="profile-upgrade-until" bind:value={profileUpgradeUntilScore} class={inputClass} disabled={!profileUpgradeAllowed} />
					</div>
				</div>

				<div class="grid sm:grid-cols-2 gap-4">
					<div>
						<label for="profile-min-score" class={labelClass}>Minimum format score</label>
						<input type="number" id="profile-min-score" bind:value={profileMinFormatScore} class={inputClass} />
						<p class="text-xs text-text-muted mt-1">Releases below this score are rejected</p>
					</div>
					<div>
						<label for="profile-cutoff" class={labelClass}>Cutoff score</label>
						<input type="number" id="profile-cutoff" bind:value={profileCutoffFormatScore} class={inputClass} />
						<p class="text-xs text-text-muted mt-1">Stop upgrading once reached</p>
					</div>
				</div>

				<div class="flex gap-2">
					<button type="submit" class="liquid-btn">
						{editingProfile ? 'Update Profile' : 'Add Profile'}
					</button>
					{#if editingProfile}
						<button type="button" class="liquid-btn !bg-white/5 !border-t-white/10 text-text-secondary hover:text-white" onclick={resetProfileForm}>
							Cancel
						</button>
					{/if}
				</div>
			</form>
		{/if}

		{#if qualityProfiles.length === 0}
			<p class="text-text-muted py-4">No quality profiles configured. Add one to control release quality preferences.</p>
		{:else}
			<div class="space-y-2">
				{#each qualityProfiles as profile}
					<div class="p-4 bg-bg-elevated/50 rounded-xl border border-white/5">
						<div class="flex items-center justify-between">
							<div>
								<div class="flex items-center gap-2 flex-wrap">
									<h3 class="font-medium text-text-primary">{profile.name}</h3>
									{#if profile.upgradeAllowed}
										<span class="px-2 py-0.5 text-xs rounded-lg bg-green-900/50 text-green-300">Upgrades</span>
									{/if}
								</div>
								<p class="text-sm text-text-secondary mt-1">
									{formatProfileQualities(profile.qualities)}
								</p>
								<p class="text-xs text-text-muted mt-1">
									Min: {profile.minFormatScore} | Cutoff: {profile.cutoffFormatScore}
									{#if profile.upgradeAllowed}
										| Until: {profile.upgradeUntilScore}
									{/if}
								</p>
							</div>
							<div class="flex gap-2">
								<button
									class="liquid-btn-sm"
									onclick={() => startEditProfile(profile)}
								>
									Edit
								</button>
								<button
									class="liquid-btn-sm !bg-white/5 !border-t-white/10 text-text-secondary hover:text-white"
									onclick={() => handleDeleteProfile(profile.id)}
								>
									Delete
								</button>
							</div>
						</div>
					</div>
				{/each}
			</div>
		{/if}
	</section>
</div>
