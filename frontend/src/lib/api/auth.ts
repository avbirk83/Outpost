import { API_BASE, apiFetch } from './core';

export interface User {
	id: number;
	username: string;
	role: 'admin' | 'user' | 'kid';
}

export interface LoginResponse {
	token: string;
	user: User;
}

export interface SetupStatus {
	setupRequired: boolean;
}

// Auth functions

export async function checkSetup(): Promise<SetupStatus> {
	const response = await apiFetch(`${API_BASE}/auth/setup`);
	if (!response.ok) {
		throw new Error(`API error: ${response.status}`);
	}
	return response.json();
}

export async function setup(username: string, password: string): Promise<User> {
	const response = await apiFetch(`${API_BASE}/auth/setup`, {
		method: 'POST',
		headers: { 'Content-Type': 'application/json' },
		body: JSON.stringify({ username, password })
	});
	if (!response.ok) {
		throw new Error(`API error: ${response.status}`);
	}
	return response.json();
}

export async function login(username: string, password: string): Promise<LoginResponse> {
	const response = await apiFetch(`${API_BASE}/auth/login`, {
		method: 'POST',
		headers: { 'Content-Type': 'application/json' },
		body: JSON.stringify({ username, password })
	});
	if (!response.ok) {
		throw new Error(`Invalid credentials`);
	}
	return response.json();
}

export async function logout(): Promise<void> {
	const response = await apiFetch(`${API_BASE}/auth/logout`, {
		method: 'POST'
	});
	if (!response.ok) {
		throw new Error(`API error: ${response.status}`);
	}
}

export async function getCurrentUser(): Promise<User | null> {
	try {
		const response = await apiFetch(`${API_BASE}/auth/me`);
		if (!response.ok) {
			return null;
		}
		return response.json();
	} catch {
		return null;
	}
}

// User management functions

export async function getUsers(): Promise<User[]> {
	const response = await apiFetch(`${API_BASE}/users`);
	if (!response.ok) {
		throw new Error(`API error: ${response.status}`);
	}
	return response.json();
}

export async function createUser(username: string, password: string, role: string): Promise<User> {
	const response = await apiFetch(`${API_BASE}/users`, {
		method: 'POST',
		headers: { 'Content-Type': 'application/json' },
		body: JSON.stringify({ username, password, role })
	});
	if (!response.ok) {
		throw new Error(`API error: ${response.status}`);
	}
	return response.json();
}

export async function updateUser(id: number, data: { username?: string; password?: string; role?: string }): Promise<User> {
	const response = await apiFetch(`${API_BASE}/users/${id}`, {
		method: 'PUT',
		headers: { 'Content-Type': 'application/json' },
		body: JSON.stringify(data)
	});
	if (!response.ok) {
		throw new Error(`API error: ${response.status}`);
	}
	return response.json();
}

export async function deleteUser(id: number): Promise<void> {
	const response = await apiFetch(`${API_BASE}/users/${id}`, {
		method: 'DELETE'
	});
	if (!response.ok) {
		throw new Error(`API error: ${response.status}`);
	}
}
