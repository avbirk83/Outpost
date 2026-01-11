/**
 * Normalize text for search comparison
 * - Converts to lowercase
 * - Removes diacritics (accents)
 * - Removes special characters
 * - Normalizes whitespace
 */
export function normalizeText(text: string): string {
	return text
		.toLowerCase()
		.normalize('NFD')
		.replace(/[\u0300-\u036f]/g, '')
		.replace(/[^a-z0-9\s]/g, ' ')
		.replace(/\s+/g, ' ')
		.trim();
}

/**
 * Calculate a search relevance score between 0-100
 * Higher scores indicate better matches
 */
export function searchScore(text: string, searchQuery: string): number {
	if (!text || !searchQuery) return 0;
	const t = normalizeText(text);
	const q = normalizeText(searchQuery);
	if (!t || !q) return 0;
	if (t === q) return 100;
	if (t.startsWith(q)) return 95;
	if (q.startsWith(t)) return 90;
	if (t.includes(q)) return 85;

	const textWords = t.split(' ').filter((w) => w.length > 0);
	const queryWords = q.split(' ').filter((w) => w.length > 0);

	// Check if any word starts with the query
	for (const word of textWords) {
		if (word.startsWith(q)) return 80;
	}

	// Check if query starts with any word
	for (const word of textWords) {
		if (word.length >= 2 && q.startsWith(word)) return 75;
	}

	// Multi-word query matching
	if (queryWords.length > 1) {
		let allMatch = true;
		let matchCount = 0;
		for (const qWord of queryWords) {
			const found = textWords.some((tWord) => tWord.includes(qWord) || qWord.includes(tWord));
			if (found) matchCount++;
			else allMatch = false;
		}
		if (allMatch) return 70;
		if (matchCount > 0) return 50 + (matchCount / queryWords.length) * 20;
	}

	// Check if any word contains the query
	for (const word of textWords) {
		if (word.includes(q)) return 60;
	}

	// Check if query contains any word
	for (const word of textWords) {
		if (word.length >= 3 && q.includes(word)) return 55;
	}

	// Subsequence matching
	let matchLen = 0;
	let tIdx = 0;
	for (let i = 0; i < q.length && tIdx < t.length; i++) {
		const foundIdx = t.indexOf(q[i], tIdx);
		if (foundIdx !== -1) {
			matchLen++;
			tIdx = foundIdx + 1;
		}
	}
	const seqScore = (matchLen / q.length) * 40;
	if (seqScore >= 30) return seqScore;

	return 0;
}
