<script lang="ts">
  import ProgressBar from './ProgressBar.svelte';
  import type { Upload } from '../stores/jobs';

  interface Props {
    uploads?: Upload[];
    oncancel?: (detail: { id: string }) => void;
    onremove?: (detail: { id: string }) => void;
    onclear?: () => void;
  }

  let {
    uploads = [],
    oncancel,
    onremove,
    onclear
  }: Props = $props();

  let activeUploads = $derived(uploads.filter((u) => u.status === 'uploading' || u.status === 'pending'));
  let completedUploads = $derived(uploads.filter((u) => u.status === 'complete'));
  let errorUploads = $derived(uploads.filter((u) => u.status === 'error'));
  let hasCompleted = $derived(completedUploads.length > 0);
</script>

{#if uploads.length > 0}
  <div class="upload-panel">
    <div class="header">
      <h3>Uploads</h3>
      {#if hasCompleted}
        <button class="clear-btn" onclick={() => onclear?.()}>
          Clear completed
        </button>
      {/if}
    </div>

    <div class="upload-list">
      {#each uploads as upload (upload.id)}
        <div class="upload-item {upload.status}">
          <div class="upload-info">
            <span class="file-name" title={upload.fileName}>
              {upload.fileName}
            </span>
            <span class="destination">
              → {upload.bucket}/{upload.path}
            </span>
          </div>

          {#if upload.status === 'uploading' || upload.status === 'pending'}
            <ProgressBar
              progress={upload.progress}
              size="sm"
            />
          {:else if upload.status === 'complete'}
            <div class="status success">
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <polyline points="20 6 9 17 4 12" />
              </svg>
              Complete
            </div>
          {:else if upload.status === 'error'}
            <div class="status error">
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <circle cx="12" cy="12" r="10" />
                <line x1="15" y1="9" x2="9" y2="15" />
                <line x1="9" y1="9" x2="15" y2="15" />
              </svg>
              {upload.error || 'Failed'}
            </div>
          {/if}

          <button
            class="remove-btn"
            title="Remove"
            onclick={() => onremove?.({ id: upload.id })}
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
  .upload-panel {
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

  .upload-list {
    max-height: 200px;
    overflow-y: auto;
  }

  .upload-item {
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
    padding: 0.75rem 1rem;
    border-bottom: 1px solid var(--border-color, #eee);
    position: relative;
  }

  .upload-item:last-child {
    border-bottom: none;
  }

  .upload-info {
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

  .destination {
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
