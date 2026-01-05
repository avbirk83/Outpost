<script lang="ts">
	interface Option {
		value: string | number;
		label: string;
	}

	interface Props {
		options: Option[];
		value: string | number;
		onchange?: (value: string | number) => void;
		id?: string;
		disabled?: boolean;
		class?: string;
	}

	let { options, value = $bindable(), onchange, id, disabled = false, class: className = '' }: Props = $props();

	let open = $state(false);
	let buttonRef: HTMLButtonElement;

	const selectedOption = $derived(options.find(o => o.value === value));

	function select(opt: Option) {
		value = opt.value;
		onchange?.(opt.value);
		open = false;
	}

	function toggle() {
		if (!disabled) {
			open = !open;
		}
	}

	function close() {
		open = false;
	}

	function handleKeydown(e: KeyboardEvent) {
		if (e.key === 'Escape') {
			close();
		} else if (e.key === 'ArrowDown' && !open) {
			open = true;
			e.preventDefault();
		}
	}
</script>

<svelte:window on:keydown={handleKeydown} />

{#if open}
	<button
		type="button"
		class="fixed inset-0 z-[500] cursor-default"
		onclick={close}
		aria-label="Close dropdown"
	></button>
{/if}

<div class="relative {className}">
	<button
		bind:this={buttonRef}
		type="button"
		{id}
		onclick={toggle}
		{disabled}
		class="w-full flex items-center justify-between gap-2 px-3 py-1.5 text-sm text-left
			bg-transparent text-text-primary transition-all rounded-lg
			hover:bg-glass
			disabled:opacity-50 disabled:cursor-not-allowed"
	>
		<span class="truncate">{selectedOption?.label || 'Select...'}</span>
		<svg
			class="w-4 h-4 text-text-muted flex-shrink-0 transition-transform {open ? 'rotate-180' : ''}"
			fill="none"
			stroke="currentColor"
			viewBox="0 0 24 24"
		>
			<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7" />
		</svg>
	</button>

	{#if open}
		<div
			class="absolute left-0 right-0 top-full mt-1 max-h-64 overflow-y-auto scrollbar-thin z-[100]
				bg-bg-card rounded-xl border border-border-subtle shadow-2xl"
		>
			{#each options as opt}
				<button
					type="button"
					onclick={() => select(opt)}
					class="w-full text-left flex items-center gap-3 px-4 py-2.5 text-sm transition-colors
						{value === opt.value ? 'bg-cream/10 text-text-primary' : 'text-text-secondary hover:bg-cream/10 hover:text-text-primary'}"
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
