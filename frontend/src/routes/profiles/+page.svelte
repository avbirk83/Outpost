<script lang="ts">
	import { goto } from '$app/navigation';
	import { profileStore } from '$lib/stores/profile';
	import { auth } from '$lib/stores/auth';
	import { AVATARS, type Profile, type CreateProfileData } from '$lib/api';
	import { onMount } from 'svelte';

	let profiles = $state<Profile[]>([]);
	let loading = $state(true);
	let showCreateModal = $state(false);
	let showEditModal = $state(false);
	let editingProfile = $state<Profile | null>(null);
	let isManaging = $state(false);

	// Form state
	let newName = $state('');
	let newAvatar = $state(AVATARS[0]);
	let newIsKid = $state(false);
	let saving = $state(false);
	let error = $state('');

	// Subscribe to profile store
	profileStore.subscribe((state) => {
		profiles = state.profiles;
		loading = state.loading;
	});

	onMount(async () => {
		await profileStore.init();
	});

	async function selectProfile(profile: Profile) {
		if (isManaging) {
			editingProfile = profile;
			newName = profile.name;
			newAvatar = profile.avatarUrl || AVATARS[0];
			newIsKid = profile.isKid;
			showEditModal = true;
		} else {
			try {
				await profileStore.select(profile.id);
				goto('/');
			} catch (e) {
				console.error('Failed to select profile:', e);
			}
		}
	}

	function openCreateModal() {
		newName = '';
		newAvatar = AVATARS[Math.floor(Math.random() * AVATARS.length)];
		newIsKid = false;
		error = '';
		showCreateModal = true;
	}

	async function createProfile() {
		if (!newName.trim()) {
			error = 'Profile name is required';
			return;
		}

		saving = true;
		error = '';

		try {
			const profile = await profileStore.create({
				name: newName.trim(),
				avatarUrl: newAvatar,
				isKid: newIsKid
			});
			showCreateModal = false;
			// Auto-select the new profile
			await profileStore.select(profile.id);
			goto('/');
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to create profile';
		} finally {
			saving = false;
		}
	}

	async function updateProfile() {
		if (!editingProfile || !newName.trim()) {
			error = 'Profile name is required';
			return;
		}

		saving = true;
		error = '';

		try {
			await profileStore.update(editingProfile.id, {
				name: newName.trim(),
				avatarUrl: newAvatar,
				isKid: newIsKid
			});
			showEditModal = false;
			editingProfile = null;
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to update profile';
		} finally {
			saving = false;
		}
	}

	async function deleteProfile() {
		if (!editingProfile) return;

		if (!confirm(`Are you sure you want to delete the profile "${editingProfile.name}"?`)) {
			return;
		}

		saving = true;
		error = '';

		try {
			await profileStore.delete(editingProfile.id);
			showEditModal = false;
			editingProfile = null;
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to delete profile';
		} finally {
			saving = false;
		}
	}

	function closeModals() {
		showCreateModal = false;
		showEditModal = false;
		editingProfile = null;
		error = '';
	}

	function getAvatarUrl(profile: Profile): string {
		return profile.avatarUrl || AVATARS[0];
	}
</script>

<div class="min-h-screen bg-bg-primary flex flex-col items-center justify-center p-8">
	<h1 class="text-4xl font-bold text-text-primary mb-2">Who's watching?</h1>
	<p class="text-text-secondary mb-12">Select a profile to continue</p>

	{#if loading}
		<div class="flex items-center gap-3">
			<div class="spinner-lg text-cream"></div>
			<p class="text-text-secondary">Loading profiles...</p>
		</div>
	{:else}
		<div class="flex flex-wrap justify-center gap-6 mb-12">
			{#each profiles as profile}
				<button
					class="group flex flex-col items-center gap-3 p-4 rounded-xl transition-all hover:bg-bg-secondary {isManaging
						? 'ring-2 ring-cream/30'
						: ''}"
					onclick={() => selectProfile(profile)}
				>
					<div
						class="w-32 h-32 rounded-xl overflow-hidden ring-4 ring-transparent group-hover:ring-cream transition-all {isManaging
							? 'ring-cream/50'
							: ''}"
					>
						<img
							src={getAvatarUrl(profile)}
							alt={profile.name}
							class="w-full h-full object-cover"
						/>
					</div>
					<span class="text-lg text-text-primary group-hover:text-cream transition-colors">
						{profile.name}
					</span>
					{#if profile.isKid}
						<span class="text-xs px-2 py-0.5 bg-green-600/20 text-green-400 rounded">Kids</span>
					{/if}
					{#if profile.isDefault}
						<span class="text-xs px-2 py-0.5 bg-cream/20 text-cream rounded">Default</span>
					{/if}
				</button>
			{/each}

			{#if profiles.length < 5}
				<button
					class="group flex flex-col items-center gap-3 p-4 rounded-xl transition-all hover:bg-bg-secondary"
					onclick={openCreateModal}
				>
					<div
						class="w-32 h-32 rounded-xl border-2 border-dashed border-border-primary flex items-center justify-center group-hover:border-cream transition-colors"
					>
						<svg
							class="w-12 h-12 text-text-tertiary group-hover:text-cream transition-colors"
							fill="none"
							stroke="currentColor"
							viewBox="0 0 24 24"
						>
							<path
								stroke-linecap="round"
								stroke-linejoin="round"
								stroke-width="2"
								d="M12 4v16m8-8H4"
							/>
						</svg>
					</div>
					<span class="text-lg text-text-secondary group-hover:text-cream transition-colors">
						Add Profile
					</span>
				</button>
			{/if}
		</div>

		<button
			class="text-text-secondary hover:text-text-primary border border-border-primary hover:border-cream px-6 py-2 rounded-lg transition-colors"
			onclick={() => (isManaging = !isManaging)}
		>
			{isManaging ? 'Done' : 'Manage Profiles'}
		</button>
	{/if}
</div>

<!-- Create Profile Modal -->
{#if showCreateModal}
	<div class="fixed inset-0 bg-black/70 flex items-center justify-center z-50 p-4">
		<div
			class="bg-bg-secondary rounded-2xl p-8 max-w-md w-full"
			onclick={(e) => e.stopPropagation()}
		>
			<h2 class="text-2xl font-bold text-text-primary mb-6">Create Profile</h2>

			{#if error}
				<div class="bg-red-500/10 border border-red-500/30 text-red-400 px-4 py-2 rounded-lg mb-4">
					{error}
				</div>
			{/if}

			<div class="space-y-6">
				<!-- Avatar Selection -->
				<div>
					<label class="block text-sm font-medium text-text-secondary mb-3">Avatar</label>
					<div class="grid grid-cols-6 gap-2">
						{#each AVATARS as avatar}
							<button
								type="button"
								class="w-12 h-12 rounded-lg overflow-hidden ring-2 transition-all {newAvatar ===
								avatar
									? 'ring-cream'
									: 'ring-transparent hover:ring-cream/50'}"
								onclick={() => (newAvatar = avatar)}
							>
								<img src={avatar} alt="Avatar option" class="w-full h-full object-cover" />
							</button>
						{/each}
					</div>
				</div>

				<!-- Name Input -->
				<div>
					<label for="profile-name" class="block text-sm font-medium text-text-secondary mb-2">
						Name
					</label>
					<input
						id="profile-name"
						type="text"
						bind:value={newName}
						placeholder="Enter profile name"
						class="w-full px-4 py-2.5 bg-bg-tertiary border border-border-primary rounded-lg text-text-primary placeholder-text-tertiary focus:outline-none focus:ring-2 focus:ring-cream/50"
						maxlength="20"
					/>
				</div>

				<!-- Kids Profile Toggle -->
				<label class="flex items-center gap-3 cursor-pointer">
					<input
						type="checkbox"
						bind:checked={newIsKid}
						class="w-5 h-5 rounded border-border-primary bg-bg-tertiary text-cream focus:ring-cream"
					/>
					<span class="text-text-primary">This is a kids profile</span>
				</label>
			</div>

			<div class="flex gap-3 mt-8">
				<button
					class="flex-1 px-4 py-2.5 bg-bg-tertiary text-text-primary rounded-lg hover:bg-bg-primary transition-colors"
					onclick={closeModals}
					disabled={saving}
				>
					Cancel
				</button>
				<button
					class="flex-1 px-4 py-2.5 bg-cream text-bg-primary font-medium rounded-lg hover:bg-cream/90 transition-colors disabled:opacity-50"
					onclick={createProfile}
					disabled={saving || !newName.trim()}
				>
					{saving ? 'Creating...' : 'Create'}
				</button>
			</div>
		</div>
	</div>
{/if}

<!-- Edit Profile Modal -->
{#if showEditModal && editingProfile}
	<div class="fixed inset-0 bg-black/70 flex items-center justify-center z-50 p-4">
		<div
			class="bg-bg-secondary rounded-2xl p-8 max-w-md w-full"
			onclick={(e) => e.stopPropagation()}
		>
			<h2 class="text-2xl font-bold text-text-primary mb-6">Edit Profile</h2>

			{#if error}
				<div class="bg-red-500/10 border border-red-500/30 text-red-400 px-4 py-2 rounded-lg mb-4">
					{error}
				</div>
			{/if}

			<div class="space-y-6">
				<!-- Avatar Selection -->
				<div>
					<label class="block text-sm font-medium text-text-secondary mb-3">Avatar</label>
					<div class="grid grid-cols-6 gap-2">
						{#each AVATARS as avatar}
							<button
								type="button"
								class="w-12 h-12 rounded-lg overflow-hidden ring-2 transition-all {newAvatar ===
								avatar
									? 'ring-cream'
									: 'ring-transparent hover:ring-cream/50'}"
								onclick={() => (newAvatar = avatar)}
							>
								<img src={avatar} alt="Avatar option" class="w-full h-full object-cover" />
							</button>
						{/each}
					</div>
				</div>

				<!-- Name Input -->
				<div>
					<label for="edit-profile-name" class="block text-sm font-medium text-text-secondary mb-2">
						Name
					</label>
					<input
						id="edit-profile-name"
						type="text"
						bind:value={newName}
						placeholder="Enter profile name"
						class="w-full px-4 py-2.5 bg-bg-tertiary border border-border-primary rounded-lg text-text-primary placeholder-text-tertiary focus:outline-none focus:ring-2 focus:ring-cream/50"
						maxlength="20"
					/>
				</div>

				<!-- Kids Profile Toggle -->
				<label class="flex items-center gap-3 cursor-pointer">
					<input
						type="checkbox"
						bind:checked={newIsKid}
						class="w-5 h-5 rounded border-border-primary bg-bg-tertiary text-cream focus:ring-cream"
					/>
					<span class="text-text-primary">This is a kids profile</span>
				</label>
			</div>

			<div class="flex gap-3 mt-8">
				{#if !editingProfile.isDefault && profiles.length > 1}
					<button
						class="px-4 py-2.5 bg-red-600/20 text-red-400 rounded-lg hover:bg-red-600/30 transition-colors"
						onclick={deleteProfile}
						disabled={saving}
					>
						Delete
					</button>
				{/if}
				<button
					class="flex-1 px-4 py-2.5 bg-bg-tertiary text-text-primary rounded-lg hover:bg-bg-primary transition-colors"
					onclick={closeModals}
					disabled={saving}
				>
					Cancel
				</button>
				<button
					class="flex-1 px-4 py-2.5 bg-cream text-bg-primary font-medium rounded-lg hover:bg-cream/90 transition-colors disabled:opacity-50"
					onclick={updateProfile}
					disabled={saving || !newName.trim()}
				>
					{saving ? 'Saving...' : 'Save'}
				</button>
			</div>
		</div>
	</div>
{/if}
