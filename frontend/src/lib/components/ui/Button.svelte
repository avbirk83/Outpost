<script lang="ts">
	import type { Snippet } from 'svelte';

	type Variant = 'primary' | 'secondary' | 'ghost' | 'danger';
	type Size = 'sm' | 'md' | 'lg';

	interface Props {
		variant?: Variant;
		size?: Size;
		disabled?: boolean;
		loading?: boolean;
		type?: 'button' | 'submit' | 'reset';
		href?: string;
		icon?: string;
		iconOnly?: boolean;
		fullWidth?: boolean;
		onclick?: (e: MouseEvent) => void;
		children?: Snippet;
	}

	let {
		variant = 'primary',
		size = 'md',
		disabled = false,
		loading = false,
		type = 'button',
		href = '',
		icon = '',
		iconOnly = false,
		fullWidth = false,
		onclick,
		children
	}: Props = $props();

	const baseClasses = 'inline-flex items-center justify-center gap-2 font-medium rounded-lg transition-all hover:scale-[1.02] active:scale-[0.98] disabled:opacity-50 disabled:cursor-not-allowed disabled:hover:scale-100 focus:outline-none focus:ring-2 focus:ring-cream/50 focus:ring-offset-2 focus:ring-offset-bg-primary';

	const variantClasses: Record<Variant, string> = {
		primary: 'bg-cream text-black hover:bg-cream/90',
		secondary: 'bg-white/5 text-text-primary hover:bg-white/10 border border-border-subtle',
		ghost: 'bg-transparent text-text-secondary hover:text-text-primary hover:bg-white/5',
		danger: 'bg-red-500/20 text-red-400 hover:bg-red-500/30'
	};

	const sizeClasses: Record<Size, string> = {
		sm: 'h-8 px-3 text-xs',
		md: 'h-10 px-4 text-sm',
		lg: 'h-12 px-6 text-base'
	};

	const iconSizeClasses: Record<Size, string> = {
		sm: 'w-3.5 h-3.5',
		md: 'w-4 h-4',
		lg: 'w-5 h-5'
	};

	const iconOnlyClasses: Record<Size, string> = {
		sm: 'w-8 h-8 p-0',
		md: 'w-10 h-10 p-0',
		lg: 'w-12 h-12 p-0'
	};

	$effect(() => {
		// Reactivity for classes
	});

	const classes = $derived(
		`${baseClasses} ${variantClasses[variant]} ${iconOnly ? iconOnlyClasses[size] : sizeClasses[size]} ${fullWidth ? 'w-full' : ''}`
	);
</script>

{#if href && !disabled}
	<a {href} class={classes}>
		{#if loading}
			<div class="animate-spin rounded-full border-2 border-current border-t-transparent {iconSizeClasses[size]}"></div>
		{:else if icon}
			<svg class={iconSizeClasses[size]} fill="none" stroke="currentColor" viewBox="0 0 24 24">
				<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d={icon} />
			</svg>
		{/if}
		{#if children && !iconOnly}
			{@render children()}
		{/if}
	</a>
{:else}
	<button
		{type}
		disabled={disabled || loading}
		{onclick}
		class={classes}
	>
		{#if loading}
			<div class="animate-spin rounded-full border-2 border-current border-t-transparent {iconSizeClasses[size]}"></div>
		{:else if icon}
			<svg class={iconSizeClasses[size]} fill="none" stroke="currentColor" viewBox="0 0 24 24">
				<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d={icon} />
			</svg>
		{/if}
		{#if children && !iconOnly}
			{@render children()}
		{/if}
	</button>
{/if}
