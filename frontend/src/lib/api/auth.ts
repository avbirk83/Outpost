import { API_BASE, apiFetch } from './core';

export interface User {
	id: number;
	username: string;
	role: 'admin' | 'user' | 'kid';
	contentRatingLimit?: 'G' | 'PG' | 'PG-13' | 'R' | 'NC-17' | null;
	requirePin?: boolean;
	isElevated?: boolean;
	hasPin?: boolean;
}

export type ContentRating = 'G' | 'PG' | 'PG-13' | 'R' | 'NC-17';

export interface PinVerifyResponse {
	valid: boolean;
	token?: string;
	error?: string;
}

export interface LoginResponse {
	token: string;
	user: User;
}

export interface SetupStatus {
	setupRequired: boolean;
}

export interface SetupWizardSteps {
	adminCreated: boolean;
	libraryAdded: boolean;
	downloadClientConfigured: boolean;
	indexerConfigured: boolean;
	qualityProfileSet: boolean;
}

export interface SetupWizardStatus {
	needsSetup: boolean;
	setupCompleted: boolean;
	steps?: SetupWizardSteps;
	canSkip?: boolean;
}

// Auth functions

export async function checkSetup(): Promise<SetupStatus> {
	const response = await apiFetch(`${API_BASE}/auth/setup`);
	if (!response.ok) {
		throw new Error(`API error: ${response.status}`);
	}
	return response.json();
}

export async function getSetupWizardStatus(): Promise<SetupWizardStatus> {
	const response = await apiFetch(`${API_BASE}/setup/status`);
	if (!response.ok) {
		throw new Error(`API error: ${response.status}`);
	}
	return response.json();
}

export async function completeSetupWizard(): Promise<{ success: boolean }> {
	const response = await apiFetch(`${API_BASE}/setup/complete`, {
		method: 'POST'
	});
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

export interface CreateUserData {
	username: string;
	password: string;
	role: string;
	contentRatingLimit?: ContentRating | null;
	requirePin?: boolean;
	pin?: string;
}

export async function createUser(data: CreateUserData): Promise<User> {
	const response = await apiFetch(`${API_BASE}/users`, {
		method: 'POST',
		headers: { 'Content-Type': 'application/json' },
		body: JSON.stringify(data)
	});
	if (!response.ok) {
		throw new Error(`API error: ${response.status}`);
	}
	return response.json();
}

export interface UpdateUserData {
	username?: string;
	password?: string;
	role?: string;
	contentRatingLimit?: ContentRating | null;
	requirePin?: boolean;
	pin?: string;
	clearPin?: boolean;
}

export async function updateUser(id: number, data: UpdateUserData): Promise<User> {
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

export async function verifyPin(pin: string): Promise<PinVerifyResponse> {
	const response = await apiFetch(`${API_BASE}/auth/verify-pin`, {
		method: 'POST',
		headers: { 'Content-Type': 'application/json' },
		body: JSON.stringify({ pin })
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
