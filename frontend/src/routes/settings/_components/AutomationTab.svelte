<script lang="ts">
	import { onMount } from 'svelte';
	import type { ScheduledTask } from '$lib/api';
	import { getSettings, saveSettings } from '$lib/api';

	interface Props {
		tasks: ScheduledTask[];
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
		triggeringTask,
		editingTaskInterval,
		savingTask,
		onTriggerTask,
		onUpdateTask,
		onSaveTaskInterval,
		onEditInterval
	}: Props = $props();

	// Upgrade settings state
	let upgradeEnabled = $state(false);
	let upgradeLimit = $state(10);
	let upgradeInterval = $state(720);
	let upgradeDeleteOld = $state(true);
	let savingUpgrade = $state(false);
	let upgradeSaved = $state(false);

	onMount(async () => {
		await loadUpgradeSettings();
	});

	async function loadUpgradeSettings() {
		try {
			const settings = await getSettings();
			upgradeEnabled = settings.upgrade_search_enabled === 'true';
			upgradeLimit = parseInt(settings.upgrade_search_limit) || 10;
			upgradeInterval = parseInt(settings.upgrade_search_interval) || 720;
			upgradeDeleteOld = settings.upgrade_delete_old !== 'false';
		} catch (e) {
			console.error('Failed to load upgrade settings:', e);
		}
	}

	async function handleSaveUpgradeSettings() {
		savingUpgrade = true;
		try {
			await saveSettings({
				upgrade_search_enabled: upgradeEnabled ? 'true' : 'false',
				upgrade_search_limit: upgradeLimit.toString(),
				upgrade_search_interval: upgradeInterval.toString(),
				upgrade_delete_old: upgradeDeleteOld ? 'true' : 'false'
			});
			upgradeSaved = true;
			setTimeout(() => upgradeSaved = false, 3000);
		} catch (e) {
			console.error('Failed to save upgrade settings:', e);
		} finally {
			savingUpgrade = false;
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
</script>

<!-- Quality Upgrades -->
<section class="glass-card p-6 space-y-4">
	<div class="flex items-center gap-3">
		<div class="w-10 h-10 rounded-xl bg-green-600/20 flex items-center justify-center">
			<svg class="w-5 h-5 text-green-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
				<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 10l7-7m0 0l7 7m-7-7v18" />
			</svg>
		</div>
		<div>
			<h2 class="text-lg font-semibold text-text-primary">Quality Upgrades</h2>
			<p class="text-sm text-text-secondary">Automatically search for better quality versions</p>
		</div>
	</div>

	<div class="space-y-4">
		<label class="flex items-center gap-3 cursor-pointer">
			<button
				type="button"
				class="relative w-12 h-6 rounded-full transition-colors {upgradeEnabled ? 'bg-green-600' : 'bg-gray-600'}"
				onclick={() => upgradeEnabled = !upgradeEnabled}
			>
				<span class="absolute left-1 top-1 w-4 h-4 bg-white rounded-full transition-transform duration-200 {upgradeEnabled ? 'translate-x-6' : ''}"></span>
			</button>
			<div>
				<span class="text-text-primary font-medium">Enable Upgrade Search</span>
				<p class="text-xs text-text-muted">Periodically search for higher quality versions of owned media</p>
			</div>
		</label>

		{#if upgradeEnabled}
			<div class="grid grid-cols-1 md:grid-cols-2 gap-4 pt-2">
				<div>
					<label class="block text-sm text-text-secondary mb-1">Items per Search</label>
					<p class="text-xs text-text-muted mb-2">How many items to search at a time</p>
					<input
						type="number"
						min="1"
						max="50"
						bind:value={upgradeLimit}
						class="w-full px-3 py-2 text-sm bg-bg-elevated border border-border-subtle rounded-lg text-text-primary focus:outline-none focus:border-cream/50"
					/>
				</div>
				<div>
					<label class="block text-sm text-text-secondary mb-1">Search Interval (minutes)</label>
					<p class="text-xs text-text-muted mb-2">How often to run upgrade searches</p>
					<input
						type="number"
						min="60"
						max="10080"
						bind:value={upgradeInterval}
						class="w-full px-3 py-2 text-sm bg-bg-elevated border border-border-subtle rounded-lg text-text-primary focus:outline-none focus:border-cream/50"
					/>
				</div>
			</div>

			<label class="flex items-center gap-2 cursor-pointer pt-2">
				<input type="checkbox" bind:checked={upgradeDeleteOld} class="form-checkbox" />
				<div>
					<span class="text-sm text-text-secondary">Delete old file after upgrade</span>
					<p class="text-xs text-text-muted">Automatically remove lower quality file when upgrade is imported</p>
				</div>
			</label>
		{/if}

		<div class="flex items-center gap-3 pt-2">
			<button class="liquid-btn" onclick={handleSaveUpgradeSettings} disabled={savingUpgrade}>
				{savingUpgrade ? 'Saving...' : 'Save Upgrade Settings'}
			</button>
			<a
				href="/upgrades"
				class="px-4 py-2 text-sm rounded-lg bg-bg-elevated hover:bg-bg-card border border-border-subtle text-text-secondary hover:text-text-primary transition-colors flex items-center gap-2"
			>
				<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14" />
				</svg>
				View Upgrades
			</a>
			{#if upgradeSaved}
				<span class="text-sm text-green-400 flex items-center gap-1">
					<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
					</svg>
					Saved!
				</span>
			{/if}
		</div>
	</div>
</section>

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
