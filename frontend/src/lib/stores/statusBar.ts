/**
 * Status Bar Store
 * Global state for the status bar including search activity, events, and system info
 */

import { writable, derived, get } from 'svelte/store';

// Search state types
export type SearchState = 'idle' | 'searching' | 'scoring';
export type IndexerStatus = 'pending' | 'searching' | 'success' | 'failed' | 'skipped';

export interface IndexerResult {
	id: number;
	name: string;
	status: IndexerStatus;
	resultCount: number;
	error?: string;
}

export interface SearchProgress {
	state: SearchState;
	query: string;
	indexers: IndexerResult[];
	totalResults: number;
	startTime: number;
}

// Activity event types
export type EventType = 'import' | 'grab' | 'approval' | 'error' | 'search' | 'download' | 'system';

export interface ActivityEvent {
	id: number;
	type: EventType;
	message: string;
	details?: string;
	timestamp: Date;
	link?: string;
}

// Download client status
export interface DownloadClientStatus {
	connected: boolean;
	name: string;
	downloadSpeed: number; // bytes per second
	uploadSpeed: number;
	activeCount: number;
	queuedCount: number;
}

// Full status bar state
interface StatusBarState {
	search: SearchProgress;
	events: ActivityEvent[];
	downloadClient: DownloadClientStatus | null;
	expanded: boolean;
}

const initialState: StatusBarState = {
	search: {
		state: 'idle',
		query: '',
		indexers: [],
		totalResults: 0,
		startTime: 0
	},
	events: [],
	downloadClient: null,
	expanded: false
};

function createStatusBarStore() {
	const { subscribe, update, set } = writable<StatusBarState>(initialState);
	let eventId = 0;

	return {
		subscribe,

		// Search methods
		startSearch: (query: string, indexerNames: string[]) => {
			update(state => ({
				...state,
				search: {
					state: 'searching',
					query,
					indexers: indexerNames.map((name, i) => ({
						id: i,
						name,
						status: 'pending' as IndexerStatus,
						resultCount: 0
					})),
					totalResults: 0,
					startTime: Date.now()
				}
			}));
		},

		updateIndexer: (name: string, status: IndexerStatus, resultCount: number = 0, error?: string) => {
			update(state => ({
				...state,
				search: {
					...state.search,
					indexers: state.search.indexers.map(idx =>
						idx.name === name ? { ...idx, status, resultCount, error } : idx
					),
					totalResults: state.search.indexers.reduce((sum, idx) =>
						sum + (idx.name === name ? resultCount : idx.resultCount), 0
					)
				}
			}));
		},

		setSearchState: (searchState: SearchState) => {
			update(state => ({
				...state,
				search: { ...state.search, state: searchState }
			}));
		},

		endSearch: (totalResults: number) => {
			update(state => ({
				...state,
				search: {
					...state.search,
					state: 'idle',
					totalResults
				}
			}));

			// Auto-clear after showing results briefly
			setTimeout(() => {
				update(state => ({
					...state,
					search: { ...initialState.search }
				}));
			}, 3000);
		},

		clearSearch: () => {
			update(state => ({
				...state,
				search: { ...initialState.search }
			}));
		},

		// Event methods
		addEvent: (type: EventType, message: string, details?: string, link?: string) => {
			const event: ActivityEvent = {
				id: eventId++,
				type,
				message,
				details,
				timestamp: new Date(),
				link
			};

			update(state => ({
				...state,
				events: [event, ...state.events].slice(0, 50) // Keep last 50 events
			}));

			return event.id;
		},

		clearEvents: () => {
			update(state => ({ ...state, events: [] }));
		},

		// Download client methods
		updateDownloadClient: (status: DownloadClientStatus | null) => {
			update(state => ({ ...state, downloadClient: status }));
		},

		// UI methods
		toggleExpanded: () => {
			update(state => ({ ...state, expanded: !state.expanded }));
		},

		setExpanded: (expanded: boolean) => {
			update(state => ({ ...state, expanded }));
		},

		// Reset
		reset: () => set(initialState)
	};
}

export const statusBar = createStatusBarStore();

// Derived stores for specific data
export const searchProgress = derived(statusBar, $state => $state.search);
export const activityEvents = derived(statusBar, $state => $state.events);
export const downloadClientStatus = derived(statusBar, $state => $state.downloadClient);
export const isExpanded = derived(statusBar, $state => $state.expanded);

// Helper to get current search stats
export const searchStats = derived(statusBar, $state => {
	const { indexers } = $state.search;
	return {
		total: indexers.length,
		completed: indexers.filter(i => i.status === 'success' || i.status === 'failed' || i.status === 'skipped').length,
		success: indexers.filter(i => i.status === 'success').length,
		failed: indexers.filter(i => i.status === 'failed').length,
		skipped: indexers.filter(i => i.status === 'skipped').length,
		pending: indexers.filter(i => i.status === 'pending' || i.status === 'searching').length
	};
});
