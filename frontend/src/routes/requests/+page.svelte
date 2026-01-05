<script lang="ts">
	import { onMount } from 'svelte';
	import { getRequests, updateRequest, deleteRequest, getTmdbImageUrl, type Request } from '$lib/api';
	import { auth } from '$lib/stores/auth';
	import { toast } from '$lib/stores/toast';
	import TypeBadge from '$lib/components/TypeBadge.svelte';
	import Select from '$lib/components/ui/Select.svelte';

	let requests: Request[] = $state([]);
	let loading = $state(true);
	let error: string | null = $state(null);
	let statusFilter = $state('');
	let processingIds: Set<number> = $state(new Set());
	let confirmingDeleteId: number | null = $state(null);
	let user = $state<{ role: string } | null>(null);

	auth.subscribe((value) => {
		user = value;
	});

	onMount(async () => {
		await loadRequests();
	});

	async function loadRequests() {
		try {
			loading = true;
			requests = await getRequests(statusFilter || undefined);
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to load requests';
		} finally {
			loading = false;
		}
	}

	async function handleApprove(id: number) {
		const request = requests.find(r => r.id === id);
		processingIds.add(id);
		processingIds = processingIds;
		try {
			await updateRequest(id, 'approved');
			await loadRequests();
			toast.success(`Approved! Searching for ${request?.title || 'item'}...`);
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to approve request';
			toast.error('Failed to approve request');
		} finally {
			processingIds.delete(id);
			processingIds = processingIds;
		}
	}

	async function handleDeny(id: number) {
		const reason = prompt('Enter reason for denial (optional):');
		processingIds.add(id);
		processingIds = processingIds;
		try {
			await updateRequest(id, 'denied', reason || undefined);
			await loadRequests();
			toast.info('Request denied');
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to deny request';
			toast.error('Failed to deny request');
		} finally {
			processingIds.delete(id);
			processingIds = processingIds;
		}
	}

	function handleDeleteClick(id: number) {
		confirmingDeleteId = id;
	}

	function cancelDelete() {
		confirmingDeleteId = null;
	}

	async function confirmDelete(id: number) {
		processingIds.add(id);
		processingIds = processingIds;
		confirmingDeleteId = null;
		try {
			await deleteRequest(id);
			await loadRequests();
			toast.success('Request deleted');
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to delete request';
			toast.error('Failed to delete request');
		} finally {
			processingIds.delete(id);
			processingIds = processingIds;
		}
	}

	function formatDate(dateStr: string): string {
		try {
			const date = new Date(dateStr);
			return date.toLocaleDateString() + ' ' + date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
		} catch {
			return dateStr;
		}
	}

	function getStatusColor(status: string): string {
		switch (status) {
			case 'requested': return 'bg-white-600/20 text-white-400';
			case 'approved': return 'bg-green-600/20 text-green-400';
			case 'denied': return 'bg-white/10 text-text-secondary';
			case 'available': return 'bg-white-600/20 text-white-400';
			default: return 'bg-bg-elevated text-text-muted';
		}
	}

	$effect(() => {
		if (statusFilter !== undefined) {
			loadRequests();
		}
	});

</script>

<svelte:head>
	<title>Requests - Outpost</title>
</svelte:head>

<div class="space-y-6">
	<div class="flex items-center justify-between">
		<h1 class="text-2xl font-bold text-text-primary">Requests</h1>

		{#if user?.role === 'admin'}
			<Select
				bind:value={statusFilter}
				options={[
					{ value: '', label: 'All Requests' },
					{ value: 'requested', label: 'Pending' },
					{ value: 'approved', label: 'Approved' },
					{ value: 'denied', label: 'Denied' },
					{ value: 'available', label: 'Available' }
				]}
				class="w-40"
			/>
		{/if}
	</div>

	{#if error}
		<div class="bg-white/5 border border-white/10 text-text-secondary px-4 py-3 rounded-xl flex items-center justify-between">
			<span>{error}</span>
			<button class="text-text-muted hover:text-text-secondary" onclick={() => (error = null)}>
				<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
				</svg>
			</button>
		</div>
	{/if}

	{#if loading}
		<div class="flex items-center justify-center py-12">
			<div class="flex items-center gap-3">
				<div class="w-6 h-6 border-2 border-white-400 border-t-transparent rounded-full animate-spin"></div>
				<p class="text-text-secondary">Loading requests...</p>
			</div>
		</div>
	{:else if requests.length === 0}
		<div class="glass-card p-12 text-center">
			<div class="w-16 h-16 mx-auto mb-4 rounded-full bg-bg-elevated flex items-center justify-center">
				<svg class="w-8 h-8 text-text-muted" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M8.228 9c.549-1.165 2.03-2 3.772-2 2.21 0 4 1.343 4 3 0 1.4-1.278 2.575-3.006 2.907-.542.104-.994.54-.994 1.093m0 3h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
				</svg>
			</div>
			<h2 class="text-xl font-semibold text-text-primary mb-2">No requests found</h2>
			<p class="text-text-secondary">
				{user?.role === 'admin' ? 'Requests from users will appear here.' : 'Request content from the Discover page.'}
			</p>
		</div>
	{:else}
		<div class="space-y-3">
			{#each requests as request}
				<div class="glass-card overflow-hidden">
					<div class="flex items-start gap-4 p-4">
						{#if request.posterPath}
							<img
								src={getTmdbImageUrl(request.posterPath, 'w92')}
								alt={request.title}
								class="w-16 h-24 object-cover rounded-lg flex-shrink-0"
							/>
						{:else}
							<div class="w-16 h-24 bg-bg-elevated rounded-lg flex items-center justify-center flex-shrink-0">
								<svg class="w-6 h-6 text-text-muted" fill="none" stroke="currentColor" viewBox="0 0 24 24">
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z" />
								</svg>
							</div>
						{/if}

						<div class="flex-1 min-w-0">
							<div class="flex items-start justify-between gap-4">
								<div>
									<h3 class="font-medium text-lg text-text-primary">
										{request.title}
										{#if request.year}
											<span class="text-text-secondary">({request.year})</span>
										{/if}
									</h3>
									<div class="flex items-center gap-2 mt-1.5 text-sm flex-wrap">
										<TypeBadge type={request.type} />
										<span class="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-medium {request.status === 'approved' ? 'bg-green-600 text-white' : request.status === 'denied' ? 'bg-white/10 text-text-secondary' : 'bg-amber-500 text-black'}">
											{#if request.status === 'approved'}
												<svg class="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
													<path stroke-linecap="round" stroke-linejoin="round" stroke-width="3" d="M5 13l4 4L19 7" />
												</svg>
											{:else if request.status === 'requested'}
												<svg class="w-3 h-3" fill="currentColor" viewBox="0 0 24 24">
													<path d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm0 18c-4.41 0-8-3.59-8-8s3.59-8 8-8 8 3.59 8 8-3.59 8-8 8zm.5-13H11v6l5.25 3.15.75-1.23-4.5-2.67V7z" />
												</svg>
											{/if}
											{request.status.charAt(0).toUpperCase() + request.status.slice(1)}
										</span>
									</div>
									{#if user?.role === 'admin' && request.username}
										<p class="text-xs text-text-muted mt-1.5">
											Requested by {request.username}
										</p>
									{/if}
									<p class="text-xs text-text-muted mt-1">
										{formatDate(request.requestedAt)}
									</p>
									{#if request.statusReason}
										<p class="text-sm text-text-secondary mt-2">
											Reason: {request.statusReason}
										</p>
									{/if}
								</div>

								<div class="flex items-center gap-2 flex-shrink-0">
									{#if user?.role === 'admin' && request.status === 'requested'}
										<button
											onclick={() => handleApprove(request.id)}
											disabled={processingIds.has(request.id)}
											class="px-3 py-1.5 rounded-lg text-sm font-medium bg-green-600 text-white hover:bg-green-500 disabled:opacity-50 transition-colors"
										>
											{processingIds.has(request.id) ? '...' : 'Approve'}
										</button>
										<button
											onclick={() => handleDeny(request.id)}
											disabled={processingIds.has(request.id)}
											class="px-3 py-1.5 rounded-lg text-sm font-medium bg-white/10 text-text-secondary hover:bg-white/20 hover:text-white disabled:opacity-50 transition-colors"
										>
											{processingIds.has(request.id) ? '...' : 'Deny'}
										</button>
									{/if}
									{#if user?.role === 'admin' || (request.status === 'requested')}
										{#if confirmingDeleteId === request.id}
											<button
												onclick={() => confirmDelete(request.id)}
												disabled={processingIds.has(request.id)}
												class="px-3 py-1.5 rounded-lg text-sm font-medium bg-red-600 text-white hover:bg-red-500 disabled:opacity-50 transition-colors"
											>
												Confirm
											</button>
											<button
												onclick={cancelDelete}
												class="px-3 py-1.5 rounded-lg text-sm font-medium bg-white/10 text-text-secondary hover:bg-white/20 hover:text-white transition-colors"
											>
												Cancel
											</button>
										{:else}
											<button
												onclick={() => handleDeleteClick(request.id)}
												disabled={processingIds.has(request.id)}
												class="px-3 py-1.5 rounded-lg text-sm font-medium bg-white/10 text-text-secondary hover:bg-white/20 hover:text-white disabled:opacity-50 transition-colors"
											>
												Delete
											</button>
										{/if}
									{/if}
								</div>
							</div>

							{#if request.overview}
								<p class="text-sm text-text-secondary line-clamp-2 mt-3">{request.overview}</p>
							{/if}
						</div>
					</div>
				</div>
			{/each}
		</div>
	{/if}
</div>
