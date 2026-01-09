import { API_BASE, apiFetch } from './core';

// Storage Management

export interface DiskUsage {
	total: number;
	free: number;
	used: number;
	usedPercent: number;
}

export interface StorageStatus {
	thresholdGb: number;
	pauseEnabled: boolean;
	upgradeDeleteOld: boolean;
	moviesSize: number;
	tvSize: number;
	musicSize: number;
	booksSize: number;
	diskUsage?: DiskUsage;
}

export async function getStorageStatus(): Promise<StorageStatus> {
	const response = await apiFetch(`${API_BASE}/storage/status`);
	if (!response.ok) throw new Error(`API error: ${response.status}`);
	return response.json();
}

// System Status

export interface SystemStatus {
	pendingRequests: number;
	activeDownloads: number;
	runningTasks: string[];
	activeSearch: string;
	diskUsed: number;
	diskTotal: number;
}

export async function getSystemStatus(): Promise<SystemStatus> {
	const response = await apiFetch(`${API_BASE}/system/status`);
	if (!response.ok) throw new Error(`API error: ${response.status}`);
	return response.json();
}

// Task/Scheduler

export interface ScheduledTask {
	id: number;
	name: string;
	description: string;
	taskType: string;
	enabled: boolean;
	intervalMinutes: number;
	lastRun: string | null;
	nextRun: string | null;
	lastDurationMs: number | null;
	lastStatus: string;
	lastError: string | null;
	runCount: number;
	failCount: number;
	isRunning: boolean;
}

export interface TaskHistory {
	id: number;
	taskId: number;
	taskName: string;
	startedAt: string;
	finishedAt: string | null;
	durationMs: number | null;
	status: string;
	itemsProcessed: number;
	itemsFound: number;
	error: string | null;
}

export async function getTasks(): Promise<ScheduledTask[]> {
	const response = await apiFetch(`${API_BASE}/tasks`);
	if (!response.ok) throw new Error(`API error: ${response.status}`);
	return response.json();
}

export async function getTask(id: number): Promise<{ task: ScheduledTask; history: TaskHistory[] }> {
	const response = await apiFetch(`${API_BASE}/tasks/${id}`);
	if (!response.ok) throw new Error(`API error: ${response.status}`);
	return response.json();
}

export async function updateTask(id: number, enabled: boolean, intervalMinutes: number): Promise<ScheduledTask> {
	const response = await apiFetch(`${API_BASE}/tasks/${id}`, {
		method: 'PUT',
		headers: { 'Content-Type': 'application/json' },
		body: JSON.stringify({ enabled, intervalMinutes })
	});
	if (!response.ok) throw new Error(`API error: ${response.status}`);
	return response.json();
}

export async function triggerTask(id: number): Promise<void> {
	const response = await apiFetch(`${API_BASE}/tasks/${id}/trigger`, { method: 'POST' });
	if (!response.ok) throw new Error(`API error: ${response.status}`);
}

export async function getTaskHistory(limit = 50): Promise<TaskHistory[]> {
	const response = await apiFetch(`${API_BASE}/tasks/history?limit=${limit}`);
	if (!response.ok) throw new Error(`API error: ${response.status}`);
	return response.json();
}
