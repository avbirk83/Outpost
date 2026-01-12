<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import DirectoryBrowser from '$lib/components/DirectoryBrowser.svelte';
	import {
		AutomationTab,
		BackupSection,
		GeneralTab,
		HealthTab,
		LogsTab,
		QualityTab,
		SourcesTab,
		StorageTab
	} from './_components';
	import { toast } from '$lib/stores/toast';
	import { auth } from '$lib/stores/auth';
	import {
		getLibraries,
		createLibrary,
		deleteLibrary,
		scanLibrary,
		getScanProgress,
		getTasks,
		updateTask,
		triggerTask,
		type Library,
		type ScanProgress,
		type ScheduledTask
	} from '$lib/api';

	// Library state
	let libraries: Library[] = $state([]);
	let loading = $state(true);
	let error: string | null = $state(null);
	let showAddForm = $state(false);
	let scanning: Record<number, boolean> = $state({});
	let scanProgress: ScanProgress | null = $state(null);
	let progressInterval: ReturnType<typeof setInterval> | null = null;

	// Library form state
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
		if (name === '' || name === lastAutoName) {
			name = label;
			lastAutoName = label;
		}
	});

	// Directory browser state
	let showBrowser = $state(false);

	// Task state
	let tasks: ScheduledTask[] = $state([]);
	let taskRefreshInterval: ReturnType<typeof setInterval> | null = null;
	let triggeringTask: Record<number, boolean> = $state({});
	let editingTaskInterval: Record<number, number> = $state({});
	let savingTask: Record<number, boolean> = $state({});

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
		await Promise.all([loadLibraries(), loadTasks()]);
		checkScanProgress();
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

	// Library functions
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
			startProgressPolling();
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to start scan';
			scanning[id] = false;
		}
	}

	function startProgressPolling() {
		if (progressInterval) return;
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
				for (const id of Object.keys(scanning)) {
					scanning[Number(id)] = false;
				}
			}
		} catch (e) {
			console.error('Failed to get scan progress:', e);
			stopProgressPolling();
		}
	}

	// Task functions
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

	<!-- GENERAL TAB -->
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

		{#if isAdmin}
			<BackupSection />
		{/if}
	{/if}

	<!-- SOURCES TAB -->
	{#if currentTab === 'sources'}
		<SourcesTab />
	{/if}

	<!-- QUALITY TAB -->
	{#if currentTab === 'quality'}
		<QualityTab />
	{/if}

	<!-- AUTOMATION TAB -->
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

	<!-- STORAGE TAB -->
	{#if currentTab === 'storage' && isAdmin}
		<StorageTab />
	{/if}

	<!-- HEALTH TAB -->
	{#if currentTab === 'health' && isAdmin}
		<HealthTab />
	{/if}

	<!-- LOGS TAB -->
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
