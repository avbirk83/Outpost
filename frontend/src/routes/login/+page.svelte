<script lang="ts">
	import { goto } from '$app/navigation';
	import { auth } from '$lib/stores/auth';
	import { checkSetup, setup, getTmdbImageUrl } from '$lib/api';
	import { onMount } from 'svelte';

	let username = $state('');
	let password = $state('');
	let confirmPassword = $state('');
	let error: string | null = $state(null);
	let loading = $state(false);
	let setupRequired = $state(false);
	let checkingSetup = $state(true);
	let posters = $state<string[]>([]);

	onMount(async () => {
		// Check setup status
		try {
			const status = await checkSetup();
			setupRequired = status.setupRequired;
		} catch (e) {
			error = 'Failed to check setup status';
		} finally {
			checkingSetup = false;
		}

		// Load trending posters for background (public endpoint, no auth needed)
		try {
			const response = await fetch('/api/public/trending-posters');
			if (response.ok) {
				const data = await response.json();
				const shuffled = data.posters
					.sort(() => Math.random() - 0.5)
					.slice(0, 30);
				posters = shuffled.map((path: string) => getTmdbImageUrl(path, 'w342'));
			}
		} catch (e) {
			console.error('Failed to load posters:', e);
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
	<title>{setupRequired ? 'Setup' : 'Sign In'} - Outpost</title>
</svelte:head>

<div class="login-page">
	<!-- Poster Grid Background -->
	<div class="poster-bg">
		<div class="poster-grid">
			{#each posters as poster, i}
				<div class="poster-item" style="animation-delay: {i * 0.1}s">
					<img src={poster} alt="" loading="lazy" />
				</div>
			{/each}
		</div>
		<div class="poster-overlay"></div>
	</div>

	<!-- Content -->
	<div class="login-wrapper">
		<!-- Logo -->
		<div class="logo-section">
			<img src="/outpost-banner.png" alt="Outpost" class="logo-banner" />
		</div>

		<!-- Login Card -->
		<div class="login-card">
			{#if checkingSetup}
				<div class="loading-state">
					<div class="spinner"></div>
				</div>
			{:else if setupRequired}
				<h2 class="card-title">Create Admin Account</h2>
				<p class="card-subtitle">Set up your first account to get started</p>

				{#if error}
					<div class="error-box">{error}</div>
				{/if}

				<form onsubmit={(e) => { e.preventDefault(); handleSetup(); }}>
					<div class="field">
						<label for="username">Username</label>
						<input
							type="text"
							id="username"
							bind:value={username}
							placeholder="Enter username"
							autocomplete="username"
						/>
					</div>
					<div class="field">
						<label for="password">Password</label>
						<input
							type="password"
							id="password"
							bind:value={password}
							placeholder="Enter password"
							autocomplete="new-password"
						/>
					</div>
					<div class="field">
						<label for="confirmPassword">Confirm Password</label>
						<input
							type="password"
							id="confirmPassword"
							bind:value={confirmPassword}
							placeholder="Confirm password"
							autocomplete="new-password"
						/>
					</div>
					<button type="submit" class="submit-btn" disabled={loading}>
						{#if loading}
							<span class="btn-spinner"></span>
						{/if}
						{loading ? 'Creating...' : 'Create Account'}
					</button>
				</form>
			{:else}
				<h2 class="card-title">Welcome Back</h2>
				<p class="card-subtitle">Sign in to your media server</p>

				{#if error}
					<div class="error-box">{error}</div>
				{/if}

				<form onsubmit={(e) => { e.preventDefault(); handleLogin(); }}>
					<div class="field">
						<label for="username">Username</label>
						<input
							type="text"
							id="username"
							bind:value={username}
							placeholder="Enter username"
							autocomplete="username"
						/>
					</div>
					<div class="field">
						<label for="password">Password</label>
						<input
							type="password"
							id="password"
							bind:value={password}
							placeholder="Enter password"
							autocomplete="current-password"
						/>
					</div>
					<button type="submit" class="submit-btn" disabled={loading}>
						{#if loading}
							<span class="btn-spinner"></span>
						{/if}
						{loading ? 'Signing in...' : 'Sign In'}
					</button>
				</form>
			{/if}
		</div>
	</div>
</div>

<style>
	.login-page {
		min-height: 100vh;
		width: 100%;
		display: flex;
		align-items: center;
		justify-content: center;
		position: relative;
		background: #0a0a0a;
		overflow: hidden;
	}

	/* Poster Grid Background */
	.poster-bg {
		position: fixed;
		inset: -50px;
		z-index: 0;
		overflow: hidden;
	}

	.poster-grid {
		display: grid;
		grid-template-columns: repeat(6, 1fr);
		gap: 8px;
		animation: scroll-up 60s linear infinite;
		padding: 8px;
	}

	.poster-item {
		aspect-ratio: 2/3;
		border-radius: 8px;
		overflow: hidden;
		opacity: 0;
		animation: fade-in 0.5s ease forwards;
	}

	.poster-item img {
		width: 100%;
		height: 100%;
		object-fit: cover;
	}

	.poster-overlay {
		position: absolute;
		inset: 0;
		background: radial-gradient(
			ellipse at center,
			rgba(10, 10, 10, 0.7) 0%,
			rgba(10, 10, 10, 0.85) 50%,
			rgba(10, 10, 10, 0.95) 100%
		);
	}

	@keyframes scroll-up {
		0% {
			transform: translateY(0);
		}
		100% {
			transform: translateY(-50%);
		}
	}

	@keyframes fade-in {
		to {
			opacity: 0.6;
		}
	}

	/* Content */
	.login-wrapper {
		position: relative;
		z-index: 10;
		display: flex;
		flex-direction: column;
		align-items: center;
		padding: 2rem;
		width: 100%;
		max-width: 400px;
	}

	/* Logo */
	.logo-section {
		margin-bottom: 2.5rem;
	}

	.logo-banner {
		height: 56px;
		width: auto;
		filter: drop-shadow(0 4px 12px rgba(0, 0, 0, 0.5));
	}

	/* Card */
	.login-card {
		width: 100%;
		background: rgba(17, 17, 17, 0.85);
		backdrop-filter: blur(20px);
		border: 1px solid rgba(245, 230, 200, 0.1);
		border-radius: 20px;
		padding: 2rem;
	}

	.loading-state {
		display: flex;
		justify-content: center;
		padding: 3rem;
	}

	.spinner {
		width: 32px;
		height: 32px;
		border: 2px solid rgba(245, 230, 200, 0.2);
		border-top-color: #F5E6C8;
		border-radius: 50%;
		animation: spin 0.8s linear infinite;
	}

	.card-title {
		font-size: 1.5rem;
		font-weight: 600;
		color: #F5E6C8;
		text-align: center;
		margin-bottom: 0.5rem;
	}

	.card-subtitle {
		font-size: 0.875rem;
		color: rgba(245, 230, 200, 0.5);
		text-align: center;
		margin-bottom: 1.5rem;
	}

	.error-box {
		background: rgba(239, 68, 68, 0.15);
		border: 1px solid rgba(239, 68, 68, 0.3);
		color: #fca5a5;
		padding: 0.75rem 1rem;
		border-radius: 10px;
		font-size: 0.875rem;
		margin-bottom: 1.25rem;
		text-align: center;
	}

	form {
		display: flex;
		flex-direction: column;
		gap: 1.25rem;
	}

	.field {
		display: flex;
		flex-direction: column;
		gap: 0.5rem;
	}

	.field label {
		font-size: 0.8125rem;
		font-weight: 500;
		color: rgba(245, 230, 200, 0.7);
	}

	.field input {
		width: 100%;
		padding: 0.875rem 1rem;
		background: rgba(255, 255, 255, 0.05);
		border: 1px solid rgba(245, 230, 200, 0.1);
		border-radius: 12px;
		color: #F5E6C8;
		font-size: 1rem;
		transition: all 0.2s ease;
	}

	.field input::placeholder {
		color: rgba(245, 230, 200, 0.3);
	}

	.field input:focus {
		outline: none;
		border-color: rgba(245, 230, 200, 0.3);
		background: rgba(255, 255, 255, 0.08);
	}

	.submit-btn {
		display: flex;
		align-items: center;
		justify-content: center;
		gap: 0.5rem;
		width: 100%;
		padding: 0.875rem 1.5rem;
		background: #E8A849;
		color: #000;
		border: none;
		border-radius: 12px;
		font-size: 1rem;
		font-weight: 600;
		cursor: pointer;
		transition: all 0.2s ease;
		margin-top: 0.5rem;
	}

	.submit-btn:hover:not(:disabled) {
		background: #F0C06A;
		transform: translateY(-1px);
	}

	.submit-btn:active:not(:disabled) {
		transform: translateY(0);
	}

	.submit-btn:disabled {
		opacity: 0.7;
		cursor: not-allowed;
	}

	.btn-spinner {
		width: 18px;
		height: 18px;
		border: 2px solid rgba(0, 0, 0, 0.2);
		border-top-color: #000;
		border-radius: 50%;
		animation: spin 0.8s linear infinite;
	}

	@keyframes spin {
		to { transform: rotate(360deg); }
	}

	@media (max-width: 480px) {
		.login-wrapper {
			padding: 1.5rem;
		}

		.login-card {
			padding: 1.5rem;
			border-radius: 16px;
		}

		.logo-banner {
			height: 48px;
		}
	}
</style>
