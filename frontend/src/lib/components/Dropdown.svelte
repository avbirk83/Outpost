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
		icon?: 'video' | 'audio' | 'subtitles' | 'none';
		inline?: boolean;
	}

	let { options, value, onchange, placeholder = 'Select...', icon = 'none', inline = false }: Props = $props();

	let open = $state(false);

	const selectedOption = $derived(options.find(o => o.value === value));

	function select(opt: Option) {
		onchange(opt.value);
		open = false;
	}

	function toggle() {
		open = !open;
	}

	function close() {
		open = false;
	}
</script>

<!-- Backdrop to close dropdown when clicking outside -->
{#if open}
	<button
		type="button"
		class="fixed inset-0 z-[100]"
		onclick={close}
		aria-label="Close dropdown"
	></button>
{/if}

<div class="relative z-[101] {inline ? 'w-full' : ''}">
	<button
		type="button"
		onclick={toggle}
		class="flex items-center gap-1 transition-all {inline ? 'w-full px-1.5 py-1 bg-transparent hover:bg-glass rounded' : 'px-3 py-1.5 w-[160px] liquid-glass hover:bg-cream/10 ' + (open ? 'rounded-t-xl !rounded-b-none' : 'rounded-xl')}"
		style={!inline && open ? 'border-bottom-color: transparent;' : ''}
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
		{/if}
		<span class="{inline ? 'text-[11px] flex-1 text-left truncate' : 'text-sm flex-1'} text-text-primary">{selectedOption?.label || placeholder}</span>
		<svg class="w-2.5 h-2.5 text-text-muted flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
			<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7" />
		</svg>
	</button>

	{#if open}
		<div class="absolute left-0 top-full py-1 max-h-64 overflow-y-auto scrollbar-thin {inline ? 'mt-1 min-w-[180px] rounded-xl bg-black/90 backdrop-blur-xl border border-white/10' : 'right-0 liquid-glass !rounded-t-none rounded-b-xl !border-t-0'}">
			{#each options as opt}
				<button
					type="button"
					onclick={() => select(opt)}
					class="w-full text-left flex items-center gap-2 px-4 py-2 text-sm transition-colors whitespace-nowrap {value === opt.value ? 'bg-glass text-text-primary' : 'text-text-secondary hover:bg-glass hover:text-text-primary'}"
				>
					{#if value === opt.value}
						<svg class="w-4 h-4 text-cream flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
						</svg>
					{:else}
						<span class="w-4 flex-shrink-0"></span>
					{/if}
					<span>{opt.label}</span>
				</button>
			{/each}
		</div>
	{/if}
</div>
