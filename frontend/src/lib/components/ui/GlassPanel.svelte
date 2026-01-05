<script lang="ts">
	import type { Snippet } from 'svelte';

	interface Props {
		variant?: 'light' | 'medium' | 'heavy' | 'card';
		sticky?: boolean;
		class?: string;
		children: Snippet;
	}

	let { variant = 'light', sticky = false, class: className = '', children }: Props = $props();

	const variantClasses = {
		light: 'bg-glass',
		medium: 'bg-glass-hover',
		heavy: 'bg-glass-focus',
		card: 'liquid-card',
	};
</script>

<div
	class="backdrop-blur-xl border border-border-subtle rounded-lg {variantClasses[variant]} {sticky ? 'sticky top-24' : ''} {className}"
>
	{@render children()}
</div>
