<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  import { listen, type UnlistenFn } from '@tauri-apps/api/event';
  import api, { type BucketInfo, type ObjectInfo, initApiPort, resetApiPort } from './lib/api';
  import ws from './lib/websocket';
  import FileDropzone from './lib/components/FileDropzone.svelte';
  import FileList from './lib/components/FileList.svelte';
  import BucketSelector from './lib/components/BucketSelector.svelte';
  import UploadPanel from './lib/components/UploadPanel.svelte';
  import DownloadPanel from './lib/components/DownloadPanel.svelte';
  import SyncPanel from './lib/components/SyncPanel.svelte';
  import WatchPanel from './lib/components/WatchPanel.svelte';
  import ToastContainer from './lib/components/ToastContainer.svelte';
  import SettingsModal from './lib/components/SettingsModal.svelte';
  import ConfirmDialog from './lib/components/ConfirmDialog.svelte';
  import BackendStatusOverlay from './lib/components/BackendStatusOverlay.svelte';
  import { success, error as showError, info } from './lib/stores/toasts';
  import {
    uploads,
    downloads,
    addUpload,
    updateUploadProgress,
    completeUpload,
    failUpload,
    removeUpload,
    clearCompletedUploads,
    addDownload,
    updateDownloadProgress,
    completeDownload,
    failDownload,
    removeDownload,
    clearCompletedDownloads,
    syncJobs,
    watchJobs,
    addSyncJob,
    updateSyncJob,
    addWatchJob,
    updateWatchJob,
  } from './lib/stores/jobs';

  // Backend status type
  type BackendStatusType = 'starting' | 'healthy' | 'unhealthy' | 'crashed' | 'restarting';

  // State (using Svelte 5 runes for reactivity)
  let buckets = $state<BucketInfo[]>([]);
  let files = $state<ObjectInfo[]>([]);
  let currentBucket = $state<string | null>(null);
  let currentPath = $state('');
  let loadingBuckets = $state(false);
  let loadingFiles = $state(false);
  let serverConnected = $state(false);
  let error = $state<string | null>(null);
  let selectedFiles = $state(new Set<string>());
  let showSettings = $state(false);
  let deleteConfirm = $state<{ open: boolean; file: ObjectInfo | null }>({ open: false, file: null });
  let backendStatus = $state<BackendStatusType>('starting');
  let backendError = $state<string | undefined>(undefined);

  // Wait for server to be ready
  async function waitForServer(maxAttempts = 30): Promise<boolean> {
    for (let i = 0; i < maxAttempts; i++) {
      if (await api.health()) {
        return true;
      }
      await new Promise((r) => setTimeout(r, 1000));
    }
    return false;
  }

  // Load buckets
  async function loadBuckets() {
    loadingBuckets = true;
    error = null;
    try {
      buckets = await api.listBuckets();
    } catch (e: any) {
      error = e.message || 'Failed to load buckets';
      console.error('Failed to load buckets:', e);
    } finally {
      loadingBuckets = false;
    }
  }

  // Load files for current bucket/path
  async function loadFiles() {
    if (!currentBucket) {
      files = [];
      return;
    }

    loadingFiles = true;
    error = null;
    try {
      files = await api.listFiles(currentBucket, currentPath);
    } catch (e: any) {
      error = e.message || 'Failed to load files';
      console.error('Failed to load files:', e);
    } finally {
      loadingFiles = false;
    }
  }

  // Handle bucket selection
  function handleBucketSelect(event: CustomEvent<{ bucket: string }>) {
    currentBucket = event.detail.bucket;
    currentPath = '';
    loadFiles();
  }

  // Handle file navigation (folders)
  function handleNavigate(detail: { path: string }) {
    currentPath = detail.path;
    loadFiles();
  }

  // Handle file drop for upload
  async function handleFileDrop(event: CustomEvent<{ files: File[] }>) {
    if (!currentBucket) {
      error = 'Please select a bucket first';
      return;
    }

    for (const file of event.detail.files) {
      const remotePath = currentPath ? `${currentPath}/${file.name}` : file.name;
      const uploadId = addUpload(file.name, currentBucket, remotePath);

      try {
        await api.uploadFile(currentBucket, remotePath, file, (progress) => {
          updateUploadProgress(uploadId, progress);
        });
        completeUpload(uploadId);
        // Refresh file list
        loadFiles();
      } catch (e: any) {
        failUpload(uploadId, e.message || 'Upload failed');
      }
    }
  }

  // Handle file download with progress tracking
  async function handleDownload(detail: { file: ObjectInfo }) {
    if (!currentBucket) return;

    const file = detail.file;
    const fileName = file.Name.split('/').pop() || 'download';
    const downloadId = addDownload(fileName, currentBucket, file.Name);

    try {
      const { promise, cancel } = api.downloadFileWithProgress(
        currentBucket,
        file.Name,
        (progress) => {
          updateDownloadProgress(downloadId, progress);
        }
      );

      const blob = await promise;
      completeDownload(downloadId);

      // Trigger browser download
      const url = URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = fileName;
      a.click();
      URL.revokeObjectURL(url);
    } catch (e: any) {
      if (e.message !== 'Download cancelled') {
        failDownload(downloadId, e.message || 'Download failed');
      }
    }
  }

  // Handle file delete - opens confirmation dialog
  function handleDelete(detail: { file: ObjectInfo }) {
    if (!currentBucket) {
      showError('No bucket selected');
      return;
    }

    if (!detail || !detail.file) {
      showError('No file selected');
      return;
    }

    deleteConfirm = { open: true, file: detail.file };
  }

  // Actually perform the delete after confirmation
  async function confirmDelete() {
    if (!currentBucket || !deleteConfirm.file) return;

    const file = deleteConfirm.file;
    const fileName = file.Name.split('/').pop() || file.Name;
    const isFolder = file.Name.endsWith('/');

    deleteConfirm = { open: false, file: null };

    try {
      info(`Deleting ${isFolder ? 'folder' : 'file'} ${fileName}...`);
      await api.deleteFile(currentBucket, file.Name);
      success(`Deleted ${fileName}`);
      loadFiles();
    } catch (e: any) {
      showError(e.message || 'Failed to delete');
    }
  }

  // Cancel delete
  function cancelDelete() {
    deleteConfirm = { open: false, file: null };
  }

  // Handle URL copy notification
  function handleCopyUrl(_detail: { url: string }) {
    success('URL copied to clipboard');
  }

  // Handle sync start
  async function handleStartSync(event: CustomEvent<{
    localPath: string;
    bucket: string;
    remotePath: string;
    direction: 'to_remote' | 'to_local';
    dryRun: boolean;
    delete: boolean;
  }>) {
    try {
      const { localPath, bucket, remotePath, direction, dryRun, delete: deleteExtra } = event.detail;
      const result = await api.startSync(localPath, bucket, remotePath, direction, {
        dryRun,
        delete: deleteExtra,
      });

      addSyncJob({
        id: result.job_id,
        localPath,
        bucket,
        remotePath,
        direction,
        status: 'running',
      });
    } catch (e: any) {
      error = e.message || 'Failed to start sync';
    }
  }

  // Handle watch start
  async function handleStartWatch(event: CustomEvent<{
    localPath: string;
    bucket: string;
    remotePath: string;
  }>) {
    try {
      const { localPath, bucket, remotePath } = event.detail;
      const result = await api.startWatch(localPath, bucket, remotePath);

      addWatchJob({
        id: result.job_id,
        localPath,
        bucket,
        remotePath,
        status: 'running',
        recentUploads: [],
      });
    } catch (e: any) {
      error = e.message || 'Failed to start watch';
    }
  }

  // Handle watch stop
  async function handleStopWatch(event: CustomEvent<{ jobId: string }>) {
    try {
      await api.stopWatch(event.detail.jobId);
      updateWatchJob(event.detail.jobId, { status: 'stopped' });
    } catch (e: any) {
      error = e.message || 'Failed to stop watch';
    }
  }

  // Breadcrumb navigation
  function navigateToPath(index: number) {
    const parts = currentPath.split('/').filter(Boolean);
    currentPath = parts.slice(0, index).join('/');
    loadFiles();
  }

  // Computed breadcrumbs
  let breadcrumbs = $derived(currentPath
    ? currentPath.split('/').filter(Boolean)
    : []);

  // WebSocket event unsubscribers
  let wsUnsubscribers: (() => void)[] = [];

  // Menu event unlisteners
  let menuUnlisteners: UnlistenFn[] = [];

  // Sidebar visibility
  let sidebarVisible = $state(true);

  // File input reference for upload
  let fileInput: HTMLInputElement;

  onMount(async () => {
    // Listen for backend status events
    menuUnlisteners.push(await listen<{ error?: string } | string>('backend-status', (event) => {
      const payload = event.payload;
      if (typeof payload === 'string') {
        backendStatus = payload as BackendStatusType;
        backendError = undefined;
      } else if (payload && typeof payload === 'object') {
        // Handle crashed status with error
        if ('error' in payload) {
          backendStatus = 'crashed';
          backendError = payload.error;
        }
      }

      // When backend becomes healthy, reload data
      if (backendStatus === 'healthy' && !serverConnected) {
        resetApiPort();
        initApiPort().then(() => {
          serverConnected = true;
          loadBuckets();
          ws.connect().catch(console.warn);
        });
      }
    }));

    // Initialize API port
    await initApiPort();

    // Wait for Go server to start
    serverConnected = await waitForServer();

    if (serverConnected) {
      // Connect WebSocket
      try {
        await ws.connect();

        // Register WebSocket event handlers
        wsUnsubscribers.push(
          ws.on('upload_complete', (event) => {
            const name = event.data?.name || 'File';
            success(`Upload complete: ${name}`);
            loadFiles(); // Refresh file list
          })
        );

        wsUnsubscribers.push(
          ws.on('sync_progress', (event) => {
            const { job_id, phase, file } = event.data;
            updateSyncJob(job_id, { progress: `${phase}: ${file}` });
          })
        );

        wsUnsubscribers.push(
          ws.on('sync_complete', (event) => {
            const { job_id, status } = event.data;
            updateSyncJob(job_id, { status });
            if (status === 'completed') {
              loadFiles(); // Refresh file list after sync
            }
          })
        );

        wsUnsubscribers.push(
          ws.on('watch_upload', (event) => {
            const { job_id, path, error: uploadError } = event.data;
            if (!uploadError) {
              updateWatchJob(job_id, {
                recentUploads: [path] // Could append to existing list
              });
              loadFiles(); // Refresh file list
            }
          })
        );

        wsUnsubscribers.push(
          ws.on('file_deleted', (event) => {
            loadFiles(); // Refresh file list
          })
        );
      } catch (e) {
        console.warn('WebSocket connection failed:', e);
      }

      // Load initial data
      await loadBuckets();

      // Register menu event handlers
      menuUnlisteners.push(await listen('menu-upload', () => {
        fileInput?.click();
      }));

      menuUnlisteners.push(await listen('menu-download', () => {
        // Download all selected files
        selectedFiles.forEach(fileName => {
          const file = files.find(f => f.Name === fileName);
          if (file) {
            handleDownload({ file });
          }
        });
      }));

      menuUnlisteners.push(await listen('menu-delete', () => {
        // Delete all selected files
        selectedFiles.forEach(fileName => {
          const file = files.find(f => f.Name === fileName);
          if (file) {
            handleDelete({ file });
          }
        });
      }));

      menuUnlisteners.push(await listen('menu-refresh', () => {
        loadFiles();
        loadBuckets();
      }));

      menuUnlisteners.push(await listen('menu-copy-url', () => {
        if (selectedFiles.size > 0 && currentBucket) {
          const fileName = Array.from(selectedFiles)[0];
          const url = api.getDownloadUrl(currentBucket, fileName);
          navigator.clipboard.writeText(url);
          success('URL copied to clipboard');
        }
      }));

      menuUnlisteners.push(await listen('menu-toggle-sidebar', () => {
        sidebarVisible = !sidebarVisible;
      }));

      menuUnlisteners.push(await listen('menu-preferences', () => {
        showSettings = true;
      }));

    } else {
      error = 'Failed to connect to backend server';
    }
  });

  onDestroy(() => {
    // Unsubscribe from WebSocket events
    wsUnsubscribers.forEach(unsub => unsub());
    ws.disconnect();
    // Cleanup menu listeners
    menuUnlisteners.forEach(unlisten => unlisten());
  });
</script>

<main>
  <!-- Hidden file input for menu-triggered uploads -->
  <input
    type="file"
    multiple
    bind:this={fileInput}
    on:change={(e) => {
      const input = e.target as HTMLInputElement;
      if (input.files && input.files.length > 0) {
        handleFileDrop({ detail: { files: Array.from(input.files) } } as CustomEvent<{ files: File[] }>);
        input.value = ''; // Reset for next upload
      }
    }}
    style="display: none;"
  />

  <div class="app-container">
    <!-- Sidebar -->
    <aside class="sidebar" class:hidden={!sidebarVisible}>
      <div class="logo">
        <svg viewBox="0 0 24 24" fill="currentColor">
          <path d="M19.35 10.04C18.67 6.59 15.64 4 12 4 9.11 4 6.6 5.64 5.35 8.04 2.34 8.36 0 10.91 0 14c0 3.31 2.69 6 6 6h13c2.76 0 5-2.24 5-5 0-2.64-2.05-4.78-4.65-4.96zM14 13v4h-4v-4H7l5-5 5 5h-3z"/>
        </svg>
        <span>BB Stream</span>
      </div>

      <BucketSelector
        {buckets}
        selected={currentBucket}
        loading={loadingBuckets}
        on:select={handleBucketSelect}
        on:refresh={loadBuckets}
      />

      <div class="sidebar-panels">
        <SyncPanel
          {buckets}
          on:startSync={handleStartSync}
        />
        <WatchPanel
          {buckets}
          on:startWatch={handleStartWatch}
          on:stopWatch={handleStopWatch}
        />
      </div>

      {#if !serverConnected}
        <div class="connection-status error">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <circle cx="12" cy="12" r="10" />
            <line x1="15" y1="9" x2="9" y2="15" />
            <line x1="9" y1="9" x2="15" y2="15" />
          </svg>
          Server disconnected
        </div>
      {/if}
    </aside>

    <!-- Main content -->
    <div class="main-content">
      <!-- Header -->
      <header class="content-header">
        <div class="breadcrumbs">
          {#if currentBucket}
            <button class="breadcrumb" on:click={() => navigateToPath(0)}>
              {currentBucket}
            </button>
            {#each breadcrumbs as crumb, i}
              <span class="breadcrumb-separator">/</span>
              <button class="breadcrumb" on:click={() => navigateToPath(i + 1)}>
                {crumb}
              </button>
            {/each}
          {:else}
            <span class="breadcrumb-placeholder">Select a bucket</span>
          {/if}
        </div>
      </header>

      <!-- Error message -->
      {#if error}
        <div class="error-banner">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <circle cx="12" cy="12" r="10" />
            <line x1="12" y1="8" x2="12" y2="12" />
            <line x1="12" y1="16" x2="12.01" y2="16" />
          </svg>
          {error}
          <button on:click={() => (error = null)}>Dismiss</button>
        </div>
      {/if}

      <!-- Drop zone -->
      <FileDropzone
        disabled={!currentBucket || !serverConnected}
        on:drop={handleFileDrop}
      />

      <!-- File list -->
      <div class="file-list-container">
        <FileList
          {files}
          loading={loadingFiles}
          {selectedFiles}
          bucket={currentBucket || ''}
          onnavigate={handleNavigate}
          ondownload={handleDownload}
          ondelete={handleDelete}
          oncopyUrl={handleCopyUrl}
        />
      </div>

      <!-- Upload panel -->
      <UploadPanel
        uploads={$uploads}
        on:remove={(e) => removeUpload(e.detail.id)}
        on:clear={clearCompletedUploads}
      />

      <!-- Download panel -->
      <DownloadPanel
        downloads={$downloads}
        on:remove={(e) => removeDownload(e.detail.id)}
        on:clear={clearCompletedDownloads}
      />
    </div>
  </div>

  <!-- Toast notifications -->
  <ToastContainer />

  <!-- Settings modal -->
  <SettingsModal
    open={showSettings}
    onclose={() => showSettings = false}
    onsaved={() => {
      // Reload buckets after settings saved
      loadBuckets();
    }}
  />

  <!-- Delete confirmation dialog -->
  <ConfirmDialog
    open={deleteConfirm.open}
    title={deleteConfirm.file?.Name.endsWith('/') ? 'Delete Folder' : 'Delete File'}
    message={`Are you sure you want to delete "${deleteConfirm.file?.Name.split('/').filter(Boolean).pop() || ''}"? This action cannot be undone.`}
    confirmLabel="Delete"
    cancelLabel="Cancel"
    variant="danger"
    onconfirm={confirmDelete}
    oncancel={cancelDelete}
  />

  <!-- Backend status overlay -->
  <BackendStatusOverlay status={backendStatus} error={backendError} />
</main>

<style>
  :global(*) {
    box-sizing: border-box;
  }

  :global(body) {
    margin: 0;
    font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Oxygen, Ubuntu, sans-serif;
    background: #f5f7fa;
    color: #333;
  }

  main {
    height: 100vh;
    overflow: hidden;
  }

  .app-container {
    display: flex;
    height: 100%;
  }

  .sidebar {
    width: 280px;
    background: white;
    border-right: 1px solid #e0e0e0;
    display: flex;
    flex-direction: column;
    overflow-y: auto;
    transition: margin-left 0.2s ease, opacity 0.2s ease;
  }

  .sidebar.hidden {
    margin-left: -280px;
    opacity: 0;
    pointer-events: none;
  }

  .sidebar-panels {
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
    padding: 0.5rem;
    flex: 1;
  }

  .logo {
    display: flex;
    align-items: center;
    gap: 0.75rem;
    padding: 1.25rem;
    border-bottom: 1px solid #e0e0e0;
    font-size: 1.25rem;
    font-weight: 600;
    color: #1976d2;
  }

  .logo svg {
    width: 32px;
    height: 32px;
  }

  .connection-status {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    padding: 0.75rem 1rem;
    margin: 0.5rem;
    border-radius: 6px;
    font-size: 0.85rem;
  }

  .connection-status.error {
    background: #ffebee;
    color: #c62828;
  }

  .connection-status svg {
    width: 18px;
    height: 18px;
  }

  .main-content {
    flex: 1;
    display: flex;
    flex-direction: column;
    overflow: hidden;
    padding: 1.5rem;
    gap: 1rem;
  }

  .content-header {
    display: flex;
    align-items: center;
    gap: 1rem;
  }

  .breadcrumbs {
    display: flex;
    align-items: center;
    gap: 0.25rem;
    font-size: 0.9rem;
  }

  .breadcrumb {
    background: none;
    border: none;
    padding: 0.25rem 0.5rem;
    cursor: pointer;
    color: #1976d2;
    border-radius: 4px;
    font-size: inherit;
  }

  .breadcrumb:hover {
    background: #e3f2fd;
  }

  .breadcrumb-separator {
    color: #999;
  }

  .breadcrumb-placeholder {
    color: #999;
  }

  .error-banner {
    display: flex;
    align-items: center;
    gap: 0.75rem;
    padding: 0.75rem 1rem;
    background: #ffebee;
    color: #c62828;
    border-radius: 8px;
  }

  .error-banner svg {
    width: 20px;
    height: 20px;
    flex-shrink: 0;
  }

  .error-banner button {
    margin-left: auto;
    background: none;
    border: none;
    color: inherit;
    cursor: pointer;
    text-decoration: underline;
  }

  .file-list-container {
    flex: 1;
    overflow: auto;
  }
</style>
