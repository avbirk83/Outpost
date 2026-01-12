<script lang="ts">
	import type { Snippet } from 'svelte';

	interface Props {
		label: string;
		id?: string;
		required?: boolean;
		error?: string;
		help?: string;
		children: Snippet;
	}

	let {
		label,
		id = '',
		required = false,
		error = '',
		help = '',
		children
	}: Props = $props();
</script>

<div class="space-y-1">
	<label for={id} class="block text-sm font-medium text-text-primary">
		{label}
		{#if required}
			<span class="text-red-400">*</span>
		{/if}
	</label>

	{@render children()}

	{#if error}
		<p class="text-xs text-red-400 mt-1">{error}</p>
	{:else if help}
		<p class="text-xs text-text-muted mt-1">{help}</p>
	{/if}
</div>
