<script lang="ts">
	interface Props {
		title?: string;
		message: string;
		onRetry?: () => void;
		retryLabel?: string;
		compact?: boolean;
		inline?: boolean;
	}

	let {
		title = 'Something went wrong',
		message,
		onRetry,
		retryLabel = 'Try again',
		compact = false,
		inline = false
	}: Props = $props();
</script>

{#if inline}
	<!-- Inline error banner -->
	<div class="bg-red-500/10 border border-red-500/20 text-red-400 px-4 py-3 rounded-xl flex items-center justify-between gap-4">
		<div class="flex items-center gap-3">
			<svg class="w-5 h-5 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
				<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
			</svg>
			<span class="text-sm">{message}</span>
		</div>
		{#if onRetry}
			<button
				onclick={onRetry}
				class="px-3 py-1.5 rounded-lg bg-red-500/20 hover:bg-red-500/30 text-red-400 text-sm font-medium transition-colors flex items-center gap-1.5"
			>
				<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
				</svg>
				{retryLabel}
			</button>
		{/if}
	</div>
{:else}
	<!-- Full error card -->
	<div class="glass-card {compact ? 'p-8' : 'p-12'} text-center">
		<div class="w-14 h-14 rounded-full bg-red-500/10 flex items-center justify-center mx-auto mb-4">
			<svg class="w-7 h-7 text-red-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
				<path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
			</svg>
		</div>

		<h3 class="text-base font-medium text-text-primary mb-1">{title}</h3>
		<p class="text-sm text-text-secondary max-w-sm mx-auto">{message}</p>

		{#if onRetry}
			<button
				onclick={onRetry}
				class="mt-4 px-4 py-2 rounded-lg bg-white/5 hover:bg-white/10 text-text-primary text-sm font-medium transition-all hover:scale-[1.02] inline-flex items-center gap-2"
			>
				<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
				</svg>
				{retryLabel}
			</button>
		{/if}
	</div>
{/if}
