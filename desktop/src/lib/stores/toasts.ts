import { writable } from 'svelte/store';

export type ToastType = 'success' | 'error' | 'info' | 'warning';

export interface Toast {
  id: string;
  message: string;
  type: ToastType;
  duration: number;
}

// Toast store
export const toasts = writable<Toast[]>([]);

let toastIdCounter = 0;

// Add a toast notification
export function addToast(
  message: string,
  type: ToastType = 'info',
  duration: number = 3000
): string {
  const id = `toast-${++toastIdCounter}`;

  toasts.update((list) => [
    ...list,
    { id, message, type, duration },
  ]);

  // Auto-remove after duration
  if (duration > 0) {
    setTimeout(() => {
      removeToast(id);
    }, duration);
  }

  return id;
}

// Remove a toast by ID
export function removeToast(id: string) {
  toasts.update((list) => list.filter((t) => t.id !== id));
}

// Convenience functions
export function success(message: string, duration = 3000) {
  return addToast(message, 'success', duration);
}

export function error(message: string, duration = 5000) {
  return addToast(message, 'error', duration);
}

export function info(message: string, duration = 3000) {
  return addToast(message, 'info', duration);
}

export function warning(message: string, duration = 4000) {
  return addToast(message, 'warning', duration);
}

// Clear all toasts
export function clearAllToasts() {
  toasts.set([]);
}
