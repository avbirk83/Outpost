<script lang="ts">
	import { downloadBackup, restoreBackup, type RestoreResult } from '$lib/api';
	import { toast } from '$lib/stores/toast';

	let downloading = $state(false);
	let restoring = $state(false);
	let restoreMode: 'merge' | 'replace' = $state('merge');
	let selectedFile: File | null = $state(null);
	let showRestoreConfirm = $state(false);
	let restoreResult: RestoreResult | null = $state(null);
	let restoreError: string | null = $state(null);

	async function handleDownload() {
		try {
			downloading = true;
			await downloadBackup();
			toast.success('Backup downloaded');
		} catch (e) {
			toast.error(e instanceof Error ? e.message : 'Failed to download backup');
		} finally {
			downloading = false;
		}
	}

	function handleFileSelect(e: Event) {
		const input = e.target as HTMLInputElement;
		if (input.files && input.files.length > 0) {
			selectedFile = input.files[0];
			restoreResult = null;
			restoreError = null;
		}
	}

	async function handleRestore() {
		if (!selectedFile) return;

		try {
			restoring = true;
			restoreResult = null;
			restoreError = null;
			const result = await restoreBackup(selectedFile, restoreMode);
			restoreResult = result;
			showRestoreConfirm = false;
			if (result.success) {
				toast.success('Backup restored successfully');
			}
		} catch (e) {
			restoreError = e instanceof Error ? e.message : 'Failed to restore backup';
			toast.error('Failed to restore backup');
		} finally {
			restoring = false;
		}
	}

	function formatRestoredItems(restored: Record<string, number>): string[] {
		const labels: Record<string, string> = {
			settings: 'Settings',
			users: 'Users',
			libraries: 'Libraries',
			downloadClients: 'Download Clients',
			prowlarrConfig: 'Prowlarr Config',
			indexers: 'Indexers',
			qualityProfiles: 'Quality Profiles',
			qualityPresets: 'Quality Presets',
			customFormats: 'Custom Formats',
			collections: 'Collections',
			collectionItems: 'Collection Items',
			namingTemplates: 'Naming Templates',
			releaseFilters: 'Release Filters',
			delayProfiles: 'Delay Profiles',
			blockedGroups: 'Blocked Groups',
			trustedGroups: 'Trusted Groups',
			scheduledTasks: 'Scheduled Tasks'
		};

		return Object.entries(restored)
			.filter(([_, count]) => count > 0)
			.map(([key, count]) => `${labels[key] || key}: ${count}`);
	}
</script>

<section class="glass-card p-6 space-y-4">
	<div class="flex items-center gap-3">
		<div class="w-10 h-10 rounded-xl bg-indigo-600/20 flex items-center justify-center">
			<svg class="w-5 h-5 text-indigo-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
				<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 7H5a2 2 0 00-2 2v9a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-3m-1 4l-3 3m0 0l-3-3m3 3V4" />
			</svg>
		</div>
		<div>
			<h2 class="text-lg font-semibold text-text-primary">Backup & Restore</h2>
			<p class="text-sm text-text-secondary">Export and import your configuration</p>
		</div>
	</div>

	<!-- Download Backup -->
	<div class="pt-2">
		<label class="block text-sm text-text-secondary mb-1.5">Download Backup</label>
		<p class="text-xs text-text-muted mb-3">
			Download a JSON file containing your settings, libraries, download clients, indexers, quality presets, and more.
		</p>
		<button
			class="liquid-btn disabled:opacity-50"
			onclick={handleDownload}
			disabled={downloading}
		>
			{#if downloading}
				<span class="flex items-center gap-2">
					<svg class="w-4 h-4 animate-spin" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
						<path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
					</svg>
					Downloading...
				</span>
			{:else}
				<span class="flex items-center gap-2">
					<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4" />
					</svg>
					Download Backup
				</span>
			{/if}
		</button>
	</div>

	<!-- Restore Backup -->
	<div class="pt-4 border-t border-border-subtle">
		<label class="block text-sm text-text-secondary mb-1.5">Restore Backup</label>
		<p class="text-xs text-text-muted mb-3">
			Upload a backup file to restore your configuration. Note: User passwords are not included in backups and will need to be reset.
		</p>

		<!-- File Input -->
		<div class="mb-4">
			<input
				type="file"
				accept=".json"
				onchange={handleFileSelect}
				class="block w-full text-sm text-text-secondary
					file:mr-4 file:py-2 file:px-4
					file:rounded-lg file:border-0
					file:text-sm file:font-medium
					file:bg-white/10 file:text-text-primary
					hover:file:bg-white/20
					file:cursor-pointer cursor-pointer"
			/>
		</div>

		{#if selectedFile}
			<!-- Restore Mode Selection -->
			<div class="mb-4 p-4 bg-bg-elevated/50 rounded-xl border border-white/5">
				<label class="block text-sm text-text-secondary mb-3">Restore Mode</label>
				<div class="space-y-2">
					<label class="flex items-start gap-3 cursor-pointer p-2 rounded-lg hover:bg-white/5 transition-colors">
						<input
							type="radio"
							name="restore-mode"
							value="merge"
							bind:group={restoreMode}
							class="mt-1"
						/>
						<div>
							<span class="text-text-primary font-medium">Merge</span>
							<p class="text-xs text-text-muted mt-0.5">
								Keep existing data and add new items from backup. Existing items with the same name are not modified.
							</p>
						</div>
					</label>
					<label class="flex items-start gap-3 cursor-pointer p-2 rounded-lg hover:bg-white/5 transition-colors">
						<input
							type="radio"
							name="restore-mode"
							value="replace"
							bind:group={restoreMode}
							class="mt-1"
						/>
						<div>
							<span class="text-text-primary font-medium">Replace</span>
							<p class="text-xs text-text-muted mt-0.5">
								Clear existing settings and replace with backup data. Libraries and users are updated but not deleted.
							</p>
						</div>
					</label>
				</div>
			</div>

			<!-- Confirm & Restore -->
			{#if showRestoreConfirm}
				<div class="p-4 bg-amber-900/20 border border-amber-600/30 rounded-xl mb-4">
					<div class="flex items-center gap-2 text-amber-400 mb-2">
						<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
						</svg>
						<span class="font-medium">Confirm Restore</span>
					</div>
					<p class="text-sm text-text-secondary mb-3">
						{#if restoreMode === 'replace'}
							This will replace your current settings with the backup. Some data may be overwritten.
						{:else}
							This will merge the backup with your current settings. New items will be added.
						{/if}
					</p>
					<div class="flex gap-2">
						<button
							class="liquid-btn disabled:opacity-50"
							onclick={handleRestore}
							disabled={restoring}
						>
							{restoring ? 'Restoring...' : 'Confirm Restore'}
						</button>
						<button
							class="liquid-btn !bg-white/5 !border-t-white/10 text-text-secondary"
							onclick={() => showRestoreConfirm = false}
							disabled={restoring}
						>
							Cancel
						</button>
					</div>
				</div>
			{:else}
				<button
					class="liquid-btn"
					onclick={() => showRestoreConfirm = true}
				>
					<span class="flex items-center gap-2">
						<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-8l-4-4m0 0L8 8m4-4v12" />
						</svg>
						Restore from Backup
					</span>
				</button>
			{/if}
		{/if}

		<!-- Restore Error -->
		{#if restoreError}
			<div class="mt-4 p-4 bg-red-900/20 border border-red-600/30 rounded-xl">
				<div class="flex items-center gap-2 text-red-400">
					<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
					</svg>
					<span class="font-medium">Restore Failed</span>
				</div>
				<p class="text-sm text-text-secondary mt-1">{restoreError}</p>
			</div>
		{/if}

		<!-- Restore Result -->
		{#if restoreResult}
			<div class="mt-4 p-4 bg-green-900/20 border border-green-600/30 rounded-xl">
				<div class="flex items-center gap-2 text-green-400 mb-2">
					<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
					</svg>
					<span class="font-medium">Restore Complete</span>
				</div>

				<div class="text-sm text-text-secondary space-y-1">
					{#each formatRestoredItems(restoreResult.restored) as item}
						<div>{item}</div>
					{/each}
				</div>

				{#if restoreResult.warnings && restoreResult.warnings.length > 0}
					<div class="mt-3 pt-3 border-t border-white/10">
						<div class="text-sm text-amber-400 font-medium mb-1">Warnings:</div>
						{#each restoreResult.warnings as warning}
							<div class="text-xs text-text-muted">{warning}</div>
						{/each}
					</div>
				{/if}
			</div>
		{/if}
	</div>
</section>
