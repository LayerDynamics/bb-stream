// API client for communicating with the Go backend
import { fetch as tauriFetch } from '@tauri-apps/plugin-http';
import { invoke } from '@tauri-apps/api/core';

// Expected backend version for compatibility check
const EXPECTED_VERSION = '0.1.0';
const EXPECTED_API_VERSION = 1;

// Dynamic port management
let apiPort: number | null = null;

async function getApiPort(): Promise<number> {
  if (apiPort !== null) {
    return apiPort;
  }
  try {
    apiPort = await invoke<number>('get_api_port');
    return apiPort;
  } catch {
    // Fallback for development
    return 8765;
  }
}

async function getApiBase(): Promise<string> {
  const port = await getApiPort();
  return `http://localhost:${port}/api`;
}

// Synchronous version for XHR calls - uses cached port or fallback
function getApiBaseSync(): string {
  const port = apiPort ?? 8765;
  return `http://localhost:${port}/api`;
}

// Reset port cache (useful when backend restarts)
export function resetApiPort(): void {
  apiPort = null;
}

// Initialize the port cache (call this early in app startup)
export async function initApiPort(): Promise<number> {
  return getApiPort();
}

// Use Tauri's fetch for cross-origin requests in webview
const safeFetch = async (url: string, options?: RequestInit): Promise<Response> => {
  try {
    // Try Tauri fetch first (works in Tauri app)
    return await tauriFetch(url, options as any);
  } catch {
    // Fall back to browser fetch (for dev/testing)
    return await fetch(url, options);
  }
};

export interface BucketInfo {
  Name: string;
  Type: string;
}

export interface ObjectInfo {
  Name: string;
  Size: number;
  ContentType: string;
  Timestamp: number;
}

export interface UploadResult {
  Name: string;
  Size: number;
  ContentType: string;
}

export interface SyncJob {
  id: string;
  status: string;
  local_path: string;
  bucket: string;
  path: string;
  direction: string;
  progress?: string;
}

export interface Job {
  id: string;
  type: string;
  status: string;
}

export interface ConfigResponse {
  key_id: string;
  has_app_key: boolean;
  default_bucket: string;
  configured: boolean;
}

export interface ConfigRequest {
  key_id?: string;
  application_key?: string;
  default_bucket?: string;
}

export interface VersionInfo {
  version: string;
  api_version: number;
}

export interface StatusInfo {
  version: string;
  api_version: number;
  uptime_seconds: number;
  active_sync_jobs: number;
  active_watch_jobs: number;
  websocket_clients: number;
}

class ApiClient {
  private async getBaseUrl(): Promise<string> {
    return getApiBase();
  }

  private async request<T>(
    endpoint: string,
    options: RequestInit = {}
  ): Promise<T> {
    const baseUrl = await this.getBaseUrl();
    const url = `${baseUrl}${endpoint}`;
    const response = await safeFetch(url, {
      ...options,
      headers: {
        'Content-Type': 'application/json',
        ...options.headers,
      },
    });

    if (!response.ok) {
      const error = await response.json().catch(() => ({ error: 'Unknown error' }));
      throw new Error(error.error || `HTTP ${response.status}`);
    }

    return response.json();
  }

  // Health check
  async health(): Promise<boolean> {
    try {
      const baseUrl = await this.getBaseUrl();
      const response = await safeFetch(`${baseUrl.replace('/api', '')}/health`);
      return response.ok;
    } catch {
      return false;
    }
  }

  // Version check
  async getVersion(): Promise<VersionInfo> {
    return this.request<VersionInfo>('/version');
  }

  // Status
  async getStatus(): Promise<StatusInfo> {
    return this.request<StatusInfo>('/status');
  }

  // Check version compatibility
  async checkVersionCompatibility(): Promise<{ compatible: boolean; message?: string }> {
    try {
      const info = await this.getVersion();
      if (info.api_version !== EXPECTED_API_VERSION) {
        return {
          compatible: false,
          message: `API version mismatch: expected ${EXPECTED_API_VERSION}, got ${info.api_version}`
        };
      }
      if (info.version !== EXPECTED_VERSION) {
        return {
          compatible: true,
          message: `Backend version (${info.version}) differs from frontend (${EXPECTED_VERSION})`
        };
      }
      return { compatible: true };
    } catch (e) {
      return { compatible: false, message: `Failed to check version: ${e}` };
    }
  }

  // Buckets
  async listBuckets(): Promise<BucketInfo[]> {
    return this.request<BucketInfo[]>('/buckets');
  }

  async listFiles(bucket: string, prefix: string = ''): Promise<ObjectInfo[]> {
    const params = prefix ? `?prefix=${encodeURIComponent(prefix)}` : '';
    return this.request<ObjectInfo[]>(`/buckets/${bucket}/files${params}`);
  }

  // Upload
  async uploadFile(
    bucket: string,
    path: string,
    file: File,
    onProgress?: (percent: number) => void
  ): Promise<UploadResult> {
    const formData = new FormData();
    formData.append('file', file);

    const xhr = new XMLHttpRequest();

    return new Promise((resolve, reject) => {
      xhr.upload.addEventListener('progress', (e) => {
        if (e.lengthComputable && onProgress) {
          onProgress((e.loaded / e.total) * 100);
        }
      });

      xhr.addEventListener('load', () => {
        if (xhr.status >= 200 && xhr.status < 300) {
          resolve(JSON.parse(xhr.responseText));
        } else {
          reject(new Error(`Upload failed: ${xhr.statusText}`));
        }
      });

      xhr.addEventListener('error', () => {
        reject(new Error('Upload failed'));
      });

      xhr.open('POST', `${getApiBaseSync()}/upload?bucket=${bucket}&path=${path}`);
      xhr.send(formData);
    });
  }

  // Download URL
  getDownloadUrl(bucket: string, path: string): string {
    return `${getApiBaseSync()}/download/${bucket}/${path}`;
  }

  // Download with progress tracking
  downloadFileWithProgress(
    bucket: string,
    path: string,
    onProgress?: (percent: number, loaded: number, total: number) => void
  ): { promise: Promise<Blob>; cancel: () => void } {
    const xhr = new XMLHttpRequest();
    let cancelled = false;

    const promise = new Promise<Blob>((resolve, reject) => {
      xhr.responseType = 'blob';

      xhr.addEventListener('progress', (e) => {
        if (e.lengthComputable && onProgress) {
          onProgress((e.loaded / e.total) * 100, e.loaded, e.total);
        }
      });

      xhr.addEventListener('load', () => {
        if (xhr.status >= 200 && xhr.status < 300) {
          resolve(xhr.response);
        } else {
          reject(new Error(`Download failed: ${xhr.statusText}`));
        }
      });

      xhr.addEventListener('error', () => {
        if (cancelled) {
          reject(new Error('Download cancelled'));
        } else {
          reject(new Error('Download failed'));
        }
      });

      xhr.addEventListener('abort', () => {
        reject(new Error('Download cancelled'));
      });

      xhr.open('GET', this.getDownloadUrl(bucket, path));
      xhr.send();
    });

    const cancel = () => {
      cancelled = true;
      xhr.abort();
    };

    return { promise, cancel };
  }

  // Delete - use XMLHttpRequest to ensure DELETE method works
  async deleteFile(bucket: string, path: string): Promise<void> {
    return new Promise((resolve, reject) => {
      const xhr = new XMLHttpRequest();
      // Encode each path segment separately to handle special characters
      const encodedPath = path.split('/').map(encodeURIComponent).join('/');
      const url = `${getApiBaseSync()}/delete/${encodeURIComponent(bucket)}/${encodedPath}`;

      xhr.addEventListener('load', () => {
        if (xhr.status >= 200 && xhr.status < 300) {
          resolve();
        } else {
          try {
            const error = JSON.parse(xhr.responseText);
            reject(new Error(error.error || `Delete failed: ${xhr.statusText}`));
          } catch {
            reject(new Error(`Delete failed: ${xhr.statusText}`));
          }
        }
      });

      xhr.addEventListener('error', () => {
        reject(new Error('Delete request failed'));
      });

      xhr.open('DELETE', url);
      xhr.setRequestHeader('Content-Type', 'application/json');
      xhr.send();
    });
  }

  // Upload with cancel support
  uploadFileWithCancel(
    bucket: string,
    path: string,
    file: File,
    onProgress?: (percent: number) => void
  ): { promise: Promise<UploadResult>; cancel: () => void } {
    const xhr = new XMLHttpRequest();
    let cancelled = false;

    const promise = new Promise<UploadResult>((resolve, reject) => {
      xhr.upload.addEventListener('progress', (e) => {
        if (e.lengthComputable && onProgress) {
          onProgress((e.loaded / e.total) * 100);
        }
      });

      xhr.addEventListener('load', () => {
        if (xhr.status >= 200 && xhr.status < 300) {
          resolve(JSON.parse(xhr.responseText));
        } else {
          reject(new Error(`Upload failed: ${xhr.statusText}`));
        }
      });

      xhr.addEventListener('error', () => {
        if (cancelled) {
          reject(new Error('Upload cancelled'));
        } else {
          reject(new Error('Upload failed'));
        }
      });

      xhr.addEventListener('abort', () => {
        reject(new Error('Upload cancelled'));
      });

      const formData = new FormData();
      formData.append('file', file);
      xhr.open('POST', `${getApiBaseSync()}/upload?bucket=${bucket}&path=${path}`);
      xhr.send(formData);
    });

    const cancel = () => {
      cancelled = true;
      xhr.abort();
    };

    return { promise, cancel };
  }

  // Sync
  async startSync(
    localPath: string,
    bucket: string,
    remotePath: string,
    direction: 'to_remote' | 'to_local',
    options: { dryRun?: boolean; delete?: boolean } = {}
  ): Promise<{ job_id: string }> {
    return this.request('/sync/start', {
      method: 'POST',
      body: JSON.stringify({
        local_path: localPath,
        bucket,
        path: remotePath,
        direction,
        dry_run: options.dryRun || false,
        delete: options.delete || false,
      }),
    });
  }

  async getSyncStatus(jobId: string): Promise<SyncJob> {
    return this.request(`/sync/status/${jobId}`);
  }

  // Watch
  async startWatch(
    localPath: string,
    bucket: string,
    remotePath: string
  ): Promise<{ job_id: string }> {
    return this.request('/watch/start', {
      method: 'POST',
      body: JSON.stringify({
        local_path: localPath,
        bucket,
        path: remotePath,
      }),
    });
  }

  async stopWatch(jobId: string): Promise<void> {
    await this.request('/watch/stop', {
      method: 'POST',
      body: JSON.stringify({ job_id: jobId }),
    });
  }

  // Jobs
  async listJobs(): Promise<Job[]> {
    return this.request('/jobs');
  }

  // Config
  async getConfig(): Promise<ConfigResponse> {
    return this.request('/config');
  }

  async setConfig(config: ConfigRequest): Promise<{ status: string }> {
    return this.request('/config', {
      method: 'POST',
      body: JSON.stringify(config),
    });
  }
}

export const api = new ApiClient();
export default api;
