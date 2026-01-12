<script lang="ts">
	import type { Snippet } from 'svelte';

	interface Props {
		padding?: 'none' | 'sm' | 'md' | 'lg';
		hover?: boolean;
		href?: string;
		onclick?: () => void;
		children: Snippet;
	}

	let {
		padding = 'md',
		hover = false,
		href = '',
		onclick,
		children
	}: Props = $props();

	const paddingClasses: Record<string, string> = {
		none: '',
		sm: 'p-4',
		md: 'p-6',
		lg: 'p-8'
	};

	const baseClasses = `bg-bg-card border border-border-subtle rounded-xl ${paddingClasses[padding]}`;
	const hoverClasses = hover ? 'transition-transform hover:-translate-y-0.5 hover:border-border-subtle/80 cursor-pointer' : '';
</script>

{#if href}
	<a {href} class="{baseClasses} {hoverClasses} block">
		{@render children()}
	</a>
{:else if onclick}
	<button {onclick} class="{baseClasses} {hoverClasses} w-full text-left">
		{@render children()}
	</button>
{:else}
	<div class="{baseClasses} {hoverClasses}">
		{@render children()}
	</div>
{/if}
