import { writable, get } from 'svelte/store';
import {
	getProfiles,
	getActiveProfile,
	selectProfile as apiSelectProfile,
	createProfile as apiCreateProfile,
	updateProfile as apiUpdateProfile,
	deleteProfile as apiDeleteProfile,
	type Profile,
	type CreateProfileData,
	type UpdateProfileData
} from '$lib/api';

function createProfileStore() {
	const { subscribe, set, update } = writable<{
		profiles: Profile[];
		activeProfile: Profile | null;
		loading: boolean;
	}>({
		profiles: [],
		activeProfile: null,
		loading: true
	});

	return {
		subscribe,
		get: () => get({ subscribe }),

		// Initialize store - fetch profiles and active profile
		init: async () => {
			update(s => ({ ...s, loading: true }));
			try {
				const [profiles, activeProfile] = await Promise.all([
					getProfiles(),
					getActiveProfile()
				]);
				set({ profiles, activeProfile, loading: false });
			} catch {
				set({ profiles: [], activeProfile: null, loading: false });
			}
		},

		// Refresh profiles list
		refresh: async () => {
			try {
				const profiles = await getProfiles();
				update(s => ({ ...s, profiles }));
			} catch {
				// Keep existing state on error
			}
		},

		// Select a profile
		select: async (profileId: number) => {
			const profile = await apiSelectProfile(profileId);
			update(s => ({ ...s, activeProfile: profile }));
			return profile;
		},

		// Create a new profile
		create: async (data: CreateProfileData) => {
			const profile = await apiCreateProfile(data);
			update(s => ({
				...s,
				profiles: [...s.profiles, profile]
			}));
			return profile;
		},

		// Update a profile
		update: async (id: number, data: UpdateProfileData) => {
			const profile = await apiUpdateProfile(id, data);
			update(s => ({
				...s,
				profiles: s.profiles.map(p => p.id === id ? profile : p),
				activeProfile: s.activeProfile?.id === id ? profile : s.activeProfile
			}));
			return profile;
		},

		// Delete a profile
		delete: async (id: number) => {
			await apiDeleteProfile(id);
			update(s => ({
				...s,
				profiles: s.profiles.filter(p => p.id !== id)
			}));
		},

		// Clear store (on logout)
		clear: () => {
			set({ profiles: [], activeProfile: null, loading: false });
		},

		// Check if a profile is selected
		hasActiveProfile: () => {
			const state = get({ subscribe });
			return state.activeProfile !== null;
		}
	};
}

export const profileStore = createProfileStore();
