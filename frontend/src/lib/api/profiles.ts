import { API_BASE, apiFetch } from './core';
import type { ContentRating } from './auth';

export interface Profile {
	id: number;
	userId: number;
	name: string;
	avatarUrl?: string | null;
	isDefault: boolean;
	isKid: boolean;
	contentRatingLimit?: ContentRating | null;
	createdAt: string;
}

export interface CreateProfileData {
	name: string;
	avatarUrl?: string;
	isKid?: boolean;
	contentRatingLimit?: ContentRating | null;
}

export interface UpdateProfileData {
	name?: string;
	avatarUrl?: string | null;
	isKid?: boolean;
	contentRatingLimit?: ContentRating | null;
}

export async function getProfiles(): Promise<Profile[]> {
	const response = await apiFetch(`${API_BASE}/profiles`);
	if (!response.ok) {
		throw new Error(`API error: ${response.status}`);
	}
	return response.json();
}

export async function getProfile(id: number): Promise<Profile> {
	const response = await apiFetch(`${API_BASE}/profiles/${id}`);
	if (!response.ok) {
		throw new Error(`API error: ${response.status}`);
	}
	return response.json();
}

export async function getActiveProfile(): Promise<Profile | null> {
	const response = await apiFetch(`${API_BASE}/profiles/active`);
	if (!response.ok) {
		throw new Error(`API error: ${response.status}`);
	}
	const data = await response.json();
	return data;
}

export async function createProfile(data: CreateProfileData): Promise<Profile> {
	const response = await apiFetch(`${API_BASE}/profiles`, {
		method: 'POST',
		headers: { 'Content-Type': 'application/json' },
		body: JSON.stringify(data)
	});
	if (!response.ok) {
		const text = await response.text();
		throw new Error(text || `API error: ${response.status}`);
	}
	return response.json();
}

export async function updateProfile(id: number, data: UpdateProfileData): Promise<Profile> {
	const response = await apiFetch(`${API_BASE}/profiles/${id}`, {
		method: 'PUT',
		headers: { 'Content-Type': 'application/json' },
		body: JSON.stringify(data)
	});
	if (!response.ok) {
		throw new Error(`API error: ${response.status}`);
	}
	return response.json();
}

export async function deleteProfile(id: number): Promise<void> {
	const response = await apiFetch(`${API_BASE}/profiles/${id}`, {
		method: 'DELETE'
	});
	if (!response.ok) {
		const text = await response.text();
		throw new Error(text || `API error: ${response.status}`);
	}
}

export async function selectProfile(id: number): Promise<Profile> {
	const response = await apiFetch(`${API_BASE}/profiles/${id}/select`, {
		method: 'POST'
	});
	if (!response.ok) {
		throw new Error(`API error: ${response.status}`);
	}
	return response.json();
}

// Available avatar options
export const AVATARS = [
	'/avatars/avatar-1.svg',
	'/avatars/avatar-2.svg',
	'/avatars/avatar-3.svg',
	'/avatars/avatar-4.svg',
	'/avatars/avatar-5.svg',
	'/avatars/avatar-6.svg',
	'/avatars/avatar-7.svg',
	'/avatars/avatar-8.svg',
	'/avatars/avatar-9.svg',
	'/avatars/avatar-10.svg',
	'/avatars/avatar-11.svg',
	'/avatars/avatar-12.svg'
];
