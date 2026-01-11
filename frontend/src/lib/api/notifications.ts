import { API_BASE, apiFetch } from './core';

export interface Notification {
	id: number;
	userId: number;
	type: 'new_content' | 'request_approved' | 'request_denied' | 'download_complete' | 'download_failed';
	title: string;
	message: string;
	imageUrl: string | null;
	link: string | null;
	read: boolean;
	createdAt: string;
}

export async function getNotifications(unreadOnly = false, limit = 50): Promise<Notification[]> {
	const params = new URLSearchParams();
	if (unreadOnly) params.set('unread', 'true');
	if (limit) params.set('limit', limit.toString());

	const response = await apiFetch(`${API_BASE}/notifications?${params}`);
	if (!response.ok) {
		throw new Error(`API error: ${response.status}`);
	}
	return response.json();
}

export async function getUnreadCount(): Promise<number> {
	const response = await apiFetch(`${API_BASE}/notifications/unread-count`);
	if (!response.ok) {
		throw new Error(`API error: ${response.status}`);
	}
	const data = await response.json();
	return data.count;
}

export async function markRead(id: number): Promise<void> {
	const response = await apiFetch(`${API_BASE}/notifications/${id}/read`, {
		method: 'POST'
	});
	if (!response.ok) {
		throw new Error(`API error: ${response.status}`);
	}
}

export async function markAllRead(): Promise<void> {
	const response = await apiFetch(`${API_BASE}/notifications/read-all`, {
		method: 'POST'
	});
	if (!response.ok) {
		throw new Error(`API error: ${response.status}`);
	}
}

export async function deleteNotification(id: number): Promise<void> {
	const response = await apiFetch(`${API_BASE}/notifications/${id}`, {
		method: 'DELETE'
	});
	if (!response.ok) {
		throw new Error(`API error: ${response.status}`);
	}
}
