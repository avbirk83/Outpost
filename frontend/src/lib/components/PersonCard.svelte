<script lang="ts">
	import { getTmdbImageUrl } from '$lib/api';

	interface Props {
		name: string;
		role: string;
		profilePath?: string | null;
		onclick?: () => void;
		class?: string;
	}

	let { name, role, profilePath, onclick, class: className = '' }: Props = $props();
</script>

<button
	{onclick}
	class="flex-shrink-0 w-36 text-center cursor-pointer group {className}"
>
	<div class="w-36 h-36 rounded-full bg-bg-elevated overflow-hidden mx-auto ring-2 ring-white/10 group-hover:ring-white/30 transition-all">
		{#if profilePath}
			<img
				src={getTmdbImageUrl(profilePath, 'w185')}
				alt={name}
				class="w-full h-full object-cover"
			/>
		{:else}
			<div class="w-full h-full flex items-center justify-center text-4xl text-text-muted bg-gradient-to-br from-bg-card to-bg-elevated">
				{name.charAt(0)}
			</div>
		{/if}
	</div>
	<p class="mt-2 text-sm font-medium text-text-primary truncate group-hover:text-white transition-colors">{name}</p>
	<p class="text-xs text-text-muted truncate">{role}</p>
</button>
