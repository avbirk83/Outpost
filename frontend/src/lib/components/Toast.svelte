<script lang="ts">
	import { toast, type Toast } from '$lib/stores/toast';

	let toasts: Toast[] = $state([]);

	$effect(() => {
		const unsubscribe = toast.subscribe(value => {
			toasts = value;
		});
		return unsubscribe;
	});
</script>

{#if toasts.length > 0}
	<div class="fixed bottom-6 right-6 z-[200] flex flex-col gap-2">
		{#each toasts as t (t.id)}
			<div
				class="flex items-center gap-3 px-4 py-3 rounded-xl shadow-lg backdrop-blur-xl animate-slide-in
					{t.type === 'success' ? 'bg-green-600/90 text-white' : ''}
					{t.type === 'error' ? 'bg-red-600/90 text-white' : ''}
					{t.type === 'info' ? 'bg-white/10 border border-white/20 text-white' : ''}"
			>
				{#if t.type === 'success'}
					<svg class="w-5 h-5 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
					</svg>
				{:else if t.type === 'error'}
					<svg class="w-5 h-5 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
					</svg>
				{:else}
					<svg class="w-5 h-5 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
					</svg>
				{/if}
				<span class="text-sm font-medium">{t.message}</span>
				<button
					onclick={() => toast.remove(t.id)}
					class="ml-2 p-1 rounded-full hover:bg-white/20 transition-colors"
					aria-label="Dismiss"
				>
					<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
					</svg>
				</button>
			</div>
		{/each}
	</div>
{/if}

<style>
	@keyframes slide-in {
		from {
			opacity: 0;
			transform: translateX(100%);
		}
		to {
			opacity: 1;
			transform: translateX(0);
		}
	}

	.animate-slide-in {
		animation: slide-in 0.2s ease-out;
	}
</style>
