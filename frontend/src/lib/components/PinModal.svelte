<script lang="ts">
	import { auth } from '$lib/stores/auth';

	interface Props {
		open: boolean;
		onSuccess?: () => void;
		onCancel?: () => void;
		message?: string;
	}

	let { open = $bindable(), onSuccess, onCancel, message = 'Enter PIN to view this content' }: Props = $props();

	let pin = $state('');
	let error = $state('');
	let verifying = $state(false);
	let shake = $state(false);

	// Handle digit input
	function handleDigit(digit: string) {
		if (pin.length < 4) {
			pin += digit;
			error = '';
		}
	}

	// Handle backspace
	function handleBackspace() {
		if (pin.length > 0) {
			pin = pin.slice(0, -1);
			error = '';
		}
	}

	// Handle clear
	function handleClear() {
		pin = '';
		error = '';
	}

	// Verify PIN when 4 digits entered
	$effect(() => {
		if (pin.length === 4 && !verifying) {
			verifyPin();
		}
	});

	async function verifyPin() {
		verifying = true;
		error = '';

		const result = await auth.verifyPin(pin);

		if (result.success) {
			open = false;
			pin = '';
			onSuccess?.();
		} else {
			error = result.error || 'Incorrect PIN';
			shake = true;
			setTimeout(() => {
				shake = false;
				pin = '';
			}, 500);
		}

		verifying = false;
	}

	function handleCancel() {
		open = false;
		pin = '';
		error = '';
		onCancel?.();
	}

	function handleKeydown(e: KeyboardEvent) {
		if (!open) return;

		if (e.key >= '0' && e.key <= '9') {
			handleDigit(e.key);
		} else if (e.key === 'Backspace') {
			handleBackspace();
		} else if (e.key === 'Escape') {
			handleCancel();
		}
	}
</script>

<svelte:window onkeydown={handleKeydown} />

{#if open}
	<div
		class="pin-overlay"
		onclick={(e) => { if (e.target === e.currentTarget) handleCancel(); }}
		role="dialog"
		aria-modal="true"
		tabindex="-1"
	>
		<div class="pin-modal" class:shake>
			<h2 class="pin-title">{message}</h2>

			<!-- PIN Display -->
			<div class="pin-display">
				{#each [0, 1, 2, 3] as i}
					<div class="pin-dot" class:filled={pin.length > i}></div>
				{/each}
			</div>

			{#if error}
				<p class="pin-error">{error}</p>
			{/if}

			<!-- Number Pad -->
			<div class="pin-pad">
				{#each ['1', '2', '3', '4', '5', '6', '7', '8', '9'] as digit}
					<button
						type="button"
						class="pin-key"
						onclick={() => handleDigit(digit)}
						disabled={verifying}
					>
						{digit}
					</button>
				{/each}
				<button
					type="button"
					class="pin-key action"
					onclick={handleClear}
					disabled={verifying}
				>
					Clear
				</button>
				<button
					type="button"
					class="pin-key"
					onclick={() => handleDigit('0')}
					disabled={verifying}
				>
					0
				</button>
				<button
					type="button"
					class="pin-key action"
					onclick={handleBackspace}
					disabled={verifying}
				>
					<svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 14l2-2m0 0l2-2m-2 2l-2-2m2 2l2 2M3 12l6.414 6.414a2 2 0 001.414.586H19a2 2 0 002-2V7a2 2 0 00-2-2h-8.172a2 2 0 00-1.414.586L3 12z" />
					</svg>
				</button>
			</div>

			<button type="button" class="cancel-btn" onclick={handleCancel}>
				Cancel
			</button>
		</div>
	</div>
{/if}

<style>
	.pin-overlay {
		position: fixed;
		inset: 0;
		background: rgba(0, 0, 0, 0.9);
		z-index: 100;
		display: flex;
		align-items: center;
		justify-content: center;
		padding: 1rem;
	}

	.pin-modal {
		background: var(--bg-primary);
		border: 1px solid var(--border-subtle);
		border-radius: 1rem;
		padding: 2rem;
		max-width: 320px;
		width: 100%;
		text-align: center;
	}

	.pin-modal.shake {
		animation: shake 0.5s cubic-bezier(0.36, 0.07, 0.19, 0.97) both;
	}

	@keyframes shake {
		10%, 90% { transform: translateX(-1px); }
		20%, 80% { transform: translateX(2px); }
		30%, 50%, 70% { transform: translateX(-4px); }
		40%, 60% { transform: translateX(4px); }
	}

	.pin-title {
		font-size: 1.125rem;
		font-weight: 600;
		color: var(--text-primary);
		margin-bottom: 1.5rem;
	}

	.pin-display {
		display: flex;
		justify-content: center;
		gap: 1rem;
		margin-bottom: 1rem;
	}

	.pin-dot {
		width: 16px;
		height: 16px;
		border-radius: 50%;
		border: 2px solid var(--border-subtle);
		background: transparent;
		transition: all 0.15s ease;
	}

	.pin-dot.filled {
		background: var(--accent-primary);
		border-color: var(--accent-primary);
	}

	.pin-error {
		color: #ef4444;
		font-size: 0.875rem;
		margin-bottom: 1rem;
	}

	.pin-pad {
		display: grid;
		grid-template-columns: repeat(3, 1fr);
		gap: 0.75rem;
		margin-bottom: 1.5rem;
	}

	.pin-key {
		aspect-ratio: 1;
		font-size: 1.5rem;
		font-weight: 600;
		background: var(--bg-secondary);
		border: 1px solid var(--border-subtle);
		border-radius: 50%;
		color: var(--text-primary);
		cursor: pointer;
		transition: all 0.15s ease;
		display: flex;
		align-items: center;
		justify-content: center;
	}

	.pin-key:hover:not(:disabled) {
		background: var(--bg-card);
		border-color: var(--accent-primary);
	}

	.pin-key:active:not(:disabled) {
		transform: scale(0.95);
	}

	.pin-key:disabled {
		opacity: 0.5;
		cursor: not-allowed;
	}

	.pin-key.action {
		font-size: 0.875rem;
		font-weight: 500;
	}

	.pin-key svg {
		width: 1.5rem;
		height: 1.5rem;
	}

	.cancel-btn {
		width: 100%;
		padding: 0.75rem 1rem;
		background: transparent;
		border: 1px solid var(--border-subtle);
		border-radius: 0.5rem;
		color: var(--text-secondary);
		font-weight: 500;
		cursor: pointer;
		transition: all 0.15s ease;
	}

	.cancel-btn:hover {
		background: var(--bg-secondary);
	}
</style>
