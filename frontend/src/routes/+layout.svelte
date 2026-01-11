<script lang="ts">
	import '../app.css';
	import favicon from '$lib/assets/favicon.svg';
	import { page } from '$app/stores';
	import { goto } from '$app/navigation';
	import { onMount } from 'svelte';
	import { auth } from '$lib/stores/auth';
	import { getSetupWizardStatus } from '$lib/api';
	import Topbar from '$lib/components/layout/Topbar.svelte';
	import StatusBar from '$lib/components/layout/StatusBar.svelte';
	import Toast from '$lib/components/Toast.svelte';

	let { children } = $props();
	let user = $state<{ id: number; username: string; role: string } | null>(null);
	let initialized = $state(false);
	let setupChecked = $state(false);

	// Subscribe to auth store
	auth.subscribe((value) => {
		user = value;
	});

	onMount(async () => {
		await auth.init();
		initialized = true;
	});

	// Check setup status for admins after initialization
	$effect(() => {
		if (!initialized || setupChecked) return;
		if (user?.role === 'admin' && $page.url.pathname !== '/setup') {
			checkSetupStatus();
		} else {
			setupChecked = true;
		}
	});

	async function checkSetupStatus() {
		try {
			const status = await getSetupWizardStatus();
			// Redirect to setup if needed and not already on setup page
			if (status.needsSetup && !status.setupCompleted) {
				goto('/setup');
				return;
			}
		} catch {
			// Ignore errors - setup check is optional
		}
		setupChecked = true;
	}

	async function handleLogout() {
		await auth.logout();
		goto('/login');
	}

	// Check if current page is a public page (no auth required)
	const isPublicPage = $derived($page.url.pathname === '/login' || $page.url.pathname === '/setup');

	$effect(() => {
		if (!initialized) return;
		if (!user && !isPublicPage) {
			goto('/login');
		}
	});

	// Check if current route is a watch page (fullscreen player)
	function isWatchPage(): boolean {
		return $page.url.pathname.startsWith('/watch/');
	}

	// Check if current route is setup page (fullscreen wizard)
	function isSetupPage(): boolean {
		return $page.url.pathname === '/setup';
	}
</script>

<svelte:head>
	<link rel="icon" href={favicon} />
	<title>Outpost</title>
</svelte:head>

{#if !initialized}
	<div class="min-h-screen bg-bg-primary text-text-primary flex items-center justify-center">
		<div class="flex items-center gap-3">
			<div class="spinner-lg text-cream"></div>
			<p class="text-text-secondary">Loading...</p>
		</div>
	</div>
{:else if $page.url.pathname === '/login' || $page.url.pathname === '/setup'}
	{@render children()}
{:else if user}
	{#if isWatchPage()}
		<!-- Fullscreen player mode - no chrome -->
		{@render children()}
	{:else}
		<!-- Standard app layout -->
		<div class="min-h-screen bg-bg-primary text-text-primary overflow-x-hidden">
			<!-- Topbar -->
			<Topbar
				username={user.username}
				isAdmin={user.role === 'admin'}
				onLogout={handleLogout}
			/>

			<!-- Main content area -->
			<main class="pt-16 pb-8">
				{@render children()}
			</main>

			<!-- Status Bar -->
			<StatusBar isAdmin={user.role === 'admin'} />
		</div>
	{/if}

	<!-- Toast notifications (always visible when logged in) -->
	<Toast />
{:else}
	<div class="min-h-screen bg-bg-primary text-text-primary flex items-center justify-center">
		<div class="flex items-center gap-3">
			<div class="spinner-lg text-cream"></div>
			<p class="text-text-secondary">Redirecting...</p>
		</div>
	</div>
{/if}
