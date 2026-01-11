<script lang="ts">
	interface Option {
		value: string | number | null;
		label: string;
	}

	interface Props {
		options: Option[];
		value: string | number | null;
		onchange: (value: string | number | null) => void;
		placeholder?: string;
		icon?: 'video' | 'audio' | 'subtitles' | 'quality' | 'none';
		inline?: boolean;
	}

	let { options, value, onchange, placeholder = 'Select...', icon = 'none', inline = false }: Props = $props();

	let open = $state(false);
	let containerRef: HTMLDivElement;

	const selectedOption = $derived(options.find(o => o.value === value));

	function select(opt: Option) {
		onchange(opt.value);
		open = false;
	}

	function toggle(e: MouseEvent) {
		// Don't stop propagation - let other dropdowns close
		open = !open;
	}

	function handleWindowClick(e: MouseEvent) {
		// Close if click is outside this dropdown
		if (open && containerRef && !containerRef.contains(e.target as Node)) {
			open = false;
		}
	}

	function handleKeydown(e: KeyboardEvent) {
		if (e.key === 'Escape' && open) {
			open = false;
		}
	}
</script>

<svelte:window onclick={handleWindowClick} onkeydown={handleKeydown} />

<div bind:this={containerRef} class="relative {inline ? 'w-full' : ''}">
	<div class="rounded-lg overflow-hidden transition-all {open ? 'rounded-b-none' : ''}">
		<button
			type="button"
			onclick={toggle}
			class="w-full flex items-center gap-2 px-2.5 py-1.5 bg-bg-elevated hover:bg-bg-elevated/80 border border-border-subtle rounded-lg transition-all {open ? 'rounded-b-none border-b-transparent' : ''}"
		>
			{#if icon === 'video'}
				<svg class="w-3.5 h-3.5 text-text-muted flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 10l4.553-2.276A1 1 0 0121 8.618v6.764a1 1 0 01-1.447.894L15 14M5 18h8a2 2 0 002-2V8a2 2 0 00-2-2H5a2 2 0 00-2 2v8a2 2 0 002 2z" />
				</svg>
			{:else if icon === 'audio'}
				<svg class="w-3.5 h-3.5 text-text-muted flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15.536 8.464a5 5 0 010 7.072m2.828-9.9a9 9 0 010 12.728M5.586 15H4a1 1 0 01-1-1v-4a1 1 0 011-1h1.586l4.707-4.707C10.923 3.663 12 4.109 12 5v14c0 .891-1.077 1.337-1.707.707L5.586 15z" />
				</svg>
			{:else if icon === 'subtitles'}
				<svg class="w-3.5 h-3.5 text-text-muted flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M7 8h10M7 12h4m1 8l-4-4H5a2 2 0 01-2-2V6a2 2 0 012-2h14a2 2 0 012 2v8a2 2 0 01-2 2h-3l-4 4z" />
				</svg>
			{:else if icon === 'quality'}
				<svg class="w-3.5 h-3.5 text-text-muted flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9.75 17L9 20l-1 1h8l-1-1-.75-3M3 13h18M5 17h14a2 2 0 002-2V5a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z" />
				</svg>
			{/if}
			<span class="flex-1 text-left truncate text-xs text-text-primary">
				{selectedOption?.label || placeholder}
			</span>
			<svg
				class="w-3 h-3 text-text-muted flex-shrink-0 transition-transform duration-200 {open ? 'rotate-180' : ''}"
				fill="none"
				stroke="currentColor"
				viewBox="0 0 24 24"
			>
				<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7" />
			</svg>
		</button>
	</div>

	{#if open}
		<div
			class="absolute left-0 right-0 top-full z-[1000] max-h-64 overflow-y-auto scrollbar-thin
				bg-bg-elevated border border-border-subtle border-t-0 rounded-b-lg shadow-xl"
		>
			{#each options as opt}
				<button
					type="button"
					onclick={() => select(opt)}
					class="w-full text-left flex items-center gap-2 px-2.5 py-1.5 text-xs transition-colors whitespace-nowrap
						{value === opt.value
							? 'bg-cream/15 text-text-primary'
							: 'text-text-secondary hover:bg-bg-card hover:text-text-primary'}"
				>
					{#if value === opt.value}
						<svg class="w-3.5 h-3.5 text-cream flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
						</svg>
					{:else}
						<span class="w-3.5 flex-shrink-0"></span>
					{/if}
					<span class="truncate">{opt.label}</span>
				</button>
			{/each}
		</div>
	{/if}
</div>
