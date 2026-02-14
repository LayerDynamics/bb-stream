import { writable } from 'svelte/store';

export interface Upload {
  id: string;
  fileName: string;
  bucket: string;
  path: string;
  progress: number;
  status: 'pending' | 'uploading' | 'complete' | 'error';
  error?: string;
}

export interface Download {
  id: string;
  fileName: string;
  bucket: string;
  path: string;
  progress: number;
  status: 'pending' | 'downloading' | 'complete' | 'error';
  error?: string;
}

export interface SyncJobInfo {
  id: string;
  localPath: string;
  bucket: string;
  remotePath: string;
  direction: 'to_remote' | 'to_local';
  status: 'running' | 'completed' | 'failed' | 'stopped';
  progress?: string;
}

export interface WatchJobInfo {
  id: string;
  localPath: string;
  bucket: string;
  remotePath: string;
  status: 'running' | 'stopped';
  recentUploads: string[];
}

// Active uploads
export const uploads = writable<Upload[]>([]);

// Active downloads
export const downloads = writable<Download[]>([]);

// Sync jobs
export const syncJobs = writable<SyncJobInfo[]>([]);

// Watch jobs
export const watchJobs = writable<WatchJobInfo[]>([]);

// Add a new upload
let uploadIdCounter = 0;
export function addUpload(fileName: string, bucket: string, path: string): string {
  const id = `upload-${++uploadIdCounter}`;
  uploads.update((list) => [
    ...list,
    {
      id,
      fileName,
      bucket,
      path,
      progress: 0,
      status: 'pending',
    },
  ]);
  return id;
}

// Update upload progress
export function updateUploadProgress(id: string, progress: number) {
  uploads.update((list) =>
    list.map((u) =>
      u.id === id ? { ...u, progress, status: 'uploading' as const } : u
    )
  );
}

// Mark upload complete
export function completeUpload(id: string) {
  uploads.update((list) =>
    list.map((u) =>
      u.id === id ? { ...u, progress: 100, status: 'complete' as const } : u
    )
  );
}

// Mark upload error
export function failUpload(id: string, error: string) {
  uploads.update((list) =>
    list.map((u) =>
      u.id === id ? { ...u, status: 'error' as const, error } : u
    )
  );
}

// Remove upload from list
export function removeUpload(id: string) {
  uploads.update((list) => list.filter((u) => u.id !== id));
}

// Clear completed uploads
export function clearCompletedUploads() {
  uploads.update((list) => list.filter((u) => u.status !== 'complete'));
}

// Add a new download
let downloadIdCounter = 0;
export function addDownload(fileName: string, bucket: string, path: string): string {
  const id = `download-${++downloadIdCounter}`;
  downloads.update((list) => [
    ...list,
    {
      id,
      fileName,
      bucket,
      path,
      progress: 0,
      status: 'pending',
    },
  ]);
  return id;
}

// Update download progress
export function updateDownloadProgress(id: string, progress: number) {
  downloads.update((list) =>
    list.map((d) =>
      d.id === id ? { ...d, progress, status: 'downloading' as const } : d
    )
  );
}

// Mark download complete
export function completeDownload(id: string) {
  downloads.update((list) =>
    list.map((d) =>
      d.id === id ? { ...d, progress: 100, status: 'complete' as const } : d
    )
  );
}

// Mark download error
export function failDownload(id: string, error: string) {
  downloads.update((list) =>
    list.map((d) =>
      d.id === id ? { ...d, status: 'error' as const, error } : d
    )
  );
}

// Remove download from list
export function removeDownload(id: string) {
  downloads.update((list) => list.filter((d) => d.id !== id));
}

// Clear completed downloads
export function clearCompletedDownloads() {
  downloads.update((list) => list.filter((d) => d.status !== 'complete'));
}

// Add sync job
export function addSyncJob(job: SyncJobInfo) {
  syncJobs.update((list) => [...list, job]);
}

// Update sync job
export function updateSyncJob(id: string, updates: Partial<SyncJobInfo>) {
  syncJobs.update((list) =>
    list.map((j) => (j.id === id ? { ...j, ...updates } : j))
  );
}

// Remove sync job
export function removeSyncJob(id: string) {
  syncJobs.update((list) => list.filter((j) => j.id !== id));
}

// Add watch job
export function addWatchJob(job: WatchJobInfo) {
  watchJobs.update((list) => [...list, job]);
}

// Update watch job
export function updateWatchJob(id: string, updates: Partial<WatchJobInfo>) {
  watchJobs.update((list) =>
    list.map((j) => (j.id === id ? { ...j, ...updates } : j))
  );
}

// Remove watch job
export function removeWatchJob(id: string) {
  watchJobs.update((list) => list.filter((j) => j.id !== id));
}
