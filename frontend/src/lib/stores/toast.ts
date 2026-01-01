/**
 * Toast notification store
 * Usage: import { toast } from '$lib/stores/toast';
 *        toast.success('Request submitted!');
 *        toast.error('Something went wrong');
 */

import { writable } from 'svelte/store';

export interface Toast {
	id: number;
	type: 'success' | 'error' | 'info';
	message: string;
}

function createToastStore() {
	const { subscribe, update } = writable<Toast[]>([]);
	let nextId = 0;

	function add(type: Toast['type'], message: string, duration = 3000) {
		const id = nextId++;
		update(toasts => [...toasts, { id, type, message }]);

		if (duration > 0) {
			setTimeout(() => remove(id), duration);
		}

		return id;
	}

	function remove(id: number) {
		update(toasts => toasts.filter(t => t.id !== id));
	}

	return {
		subscribe,
		success: (message: string, duration?: number) => add('success', message, duration),
		error: (message: string, duration?: number) => add('error', message, duration),
		info: (message: string, duration?: number) => add('info', message, duration),
		remove
	};
}

export const toast = createToastStore();
