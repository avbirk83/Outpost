<script lang="ts">
	import type { Snippet } from 'svelte';

	interface Props {
		children: Snippet;
		class?: string;
		sticky?: boolean;
	}

	let { children, class: className = '', sticky = false }: Props = $props();
</script>

<div
	class="bg-glass backdrop-blur-xl border border-border-subtle rounded-lg {sticky ? 'sticky top-24' : ''} {className}"
	style="background: rgba(255, 255, 255, 0.06);"
>
	{@render children()}
</div>
