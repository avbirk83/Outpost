<script lang="ts">
	import type { Snippet } from 'svelte';

	interface Props {
		variant?: 'light' | 'medium' | 'heavy' | 'card';
		class?: string;
		children: Snippet;
	}

	let { variant = 'light', class: className = '', children }: Props = $props();

	const variantClasses = {
		light: 'glass',
		medium: 'glass-medium',
		heavy: 'glass-heavy',
		card: 'glass-card',
	};
</script>

<div class="{variantClasses[variant]} {className}">
	{@render children()}
</div>
