import { API_BASE, apiFetch } from './core';

export interface TraktConfig {
	connected: boolean;
	username?: string;
	syncEnabled: boolean;
	syncWatched: boolean;
	syncRatings: boolean;
	syncWatchlist: boolean;
	lastSyncedAt?: string;
}

export interface TraktSyncResult {
	success: boolean;
	pulled: {
		movies: number;
		shows: number;
	};
	pushed: {
		movies: number;
		episodes: number;
	};
	errors: string[];
}

export interface TraktTestResult {
	success: boolean;
	connected: boolean;
	username?: string;
	error?: string;
}

// Get OAuth authorization URL
export async function getTraktAuthURL(redirectUri: string): Promise<{ url: string }> {
	const response = await apiFetch(`${API_BASE}/trakt/auth-url?redirect_uri=${encodeURIComponent(redirectUri)}`);
	if (!response.ok) {
		const errorText = await response.text();
		throw new Error(errorText || `API error: ${response.status}`);
	}
	return response.json();
}

// Exchange OAuth code for tokens
export async function exchangeTraktCode(code: string, redirectUri: string): Promise<{ success: boolean; username: string }> {
	const response = await apiFetch(`${API_BASE}/trakt/callback`, {
		method: 'POST',
		headers: { 'Content-Type': 'application/json' },
		body: JSON.stringify({ code, redirectUri })
	});
	if (!response.ok) {
		const errorText = await response.text();
		throw new Error(errorText || `API error: ${response.status}`);
	}
	return response.json();
}

// Get current Trakt configuration
export async function getTraktConfig(): Promise<TraktConfig> {
	const response = await apiFetch(`${API_BASE}/trakt/config`);
	if (!response.ok) {
		const errorText = await response.text();
		throw new Error(errorText || `API error: ${response.status}`);
	}
	return response.json();
}

// Update Trakt sync settings
export async function updateTraktConfig(config: {
	syncEnabled: boolean;
	syncWatched: boolean;
	syncRatings: boolean;
	syncWatchlist: boolean;
}): Promise<{ success: boolean }> {
	const response = await apiFetch(`${API_BASE}/trakt/config`, {
		method: 'PUT',
		headers: { 'Content-Type': 'application/json' },
		body: JSON.stringify(config)
	});
	if (!response.ok) {
		const errorText = await response.text();
		throw new Error(errorText || `API error: ${response.status}`);
	}
	return response.json();
}

// Disconnect Trakt account
export async function disconnectTrakt(): Promise<{ success: boolean }> {
	const response = await apiFetch(`${API_BASE}/trakt/disconnect`, {
		method: 'POST'
	});
	if (!response.ok) {
		const errorText = await response.text();
		throw new Error(errorText || `API error: ${response.status}`);
	}
	return response.json();
}

// Trigger manual sync
export async function syncTrakt(): Promise<TraktSyncResult> {
	const response = await apiFetch(`${API_BASE}/trakt/sync`, {
		method: 'POST'
	});
	if (!response.ok) {
		const errorText = await response.text();
		throw new Error(errorText || `API error: ${response.status}`);
	}
	return response.json();
}

// Test Trakt connection
export async function testTraktConnection(): Promise<TraktTestResult> {
	const response = await apiFetch(`${API_BASE}/trakt/test`);
	if (!response.ok) {
		const errorText = await response.text();
		throw new Error(errorText || `API error: ${response.status}`);
	}
	return response.json();
}

// Format last synced date
export function formatLastSynced(dateStr?: string): string {
	if (!dateStr) return 'Never';
	const date = new Date(dateStr);
	const now = new Date();
	const diffMs = now.getTime() - date.getTime();
	const diffMins = Math.floor(diffMs / 60000);
	const diffHours = Math.floor(diffMs / 3600000);
	const diffDays = Math.floor(diffMs / 86400000);

	if (diffMins < 1) return 'Just now';
	if (diffMins < 60) return `${diffMins} minute${diffMins === 1 ? '' : 's'} ago`;
	if (diffHours < 24) return `${diffHours} hour${diffHours === 1 ? '' : 's'} ago`;
	if (diffDays < 7) return `${diffDays} day${diffDays === 1 ? '' : 's'} ago`;
	return date.toLocaleDateString();
}
