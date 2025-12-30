<script lang="ts">
	import { goto } from '$app/navigation';
	import { auth } from '$lib/stores/auth';
	import { checkSetup, setup } from '$lib/api';
	import { onMount } from 'svelte';

	let username = $state('');
	let password = $state('');
	let confirmPassword = $state('');
	let error: string | null = $state(null);
	let loading = $state(false);
	let setupRequired = $state(false);
	let checkingSetup = $state(true);

	onMount(async () => {
		try {
			const status = await checkSetup();
			setupRequired = status.setupRequired;
		} catch (e) {
			error = 'Failed to check setup status';
		} finally {
			checkingSetup = false;
		}
	});

	async function handleLogin() {
		if (!username || !password) {
			error = 'Please enter username and password';
			return;
		}

		try {
			loading = true;
			error = null;
			await auth.login(username, password);
			goto('/');
		} catch (e) {
			error = 'Invalid username or password';
		} finally {
			loading = false;
		}
	}

	async function handleSetup() {
		if (!username || !password) {
			error = 'Please enter username and password';
			return;
		}

		if (password !== confirmPassword) {
			error = 'Passwords do not match';
			return;
		}

		if (password.length < 4) {
			error = 'Password must be at least 4 characters';
			return;
		}

		try {
			loading = true;
			error = null;
			await setup(username, password);
			// Now log in with the new account
			await auth.login(username, password);
			goto('/');
		} catch (e) {
			error = e instanceof Error ? e.message : 'Setup failed';
		} finally {
			loading = false;
		}
	}
</script>

<svelte:head>
	<title>{setupRequired ? 'Setup' : 'Login'} - Outpost</title>
</svelte:head>

<div class="min-h-screen flex items-center justify-center bg-gray-900">
	<div class="bg-gray-800 p-8 rounded-lg shadow-lg w-full max-w-md">
		{#if checkingSetup}
			<p class="text-gray-400 text-center">Loading...</p>
		{:else if setupRequired}
			<h1 class="text-2xl font-bold text-center mb-6">Welcome to Outpost</h1>
			<p class="text-gray-400 text-center mb-6">Create your admin account to get started.</p>

			{#if error}
				<div class="bg-white/5 border border-white/10 text-text-secondary px-4 py-3 rounded mb-4">
					{error}
				</div>
			{/if}

			<form onsubmit={(e) => { e.preventDefault(); handleSetup(); }} class="space-y-4">
				<div>
					<label for="username" class="block text-sm text-gray-400 mb-1">Username</label>
					<input
						type="text"
						id="username"
						bind:value={username}
						class="w-full bg-gray-700 border border-gray-600 rounded px-3 py-2"
						placeholder="admin"
						autocomplete="username"
					/>
				</div>
				<div>
					<label for="password" class="block text-sm text-gray-400 mb-1">Password</label>
					<input
						type="password"
						id="password"
						bind:value={password}
						class="w-full bg-gray-700 border border-gray-600 rounded px-3 py-2"
						placeholder="Enter password"
						autocomplete="new-password"
					/>
				</div>
				<div>
					<label for="confirmPassword" class="block text-sm text-gray-400 mb-1">Confirm Password</label>
					<input
						type="password"
						id="confirmPassword"
						bind:value={confirmPassword}
						class="w-full bg-gray-700 border border-gray-600 rounded px-3 py-2"
						placeholder="Confirm password"
						autocomplete="new-password"
					/>
				</div>
				<button
					type="submit"
					class="w-full bg-blue-600 hover:bg-blue-700 py-2 rounded font-medium disabled:opacity-50"
					disabled={loading}
				>
					{loading ? 'Creating...' : 'Create Account'}
				</button>
			</form>
		{:else}
			<h1 class="text-2xl font-bold text-center mb-6">Login to Outpost</h1>

			{#if error}
				<div class="bg-white/5 border border-white/10 text-text-secondary px-4 py-3 rounded mb-4">
					{error}
				</div>
			{/if}

			<form onsubmit={(e) => { e.preventDefault(); handleLogin(); }} class="space-y-4">
				<div>
					<label for="username" class="block text-sm text-gray-400 mb-1">Username</label>
					<input
						type="text"
						id="username"
						bind:value={username}
						class="w-full bg-gray-700 border border-gray-600 rounded px-3 py-2"
						placeholder="Username"
						autocomplete="username"
					/>
				</div>
				<div>
					<label for="password" class="block text-sm text-gray-400 mb-1">Password</label>
					<input
						type="password"
						id="password"
						bind:value={password}
						class="w-full bg-gray-700 border border-gray-600 rounded px-3 py-2"
						placeholder="Password"
						autocomplete="current-password"
					/>
				</div>
				<button
					type="submit"
					class="w-full bg-blue-600 hover:bg-blue-700 py-2 rounded font-medium disabled:opacity-50"
					disabled={loading}
				>
					{loading ? 'Logging in...' : 'Login'}
				</button>
			</form>
		{/if}
	</div>
</div>
