<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { page } from '$app/stores';
	import { exchangeTraktCode } from '$lib/api';

	let status = $state<'processing' | 'success' | 'error'>('processing');
	let errorMessage = $state('');
	let username = $state('');

	onMount(async () => {
		// Get the authorization code from URL params
		const code = $page.url.searchParams.get('code');

		if (!code) {
			status = 'error';
			errorMessage = 'No authorization code received from Trakt';
			return;
		}

		try {
			// Exchange the code for tokens
			const redirectUri = `${window.location.origin}/settings/trakt-callback`;
			const result = await exchangeTraktCode(code, redirectUri);

			if (result.success) {
				status = 'success';
				username = result.username;

				// Redirect back to settings after a short delay
				setTimeout(() => {
					const returnUrl = sessionStorage.getItem('trakt_return_url') || '/settings';
					sessionStorage.removeItem('trakt_return_url');
					goto(returnUrl);
				}, 2000);
			} else {
				status = 'error';
				errorMessage = 'Failed to connect to Trakt';
			}
		} catch (e) {
			status = 'error';
			errorMessage = e instanceof Error ? e.message : 'An error occurred';
		}
	});
</script>

<svelte:head>
	<title>Trakt Connection | Outpost</title>
</svelte:head>

<main class="min-h-screen flex items-center justify-center bg-bg-base p-4">
	<div class="glass-card p-8 max-w-md w-full text-center">
		{#if status === 'processing'}
			<div class="animate-spin w-12 h-12 border-4 border-red-500 border-t-transparent rounded-full mx-auto mb-4"></div>
			<h1 class="text-xl font-semibold text-text-primary mb-2">Connecting to Trakt...</h1>
			<p class="text-text-secondary">Please wait while we complete the connection</p>
		{:else if status === 'success'}
			<div class="w-12 h-12 rounded-full bg-green-500/20 flex items-center justify-center mx-auto mb-4">
				<svg class="w-6 h-6 text-green-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
				</svg>
			</div>
			<h1 class="text-xl font-semibold text-text-primary mb-2">Connected!</h1>
			<p class="text-text-secondary mb-4">Welcome, <span class="text-text-primary font-medium">{username}</span></p>
			<p class="text-sm text-text-muted">Redirecting you back...</p>
		{:else}
			<div class="w-12 h-12 rounded-full bg-red-500/20 flex items-center justify-center mx-auto mb-4">
				<svg class="w-6 h-6 text-red-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
				</svg>
			</div>
			<h1 class="text-xl font-semibold text-text-primary mb-2">Connection Failed</h1>
			<p class="text-red-400 mb-4">{errorMessage}</p>
			<a href="/settings" class="liquid-btn inline-block">Back to Settings</a>
		{/if}
	</div>
</main>
