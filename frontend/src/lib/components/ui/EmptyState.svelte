<script lang="ts">
	import type { Snippet } from 'svelte';

	interface Props {
		icon?: string;
		title: string;
		description?: string;
		action?: () => void;
		actionLabel?: string;
		actionIcon?: string;
		compact?: boolean;
		children?: Snippet;
	}

	let {
		icon = 'M20 13V6a2 2 0 00-2-2H6a2 2 0 00-2 2v7m16 0v5a2 2 0 01-2 2H6a2 2 0 01-2-2v-5m16 0h-2.586a1 1 0 00-.707.293l-2.414 2.414a1 1 0 01-.707.293h-3.172a1 1 0 01-.707-.293l-2.414-2.414A1 1 0 006.586 13H4',
		title,
		description = '',
		action,
		actionLabel = '',
		actionIcon = '',
		compact = false,
		children
	}: Props = $props();
</script>

<div class="glass-card {compact ? 'p-8' : 'p-12'} text-center">
	<div class="w-14 h-14 rounded-full bg-bg-elevated flex items-center justify-center mx-auto mb-4">
		<svg class="w-7 h-7 text-text-muted" fill="none" stroke="currentColor" viewBox="0 0 24 24">
			<path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d={icon} />
		</svg>
	</div>

	<h3 class="text-base font-medium text-text-primary mb-1">{title}</h3>

	{#if description}
		<p class="text-sm text-text-secondary max-w-sm mx-auto">{description}</p>
	{/if}

	{#if children}
		<div class="mt-4">
			{@render children()}
		</div>
	{:else if action && actionLabel}
		<button
			onclick={action}
			class="mt-4 px-4 py-2 rounded-lg bg-cream text-black text-sm font-medium hover:bg-cream/90 transition-all hover:scale-[1.02] inline-flex items-center gap-2"
		>
			{#if actionIcon}
				<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d={actionIcon} />
				</svg>
			{/if}
			{actionLabel}
		</button>
	{/if}
</div>
