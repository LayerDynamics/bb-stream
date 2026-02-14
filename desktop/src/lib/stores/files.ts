import { writable, derived } from 'svelte/store';
import type { BucketInfo, ObjectInfo } from '../api';

// Current bucket selection
export const currentBucket = writable<string | null>(null);

// Current path within bucket
export const currentPath = writable<string>('');

// List of buckets
export const buckets = writable<BucketInfo[]>([]);

// Files in current bucket/path
export const files = writable<ObjectInfo[]>([]);

// Loading states
export const isLoadingBuckets = writable(false);
export const isLoadingFiles = writable(false);

// Error state
export const error = writable<string | null>(null);

// Derived store for breadcrumb navigation
export const breadcrumbs = derived(
  [currentBucket, currentPath],
  ([$currentBucket, $currentPath]) => {
    const parts: { name: string; path: string }[] = [];

    if ($currentBucket) {
      parts.push({ name: $currentBucket, path: '' });

      if ($currentPath) {
        const pathParts = $currentPath.split('/').filter(Boolean);
        let accumulated = '';
        for (const part of pathParts) {
          accumulated = accumulated ? `${accumulated}/${part}` : part;
          parts.push({ name: part, path: accumulated });
        }
      }
    }

    return parts;
  }
);

// Selected files for batch operations
export const selectedFiles = writable<Set<string>>(new Set());

// Clear selection
export function clearSelection() {
  selectedFiles.set(new Set());
}

// Toggle file selection
export function toggleSelection(fileName: string) {
  selectedFiles.update((selected) => {
    const newSet = new Set(selected);
    if (newSet.has(fileName)) {
      newSet.delete(fileName);
    } else {
      newSet.add(fileName);
    }
    return newSet;
  });
}

// Select all files
export function selectAll(fileNames: string[]) {
  selectedFiles.set(new Set(fileNames));
}
