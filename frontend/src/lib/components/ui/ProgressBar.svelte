<script lang="ts">
	interface Props {
		percent: number;
		variant?: 'default' | 'card' | 'episode';
		class?: string;
	}

	let { percent, variant = 'default', class: className = '' }: Props = $props();

	const heightClass = variant === 'card' ? 'h-[3px]' : variant === 'episode' ? 'h-[3px]' : 'h-1';
</script>

<div class="w-full {heightClass} bg-white/20 rounded-full overflow-hidden {className}">
	<div
		class="{heightClass} bg-amber-400 rounded-full transition-all duration-300"
		style="width: {Math.min(100, Math.max(0, percent))}%"
	></div>
</div>
