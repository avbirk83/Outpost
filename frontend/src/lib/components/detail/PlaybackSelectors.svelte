<script lang="ts">
	import Select from '$lib/components/ui/Select.svelte';

	interface SelectOption {
		value: number | string | null;
		label: string;
	}

	interface Props {
		videoOptions?: SelectOption[];
		audioOptions?: SelectOption[];
		subOptions?: SelectOption[];
		selectedVideo?: number | string | null;
		selectedAudio?: number | string | null;
		selectedSub?: number | string | null;
		onVideoChange?: (value: number | string | null) => void;
		onAudioChange?: (value: number | string | null) => void;
		onSubChange?: (value: number | string | null) => void;
		class?: string;
	}

	let {
		videoOptions = [],
		audioOptions = [],
		subOptions = [],
		selectedVideo = $bindable(),
		selectedAudio = $bindable(),
		selectedSub = $bindable(),
		onVideoChange,
		onAudioChange,
		onSubChange,
		class: className = ''
	}: Props = $props();

	// Convert options for Select component (handles null values)
	function toSelectOptions(opts: SelectOption[]) {
		return opts.map(o => ({
			value: o.value === null ? '' : o.value,
			label: o.label
		}));
	}

	// Handle value conversion for null
	function handleVideoSelect(val: string | number) {
		const converted = val === '' ? null : val;
		selectedVideo = converted;
		onVideoChange?.(converted);
	}

	function handleAudioSelect(val: string | number) {
		const converted = val === '' ? null : val;
		selectedAudio = converted;
		onAudioChange?.(converted);
	}

	function handleSubSelect(val: string | number) {
		const converted = val === '' ? null : val;
		selectedSub = converted;
		onSubChange?.(converted);
	}
</script>

<div class="flex items-center gap-2 {className}">
	<!-- Video selector -->
	{#if videoOptions.length > 0}
		<div class="flex items-center gap-2">
			<svg class="w-4 h-4 text-text-muted flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
				<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 10l4.553-2.276A1 1 0 0121 8.618v6.764a1 1 0 01-1.447.894L15 14M5 18h8a2 2 0 002-2V8a2 2 0 00-2-2H5a2 2 0 00-2 2v8a2 2 0 002 2z" />
			</svg>
			<Select
				value={selectedVideo ?? ''}
				options={toSelectOptions(videoOptions)}
				onchange={handleVideoSelect}
				class="min-w-[120px]"
			/>
		</div>
	{/if}

	<!-- Audio selector -->
	{#if audioOptions.length > 0}
		<div class="flex items-center gap-2">
			<svg class="w-4 h-4 text-text-muted flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
				<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15.536 8.464a5 5 0 010 7.072m2.828-9.9a9 9 0 010 12.728M5.586 15H4a1 1 0 01-1-1v-4a1 1 0 011-1h1.586l4.707-4.707C10.923 3.663 12 4.109 12 5v14c0 .891-1.077 1.337-1.707.707L5.586 15z" />
			</svg>
			<Select
				value={selectedAudio ?? ''}
				options={toSelectOptions(audioOptions)}
				onchange={handleAudioSelect}
				class="min-w-[140px]"
			/>
		</div>
	{/if}

	<!-- Subtitle selector -->
	{#if subOptions.length > 0}
		<div class="flex items-center gap-2">
			<svg class="w-4 h-4 text-text-muted flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
				<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M7 8h10M7 12h4m1 8l-4-4H5a2 2 0 01-2-2V6a2 2 0 012-2h14a2 2 0 012 2v8a2 2 0 01-2 2h-3l-4 4z" />
			</svg>
			<Select
				value={selectedSub ?? ''}
				options={toSelectOptions(subOptions)}
				onchange={handleSubSelect}
				class="min-w-[100px]"
			/>
		</div>
	{/if}
</div>
