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

// Logs

export interface LogEntry {
	timestamp: string;
	level: 'DEBUG' | 'INFO' | 'WARN' | 'ERROR';
	source: string;
	message: string;
}

export interface LogsResponse {
	entries: LogEntry[];
	total: number;
	hasMore: boolean;
}

export interface LogsQuery {
	level?: string;
	source?: string;
	search?: string;
	limit?: number;
}

export async function getLogs(query: LogsQuery = {}): Promise<LogsResponse> {
	const params = new URLSearchParams();
	if (query.level) params.set('level', query.level);
	if (query.source) params.set('source', query.source);
	if (query.search) params.set('search', query.search);
	if (query.limit) params.set('limit', String(query.limit));

	const url = `${API_BASE}/logs${params.toString() ? '?' + params.toString() : ''}`;
	const response = await apiFetch(url);
	if (!response.ok) throw new Error(`API error: ${response.status}`);
	return response.json();
}

export async function downloadLogs(): Promise<void> {
	const response = await apiFetch(`${API_BASE}/logs/download`);
	if (!response.ok) throw new Error(`API error: ${response.status}`);

	const blob = await response.blob();
	const url = URL.createObjectURL(blob);
	const a = document.createElement('a');
	a.href = url;
	a.download = 'outpost-logs.txt';
	document.body.appendChild(a);
	a.click();
	document.body.removeChild(a);
	URL.revokeObjectURL(url);
}

// Storage Analytics

export interface LibrarySize {
	id: number;
	name: string;
	type: string;
	size: number;
	count: number;
}

export interface QualitySize {
	quality: string;
	size: number;
	count: number;
}

export interface YearSize {
	year: number;
	size: number;
	count: number;
}

export interface LargestItem {
	id: number;
	type: 'movie' | 'episode';
	title: string;
	year: number;
	size: number;
	quality: string;
	path: string;
}

export interface DuplicateCopy {
	id: number;
	quality: string;
	size: number;
	path: string;
}

export interface DuplicateItem {
	tmdbId: number;
	title: string;
	year: number;
	type: string;
	copies: DuplicateCopy[];
}

export interface StorageAnalytics {
	total: number;
	used: number;
	free: number;
	byLibrary: LibrarySize[];
	byQuality: QualitySize[];
	byYear: YearSize[];
	largest: LargestItem[];
	duplicates: DuplicateItem[];
}

export async function getStorageAnalytics(): Promise<StorageAnalytics> {
	const response = await apiFetch(`${API_BASE}/storage/analytics`);
	if (!response.ok) throw new Error(`API error: ${response.status}`);
	return response.json();
}

// Health Checks

export type HealthCheckStatus = 'healthy' | 'warning' | 'unhealthy';

export interface HealthCheck {
	name: string;
	status: HealthCheckStatus;
	message: string;
	latency?: number;
	lastCheck: string;
	error?: string;
}

export interface FullHealthResponse {
	overall: HealthCheckStatus;
	checks: HealthCheck[];
	lastFullCheck: string;
}

export async function getHealthFull(): Promise<FullHealthResponse> {
	const response = await apiFetch(`${API_BASE}/health/full`);
	if (!response.ok) throw new Error(`API error: ${response.status}`);
	return response.json();
}

export async function recheckHealth(name: string): Promise<HealthCheck> {
	const response = await apiFetch(`${API_BASE}/health/check/${encodeURIComponent(name)}`, {
		method: 'POST'
	});
	if (!response.ok) throw new Error(`API error: ${response.status}`);
	return response.json();
}
