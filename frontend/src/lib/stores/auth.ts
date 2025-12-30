import { writable } from 'svelte/store';
import { getCurrentUser, login as apiLogin, logout as apiLogout, type User } from '$lib/api';

function createAuthStore() {
	const { subscribe, set } = writable<User | null>(null);
	let initialized = false;

	return {
		subscribe,
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
		}
	};
}

export const auth = createAuthStore();
