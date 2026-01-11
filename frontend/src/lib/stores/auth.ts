import { writable, get } from 'svelte/store';
import { getCurrentUser, login as apiLogin, logout as apiLogout, verifyPin as apiVerifyPin, type User } from '$lib/api';

function createAuthStore() {
	const { subscribe, set, update } = writable<User | null>(null);
	let initialized = false;

	return {
		subscribe,
		get: () => get({ subscribe }),
		init: async () => {
			if (initialized) return;
			initialized = true;
			const user = await getCurrentUser();
			set(user);
		},
		login: async (username: string, password: string) => {
			const response = await apiLogin(username, password);
			set(response.user);
			return response.user;
		},
		logout: async () => {
			await apiLogout();
			set(null);
		},
		setUser: (user: User | null) => {
			set(user);
		},
		// PIN elevation methods
		verifyPin: async (pin: string): Promise<{ success: boolean; error?: string }> => {
			try {
				const response = await apiVerifyPin(pin);
				if (response.valid) {
					// Update user to reflect elevation
					update(user => user ? { ...user, isElevated: true } : null);
					return { success: true };
				}
				return { success: false, error: response.error || 'Incorrect PIN' };
			} catch (e) {
				return { success: false, error: 'Failed to verify PIN' };
			}
		},
		clearElevation: () => {
			update(user => user ? { ...user, isElevated: false } : null);
			// Clear elevation cookie
			document.cookie = 'elevation_token=; expires=Thu, 01 Jan 1970 00:00:00 UTC; path=/;';
		},
		refreshUser: async () => {
			const user = await getCurrentUser();
			set(user);
			return user;
		}
	};
}

export const auth = createAuthStore();
