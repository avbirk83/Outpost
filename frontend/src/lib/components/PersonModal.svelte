<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import { goto } from '$app/navigation';
	import { getPersonDetail, getTmdbImageUrl, getImageUrl, type PersonDetail } from '$lib/api';

	interface Props {
		personId: number | null;
		personName?: string;
		onClose: () => void;
	}

	let { personId, personName = '', onClose }: Props = $props();

	let person = $state<PersonDetail | null>(null);
	let loading = $state(true);
	let error = $state<string | null>(null);
	let portalTarget: HTMLElement | null = null;
	let overlayEl: HTMLElement;

	// Portal setup
	onMount(() => {
		portalTarget = document.createElement('div');
		portalTarget.id = 'person-modal-portal';
		document.body.appendChild(portalTarget);
		document.body.style.overflow = 'hidden';
	});

	onDestroy(() => {
		if (portalTarget?.parentNode) {
			portalTarget.parentNode.removeChild(portalTarget);
		}
		document.body.style.overflow = '';
	});

	// Portal the overlay
	$effect(() => {
		if (personId && overlayEl && portalTarget) {
			portalTarget.appendChild(overlayEl);
		}
	});

	// Fetch person data when ID changes
	$effect(() => {
		if (personId) {
			loading = true;
			error = null;
			getPersonDetail(personId)
				.then(data => { person = data; })
				.catch(e => { error = e.message; })
				.finally(() => { loading = false; });
		}
	});

	function handleKeydown(e: KeyboardEvent) {
		if (e.key === 'Escape') {
			onClose();
		}
	}

	function handleBackdropClick(e: MouseEvent) {
		if (e.target === e.currentTarget) {
			onClose();
		}
	}

	function navigateToMedia(item: { id: number; type: 'movie' | 'show' }) {
		onClose();
		if (item.type === 'movie') {
			goto(`/movies/${item.id}`);
		} else {
			goto(`/tv/${item.id}`);
		}
	}

	function calculateAge(birthday: string, deathday: string | null): number | null {
		if (!birthday) return null;
		const birth = new Date(birthday);
		const end = deathday ? new Date(deathday) : new Date();
		const age = end.getFullYear() - birth.getFullYear();
		return age;
	}

	function formatDate(dateStr: string | null): string {
		if (!dateStr) return '';
		return new Date(dateStr).toLocaleDateString('en-US', {
			year: 'numeric',
			month: 'long',
			day: 'numeric'
		});
	}
</script>

{#if personId}
	<div
		bind:this={overlayEl}
		class="search-backdrop animate-fade-in"
		onclick={handleBackdropClick}
		onkeydown={handleKeydown}
		role="dialog"
		aria-modal="true"
		aria-label={personName || 'Person details'}
		tabindex="-1"
	>
		<div class="max-w-3xl mx-auto mt-[10vh] animate-slide-down px-4 max-h-[80vh] overflow-y-auto scrollbar-thin">
			<div class="liquid-panel p-6 relative">
				<!-- Close button -->
				<button onclick={onClose} class="btn-close absolute top-4 right-4" aria-label="Close">
					<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
					</svg>
				</button>

				{#if loading}
					<div class="flex items-center justify-center py-16">
						<div class="spinner-xl text-cream"></div>
					</div>
				{:else if error}
					<div class="text-center py-16 text-text-muted">
						<p>Failed to load person details</p>
					</div>
				{:else if person}
					<div class="flex gap-6">
						<!-- Profile image -->
						<div class="flex-shrink-0">
							<div class="w-32 h-48 rounded-lg overflow-hidden bg-bg-elevated">
								{#if person.profilePath}
									<img
										src={getTmdbImageUrl(person.profilePath, 'w185')}
										alt={person.name}
										class="w-full h-full object-cover"
									/>
								{:else}
									<div class="w-full h-full flex items-center justify-center text-4xl text-text-muted">
										{person.name.charAt(0)}
									</div>
								{/if}
							</div>
						</div>

						<!-- Info -->
						<div class="flex-1 min-w-0">
							<h2 class="text-2xl font-bold text-white mb-1">{person.name}</h2>
							{#if person.knownFor}
								<p class="text-sm text-text-muted mb-3">{person.knownFor}</p>
							{/if}

							<!-- Birth/Death info -->
							<div class="space-y-1 text-sm mb-4">
								{#if person.birthday}
									<p class="text-text-secondary">
										<span class="text-text-muted">Born:</span>
										{formatDate(person.birthday)}
										{#if !person.deathday}
											<span class="text-text-muted ml-1">
												(age {calculateAge(person.birthday, null)})
											</span>
										{/if}
									</p>
								{/if}
								{#if person.deathday}
									<p class="text-text-secondary">
										<span class="text-text-muted">Died:</span>
										{formatDate(person.deathday)}
										<span class="text-text-muted ml-1">
											(age {calculateAge(person.birthday, person.deathday)})
										</span>
									</p>
								{/if}
								{#if person.placeOfBirth}
									<p class="text-text-secondary">
										<span class="text-text-muted">From:</span>
										{person.placeOfBirth}
									</p>
								{/if}
							</div>

							<!-- Biography (truncated) -->
							{#if person.biography}
								<p class="text-sm text-text-secondary line-clamp-4">
									{person.biography}
								</p>
							{/if}
						</div>
					</div>

					<!-- Also in your library -->
					{#if person.alsoInLibrary?.length > 0}
						<div class="mt-6">
							<h3 class="text-sm font-medium text-text-muted uppercase tracking-wide mb-3 flex items-center gap-2">
								<svg class="w-4 h-4 text-green-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
								</svg>
								Also in Your Library
							</h3>
							<div class="flex gap-3 overflow-x-auto pb-2 scrollbar-thin">
								{#each person.alsoInLibrary as item}
									<button
										onclick={() => navigateToMedia(item)}
										class="flex-shrink-0 w-24 group text-left focus:outline-none"
									>
										<div class="aspect-[2/3] rounded-lg overflow-hidden bg-bg-elevated ring-2 ring-green-500/50 group-hover:ring-green-400 group-focus:ring-green-400 transition-all">
											{#if item.posterPath}
												<img
													src={getImageUrl(item.posterPath)}
													alt={item.title}
													class="w-full h-full object-cover"
												/>
											{:else}
												<div class="w-full h-full flex items-center justify-center text-text-muted">
													<svg class="w-8 h-8" fill="none" stroke="currentColor" viewBox="0 0 24 24">
														<path stroke-linecap="round" stroke-linejoin="round" stroke-width="1" d="M7 4v16M17 4v16M3 8h4m10 0h4M3 12h18M3 16h4m10 0h4" />
													</svg>
												</div>
											{/if}
										</div>
										<p class="mt-1.5 text-xs text-white truncate">{item.title}</p>
										<p class="text-[10px] text-text-muted">{item.year}</p>
									</button>
								{/each}
							</div>
						</div>
					{/if}

					<!-- Known for / Credits -->
					{#if person.credits?.length > 0}
						<div class="mt-6">
							<h3 class="text-sm font-medium text-text-muted uppercase tracking-wide mb-3">
								Known For
							</h3>
							<div class="flex gap-3 overflow-x-auto pb-2 scrollbar-thin">
								{#each person.credits.slice(0, 12) as credit}
									<a
										href="/discover/{credit.mediaType === 'movie' ? 'movie' : 'show'}/{credit.id}"
										onclick={() => onClose()}
										class="flex-shrink-0 w-24 group text-left focus:outline-none"
									>
										<div class="aspect-[2/3] rounded-lg overflow-hidden bg-bg-elevated group-hover:ring-2 group-focus:ring-2 ring-white/30 transition-all">
											{#if credit.posterPath}
												<img
													src={getTmdbImageUrl(credit.posterPath, 'w185')}
													alt={credit.title}
													class="w-full h-full object-cover"
												/>
											{:else}
												<div class="w-full h-full flex items-center justify-center text-text-muted">
													<svg class="w-8 h-8" fill="none" stroke="currentColor" viewBox="0 0 24 24">
														<path stroke-linecap="round" stroke-linejoin="round" stroke-width="1" d="M7 4v16M17 4v16M3 8h4m10 0h4M3 12h18M3 16h4m10 0h4" />
													</svg>
												</div>
											{/if}
										</div>
										<p class="mt-1.5 text-xs text-white truncate">{credit.title}</p>
										{#if credit.character}
											<p class="text-[10px] text-text-muted truncate">{credit.character}</p>
										{/if}
									</a>
								{/each}
							</div>
						</div>
					{/if}
				{/if}
			</div>
		</div>
	</div>
{/if}
