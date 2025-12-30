<script lang="ts">
	import type { Snippet } from 'svelte';

	interface Props {
		label: string;
		value: string | number;
		sublabel?: string;
		trend?: 'up' | 'down' | 'neutral';
		trendValue?: string;
		icon?: Snippet;
		color?: 'default' | 'outpost' | 'ember' | 'green' | 'blue';
	}

	let { label, value, sublabel, trend, trendValue, icon, color = 'default' }: Props = $props();

	const colorClasses = {
		default: 'bg-bg-card border-white/5',
		outpost: 'bg-outpost-950/50 border-outpost-800/30',
		ember: 'bg-white-950/50 border-white-800/30',
		green: 'bg-green-950/50 border-green-800/30',
		blue: 'bg-blue-950/50 border-blue-800/30',
	};

	const iconColors = {
		default: 'text-text-secondary bg-bg-elevated',
		outpost: 'text-outpost-400 bg-outpost-900/50',
		ember: 'text-white-400 bg-white-900/50',
		green: 'text-green-400 bg-green-900/50',
		blue: 'text-blue-400 bg-blue-900/50',
	};
</script>

<div class="glass-card p-5 {colorClasses[color]}">
	<div class="flex items-start justify-between">
		<div class="flex-1">
			<p class="text-sm text-text-secondary">{label}</p>
			<p class="text-3xl font-bold text-text-primary mt-1">{value}</p>

			{#if sublabel || (trend && trendValue)}
				<div class="flex items-center gap-2 mt-2">
					{#if trend && trendValue}
						<span class="flex items-center gap-0.5 text-sm {trend === 'up' ? 'text-green-400' : trend === 'down' ? 'text-red-400' : 'text-text-secondary'}">
							{#if trend === 'up'}
								<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 10l7-7m0 0l7 7m-7-7v18" />
								</svg>
							{:else if trend === 'down'}
								<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 14l-7 7m0 0l-7-7m7 7V3" />
								</svg>
							{:else}
								<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 12h14" />
								</svg>
							{/if}
							{trendValue}
						</span>
					{/if}
					{#if sublabel}
						<span class="text-sm text-text-muted">{sublabel}</span>
					{/if}
				</div>
			{/if}
		</div>

		{#if icon}
			<div class="w-12 h-12 rounded-xl flex items-center justify-center {iconColors[color]}">
				{@render icon()}
			</div>
		{/if}
	</div>
</div>
