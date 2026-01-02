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

// ============================================
// JSON Parsing Utilities
// ============================================

export interface CastMember {
	id?: number;
	name: string;
	character: string;
	profile_path?: string;
}

export interface CrewMember {
	id?: number;
	name: string;
	job: string;
	department: string;
	profile_path?: string;
}

/**
 * Parse genres JSON string to array
 */
export function parseGenres(g?: string): string[] {
	if (!g) return [];
	try { return JSON.parse(g); } catch { return []; }
}

/**
 * Parse cast JSON string to array
 */
export function parseCast(c?: string): CastMember[] {
	if (!c) return [];
	try { return JSON.parse(c); } catch { return []; }
}

/**
 * Parse crew JSON string to array
 */
export function parseCrew(c?: string): CrewMember[] {
	if (!c) return [];
	try { return JSON.parse(c); } catch { return []; }
}

// ============================================
// Display Formatting
// ============================================

/**
 * Format money with full currency format (e.g., "$1,234,567")
 */
export function formatMoneyFull(amount?: number): string {
	if (!amount || amount === 0) return '-';
	return new Intl.NumberFormat('en-US', {
		style: 'currency',
		currency: 'USD',
		maximumFractionDigits: 0
	}).format(amount);
}

/**
 * Format video resolution to human-readable string
 */
export function formatResolution(width?: number, height?: number): string {
	if (!width && !height) return '';
	const w = width || 0;
	const h = height || 0;
	if (w >= 3840 || h >= 2160) return '4K';
	if (w >= 1920 || h >= 1080) return '1080p';
	if (w >= 1280 || h >= 720) return '720p';
	if (h > 0) return `${h}p`;
	return '';
}

/**
 * Format audio channels to human-readable string
 */
export function formatAudioChannels(channels?: number): string {
	if (!channels) return '';
	if (channels >= 8) return '7.1';
	if (channels >= 6) return '5.1';
	if (channels === 2) return 'Stereo';
	return `${channels}ch`;
}

/**
 * Format file size in bytes to human-readable string
 */
export function formatFileSize(bytes?: number): string {
	if (!bytes) return '';
	if (bytes >= 1_000_000_000) return `${(bytes / 1_000_000_000).toFixed(1)} GB`;
	if (bytes >= 1_000_000) return `${(bytes / 1_000_000).toFixed(0)} MB`;
	return `${bytes} bytes`;
}

// ============================================
// Internationalization Helpers
// ============================================

/**
 * Get language name from ISO code
 */
export function getLanguageName(code?: string): string {
	if (!code || code === 'und') return 'Unknown';
	try {
		const displayNames = new Intl.DisplayNames(['en'], { type: 'language' });
		return displayNames.of(code) || code;
	} catch {
		return code;
	}
}

/**
 * Get country name from ISO code
 */
export function getCountryName(code?: string): string {
	if (!code) return '';
	try {
		const displayNames = new Intl.DisplayNames(['en'], { type: 'region' });
		return displayNames.of(code.toUpperCase()) || code;
	} catch {
		return code;
	}
}

/**
 * Get country flag emoji from ISO code
 */
export function getCountryFlag(code?: string): string {
	if (!code || code.length !== 2) return '';
	return code.toUpperCase().split('').map(c => String.fromCodePoint(127397 + c.charCodeAt(0))).join('');
}

// ============================================
// Status Helpers
// ============================================

/**
 * Get Tailwind color class for media status
 */
export function getStatusColor(status?: string): string {
	switch (status?.toLowerCase()) {
		case 'released':
			return 'text-green-400';
		case 'in production':
		case 'post production':
			return 'text-yellow-400';
		case 'planned':
		case 'rumored':
			return 'text-text-muted';
		case 'ended':
			return 'text-text-muted';
		case 'returning series':
			return 'text-green-400';
		case 'canceled':
			return 'text-red-400';
		default:
			return 'text-green-400';
	}
}
