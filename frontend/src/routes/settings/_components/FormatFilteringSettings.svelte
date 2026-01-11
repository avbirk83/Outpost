<script lang="ts">
	import { onMount } from 'svelte';
	import type { FormatSettings } from '$lib/api';
	import { getFormatSettings, saveFormatSettings } from '$lib/api';

	// Format settings state
	let formatSettings: FormatSettings | null = $state(null);
	let savingFormats = $state(false);
	let formatsSaved = $state(false);
	let newFormat = $state('');
	let newRejectKeyword = $state('');

	// All containers and keywords (sorted for display)
	const sortedContainers = $derived(
		[...(formatSettings?.acceptedContainers ?? [])].sort()
	);
	const sortedRejectedKeywords = $derived(
		[...(formatSettings?.rejectedKeywords ?? [])].sort()
	);

	onMount(async () => {
		await loadFormatSettings();
	});

	async function loadFormatSettings() {
		try {
			formatSettings = await getFormatSettings();
		} catch (e) {
			console.error('Failed to load format settings:', e);
		}
	}

	async function handleSaveFormatSettings() {
		if (!formatSettings) return;
		savingFormats = true;
		try {
			await saveFormatSettings(formatSettings);
			formatsSaved = true;
			setTimeout(() => formatsSaved = false, 3000);
		} catch (e) {
			console.error('Failed to save format settings:', e);
		} finally {
			savingFormats = false;
		}
	}

	function resetToDefaults() {
		formatSettings = {
			acceptedContainers: ['mkv', 'mp4', 'avi', 'mov', 'webm', 'm4v', 'ts', 'm2ts', 'wmv', 'flv'],
			rejectedKeywords: [
				'bdmv', 'video_ts', 'iso', 'full disc', 'complete disc', 'disc1', 'disc2',
				'rar', 'zip', '7z',
				'cam', 'camrip', 'hdcam', 'hdts', 'telesync', 'telecine', 'ts-scr',
				'dvdscr', 'dvdscreener', 'screener', 'scr', 'r5', 'workprint',
				'sample',
				'3d', 'hsbs', 'hou',
			],
			autoBlocklist: true,
		};
	}

	function addContainer() {
		if (!formatSettings || !newFormat.trim()) return;
		const format = newFormat.trim().toLowerCase().replace(/^\./, '');
		if (!format) return;
		if (formatSettings.acceptedContainers.includes(format)) {
			newFormat = '';
			return;
		}
		formatSettings.acceptedContainers = [...formatSettings.acceptedContainers, format];
		newFormat = '';
	}

	function removeContainer(format: string) {
		if (!formatSettings) return;
		formatSettings.acceptedContainers = formatSettings.acceptedContainers.filter(c => c !== format);
	}

	function handleFormatKeydown(e: KeyboardEvent) {
		if (e.key === 'Enter') {
			e.preventDefault();
			addContainer();
		}
	}

	function addRejectKeyword() {
		if (!formatSettings || !newRejectKeyword.trim()) return;
		const keyword = newRejectKeyword.trim().toLowerCase();
		if (!formatSettings.rejectedKeywords) {
			formatSettings.rejectedKeywords = [];
		}
		if (formatSettings.rejectedKeywords.includes(keyword)) {
			newRejectKeyword = '';
			return;
		}
		formatSettings.rejectedKeywords = [...formatSettings.rejectedKeywords, keyword];
		newRejectKeyword = '';
	}

	function removeRejectKeyword(keyword: string) {
		if (!formatSettings) return;
		formatSettings.rejectedKeywords = formatSettings.rejectedKeywords.filter(k => k !== keyword);
	}

	function handleRejectKeydown(e: KeyboardEvent) {
		if (e.key === 'Enter') {
			e.preventDefault();
			addRejectKeyword();
		}
	}
</script>

<section class="glass-card p-6 space-y-4">
	<div class="flex items-center gap-3">
		<div class="w-10 h-10 rounded-xl bg-red-600/20 flex items-center justify-center">
			<svg class="w-5 h-5 text-red-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
				<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
			</svg>
		</div>
		<div>
			<h2 class="text-lg font-semibold text-text-primary">Format Filtering</h2>
			<p class="text-sm text-text-secondary">Control which file formats can be downloaded</p>
		</div>
	</div>

	{#if formatSettings}
		<div class="space-y-4">
			<div>
				<label class="block text-sm text-text-secondary mb-2">Accepted Container Formats</label>
				<p class="text-xs text-text-muted mb-3">File extensions that will be accepted. Add or remove as needed.</p>
				<div class="flex items-center gap-2 mb-3">
					<input
						type="text"
						bind:value={newFormat}
						onkeydown={handleFormatKeydown}
						placeholder="e.g. mkv, mp4, avi"
						class="flex-1 max-w-[200px] px-3 py-1.5 text-sm bg-bg-elevated border border-border-subtle rounded-lg text-text-primary placeholder:text-text-muted focus:outline-none focus:border-cream/50"
					/>
					<button
						type="button"
						onclick={addContainer}
						disabled={!newFormat.trim()}
						class="px-3 py-1.5 text-sm rounded-lg bg-green-500/20 text-green-400 hover:bg-green-500/30 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
					>
						Add
					</button>
				</div>
				{#if sortedContainers.length > 0}
					<div class="flex flex-wrap gap-2">
						{#each sortedContainers as container}
							<div class="flex items-center gap-1 px-3 py-1.5 text-sm rounded-lg bg-green-600 text-white font-medium uppercase">
								<span>{container}</span>
								<button
									type="button"
									onclick={() => removeContainer(container)}
									class="ml-1 hover:text-green-200 transition-colors"
									aria-label="Remove {container}"
								>
									<svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
										<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
									</svg>
								</button>
							</div>
						{/each}
					</div>
				{:else}
					<p class="text-xs text-text-muted italic">No accepted formats - all containers will be rejected!</p>
				{/if}
			</div>

			<!-- Rejected Keywords -->
			<div class="pt-4 border-t border-white/5">
				<label class="block text-sm text-text-secondary mb-2">Rejected Keywords</label>
				<p class="text-xs text-text-muted mb-3">Releases containing these keywords will be rejected (case-insensitive).</p>
				<div class="flex items-center gap-2 mb-3">
					<input
						type="text"
						bind:value={newRejectKeyword}
						onkeydown={handleRejectKeydown}
						placeholder="e.g. bdmv, rar, cam, hdts"
						class="flex-1 max-w-[200px] px-3 py-1.5 text-sm bg-bg-elevated border border-border-subtle rounded-lg text-text-primary placeholder:text-text-muted focus:outline-none focus:border-cream/50"
					/>
					<button
						type="button"
						onclick={addRejectKeyword}
						disabled={!newRejectKeyword.trim()}
						class="px-3 py-1.5 text-sm rounded-lg bg-red-500/20 text-red-400 hover:bg-red-500/30 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
					>
						Add
					</button>
				</div>
				{#if sortedRejectedKeywords.length > 0}
					<div class="flex flex-wrap gap-2">
						{#each sortedRejectedKeywords as keyword}
							<div class="flex items-center gap-1 px-3 py-1.5 text-sm rounded-lg bg-red-600 text-white font-medium">
								<span>{keyword}</span>
								<button
									type="button"
									onclick={() => removeRejectKeyword(keyword)}
									class="ml-1 hover:text-red-200 transition-colors"
									aria-label="Remove {keyword}"
								>
									<svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
										<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
									</svg>
								</button>
							</div>
						{/each}
					</div>
				{:else}
					<p class="text-xs text-text-muted italic">No rejected keywords - all releases will be accepted</p>
				{/if}
			</div>

			<!-- Auto-blocklist toggle -->
			<div class="pt-4 border-t border-white/5">
				<label class="flex items-center gap-2 cursor-pointer">
					<input type="checkbox" bind:checked={formatSettings.autoBlocklist} class="form-checkbox" />
					<div>
						<span class="text-sm text-text-secondary">Auto-Blocklist</span>
						<p class="text-xs text-text-muted">Automatically add rejected releases to blocklist</p>
					</div>
				</label>
			</div>

			<div class="flex items-center gap-3 pt-4">
				<button class="liquid-btn" onclick={handleSaveFormatSettings} disabled={savingFormats}>
					{savingFormats ? 'Saving...' : 'Save Format Settings'}
				</button>
				<button
					type="button"
					onclick={resetToDefaults}
					class="px-4 py-2 text-sm rounded-lg bg-bg-elevated hover:bg-bg-card border border-border-subtle text-text-secondary transition-colors"
				>
					Reset to Defaults
				</button>
				{#if formatsSaved}
					<span class="text-sm text-green-400 flex items-center gap-1">
						<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
						</svg>
						Saved!
					</span>
				{/if}
			</div>
		</div>
	{:else}
		<div class="flex items-center gap-3 py-4">
			<div class="spinner-md text-red-400"></div>
			<span class="text-text-secondary">Loading format settings...</span>
		</div>
	{/if}
</section>
