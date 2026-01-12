<script lang="ts">
	import type { Snippet } from 'svelte';

	interface Props {
		maxWidth?: 'sm' | 'md' | 'lg' | 'xl' | '2xl' | '4xl' | '7xl' | 'full';
		children: Snippet;
	}

	let { maxWidth = '7xl', children }: Props = $props();

	const maxWidthClasses: Record<string, string> = {
		sm: 'max-w-sm',
		md: 'max-w-md',
		lg: 'max-w-lg',
		xl: 'max-w-xl',
		'2xl': 'max-w-2xl',
		'4xl': 'max-w-4xl',
		'7xl': 'max-w-7xl',
		full: 'max-w-full'
	};
</script>

<div class="{maxWidthClasses[maxWidth]} mx-auto px-4 py-6 space-y-6">
	{@render children()}
</div>
