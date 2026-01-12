import { API_BASE, apiFetch } from './core';

export interface IntroDetectionResult {
	success: boolean;
	message: string;
	results: Array<{
		seasonId: number;
		seasonNumber: number;
		status: 'completed' | 'failed';
		error?: string;
	}>;
}

export async function detectShowIntros(id: number): Promise<IntroDetectionResult> {
	const response = await apiFetch(`${API_BASE}/shows/${id}/detect-intros`, {
		method: 'POST'
	});
	if (!response.ok) {
		const text = await response.text();
		throw new Error(text || `API error: ${response.status}`);
	}
	return response.json();
}
