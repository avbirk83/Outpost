<script lang="ts">
	import { formatRuntime } from '$lib/utils';

	interface Props {
		id: number;
		episodeNumber: number;
		title: string;
		overview?: string;
		stillPath?: string | null;
		runtime?: number;
		airDate?: string;
		progress?: number;
		watched?: boolean;
	}

	let { id, episodeNumber, title, overview, stillPath, runtime, airDate, progress, watched }: Props = $props();

	function formatDate(dateStr: string): string {
		const date = new Date(dateStr);
		return date.toLocaleDateString('en-US', { month: 'short', day: 'numeric', year: 'numeric' });
	}
</script>

<a
	href={`/watch/episode/${id}`}
	class="group flex gap-4 p-3 rounded-xl bg-bg-card/50 hover:bg-bg-card border border-white/5 hover:border-white/10 transition-all duration-200"
>
	<!-- Episode thumbnail -->
	<div class="relative flex-shrink-0 w-40 aspect-video rounded-lg overflow-hidden bg-bg-elevated">
		{#if stillPath}
			<img
				src={`/api/images/still${stillPath}`}
				alt={title}
				class="w-full h-full object-cover transition-transform duration-300 group-hover:scale-105"
				loading="lazy"
			/>
		{:else}
			<div class="w-full h-full flex items-center justify-center">
				<svg class="w-8 h-8 text-text-muted" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M14.752 11.168l-3.197-2.132A1 1 0 0010 9.87v4.263a1 1 0 001.555.832l3.197-2.132a1 1 0 000-1.664z" />
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
				</svg>
			</div>
		{/if}

		<!-- Play overlay -->
		<div class="absolute inset-0 flex items-center justify-center bg-black/40 opacity-0 group-hover:opacity-100 transition-opacity duration-200">
			<div class="w-10 h-10 rounded-full bg-white/30 flex items-center justify-center">
				<svg class="w-5 h-5 text-white ml-0.5" fill="currentColor" viewBox="0 0 24 24">
					<path d="M8 5v14l11-7z" />
				</svg>
			</div>
		</div>

		<!-- Progress bar -->
		{#if progress !== undefined && progress > 0}
			<div class="absolute bottom-0 left-0 right-0 h-1 bg-black/50">
				<div
					class="h-full bg-outpost-500"
					style="width: {progress}%"
				></div>
			</div>
		{/if}

		<!-- Watched badge -->
		{#if watched}
			<div class="absolute top-1 right-1 w-5 h-5 rounded-full bg-green-500 flex items-center justify-center">
				<svg class="w-3 h-3 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="3" d="M5 13l4 4L19 7" />
				</svg>
			</div>
		{/if}
	</div>

	<!-- Episode info -->
	<div class="flex-1 min-w-0 py-1">
		<div class="flex items-center gap-2">
			<span class="text-sm font-medium text-outpost-400">E{episodeNumber}</span>
			<h4 class="text-base font-medium text-text-primary truncate group-hover:text-outpost-400 transition-colors">
				{title}
			</h4>
		</div>

		{#if overview}
			<p class="text-sm text-text-secondary line-clamp-2 mt-1.5">{overview}</p>
		{/if}

		<div class="flex items-center gap-3 mt-2 text-xs text-text-muted">
			{#if runtime}
				<span class="flex items-center gap-1">
					<svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
					</svg>
					{formatRuntime(runtime)}
				</span>
			{/if}
			{#if airDate}
				<span class="flex items-center gap-1">
					<svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 7V3m8 4V3m-9 8h10M5 21h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z" />
					</svg>
					{formatDate(airDate)}
				</span>
			{/if}
		</div>
	</div>
</a>
