<script lang="ts">
	import { onMount } from 'svelte';
	import type { ScheduledTask, StorageStatus, FormatSettings } from '$lib/api';
	import { getFormatSettings, saveFormatSettings } from '$lib/api';

	interface Props {
		tasks: ScheduledTask[];
		storageStatus: StorageStatus | null;
		triggeringTask: Record<number, boolean>;
		editingTaskInterval: Record<number, number>;
		savingTask: Record<number, boolean>;
		onTriggerTask: (id: number) => void;
		onUpdateTask: (task: ScheduledTask, enabled: boolean, interval: number) => void;
		onSaveTaskInterval: (task: ScheduledTask) => void;
		onEditInterval: (taskId: number, value: number) => void;
	}

	let {
		tasks,
		storageStatus,
		triggeringTask,
		editingTaskInterval,
		savingTask,
		onTriggerTask,
		onUpdateTask,
		onSaveTaskInterval,
		onEditInterval
	}: Props = $props();

	// Format settings state
	let formatSettings: FormatSettings | null = $state(null);
	let savingFormats = $state(false);
	let formatsSaved = $state(false);
	let newFormat = $state('');
	let newRejectKeyword = $state('');

	// All containers and keywords (sorted for display)
	const sortedContainers = $derived(
		[...(formatSettings?.acceptedContainers ?? [])].sort()
	);
	const sortedRejectedKeywords = $derived(
		[...(formatSettings?.rejectedKeywords ?? [])].sort()
	);

	onMount(async () => {
		await loadFormatSettings();
	});

	async function loadFormatSettings() {
		try {
			formatSettings = await getFormatSettings();
		} catch (e) {
			console.error('Failed to load format settings:', e);
		}
	}

	async function handleSaveFormatSettings() {
		if (!formatSettings) return;
		savingFormats = true;
		try {
			await saveFormatSettings(formatSettings);
			formatsSaved = true;
			setTimeout(() => formatsSaved = false, 3000);
		} catch (e) {
			console.error('Failed to save format settings:', e);
		} finally {
			savingFormats = false;
		}
	}

	function resetToDefaults() {
		formatSettings = {
			acceptedContainers: ['mkv', 'mp4', 'avi', 'mov', 'webm', 'm4v', 'ts', 'm2ts', 'wmv', 'flv'],
			rejectedKeywords: [
				// Disc releases
				'bdmv', 'video_ts', 'iso', 'full disc', 'complete disc', 'disc1', 'disc2',
				// Archives
				'rar', 'zip', '7z',
				// Low quality captures
				'cam', 'camrip', 'hdcam', 'hdts', 'telesync', 'telecine', 'ts-scr',
				'dvdscr', 'dvdscreener', 'screener', 'scr', 'r5', 'workprint',
				// Samples
				'sample',
				// 3D
				'3d', 'hsbs', 'hou',
			],
			autoBlocklist: true,
		};
	}

	function addContainer() {
		if (!formatSettings || !newFormat.trim()) return;
		const format = newFormat.trim().toLowerCase().replace(/^\./, ''); // Remove leading dot if present
		if (!format) return;
		if (formatSettings.acceptedContainers.includes(format)) {
			newFormat = '';
			return;
		}
		formatSettings.acceptedContainers = [...formatSettings.acceptedContainers, format];
		newFormat = '';
	}

	function removeContainer(format: string) {
		if (!formatSettings) return;
		formatSettings.acceptedContainers = formatSettings.acceptedContainers.filter(c => c !== format);
	}

	function handleFormatKeydown(e: KeyboardEvent) {
		if (e.key === 'Enter') {
			e.preventDefault();
			addContainer();
		}
	}

	function addRejectKeyword() {
		if (!formatSettings || !newRejectKeyword.trim()) return;
		const keyword = newRejectKeyword.trim().toLowerCase();
		if (!formatSettings.rejectedKeywords) {
			formatSettings.rejectedKeywords = [];
		}
		if (formatSettings.rejectedKeywords.includes(keyword)) {
			newRejectKeyword = '';
			return;
		}
		formatSettings.rejectedKeywords = [...formatSettings.rejectedKeywords, keyword];
		newRejectKeyword = '';
	}

	function removeRejectKeyword(keyword: string) {
		if (!formatSettings) return;
		formatSettings.rejectedKeywords = formatSettings.rejectedKeywords.filter(k => k !== keyword);
	}

	function handleRejectKeydown(e: KeyboardEvent) {
		if (e.key === 'Enter') {
			e.preventDefault();
			addRejectKeyword();
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

	function formatSize(bytes: number): string {
		if (bytes === 0) return '0 B';
		const gb = bytes / (1024 * 1024 * 1024);
		if (gb >= 1000) return (gb / 1024).toFixed(1) + ' TB';
		if (gb >= 1) return gb.toFixed(1) + ' GB';
		const mb = bytes / (1024 * 1024);
		return mb.toFixed(0) + ' MB';
	}

	function getStorageBarColor(usedPercent: number, freeGb: number, thresholdGb: number): string {
		if (freeGb < thresholdGb) return 'bg-red-500';
		if (usedPercent > 80) return 'bg-yellow-500';
		return 'bg-green-500';
	}
</script>

<!-- Scheduled Tasks -->
<section class="glass-card p-6 space-y-4">
	<div class="flex items-center gap-3">
		<div class="w-10 h-10 rounded-xl bg-blue-600/20 flex items-center justify-center">
			<svg class="w-5 h-5 text-blue-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
				<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
			</svg>
		</div>
		<div>
			<h2 class="text-lg font-semibold text-text-primary">Scheduled Tasks</h2>
			<p class="text-sm text-text-secondary">Background jobs and automation</p>
		</div>
	</div>

	{#if tasks && tasks.length > 0}
		<div class="overflow-x-auto">
			<table class="w-full text-sm">
				<thead>
					<tr class="text-xs text-text-muted uppercase tracking-wide border-b border-border-subtle">
						<th class="text-left py-3 px-2 font-medium">Task</th>
						<th class="text-center py-3 px-2 font-medium w-20">Last Run</th>
						<th class="text-center py-3 px-2 font-medium w-20">Duration</th>
						<th class="text-center py-3 px-2 font-medium w-20">Next Run</th>
						<th class="text-center py-3 px-2 font-medium w-28">Interval</th>
						<th class="text-center py-3 px-2 font-medium w-16">Enabled</th>
						<th class="text-center py-3 px-2 font-medium w-20">Action</th>
					</tr>
				</thead>
				<tbody class="divide-y divide-white/5">
					{#each tasks as task (task.id)}
						<tr class="hover:bg-glass transition-colors">
							<td class="py-3 px-2">
								<div class="flex items-center gap-3">
									<div class="w-8 h-8 rounded-lg flex-shrink-0 {task.isRunning ? 'bg-blue-500/20' : task.enabled ? 'bg-green-500/20' : 'bg-gray-500/20'} flex items-center justify-center">
										{#if task.isRunning}
											<div class="spinner-sm text-blue-400"></div>
										{:else if task.enabled}
											<svg class="w-4 h-4 text-green-400" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" /></svg>
										{:else}
											<svg class="w-4 h-4 text-gray-500" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" /></svg>
										{/if}
									</div>
									<div class="flex-1">
										<div class="font-medium text-text-primary">{task.name}</div>
										{#if task.description}
											<div class="text-xs text-text-muted">{task.description}</div>
										{/if}
										{#if task.lastStatus === 'failed'}
											<span class="text-xs text-red-400">Failed</span>
										{/if}
									</div>
								</div>
							</td>
							<td class="py-3 px-2 text-center text-text-secondary text-xs">{formatTimeAgo(task.lastRun)}</td>
							<td class="py-3 px-2 text-center text-text-secondary text-xs">{formatDuration(task.lastDurationMs)}</td>
							<td class="py-3 px-2 text-center text-text-secondary text-xs">{formatNextRun(task.nextRun)}</td>
							<td class="py-3 px-2 text-center">
								<div class="flex items-center gap-1 justify-center">
									<input
										type="number"
										min="1"
										class="liquid-input !w-16 !px-1 !py-1 text-xs text-center"
										value={editingTaskInterval[task.id] ?? task.intervalMinutes}
										oninput={(e) => onEditInterval(task.id, parseInt((e.target as HTMLInputElement).value) || task.intervalMinutes)}
									/>
									<span class="text-xs text-text-muted">min</span>
									{#if editingTaskInterval[task.id] && editingTaskInterval[task.id] !== task.intervalMinutes}
										<button
											class="liquid-btn-sm !px-1.5 !py-0.5 text-xs"
											disabled={savingTask[task.id]}
											onclick={() => onSaveTaskInterval(task)}
										>
											{#if savingTask[task.id]}
												<span class="w-2 h-2 border border-white/50 border-t-white rounded-full animate-spin inline-block"></span>
											{:else}
												<svg class="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" /></svg>
											{/if}
										</button>
									{/if}
								</div>
							</td>
							<td class="py-3 px-2 text-center">
								<button class="relative w-10 h-5 rounded-full transition-colors mx-auto block {task.enabled ? 'bg-green-600' : 'bg-gray-600'}" onclick={() => onUpdateTask(task, !task.enabled, task.intervalMinutes)}>
									<span class="absolute left-0.5 top-0.5 w-4 h-4 bg-white rounded-full transition-transform duration-200 {task.enabled ? 'translate-x-5' : ''}"></span>
								</button>
							</td>
							<td class="py-3 px-2 text-center">
								<button class="liquid-btn-sm !px-3 !py-1 text-xs min-w-[60px]" disabled={task.isRunning || triggeringTask[task.id]} onclick={() => onTriggerTask(task.id)}>
									{#if task.isRunning || triggeringTask[task.id]}
										<span class="w-3 h-3 border-2 border-white/50 border-t-white rounded-full animate-spin inline-block"></span>
									{:else}
										Run
									{/if}
								</button>
							</td>
						</tr>
						{#if task.lastStatus === 'failed' && task.lastError}
							<tr><td colspan="7" class="px-2 pb-3"><div class="p-2 bg-red-500/10 rounded border border-red-500/20 text-xs text-red-400">{task.lastError}</div></td></tr>
						{/if}
					{/each}
				</tbody>
			</table>
		</div>
	{:else}
		<div class="flex items-center gap-3 py-4">
			<div class="spinner-md text-blue-400"></div>
			<span class="text-text-secondary">Loading tasks...</span>
		</div>
	{/if}
</section>

<!-- Format Settings -->
<section class="glass-card p-6 space-y-4">
	<div class="flex items-center gap-3">
		<div class="w-10 h-10 rounded-xl bg-red-600/20 flex items-center justify-center">
			<svg class="w-5 h-5 text-red-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
				<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
			</svg>
		</div>
		<div>
			<h2 class="text-lg font-semibold text-text-primary">Format Filtering</h2>
			<p class="text-sm text-text-secondary">Control which file formats can be downloaded</p>
		</div>
	</div>

	{#if formatSettings}
		<div class="space-y-4">
			<div>
				<label class="block text-sm text-text-secondary mb-2">Accepted Container Formats</label>
				<p class="text-xs text-text-muted mb-3">File extensions that will be accepted. Add or remove as needed.</p>
				<div class="flex items-center gap-2 mb-3">
					<input
						type="text"
						bind:value={newFormat}
						onkeydown={handleFormatKeydown}
						placeholder="e.g. mkv, mp4, avi"
						class="flex-1 max-w-[200px] px-3 py-1.5 text-sm bg-bg-elevated border border-border-subtle rounded-lg text-text-primary placeholder:text-text-muted focus:outline-none focus:border-cream/50"
					/>
					<button
						type="button"
						onclick={addContainer}
						disabled={!newFormat.trim()}
						class="px-3 py-1.5 text-sm rounded-lg bg-green-500/20 text-green-400 hover:bg-green-500/30 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
					>
						Add
					</button>
				</div>
				{#if sortedContainers.length > 0}
					<div class="flex flex-wrap gap-2">
						{#each sortedContainers as container}
							<div class="flex items-center gap-1 px-3 py-1.5 text-sm rounded-lg bg-green-600 text-white font-medium uppercase">
								<span>{container}</span>
								<button
									type="button"
									onclick={() => removeContainer(container)}
									class="ml-1 hover:text-green-200 transition-colors"
									aria-label="Remove {container}"
								>
									<svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
										<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
									</svg>
								</button>
							</div>
						{/each}
					</div>
				{:else}
					<p class="text-xs text-text-muted italic">No accepted formats - all containers will be rejected!</p>
				{/if}
			</div>

			<!-- Rejected Keywords -->
			<div class="pt-4 border-t border-white/5">
				<label class="block text-sm text-text-secondary mb-2">Rejected Keywords</label>
				<p class="text-xs text-text-muted mb-3">Releases containing these keywords will be rejected (case-insensitive).</p>
				<div class="flex items-center gap-2 mb-3">
					<input
						type="text"
						bind:value={newRejectKeyword}
						onkeydown={handleRejectKeydown}
						placeholder="e.g. bdmv, rar, cam, hdts"
						class="flex-1 max-w-[200px] px-3 py-1.5 text-sm bg-bg-elevated border border-border-subtle rounded-lg text-text-primary placeholder:text-text-muted focus:outline-none focus:border-cream/50"
					/>
					<button
						type="button"
						onclick={addRejectKeyword}
						disabled={!newRejectKeyword.trim()}
						class="px-3 py-1.5 text-sm rounded-lg bg-red-500/20 text-red-400 hover:bg-red-500/30 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
					>
						Add
					</button>
				</div>
				{#if sortedRejectedKeywords.length > 0}
					<div class="flex flex-wrap gap-2">
						{#each sortedRejectedKeywords as keyword}
							<div class="flex items-center gap-1 px-3 py-1.5 text-sm rounded-lg bg-red-600 text-white font-medium">
								<span>{keyword}</span>
								<button
									type="button"
									onclick={() => removeRejectKeyword(keyword)}
									class="ml-1 hover:text-red-200 transition-colors"
									aria-label="Remove {keyword}"
								>
									<svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
										<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
									</svg>
								</button>
							</div>
						{/each}
					</div>
				{:else}
					<p class="text-xs text-text-muted italic">No rejected keywords - all releases will be accepted</p>
				{/if}
			</div>

			<!-- Auto-blocklist toggle -->
			<div class="pt-4 border-t border-white/5">
				<label class="flex items-center gap-2 cursor-pointer">
					<input type="checkbox" bind:checked={formatSettings.autoBlocklist} class="form-checkbox" />
					<div>
						<span class="text-sm text-text-secondary">Auto-Blocklist</span>
						<p class="text-xs text-text-muted">Automatically add rejected releases to blocklist</p>
					</div>
				</label>
			</div>

			<div class="flex items-center gap-3 pt-4">
				<button class="liquid-btn" onclick={handleSaveFormatSettings} disabled={savingFormats}>
					{savingFormats ? 'Saving...' : 'Save Format Settings'}
				</button>
				<button
					type="button"
					onclick={resetToDefaults}
					class="px-4 py-2 text-sm rounded-lg bg-bg-elevated hover:bg-bg-card border border-border-subtle text-text-secondary transition-colors"
				>
					Reset to Defaults
				</button>
				{#if formatsSaved}
					<span class="text-sm text-green-400 flex items-center gap-1">
						<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
						</svg>
						Saved!
					</span>
				{/if}
			</div>
		</div>
	{:else}
		<div class="flex items-center gap-3 py-4">
			<div class="spinner-md text-red-400"></div>
			<span class="text-text-secondary">Loading format settings...</span>
		</div>
	{/if}
</section>

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
		{@const totalMedia = storageStatus.moviesSize + storageStatus.tvSize + storageStatus.musicSize + storageStatus.booksSize}

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

		{#if totalMedia > 0}
			<div class="text-sm text-text-secondary">
				Total media: <span class="text-text-primary font-medium">{formatSize(totalMedia)}</span>
			</div>
		{/if}

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
			<div class="spinner-md text-amber"></div>
			<span class="text-text-secondary">Loading storage status...</span>
		</div>
	{/if}
</section>
