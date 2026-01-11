<script lang="ts">
	import { onMount } from 'svelte';
	import {
		getHealthFull,
		recheckHealth,
		type FullHealthResponse,
		type HealthCheck,
		type HealthCheckStatus
	} from '$lib/api';

	let health: FullHealthResponse | null = $state(null);
	let loading = $state(true);
	let error: string | null = $state(null);
	let recheckingChecks: Set<string> = $state(new Set());

	onMount(() => {
		loadHealth();
	});

	async function loadHealth() {
		try {
			loading = true;
			error = null;
			health = await getHealthFull();
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to load health status';
		} finally {
			loading = false;
		}
	}

	async function runSingleCheck(name: string) {
		recheckingChecks = new Set([...recheckingChecks, name]);
		try {
			const result = await recheckHealth(name);
			// Update the check in the health response
			if (health) {
				const index = health.checks.findIndex(c => c.name === name);
				if (index !== -1) {
					health.checks[index] = result;
					// Recalculate overall status
					health.overall = calculateOverallStatus(health.checks);
				}
			}
		} catch (e) {
			console.error('Failed to recheck:', e);
		} finally {
			const newSet = new Set(recheckingChecks);
			newSet.delete(name);
			recheckingChecks = newSet;
		}
	}

	function calculateOverallStatus(checks: HealthCheck[]): HealthCheckStatus {
		let hasWarning = false;
		for (const check of checks) {
			if (check.status === 'unhealthy') return 'unhealthy';
			if (check.status === 'warning') hasWarning = true;
		}
		return hasWarning ? 'warning' : 'healthy';
	}

	function getStatusColor(status: HealthCheckStatus): string {
		switch (status) {
			case 'healthy': return 'bg-green-500';
			case 'warning': return 'bg-amber-500';
			case 'unhealthy': return 'bg-red-500';
		}
	}

	function getStatusBgColor(status: HealthCheckStatus): string {
		switch (status) {
			case 'healthy': return 'bg-green-600/20';
			case 'warning': return 'bg-amber-600/20';
			case 'unhealthy': return 'bg-red-600/20';
		}
	}

	function getStatusTextColor(status: HealthCheckStatus): string {
		switch (status) {
			case 'healthy': return 'text-green-400';
			case 'warning': return 'text-amber-400';
			case 'unhealthy': return 'text-red-400';
		}
	}

	function getStatusIcon(status: HealthCheckStatus): string {
		switch (status) {
			case 'healthy': return 'M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z';
			case 'warning': return 'M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z';
			case 'unhealthy': return 'M10 14l2-2m0 0l2-2m-2 2l-2-2m2 2l2 2m7-2a9 9 0 11-18 0 9 9 0 0118 0z';
		}
	}

	function getOverallLabel(status: HealthCheckStatus): string {
		switch (status) {
			case 'healthy': return 'All Systems Operational';
			case 'warning': return 'Some Services Degraded';
			case 'unhealthy': return 'System Issues Detected';
		}
	}

	function formatLatency(ms: number | undefined): string {
		if (ms === undefined) return '-';
		if (ms < 1000) return `${ms}ms`;
		return `${(ms / 1000).toFixed(1)}s`;
	}

	function formatTime(timestamp: string): string {
		const date = new Date(timestamp);
		return date.toLocaleTimeString();
	}

	function getCheckNameForApi(name: string): string {
		// Convert display name to API check name
		const lower = name.toLowerCase();
		if (lower === 'database') return 'database';
		if (lower === 'tmdb api') return 'tmdb';
		if (lower === 'prowlarr') return 'prowlarr';
		if (lower.startsWith('download client:')) {
			return 'download_client_' + name.substring(17).trim();
		}
		if (lower.startsWith('indexer:')) {
			return 'indexer_' + name.substring(9).trim();
		}
		if (lower.startsWith('disk:')) {
			return 'disk_' + name.substring(6).trim();
		}
		return lower.replace(/[^a-z0-9]/g, '_');
	}

	// Group checks by category
	const groupedChecks = $derived.by(() => {
		if (!health) return {};

		const groups: Record<string, HealthCheck[]> = {
			'Core Services': [],
			'Download Clients': [],
			'Indexers': [],
			'Storage': [],
			'External APIs': []
		};

		for (const check of health.checks) {
			const name = check.name.toLowerCase();
			if (name === 'database') {
				groups['Core Services'].push(check);
			} else if (name.startsWith('download client')) {
				groups['Download Clients'].push(check);
			} else if (name === 'prowlarr' || name.startsWith('indexer')) {
				groups['Indexers'].push(check);
			} else if (name.startsWith('disk')) {
				groups['Storage'].push(check);
			} else if (name.includes('tmdb') || name.includes('api')) {
				groups['External APIs'].push(check);
			} else {
				groups['Core Services'].push(check);
			}
		}

		// Remove empty groups
		return Object.fromEntries(
			Object.entries(groups).filter(([_, checks]) => checks.length > 0)
		);
	});

	// Count by status
	const statusCounts = $derived.by(() => {
		if (!health) return { healthy: 0, warning: 0, unhealthy: 0 };
		return health.checks.reduce((acc, check) => {
			acc[check.status]++;
			return acc;
		}, { healthy: 0, warning: 0, unhealthy: 0 } as Record<HealthCheckStatus, number>);
	});
</script>

<div class="space-y-6">
	{#if loading}
		<div class="flex items-center justify-center py-12">
			<svg class="w-8 h-8 animate-spin text-text-muted" fill="none" viewBox="0 0 24 24">
				<circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
				<path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
			</svg>
			<span class="ml-3 text-text-muted">Checking system health...</span>
		</div>
	{:else if error}
		<div class="bg-red-900/20 border border-red-600/30 text-red-400 px-4 py-3 rounded-xl">
			{error}
			<button class="ml-4 underline" onclick={loadHealth}>Retry</button>
		</div>
	{:else if health}
		<!-- Overall Status Banner -->
		<div class="glass-card p-6 {getStatusBgColor(health.overall)} border-l-4 {health.overall === 'healthy' ? 'border-green-500' : health.overall === 'warning' ? 'border-amber-500' : 'border-red-500'}">
			<div class="flex items-center justify-between">
				<div class="flex items-center gap-4">
					<div class="w-12 h-12 rounded-xl {getStatusBgColor(health.overall)} flex items-center justify-center">
						<svg class="w-7 h-7 {getStatusTextColor(health.overall)}" fill="none" stroke="currentColor" viewBox="0 0 24 24">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d={getStatusIcon(health.overall)} />
						</svg>
					</div>
					<div>
						<h2 class="text-xl font-bold {getStatusTextColor(health.overall)}">
							{getOverallLabel(health.overall)}
						</h2>
						<p class="text-sm text-text-secondary mt-1">
							Last checked: {formatTime(health.lastFullCheck)}
						</p>
					</div>
				</div>
				<div class="flex items-center gap-4">
					<div class="flex items-center gap-3 text-sm">
						{#if statusCounts.healthy > 0}
							<span class="flex items-center gap-1">
								<span class="w-2 h-2 rounded-full bg-green-500"></span>
								<span class="text-text-secondary">{statusCounts.healthy}</span>
							</span>
						{/if}
						{#if statusCounts.warning > 0}
							<span class="flex items-center gap-1">
								<span class="w-2 h-2 rounded-full bg-amber-500"></span>
								<span class="text-text-secondary">{statusCounts.warning}</span>
							</span>
						{/if}
						{#if statusCounts.unhealthy > 0}
							<span class="flex items-center gap-1">
								<span class="w-2 h-2 rounded-full bg-red-500"></span>
								<span class="text-text-secondary">{statusCounts.unhealthy}</span>
							</span>
						{/if}
					</div>
					<button
						class="liquid-btn flex items-center gap-2"
						onclick={loadHealth}
						disabled={loading}
					>
						<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
						</svg>
						Refresh All
					</button>
				</div>
			</div>
		</div>

		<!-- Health Check Groups -->
		{#each Object.entries(groupedChecks) as [groupName, checks]}
			<div class="space-y-3">
				<h3 class="text-sm font-semibold text-text-muted uppercase tracking-wider">{groupName}</h3>

				<div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
					{#each checks as check}
						{@const isRechecking = recheckingChecks.has(check.name)}
						<div class="glass-card p-4 flex flex-col">
							<div class="flex items-start justify-between mb-3">
								<div class="flex items-center gap-3">
									<div class="w-3 h-3 rounded-full {getStatusColor(check.status)}"></div>
									<span class="text-text-primary font-medium">{check.name}</span>
								</div>
								<button
									class="p-1.5 rounded-lg hover:bg-white/10 transition-colors text-text-muted hover:text-text-primary disabled:opacity-50"
									onclick={() => runSingleCheck(getCheckNameForApi(check.name))}
									disabled={isRechecking}
									title="Recheck"
								>
									<svg class="w-4 h-4 {isRechecking ? 'animate-spin' : ''}" fill="none" stroke="currentColor" viewBox="0 0 24 24">
										<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
									</svg>
								</button>
							</div>

							<div class="flex-1">
								<p class="text-sm {getStatusTextColor(check.status)} mb-1">{check.message}</p>
								{#if check.error}
									<p class="text-xs text-red-400 mt-1 line-clamp-2" title={check.error}>{check.error}</p>
								{/if}
							</div>

							<div class="flex items-center justify-between mt-3 pt-3 border-t border-white/5 text-xs text-text-muted">
								{#if check.latency !== undefined}
									<span>Latency: {formatLatency(check.latency)}</span>
								{:else}
									<span></span>
								{/if}
								<span>{formatTime(check.lastCheck)}</span>
							</div>
						</div>
					{/each}
				</div>
			</div>
		{/each}
	{/if}
</div>
