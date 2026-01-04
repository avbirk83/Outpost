<script lang="ts">
	import '../app.css';
	import favicon from '$lib/assets/favicon.svg';
	import { page } from '$app/stores';
	import { goto } from '$app/navigation';
	import { onMount } from 'svelte';
	import { auth } from '$lib/stores/auth';
	import Topbar from '$lib/components/layout/Topbar.svelte';
	import Toast from '$lib/components/Toast.svelte';

	let { children } = $props();
	let user = $state<{ id: number; username: string; role: string } | null>(null);
	let initialized = $state(false);

	// Subscribe to auth store
	auth.subscribe((value) => {
		user = value;
	});

	onMount(async () => {
		await auth.init();
		initialized = true;
	});

	async function handleLogout() {
		await auth.logout();
		goto('/login');
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
	<div class="min-h-screen bg-[#0a0a0a] text-text-primary flex items-center justify-center">
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
		<div class="min-h-screen bg-[#0a0a0a] text-text-primary overflow-x-hidden">
			<!-- Topbar -->
			<Topbar
				username={user.username}
				isAdmin={user.role === 'admin'}
				onLogout={handleLogout}
			/>

			<!-- Main content area -->
			<main class="pt-16">
				{@render children()}
			</main>
		</div>
	{/if}

	<!-- Toast notifications (always visible when logged in) -->
	<Toast />
{:else}
	<div class="min-h-screen bg-[#0a0a0a] text-text-primary flex items-center justify-center">
		<div class="flex items-center gap-3">
			<div class="w-6 h-6 border-2 border-white/50 border-t-transparent rounded-full animate-spin"></div>
			<p class="text-text-secondary">Redirecting...</p>
		</div>
	</div>
{/if}
