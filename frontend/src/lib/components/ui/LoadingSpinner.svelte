<script lang="ts">
	type Size = 'sm' | 'md' | 'lg' | 'xl';

	interface Props {
		size?: Size;
		text?: string;
		fullPage?: boolean;
	}

	let { size = 'md', text = '', fullPage = false }: Props = $props();

	const sizeClasses: Record<Size, string> = {
		sm: 'w-4 h-4 border-2',
		md: 'w-6 h-6 border-2',
		lg: 'w-8 h-8 border-3',
		xl: 'w-12 h-12 border-4'
	};
</script>

{#if fullPage}
	<div class="flex items-center justify-center min-h-[400px]">
		<div class="flex flex-col items-center gap-3">
			<div class="animate-spin rounded-full border-cream/30 border-t-cream {sizeClasses[size]}"></div>
			{#if text}
				<p class="text-sm text-text-secondary">{text}</p>
			{/if}
		</div>
	</div>
{:else}
	<div class="flex items-center justify-center gap-3">
		<div class="animate-spin rounded-full border-cream/30 border-t-cream {sizeClasses[size]}"></div>
		{#if text}
			<p class="text-sm text-text-secondary">{text}</p>
		{/if}
	</div>
{/if}
