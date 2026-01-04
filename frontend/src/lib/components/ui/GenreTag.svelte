<script lang="ts">
	interface Props {
		genre: string;
		href?: string;
		onclick?: () => void;
		class?: string;
	}

	let { genre, href, onclick, class: className = '' }: Props = $props();
</script>

{#if href}
	<a
		{href}
		class="inline-flex items-center px-3.5 py-1.5 bg-glass backdrop-blur-xl border border-border-subtle rounded-full text-sm text-text-secondary hover:bg-glass-hover hover:text-text-primary transition-all {className}"
	>
		{genre}
	</a>
{:else}
	<button
		{onclick}
		class="inline-flex items-center px-3.5 py-1.5 bg-glass backdrop-blur-xl border border-border-subtle rounded-full text-sm text-text-secondary hover:bg-glass-hover hover:text-text-primary transition-all {className}"
	>
		{genre}
	</button>
{/if}
