<script lang="ts">
	import type { Snippet } from 'svelte';

	interface Props {
		variant?: 'default' | 'success' | 'info' | 'warning' | 'error' | 'muted';
		size?: 'sm' | 'md';
		children: Snippet;
	}

	let { variant = 'default', size = 'sm', children }: Props = $props();

	const baseClass = size === 'sm' ? 'liquid-badge-sm' : 'liquid-badge';

	const variantClasses: Record<string, string> = {
		default: '',
		success: '!bg-green-500/20 !border-green-400/40 text-green-400',
		info: '!bg-blue-500/20 !border-blue-400/40 text-blue-400',
		warning: '!bg-amber-500/20 !border-amber-400/40 text-amber-400',
		error: '!bg-red-500/20 !border-red-400/40 text-red-400',
		muted: '!bg-white/5 text-text-muted'
	};
</script>

<span class="{baseClass} {variantClasses[variant]}">
	{@render children()}
</span>
