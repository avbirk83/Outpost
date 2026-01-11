<script lang="ts">
	import { goto } from '$app/navigation';
	import { onMount } from 'svelte';
	import DirectoryBrowser from '$lib/components/DirectoryBrowser.svelte';
	import { toast } from '$lib/stores/toast';
	import {
		getSetupWizardStatus,
		completeSetupWizard,
		getLibraries,
		createLibrary,
		scanLibrary,
		getDownloadClients,
		createDownloadClient,
		testDownloadClient,
		getProwlarrConfig,
		saveProwlarrConfig,
		testProwlarrConnection,
		syncProwlarr,
		getQualityPresets,
		type Library,
		type DownloadClient,
		type QualityPreset,
		type SetupWizardStatus
	} from '$lib/api';

	// Wizard state
	let currentStep = $state(0);
	let loading = $state(true);
	let wizardStatus: SetupWizardStatus | null = $state(null);

	// Step definitions
	const steps = [
		{ id: 'welcome', title: 'Welcome', required: true },
		{ id: 'library', title: 'Add Library', required: true },
		{ id: 'download', title: 'Download Client', required: false },
		{ id: 'indexers', title: 'Indexers', required: false },
		{ id: 'quality', title: 'Quality', required: false },
		{ id: 'complete', title: 'Complete', required: true }
	];

	// Library state
	let libraries: Library[] = $state([]);
	let libraryName = $state('Movies');
	let libraryPath = $state('');
	let libraryType: Library['type'] = $state('movies');
	let showBrowser = $state(false);
	let addingLibrary = $state(false);

	const typeLabels: Record<string, string> = {
		movies: 'Movies',
		tv: 'TV Shows',
		anime: 'Anime',
		music: 'Music',
		books: 'Books'
	};

	// Download client state
	let downloadClients: DownloadClient[] = $state([]);
	let clientName = $state('');
	let clientType: DownloadClient['type'] = $state('qbittorrent');
	let clientHost = $state('localhost');
	let clientPort = $state(8080);
	let clientUsername = $state('');
	let clientPassword = $state('');
	let clientApiKey = $state('');
	let clientUseTls = $state(false);
	let clientCategory = $state('');
	let testingClient = $state(false);
	let clientTestResult: { success: boolean; message?: string; error?: string } | null = $state(null);

	// Prowlarr state
	let prowlarrUrl = $state('');
	let prowlarrApiKey = $state('');
	let testingProwlarr = $state(false);
	let syncingProwlarr = $state(false);
	let prowlarrTestResult: { success: boolean; error?: string; indexerCount?: number } | null = $state(null);
	let prowlarrSyncResult: { success: boolean; error?: string; synced?: number } | null = $state(null);
	let savingProwlarr = $state(false);

	// Quality state
	let qualityPresets: QualityPreset[] = $state([]);

	// Completion state
	let completing = $state(false);
	let scanningLibraries = $state(false);

	onMount(async () => {
		try {
			wizardStatus = await getSetupWizardStatus();

			// If setup is already complete, redirect to home
			if (wizardStatus.setupCompleted) {
				goto('/');
				return;
			}

			// Load existing data
			await Promise.all([
				loadLibraries(),
				loadDownloadClients(),
				loadProwlarrConfig(),
				loadQualityPresets()
			]);
		} catch (e) {
			console.error('Failed to load setup status:', e);
		} finally {
			loading = false;
		}
	});

	async function loadLibraries() {
		try {
			libraries = await getLibraries();
		} catch { /* Ignore */ }
	}

	async function loadDownloadClients() {
		try {
			downloadClients = await getDownloadClients();
		} catch { /* Ignore */ }
	}

	async function loadProwlarrConfig() {
		try {
			const config = await getProwlarrConfig();
			if (config) {
				prowlarrUrl = config.url || '';
				prowlarrApiKey = config.apiKey || '';
			}
		} catch { /* Ignore */ }
	}

	async function loadQualityPresets() {
		try {
			qualityPresets = await getQualityPresets();
		} catch { /* Ignore */ }
	}

	// Auto-update library name when type changes
	$effect(() => {
		libraryName = typeLabels[libraryType] || 'Library';
	});

	function nextStep() {
		if (currentStep < steps.length - 1) {
			currentStep++;
		}
	}

	function prevStep() {
		if (currentStep > 0) {
			currentStep--;
		}
	}

	function skipStep() {
		nextStep();
	}

	// Library functions
	async function handleAddLibrary() {
		if (!libraryPath.trim()) {
			toast.error('Please select a directory');
			return;
		}

		addingLibrary = true;
		try {
			await createLibrary({
				name: libraryName,
				path: libraryPath,
				type: libraryType,
				scanInterval: 24
			});
			await loadLibraries();
			libraryPath = '';
			toast.success('Library added');
		} catch (e) {
			toast.error('Failed to add library');
		} finally {
			addingLibrary = false;
		}
	}

	// Download client functions
	async function handleTestClient() {
		// For setup wizard, we create the client first to test it
		// This simplifies the flow and allows using the standard test endpoint
		testingClient = true;
		clientTestResult = null;
		try {
			// Create the client
			const client = await createDownloadClient({
				name: clientName || clientType,
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
			// Test it
			const result = await testDownloadClient(client.id);
			clientTestResult = result;
			if (result.success) {
				await loadDownloadClients();
				// Reset form
				clientName = '';
				clientHost = 'localhost';
				clientPort = 8080;
				clientUsername = '';
				clientPassword = '';
				clientApiKey = '';
				toast.success('Download client added and connected');
			} else {
				// Delete the client if test failed
				await fetch(`/api/download-clients/${client.id}`, { method: 'DELETE' });
				clientTestResult = { success: false, error: result.error || result.message || 'Connection failed' };
			}
		} catch (e) {
			clientTestResult = { success: false, error: 'Failed to add client' };
		} finally {
			testingClient = false;
		}
	}

	// Prowlarr functions
	async function handleTestProwlarr() {
		if (!prowlarrUrl || !prowlarrApiKey) {
			toast.error('Please enter URL and API key');
			return;
		}

		testingProwlarr = true;
		prowlarrTestResult = null;
		try {
			const result = await testProwlarrConnection(prowlarrUrl, prowlarrApiKey);
			prowlarrTestResult = result;
		} catch (e) {
			prowlarrTestResult = { success: false, error: 'Connection failed' };
		} finally {
			testingProwlarr = false;
		}
	}

	async function handleSaveProwlarr() {
		savingProwlarr = true;
		try {
			await saveProwlarrConfig({
				url: prowlarrUrl,
				apiKey: prowlarrApiKey,
				autoSync: true,
				syncIntervalHours: 24
			});
			toast.success('Prowlarr configured');
		} catch (e) {
			toast.error('Failed to save Prowlarr config');
		} finally {
			savingProwlarr = false;
		}
	}

	async function handleSyncProwlarr() {
		syncingProwlarr = true;
		prowlarrSyncResult = null;
		try {
			const result = await syncProwlarr();
			prowlarrSyncResult = result;
			toast.success(`Synced ${result.synced || 0} indexers`);
		} catch (e) {
			prowlarrSyncResult = { success: false, error: 'Sync failed' };
		} finally {
			syncingProwlarr = false;
		}
	}

	// Completion functions
	async function handleComplete(startScan: boolean = false) {
		completing = true;
		try {
			if (startScan && libraries.length > 0) {
				scanningLibraries = true;
				// Start scanning all libraries
				for (const lib of libraries) {
					try {
						await scanLibrary(lib.id);
					} catch { /* Ignore scan errors */ }
				}
			}

			await completeSetupWizard();
			goto('/');
		} catch (e) {
			toast.error('Failed to complete setup');
			completing = false;
			scanningLibraries = false;
		}
	}

	// Check if can proceed from current step
	const canProceed = $derived(() => {
		switch (steps[currentStep].id) {
			case 'library':
				return libraries.length > 0;
			default:
				return true;
		}
	});

	// Get default port for client type
	function getDefaultPort(type: string): number {
		switch (type) {
			case 'qbittorrent': return 8080;
			case 'transmission': return 9091;
			case 'sabnzbd': return 8080;
			case 'nzbget': return 6789;
			default: return 8080;
		}
	}

	$effect(() => {
		clientPort = getDefaultPort(clientType);
	});
</script>

<svelte:head>
	<title>Setup - Outpost</title>
</svelte:head>

<div class="setup-page">
	<!-- Background -->
	<div class="setup-bg">
		<div class="bg-gradient"></div>
	</div>

	{#if loading}
		<div class="loading-state">
			<div class="spinner"></div>
			<p>Loading...</p>
		</div>
	{:else}
		<div class="setup-content">
			<!-- Logo -->
			<div class="logo-section">
				<img src="/outpost-banner.png" alt="Outpost" class="logo-banner" />
			</div>

			<!-- Progress indicator -->
			<div class="progress-dots">
				{#each steps as step, i}
					<button
						class="dot"
						class:active={i === currentStep}
						class:completed={i < currentStep}
						onclick={() => { if (i < currentStep) currentStep = i; }}
						disabled={i > currentStep}
					>
						{#if i < currentStep}
							<svg class="check-icon" fill="none" stroke="currentColor" viewBox="0 0 24 24">
								<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
							</svg>
						{/if}
					</button>
				{/each}
			</div>

			<!-- Step content -->
			<div class="step-card">
				{#if steps[currentStep].id === 'welcome'}
					<!-- Welcome Step -->
					<div class="step-content welcome-step">
						<h1>Welcome to Outpost</h1>
						<p class="subtitle">Your personal media server</p>
						<p class="description">
							Outpost helps you organize and stream your movies, TV shows, music, and books.
							Let's get your server set up in just a few steps.
						</p>
						<div class="step-actions">
							<button class="primary-btn" onclick={nextStep}>
								Get Started
								<svg class="btn-icon" fill="none" stroke="currentColor" viewBox="0 0 24 24">
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7" />
								</svg>
							</button>
						</div>
					</div>

				{:else if steps[currentStep].id === 'library'}
					<!-- Library Step -->
					<div class="step-content">
						<h2>Add Your Media Library</h2>
						<p class="step-subtitle">Tell Outpost where your media files are stored</p>

						{#if libraries.length > 0}
							<div class="added-items">
								{#each libraries as lib}
									<div class="item-chip">
										<span class="chip-icon">
											{#if lib.type === 'movies'}
												<svg fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M7 4v16M17 4v16M3 8h4m10 0h4M3 12h18M3 16h4m10 0h4M4 20h16a1 1 0 001-1V5a1 1 0 00-1-1H4a1 1 0 00-1 1v14a1 1 0 001 1z" /></svg>
											{:else if lib.type === 'tv' || lib.type === 'anime'}
												<svg fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9.75 17L9 20l-1 1h8l-1-1-.75-3M3 13h18M5 17h14a2 2 0 002-2V5a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z" /></svg>
											{:else if lib.type === 'music'}
												<svg fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 19V6l12-3v13M9 19c0 1.105-1.343 2-3 2s-3-.895-3-2 1.343-2 3-2 3 .895 3 2zm12-3c0 1.105-1.343 2-3 2s-3-.895-3-2 1.343-2 3-2 3 .895 3 2zM9 10l12-3" /></svg>
											{:else}
												<svg fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 6.253v13m0-13C10.832 5.477 9.246 5 7.5 5S4.168 5.477 3 6.253v13C4.168 18.477 5.754 18 7.5 18s3.332.477 4.5 1.253m0-13C13.168 5.477 14.754 5 16.5 5c1.747 0 3.332.477 4.5 1.253v13C19.832 18.477 18.247 18 16.5 18c-1.746 0-3.332.477-4.5 1.253" /></svg>
											{/if}
										</span>
										<span class="chip-text">{lib.name}</span>
									</div>
								{/each}
							</div>
						{/if}

						<div class="form-section">
							<div class="form-row">
								<label>Library Type</label>
								<select bind:value={libraryType} class="select-input">
									<option value="movies">Movies</option>
									<option value="tv">TV Shows</option>
									<option value="anime">Anime</option>
									<option value="music">Music</option>
									<option value="books">Books</option>
								</select>
							</div>

							<div class="form-row">
								<label>Name</label>
								<input type="text" bind:value={libraryName} class="text-input" placeholder="Library name" />
							</div>

							<div class="form-row">
								<label>Path</label>
								<div class="path-input-group">
									<input type="text" bind:value={libraryPath} class="text-input" placeholder="/path/to/media" />
									<button class="browse-btn" onclick={() => showBrowser = true}>Browse</button>
								</div>
							</div>

							<button
								class="secondary-btn add-btn"
								onclick={handleAddLibrary}
								disabled={addingLibrary || !libraryPath}
							>
								{addingLibrary ? 'Adding...' : 'Add Library'}
							</button>
						</div>

						<div class="step-actions">
							<button class="back-btn" onclick={prevStep}>Back</button>
							<button class="primary-btn" onclick={nextStep} disabled={!canProceed()}>
								Continue
								<svg class="btn-icon" fill="none" stroke="currentColor" viewBox="0 0 24 24">
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7" />
								</svg>
							</button>
						</div>
					</div>

				{:else if steps[currentStep].id === 'download'}
					<!-- Download Client Step -->
					<div class="step-content">
						<h2>Connect Download Client</h2>
						<p class="step-subtitle">Connect your download client for automated downloads</p>

						{#if downloadClients.length > 0}
							<div class="added-items">
								{#each downloadClients as client}
									<div class="item-chip">
										<span class="chip-icon">
											<svg fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4" /></svg>
										</span>
										<span class="chip-text">{client.name}</span>
									</div>
								{/each}
							</div>
						{/if}

						<div class="form-section">
							<div class="form-row">
								<label>Type</label>
								<select bind:value={clientType} class="select-input">
									<option value="qbittorrent">qBittorrent</option>
									<option value="transmission">Transmission</option>
									<option value="sabnzbd">SABnzbd</option>
									<option value="nzbget">NZBGet</option>
								</select>
							</div>

							<div class="form-row">
								<label>Name (optional)</label>
								<input type="text" bind:value={clientName} class="text-input" placeholder={clientType} />
							</div>

							<div class="form-row-split">
								<div class="form-row">
									<label>Host</label>
									<input type="text" bind:value={clientHost} class="text-input" placeholder="localhost" />
								</div>
								<div class="form-row" style="max-width: 120px;">
									<label>Port</label>
									<input type="number" bind:value={clientPort} class="text-input" />
								</div>
							</div>

							{#if clientType === 'sabnzbd' || clientType === 'nzbget'}
								<div class="form-row">
									<label>API Key</label>
									<input type="password" bind:value={clientApiKey} class="text-input" placeholder="API key" />
								</div>
							{:else}
								<div class="form-row-split">
									<div class="form-row">
										<label>Username</label>
										<input type="text" bind:value={clientUsername} class="text-input" placeholder="Username" />
									</div>
									<div class="form-row">
										<label>Password</label>
										<input type="password" bind:value={clientPassword} class="text-input" placeholder="Password" />
									</div>
								</div>
							{/if}

							<div class="form-row">
								<label>Category (optional)</label>
								<input type="text" bind:value={clientCategory} class="text-input" placeholder="outpost" />
							</div>

							<div class="form-row checkbox-row">
								<label class="checkbox-label">
									<input type="checkbox" bind:checked={clientUseTls} />
									<span>Use HTTPS</span>
								</label>
							</div>

							{#if clientTestResult}
								<div class="test-result" class:success={clientTestResult.success} class:error={!clientTestResult.success}>
									{clientTestResult.success ? 'Connection successful' : (clientTestResult.error || clientTestResult.message || 'Connection failed')}
								</div>
							{/if}

							<div class="form-buttons">
								<button class="secondary-btn add-btn" onclick={handleTestClient} disabled={testingClient || !clientHost}>
									{testingClient ? 'Adding & Testing...' : 'Add & Test Connection'}
								</button>
							</div>
						</div>

						<div class="step-actions">
							<button class="back-btn" onclick={prevStep}>Back</button>
							<div class="action-group">
								<button class="skip-btn" onclick={skipStep}>Skip for now</button>
								<button class="primary-btn" onclick={nextStep}>
									Continue
									<svg class="btn-icon" fill="none" stroke="currentColor" viewBox="0 0 24 24">
										<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7" />
									</svg>
								</button>
							</div>
						</div>
					</div>

				{:else if steps[currentStep].id === 'indexers'}
					<!-- Indexers Step -->
					<div class="step-content">
						<h2>Connect Indexers</h2>
						<p class="step-subtitle">Connect to Prowlarr to sync your indexers automatically</p>

						<div class="form-section">
							<div class="form-row">
								<label>Prowlarr URL</label>
								<input type="text" bind:value={prowlarrUrl} class="text-input" placeholder="http://localhost:9696" />
							</div>

							<div class="form-row">
								<label>API Key</label>
								<input type="password" bind:value={prowlarrApiKey} class="text-input" placeholder="Prowlarr API key" />
							</div>

							{#if prowlarrTestResult}
								<div class="test-result" class:success={prowlarrTestResult.success} class:error={!prowlarrTestResult.success}>
									{prowlarrTestResult.success ? 'Connection successful' : (prowlarrTestResult.error || 'Connection failed')}
									{#if prowlarrTestResult.indexerCount}
										<span class="result-detail"> - {prowlarrTestResult.indexerCount} indexers found</span>
									{/if}
								</div>
							{/if}

							{#if prowlarrSyncResult}
								<div class="test-result" class:success={prowlarrSyncResult.success} class:error={!prowlarrSyncResult.success}>
									{prowlarrSyncResult.success ? `Synced ${prowlarrSyncResult.synced || 0} indexers` : (prowlarrSyncResult.error || 'Sync failed')}
								</div>
							{/if}

							<div class="form-buttons">
								<button class="secondary-btn" onclick={handleTestProwlarr} disabled={testingProwlarr || !prowlarrUrl || !prowlarrApiKey}>
									{testingProwlarr ? 'Testing...' : 'Test Connection'}
								</button>
								{#if prowlarrTestResult?.success}
									<button class="secondary-btn" onclick={handleSaveProwlarr} disabled={savingProwlarr}>
										{savingProwlarr ? 'Saving...' : 'Save Config'}
									</button>
									<button class="secondary-btn add-btn" onclick={handleSyncProwlarr} disabled={syncingProwlarr}>
										{syncingProwlarr ? 'Syncing...' : 'Sync Indexers'}
									</button>
								{/if}
							</div>
						</div>

						<p class="helper-text">You can also add indexers manually in Settings later.</p>

						<div class="step-actions">
							<button class="back-btn" onclick={prevStep}>Back</button>
							<div class="action-group">
								<button class="skip-btn" onclick={skipStep}>Skip for now</button>
								<button class="primary-btn" onclick={nextStep}>
									Continue
									<svg class="btn-icon" fill="none" stroke="currentColor" viewBox="0 0 24 24">
										<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7" />
									</svg>
								</button>
							</div>
						</div>
					</div>

				{:else if steps[currentStep].id === 'quality'}
					<!-- Quality Step -->
					<div class="step-content">
						<h2>Quality Preferences</h2>
						<p class="step-subtitle">Choose your preferred quality settings</p>

						<div class="quality-info">
							<p>Outpost comes with default quality presets that work for most users.</p>
							<p>You can customize these later in Settings.</p>
						</div>

						{#if qualityPresets.filter(p => p.enabled).length > 0}
							<div class="preset-list">
								{#each qualityPresets.filter(p => p.enabled) as preset}
									<div class="preset-item">
										<div class="preset-info">
											<span class="preset-name">{preset.name}</span>
											<span class="preset-type">{preset.mediaType || 'movie'}</span>
										</div>
										<div class="preset-details">
											{preset.resolution} - {preset.source}
										</div>
									</div>
								{/each}
							</div>
						{:else}
							<div class="no-presets">
								<p>No quality presets configured yet.</p>
								<p class="helper-text">Default presets will be created automatically.</p>
							</div>
						{/if}

						<div class="step-actions">
							<button class="back-btn" onclick={prevStep}>Back</button>
							<div class="action-group">
								<button class="skip-btn" onclick={skipStep}>Use defaults</button>
								<button class="primary-btn" onclick={nextStep}>
									Continue
									<svg class="btn-icon" fill="none" stroke="currentColor" viewBox="0 0 24 24">
										<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7" />
									</svg>
								</button>
							</div>
						</div>
					</div>

				{:else if steps[currentStep].id === 'complete'}
					<!-- Complete Step -->
					<div class="step-content complete-step">
						<div class="complete-icon">
							<svg fill="none" stroke="currentColor" viewBox="0 0 24 24">
								<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" />
							</svg>
						</div>
						<h2>You're All Set!</h2>
						<p class="step-subtitle">Your Outpost server is ready to go</p>

						<div class="summary">
							<div class="summary-item" class:completed={libraries.length > 0}>
								<span class="summary-icon">
									{#if libraries.length > 0}
										<svg fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" /></svg>
									{:else}
										<svg fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" /></svg>
									{/if}
								</span>
								<span class="summary-text">
									{libraries.length} {libraries.length === 1 ? 'library' : 'libraries'} added
								</span>
							</div>
							<div class="summary-item" class:completed={downloadClients.length > 0}>
								<span class="summary-icon">
									{#if downloadClients.length > 0}
										<svg fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" /></svg>
									{:else}
										<svg fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" /></svg>
									{/if}
								</span>
								<span class="summary-text">
									{#if downloadClients.length > 0}
										Download client configured
									{:else}
										Download client skipped
									{/if}
								</span>
							</div>
							<div class="summary-item" class:completed={prowlarrUrl !== ''}>
								<span class="summary-icon">
									{#if prowlarrUrl}
										<svg fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" /></svg>
									{:else}
										<svg fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" /></svg>
									{/if}
								</span>
								<span class="summary-text">
									{#if prowlarrUrl}
										Indexers connected
									{:else}
										Indexers skipped
									{/if}
								</span>
							</div>
						</div>

						<div class="step-actions complete-actions">
							<button class="back-btn" onclick={prevStep}>Back</button>
							<div class="complete-buttons">
								{#if libraries.length > 0}
									<button
										class="primary-btn scan-btn"
										onclick={() => handleComplete(true)}
										disabled={completing}
									>
										{#if scanningLibraries}
											<span class="btn-spinner"></span>
											Scanning...
										{:else}
											Start Scanning Libraries
										{/if}
									</button>
								{/if}
								<button
									class="secondary-btn"
									onclick={() => handleComplete(false)}
									disabled={completing}
								>
									{completing && !scanningLibraries ? 'Completing...' : 'Go to Home'}
								</button>
							</div>
						</div>
					</div>
				{/if}
			</div>
		</div>
	{/if}
</div>

<!-- Directory Browser Modal -->
{#if showBrowser}
	<DirectoryBrowser
		bind:open={showBrowser}
		currentPath={libraryPath || '/'}
		onSelect={(p) => { libraryPath = p; showBrowser = false; }}
	/>
{/if}

<style>
	.setup-page {
		min-height: 100vh;
		width: 100%;
		display: flex;
		align-items: center;
		justify-content: center;
		position: relative;
		background: #0a0a0a;
		overflow: hidden;
	}

	.setup-bg {
		position: fixed;
		inset: 0;
		z-index: 0;
	}

	.bg-gradient {
		position: absolute;
		inset: 0;
		background: radial-gradient(
			ellipse at 50% 0%,
			rgba(232, 168, 73, 0.08) 0%,
			transparent 50%
		);
	}

	.loading-state {
		display: flex;
		flex-direction: column;
		align-items: center;
		gap: 1rem;
		color: rgba(245, 230, 200, 0.5);
	}

	.spinner {
		width: 40px;
		height: 40px;
		border: 3px solid rgba(245, 230, 200, 0.2);
		border-top-color: #F5E6C8;
		border-radius: 50%;
		animation: spin 0.8s linear infinite;
	}

	@keyframes spin {
		to { transform: rotate(360deg); }
	}

	.setup-content {
		position: relative;
		z-index: 10;
		display: flex;
		flex-direction: column;
		align-items: center;
		padding: 2rem;
		width: 100%;
		max-width: 560px;
	}

	.logo-section {
		margin-bottom: 2rem;
	}

	.logo-banner {
		height: 48px;
		width: auto;
		filter: drop-shadow(0 4px 12px rgba(0, 0, 0, 0.5));
	}

	/* Progress dots */
	.progress-dots {
		display: flex;
		gap: 0.75rem;
		margin-bottom: 2rem;
	}

	.dot {
		width: 12px;
		height: 12px;
		border-radius: 50%;
		background: rgba(245, 230, 200, 0.2);
		border: none;
		cursor: pointer;
		transition: all 0.2s ease;
		display: flex;
		align-items: center;
		justify-content: center;
		padding: 0;
	}

	.dot:disabled {
		cursor: default;
	}

	.dot.active {
		background: #E8A849;
		width: 24px;
		border-radius: 6px;
	}

	.dot.completed {
		background: #22c55e;
	}

	.check-icon {
		width: 8px;
		height: 8px;
		color: white;
	}

	/* Step card */
	.step-card {
		width: 100%;
		background: rgba(17, 17, 17, 0.85);
		backdrop-filter: blur(20px);
		border: 1px solid rgba(245, 230, 200, 0.1);
		border-radius: 20px;
		padding: 2rem;
	}

	.step-content {
		display: flex;
		flex-direction: column;
	}

	.step-content h1,
	.step-content h2 {
		font-size: 1.5rem;
		font-weight: 600;
		color: #F5E6C8;
		text-align: center;
		margin-bottom: 0.5rem;
	}

	.subtitle,
	.step-subtitle {
		font-size: 0.875rem;
		color: rgba(245, 230, 200, 0.5);
		text-align: center;
		margin-bottom: 1.5rem;
	}

	.description {
		font-size: 0.9375rem;
		color: rgba(245, 230, 200, 0.7);
		text-align: center;
		line-height: 1.6;
		margin-bottom: 2rem;
	}

	/* Welcome step */
	.welcome-step {
		text-align: center;
	}

	/* Form sections */
	.form-section {
		margin-bottom: 1.5rem;
	}

	.form-row {
		margin-bottom: 1rem;
	}

	.form-row label {
		display: block;
		font-size: 0.8125rem;
		font-weight: 500;
		color: rgba(245, 230, 200, 0.7);
		margin-bottom: 0.5rem;
	}

	.form-row-split {
		display: grid;
		grid-template-columns: 1fr 1fr;
		gap: 1rem;
	}

	.text-input,
	.select-input {
		width: 100%;
		padding: 0.75rem 1rem;
		background: rgba(255, 255, 255, 0.05);
		border: 1px solid rgba(245, 230, 200, 0.1);
		border-radius: 10px;
		color: #F5E6C8;
		font-size: 0.9375rem;
		transition: all 0.2s ease;
	}

	.text-input::placeholder {
		color: rgba(245, 230, 200, 0.3);
	}

	.text-input:focus,
	.select-input:focus {
		outline: none;
		border-color: rgba(245, 230, 200, 0.3);
		background: rgba(255, 255, 255, 0.08);
	}

	.select-input {
		cursor: pointer;
	}

	.select-input option {
		background: #1a1a1a;
		color: #F5E6C8;
	}

	.path-input-group {
		display: flex;
		gap: 0.5rem;
	}

	.path-input-group .text-input {
		flex: 1;
	}

	.browse-btn {
		padding: 0.75rem 1rem;
		background: rgba(245, 230, 200, 0.1);
		border: 1px solid rgba(245, 230, 200, 0.2);
		border-radius: 10px;
		color: #F5E6C8;
		font-size: 0.875rem;
		font-weight: 500;
		cursor: pointer;
		transition: all 0.2s ease;
		white-space: nowrap;
	}

	.browse-btn:hover {
		background: rgba(245, 230, 200, 0.15);
	}

	.checkbox-row {
		margin-top: 0.5rem;
	}

	.checkbox-label {
		display: flex !important;
		align-items: center;
		gap: 0.5rem;
		cursor: pointer;
	}

	.checkbox-label input {
		width: 18px;
		height: 18px;
		accent-color: #E8A849;
	}

	/* Added items */
	.added-items {
		display: flex;
		flex-wrap: wrap;
		gap: 0.5rem;
		margin-bottom: 1.5rem;
	}

	.item-chip {
		display: flex;
		align-items: center;
		gap: 0.5rem;
		padding: 0.5rem 0.75rem;
		background: rgba(34, 197, 94, 0.15);
		border: 1px solid rgba(34, 197, 94, 0.3);
		border-radius: 8px;
		color: #86efac;
		font-size: 0.8125rem;
	}

	.chip-icon {
		width: 16px;
		height: 16px;
	}

	.chip-icon svg {
		width: 100%;
		height: 100%;
	}

	/* Test results */
	.test-result {
		padding: 0.75rem 1rem;
		border-radius: 8px;
		font-size: 0.875rem;
		margin-bottom: 1rem;
	}

	.test-result.success {
		background: rgba(34, 197, 94, 0.15);
		border: 1px solid rgba(34, 197, 94, 0.3);
		color: #86efac;
	}

	.test-result.error {
		background: rgba(239, 68, 68, 0.15);
		border: 1px solid rgba(239, 68, 68, 0.3);
		color: #fca5a5;
	}

	.result-detail {
		opacity: 0.8;
	}

	.form-buttons {
		display: flex;
		gap: 0.75rem;
		flex-wrap: wrap;
	}

	.helper-text {
		font-size: 0.8125rem;
		color: rgba(245, 230, 200, 0.4);
		text-align: center;
		margin-top: 1rem;
	}

	/* Quality section */
	.quality-info {
		text-align: center;
		margin-bottom: 1.5rem;
	}

	.quality-info p {
		color: rgba(245, 230, 200, 0.6);
		font-size: 0.875rem;
		margin-bottom: 0.5rem;
	}

	.preset-list {
		display: flex;
		flex-direction: column;
		gap: 0.75rem;
		margin-bottom: 1.5rem;
	}

	.preset-item {
		display: flex;
		justify-content: space-between;
		align-items: center;
		padding: 0.75rem 1rem;
		background: rgba(255, 255, 255, 0.05);
		border: 1px solid rgba(245, 230, 200, 0.1);
		border-radius: 10px;
	}

	.preset-info {
		display: flex;
		align-items: center;
		gap: 0.75rem;
	}

	.preset-name {
		font-weight: 500;
		color: #F5E6C8;
	}

	.preset-type {
		font-size: 0.75rem;
		padding: 0.25rem 0.5rem;
		background: rgba(245, 230, 200, 0.1);
		border-radius: 4px;
		color: rgba(245, 230, 200, 0.6);
		text-transform: uppercase;
	}

	.preset-details {
		font-size: 0.8125rem;
		color: rgba(245, 230, 200, 0.5);
	}

	.no-presets {
		text-align: center;
		padding: 1.5rem;
		background: rgba(255, 255, 255, 0.03);
		border-radius: 10px;
		margin-bottom: 1.5rem;
	}

	.no-presets p {
		color: rgba(245, 230, 200, 0.6);
		font-size: 0.875rem;
	}

	/* Complete step */
	.complete-step {
		align-items: center;
	}

	.complete-icon {
		width: 64px;
		height: 64px;
		margin-bottom: 1rem;
		color: #22c55e;
	}

	.complete-icon svg {
		width: 100%;
		height: 100%;
	}

	.summary {
		display: flex;
		flex-direction: column;
		gap: 0.75rem;
		width: 100%;
		margin: 1.5rem 0;
	}

	.summary-item {
		display: flex;
		align-items: center;
		gap: 0.75rem;
		padding: 0.75rem 1rem;
		background: rgba(255, 255, 255, 0.03);
		border-radius: 10px;
		color: rgba(245, 230, 200, 0.5);
	}

	.summary-item.completed {
		color: #86efac;
		background: rgba(34, 197, 94, 0.08);
	}

	.summary-icon {
		width: 20px;
		height: 20px;
	}

	.summary-icon svg {
		width: 100%;
		height: 100%;
	}

	.complete-actions {
		width: 100%;
	}

	.complete-buttons {
		display: flex;
		flex-direction: column;
		gap: 0.75rem;
		align-items: stretch;
	}

	/* Buttons */
	.step-actions {
		display: flex;
		justify-content: space-between;
		align-items: center;
		margin-top: 1.5rem;
		padding-top: 1.5rem;
		border-top: 1px solid rgba(245, 230, 200, 0.1);
	}

	.action-group {
		display: flex;
		align-items: center;
		gap: 1rem;
	}

	.primary-btn {
		display: flex;
		align-items: center;
		justify-content: center;
		gap: 0.5rem;
		padding: 0.75rem 1.5rem;
		background: #E8A849;
		color: #000;
		border: none;
		border-radius: 10px;
		font-size: 0.9375rem;
		font-weight: 600;
		cursor: pointer;
		transition: all 0.2s ease;
	}

	.primary-btn:hover:not(:disabled) {
		background: #F0C06A;
		transform: translateY(-1px);
	}

	.primary-btn:disabled {
		opacity: 0.5;
		cursor: not-allowed;
	}

	.btn-icon {
		width: 18px;
		height: 18px;
	}

	.secondary-btn {
		padding: 0.75rem 1.25rem;
		background: rgba(245, 230, 200, 0.1);
		border: 1px solid rgba(245, 230, 200, 0.2);
		border-radius: 10px;
		color: #F5E6C8;
		font-size: 0.875rem;
		font-weight: 500;
		cursor: pointer;
		transition: all 0.2s ease;
	}

	.secondary-btn:hover:not(:disabled) {
		background: rgba(245, 230, 200, 0.15);
	}

	.secondary-btn:disabled {
		opacity: 0.5;
		cursor: not-allowed;
	}

	.secondary-btn.add-btn {
		background: rgba(34, 197, 94, 0.15);
		border-color: rgba(34, 197, 94, 0.3);
		color: #86efac;
	}

	.secondary-btn.add-btn:hover:not(:disabled) {
		background: rgba(34, 197, 94, 0.25);
	}

	.back-btn {
		padding: 0.75rem 1rem;
		background: transparent;
		border: none;
		color: rgba(245, 230, 200, 0.5);
		font-size: 0.875rem;
		cursor: pointer;
		transition: color 0.2s ease;
	}

	.back-btn:hover {
		color: #F5E6C8;
	}

	.skip-btn {
		padding: 0.5rem;
		background: transparent;
		border: none;
		color: rgba(245, 230, 200, 0.4);
		font-size: 0.8125rem;
		cursor: pointer;
		transition: color 0.2s ease;
	}

	.skip-btn:hover {
		color: rgba(245, 230, 200, 0.7);
	}

	.scan-btn {
		width: 100%;
	}

	.btn-spinner {
		width: 18px;
		height: 18px;
		border: 2px solid rgba(0, 0, 0, 0.2);
		border-top-color: #000;
		border-radius: 50%;
		animation: spin 0.8s linear infinite;
	}

	@media (max-width: 480px) {
		.setup-content {
			padding: 1.5rem;
		}

		.step-card {
			padding: 1.5rem;
			border-radius: 16px;
		}

		.form-row-split {
			grid-template-columns: 1fr;
		}

		.action-group {
			flex-direction: column-reverse;
			align-items: stretch;
			gap: 0.5rem;
		}

		.skip-btn {
			text-align: center;
		}
	}
</style>
