<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import Select from '$lib/components/ui/Select.svelte';
	import DirectoryBrowser from '$lib/components/DirectoryBrowser.svelte';
	import AutomationTab from './_components/AutomationTab.svelte';
	import BackupSection from './_components/BackupSection.svelte';
	import GeneralTab from './_components/GeneralTab.svelte';
	import HealthTab from './_components/HealthTab.svelte';
	import LogsTab from './_components/LogsTab.svelte';
	import StorageTab from './_components/StorageTab.svelte';
	import TraktSettings from './_components/TraktSettings.svelte';
	import OpenSubtitlesSettings from './_components/OpenSubtitlesSettings.svelte';
	import FormatFilteringSettings from './_components/FormatFilteringSettings.svelte';
	import TMDBSettings from './_components/TMDBSettings.svelte';
	import { toast } from '$lib/stores/toast';
	import { auth } from '$lib/stores/auth';
	import {
		getLibraries,
		createLibrary,
		deleteLibrary,
		scanLibrary,
		getScanProgress,
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
		toggleQualityPresetEnabled,
		updateQualityPresetPriority,
		getTasks,
		updateTask,
		triggerTask,
		getProwlarrConfig,
		saveProwlarrConfig,
		testProwlarrConnection,
		syncProwlarr,
		getIndexerTags,
		type Library,
		type ProwlarrConfig,
		type IndexerTag,
		type DownloadClient,
		type Indexer,
		type QualityPreset,
		type ScanProgress,
		type ScheduledTask
	} from '$lib/api';

	let libraries: Library[] = $state([]);
	let downloadClients: DownloadClient[] = $state([]);
	let indexers: Indexer[] = $state([]);
	let qualityPresets: QualityPreset[] = $state([]);
	let tasks: ScheduledTask[] = $state([]);
	let taskRefreshInterval: ReturnType<typeof setInterval> | null = null;
	let triggeringTask: Record<number, boolean> = $state({});
	let editingTaskInterval: Record<number, number> = $state({});
	let savingTask: Record<number, boolean> = $state({});
	let loading = $state(true);
	let error: string | null = $state(null);
	let showAddForm = $state(false);
	let showAddClientForm = $state(false);
	let showAddIndexerForm = $state(false);
	let showAddPresetForm = $state(false);
	let editingPreset: QualityPreset | null = $state(null);
	let presetMediaTab: "movie" | "tv" | "anime" = $state("movie");
	const filteredPresets = $derived(qualityPresets.filter(p => (p.mediaType || 'movie') === presetMediaTab));
	let draggingPresetId: number | null = $state(null);
	let dragOverPresetId: number | null = $state(null);
	let scanning: Record<number, boolean> = $state({});
	let testing: Record<number, boolean> = $state({});
	let testingIndexer: Record<number, boolean> = $state({});
	let testResults: Record<number, { success: boolean; message: string }> = $state({});
	let indexerTestResults: Record<number, { success: boolean; message: string }> = $state({});
	let scanProgress: ScanProgress | null = $state(null);
	let progressInterval: ReturnType<typeof setInterval> | null = null;

	// Prowlarr state
	let prowlarrConfig: ProwlarrConfig | null = $state(null);
	let prowlarrUrl = $state('');
	let prowlarrApiKey = $state('');
	let prowlarrAutoSync = $state(true);
	let prowlarrSyncInterval = $state(24);
	let prowlarrTesting = $state(false);
	let prowlarrTestResult: { success: boolean; message: string; indexerCount?: number } | null = $state(null);
	let prowlarrSyncing = $state(false);
	let prowlarrSyncResult: { success: boolean; message: string; synced?: number } | null = $state(null);
	let indexerTags: IndexerTag[] = $state([]);

	// Form state
	let name = $state('');
	let path = $state('');
	let type: Library['type'] = $state('movies');

	// Auto-populate library name when type changes
	const typeLabels: Record<string, string> = {
		movies: 'Movies',
		tv: 'TV Shows',
		anime: 'Anime',
		music: 'Music',
		books: 'Books'
	};
	let lastAutoName = $state('');
	$effect(() => {
		const label = typeLabels[type] || '';
		// Only auto-populate if name is empty or matches last auto-populated value
		if (name === '' || name === lastAutoName) {
			name = label;
			lastAutoName = label;
		}
	});

	// Directory browser state
	let showBrowser = $state(false);

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
	let editingClient: DownloadClient | null = $state(null);

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

	
	// Tab navigation
	type SettingsTab = 'general' | 'quality' | 'sources' | 'automation' | 'storage' | 'health' | 'logs';
	let currentTab: SettingsTab = $state('general');

	// Check if user is admin for admin-only tabs
	const isAdmin = $derived($auth?.role === 'admin');

	const baseTabs: { id: SettingsTab; label: string; icon: string; adminOnly?: boolean }[] = [
		{ id: 'general', label: 'General', icon: 'M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z M15 12a3 3 0 11-6 0 3 3 0 016 0z' },
		{ id: 'quality', label: 'Quality', icon: 'M9 12l2 2 4-4M7.835 4.697a3.42 3.42 0 001.946-.806 3.42 3.42 0 014.438 0 3.42 3.42 0 001.946.806 3.42 3.42 0 013.138 3.138 3.42 3.42 0 00.806 1.946 3.42 3.42 0 010 4.438 3.42 3.42 0 00-.806 1.946 3.42 3.42 0 01-3.138 3.138 3.42 3.42 0 00-1.946.806 3.42 3.42 0 01-4.438 0 3.42 3.42 0 00-1.946-.806 3.42 3.42 0 01-3.138-3.138 3.42 3.42 0 00-.806-1.946 3.42 3.42 0 010-4.438 3.42 3.42 0 00.806-1.946 3.42 3.42 0 013.138-3.138z' },
		{ id: 'sources', label: 'Sources', icon: 'M21 12a9 9 0 01-9 9m9-9a9 9 0 00-9-9m9 9H3m9 9a9 9 0 01-9-9m9 9c1.657 0 3-4.03 3-9s-1.343-9-3-9m0 18c-1.657 0-3-4.03-3-9s1.343-9 3-9m-9 9a9 9 0 019-9' },
		{ id: 'automation', label: 'Automation', icon: 'M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15' },
		{ id: 'storage', label: 'Storage', icon: 'M4 7v10c0 2.21 3.582 4 8 4s8-1.79 8-4V7M4 7c0 2.21 3.582 4 8 4s8-1.79 8-4M4 7c0-2.21 3.582-4 8-4s8 1.79 8 4', adminOnly: true },
		{ id: 'health', label: 'Health', icon: 'M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z', adminOnly: true },
		{ id: 'logs', label: 'Logs', icon: 'M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z', adminOnly: true }
	];

	const tabs = $derived(baseTabs.filter(tab => !tab.adminOnly || isAdmin));

	onMount(async () => {
		await Promise.all([loadLibraries(), loadDownloadClients(), loadIndexers(), loadProwlarrConfig(), loadQualityPresets(), loadTasks()]);
		// Check if a scan is already running
		checkScanProgress();
		// Start task refresh interval
		taskRefreshInterval = setInterval(loadTasks, 5000);
	});

	onDestroy(() => {
		if (progressInterval) {
			clearInterval(progressInterval);
			progressInterval = null;
		}
		if (taskRefreshInterval) {
			clearInterval(taskRefreshInterval);
			taskRefreshInterval = null;
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
			toast.success('Library added');
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to add library';
			toast.error('Failed to add library');
		}
	}

	async function handleDelete(id: number) {
		if (!confirm('Are you sure you want to delete this library?')) return;
		try {
			await deleteLibrary(id);
			await loadLibraries();
			toast.success('Library deleted');
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to delete library';
			toast.error('Failed to delete library');
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
			if (editingClient) {
				// Update existing client
				await updateDownloadClient(editingClient.id, {
					name: clientName,
					type: clientType,
					host: clientHost,
					port: clientPort,
					username: clientUsername || undefined,
					password: clientPassword || undefined,
					apiKey: clientApiKey || undefined,
					useTls: clientUseTls,
					category: clientCategory || undefined,
					priority: editingClient.priority,
					enabled: editingClient.enabled
				});
				toast.success('Download client updated');
			} else {
				// Create new client
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
				toast.success('Download client added');
			}
			resetClientForm();
			showAddClientForm = false;
			await loadDownloadClients();
		} catch (e) {
			error = e instanceof Error ? e.message : editingClient ? 'Failed to update download client' : 'Failed to add download client';
			toast.error(editingClient ? 'Failed to update client' : 'Failed to add client');
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
		editingClient = null;
	}

	function handleEditClient(client: DownloadClient) {
		editingClient = client;
		clientName = client.name;
		clientType = client.type;
		clientHost = client.host;
		clientPort = client.port;
		clientUsername = client.username || '';
		clientPassword = client.password || '';
		clientApiKey = client.apiKey || '';
		clientUseTls = client.useTls;
		clientCategory = client.category || '';
		showAddClientForm = true;
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
			toast.success('Download client deleted');
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to delete download client';
			toast.error('Failed to delete client');
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
	// Prowlarr functions
	async function loadProwlarrConfig() {
		try {
			prowlarrConfig = await getProwlarrConfig();
			if (prowlarrConfig) {
				prowlarrUrl = prowlarrConfig.url;
				prowlarrAutoSync = prowlarrConfig.autoSync;
				prowlarrSyncInterval = prowlarrConfig.syncIntervalHours;
			}
			indexerTags = await getIndexerTags();
		} catch (e) {
			console.error('Failed to load Prowlarr config:', e);
		}
	}

	async function handleProwlarrTest() {
		if (!prowlarrUrl || !prowlarrApiKey) {
			prowlarrTestResult = { success: false, message: 'URL and API key are required' };
			return;
		}
		prowlarrTesting = true;
		prowlarrTestResult = null;
		try {
			const result = await testProwlarrConnection(prowlarrUrl, prowlarrApiKey);
			if (result.success) {
				prowlarrTestResult = { success: true, message: `Connection successful! Found ${result.indexerCount} indexers.`, indexerCount: result.indexerCount };
			} else {
				prowlarrTestResult = { success: false, message: result.error || 'Connection failed' };
			}
		} catch (e) {
			prowlarrTestResult = { success: false, message: 'Connection failed: ' + (e as Error).message };
		}
		prowlarrTesting = false;
	}

	async function handleProwlarrSave() {
		try {
			await saveProwlarrConfig({
				url: prowlarrUrl,
				apiKey: prowlarrApiKey,
				autoSync: prowlarrAutoSync,
				syncIntervalHours: prowlarrSyncInterval
			});
			await loadProwlarrConfig();
			toast.success('Prowlarr settings saved');
		} catch (e) {
			console.error('Failed to save Prowlarr config:', e);
			toast.error('Failed to save Prowlarr settings');
		}
	}

	async function handleProwlarrSync() {
		prowlarrSyncing = true;
		prowlarrSyncResult = null;
		try {
			const result = await syncProwlarr();
			if (result.success) {
				prowlarrSyncResult = { success: true, message: `Synced ${result.synced} indexers from Prowlarr`, synced: result.synced };
				await loadIndexers();
				await loadProwlarrConfig();
			} else {
				prowlarrSyncResult = { success: false, message: result.error || 'Sync failed' };
			}
		} catch (e) {
			prowlarrSyncResult = { success: false, message: 'Sync failed: ' + (e as Error).message };
		}
		prowlarrSyncing = false;
	}

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
			toast.success('Indexer added');
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to add indexer';
			toast.error('Failed to add indexer');
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
			toast.success('Indexer deleted');
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to delete indexer';
			toast.error('Failed to delete indexer');
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
			toast.success('Quality preset added');
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to add quality preset';
			toast.error('Failed to add preset');
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
			toast.success('Quality preset updated');
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to update quality preset';
			toast.error('Failed to update preset');
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
			toast.success('Quality preset deleted');
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to delete quality preset';
			toast.error('Failed to delete preset');
		}
	}

	async function handleSetDefaultPreset(id: number) {
		try {
			await setDefaultQualityPreset(id);
			await loadQualityPresets();
			toast.success('Default preset updated');
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to set default preset';
			toast.error('Failed to set default preset');
		}
	}

	async function updateAnimePresetPreference(id: number, field: string, value: boolean | string) {
		const response = await fetch(`/api/quality/presets/${id}/anime-preferences`, {
			method: 'PATCH',
			headers: { 'Content-Type': 'application/json' },
			body: JSON.stringify({ [field]: value })
		});
		if (!response.ok) {
			throw new Error('Failed to update anime preferences');
		}
	}

	// Drag and drop handlers for preset reordering
	function handlePresetDragStart(e: DragEvent, presetId: number) {
		draggingPresetId = presetId;
		if (e.dataTransfer) {
			e.dataTransfer.effectAllowed = 'move';
			e.dataTransfer.setData('text/plain', String(presetId));
		}
	}

	function handlePresetDragOver(e: DragEvent, presetId: number) {
		e.preventDefault();
		if (e.dataTransfer) {
			e.dataTransfer.dropEffect = 'move';
		}
		if (draggingPresetId !== presetId) {
			dragOverPresetId = presetId;
		}
	}

	function handlePresetDragLeave() {
		dragOverPresetId = null;
	}

	function handlePresetDragEnd() {
		draggingPresetId = null;
		dragOverPresetId = null;
	}

	async function handlePresetDrop(e: DragEvent, targetPresetId: number) {
		e.preventDefault();
		if (draggingPresetId === null || draggingPresetId === targetPresetId) {
			handlePresetDragEnd();
			return;
		}

		// Find the dragged and target presets
		const draggedIndex = qualityPresets.findIndex(p => p.id === draggingPresetId);
		const targetIndex = qualityPresets.findIndex(p => p.id === targetPresetId);

		if (draggedIndex === -1 || targetIndex === -1) {
			handlePresetDragEnd();
			return;
		}

		// Create new array with reordered presets
		const newPresets = [...qualityPresets];
		const [removed] = newPresets.splice(draggedIndex, 1);
		newPresets.splice(targetIndex, 0, removed);

		// Update priorities based on new positions
		try {
			for (let i = 0; i < newPresets.length; i++) {
				if (newPresets[i].priority !== i + 1) {
					await updateQualityPresetPriority(newPresets[i].id, i + 1);
				}
			}
			await loadQualityPresets();
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to reorder presets';
		}

		handlePresetDragEnd();
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

	// Task management functions
	async function loadTasks() {
		try {
			tasks = await getTasks();
		} catch (e) {
			console.error('Failed to load tasks:', e);
		}
	}

	async function handleTriggerTask(taskId: number) {
		triggeringTask[taskId] = true;
		try {
			await triggerTask(taskId);
			await loadTasks();
		} catch (e) {
			console.error('Failed to trigger task:', e);
		} finally {
			triggeringTask[taskId] = false;
		}
	}

	async function handleUpdateTask(task: ScheduledTask, enabled: boolean, intervalMinutes: number) {
		try {
			await updateTask(task.id, enabled, intervalMinutes);
			await loadTasks();
		} catch (e) {
			console.error('Failed to update task:', e);
		}
	}

	async function handleSaveTaskInterval(task: ScheduledTask) {
		const newInterval = editingTaskInterval[task.id];
		if (!newInterval || newInterval === task.intervalMinutes) return;
		savingTask[task.id] = true;
		try {
			await updateTask(task.id, task.enabled, newInterval);
			await loadTasks();
			delete editingTaskInterval[task.id];
		} catch (e) {
			console.error('Failed to save task interval:', e);
		} finally {
			savingTask[task.id] = false;
		}
	}

	function formatDuration(ms: number | null): string {
		if (ms === null) return '-';
		if (ms < 1000) return `${ms}ms`;
		if (ms < 60000) return `${(ms / 1000).toFixed(1)}s`;
		return `${Math.floor(ms / 60000)}m ${Math.round((ms % 60000) / 1000)}s`;
	}

	function formatTimeAgo(dateStr: string | null): string {
		if (!dateStr) return 'Never';
		const date = new Date(dateStr);
		const now = new Date();
		const diff = now.getTime() - date.getTime();
		const mins = Math.floor(diff / 60000);
		if (mins < 1) return 'Just now';
		if (mins < 60) return `${mins}m ago`;
		const hours = Math.floor(mins / 60);
		if (hours < 24) return `${hours}h ago`;
		const days = Math.floor(hours / 24);
		return `${days}d ago`;
	}

	function formatNextRun(dateStr: string | null): string {
		if (!dateStr) return '-';
		const date = new Date(dateStr);
		const now = new Date();
		const diff = date.getTime() - now.getTime();
		if (diff < 0) return 'Overdue';
		const mins = Math.floor(diff / 60000);
		if (mins < 1) return 'Soon';
		if (mins < 60) return `in ${mins}m`;
		const hours = Math.floor(mins / 60);
		if (hours < 24) return `in ${hours}h`;
		return `in ${Math.floor(hours / 24)}d`;
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
		<div class="bg-glass border border-border-subtle text-text-secondary px-4 py-3 rounded-xl flex items-center justify-between">
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
					{currentTab === tab.id ? 'bg-white/10 text-text-primary' : 'text-text-muted hover:text-text-secondary hover:bg-glass'}"
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
		<GeneralTab
			{libraries}
			{loading}
			{showAddForm}
			{name}
			{path}
			{type}
			{scanning}
			{scanProgress}
			onShowAddForm={(show) => showAddForm = show}
			onNameChange={(value) => name = value}
			onPathChange={(value) => path = value}
			onTypeChange={(value) => type = value}
			onAddLibrary={handleAddLibrary}
			onDeleteLibrary={handleDelete}
			onScanLibrary={handleScan}
			onBrowse={() => showBrowser = true}
		/>

		<!-- Backup & Restore (Admin only) -->
		{#if isAdmin}
			<BackupSection />
		{/if}
	{/if}

	<!-- ============================================ -->
	<!-- SOURCES TAB -->
	<!-- ============================================ -->
	{#if currentTab === 'sources'}

	<!-- TMDB -->
	<TMDBSettings />

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

		{#if editingClient}
		<div class="text-sm text-text-secondary">
			Editing: <span class="text-text-primary font-medium">{editingClient.name}</span>
		</div>
		{/if}

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
				<div class="flex items-center">
					<label class="flex items-center gap-2 cursor-pointer">
						<input type="checkbox" bind:checked={clientUseTls} class="form-checkbox" />
						<span class="text-sm text-text-secondary">Use HTTPS</span>
					</label>
				</div>
				<p class="text-xs text-text-muted">Categories are set automatically based on content type (movies-outpost, tv-outpost, etc.)</p>
				<button type="submit" class="liquid-btn">
					{editingClient ? 'Save Changes' : 'Add Client'}
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
									class="liquid-btn-sm !bg-white/5 !border-t-white/10 text-text-secondary hover:text-text-primary"
									onclick={() => handleEditClient(client)}
								>
									Edit
								</button>
								<button
									class="liquid-btn-sm !bg-white/5 !border-t-white/10 text-text-secondary hover:text-red-400"
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

	<!-- Prowlarr Sync -->
	<section class="glass-card p-6 space-y-4">
		<div class="flex items-center justify-between">
			<div class="flex items-center gap-3">
				<div class="w-10 h-10 rounded-xl flex items-center justify-center overflow-hidden">
					<img src="/icons/prowlarr.png" alt="Prowlarr" class="w-full h-full object-cover" />
				</div>
				<div>
					<h2 class="text-lg font-semibold text-text-primary">Prowlarr</h2>
					<p class="text-sm text-text-secondary">Import indexers from Prowlarr with capabilities and tags</p>
				</div>
			</div>
		</div>

		<div class="p-4 bg-bg-elevated/50 rounded-xl space-y-4 border border-white/5">
			<div class="grid sm:grid-cols-2 gap-4">
				<div>
					<label for="prowlarr-url" class={labelClass}>Prowlarr URL</label>
					<input type="url" id="prowlarr-url" bind:value={prowlarrUrl} class={inputClass} placeholder="http://localhost:9696" />
				</div>
				<div>
					<label for="prowlarr-apikey" class={labelClass}>API Key</label>
					<input type="password" id="prowlarr-apikey" bind:value={prowlarrApiKey} class={inputClass} placeholder="API Key from Prowlarr settings" />
				</div>
			</div>
			<div class="grid sm:grid-cols-3 gap-4 items-end">
				<div class="flex items-center gap-2">
					<input type="checkbox" id="prowlarr-autosync" bind:checked={prowlarrAutoSync} class="form-checkbox" />
					<label for="prowlarr-autosync" class="text-sm text-text-secondary">Auto-sync</label>
				</div>
				<div>
					<label for="prowlarr-interval" class={labelClass}>Sync Interval (hours)</label>
					<input type="number" id="prowlarr-interval" bind:value={prowlarrSyncInterval} min="1" max="168" class={inputClass} />
				</div>
				<div class="flex gap-2">
					<button
						class="liquid-btn-sm flex-1"
						onclick={handleProwlarrTest}
						disabled={prowlarrTesting}
					>
						{prowlarrTesting ? 'Testing...' : 'Test'}
					</button>
					<button
						class="liquid-btn-sm flex-1"
						onclick={handleProwlarrSave}
					>
						Save
					</button>
				</div>
			</div>
			{#if prowlarrTestResult}
				<div class="flex items-center justify-between">
					<div class="text-sm {prowlarrTestResult.success ? 'text-green-400' : 'text-red-400'}">
						{prowlarrTestResult.message}
					</div>
					{#if prowlarrTestResult.success}
						<button
							class="liquid-btn-sm"
							onclick={async () => { await handleProwlarrSave(); await handleProwlarrSync(); }}
							disabled={prowlarrSyncing}
						>
							{prowlarrSyncing ? 'Syncing...' : 'Save & Sync'}
						</button>
					{/if}
				</div>
			{/if}

			{#if prowlarrConfig?.lastSync}
				<div class="flex items-center justify-between text-sm text-text-secondary pt-2 border-t border-white/5">
					<span>Last synced: {new Date(prowlarrConfig.lastSync).toLocaleString()}</span>
					<button
						class="liquid-btn-sm"
						onclick={handleProwlarrSync}
						disabled={prowlarrSyncing}
					>
						{prowlarrSyncing ? 'Syncing...' : 'Sync Now'}
					</button>
				</div>
			{:else if prowlarrConfig}
				<div class="flex items-center justify-between text-sm text-text-secondary pt-2 border-t border-white/5">
					<span>Not synced yet</span>
					<button
						class="liquid-btn-sm"
						onclick={handleProwlarrSync}
						disabled={prowlarrSyncing}
					>
						{prowlarrSyncing ? 'Syncing...' : 'Sync Now'}
					</button>
				</div>
			{/if}
			{#if prowlarrSyncResult}
				<div class="text-sm {prowlarrSyncResult.success ? 'text-green-400' : 'text-red-400'}">
					{prowlarrSyncResult.message}
				</div>
			{/if}
		</div>

		{#if indexerTags.length > 0}
			<div class="p-4 bg-bg-elevated/50 rounded-xl border border-white/5">
				<h3 class="text-sm font-medium text-text-primary mb-2">Synced Tags</h3>
				<div class="flex flex-wrap gap-2">
					{#each indexerTags as tag}
						<span class="px-2 py-1 text-xs rounded-lg bg-orange-900/30 text-orange-300 border border-orange-600/20">
							{tag.name} ({tag.indexerCount})
						</span>
					{/each}
				</div>
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
										{#if idx.syncedFromProwlarr}
											<span class="px-2 py-0.5 text-xs rounded-lg bg-orange-900/30 text-orange-300">Prowlarr</span>
										{/if}
									</div>
									{#if idx.syncedFromProwlarr}
										<div class="flex flex-wrap gap-1 mt-1">
											{#if idx.supportsMovies}<span class="px-1.5 py-0.5 text-[10px] rounded bg-white/5 text-text-muted">Movies</span>{/if}
											{#if idx.supportsTV}<span class="px-1.5 py-0.5 text-[10px] rounded bg-white/5 text-text-muted">TV</span>{/if}
											{#if idx.supportsAnime}<span class="px-1.5 py-0.5 text-[10px] rounded bg-white/5 text-text-muted">Anime</span>{/if}
											{#if idx.supportsMusic}<span class="px-1.5 py-0.5 text-[10px] rounded bg-white/5 text-text-muted">Music</span>{/if}
											{#if idx.supportsBooks}<span class="px-1.5 py-0.5 text-[10px] rounded bg-white/5 text-text-muted">Books</span>{/if}
											{#if idx.supportsImdb}<span class="px-1.5 py-0.5 text-[10px] rounded bg-blue-900/30 text-blue-300">IMDB</span>{/if}
											{#if idx.supportsTmdb}<span class="px-1.5 py-0.5 text-[10px] rounded bg-blue-900/30 text-blue-300">TMDB</span>{/if}
											{#if idx.supportsTvdb}<span class="px-1.5 py-0.5 text-[10px] rounded bg-blue-900/30 text-blue-300">TVDB</span>{/if}
										</div>
									{:else}
										<p class="text-sm text-text-secondary truncate max-w-sm">
											{idx.url}
										</p>
									{/if}
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
									class="liquid-btn-sm !bg-white/5 !border-t-white/10 text-text-secondary hover:text-text-primary"
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

	<!-- Trakt.tv -->
	<TraktSettings />

	<!-- OpenSubtitles -->
	<OpenSubtitlesSettings />

	{/if}

	<!-- ============================================ -->
	<!-- QUALITY TAB -->
	<!-- ============================================ -->
	{#if currentTab === 'quality'}

	<!-- Quality Presets -->
	<section class="glass-card p-6 space-y-4">
		<div class="flex items-center justify-between">
			<div class="flex items-center gap-3">
				<div class="w-10 h-10 rounded-xl bg-amber-600/20 flex items-center justify-center">
					<svg class="w-5 h-5 text-amber-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
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

		<!-- Media Type Tabs -->
		<div class="flex gap-1 p-1 bg-bg-card rounded-xl">
			<button
				type="button"
				class="flex-1 px-4 py-2 text-sm font-medium rounded-lg transition-colors {presetMediaTab === 'movie' ? 'bg-cream/20 text-cream border border-cream/30' : 'text-text-secondary hover:text-text-primary hover:bg-glass'}"
				onclick={() => presetMediaTab = 'movie'}
			>
				Movies
			</button>
			<button
				type="button"
				class="flex-1 px-4 py-2 text-sm font-medium rounded-lg transition-colors {presetMediaTab === 'tv' ? 'bg-cream/20 text-cream border border-cream/30' : 'text-text-secondary hover:text-text-primary hover:bg-glass'}"
				onclick={() => presetMediaTab = 'tv'}
			>
				TV Shows
			</button>
			<button
				type="button"
				class="flex-1 px-4 py-2 text-sm font-medium rounded-lg transition-colors {presetMediaTab === 'anime' ? 'bg-cream/20 text-cream border border-cream/30' : 'text-text-secondary hover:text-text-primary hover:bg-glass'}"
				onclick={() => presetMediaTab = 'anime'}
			>
				Anime
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
								class="px-3 py-1.5 text-xs rounded-lg transition-colors font-medium {presetHdrFormats.includes(hdr) ? 'bg-amber-600 text-white' : 'bg-bg-card text-text-muted hover:bg-bg-elevated'}"
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
								class="px-3 py-1.5 text-xs rounded-lg transition-colors font-medium {presetAudioFormats.includes(audio) ? 'bg-amber-600 text-white' : 'bg-bg-card text-text-muted hover:bg-bg-elevated'}"
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
						<input type="checkbox" bind:checked={presetPreferSeasonPacks} class="form-checkbox" />
						<span class="text-sm text-text-secondary">Prefer season packs for TV</span>
					</label>
					<label class="flex items-center gap-2 cursor-pointer">
						<input type="checkbox" bind:checked={presetAutoUpgrade} class="form-checkbox" />
						<span class="text-sm text-text-secondary">Auto-upgrade when better quality found</span>
					</label>
				</div>

				<div class="flex gap-2">
					<button type="submit" class="liquid-btn">
						{editingPreset ? 'Update Preset' : 'Add Preset'}
					</button>
					{#if editingPreset}
						<button type="button" class="liquid-btn !bg-white/5 !border-t-white/10 text-text-secondary hover:text-text-primary" onclick={resetPresetForm}>
							Cancel
						</button>
					{/if}
				</div>
			</form>
		{/if}

		
		{#if filteredPresets.length === 0}
			<p class="text-text-muted py-4">No quality presets configured. Built-in presets will be added on first use.</p>
		{:else}
			<div class="space-y-2">
				{#each filteredPresets as preset}
					<div
						class="p-4 bg-bg-elevated/50 rounded-xl border transition-all
							{preset.isDefault ? 'ring-1 ring-amber-500/50' : ''}
							{!preset.enabled ? 'opacity-50' : ''}
							{draggingPresetId === preset.id ? 'opacity-50 scale-[0.98] border-amber-500/50' : 'border-white/5'}
							{dragOverPresetId === preset.id ? 'border-amber-400 bg-amber-500/10' : ''}"
						draggable="true"
						ondragstart={(e: DragEvent) => handlePresetDragStart(e, preset.id)}
						ondragover={(e: DragEvent) => handlePresetDragOver(e, preset.id)}
						ondragleave={handlePresetDragLeave}
						ondragend={handlePresetDragEnd}
						ondrop={(e: DragEvent) => handlePresetDrop(e, preset.id)}
					>
						<div class="flex items-center justify-between gap-4">
							<!-- Drag Handle -->
							<div class="flex-shrink-0 cursor-grab active:cursor-grabbing text-text-muted hover:text-text-secondary" title="Drag to reorder">
								<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M4 8h16M4 16h16" />
								</svg>
							</div>

							<!-- Enable/Disable Toggle -->
							<button
								class="flex-shrink-0 w-10 h-6 rounded-full transition-colors {preset.enabled ? 'bg-amber-600' : 'bg-white/10'}"
								onclick={async () => {
									try {
										await toggleQualityPresetEnabled(preset.id, !preset.enabled);
										preset.enabled = !preset.enabled;
									} catch (e) {
										console.error('Failed to toggle preset:', e);
									}
								}}
								title={preset.enabled ? 'Disable preset' : 'Enable preset'}
							>
								<div class="w-4 h-4 m-1 rounded-full bg-white transition-transform {preset.enabled ? 'translate-x-4' : 'translate-x-0'}"></div>
							</button>

							<div class="flex-1 min-w-0">
								<div class="flex items-center gap-2 flex-wrap">
									<h3 class="font-medium text-text-primary">{preset.name}</h3>
									<span class="px-2 py-0.5 text-xs rounded-lg {getResolutionColor(preset.resolution)} text-white">
										{preset.resolution.toUpperCase()}
									</span>
									{#if preset.isBuiltIn}
										<span class="px-2 py-0.5 text-xs rounded-lg bg-bg-card text-text-muted">Built-in</span>
									{/if}
									{#if preset.isDefault}
										<span class="px-2 py-0.5 text-xs rounded-lg bg-amber-900/50 text-amber-300">Default</span>
									{/if}
									{#if preset.autoUpgrade}
										<span class="px-2 py-0.5 text-xs rounded-lg bg-green-900/50 text-green-300">Auto-upgrade</span>
									{/if}
									<span class="px-2 py-0.5 text-xs rounded-lg bg-white/5 text-text-muted" title="Priority (lower = higher priority)">
										#{preset.priority}
									</span>
								</div>
								<p class="text-sm text-text-secondary mt-1">
									{getPresetDescription(preset)}
								</p>
								{#if preset.audioFormats && preset.audioFormats.length > 0}
									<p class="text-xs text-text-muted mt-1">
										Audio: {preset.audioFormats.map(a => a.toUpperCase()).join(', ')}
									</p>
								{/if}

								<!-- Anime preferences (editable inline) -->
								{#if preset.mediaType === 'anime'}
									<div class="flex items-center gap-4 mt-3 pt-3 border-t border-white/5">
										<label class="flex items-center gap-2 cursor-pointer">
											<input
												type="checkbox"
												checked={preset.preferDualAudio}
												onchange={async (e) => {
													const target = e.target as HTMLInputElement;
													try {
														await updateAnimePresetPreference(preset.id, 'preferDualAudio', target.checked);
														preset.preferDualAudio = target.checked;
													} catch (err) {
														console.error('Failed to update preference:', err);
													}
												}}
												class="form-checkbox"
											/>
											<span class="text-xs text-text-secondary">Dual Audio</span>
										</label>
										<label class="flex items-center gap-2 cursor-pointer">
											<input
												type="checkbox"
												checked={preset.preferDubbed}
												onchange={async (e) => {
													const target = e.target as HTMLInputElement;
													try {
														await updateAnimePresetPreference(preset.id, 'preferDubbed', target.checked);
														preset.preferDubbed = target.checked;
													} catch (err) {
														console.error('Failed to update preference:', err);
													}
												}}
												class="form-checkbox"
											/>
											<span class="text-xs text-text-secondary">Dubbed</span>
										</label>
										<div class="flex items-center gap-2">
											<span class="text-xs text-text-muted">Language:</span>
											<select
												value={preset.preferredLanguage || 'any'}
												onchange={async (e) => {
													const target = e.target as HTMLSelectElement;
													try {
														await updateAnimePresetPreference(preset.id, 'preferredLanguage', target.value);
														preset.preferredLanguage = target.value;
													} catch (err) {
														console.error('Failed to update preference:', err);
													}
												}}
												class="form-select-sm"
											>
												<option value="any">Any</option>
												<option value="english">English</option>
												<option value="japanese">Japanese</option>
											</select>
										</div>
									</div>
								{/if}
							</div>
							<div class="flex gap-2 flex-shrink-0">
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
										class="liquid-btn-sm !bg-white/5 !border-t-white/10 text-text-secondary hover:text-text-primary"
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

	<!-- Format Filtering -->
	<FormatFilteringSettings />

	{/if}

	<!-- ============================================ -->
	<!-- AUTOMATION TAB -->
	<!-- ============================================ -->
	{#if currentTab === 'automation'}
		<AutomationTab
			{tasks}
			{triggeringTask}
			{editingTaskInterval}
			{savingTask}
			onTriggerTask={handleTriggerTask}
			onUpdateTask={handleUpdateTask}
			onSaveTaskInterval={handleSaveTaskInterval}
			onEditInterval={(taskId, value) => editingTaskInterval[taskId] = value}
		/>
	{/if}

	<!-- ============================================ -->
	<!-- STORAGE TAB -->
	<!-- ============================================ -->
	{#if currentTab === 'storage' && isAdmin}
		<StorageTab />
	{/if}

	<!-- ============================================ -->
	<!-- HEALTH TAB -->
	<!-- ============================================ -->
	{#if currentTab === 'health' && isAdmin}
		<HealthTab />
	{/if}

	<!-- ============================================ -->
	<!-- LOGS TAB -->
	<!-- ============================================ -->
	{#if currentTab === 'logs' && isAdmin}
		<LogsTab />
	{/if}
</div>

<!-- Directory Browser Modal -->
<DirectoryBrowser
	bind:open={showBrowser}
	bind:currentPath={path}
	onSelect={(selectedPath) => path = selectedPath}
/>
