<script lang="ts">
  import ProgressBar from './ProgressBar.svelte';
  import type { Download } from '../stores/jobs';

  interface Props {
    downloads?: Download[];
    oncancel?: (detail: { id: string }) => void;
    onremove?: (detail: { id: string }) => void;
    onclear?: () => void;
  }

  let {
    downloads = [],
    oncancel,
    onremove,
    onclear
  }: Props = $props();

  let activeDownloads = $derived(downloads.filter((d) => d.status === 'downloading' || d.status === 'pending'));
  let completedDownloads = $derived(downloads.filter((d) => d.status === 'complete'));
  let errorDownloads = $derived(downloads.filter((d) => d.status === 'error'));
  let hasCompleted = $derived(completedDownloads.length > 0);
</script>

{#if downloads.length > 0}
  <div class="download-panel">
    <div class="header">
      <h3>Downloads</h3>
      {#if hasCompleted}
        <button class="clear-btn" onclick={() => onclear?.()}>
          Clear completed
        </button>
      {/if}
    </div>

    <div class="download-list">
      {#each downloads as download (download.id)}
        <div class="download-item {download.status}">
          <div class="download-info">
            <span class="file-name" title={download.fileName}>
              {download.fileName}
            </span>
            <span class="source">
              {download.bucket}/{download.path}
            </span>
          </div>

          {#if download.status === 'downloading' || download.status === 'pending'}
            <ProgressBar
              progress={download.progress}
              size="sm"
            />
          {:else if download.status === 'complete'}
            <div class="status success">
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <polyline points="20 6 9 17 4 12" />
              </svg>
              Complete
            </div>
          {:else if download.status === 'error'}
            <div class="status error">
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <circle cx="12" cy="12" r="10" />
                <line x1="15" y1="9" x2="9" y2="15" />
                <line x1="9" y1="9" x2="15" y2="15" />
              </svg>
              {download.error || 'Failed'}
            </div>
          {/if}

          <button
            class="remove-btn"
            title="Remove"
            onclick={() => onremove?.({ id: download.id })}
          >
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <line x1="18" y1="6" x2="6" y2="18" />
              <line x1="6" y1="6" x2="18" y2="18" />
            </svg>
          </button>
        </div>
      {/each}
    </div>
  </div>
{/if}

<style>
  .download-panel {
    border: 1px solid var(--border-color, #ddd);
    border-radius: 8px;
    overflow: hidden;
    margin-top: 1rem;
  }

  .header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 0.75rem 1rem;
    background: var(--bg-secondary, #f5f5f5);
    border-bottom: 1px solid var(--border-color, #ddd);
  }

  h3 {
    margin: 0;
    font-size: 0.9rem;
    font-weight: 600;
  }

  .clear-btn {
    background: none;
    border: none;
    color: var(--primary-color, #1976d2);
    font-size: 0.8rem;
    cursor: pointer;
    padding: 0.25rem 0.5rem;
    border-radius: 4px;
  }

  .clear-btn:hover {
    background: var(--bg-hover, #e0e0e0);
  }

  .download-list {
    max-height: 200px;
    overflow-y: auto;
  }

  .download-item {
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
    padding: 0.75rem 1rem;
    border-bottom: 1px solid var(--border-color, #eee);
    position: relative;
  }

  .download-item:last-child {
    border-bottom: none;
  }

  .download-info {
    display: flex;
    flex-direction: column;
    gap: 0.25rem;
    padding-right: 2rem;
  }

  .file-name {
    font-weight: 500;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .source {
    font-size: 0.8rem;
    color: var(--text-secondary, #666);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .status {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    font-size: 0.85rem;
  }

  .status svg {
    width: 16px;
    height: 16px;
  }

  .status.success {
    color: var(--success-color, #4caf50);
  }

  .status.error {
    color: var(--error-color, #f44336);
  }

  .remove-btn {
    position: absolute;
    top: 0.75rem;
    right: 0.75rem;
    background: none;
    border: none;
    padding: 0.25rem;
    cursor: pointer;
    color: var(--text-secondary, #999);
    border-radius: 4px;
    opacity: 0.5;
    transition: all 0.2s;
  }

  .remove-btn:hover {
    opacity: 1;
    background: var(--bg-hover, #e0e0e0);
  }

  .remove-btn svg {
    width: 16px;
    height: 16px;
    display: block;
  }
</style>
