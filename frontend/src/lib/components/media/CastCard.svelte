<script lang="ts">
	import { getTmdbImageUrl } from '$lib/api';

	interface Props {
		name: string;
		role: string;
		photo?: string | null;
		onclick?: () => void;
		class?: string;
	}

	let { name, role, photo, onclick, class: className = '' }: Props = $props();
</script>

<button
	{onclick}
	class="flex-shrink-0 w-[85px] text-center p-2 rounded-xl cursor-pointer hover:bg-glass transition-all focus:outline-none focus-visible:ring-2 focus-visible:ring-border-focus {className}"
>
	<!-- Photo -->
	<div class="w-16 h-16 rounded-full bg-glass mx-auto mb-2 overflow-hidden">
		{#if photo}
			<img
				src={getTmdbImageUrl(photo, 'w185')}
				alt={name}
				class="w-full h-full object-cover"
				loading="lazy"
			/>
		{:else}
			<div class="w-full h-full flex items-center justify-center text-xl text-text-muted bg-gradient-to-br from-bg-card to-bg-elevated">
				{name.charAt(0)}
			</div>
		{/if}
	</div>

	<!-- Name -->
	<p class="text-[11px] font-medium text-text-primary truncate">{name}</p>
	<p class="text-[10px] text-text-muted truncate">{role}</p>
</button>
