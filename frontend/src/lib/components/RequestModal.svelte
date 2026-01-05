<script lang="ts">
	import { getQualityPresets, getTmdbImageUrl, type QualityPreset } from '$lib/api';
	import { onMount } from 'svelte';
	import TypeBadge from './TypeBadge.svelte';
	import Select from './ui/Select.svelte';

	interface MediaItem {
		title: string;
		year?: number;
		type: 'movie' | 'show';
		posterPath?: string | null;
		backdropPath?: string | null;
		overview?: string;
	}

	interface Props {
		item: MediaItem;
		mode: 'request' | 'approve';
		onConfirm: (qualityPresetId: number) => void;
		onCancel: () => void;
	}

	let { item, mode, onConfirm, onCancel }: Props = $props();

	let presets: QualityPreset[] = $state([]);
	let selectedPresetId: number = $state(0);
	let loading = $state(true);

	onMount(async () => {
		try {
			const allPresets = await getQualityPresets();
			// Only show enabled presets
			presets = allPresets.filter(p => p.enabled);
			// Select the default preset (if enabled), or first enabled one
			const defaultPreset = presets.find(p => p.isDefault);
			if (defaultPreset) {
				selectedPresetId = defaultPreset.id;
			} else if (presets.length > 0) {
				selectedPresetId = presets[0].id;
			}
		} catch (e) {
			console.error('Failed to load quality presets:', e);
		} finally {
			loading = false;
		}
	});

	function handleConfirm() {
		onConfirm(selectedPresetId);
	}

	function handleBackdropClick(e: MouseEvent) {
		if (e.target === e.currentTarget) {
			onCancel();
		}
	}

	function handleKeydown(e: KeyboardEvent) {
		if (e.key === 'Escape') {
			onCancel();
		}
	}

	function getPresetDescription(preset: QualityPreset): string {
		const parts: string[] = [];
		if (preset.resolution && preset.resolution !== 'any') parts.push(preset.resolution.toUpperCase());
		if (preset.source && preset.source !== 'any') parts.push(preset.source);
		if (preset.codec) parts.push(preset.codec.toUpperCase());
		if (preset.hdrFormats) {
			try {
				const formats = JSON.parse(preset.hdrFormats);
				if (formats.length > 0) parts.push('HDR');
			} catch {}
		}
		return parts.join(' · ') || 'Any quality';
	}

	const buttonText = mode === 'request' ? 'Request' : 'Approve & Search';
	const descriptionText = mode === 'request'
		? 'Your request will be sent for approval. Once approved, the item will be searched for on configured indexers.'
		: 'The item will be added to the wanted list and searched for on your indexers. If found, it will be sent to your download client automatically.';
</script>

<svelte:window onkeydown={handleKeydown} />

<div
	class="fixed inset-0 z-50 flex items-center justify-center bg-black/70 backdrop-blur-sm"
	onclick={handleBackdropClick}
	role="dialog"
	aria-modal="true"
	aria-labelledby="modal-title"
>
	<div class="bg-bg-base border border-white/10 rounded-2xl shadow-2xl max-w-md w-full mx-4 overflow-hidden">
		<!-- Header with backdrop -->
		<div class="relative h-32 bg-gradient-to-b from-white/10 to-transparent">
			{#if item.backdropPath}
				<img
					src={getTmdbImageUrl(item.backdropPath, 'w780')}
					alt=""
					class="absolute inset-0 w-full h-full object-cover opacity-30"
				/>
			{/if}
			<div class="absolute inset-0 bg-gradient-to-t from-bg-base via-bg-base/50 to-transparent" />

			<div class="absolute bottom-0 left-0 right-0 p-4 flex items-end gap-4">
				{#if item.posterPath}
					<img
						src={getTmdbImageUrl(item.posterPath, 'w154')}
						alt={item.title}
						class="w-16 h-24 object-cover rounded-lg shadow-lg flex-shrink-0 -mb-8"
					/>
				{:else}
					<div class="w-16 h-24 bg-bg-elevated rounded-lg flex items-center justify-center flex-shrink-0 -mb-8">
						<svg class="w-6 h-6 text-text-muted" fill="none" stroke="currentColor" viewBox="0 0 24 24">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z" />
						</svg>
					</div>
				{/if}

				<div class="flex-1 min-w-0">
					<h2 id="modal-title" class="text-lg font-semibold text-text-primary truncate">
						{item.title}
					</h2>
					<div class="flex items-center gap-2 mt-1">
						<TypeBadge type={item.type} />
						{#if item.year}
							<span class="text-sm text-text-secondary">{item.year}</span>
						{/if}
					</div>
				</div>
			</div>
		</div>

		<!-- Content -->
		<div class="p-4 pt-12 space-y-4">
			<div>
				<label for="quality-preset" class="block text-sm font-medium text-text-secondary mb-2">
					Quality Preset
				</label>
				{#if loading}
					<div class="h-11 bg-bg-elevated rounded-xl animate-pulse"></div>
				{:else if presets.length === 0}
					<p class="text-sm text-text-muted">No quality presets found. Default settings will be used.</p>
				{:else}
					<Select
						bind:value={selectedPresetId}
						options={presets.map(p => ({
							value: p.id,
							label: p.name + (p.isDefault ? ' (Default)' : ''),
						}))}
					/>
					{#if selectedPresetId}
						{@const selectedPreset = presets.find(p => p.id === selectedPresetId)}
						{#if selectedPreset}
							<p class="text-xs text-text-muted mt-1.5">
								{getPresetDescription(selectedPreset)}
							</p>
						{/if}
					{/if}
				{/if}
			</div>

			<div class="bg-white/5 rounded-xl p-3">
				<p class="text-sm text-text-secondary">
					{descriptionText}
				</p>
			</div>
		</div>

		<!-- Footer -->
		<div class="flex items-center justify-end gap-3 p-4 border-t border-white/10">
			<button
				onclick={onCancel}
				class="px-4 py-2 rounded-lg text-sm font-medium bg-white/10 text-text-secondary hover:bg-white/20 hover:text-white transition-colors"
			>
				Cancel
			</button>
			<button
				onclick={handleConfirm}
				disabled={loading || (presets.length > 0 && !selectedPresetId)}
				class="px-4 py-2 rounded-lg text-sm font-medium bg-green-600 text-white hover:bg-green-500 disabled:opacity-50 transition-colors flex items-center gap-2"
			>
				{#if mode === 'request'}
					<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" />
					</svg>
				{:else}
					<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
					</svg>
				{/if}
				{buttonText}
			</button>
		</div>
	</div>
</div>
