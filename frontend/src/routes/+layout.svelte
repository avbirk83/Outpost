<script lang="ts">
	import '../app.css';
	import favicon from '$lib/assets/favicon.svg';
	import { page } from '$app/stores';
	import { goto } from '$app/navigation';
	import { onMount } from 'svelte';
	import { auth } from '$lib/stores/auth';
	import Sidebar from '$lib/components/Sidebar.svelte';
	import TopBar from '$lib/components/TopBar.svelte';
	import SearchOverlay from '$lib/components/SearchOverlay.svelte';

	let { children } = $props();
	let user = $state<{ id: number; username: string; role: string } | null>(null);
	let initialized = $state(false);
	let searchOpen = $state(false);

	// Subscribe to auth store
	auth.subscribe((value) => {
		user = value;
	});

	onMount(async () => {
		await auth.init();
		initialized = true;

		// Global keyboard shortcut for search
		function handleKeydown(e: KeyboardEvent) {
			if ((e.ctrlKey || e.metaKey) && e.key === 'k') {
				e.preventDefault();
				searchOpen = true;
			}
		}

		window.addEventListener('keydown', handleKeydown);
		return () => window.removeEventListener('keydown', handleKeydown);
	});

	async function handleLogout() {
		await auth.logout();
		goto('/login');
	}

	function openSearch() {
		searchOpen = true;
	}

	function closeSearch() {
		searchOpen = false;
	}

	// Check if current page is login page
	$effect(() => {
		if (!initialized) return;
		const isLoginPage = $page.url.pathname === '/login';
		if (!user && !isLoginPage) {
			goto('/login');
		}
	});

	// Check if current route is a watch page (fullscreen player)
	function isWatchPage(): boolean {
		return $page.url.pathname.startsWith('/watch/');
	}
</script>

<svelte:head>
	<link rel="icon" href={favicon} />
	<title>Outpost</title>
</svelte:head>

{#if !initialized}
	<div class="min-h-screen bg-bg-primary text-text-primary flex items-center justify-center">
		<div class="flex items-center gap-3">
			<div class="w-6 h-6 border-2 border-white/50 border-t-transparent rounded-full animate-spin"></div>
			<p class="text-text-secondary">Loading...</p>
		</div>
	</div>
{:else if $page.url.pathname === '/login'}
	{@render children()}
{:else if user}
	{#if isWatchPage()}
		<!-- Fullscreen player mode - no chrome -->
		{@render children()}
	{:else}
		<!-- Standard app layout -->
		<div class="min-h-screen bg-bg-primary text-text-primary">
			<!-- Sidebar -->
			<Sidebar isAdmin={user.role === 'admin'} />

			<!-- TopBar -->
			<TopBar
				username={user.username}
				isAdmin={user.role === 'admin'}
				onLogout={handleLogout}
				onSearchClick={openSearch}
			/>

			<!-- Main content area -->
			<main class="pl-16 pt-16">
				<div class="p-6">
					{@render children()}
				</div>
			</main>

			<!-- Search overlay -->
			<SearchOverlay open={searchOpen} onClose={closeSearch} />
		</div>
	{/if}
{:else}
	<div class="min-h-screen bg-bg-primary text-text-primary flex items-center justify-center">
		<div class="flex items-center gap-3">
			<div class="w-6 h-6 border-2 border-white/50 border-t-transparent rounded-full animate-spin"></div>
			<p class="text-text-secondary">Redirecting...</p>
		</div>
	</div>
{/if}
