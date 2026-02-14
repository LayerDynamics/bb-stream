<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  import api, { type StatusInfo } from '../api';
  import { uploads, downloads, syncJobs, watchJobs } from '../stores/jobs';

  interface Props {
    connected?: boolean;
  }

  let { connected = false }: Props = $props();

  let status = $state<StatusInfo | null>(null);
  let expanded = $state(false);
  let pollInterval: ReturnType<typeof setInterval> | null = null;

  // Derived counts from stores
  let activeUploads = $derived($uploads.filter(u => u.status === 'uploading' || u.status === 'pending').length);
  let activeDownloads = $derived($downloads.filter(d => d.status === 'downloading' || d.status === 'pending').length);
  let activeSyncs = $derived($syncJobs.filter(j => j.status === 'running').length);
  let activeWatches = $derived($watchJobs.filter(j => j.status === 'running').length);
  let totalActive = $derived(activeUploads + activeDownloads + activeSyncs + activeWatches);

  async function fetchStatus() {
    try {
      status = await api.getStatus();
    } catch {
      status = null;
    }
  }

  function formatUptime(seconds: number): string {
    if (seconds < 60) return `${seconds}s`;
    if (seconds < 3600) return `${Math.floor(seconds / 60)}m`;
    if (seconds < 86400) return `${Math.floor(seconds / 3600)}h ${Math.floor((seconds % 3600) / 60)}m`;
    return `${Math.floor(seconds / 86400)}d ${Math.floor((seconds % 86400) / 3600)}h`;
  }

  onMount(() => {
    if (connected) {
      fetchStatus();
      pollInterval = setInterval(fetchStatus, 30000); // Poll every 30s
    }
  });

  onDestroy(() => {
    if (pollInterval) {
      clearInterval(pollInterval);
    }
  });

  // Re-fetch when connection status changes
  $effect(() => {
    if (connected && !pollInterval) {
      fetchStatus();
      pollInterval = setInterval(fetchStatus, 30000);
    } else if (!connected && pollInterval) {
      clearInterval(pollInterval);
      pollInterval = null;
      status = null;
    }
  });
</script>

<div class="status-bar" class:expanded>
  <button class="status-bar-content" onclick={() => expanded = !expanded}>
    <div class="status-left">
      <div class="connection-status" class:connected>
        <span class="status-dot"></span>
        <span class="status-text">{connected ? 'Connected' : 'Disconnected'}</span>
      </div>

      {#if status}
        <div class="divider"></div>
        <div class="uptime" title="Backend uptime">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <circle cx="12" cy="12" r="10" />
            <polyline points="12 6 12 12 16 14" />
          </svg>
          <span>{formatUptime(status.uptime_seconds)}</span>
        </div>
      {/if}
    </div>

    <div class="status-right">
      {#if totalActive > 0}
        <div class="activity-indicator">
          <div class="activity-dot"></div>
          <span>{totalActive} active</span>
        </div>
      {/if}

      {#if activeUploads > 0}
        <div class="stat" title="{activeUploads} uploads in progress">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4" />
            <polyline points="17 8 12 3 7 8" />
            <line x1="12" y1="3" x2="12" y2="15" />
          </svg>
          <span>{activeUploads}</span>
        </div>
      {/if}

      {#if activeDownloads > 0}
        <div class="stat" title="{activeDownloads} downloads in progress">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4" />
            <polyline points="7 10 12 15 17 10" />
            <line x1="12" y1="15" x2="12" y2="3" />
          </svg>
          <span>{activeDownloads}</span>
        </div>
      {/if}

      {#if activeSyncs > 0}
        <div class="stat" title="{activeSyncs} sync jobs running">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M23 4v6h-6M1 20v-6h6" />
            <path d="M3.51 9a9 9 0 0 1 14.85-3.36L23 10M1 14l4.64 4.36A9 9 0 0 0 20.49 15" />
          </svg>
          <span>{activeSyncs}</span>
        </div>
      {/if}

      {#if activeWatches > 0}
        <div class="stat watching" title="{activeWatches} watchers active">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <circle cx="12" cy="12" r="10" />
            <polyline points="12 6 12 12 16 14" />
          </svg>
          <span>{activeWatches}</span>
        </div>
      {/if}

      <svg class="expand-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
        <polyline points={expanded ? "18 15 12 9 6 15" : "6 9 12 15 18 9"} />
      </svg>
    </div>
  </button>

  {#if expanded && status}
    <div class="status-details">
      <div class="detail-row">
        <span class="detail-label">Version</span>
        <span class="detail-value">{status.version}</span>
      </div>
      <div class="detail-row">
        <span class="detail-label">API Version</span>
        <span class="detail-value">{status.api_version}</span>
      </div>
      <div class="detail-row">
        <span class="detail-label">Uptime</span>
        <span class="detail-value">{formatUptime(status.uptime_seconds)}</span>
      </div>
      <div class="detail-row">
        <span class="detail-label">Active Sync Jobs</span>
        <span class="detail-value">{status.active_sync_jobs}</span>
      </div>
      <div class="detail-row">
        <span class="detail-label">Active Watch Jobs</span>
        <span class="detail-value">{status.active_watch_jobs}</span>
      </div>
      <div class="detail-row">
        <span class="detail-label">WebSocket Clients</span>
        <span class="detail-value">{status.websocket_clients}</span>
      </div>
    </div>
  {/if}
</div>

<style>
  .status-bar {
    position: fixed;
    bottom: 0;
    left: 0;
    right: 0;
    background: white;
    border-top: 1px solid var(--border-color, #e0e0e0);
    z-index: 100;
    transition: box-shadow 0.2s;
  }

  .status-bar.expanded {
    box-shadow: 0 -4px 20px rgba(0, 0, 0, 0.1);
  }

  .status-bar-content {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 0.5rem 1rem;
    width: 100%;
    background: none;
    border: none;
    cursor: pointer;
    text-align: left;
  }

  .status-bar-content:hover {
    background: var(--bg-hover, #f5f5f5);
  }

  .status-left,
  .status-right {
    display: flex;
    align-items: center;
    gap: 0.75rem;
  }

  .connection-status {
    display: flex;
    align-items: center;
    gap: 0.375rem;
  }

  .status-dot {
    width: 8px;
    height: 8px;
    border-radius: 50%;
    background: #f44336;
  }

  .connection-status.connected .status-dot {
    background: #4caf50;
  }

  .status-text {
    font-size: 0.8rem;
    color: var(--text-secondary, #666);
  }

  .divider {
    width: 1px;
    height: 16px;
    background: var(--border-color, #e0e0e0);
  }

  .uptime {
    display: flex;
    align-items: center;
    gap: 0.25rem;
    font-size: 0.8rem;
    color: var(--text-secondary, #666);
  }

  .uptime svg {
    width: 14px;
    height: 14px;
  }

  .activity-indicator {
    display: flex;
    align-items: center;
    gap: 0.375rem;
    font-size: 0.75rem;
    color: var(--primary-color, #1976d2);
    background: rgba(25, 118, 210, 0.1);
    padding: 0.25rem 0.5rem;
    border-radius: 12px;
  }

  .activity-dot {
    width: 6px;
    height: 6px;
    border-radius: 50%;
    background: var(--primary-color, #1976d2);
    animation: pulse 1.5s ease-in-out infinite;
  }

  @keyframes pulse {
    0%, 100% {
      opacity: 1;
    }
    50% {
      opacity: 0.4;
    }
  }

  .stat {
    display: flex;
    align-items: center;
    gap: 0.25rem;
    font-size: 0.75rem;
    color: var(--text-secondary, #666);
    padding: 0.25rem 0.5rem;
    background: var(--bg-secondary, #f5f5f5);
    border-radius: 4px;
  }

  .stat.watching {
    color: #4caf50;
    background: rgba(76, 175, 80, 0.1);
  }

  .stat svg {
    width: 14px;
    height: 14px;
  }

  .expand-icon {
    width: 16px;
    height: 16px;
    color: var(--text-secondary, #999);
    transition: transform 0.2s;
  }

  .status-details {
    padding: 0.75rem 1rem;
    border-top: 1px solid var(--border-color, #eee);
    background: var(--bg-secondary, #fafafa);
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(150px, 1fr));
    gap: 0.5rem;
  }

  .detail-row {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 0.25rem 0;
  }

  .detail-label {
    font-size: 0.75rem;
    color: var(--text-secondary, #666);
  }

  .detail-value {
    font-size: 0.75rem;
    font-weight: 500;
    color: var(--text-primary, #333);
  }
</style>
