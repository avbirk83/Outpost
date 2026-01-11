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
	rejectedKeywords: string[];
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

// Backup and Restore

export interface RestoreResult {
	success: boolean;
	restored: Record<string, number>;
	warnings: string[];
	errors?: string[];
}

export async function downloadBackup(): Promise<void> {
	const response = await apiFetch(`${API_BASE}/backup`, {
		method: 'POST'
	});
	if (!response.ok) {
		throw new Error(`API error: ${response.status}`);
	}

	// Get the filename from Content-Disposition header or use default
	const contentDisposition = response.headers.get('Content-Disposition');
	let filename = 'outpost-backup.json';
	if (contentDisposition) {
		const match = contentDisposition.match(/filename="(.+)"/);
		if (match) filename = match[1];
	}

	// Create blob and download
	const blob = await response.blob();
	const url = URL.createObjectURL(blob);
	const a = document.createElement('a');
	a.href = url;
	a.download = filename;
	document.body.appendChild(a);
	a.click();
	document.body.removeChild(a);
	URL.revokeObjectURL(url);
}

export async function restoreBackup(file: File, mode: 'replace' | 'merge'): Promise<RestoreResult> {
	const formData = new FormData();
	formData.append('file', file);

	const response = await apiFetch(`${API_BASE}/backup/restore?mode=${mode}`, {
		method: 'POST',
		body: formData
	});

	if (!response.ok) {
		const text = await response.text();
		throw new Error(text || `API error: ${response.status}`);
	}

	return response.json();
}
