<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import { goto } from '$app/navigation';
	import { getSystemStatus, type SystemStatus } from '$lib/api';
	import {
		statusBar,
		searchProgress,
		activityEvents,
		searchStats,
		type ActivityEvent,
		type EventType
	} from '$lib/stores/statusBar';

	interface Props {
		isAdmin?: boolean;
	}

	let { isAdmin = false }: Props = $props();

	let status: SystemStatus | null = $state(null);
	let statusInterval: ReturnType<typeof setInterval> | null = null;
	let expanded = $state(false);

	// Subscribe to stores
	let search = $derived($searchProgress);
	let events = $derived($activityEvents);
	let stats = $derived($searchStats);

	onMount(() => {
		loadStatus();
		statusInterval = setInterval(loadStatus, 5000); // Faster refresh for status
	});

	onDestroy(() => {
		if (statusInterval) clearInterval(statusInterval);
	});

	async function loadStatus() {
		try {
			status = await getSystemStatus();
		} catch (e) {
			// Silently fail
		}
	}

	function formatBytes(bytes: number): string {
		if (bytes === 0) return '0 B';
		const gb = bytes / (1024 * 1024 * 1024);
		if (gb >= 1000) return (gb / 1024).toFixed(1) + ' TB';
		if (gb >= 1) return gb.toFixed(1) + ' GB';
		const mb = bytes / (1024 * 1024);
		return mb.toFixed(0) + ' MB';
	}

	function formatSpeed(bytesPerSec: number): string {
		if (bytesPerSec === 0) return '0 B/s';
		const mbps = bytesPerSec / (1024 * 1024);
		if (mbps >= 1) return mbps.toFixed(1) + ' MB/s';
		const kbps = bytesPerSec / 1024;
		return kbps.toFixed(0) + ' KB/s';
	}

	function getDiskPercent(): number {
		if (!status || status.diskTotal === 0) return 0;
		return Math.round((status.diskUsed / status.diskTotal) * 100);
	}

	function getDiskWarning(): boolean {
		return getDiskPercent() > 90;
	}

	function getEventIcon(type: EventType): string {
		switch (type) {
			case 'import': return 'M5 13l4 4L19 7';
			case 'grab': return 'M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4';
			case 'approval': return 'M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z';
			case 'error': return 'M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z';
			case 'search': return 'M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z';
			case 'download': return 'M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4';
			case 'system': return 'M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z';
			default: return 'M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z';
		}
	}

	function getEventColor(type: EventType): string {
		switch (type) {
			case 'import': return 'text-green-400';
			case 'grab': return 'text-blue-400';
			case 'approval': return 'text-amber-400';
			case 'error': return 'text-red-400';
			case 'search': return 'text-purple-400';
			case 'download': return 'text-blue-400';
			case 'system': return 'text-text-muted';
			default: return 'text-text-muted';
		}
	}

	function formatRelativeTime(date: Date): string {
		const now = new Date();
		const diff = now.getTime() - date.getTime();
		const minutes = Math.floor(diff / 60000);
		const hours = Math.floor(diff / 3600000);

		if (minutes < 1) return 'just now';
		if (minutes < 60) return `${minutes}m ago`;
		if (hours < 24) return `${hours}h ago`;
		return date.toLocaleDateString();
	}

	function handleClick(link?: string) {
		if (link) {
			goto(link);
			expanded = false;
		}
	}

	function toggleExpanded() {
		expanded = !expanded;
	}
</script>

{#if status && isAdmin}
	<!-- Activity Feed Panel (expandable) -->
	{#if expanded}
		<!-- svelte-ignore a11y_click_events_have_key_events a11y_no_static_element_interactions -->
		<div
			class="fixed inset-0 z-30"
			onclick={() => expanded = false}
		></div>
		<div class="fixed bottom-8 left-0 right-0 max-h-80 bg-[#111111]/98 backdrop-blur-md border-t border-white/10 z-40 overflow-hidden animate-slideUp">
			<div class="flex items-center justify-between px-4 py-2 border-b border-white/5">
				<h3 class="text-sm font-medium text-text-primary">Recent Activity</h3>
				<button
					class="text-text-muted hover:text-text-primary transition-colors"
					onclick={() => expanded = false}
					aria-label="Close activity feed"
				>
					<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
					</svg>
				</button>
			</div>
			<div class="overflow-y-auto max-h-64 p-2">
				{#if events.length === 0}
					<div class="text-center py-8 text-text-muted text-sm">
						No recent activity
					</div>
				{:else}
					<div class="space-y-1">
						{#each events.slice(0, 20) as event}
							<!-- svelte-ignore a11y_click_events_have_key_events a11y_no_static_element_interactions -->
							<div
								class="flex items-start gap-3 px-3 py-2 rounded-lg hover:bg-white/5 transition-colors {event.link ? 'cursor-pointer' : ''}"
								onclick={() => handleClick(event.link)}
							>
								<svg class="w-4 h-4 mt-0.5 flex-shrink-0 {getEventColor(event.type)}" fill="none" stroke="currentColor" viewBox="0 0 24 24">
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d={getEventIcon(event.type)} />
								</svg>
								<div class="flex-1 min-w-0">
									<p class="text-sm text-text-primary truncate">{event.message}</p>
									{#if event.details}
										<p class="text-xs text-text-muted truncate">{event.details}</p>
									{/if}
								</div>
								<span class="text-xs text-text-muted flex-shrink-0">
									{formatRelativeTime(event.timestamp)}
								</span>
							</div>
						{/each}
					</div>
				{/if}
			</div>
		</div>
	{/if}

	<!-- Main Status Bar -->
	<div class="fixed bottom-0 left-0 right-0 h-8 bg-[#111111]/95 backdrop-blur-sm border-t border-white/10 flex items-center justify-end px-4 text-xs z-40">
		<!-- All content on right side -->
		<div class="flex items-center gap-4">
			<!-- Downloads/Pending stats -->
			{#if status.activeDownloads > 0}
				<button
					class="flex items-center gap-1.5 text-blue-400 hover:text-blue-300 transition-colors"
					onclick={() => goto('/activity')}
					aria-label="View downloads"
				>
					<svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4" />
					</svg>
					<span>{status.activeDownloads} downloading</span>
				</button>
			{/if}
			{#if status.pendingRequests > 0}
				<button
					class="flex items-center gap-1.5 text-amber-400 hover:text-amber-300 transition-colors"
					onclick={() => goto('/activity')}
					aria-label="View pending requests"
				>
					<svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
					</svg>
					<span>{status.pendingRequests} pending</span>
				</button>
			{/if}

			<!-- Separator if we have stats -->
			{#if status.activeDownloads > 0 || status.pendingRequests > 0}
				<div class="w-px h-4 bg-white/10"></div>
			{/if}

			<!-- Activity / Search Status -->
			<button
				class="flex items-center gap-2 hover:bg-white/5 px-2 py-1 rounded transition-colors"
				onclick={toggleExpanded}
				aria-label="Toggle activity feed"
			>
				{#if search.state === 'searching'}
					<div class="flex items-center gap-2 text-purple-400">
						<div class="spinner-xs"></div>
						<span>Searching {stats.total} indexers...</span>
						<div class="flex items-center gap-1 text-xs">
							{#if stats.success > 0}
								<span class="text-green-400">{stats.success}</span>
							{/if}
							{#if stats.failed > 0}
								<span class="text-red-400">{stats.failed}</span>
							{/if}
							{#if stats.pending > 0}
								<span class="text-text-muted">{stats.pending}</span>
							{/if}
						</div>
					</div>
				{:else if search.state === 'scoring'}
					<div class="flex items-center gap-2 text-amber-400">
						<div class="spinner-xs"></div>
						<span>Scoring {search.totalResults} results...</span>
					</div>
				{:else if search.totalResults > 0}
					<div class="flex items-center gap-2 text-green-400">
						<svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
						</svg>
						<span>Found {search.totalResults} results</span>
					</div>
				{:else if status.activeSearch}
					<div class="flex items-center gap-2 text-purple-400">
						<div class="spinner-xs"></div>
						<svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
						</svg>
						<span>Searching: {status.activeSearch}</span>
					</div>
				{:else if status.runningTasks.length > 0}
					<div class="flex items-center gap-2 text-blue-400">
						<div class="spinner-xs"></div>
						<span>{status.runningTasks.join(', ')}</span>
					</div>
				{:else}
					<span class="text-text-muted flex items-center gap-1.5">
						<svg class="w-3 h-3" fill="currentColor" viewBox="0 0 24 24">
							<circle cx="12" cy="12" r="3" />
						</svg>
						Idle
					</span>
				{/if}
			</button>

			<!-- Separator -->
			<div class="w-px h-4 bg-white/10"></div>

			<!-- Disk usage -->
			<button
				class="flex items-center gap-2 hover:bg-white/5 -mr-2 px-2 py-1 rounded transition-colors {getDiskWarning() ? 'text-red-400' : ''}"
				onclick={() => goto('/settings?tab=storage')}
				aria-label="View storage settings"
			>
				{#if getDiskWarning()}
					<svg class="w-3.5 h-3.5 text-red-400 animate-pulse" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
					</svg>
				{:else}
					<svg class="w-3.5 h-3.5 text-text-muted" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 7v10c0 2.21 3.582 4 8 4s8-1.79 8-4V7M4 7c0 2.21 3.582 4 8 4s8-1.79 8-4M4 7c0-2.21 3.582-4 8-4s8 1.79 8 4" />
					</svg>
				{/if}
				<div class="flex items-center gap-1.5">
					<div class="w-20 h-1.5 rounded-full bg-white/10 overflow-hidden">
						<div
							class="h-full rounded-full transition-all duration-300 {getDiskPercent() > 90 ? 'bg-red-500' : getDiskPercent() > 75 ? 'bg-amber-500' : 'bg-green-500'}"
							style="width: {getDiskPercent()}%"
						></div>
					</div>
					<span class="text-text-muted">
						{formatBytes(status.diskUsed)} / {formatBytes(status.diskTotal)}
					</span>
				</div>
			</button>
		</div>
	</div>
{/if}

<style>
	@keyframes slideUp {
		from {
			transform: translateY(100%);
			opacity: 0;
		}
		to {
			transform: translateY(0);
			opacity: 1;
		}
	}

	.animate-slideUp {
		animation: slideUp 0.2s ease-out;
	}
</style>
