<script lang="ts">
	import type { Snippet } from 'svelte';

	interface Props {
		onclick?: (e: MouseEvent) => void;
		href?: string;
		disabled?: boolean;
		title?: string;
		variant?: 'default' | 'green' | 'yellow' | 'red' | 'pink';
		active?: boolean;
		children: Snippet;
	}

	let { onclick, href, disabled = false, title = '', variant = 'default', active = false, children }: Props = $props();

	const baseClasses = 'w-11 h-11 rounded-full flex items-center justify-center transition-all disabled:opacity-50';

	let classes = $derived.by(() => {
		if (variant === 'default') {
			return active
				? `${baseClasses} bg-green-600 border border-green-500 text-white hover:bg-green-500`
				: `${baseClasses} bg-white/10 border border-white/20 text-white hover:bg-white/20`;
		}
		const variantStyles: Record<string, string> = {
			green: 'bg-green-600 border border-green-500 text-white hover:bg-green-500',
			yellow: 'bg-yellow-600 border border-yellow-500 text-white cursor-default',
			red: 'bg-red-600 border border-red-500 text-white hover:bg-red-500',
			pink: 'bg-pink-500/20 border border-pink-400/40 text-pink-400 hover:bg-pink-500/30',
		};
		return `${baseClasses} ${variantStyles[variant] || variantStyles.default}`;
	});
</script>

{#if href}
	<a {href} class={classes} {title}>
		{@render children()}
	</a>
{:else}
	<button {onclick} {disabled} class={classes} {title}>
		{@render children()}
	</button>
{/if}
