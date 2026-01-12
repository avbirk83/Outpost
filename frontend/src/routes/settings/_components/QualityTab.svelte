<script lang="ts">
	import { onMount } from 'svelte';
	import Select from '$lib/components/ui/Select.svelte';
	import FormatFilteringSettings from './FormatFilteringSettings.svelte';
	import { toast } from '$lib/stores/toast';
	import {
		getQualityPresets,
		createQualityPreset,
		updateQualityPreset,
		deleteQualityPreset,
		setDefaultQualityPreset,
		toggleQualityPresetEnabled,
		updateQualityPresetPriority,
		type QualityPreset
	} from '$lib/api';

	let qualityPresets: QualityPreset[] = $state([]);
	let showAddPresetForm = $state(false);
	let editingPreset: QualityPreset | null = $state(null);
	let presetMediaTab: "movie" | "tv" | "anime" = $state("movie");
	const filteredPresets = $derived(qualityPresets.filter(p => (p.mediaType || 'movie') === presetMediaTab));
	let draggingPresetId: number | null = $state(null);
	let dragOverPresetId: number | null = $state(null);

	// Form state
	let presetName = $state('');
	let presetResolution = $state('1080p');
	let presetSource = $state('web');
	let presetHdrFormats: string[] = $state([]);
	let presetCodec = $state('any');
	let presetAudioFormats: string[] = $state([]);
	let presetPreferredEdition = $state('any');
	let presetMinSeeders = $state(3);
	let presetPreferSeasonPacks = $state(true);
	let presetAutoUpgrade = $state(true);

	// Options
	const resolutionOptions = ['4k', '1080p', '720p', '480p'];
	const sourceOptions = ['remux', 'bluray', 'web', 'any'];
	const hdrOptions = ['dv', 'hdr10+', 'hdr10', 'hlg'];
	const audioOptions = ['atmos', 'truehd', 'dtshd', 'dtsx', 'dd+'];
	const codecOptions = ['any', 'hevc', 'av1'];
	const editionOptions = ['any', 'theatrical', 'directors', 'extended', 'unrated'];

	const inputClass = "liquid-input w-full px-4 py-2.5";
	const labelClass = "block text-sm text-text-secondary mb-1.5";

	onMount(async () => {
		await loadQualityPresets();
	});

	async function loadQualityPresets() {
		try {
			qualityPresets = (await getQualityPresets()) || [];
		} catch (e) {
			console.error('Failed to load quality presets:', e);
			qualityPresets = [];
		}
	}

	function resetPresetForm() {
		presetName = '';
		presetResolution = '1080p';
		presetSource = 'web';
		presetHdrFormats = [];
		presetCodec = 'any';
		presetAudioFormats = [];
		presetPreferredEdition = 'any';
		presetMinSeeders = 3;
		presetPreferSeasonPacks = true;
		presetAutoUpgrade = true;
		editingPreset = null;
	}

	async function handleAddPreset() {
		try {
			await createQualityPreset({
				name: presetName,
				resolution: presetResolution,
				source: presetSource,
				hdrFormats: presetHdrFormats,
				codec: presetCodec,
				audioFormats: presetAudioFormats,
				preferredEdition: presetPreferredEdition,
				minSeeders: presetMinSeeders,
				preferSeasonPacks: presetPreferSeasonPacks,
				autoUpgrade: presetAutoUpgrade
			});
			resetPresetForm();
			showAddPresetForm = false;
			await loadQualityPresets();
			toast.success('Quality preset added');
		} catch (e) {
			toast.error('Failed to add preset');
		}
	}

	async function handleUpdatePreset() {
		if (!editingPreset) return;
		try {
			await updateQualityPreset(editingPreset.id, {
				name: presetName,
				resolution: presetResolution,
				source: presetSource,
				hdrFormats: presetHdrFormats,
				codec: presetCodec,
				audioFormats: presetAudioFormats,
				preferredEdition: presetPreferredEdition,
				minSeeders: presetMinSeeders,
				preferSeasonPacks: presetPreferSeasonPacks,
				autoUpgrade: presetAutoUpgrade
			});
			resetPresetForm();
			await loadQualityPresets();
			toast.success('Quality preset updated');
		} catch (e) {
			toast.error('Failed to update preset');
		}
	}

	function startEditPreset(preset: QualityPreset) {
		editingPreset = preset;
		presetName = preset.name;
		presetResolution = preset.resolution;
		presetSource = preset.source;
		presetHdrFormats = preset.hdrFormats || [];
		presetCodec = preset.codec || 'any';
		presetAudioFormats = preset.audioFormats || [];
		presetPreferredEdition = preset.preferredEdition || 'any';
		presetMinSeeders = preset.minSeeders;
		presetPreferSeasonPacks = preset.preferSeasonPacks;
		presetAutoUpgrade = preset.autoUpgrade;
		showAddPresetForm = false;
	}

	async function handleDeletePreset(id: number) {
		if (!confirm('Are you sure you want to delete this quality preset?')) return;
		try {
			await deleteQualityPreset(id);
			await loadQualityPresets();
			toast.success('Quality preset deleted');
		} catch (e) {
			toast.error('Failed to delete preset');
		}
	}

	async function handleSetDefaultPreset(id: number) {
		try {
			await setDefaultQualityPreset(id);
			await loadQualityPresets();
			toast.success('Default preset updated');
		} catch (e) {
			toast.error('Failed to set default preset');
		}
	}

	async function updateAnimePresetPreference(id: number, field: string, value: boolean | string) {
		const response = await fetch(`/api/quality/presets/${id}/anime-preferences`, {
			method: 'PATCH',
			headers: { 'Content-Type': 'application/json' },
			body: JSON.stringify({ [field]: value })
		});
		if (!response.ok) {
			throw new Error('Failed to update anime preferences');
		}
	}

	// Drag and drop handlers
	function handlePresetDragStart(e: DragEvent, presetId: number) {
		draggingPresetId = presetId;
		if (e.dataTransfer) {
			e.dataTransfer.effectAllowed = 'move';
			e.dataTransfer.setData('text/plain', String(presetId));
		}
	}

	function handlePresetDragOver(e: DragEvent, presetId: number) {
		e.preventDefault();
		if (e.dataTransfer) {
			e.dataTransfer.dropEffect = 'move';
		}
		if (draggingPresetId !== presetId) {
			dragOverPresetId = presetId;
		}
	}

	function handlePresetDragLeave() {
		dragOverPresetId = null;
	}

	function handlePresetDragEnd() {
		draggingPresetId = null;
		dragOverPresetId = null;
	}

	async function handlePresetDrop(e: DragEvent, targetPresetId: number) {
		e.preventDefault();
		if (draggingPresetId === null || draggingPresetId === targetPresetId) {
			handlePresetDragEnd();
			return;
		}

		const draggedIndex = qualityPresets.findIndex(p => p.id === draggingPresetId);
		const targetIndex = qualityPresets.findIndex(p => p.id === targetPresetId);

		if (draggedIndex === -1 || targetIndex === -1) {
			handlePresetDragEnd();
			return;
		}

		const newPresets = [...qualityPresets];
		const [removed] = newPresets.splice(draggedIndex, 1);
		newPresets.splice(targetIndex, 0, removed);

		try {
			for (let i = 0; i < newPresets.length; i++) {
				if (newPresets[i].priority !== i + 1) {
					await updateQualityPresetPriority(newPresets[i].id, i + 1);
				}
			}
			await loadQualityPresets();
		} catch (e) {
			console.error('Failed to reorder presets:', e);
		}

		handlePresetDragEnd();
	}

	function toggleHdrFormat(format: string) {
		if (presetHdrFormats.includes(format)) {
			presetHdrFormats = presetHdrFormats.filter(f => f !== format);
		} else {
			presetHdrFormats = [...presetHdrFormats, format];
		}
	}

	function toggleAudioFormat(format: string) {
		if (presetAudioFormats.includes(format)) {
			presetAudioFormats = presetAudioFormats.filter(f => f !== format);
		} else {
			presetAudioFormats = [...presetAudioFormats, format];
		}
	}

	function getPresetDescription(preset: QualityPreset): string {
		const parts: string[] = [];
		parts.push(preset.resolution.toUpperCase());
		if (preset.source !== 'any') parts.push(preset.source.toUpperCase());
		if (preset.hdrFormats && preset.hdrFormats.length > 0) {
			parts.push(preset.hdrFormats.map(h => h.toUpperCase()).join('/'));
		}
		return parts.join(' · ');
	}

	function getResolutionColor(resolution: string): string {
		switch (resolution) {
			case '4k': return 'bg-purple-600';
			case '1080p': return 'bg-blue-600';
			case '720p': return 'bg-green-600';
			default: return 'bg-bg-elevated';
		}
	}
</script>

<!-- Quality Presets -->
<section class="glass-card p-6 space-y-4">
	<div class="flex items-center justify-between">
		<div class="flex items-center gap-3">
			<div class="w-10 h-10 rounded-xl bg-amber-600/20 flex items-center justify-center">
				<svg class="w-5 h-5 text-amber-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4M7.835 4.697a3.42 3.42 0 001.946-.806 3.42 3.42 0 014.438 0 3.42 3.42 0 001.946.806 3.42 3.42 0 013.138 3.138 3.42 3.42 0 00.806 1.946 3.42 3.42 0 010 4.438 3.42 3.42 0 00-.806 1.946 3.42 3.42 0 01-3.138 3.138 3.42 3.42 0 00-1.946.806 3.42 3.42 0 01-4.438 0 3.42 3.42 0 00-1.946-.806 3.42 3.42 0 01-3.138-3.138 3.42 3.42 0 00-.806-1.946 3.42 3.42 0 010-4.438 3.42 3.42 0 00.806-1.946 3.42 3.42 0 013.138-3.138z" />
				</svg>
			</div>
			<div>
				<h2 class="text-lg font-semibold text-text-primary">Quality Presets</h2>
				<p class="text-sm text-text-secondary">Define target quality for automatic downloads</p>
			</div>
		</div>
		<button
			class="liquid-btn-sm"
			onclick={() => { showAddPresetForm = !showAddPresetForm; resetPresetForm(); }}
		>
			{showAddPresetForm ? 'Cancel' : 'Add Preset'}
		</button>
	</div>

	<!-- Media Type Tabs -->
	<div class="flex gap-1 p-1 bg-bg-card rounded-xl">
		<button
			type="button"
			class="flex-1 px-4 py-2 text-sm font-medium rounded-lg transition-colors {presetMediaTab === 'movie' ? 'bg-cream/20 text-cream border border-cream/30' : 'text-text-secondary hover:text-text-primary hover:bg-glass'}"
			onclick={() => presetMediaTab = 'movie'}
		>
			Movies
		</button>
		<button
			type="button"
			class="flex-1 px-4 py-2 text-sm font-medium rounded-lg transition-colors {presetMediaTab === 'tv' ? 'bg-cream/20 text-cream border border-cream/30' : 'text-text-secondary hover:text-text-primary hover:bg-glass'}"
			onclick={() => presetMediaTab = 'tv'}
		>
			TV Shows
		</button>
		<button
			type="button"
			class="flex-1 px-4 py-2 text-sm font-medium rounded-lg transition-colors {presetMediaTab === 'anime' ? 'bg-cream/20 text-cream border border-cream/30' : 'text-text-secondary hover:text-text-primary hover:bg-glass'}"
			onclick={() => presetMediaTab = 'anime'}
		>
			Anime
		</button>
	</div>

	{#if showAddPresetForm || editingPreset}
		<form
			class="p-4 bg-bg-elevated/50 rounded-xl space-y-4 border border-white/5"
			onsubmit={(e) => { e.preventDefault(); editingPreset ? handleUpdatePreset() : handleAddPreset(); }}
		>
			<div>
				<label for="preset-name" class={labelClass}>Preset Name</label>
				<input type="text" id="preset-name" bind:value={presetName} required class={inputClass} placeholder="My Custom Preset" disabled={editingPreset?.isBuiltIn} />
			</div>

			<div class="grid sm:grid-cols-2 gap-4">
				<div>
					<label for="preset-resolution" class={labelClass}>Resolution</label>
					<Select
						id="preset-resolution"
						bind:value={presetResolution}
						options={resolutionOptions.map(res => ({ value: res, label: res.toUpperCase() }))}
					/>
				</div>
				<div>
					<label for="preset-source" class={labelClass}>Source</label>
					<Select
						id="preset-source"
						bind:value={presetSource}
						options={sourceOptions.map(src => ({ value: src, label: src.charAt(0).toUpperCase() + src.slice(1) }))}
					/>
				</div>
			</div>

			<div>
				<label class="block text-sm text-text-secondary mb-2">HDR Formats (optional)</label>
				<p class="text-xs text-text-muted mb-3">Select preferred HDR formats. Empty = SDR is acceptable.</p>
				<div class="flex flex-wrap gap-2">
					{#each hdrOptions as hdr}
						<button
							type="button"
							class="px-3 py-1.5 text-xs rounded-lg transition-colors font-medium {presetHdrFormats.includes(hdr) ? 'bg-amber-600 text-white' : 'bg-bg-card text-text-muted hover:bg-bg-elevated'}"
							onclick={() => toggleHdrFormat(hdr)}
						>
							{hdr.toUpperCase()}
						</button>
					{/each}
				</div>
			</div>

			<div>
				<label class="block text-sm text-text-secondary mb-2">Audio Formats (optional)</label>
				<p class="text-xs text-text-muted mb-3">Select preferred audio formats. Empty = any audio.</p>
				<div class="flex flex-wrap gap-2">
					{#each audioOptions as audio}
						<button
							type="button"
							class="px-3 py-1.5 text-xs rounded-lg transition-colors font-medium {presetAudioFormats.includes(audio) ? 'bg-amber-600 text-white' : 'bg-bg-card text-text-muted hover:bg-bg-elevated'}"
							onclick={() => toggleAudioFormat(audio)}
						>
							{audio.toUpperCase()}
						</button>
					{/each}
				</div>
			</div>

			<div class="grid sm:grid-cols-3 gap-4">
				<div>
					<label for="preset-codec" class={labelClass}>Codec</label>
					<Select
						id="preset-codec"
						bind:value={presetCodec}
						options={codecOptions.map(codec => ({ value: codec, label: codec.toUpperCase() }))}
					/>
				</div>
				<div>
					<label for="preset-edition" class={labelClass}>Preferred Edition</label>
					<Select
						id="preset-edition"
						bind:value={presetPreferredEdition}
						options={editionOptions.map(edition => ({ value: edition, label: edition.charAt(0).toUpperCase() + edition.slice(1) }))}
					/>
				</div>
				<div>
					<label for="preset-seeders" class={labelClass}>Min Seeders</label>
					<input type="number" id="preset-seeders" bind:value={presetMinSeeders} min="0" class={inputClass} />
				</div>
			</div>

			<div class="flex flex-wrap gap-6">
				<label class="flex items-center gap-2 cursor-pointer">
					<input type="checkbox" bind:checked={presetPreferSeasonPacks} class="form-checkbox" />
					<span class="text-sm text-text-secondary">Prefer season packs for TV</span>
				</label>
				<label class="flex items-center gap-2 cursor-pointer">
					<input type="checkbox" bind:checked={presetAutoUpgrade} class="form-checkbox" />
					<span class="text-sm text-text-secondary">Auto-upgrade when better quality found</span>
				</label>
			</div>

			<div class="flex gap-2">
				<button type="submit" class="liquid-btn">
					{editingPreset ? 'Update Preset' : 'Add Preset'}
				</button>
				{#if editingPreset}
					<button type="button" class="liquid-btn !bg-white/5 !border-t-white/10 text-text-secondary hover:text-text-primary" onclick={resetPresetForm}>
						Cancel
					</button>
				{/if}
			</div>
		</form>
	{/if}

	{#if filteredPresets.length === 0}
		<p class="text-text-muted py-4">No quality presets configured. Built-in presets will be added on first use.</p>
	{:else}
		<div class="space-y-2">
			{#each filteredPresets as preset}
				<!-- svelte-ignore a11y_no_static_element_interactions -->
				<div
					class="p-4 bg-bg-elevated/50 rounded-xl border transition-all
						{preset.isDefault ? 'ring-1 ring-amber-500/50' : ''}
						{!preset.enabled ? 'opacity-50' : ''}
						{draggingPresetId === preset.id ? 'opacity-50 scale-[0.98] border-amber-500/50' : 'border-white/5'}
						{dragOverPresetId === preset.id ? 'border-amber-400 bg-amber-500/10' : ''}"
					draggable="true"
					ondragstart={(e: DragEvent) => handlePresetDragStart(e, preset.id)}
					ondragover={(e: DragEvent) => handlePresetDragOver(e, preset.id)}
					ondragleave={handlePresetDragLeave}
					ondragend={handlePresetDragEnd}
					ondrop={(e: DragEvent) => handlePresetDrop(e, preset.id)}
				>
					<div class="flex items-center justify-between gap-4">
						<!-- Drag Handle -->
						<div class="flex-shrink-0 cursor-grab active:cursor-grabbing text-text-muted hover:text-text-secondary" title="Drag to reorder">
							<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
								<path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M4 8h16M4 16h16" />
							</svg>
						</div>

						<!-- Enable/Disable Toggle -->
						<button
							class="flex-shrink-0 w-10 h-6 rounded-full transition-colors {preset.enabled ? 'bg-amber-600' : 'bg-white/10'}"
							onclick={async () => {
								try {
									await toggleQualityPresetEnabled(preset.id, !preset.enabled);
									preset.enabled = !preset.enabled;
								} catch (e) {
									console.error('Failed to toggle preset:', e);
								}
							}}
							title={preset.enabled ? 'Disable preset' : 'Enable preset'}
						>
							<div class="w-4 h-4 m-1 rounded-full bg-white transition-transform {preset.enabled ? 'translate-x-4' : 'translate-x-0'}"></div>
						</button>

						<div class="flex-1 min-w-0">
							<div class="flex items-center gap-2 flex-wrap">
								<h3 class="font-medium text-text-primary">{preset.name}</h3>
								<span class="px-2 py-0.5 text-xs rounded-lg {getResolutionColor(preset.resolution)} text-white">
									{preset.resolution.toUpperCase()}
								</span>
								{#if preset.isBuiltIn}
									<span class="px-2 py-0.5 text-xs rounded-lg bg-bg-card text-text-muted">Built-in</span>
								{/if}
								{#if preset.isDefault}
									<span class="px-2 py-0.5 text-xs rounded-lg bg-amber-900/50 text-amber-300">Default</span>
								{/if}
								{#if preset.autoUpgrade}
									<span class="px-2 py-0.5 text-xs rounded-lg bg-green-900/50 text-green-300">Auto-upgrade</span>
								{/if}
								<span class="px-2 py-0.5 text-xs rounded-lg bg-white/5 text-text-muted" title="Priority (lower = higher priority)">
									#{preset.priority}
								</span>
							</div>
							<p class="text-sm text-text-secondary mt-1">
								{getPresetDescription(preset)}
							</p>
							{#if preset.audioFormats && preset.audioFormats.length > 0}
								<p class="text-xs text-text-muted mt-1">
									Audio: {preset.audioFormats.map(a => a.toUpperCase()).join(', ')}
								</p>
							{/if}

							<!-- Anime preferences -->
							{#if preset.mediaType === 'anime'}
								<div class="flex items-center gap-4 mt-3 pt-3 border-t border-white/5">
									<label class="flex items-center gap-2 cursor-pointer">
										<input
											type="checkbox"
											checked={preset.preferDualAudio}
											onchange={async (e) => {
												const target = e.target as HTMLInputElement;
												try {
													await updateAnimePresetPreference(preset.id, 'preferDualAudio', target.checked);
													preset.preferDualAudio = target.checked;
												} catch (err) {
													console.error('Failed to update preference:', err);
												}
											}}
											class="form-checkbox"
										/>
										<span class="text-xs text-text-secondary">Dual Audio</span>
									</label>
									<label class="flex items-center gap-2 cursor-pointer">
										<input
											type="checkbox"
											checked={preset.preferDubbed}
											onchange={async (e) => {
												const target = e.target as HTMLInputElement;
												try {
													await updateAnimePresetPreference(preset.id, 'preferDubbed', target.checked);
													preset.preferDubbed = target.checked;
												} catch (err) {
													console.error('Failed to update preference:', err);
												}
											}}
											class="form-checkbox"
										/>
										<span class="text-xs text-text-secondary">Dubbed</span>
									</label>
									<div class="flex items-center gap-2">
										<span class="text-xs text-text-muted">Language:</span>
										<select
											value={preset.preferredLanguage || 'any'}
											onchange={async (e) => {
												const target = e.target as HTMLSelectElement;
												try {
													await updateAnimePresetPreference(preset.id, 'preferredLanguage', target.value);
													preset.preferredLanguage = target.value;
												} catch (err) {
													console.error('Failed to update preference:', err);
												}
											}}
											class="form-select-sm"
										>
											<option value="any">Any</option>
											<option value="english">English</option>
											<option value="japanese">Japanese</option>
										</select>
									</div>
								</div>
							{/if}
						</div>
						<div class="flex gap-2 flex-shrink-0">
							{#if !preset.isDefault}
								<button
									class="liquid-btn-sm"
									onclick={() => handleSetDefaultPreset(preset.id)}
								>
									Set Default
								</button>
							{/if}
							{#if !preset.isBuiltIn}
								<button
									class="liquid-btn-sm"
									onclick={() => startEditPreset(preset)}
								>
									Edit
								</button>
								<button
									class="liquid-btn-sm !bg-white/5 !border-t-white/10 text-text-secondary hover:text-text-primary"
									onclick={() => handleDeletePreset(preset.id)}
								>
									Delete
								</button>
							{/if}
						</div>
					</div>
				</div>
			{/each}
		</div>
	{/if}
</section>

<!-- Format Filtering -->
<FormatFilteringSettings />
