/**
 * Search utilities with status bar integration
 * Wraps search API calls to report progress to the status bar
 */

import { searchIndexers, searchIndexersScored, getIndexers, type SearchParams, type SearchResult, type ScoredSearchResult } from '$lib/api';
import { statusBar } from '$lib/stores/statusBar';

/**
 * Search indexers with status bar progress reporting
 */
export async function searchWithProgress(params: SearchParams): Promise<SearchResult[]> {
	// Get enabled indexers to show in status
	let indexerNames: string[] = [];
	try {
		const indexers = await getIndexers();
		indexerNames = indexers.filter(i => i.enabled).map(i => i.name);
	} catch {
		indexerNames = ['Indexers'];
	}

	// Start search in status bar
	statusBar.startSearch(params.q || 'search', indexerNames);

	try {
		// Simulate per-indexer progress (the actual API doesn't report per-indexer)
		// Mark all as searching
		for (const name of indexerNames) {
			statusBar.updateIndexer(name, 'searching');
		}

		const results = await searchIndexers(params);

		// Mark all as complete
		const perIndexer = Math.ceil(results.length / Math.max(indexerNames.length, 1));
		for (const name of indexerNames) {
			statusBar.updateIndexer(name, 'success', perIndexer);
		}

		// End search
		statusBar.endSearch(results.length);

		// Log activity event
		statusBar.addEvent('search', `Search completed: "${params.q}"`, `Found ${results.length} results`);

		return results;
	} catch (error) {
		// Mark all as failed
		for (const name of indexerNames) {
			statusBar.updateIndexer(name, 'failed', 0, error instanceof Error ? error.message : 'Unknown error');
		}
		statusBar.endSearch(0);
		statusBar.addEvent('error', `Search failed: "${params.q}"`, error instanceof Error ? error.message : 'Unknown error');
		throw error;
	}
}

/**
 * Search indexers with scoring and status bar progress reporting
 */
export async function searchScoredWithProgress(params: SearchParams): Promise<ScoredSearchResult[]> {
	// Get enabled indexers to show in status
	let indexerNames: string[] = [];
	try {
		const indexers = await getIndexers();
		indexerNames = indexers.filter(i => i.enabled).map(i => i.name);
	} catch {
		indexerNames = ['Indexers'];
	}

	// Start search in status bar
	statusBar.startSearch(params.q || 'search', indexerNames);

	try {
		// Mark all as searching
		for (const name of indexerNames) {
			statusBar.updateIndexer(name, 'searching');
		}

		const results = await searchIndexersScored(params);

		// Set to scoring state briefly
		statusBar.setSearchState('scoring');

		// Mark all as complete
		const perIndexer = Math.ceil(results.length / Math.max(indexerNames.length, 1));
		for (const name of indexerNames) {
			statusBar.updateIndexer(name, 'success', perIndexer);
		}

		// End search
		statusBar.endSearch(results.length);

		// Log activity event
		statusBar.addEvent('search', `Scored search completed: "${params.q}"`, `Found ${results.length} scored results`);

		return results;
	} catch (error) {
		// Mark all as failed
		for (const name of indexerNames) {
			statusBar.updateIndexer(name, 'failed', 0, error instanceof Error ? error.message : 'Unknown error');
		}
		statusBar.endSearch(0);
		statusBar.addEvent('error', `Search failed: "${params.q}"`, error instanceof Error ? error.message : 'Unknown error');
		throw error;
	}
}

/**
 * Log a grab event to the status bar
 */
export function logGrab(title: string, indexerName?: string) {
	statusBar.addEvent('grab', `Grabbed: ${title}`, indexerName ? `From ${indexerName}` : undefined, '/activity');
}

/**
 * Log an import event to the status bar
 */
export function logImport(title: string, path?: string) {
	statusBar.addEvent('import', `Imported: ${title}`, path, '/activity');
}

/**
 * Log an approval event to the status bar
 */
export function logApproval(title: string, user?: string) {
	statusBar.addEvent('approval', `Approved: ${title}`, user ? `By ${user}` : undefined, '/requests');
}

/**
 * Log an error event to the status bar
 */
export function logError(message: string, details?: string) {
	statusBar.addEvent('error', message, details);
}

/**
 * Log a system event to the status bar
 */
export function logSystem(message: string, details?: string) {
	statusBar.addEvent('system', message, details);
}
