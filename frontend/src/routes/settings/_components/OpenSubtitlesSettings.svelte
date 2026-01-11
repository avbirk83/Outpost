<script lang="ts">
	import { onMount } from 'svelte';
	import { getSettings, saveSettings, testOpenSubtitlesConnection, COMMON_LANGUAGES } from '$lib/api';

	// OpenSubtitles settings state
	let osApiKey = $state('');
	let osLanguages = $state<string[]>(['en']);
	let osAutoDownload = $state(false);
	let osHearingImpaired = $state('include');
	let savingOS = $state(false);
	let osSaved = $state(false);
	let testingOS = $state(false);
	let osTestResult = $state<{ success: boolean; message: string } | null>(null);

	onMount(async () => {
		await loadOSSettings();
	});

	async function loadOSSettings() {
		try {
			const settings = await getSettings();
			osApiKey = settings.opensubtitles_api_key || '';
			osLanguages = settings.opensubtitles_languages ? settings.opensubtitles_languages.split(',') : ['en'];
			osAutoDownload = settings.opensubtitles_auto_download === 'true';
			osHearingImpaired = settings.opensubtitles_hearing_impaired || 'include';
		} catch (e) {
			console.error('Failed to load OpenSubtitles settings:', e);
		}
	}

	async function handleSaveOSSettings() {
		savingOS = true;
		try {
			await saveSettings({
				opensubtitles_api_key: osApiKey,
				opensubtitles_languages: osLanguages.join(','),
				opensubtitles_auto_download: osAutoDownload ? 'true' : 'false',
				opensubtitles_hearing_impaired: osHearingImpaired
			});
			osSaved = true;
			setTimeout(() => osSaved = false, 3000);
		} catch (e) {
			console.error('Failed to save OpenSubtitles settings:', e);
		} finally {
			savingOS = false;
		}
	}

	async function handleTestOS() {
		testingOS = true;
		osTestResult = null;
		try {
			const result = await testOpenSubtitlesConnection(osApiKey);
			osTestResult = {
				success: result.success,
				message: result.success ? 'Connection successful!' : (result.error || 'Connection failed')
			};
		} catch (e) {
			osTestResult = {
				success: false,
				message: e instanceof Error ? e.message : 'Test failed'
			};
		} finally {
			testingOS = false;
		}
	}

	function toggleLanguage(code: string) {
		if (osLanguages.includes(code)) {
			osLanguages = osLanguages.filter(l => l !== code);
		} else {
			osLanguages = [...osLanguages, code];
		}
	}
</script>

<section class="glass-card p-6 space-y-4">
	<div class="flex items-center gap-3">
		<div class="w-10 h-10 rounded-xl bg-[#8BC34A] flex items-center justify-center">
			<!-- OpenSubtitles Logo - CC/subtitle icon -->
			<svg class="w-6 h-6" viewBox="0 0 24 24" fill="white">
				<path d="M20 4H4c-1.1 0-2 .9-2 2v12c0 1.1.9 2 2 2h16c1.1 0 2-.9 2-2V6c0-1.1-.9-2-2-2zM4 12h4v2H4v-2zm10 6H4v-2h10v2zm6 0h-4v-2h4v2zm0-4H10v-2h10v2z"/>
			</svg>
		</div>
		<div>
			<h2 class="text-lg font-semibold text-text-primary">OpenSubtitles</h2>
			<p class="text-sm text-text-secondary">Automatically download subtitles for your media</p>
		</div>
	</div>

	<div class="space-y-4">
		<!-- API Key -->
		<div>
			<label class="block text-sm text-text-secondary mb-1">API Key</label>
			<p class="text-xs text-text-muted mb-2">Get your free API key from <a href="https://www.opensubtitles.com/consumers" target="_blank" rel="noopener noreferrer" class="text-amber-400 hover:text-amber-300">opensubtitles.com</a></p>
			<div class="flex gap-2">
				<input
					type="password"
					bind:value={osApiKey}
					placeholder="Enter your API key..."
					class="flex-1 px-3 py-2 text-sm bg-bg-elevated border border-border-subtle rounded-lg text-text-primary placeholder:text-text-muted focus:outline-none focus:border-cream/50"
				/>
				<button
					onclick={handleTestOS}
					disabled={!osApiKey || testingOS}
					class="px-4 py-2 text-sm rounded-lg bg-bg-elevated hover:bg-bg-card border border-border-subtle text-text-secondary hover:text-text-primary disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
				>
					{testingOS ? 'Testing...' : 'Test'}
				</button>
			</div>
			{#if osTestResult}
				<p class="text-xs mt-2 {osTestResult.success ? 'text-green-400' : 'text-red-400'}">
					{osTestResult.message}
				</p>
			{/if}
		</div>

		<!-- Languages -->
		<div>
			<label class="block text-sm text-text-secondary mb-1">Preferred Languages</label>
			<p class="text-xs text-text-muted mb-2">Select which subtitle languages to download</p>
			<div class="flex flex-wrap gap-2">
				{#each COMMON_LANGUAGES.slice(0, 12) as lang}
					<button
						type="button"
						onclick={() => toggleLanguage(lang.code)}
						class="px-3 py-1.5 text-sm rounded-lg transition-colors {osLanguages.includes(lang.code)
							? 'bg-purple-500/30 text-purple-300 border border-purple-500/50'
							: 'bg-bg-elevated border border-border-subtle text-text-secondary hover:text-text-primary hover:border-border-hover'}"
					>
						{lang.name}
					</button>
				{/each}
			</div>
		</div>

		<!-- Hearing Impaired -->
		<div>
			<label class="block text-sm text-text-secondary mb-1">Hearing Impaired Subtitles</label>
			<div class="flex gap-3">
				{#each [
					{ value: 'include', label: 'Include' },
					{ value: 'only', label: 'Only HI' },
					{ value: 'exclude', label: 'Exclude' }
				] as option}
					<label class="flex items-center gap-2 cursor-pointer">
						<input
							type="radio"
							name="hi-preference"
							value={option.value}
							checked={osHearingImpaired === option.value}
							onchange={() => osHearingImpaired = option.value}
							class="accent-purple-500"
						/>
						<span class="text-sm text-text-secondary">{option.label}</span>
					</label>
				{/each}
			</div>
		</div>

		<!-- Auto Download -->
		<label class="flex items-center gap-3 cursor-pointer pt-2">
			<button
				type="button"
				class="relative w-12 h-6 rounded-full transition-colors {osAutoDownload ? 'bg-purple-600' : 'bg-gray-600'}"
				onclick={() => osAutoDownload = !osAutoDownload}
			>
				<span class="absolute left-1 top-1 w-4 h-4 bg-white rounded-full transition-transform duration-200 {osAutoDownload ? 'translate-x-6' : ''}"></span>
			</button>
			<div>
				<span class="text-text-primary font-medium">Auto-download Subtitles</span>
				<p class="text-xs text-text-muted">Automatically download subtitles when new media is imported</p>
			</div>
		</label>

		<div class="flex items-center gap-3 pt-2">
			<button class="liquid-btn" onclick={handleSaveOSSettings} disabled={savingOS}>
				{savingOS ? 'Saving...' : 'Save Subtitle Settings'}
			</button>
			{#if osSaved}
				<span class="text-sm text-green-400 flex items-center gap-1">
					<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
					</svg>
					Saved!
				</span>
			{/if}
		</div>
	</div>
</section>
