<script lang="ts">
	import { onMount } from 'svelte';
	import { getUsers, createUser, updateUser, deleteUser, type User } from '$lib/api';

	let users: User[] = $state([]);
	let loading = $state(true);
	let error: string | null = $state(null);
	let showAddForm = $state(false);
	let editingUser: User | null = $state(null);

	// Form state
	let username = $state('');
	let password = $state('');
	let role = $state('user');

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

		try {
			await createUser(username, password, role);
			username = '';
			password = '';
			role = 'user';
			showAddForm = false;
			await loadUsers();
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to create user';
		}
	}

	async function handleUpdateUser() {
		if (!editingUser) return;

		try {
			const data: { username?: string; password?: string; role?: string } = {
				username: username,
				role: role
			};
			if (password) {
				data.password = password;
			}
			await updateUser(editingUser.id, data);
			editingUser = null;
			username = '';
			password = '';
			role = 'user';
			await loadUsers();
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to update user';
		}
	}

	async function handleDeleteUser(id: number) {
		if (!confirm('Are you sure you want to delete this user?')) return;

		try {
			await deleteUser(id);
			await loadUsers();
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to delete user';
		}
	}

	function startEdit(user: User) {
		editingUser = user;
		username = user.username;
		password = '';
		role = user.role;
		showAddForm = false;
	}

	function cancelEdit() {
		editingUser = null;
		username = '';
		password = '';
		role = 'user';
	}

	function cancelAdd() {
		showAddForm = false;
		username = '';
		password = '';
		role = 'user';
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
		<div>
			<h1 class="text-3xl font-bold text-text-primary">User Management</h1>
			<p class="text-text-secondary mt-1">Manage users and their access levels</p>
		</div>
		{#if !showAddForm && !editingUser}
			<button
				class="liquid-btn inline-flex items-center gap-2"
				onclick={() => { showAddForm = true; editingUser = null; }}
			>
				<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 6v6m0 0v6m0-6h6m-6 0H6" />
				</svg>
				Add User
			</button>
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
					<select id="role" bind:value={role} class={selectClass}>
						<option value="user">User</option>
						<option value="admin">Admin</option>
						<option value="kid">Kid</option>
					</select>
				</div>
			</div>
			<div class="flex gap-3 pt-2">
				<button type="submit" class="liquid-btn">
					Create User
				</button>
				<button type="button" onclick={cancelAdd} class="liquid-btn !bg-white/5 !border-t-white/10 text-text-secondary hover:text-white">
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
					<select id="edit-role" bind:value={role} class={selectClass}>
						<option value="user">User</option>
						<option value="admin">Admin</option>
						<option value="kid">Kid</option>
					</select>
				</div>
			</div>
			<div class="flex gap-3 pt-2">
				<button type="submit" class="liquid-btn">
					Save Changes
				</button>
				<button type="button" onclick={cancelEdit} class="liquid-btn !bg-white/5 !border-t-white/10 text-text-secondary hover:text-white">
					Cancel
				</button>
			</div>
		</form>
	{/if}

	{#if loading}
		<div class="flex items-center justify-center py-12">
			<div class="flex items-center gap-3">
				<div class="w-6 h-6 border-2 border-white/30 border-t-transparent rounded-full animate-spin"></div>
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
				<thead class="border-b border-white/10">
					<tr>
						<th class="text-left px-4 py-3 text-sm font-medium text-text-muted">User</th>
						<th class="text-left px-4 py-3 text-sm font-medium text-text-muted">Role</th>
						<th class="text-right px-4 py-3 text-sm font-medium text-text-muted">Actions</th>
					</tr>
				</thead>
				<tbody class="divide-y divide-white/5">
					{#each users as user}
						<tr class="hover:bg-white/5 transition-colors">
							<td class="px-4 py-4">
								<div class="flex items-center gap-3">
									<div class="w-10 h-10 rounded-full bg-bg-elevated flex items-center justify-center">
										<svg class="w-5 h-5 text-text-muted" fill="none" stroke="currentColor" viewBox="0 0 24 24">
											<path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d={getRoleIcon(user.role)} />
										</svg>
									</div>
									<span class="font-medium text-text-primary">{user.username}</span>
								</div>
							</td>
							<td class="px-4 py-4">
								<span class="{getRoleBadgeColor(user.role)}">
									{user.role.charAt(0).toUpperCase() + user.role.slice(1)}
								</span>
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
										class="liquid-btn-sm !bg-white/5 !border-t-white/10 text-text-secondary hover:text-white"
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
