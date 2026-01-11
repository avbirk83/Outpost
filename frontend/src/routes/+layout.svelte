<script lang="ts">
	import '../app.css';
	import favicon from '$lib/assets/favicon.svg';
	import { page } from '$app/stores';
	import { goto } from '$app/navigation';
	import { onMount } from 'svelte';
	import { auth } from '$lib/stores/auth';
	import { profileStore } from '$lib/stores/profile';
	import { getSetupWizardStatus, type Profile } from '$lib/api';
	import Topbar from '$lib/components/layout/Topbar.svelte';
	import StatusBar from '$lib/components/layout/StatusBar.svelte';
	import Toast from '$lib/components/Toast.svelte';

	let { children } = $props();
	let user = $state<{ id: number; username: string; role: string } | null>(null);
	let activeProfile = $state<Profile | null>(null);
	let profilesLoading = $state(true);
	let initialized = $state(false);
	let setupChecked = $state(false);

	// Subscribe to auth store
	auth.subscribe((value) => {
		user = value;
	});

	// Subscribe to profile store
	profileStore.subscribe((state) => {
		activeProfile = state.activeProfile;
		profilesLoading = state.loading;
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
		profileStore.clear();
		await auth.logout();
		goto('/login');
	}

	// Check if current page is a public page (no auth required)
	const isPublicPage = $derived(
		$page.url.pathname === '/login' ||
			$page.url.pathname === '/setup' ||
			$page.url.pathname === '/profiles'
	);

	// Initialize profiles when user is logged in
	$effect(() => {
		if (!initialized || !user) return;
		profileStore.init();
	});

	$effect(() => {
		if (!initialized) return;
		if (!user && !isPublicPage) {
			goto('/login');
		}
	});

	// Profile gate - redirect to /profiles if no active profile selected
	// (except for public pages and the profiles page itself)
	$effect(() => {
		if (!initialized || !user || profilesLoading) return;
		const path = $page.url.pathname;

		// Skip profile gate for certain pages
		if (
			path === '/login' ||
			path === '/setup' ||
			path === '/profiles' ||
			path.startsWith('/watch/')
		) {
			return;
		}

		// If no active profile, redirect to profile selector
		if (!activeProfile) {
			goto('/profiles');
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

	// Check if current route is profiles page
	function isProfilesPage(): boolean {
		return $page.url.pathname === '/profiles';
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
	{#if isWatchPage() || isProfilesPage()}
		<!-- Fullscreen mode - no chrome -->
		{@render children()}
	{:else}
		<!-- Standard app layout -->
		<div class="min-h-screen bg-bg-primary text-text-primary overflow-x-hidden">
			<!-- Topbar -->
			<Topbar
				username={user.username}
				isAdmin={user.role === 'admin'}
				{activeProfile}
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
