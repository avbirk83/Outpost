<script lang="ts">
	import { onMount } from 'svelte';
	import { getUsers, createUser, updateUser, deleteUser, type User, type ContentRating } from '$lib/api';
	import Select from '$lib/components/ui/Select.svelte';
	import { toast } from '$lib/stores/toast';

	let users: User[] = $state([]);
	let loading = $state(true);
	let error: string | null = $state(null);
	let showAddForm = $state(false);
	let editingUser: User | null = $state(null);

	// Form state
	let username = $state('');
	let password = $state('');
	let role = $state('user');

	// Parental controls state
	let contentRatingLimit = $state<ContentRating | null>(null);
	let requirePin = $state(false);
	let pin = $state('');
	let clearPin = $state(false);

	const contentRatingOptions = [
		{ value: '', label: 'No Limit' },
		{ value: 'G', label: 'G - General Audiences' },
		{ value: 'PG', label: 'PG - Parental Guidance' },
		{ value: 'PG-13', label: 'PG-13 - Parents Strongly Cautioned' },
		{ value: 'R', label: 'R - Restricted' },
		{ value: 'NC-17', label: 'NC-17 - Adults Only' }
	];

	onMount(async () => {
		await loadUsers();
	});

	async function loadUsers() {
		try {
			loading = true;
			users = await getUsers();
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to load users';
		} finally {
			loading = false;
		}
	}

	async function handleAddUser() {
		if (!username || !password) {
			error = 'Username and password are required';
			return;
		}

		if (requirePin && (!pin || pin.length !== 4)) {
			error = 'PIN must be 4 digits';
			return;
		}

		try {
			await createUser({
				username,
				password,
				role,
				contentRatingLimit: contentRatingLimit || null,
				requirePin,
				pin: requirePin ? pin : undefined
			});
			resetForm();
			showAddForm = false;
			await loadUsers();
			toast.success('User created');
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to create user';
			toast.error('Failed to create user');
		}
	}

	async function handleUpdateUser() {
		if (!editingUser) return;

		if (requirePin && pin && pin.length !== 4) {
			error = 'PIN must be 4 digits';
			return;
		}

		try {
			await updateUser(editingUser.id, {
				username,
				password: password || undefined,
				role,
				contentRatingLimit: contentRatingLimit || null,
				requirePin,
				pin: pin || undefined,
				clearPin
			});
			editingUser = null;
			resetForm();
			await loadUsers();
			toast.success('User updated');
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to update user';
			toast.error('Failed to update user');
		}
	}

	function resetForm() {
		username = '';
		password = '';
		role = 'user';
		contentRatingLimit = null;
		requirePin = false;
		pin = '';
		clearPin = false;
	}

	async function handleDeleteUser(id: number) {
		if (!confirm('Are you sure you want to delete this user?')) return;

		try {
			await deleteUser(id);
			await loadUsers();
			toast.success('User deleted');
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to delete user';
			toast.error('Failed to delete user');
		}
	}

	function startEdit(user: User) {
		editingUser = user;
		username = user.username;
		password = '';
		role = user.role;
		contentRatingLimit = user.contentRatingLimit || null;
		requirePin = user.requirePin || false;
		pin = '';
		clearPin = false;
		showAddForm = false;
	}

	function cancelEdit() {
		editingUser = null;
		resetForm();
	}

	function cancelAdd() {
		showAddForm = false;
		resetForm();
	}

	function getRoleBadgeColor(userRole: string): string {
		switch (userRole) {
			case 'admin':
				return 'liquid-badge !bg-white-500/20 !border-t-white-400/40 text-white-400';
			case 'kid':
				return 'liquid-badge !bg-green-500/20 !border-t-green-400/40 text-green-400';
			default:
				return 'liquid-badge !bg-blue-500/20 !border-t-blue-400/40 text-blue-400';
		}
	}

	function getRoleIcon(userRole: string): string {
		switch (userRole) {
			case 'admin':
				return 'M9 12l2 2 4-4m5.618-4.016A11.955 11.955 0 0112 2.944a11.955 11.955 0 01-8.618 3.04A12.02 12.02 0 003 9c0 5.591 3.824 10.29 9 11.622 5.176-1.332 9-6.03 9-11.622 0-1.042-.133-2.052-.382-3.016z';
			case 'kid':
				return 'M14.828 14.828a4 4 0 01-5.656 0M9 10h.01M15 10h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z';
			default:
				return 'M16 7a4 4 0 11-8 0 4 4 0 018 0zM12 14a7 7 0 00-7 7h14a7 7 0 00-7-7z';
		}
	}

	const inputClass = "liquid-input w-full px-4 py-2.5";
	const selectClass = "liquid-select w-full px-4 py-2.5";
	const labelClass = "block text-sm text-text-secondary mb-1.5";
</script>

<svelte:head>
	<title>User Management - Outpost</title>
</svelte:head>

<div class="space-y-6">
	<div class="flex items-center justify-between">
		<h1 class="text-2xl font-bold text-text-primary">Users</h1>
		{#if !showAddForm && !editingUser}
			<div class="inline-flex items-center p-1.5 rounded-xl bg-black/40 backdrop-blur-md border border-white/10">
				<button
					class="flex items-center gap-2 px-3 py-1.5 text-sm rounded-lg text-text-muted hover:text-text-primary hover:bg-glass transition-colors"
					onclick={() => { showAddForm = true; editingUser = null; }}
				>
					<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 6v6m0 0v6m0-6h6m-6 0H6" />
					</svg>
					Add User
				</button>
			</div>
		{/if}
	</div>

	{#if error}
		<div class="bg-glass border border-border-subtle text-text-secondary px-4 py-3 rounded-xl flex items-center justify-between">
			<span>{error}</span>
			<button class="text-text-muted hover:text-text-secondary" onclick={() => (error = null)}>
				<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
				</svg>
			</button>
		</div>
	{/if}

	{#if showAddForm}
		<form
			class="glass-card p-6 space-y-4"
			onsubmit={(e) => { e.preventDefault(); handleAddUser(); }}
		>
			<div class="flex items-center gap-3 mb-2">
				<div class="w-10 h-10 rounded-xl bg-white-600/20 flex items-center justify-center">
					<svg class="w-5 h-5 text-white-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M18 9v3m0 0v3m0-3h3m-3 0h-3m-2-5a4 4 0 11-8 0 4 4 0 018 0zM3 20a6 6 0 0112 0v1H3v-1z" />
					</svg>
				</div>
				<h2 class="text-lg font-semibold text-text-primary">Add New User</h2>
			</div>
			<div class="grid sm:grid-cols-3 gap-4">
				<div>
					<label for="username" class={labelClass}>Username</label>
					<input
						type="text"
						id="username"
						bind:value={username}
						required
						class={inputClass}
						placeholder="Enter username"
					/>
				</div>
				<div>
					<label for="password" class={labelClass}>Password</label>
					<input
						type="password"
						id="password"
						bind:value={password}
						required
						class={inputClass}
						placeholder="Enter password"
					/>
				</div>
				<div>
					<label for="role" class={labelClass}>Role</label>
					<Select
						id="role"
						bind:value={role}
						options={[
							{ value: 'user', label: 'User' },
							{ value: 'admin', label: 'Admin' },
							{ value: 'kid', label: 'Kid' }
						]}
					/>
				</div>
			</div>

			<!-- Parental Controls Section -->
			<div class="border-t border-border-subtle pt-4 mt-4">
				<h3 class="text-sm font-medium text-text-primary mb-3">Parental Controls</h3>
				<div class="grid sm:grid-cols-3 gap-4">
					<div>
						<label for="contentRating" class={labelClass}>Content Rating Limit</label>
						<select
							id="contentRating"
							bind:value={contentRatingLimit}
							class={selectClass}
						>
							{#each contentRatingOptions as option}
								<option value={option.value || null}>{option.label}</option>
							{/each}
						</select>
						<p class="text-xs text-text-muted mt-1">User will only see content at or below this rating</p>
					</div>
					<div>
						<label class={labelClass}>Require PIN</label>
						<label class="flex items-center gap-2 mt-2 cursor-pointer">
							<input
								type="checkbox"
								bind:checked={requirePin}
								class="w-4 h-4 rounded border-border-subtle"
							/>
							<span class="text-sm text-text-secondary">Require PIN for restricted content</span>
						</label>
					</div>
					{#if requirePin}
						<div>
							<label for="pin" class={labelClass}>Set PIN (4 digits)</label>
							<input
								type="password"
								id="pin"
								bind:value={pin}
								maxlength="4"
								pattern="[0-9]{4}"
								class={inputClass}
								placeholder="0000"
							/>
						</div>
					{/if}
				</div>
			</div>

			<div class="flex gap-3 pt-2">
				<button type="submit" class="liquid-btn">
					Create User
				</button>
				<button type="button" onclick={cancelAdd} class="liquid-btn !bg-white/5 !border-t-white/10 text-text-secondary hover:text-text-primary">
					Cancel
				</button>
			</div>
		</form>
	{/if}

	{#if editingUser}
		<form
			class="glass-card p-6 space-y-4"
			onsubmit={(e) => { e.preventDefault(); handleUpdateUser(); }}
		>
			<div class="flex items-center gap-3 mb-2">
				<div class="w-10 h-10 rounded-xl bg-white-600/20 flex items-center justify-center">
					<svg class="w-5 h-5 text-white-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z" />
					</svg>
				</div>
				<h2 class="text-lg font-semibold text-text-primary">Edit User</h2>
			</div>
			<div class="grid sm:grid-cols-3 gap-4">
				<div>
					<label for="edit-username" class={labelClass}>Username</label>
					<input
						type="text"
						id="edit-username"
						bind:value={username}
						required
						class={inputClass}
					/>
				</div>
				<div>
					<label for="edit-password" class={labelClass}>Password</label>
					<input
						type="password"
						id="edit-password"
						bind:value={password}
						class={inputClass}
						placeholder="Leave blank to keep current"
					/>
				</div>
				<div>
					<label for="edit-role" class={labelClass}>Role</label>
					<Select
						id="edit-role"
						bind:value={role}
						options={[
							{ value: 'user', label: 'User' },
							{ value: 'admin', label: 'Admin' },
							{ value: 'kid', label: 'Kid' }
						]}
					/>
				</div>
			</div>

			<!-- Parental Controls Section -->
			<div class="border-t border-border-subtle pt-4 mt-4">
				<h3 class="text-sm font-medium text-text-primary mb-3">Parental Controls</h3>
				<div class="grid sm:grid-cols-3 gap-4">
					<div>
						<label for="edit-contentRating" class={labelClass}>Content Rating Limit</label>
						<select
							id="edit-contentRating"
							bind:value={contentRatingLimit}
							class={selectClass}
						>
							{#each contentRatingOptions as option}
								<option value={option.value || null}>{option.label}</option>
							{/each}
						</select>
						<p class="text-xs text-text-muted mt-1">User will only see content at or below this rating</p>
					</div>
					<div>
						<label class={labelClass}>Require PIN</label>
						<label class="flex items-center gap-2 mt-2 cursor-pointer">
							<input
								type="checkbox"
								bind:checked={requirePin}
								class="w-4 h-4 rounded border-border-subtle"
							/>
							<span class="text-sm text-text-secondary">Require PIN for restricted content</span>
						</label>
					</div>
					{#if requirePin}
						<div>
							<label for="edit-pin" class={labelClass}>
								{editingUser.hasPin ? 'Change PIN (4 digits)' : 'Set PIN (4 digits)'}
							</label>
							<input
								type="password"
								id="edit-pin"
								bind:value={pin}
								maxlength="4"
								pattern="[0-9]{4}"
								class={inputClass}
								placeholder={editingUser.hasPin ? 'Leave blank to keep current' : '0000'}
							/>
							{#if editingUser.hasPin}
								<label class="flex items-center gap-2 mt-2 cursor-pointer">
									<input
										type="checkbox"
										bind:checked={clearPin}
										class="w-4 h-4 rounded border-border-subtle"
									/>
									<span class="text-xs text-text-muted">Remove PIN</span>
								</label>
							{/if}
						</div>
					{/if}
				</div>
			</div>

			<div class="flex gap-3 pt-2">
				<button type="submit" class="liquid-btn">
					Save Changes
				</button>
				<button type="button" onclick={cancelEdit} class="liquid-btn !bg-white/5 !border-t-white/10 text-text-secondary hover:text-text-primary">
					Cancel
				</button>
			</div>
		</form>
	{/if}

	{#if loading}
		<div class="flex items-center justify-center py-12">
			<div class="flex items-center gap-3">
				<div class="spinner-lg text-cream"></div>
				<p class="text-text-secondary">Loading users...</p>
			</div>
		</div>
	{:else if users.length === 0}
		<div class="glass-card p-12 text-center">
			<div class="w-16 h-16 mx-auto mb-4 rounded-full bg-bg-elevated flex items-center justify-center">
				<svg class="w-8 h-8 text-text-muted" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M12 4.354a4 4 0 110 5.292M15 21H3v-1a6 6 0 0112 0v1zm0 0h6v-1a6 6 0 00-9-5.197M13 7a4 4 0 11-8 0 4 4 0 018 0z" />
				</svg>
			</div>
			<h2 class="text-xl font-semibold text-text-primary mb-2">No users found</h2>
			<p class="text-text-secondary">Create a user to get started.</p>
		</div>
	{:else}
		<div class="glass-card overflow-hidden">
			<table class="w-full">
				<thead class="border-b border-border-subtle">
					<tr>
						<th class="text-left px-4 py-3 text-sm font-medium text-text-muted">User</th>
						<th class="text-left px-4 py-3 text-sm font-medium text-text-muted">Role</th>
						<th class="text-left px-4 py-3 text-sm font-medium text-text-muted">Content Limit</th>
						<th class="text-right px-4 py-3 text-sm font-medium text-text-muted">Actions</th>
					</tr>
				</thead>
				<tbody class="divide-y divide-white/5">
					{#each users as user}
						<tr class="hover:bg-glass transition-colors">
							<td class="px-4 py-4">
								<div class="flex items-center gap-3">
									<div class="w-10 h-10 rounded-full bg-bg-elevated flex items-center justify-center">
										<svg class="w-5 h-5 text-text-muted" fill="none" stroke="currentColor" viewBox="0 0 24 24">
											<path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d={getRoleIcon(user.role)} />
										</svg>
									</div>
									<div>
										<span class="font-medium text-text-primary">{user.username}</span>
										{#if user.requirePin}
											<span class="ml-2 text-xs text-amber-400" title="PIN protected">
												<svg class="w-3.5 h-3.5 inline" fill="none" stroke="currentColor" viewBox="0 0 24 24">
													<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z" />
												</svg>
											</span>
										{/if}
									</div>
								</div>
							</td>
							<td class="px-4 py-4">
								<span class="{getRoleBadgeColor(user.role)}">
									{user.role.charAt(0).toUpperCase() + user.role.slice(1)}
								</span>
							</td>
							<td class="px-4 py-4">
								{#if user.contentRatingLimit}
									<span class="text-sm text-text-secondary">{user.contentRatingLimit}</span>
								{:else}
									<span class="text-sm text-text-muted">None</span>
								{/if}
							</td>
							<td class="px-4 py-4 text-right">
								<div class="flex items-center justify-end gap-2">
									<button
										onclick={() => startEdit(user)}
										class="liquid-btn-sm"
									>
										Edit
									</button>
									<button
										onclick={() => handleDeleteUser(user.id)}
										class="liquid-btn-sm !bg-white/5 !border-t-white/10 text-text-secondary hover:text-text-primary"
									>
										Delete
									</button>
								</div>
							</td>
						</tr>
					{/each}
				</tbody>
			</table>
		</div>
	{/if}

	<div class="glass-card p-6">
		<div class="flex items-center gap-3 mb-4">
			<div class="w-10 h-10 rounded-xl bg-bg-elevated flex items-center justify-center">
				<svg class="w-5 h-5 text-text-muted" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
				</svg>
			</div>
			<h2 class="text-lg font-semibold text-text-primary">Role Descriptions</h2>
		</div>
		<div class="grid sm:grid-cols-3 gap-4">
			<div class="liquid-panel p-4">
				<div class="flex items-center gap-2 mb-2">
					<span class="liquid-badge-sm !bg-white-500/20 !border-t-white-400/40 text-white-400">Admin</span>
				</div>
				<p class="text-sm text-text-secondary">Full access - can manage users, libraries, and settings</p>
			</div>
			<div class="liquid-panel p-4">
				<div class="flex items-center gap-2 mb-2">
					<span class="liquid-badge-sm !bg-blue-500/20 !border-t-blue-400/40 text-blue-400">User</span>
				</div>
				<p class="text-sm text-text-secondary">Standard access - can browse, watch, and request content</p>
			</div>
			<div class="liquid-panel p-4">
				<div class="flex items-center gap-2 mb-2">
					<span class="liquid-badge-sm !bg-green-500/20 !border-t-green-400/40 text-green-400">Kid</span>
				</div>
				<p class="text-sm text-text-secondary">Filtered access - only sees age-appropriate content</p>
			</div>
		</div>
	</div>
</div>
