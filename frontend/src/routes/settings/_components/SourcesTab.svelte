<script lang="ts">
	import { onMount } from 'svelte';
	import Select from '$lib/components/ui/Select.svelte';
	import TMDBSettings from './TMDBSettings.svelte';
	import TraktSettings from './TraktSettings.svelte';
	import OpenSubtitlesSettings from './OpenSubtitlesSettings.svelte';
	import { toast } from '$lib/stores/toast';
	import {
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
		getProwlarrConfig,
		saveProwlarrConfig,
		testProwlarrConnection,
		syncProwlarr,
		getIndexerTags,
		type ProwlarrConfig,
		type IndexerTag,
		type DownloadClient,
		type Indexer
	} from '$lib/api';

	// Download clients state
	let downloadClients: DownloadClient[] = $state([]);
	let showAddClientForm = $state(false);
	let editingClient: DownloadClient | null = $state(null);
	let testing: Record<number, boolean> = $state({});
	let testResults: Record<number, { success: boolean; message: string }> = $state({});

	// Client form state
	let clientName = $state('');
	let clientType: DownloadClient['type'] = $state('qbittorrent');
	let clientHost = $state('');
	let clientPort = $state(8080);
	let clientUsername = $state('');
	let clientPassword = $state('');
	let clientApiKey = $state('');
	let clientUseTls = $state(false);
	let clientCategory = $state('');

	// Indexers state
	let indexers: Indexer[] = $state([]);
	let showAddIndexerForm = $state(false);
	let testingIndexer: Record<number, boolean> = $state({});
	let indexerTestResults: Record<number, { success: boolean; message: string }> = $state({});

	// Indexer form state
	let indexerName = $state('');
	let indexerType: Indexer['type'] = $state('torznab');
	let indexerUrl = $state('');
	let indexerApiKey = $state('');
	let indexerCategories = $state('');

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

	const inputClass = "liquid-input w-full px-4 py-2.5";
	const labelClass = "block text-sm text-text-secondary mb-1.5";

	onMount(async () => {
		await Promise.all([loadDownloadClients(), loadIndexers(), loadProwlarrConfig()]);
	});

	// Download client functions
	async function loadDownloadClients() {
		try {
			downloadClients = await getDownloadClients();
		} catch (e) {
			console.error('Failed to load download clients:', e);
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

	function getDefaultPort(type: string): number {
		switch (type) {
			case 'qbittorrent': return 8080;
			case 'transmission': return 9091;
			case 'sabnzbd': return 8080;
			case 'nzbget': return 6789;
			default: return 8080;
		}
	}

	async function handleAddClient() {
		try {
			if (editingClient) {
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
			toast.error(editingClient ? 'Failed to update client' : 'Failed to add client');
		}
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
			toast.error('Failed to delete client');
		}
	}

	async function handleToggleClient(client: DownloadClient) {
		try {
			await updateDownloadClient(client.id, { ...client, enabled: !client.enabled });
			await loadDownloadClients();
		} catch (e) {
			console.error('Failed to update download client:', e);
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

	// Indexer functions
	async function loadIndexers() {
		try {
			indexers = await getIndexers();
		} catch (e) {
			console.error('Failed to load indexers:', e);
		}
	}

	function resetIndexerForm() {
		indexerName = '';
		indexerType = 'torznab';
		indexerUrl = '';
		indexerApiKey = '';
		indexerCategories = '';
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
			toast.error('Failed to add indexer');
		}
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
			toast.error('Failed to delete indexer');
		}
	}

	async function handleToggleIndexer(idx: Indexer) {
		try {
			await updateIndexer(idx.id, { ...idx, enabled: !idx.enabled });
			await loadIndexers();
		} catch (e) {
			console.error('Failed to update indexer:', e);
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
</script>

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
