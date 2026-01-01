/**
 * Shared formatting utilities
 * Available: getYear(), formatRuntime(), formatTime(), formatMoney(), formatTimeLeft()
 */

/**
 * Extract year from a date string (e.g., "2024-05-15" -> "2024")
 */
export function getYear(dateStr: string | undefined): string {
	if (!dateStr) return '';
	return dateStr.substring(0, 4);
}

/**
 * Format runtime in minutes to "Xh Ym" format
 */
export function formatRuntime(minutes: number | undefined): string {
	if (!minutes) return '';
	const h = Math.floor(minutes / 60);
	const m = minutes % 60;
	return h > 0 ? `${h}h ${m}m` : `${m}m`;
}

/**
 * Format seconds to "H:MM:SS" or "M:SS" format
 */
export function formatTime(seconds: number): string {
	const h = Math.floor(seconds / 3600);
	const m = Math.floor((seconds % 3600) / 60);
	const s = Math.floor(seconds % 60);
	if (h > 0) {
		return `${h}:${m.toString().padStart(2, '0')}:${s.toString().padStart(2, '0')}`;
	}
	return `${m}:${s.toString().padStart(2, '0')}`;
}

/**
 * Format money amount to "$X.XM" or "$X.XB" format
 */
export function formatMoney(amount: number | undefined): string {
	if (!amount) return '';
	if (amount >= 1_000_000_000) {
		return `$${(amount / 1_000_000_000).toFixed(1)}B`;
	}
	if (amount >= 1_000_000) {
		return `$${(amount / 1_000_000).toFixed(1)}M`;
	}
	return `$${amount.toLocaleString()}`;
}

/**
 * Format time remaining from position and duration
 */
export function formatTimeLeft(position: number, duration: number): string {
	const remaining = Math.max(0, duration - position);
	const minutes = Math.floor(remaining / 60);
	if (minutes >= 60) {
		const hours = Math.floor(minutes / 60);
		const mins = minutes % 60;
		return `${hours}h ${mins}m left`;
	}
	return `${minutes}m left`;
}
