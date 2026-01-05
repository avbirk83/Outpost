<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import { getSystemStatus, type SystemStatus } from '$lib/api';

	interface Props {
		isAdmin?: boolean;
	}

	let { isAdmin = false }: Props = $props();

	let status: SystemStatus | null = $state(null);
	let statusInterval: ReturnType<typeof setInterval> | null = null;

	onMount(() => {
		loadStatus();
		statusInterval = setInterval(loadStatus, 10000);
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

	function getDiskPercent(): number {
		if (!status || status.diskTotal === 0) return 0;
		return Math.round((status.diskUsed / status.diskTotal) * 100);
	}
</script>

{#if status && isAdmin}
	<div class="fixed bottom-0 left-0 right-0 h-8 bg-[#111111]/95 backdrop-blur-sm border-t border-white/10 flex items-center justify-between px-4 text-xs z-40">
		<!-- Left: Activity -->
		<div class="flex items-center gap-4">
			{#if status.runningTasks.length > 0}
				<div class="flex items-center gap-2 text-blue-400">
					<div class="w-3 h-3 border-2 border-blue-400 border-t-transparent rounded-full animate-spin"></div>
					<span>{status.runningTasks.join(', ')}</span>
				</div>
			{:else}
				<span class="text-text-muted">Idle</span>
			{/if}
		</div>

		<!-- Center: Quick stats -->
		<div class="flex items-center gap-6">
			{#if status.activeDownloads > 0}
				<div class="flex items-center gap-1.5 text-blue-400">
					<svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4" />
					</svg>
					<span>{status.activeDownloads} downloading</span>
				</div>
			{/if}
			{#if status.pendingRequests > 0}
				<div class="flex items-center gap-1.5 text-amber-400">
					<svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 7h12m0 0l-4-4m4 4l-4 4m0 6H4m0 0l4 4m-4-4l4-4" />
					</svg>
					<span>{status.pendingRequests} pending</span>
				</div>
			{/if}
		</div>

		<!-- Right: Disk usage -->
		<div class="flex items-center gap-2">
			<svg class="w-3.5 h-3.5 text-text-muted" fill="none" stroke="currentColor" viewBox="0 0 24 24">
				<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 7v10c0 2.21 3.582 4 8 4s8-1.79 8-4V7M4 7c0 2.21 3.582 4 8 4s8-1.79 8-4M4 7c0-2.21 3.582-4 8-4s8 1.79 8 4" />
			</svg>
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
		</div>
	</div>
{/if}
