<script lang="ts">
	import type { Snippet } from 'svelte';

	interface Props {
		open: boolean;
		onClose: () => void;
		title?: string;
		subtitle?: string;
		size?: 'sm' | 'md' | 'lg' | 'xl' | 'full';
		showClose?: boolean;
		header?: Snippet;
		footer?: Snippet;
		children: Snippet;
	}

	let {
		open = $bindable(false),
		onClose,
		title = '',
		subtitle = '',
		size = 'md',
		showClose = true,
		header,
		footer,
		children
	}: Props = $props();

	const sizeClasses: Record<string, string> = {
		sm: 'max-w-sm',
		md: 'max-w-md',
		lg: 'max-w-lg',
		xl: 'max-w-xl',
		full: 'max-w-4xl'
	};

	function handleBackdropClick(e: MouseEvent) {
		if (e.target === e.currentTarget) {
			onClose();
		}
	}

	function handleKeydown(e: KeyboardEvent) {
		if (e.key === 'Escape') {
			onClose();
		}
	}
</script>

<svelte:window onkeydown={handleKeydown} />

{#if open}
	<!-- Backdrop -->
	<div
		class="fixed inset-0 bg-black/80 backdrop-blur-sm z-50 flex items-center justify-center p-4"
		onclick={handleBackdropClick}
		role="dialog"
		aria-modal="true"
		aria-labelledby={title ? 'modal-title' : undefined}
	>
		<!-- Modal Card -->
		<div class="bg-bg-card border border-border-subtle rounded-2xl {sizeClasses[size]} w-full max-h-[90vh] flex flex-col animate-in fade-in zoom-in-95 duration-200">
			<!-- Header -->
			{#if header}
				<div class="p-6 border-b border-border-subtle flex-shrink-0">
					{@render header()}
				</div>
			{:else if title}
				<div class="p-6 border-b border-border-subtle flex items-start justify-between gap-4 flex-shrink-0">
					<div>
						<h2 id="modal-title" class="text-lg font-semibold text-text-primary">{title}</h2>
						{#if subtitle}
							<p class="text-sm text-text-secondary mt-0.5">{subtitle}</p>
						{/if}
					</div>
					{#if showClose}
						<button
							onclick={onClose}
							class="p-1.5 rounded-lg hover:bg-white/10 text-text-muted hover:text-text-primary transition-colors"
							aria-label="Close modal"
						>
							<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
								<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
							</svg>
						</button>
					{/if}
				</div>
			{/if}

			<!-- Body -->
			<div class="p-6 overflow-y-auto flex-1">
				{@render children()}
			</div>

			<!-- Footer -->
			{#if footer}
				<div class="p-6 border-t border-border-subtle flex justify-end gap-3 flex-shrink-0">
					{@render footer()}
				</div>
			{/if}
		</div>
	</div>
{/if}
