<script lang="ts">
	import { goto } from '$app/navigation';
	import {
		createSmartPlaylist,
		previewSmartPlaylist,
		getImageUrl,
		stringifyRules,
		RULE_FIELDS,
		OPERATORS,
		type PlaylistRules,
		type RuleCondition,
		type SmartPlaylistItem
	} from '$lib/api';
	import MediaCard from '$lib/components/media/MediaCard.svelte';
	import Select from '$lib/components/ui/Select.svelte';

	// Form state
	let name = $state('');
	let description = $state('');
	let mediaType = $state<'both' | 'movies' | 'shows'>('both');
	let sortBy = $state<'added' | 'title' | 'year' | 'rating' | 'runtime'>('added');
	let sortOrder = $state<'asc' | 'desc'>('desc');
	let limitCount = $state<number | null>(null);
	let autoRefresh = $state(true);

	// Rules state
	let matchType = $state<'all' | 'any'>('all');
	let conditions = $state<RuleCondition[]>([]);

	// Preview state
	let previewItems: SmartPlaylistItem[] = $state([]);
	let loadingPreview = $state(false);
	let previewError: string | null = $state(null);

	// Form state
	let saving = $state(false);
	let error: string | null = $state(null);

	function addCondition() {
		conditions = [...conditions, { field: 'genre', operator: 'contains', value: '' }];
	}

	function removeCondition(index: number) {
		conditions = conditions.filter((_, i) => i !== index);
	}

	function updateCondition(index: number, field: keyof RuleCondition, value: string | number | boolean) {
		conditions = conditions.map((c, i) => {
			if (i === index) {
				const updated = { ...c, [field]: value };
				// Reset operator and value when field changes
				if (field === 'field') {
					const fieldDef = RULE_FIELDS.find(f => f.value === value);
					const operators = OPERATORS[fieldDef?.type || 'text'];
					updated.operator = operators[0].value;
					updated.value = fieldDef?.type === 'boolean' ? true : '';
				}
				return updated;
			}
			return c;
		});
	}

	function getFieldType(fieldValue: string): string {
		const field = RULE_FIELDS.find(f => f.value === fieldValue);
		return field?.type || 'text';
	}

	function getFieldOptions(fieldValue: string): string[] | undefined {
		const field = RULE_FIELDS.find(f => f.value === fieldValue);
		return field?.options as string[] | undefined;
	}

	async function handlePreview() {
		if (loadingPreview) return;
		loadingPreview = true;
		previewError = null;

		const rules: PlaylistRules = { match: matchType, conditions };

		try {
			const result = await previewSmartPlaylist({
				rules: stringifyRules(rules),
				sortBy,
				sortOrder,
				limitCount: limitCount || undefined,
				mediaType
			});
			previewItems = result.items;
		} catch (e) {
			previewError = e instanceof Error ? e.message : 'Failed to preview';
		} finally {
			loadingPreview = false;
		}
	}

	async function handleSubmit(e: Event) {
		e.preventDefault();
		if (saving || !name.trim()) return;

		saving = true;
		error = null;

		const rules: PlaylistRules = { match: matchType, conditions };

		try {
			const playlist = await createSmartPlaylist({
				name: name.trim(),
				description: description.trim() || undefined,
				rules: stringifyRules(rules),
				sortBy,
				sortOrder,
				limitCount: limitCount || undefined,
				mediaType,
				autoRefresh
			});
			goto(`/playlists/${playlist.id}`);
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to create playlist';
			saving = false;
		}
	}

	const sortOptions = [
		{ value: 'added', label: 'Date Added' },
		{ value: 'title', label: 'Title' },
		{ value: 'year', label: 'Year' },
		{ value: 'rating', label: 'Rating' },
		{ value: 'runtime', label: 'Runtime' }
	];

	const mediaTypeOptions = [
		{ value: 'both', label: 'Movies & TV Shows' },
		{ value: 'movies', label: 'Movies Only' },
		{ value: 'shows', label: 'TV Shows Only' }
	];
</script>

<svelte:head>
	<title>Create Smart Playlist - Outpost</title>
</svelte:head>

<div class="px-[60px] py-8 max-w-4xl">
	<div class="mb-8">
		<button onclick={() => goto('/playlists')} class="text-text-muted hover:text-text-primary transition-colors flex items-center gap-2 mb-4">
			<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
				<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 19l-7-7 7-7" />
			</svg>
			Back to Playlists
		</button>
		<h1 class="text-3xl font-bold text-text-primary">Create Smart Playlist</h1>
		<p class="text-text-muted mt-1">Define rules to automatically populate this playlist</p>
	</div>

	{#if error}
		<div class="bg-red-500/10 border border-red-500/30 text-red-400 px-4 py-3 rounded-xl mb-6 flex items-center justify-between">
			<span>{error}</span>
			<button onclick={() => error = null} class="text-red-400 hover:text-red-300">
				<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
				</svg>
			</button>
		</div>
	{/if}

	<form onsubmit={handleSubmit} class="space-y-8">
		<!-- Basic Info -->
		<section class="bg-bg-card border border-border-subtle rounded-xl p-6 space-y-4">
			<h2 class="text-lg font-semibold text-text-primary">Basic Information</h2>

			<div class="grid grid-cols-2 gap-4">
				<div class="col-span-2">
					<label for="name" class="block text-sm font-medium text-text-secondary mb-2">Name</label>
					<input
						id="name"
						type="text"
						bind:value={name}
						placeholder="My Smart Playlist"
						required
						class="w-full px-4 py-2.5 rounded-lg bg-bg-secondary border border-border-subtle text-text-primary placeholder-text-muted focus:outline-none focus:border-accent-primary"
					/>
				</div>
				<div class="col-span-2">
					<label for="description" class="block text-sm font-medium text-text-secondary mb-2">Description (optional)</label>
					<textarea
						id="description"
						bind:value={description}
						placeholder="Describe what this playlist contains..."
						rows="2"
						class="w-full px-4 py-2.5 rounded-lg bg-bg-secondary border border-border-subtle text-text-primary placeholder-text-muted focus:outline-none focus:border-accent-primary resize-none"
					></textarea>
				</div>
			</div>
		</section>

		<!-- Media Type & Sorting -->
		<section class="bg-bg-card border border-border-subtle rounded-xl p-6 space-y-4">
			<h2 class="text-lg font-semibold text-text-primary">Settings</h2>

			<div class="grid grid-cols-2 gap-4">
				<div>
					<label class="block text-sm font-medium text-text-secondary mb-2">Media Type</label>
					<Select bind:value={mediaType} options={mediaTypeOptions} class="w-full" />
				</div>
				<div>
					<label class="block text-sm font-medium text-text-secondary mb-2">Sort By</label>
					<div class="flex gap-2">
						<Select bind:value={sortBy} options={sortOptions} class="flex-1" />
						<button
							type="button"
							onclick={() => sortOrder = sortOrder === 'asc' ? 'desc' : 'asc'}
							class="px-3 py-2.5 rounded-lg bg-bg-secondary border border-border-subtle text-text-muted hover:text-text-primary transition-colors"
							title={sortOrder === 'asc' ? 'Ascending' : 'Descending'}
						>
							{#if sortOrder === 'asc'}
								<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 4h13M3 8h9m-9 4h6m4 0l4-4m0 0l4 4m-4-4v12" />
								</svg>
							{:else}
								<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 4h13M3 8h9m-9 4h9m5-4v12m0 0l-4-4m4 4l4-4" />
								</svg>
							{/if}
						</button>
					</div>
				</div>
				<div>
					<label class="block text-sm font-medium text-text-secondary mb-2">Limit (optional)</label>
					<input
						type="number"
						bind:value={limitCount}
						placeholder="No limit"
						min="1"
						max="1000"
						class="w-full px-4 py-2.5 rounded-lg bg-bg-secondary border border-border-subtle text-text-primary placeholder-text-muted focus:outline-none focus:border-accent-primary"
					/>
				</div>
				<div class="flex items-center">
					<label class="flex items-center gap-3 cursor-pointer">
						<input type="checkbox" bind:checked={autoRefresh} class="w-5 h-5 rounded border-border-subtle bg-bg-secondary text-accent-primary focus:ring-accent-primary" />
						<span class="text-text-primary">Auto-refresh</span>
					</label>
				</div>
			</div>
		</section>

		<!-- Rules -->
		<section class="bg-bg-card border border-border-subtle rounded-xl p-6 space-y-4">
			<div class="flex items-center justify-between">
				<h2 class="text-lg font-semibold text-text-primary">Rules</h2>
				<div class="flex items-center gap-2 text-sm">
					<span class="text-text-muted">Match</span>
					<select
						bind:value={matchType}
						class="px-3 py-1.5 rounded-lg bg-bg-secondary border border-border-subtle text-text-primary focus:outline-none focus:border-accent-primary"
					>
						<option value="all">All</option>
						<option value="any">Any</option>
					</select>
					<span class="text-text-muted">of the following conditions</span>
				</div>
			</div>

			{#if conditions.length === 0}
				<div class="text-center py-8 text-text-muted">
					<p class="mb-4">No rules defined. This playlist will match all media.</p>
					<button
						type="button"
						onclick={addCondition}
						class="px-4 py-2 rounded-lg bg-accent-primary text-white font-medium hover:bg-accent-primary/90 transition-colors inline-flex items-center gap-2"
					>
						<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" />
						</svg>
						Add Rule
					</button>
				</div>
			{:else}
				<div class="space-y-3">
					{#each conditions as condition, i}
						{@const fieldType = getFieldType(condition.field)}
						{@const fieldOptions = getFieldOptions(condition.field)}
						<div class="flex items-center gap-3 p-4 bg-bg-secondary rounded-lg">
							<!-- Field -->
							<select
								value={condition.field}
								onchange={(e) => updateCondition(i, 'field', e.currentTarget.value)}
								class="px-3 py-2 rounded-lg bg-bg-tertiary border border-border-subtle text-text-primary focus:outline-none focus:border-accent-primary min-w-[140px]"
							>
								{#each RULE_FIELDS as field}
									<option value={field.value}>{field.label}</option>
								{/each}
							</select>

							<!-- Operator -->
							<select
								value={condition.operator}
								onchange={(e) => updateCondition(i, 'operator', e.currentTarget.value)}
								class="px-3 py-2 rounded-lg bg-bg-tertiary border border-border-subtle text-text-primary focus:outline-none focus:border-accent-primary min-w-[160px]"
							>
								{#each OPERATORS[fieldType] || OPERATORS.text as op}
									<option value={op.value}>{op.label}</option>
								{/each}
							</select>

							<!-- Value -->
							{#if fieldType === 'boolean'}
								<select
									value={condition.value ? 'true' : 'false'}
									onchange={(e) => updateCondition(i, 'value', e.currentTarget.value === 'true')}
									class="px-3 py-2 rounded-lg bg-bg-tertiary border border-border-subtle text-text-primary focus:outline-none focus:border-accent-primary flex-1"
								>
									<option value="true">Yes</option>
									<option value="false">No</option>
								</select>
							{:else if fieldType === 'select' && fieldOptions}
								<select
									value={condition.value}
									onchange={(e) => updateCondition(i, 'value', e.currentTarget.value)}
									class="px-3 py-2 rounded-lg bg-bg-tertiary border border-border-subtle text-text-primary focus:outline-none focus:border-accent-primary flex-1"
								>
									<option value="">Select...</option>
									{#each fieldOptions as opt}
										<option value={opt}>{opt}</option>
									{/each}
								</select>
							{:else if fieldType === 'number'}
								<input
									type="number"
									value={condition.value}
									onchange={(e) => updateCondition(i, 'value', Number(e.currentTarget.value))}
									placeholder="Enter value..."
									class="px-3 py-2 rounded-lg bg-bg-tertiary border border-border-subtle text-text-primary placeholder-text-muted focus:outline-none focus:border-accent-primary flex-1"
								/>
							{:else if fieldType === 'duration'}
								<input
									type="text"
									value={condition.value}
									onchange={(e) => updateCondition(i, 'value', e.currentTarget.value)}
									placeholder="e.g., 7d, 30d, 1y"
									class="px-3 py-2 rounded-lg bg-bg-tertiary border border-border-subtle text-text-primary placeholder-text-muted focus:outline-none focus:border-accent-primary flex-1"
								/>
							{:else}
								<input
									type="text"
									value={condition.value}
									onchange={(e) => updateCondition(i, 'value', e.currentTarget.value)}
									placeholder="Enter value..."
									class="px-3 py-2 rounded-lg bg-bg-tertiary border border-border-subtle text-text-primary placeholder-text-muted focus:outline-none focus:border-accent-primary flex-1"
								/>
							{/if}

							<!-- Remove -->
							<button
								type="button"
								onclick={() => removeCondition(i)}
								class="p-2 rounded-lg text-text-muted hover:text-red-400 hover:bg-red-500/10 transition-colors"
								title="Remove rule"
							>
								<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
								</svg>
							</button>
						</div>
					{/each}
				</div>

				<button
					type="button"
					onclick={addCondition}
					class="w-full px-4 py-3 rounded-lg border border-dashed border-border-subtle text-text-muted hover:text-text-primary hover:border-border-hover transition-colors flex items-center justify-center gap-2"
				>
					<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" />
					</svg>
					Add Another Rule
				</button>
			{/if}
		</section>

		<!-- Preview -->
		<section class="bg-bg-card border border-border-subtle rounded-xl p-6 space-y-4">
			<div class="flex items-center justify-between">
				<h2 class="text-lg font-semibold text-text-primary">Preview</h2>
				<button
					type="button"
					onclick={handlePreview}
					disabled={loadingPreview}
					class="px-4 py-2 rounded-lg bg-bg-secondary border border-border-subtle text-text-secondary hover:text-text-primary hover:bg-bg-tertiary transition-colors flex items-center gap-2 disabled:opacity-50"
				>
					{#if loadingPreview}
						<div class="animate-spin w-4 h-4 border-2 border-current border-t-transparent rounded-full"></div>
						Loading...
					{:else}
						<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z" />
						</svg>
						Preview Results
					{/if}
				</button>
			</div>

			{#if previewError}
				<div class="text-red-400 text-sm">{previewError}</div>
			{/if}

			{#if previewItems.length > 0}
				<div class="grid grid-cols-[repeat(auto-fill,minmax(120px,1fr))] gap-3">
					{#each previewItems.slice(0, 12) as item}
						<div class="aspect-[2/3] rounded-lg overflow-hidden bg-bg-tertiary relative group">
							{#if item.posterPath}
								<img
									src={getImageUrl(item.posterPath)}
									alt={item.title}
									class="w-full h-full object-cover"
								/>
							{:else}
								<div class="w-full h-full flex items-center justify-center text-text-muted text-xs p-2 text-center">
									{item.title}
								</div>
							{/if}
							<div class="absolute inset-x-0 bottom-0 bg-gradient-to-t from-black/80 to-transparent p-2 opacity-0 group-hover:opacity-100 transition-opacity">
								<p class="text-white text-xs font-medium truncate">{item.title}</p>
								<p class="text-white/60 text-xs">{item.year}</p>
							</div>
						</div>
					{/each}
				</div>
				{#if previewItems.length > 12}
					<p class="text-text-muted text-sm text-center">
						+{previewItems.length - 12} more items
					</p>
				{/if}
			{:else if !loadingPreview}
				<p class="text-text-muted text-sm text-center py-8">
					Click "Preview Results" to see what this playlist will contain
				</p>
			{/if}
		</section>

		<!-- Actions -->
		<div class="flex gap-4">
			<button
				type="button"
				onclick={() => goto('/playlists')}
				class="flex-1 px-6 py-3 rounded-xl border border-border-subtle text-text-secondary hover:bg-bg-secondary transition-colors"
			>
				Cancel
			</button>
			<button
				type="submit"
				disabled={saving || !name.trim()}
				class="flex-1 px-6 py-3 rounded-xl bg-accent-primary text-white font-medium hover:bg-accent-primary/90 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
			>
				{saving ? 'Creating...' : 'Create Playlist'}
			</button>
		</div>
	</form>
</div>
