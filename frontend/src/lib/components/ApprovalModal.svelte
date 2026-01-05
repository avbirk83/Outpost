<script lang="ts">
	import { getQualityProfiles, getTmdbImageUrl, type Request, type QualityProfile } from '$lib/api';
	import { onMount } from 'svelte';
	import TypeBadge from './TypeBadge.svelte';

	interface Props {
		request: Request;
		onApprove: (qualityProfileId: number) => void;
		onCancel: () => void;
	}

	let { request, onApprove, onCancel }: Props = $props();

	let profiles: QualityProfile[] = $state([]);
	let selectedProfileId: number = $state(0);
	let loading = $state(true);

	onMount(async () => {
		try {
			profiles = await getQualityProfiles();
			if (profiles.length > 0) {
				selectedProfileId = profiles[0].id;
			}
		} catch (e) {
			console.error('Failed to load quality profiles:', e);
		} finally {
			loading = false;
		}
	});

	function handleApprove() {
		onApprove(selectedProfileId);
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
</script>

<svelte:window onkeydown={handleKeydown} />

<div
	class="modal-overlay"
	onclick={handleBackdropClick}
	role="dialog"
	aria-modal="true"
	aria-labelledby="modal-title"
>
	<div class="modal-container">
		<!-- Header with poster -->
		<div class="relative h-32 bg-gradient-to-b from-white/10 to-transparent">
			{#if request.backdropPath}
				<img
					src={getTmdbImageUrl(request.backdropPath, 'w780')}
					alt=""
					class="absolute inset-0 w-full h-full object-cover opacity-30"
				/>
			{/if}
			<div class="absolute inset-0 bg-gradient-to-t from-bg-base via-bg-base/50 to-transparent" />

			<div class="absolute bottom-0 left-0 right-0 p-4 flex items-end gap-4">
				{#if request.posterPath}
					<img
						src={getTmdbImageUrl(request.posterPath, 'w154')}
						alt={request.title}
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
						{request.title}
					</h2>
					<div class="flex items-center gap-2 mt-1">
						<TypeBadge type={request.type} />
						{#if request.year}
							<span class="text-sm text-text-secondary">{request.year}</span>
						{/if}
					</div>
				</div>
			</div>
		</div>

		<!-- Content -->
		<div class="p-4 pt-12 space-y-4">
			<div>
				<label for="quality-profile" class="block text-sm font-medium text-text-secondary mb-2">
					Quality Profile
				</label>
				{#if loading}
					<div class="h-10 bg-bg-elevated rounded-lg animate-pulse"></div>
				{:else if profiles.length === 0}
					<p class="text-sm text-text-muted">No quality profiles found. Using default.</p>
				{:else}
					<select
						id="quality-profile"
						bind:value={selectedProfileId}
						class="w-full px-3 py-2.5 bg-bg-elevated border border-white/10 rounded-lg text-text-primary focus:outline-none focus:ring-2 focus:ring-white/30"
					>
						{#each profiles as profile}
							<option value={profile.id}>{profile.name}</option>
						{/each}
					</select>
				{/if}
			</div>

			<div class="bg-white/5 rounded-lg p-3">
				<p class="text-sm text-text-secondary">
					<span class="text-text-primary font-medium">What happens next:</span><br/>
					The item will be added to the wanted list and searched for on your indexers.
					If found, it will be sent to your download client automatically.
				</p>
			</div>
		</div>

		<!-- Footer -->
		<div class="flex items-center justify-end gap-3 p-4 border-t border-border-subtle">
			<button
				onclick={onCancel}
				class="px-4 py-2 rounded-lg text-sm font-medium bg-white/10 text-text-secondary hover:bg-white/20 hover:text-text-primary transition-colors"
			>
				Cancel
			</button>
			<button
				onclick={handleApprove}
				disabled={loading}
				class="px-4 py-2 rounded-lg text-sm font-medium bg-green-600 text-white hover:bg-green-500 disabled:opacity-50 transition-colors flex items-center gap-2"
			>
				<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
				</svg>
				Approve & Search
			</button>
		</div>
	</div>
</div>
