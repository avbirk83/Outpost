import { API_BASE, apiFetch } from './core';

export async function getSettings(): Promise<Record<string, string>> {
	const response = await apiFetch(`${API_BASE}/settings`);
	if (!response.ok) {
		throw new Error(`API error: ${response.status}`);
	}
	return response.json();
}

export async function saveSettings(settings: Record<string, string>): Promise<void> {
	const response = await apiFetch(`${API_BASE}/settings`, {
		method: 'POST',
		headers: { 'Content-Type': 'application/json' },
		body: JSON.stringify(settings)
	});
	if (!response.ok) {
		throw new Error(`API error: ${response.status}`);
	}
}

export async function refreshAllMetadata(): Promise<{ refreshed: number; errors: number; total: number }> {
	const response = await apiFetch(`${API_BASE}/metadata/refresh`, {
		method: 'POST'
	});
	if (!response.ok) {
		throw new Error(`API error: ${response.status}`);
	}
	return response.json();
}

export async function clearLibraryData(): Promise<{ success: boolean; message: string }> {
	const response = await apiFetch(`${API_BASE}/library/clear`, {
		method: 'POST'
	});
	if (!response.ok) {
		throw new Error(`API error: ${response.status}`);
	}
	return response.json();
}

// Naming Templates

export interface NamingTemplate {
	id: number;
	type: 'movie' | 'tv' | 'daily';
	folderTemplate: string;
	fileTemplate: string;
	isDefault: boolean;
}

export async function getNamingTemplates(): Promise<NamingTemplate[]> {
	const response = await apiFetch(`${API_BASE}/settings/naming`);
	if (!response.ok) throw new Error(`API error: ${response.status}`);
	return response.json();
}

export async function updateNamingTemplate(template: NamingTemplate): Promise<NamingTemplate> {
	const response = await apiFetch(`${API_BASE}/settings/naming`, {
		method: 'PUT',
		headers: { 'Content-Type': 'application/json' },
		body: JSON.stringify(template),
	});
	if (!response.ok) throw new Error(`API error: ${response.status}`);
	return response.json();
}

// Format Settings (pre-grab container/format filtering)

export interface FormatSettings {
	acceptedContainers: string[];
	rejectDiscs: boolean;
	rejectArchives: boolean;
	autoBlocklist: boolean;
}

export async function getFormatSettings(): Promise<FormatSettings> {
	const response = await apiFetch(`${API_BASE}/settings/formats`);
	if (!response.ok) throw new Error(`API error: ${response.status}`);
	return response.json();
}

export async function saveFormatSettings(settings: FormatSettings): Promise<FormatSettings> {
	const response = await apiFetch(`${API_BASE}/settings/formats`, {
		method: 'PUT',
		headers: { 'Content-Type': 'application/json' },
		body: JSON.stringify(settings),
	});
	if (!response.ok) throw new Error(`API error: ${response.status}`);
	return response.json();
}
