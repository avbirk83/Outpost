<script lang="ts">
	import { getTmdbImageUrl } from '$lib/api';

	// Unified state - simplified user-facing flow
	export type QueueState =
		| 'pending'      // User requested, waiting approval
		| 'searching'    // Looking for download (covers approved, wanted, searching)
		| 'downloading'  // Actively downloading (covers queued, downloading)
		| 'paused'       // Download paused
		| 'stalled'      // Download stalled
		| 'importing'    // Moving/processing files
		| 'failed'       // Something went wrong
		| 'denied';      // Request denied

	interface Props {
		id: string;
		tmdbId: number;
		title: string;
		year?: number;
		type: 'movie' | 'show';
		posterPath?: string | null;
		state: QueueState;
		seasons?: string; // JSON array of season numbers (for TV shows)

		// Progress info (for download/import)
		progress?: number;
		downloaded?: number;
		size?: number;
		speed?: number;
		eta?: number;

		// Additional info
		quality?: string;
		indexer?: string;
		username?: string;
		timestamp?: string;
		error?: string;

		// Callbacks
		onApprove?: () => void;
		onDeny?: () => void;
		onSearch?: () => void;
		onCancel?: () => void;
		onRemove?: () => void;

		// State
		processing?: boolean;
		searching?: boolean;
		isAdmin?: boolean;
		confirmingCancel?: boolean;
	}

	let {
		id,
		tmdbId,
		title,
		year,
		type,
		posterPath,
		state,
		seasons,
		progress = 0,
		downloaded = 0,
		size = 0,
		speed = 0,
		eta = 0,
		quality,
		indexer,
		username,
		timestamp,
		error,
		onApprove,
		onDeny,
		onSearch,
		onCancel,
		onRemove,
		processing = false,
		searching = false,
		isAdmin = false,
		confirmingCancel = false
	}: Props = $props();

	// Format seasons for display
	function formatSeasons(seasonsJson?: string): string | null {
		if (!seasonsJson) return null;
		try {
			const arr = JSON.parse(seasonsJson) as number[];
			if (arr.length === 0) return null;
			if (arr.length === 1) return `S${arr[0]}`;
			return `S${arr.join(', S')}`;
		} catch {
			return null;
		}
	}

	const formattedSeasons = $derived(formatSeasons(seasons));

	// Badge colors and text based on state
	const stateConfig: Record<QueueState, { color: string; bg: string; text: string }> = {
		pending: { color: 'text-amber-400', bg: 'bg-amber-500/20', text: 'Pending' },
		searching: { color: 'text-purple-400', bg: 'bg-purple-500/20', text: 'Searching...' },
		downloading: { color: 'text-blue-400', bg: 'bg-blue-500/20', text: 'Downloading' },
		paused: { color: 'text-yellow-400', bg: 'bg-yellow-500/20', text: 'Paused' },
		stalled: { color: 'text-orange-400', bg: 'bg-orange-500/20', text: 'Stalled' },
		importing: { color: 'text-cyan-400', bg: 'bg-cyan-500/20', text: 'Importing' },
		failed: { color: 'text-red-400', bg: 'bg-red-500/20', text: 'Failed' },
		denied: { color: 'text-red-400', bg: 'bg-red-500/20', text: 'Denied' }
	};

	const config = $derived(stateConfig[state] || stateConfig.pending);
	const showProgress = $derived(['downloading', 'importing'].includes(state));
	const showActions = $derived(!['denied'].includes(state));
	const isFailed = $derived(state === 'failed');

	// Debug: log component state
	$effect(() => {
		console.log('QueueCard rendered:', { id, title, state, hasOnCancel: !!onCancel, hasOnRemove: !!onRemove });
	});
	// Only show poster overlay for active states
	const showPosterOverlay = $derived(['searching', 'downloading', 'importing', 'failed'].includes(state));

	function formatBytes(bytes: number): string {
		if (bytes === 0) return '0 B';
		const k = 1024;
		const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
		const i = Math.floor(Math.log(bytes) / Math.log(k));
		return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i];
	}

	function formatSpeed(bytes: number): string {
		return formatBytes(bytes) + '/s';
	}

	function formatEta(seconds: number): string {
		if (!seconds || seconds <= 0) return '';
		if (seconds < 60) return `${seconds}s`;
		if (seconds < 3600) return `${Math.floor(seconds / 60)}m`;
		return `${Math.floor(seconds / 3600)}h ${Math.floor((seconds % 3600) / 60)}m`;
	}
</script>

<div class="bg-bg-card border border-border-subtle rounded-xl p-4 {isFailed ? 'border-l-2 border-l-red-500' : ''}">
	<div class="flex items-start gap-3">
		<!-- Poster -->
		{#if posterPath}
			<div class="w-12 h-18 flex-shrink-0 rounded-lg overflow-hidden bg-bg-elevated relative">
				<img
					src={getTmdbImageUrl(posterPath, 'w92')}
					alt={title}
					class="w-full h-full object-cover"
				/>
				<!-- State icon overlay - only for active states -->
				{#if showPosterOverlay}
					<div class="absolute inset-0 flex items-center justify-center bg-black/50">
						{#if state === 'searching' || state === 'importing'}
							<div class="spinner-sm text-white"></div>
						{:else if state === 'downloading'}
							<svg class="w-5 h-5 text-blue-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
								<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4" />
							</svg>
						{:else if state === 'imported'}
							<svg class="w-5 h-5 text-green-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
								<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
							</svg>
						{:else if state === 'failed' || state === 'denied'}
							<svg class="w-5 h-5 text-red-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
								<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
							</svg>
						{/if}
					</div>
				{/if}
			</div>
		{:else}
			<div class="w-12 h-18 flex-shrink-0 rounded-lg {config.bg} flex items-center justify-center">
				{#if state === 'searching' || state === 'importing'}
					<div class="spinner-sm {config.color}"></div>
				{:else}
					<svg class="w-5 h-5 {config.color}" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z" />
					</svg>
				{/if}
			</div>
		{/if}

		<!-- Content -->
		<div class="flex-1 min-w-0">
			<div class="flex items-start justify-between gap-2">
				<div class="min-w-0">
					<h3 class="font-medium text-text-primary truncate">
						{title}{#if year}<span class="text-text-muted"> ({year})</span>{/if}
					</h3>
					<div class="flex items-center gap-2 mt-1 text-xs text-text-muted flex-wrap">
						<!-- State badge -->
						<span class="{config.color} font-medium">{config.text}</span>
						<span>·</span>
						<span class="capitalize">{type}</span>
						{#if formattedSeasons}
							<span>·</span>
							<span>{formattedSeasons}</span>
						{/if}
						{#if quality}
							<span>·</span>
							<span>{quality}</span>
						{/if}
						{#if indexer}
							<span>·</span>
							<span>{indexer}</span>
						{/if}
						{#if size > 0 && !showProgress}
							<span>·</span>
							<span>{formatBytes(size)}</span>
						{/if}
						{#if username}
							<span>·</span>
							<span>by {username}</span>
						{/if}
						{#if timestamp}
							<span>·</span>
							<span>{timestamp}</span>
						{/if}
					</div>
				</div>

				<!-- Actions -->
				{#if showActions}
					<div class="flex items-center gap-1 flex-shrink-0">
						{#if state === 'pending' && isAdmin}
							<button
								class="px-2 py-1 text-xs rounded-lg bg-green-500/20 text-green-400 hover:bg-green-500/30 transition-colors disabled:opacity-50"
								onclick={onApprove}
								disabled={processing}
							>
								{processing ? '...' : 'Approve'}
							</button>
							<button
								class="px-2 py-1 text-xs rounded-lg bg-red-500/20 text-red-400 hover:bg-red-500/30 transition-colors disabled:opacity-50"
								onclick={onDeny}
								disabled={processing}
							>
								Deny
							</button>
						{:else if state === 'pending'}
							<button
								class="px-2 py-1 text-xs rounded-lg bg-white/5 text-text-secondary hover:text-text-primary hover:bg-white/10 transition-colors disabled:opacity-50"
								onclick={() => { console.log('QueueCard cancel (pending) clicked, onRemove:', typeof onRemove); if (onRemove) onRemove(); }}
								disabled={processing}
							>
								Cancel
							</button>
						{:else if state === 'searching'}
							<button
								class="px-2 py-1 text-xs rounded-lg bg-white/5 text-text-secondary hover:text-text-primary hover:bg-white/10 transition-colors disabled:opacity-50"
								onclick={() => { console.log('QueueCard cancel (searching) clicked, onRemove:', typeof onRemove); if (onRemove) onRemove(); }}
								disabled={processing}
							>
								Cancel
							</button>
						{:else if ['downloading', 'paused', 'stalled'].includes(state)}
							<button
								class="px-2 py-1 text-xs rounded-lg transition-colors disabled:opacity-50 {confirmingCancel ? 'bg-red-500 text-white' : 'bg-red-500/20 text-red-400 hover:bg-red-500/30'}"
								onclick={() => { console.log('QueueCard cancel button clicked, state:', state, 'onCancel:', typeof onCancel); if (onCancel) onCancel(); }}
								disabled={processing}
							>
								{processing ? '...' : confirmingCancel ? 'Confirm?' : 'Cancel'}
							</button>
						{:else if state === 'importing'}
							<!-- No actions during import -->
						{:else if state === 'failed'}
							<button
								class="px-2 py-1 text-xs rounded-lg bg-purple-500/20 text-purple-400 hover:bg-purple-500/30 transition-colors disabled:opacity-50"
								onclick={onSearch}
								disabled={searching}
							>
								Retry
							</button>
							<button
								class="px-2 py-1 text-xs rounded-lg bg-white/5 text-text-secondary hover:text-text-primary hover:bg-white/10 transition-colors disabled:opacity-50"
								onclick={onRemove}
								disabled={processing}
							>
								Remove
							</button>
						{/if}
					</div>
				{/if}
			</div>

			<!-- Progress bar for downloading/importing -->
			{#if showProgress}
				<div class="mt-3">
					<div class="w-full bg-bg-elevated rounded-full h-1.5 overflow-hidden">
						<div
							class="h-full transition-all duration-300 {state === 'importing' ? 'bg-yellow-500' : 'bg-blue-500'}"
							style="width: {progress}%"
						></div>
					</div>
					<div class="flex justify-between mt-1 text-xs text-text-muted">
						<span>
							{progress.toFixed(1)}%
							{#if size > 0}
								 · {formatBytes(downloaded)} / {formatBytes(size)}
							{/if}
						</span>
						{#if state === 'downloading' && speed > 0}
							<span>
								{formatSpeed(speed)}
								{#if eta > 0} · {formatEta(eta)}{/if}
							</span>
						{:else if state === 'importing'}
							<span>Moving files...</span>
						{/if}
					</div>
				</div>
			{/if}

			<!-- Error message -->
			{#if error}
				<p class="text-xs text-red-400 mt-2">{error}</p>
			{/if}
		</div>
	</div>
</div>
