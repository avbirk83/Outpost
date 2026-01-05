<script lang="ts">
	import type { Snippet } from 'svelte';

	interface Props {
		onclick?: (e: MouseEvent) => void;
		href?: string;
		disabled?: boolean;
		title?: string;
		variant?: 'default' | 'green' | 'yellow' | 'pink';
		active?: boolean;
		children: Snippet;
	}

	let { onclick, href, disabled = false, title = '', variant = 'default', active = false, children }: Props = $props();

	const baseClasses = 'btn-icon-circle-lg disabled:opacity-50';

	let classes = $derived.by(() => {
		if (variant === 'default') {
			return active
				? `${baseClasses} bg-green-600 text-white hover:bg-green-500`
				: `${baseClasses} bg-cream/10 text-text-secondary hover:bg-cream/20 hover:text-cream`;
		}
		const variantStyles: Record<string, string> = {
			green: 'bg-green-600 text-white hover:bg-green-500',
			yellow: 'bg-amber-500 text-black cursor-default',
			pink: 'bg-pink-500/20 text-pink-400 hover:bg-pink-500/30',
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
