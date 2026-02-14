<script lang="ts">
  import { createEventDispatcher } from 'svelte';
  import type { SyncJobInfo } from '../stores/jobs';
  import { syncJobs, removeSyncJob } from '../stores/jobs';

  export let buckets: { Name: string }[] = [];

  const dispatch = createEventDispatcher<{
    startSync: {
      localPath: string;
      bucket: string;
      remotePath: string;
      direction: 'to_remote' | 'to_local';
      dryRun: boolean;
      delete: boolean;
    };
  }>();

  let localPath = '';
  let selectedBucket = '';
  let remotePath = '';
  let direction: 'to_remote' | 'to_local' = 'to_remote';
  let dryRun = false;
  let deleteExtra = false;
  let expanded = false;

  function handleStartSync() {
    if (!localPath || !selectedBucket) return;

    dispatch('startSync', {
      localPath,
      bucket: selectedBucket,
      remotePath,
      direction,
      dryRun,
      delete: deleteExtra,
    });
  }

  function getStatusColor(status: string): string {
    switch (status) {
      case 'running': return 'var(--primary-color, #1976d2)';
      case 'completed': return 'var(--success-color, #4caf50)';
      case 'failed': return 'var(--error-color, #d32f2f)';
      default: return 'var(--text-secondary, #666)';
    }
  }
</script>

<div class="sync-panel">
  <button class="panel-toggle" on:click={() => expanded = !expanded}>
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
      <path d="M23 4v6h-6M1 20v-6h6" />
      <path d="M3.51 9a9 9 0 0 1 14.85-3.36L23 10M1 14l4.64 4.36A9 9 0 0 0 20.49 15" />
    </svg>
    <span>Sync</span>
    <span class="toggle-icon">{expanded ? '-' : '+'}</span>
  </button>

  {#if expanded}
    <div class="panel-content">
      <!-- New sync form -->
      <div class="sync-form">
        <div class="form-group">
          <label for="local-path">Local Path</label>
          <input
            id="local-path"
            type="text"
            bind:value={localPath}
            placeholder="/path/to/folder"
          />
        </div>

        <div class="form-group">
          <label for="bucket">Bucket</label>
          <select id="bucket" bind:value={selectedBucket}>
            <option value="">Select bucket...</option>
            {#each buckets as bucket}
              <option value={bucket.Name}>{bucket.Name}</option>
            {/each}
          </select>
        </div>

        <div class="form-group">
          <label for="remote-path">Remote Path</label>
          <input
            id="remote-path"
            type="text"
            bind:value={remotePath}
            placeholder="folder/subfolder"
          />
        </div>

        <div class="form-group">
          <label>Direction</label>
          <div class="radio-group">
            <label class="radio-label">
              <input type="radio" bind:group={direction} value="to_remote" />
              Local to Remote
            </label>
            <label class="radio-label">
              <input type="radio" bind:group={direction} value="to_local" />
              Remote to Local
            </label>
          </div>
        </div>

        <div class="form-group checkboxes">
          <label class="checkbox-label">
            <input type="checkbox" bind:checked={dryRun} />
            Dry run (preview only)
          </label>
          <label class="checkbox-label">
            <input type="checkbox" bind:checked={deleteExtra} />
            Delete extra files
          </label>
        </div>

        <button
          class="start-btn"
          on:click={handleStartSync}
          disabled={!localPath || !selectedBucket}
        >
          Start Sync
        </button>
      </div>

      <!-- Active sync jobs -->
      {#if $syncJobs.length > 0}
        <div class="jobs-section">
          <h4>Active Sync Jobs</h4>
          {#each $syncJobs as job}
            <div class="job-item">
              <div class="job-info">
                <span class="job-direction">{job.direction === 'to_remote' ? '↑' : '↓'}</span>
                <span class="job-path">{job.localPath}</span>
                <span class="job-arrow">→</span>
                <span class="job-bucket">{job.bucket}/{job.remotePath || ''}</span>
              </div>
              <div class="job-status" style="color: {getStatusColor(job.status)}">
                {job.status}
              </div>
              {#if job.progress}
                <div class="job-progress">{job.progress}</div>
              {/if}
              <button class="remove-btn" on:click={() => removeSyncJob(job.id)}>
                ×
              </button>
            </div>
          {/each}
        </div>
      {/if}
    </div>
  {/if}
</div>

<style>
  .sync-panel {
    background: white;
    border-radius: 8px;
    border: 1px solid var(--border-color, #ddd);
    overflow: hidden;
  }

  .panel-toggle {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    width: 100%;
    padding: 0.75rem 1rem;
    background: none;
    border: none;
    cursor: pointer;
    font-size: 0.9rem;
    font-weight: 500;
    color: var(--text-primary, #333);
  }

  .panel-toggle:hover {
    background: var(--bg-hover, #f5f5f5);
  }

  .panel-toggle svg {
    width: 18px;
    height: 18px;
    color: var(--primary-color, #1976d2);
  }

  .toggle-icon {
    margin-left: auto;
    font-size: 1.2rem;
    color: var(--text-secondary, #666);
  }

  .panel-content {
    padding: 1rem;
    border-top: 1px solid var(--border-color, #eee);
  }

  .sync-form {
    display: flex;
    flex-direction: column;
    gap: 0.75rem;
  }

  .form-group {
    display: flex;
    flex-direction: column;
    gap: 0.25rem;
  }

  .form-group label {
    font-size: 0.8rem;
    color: var(--text-secondary, #666);
  }

  .form-group input[type="text"],
  .form-group select {
    padding: 0.5rem;
    border: 1px solid var(--border-color, #ddd);
    border-radius: 4px;
    font-size: 0.9rem;
  }

  .radio-group {
    display: flex;
    gap: 1rem;
  }

  .radio-label,
  .checkbox-label {
    display: flex;
    align-items: center;
    gap: 0.25rem;
    font-size: 0.9rem;
    cursor: pointer;
  }

  .checkboxes {
    flex-direction: row;
    gap: 1rem;
  }

  .start-btn {
    padding: 0.5rem 1rem;
    background: var(--primary-color, #1976d2);
    color: white;
    border: none;
    border-radius: 4px;
    cursor: pointer;
    font-size: 0.9rem;
  }

  .start-btn:disabled {
    background: var(--disabled-color, #ccc);
    cursor: not-allowed;
  }

  .start-btn:hover:not(:disabled) {
    background: var(--primary-hover, #1565c0);
  }

  .jobs-section {
    margin-top: 1rem;
    padding-top: 1rem;
    border-top: 1px solid var(--border-color, #eee);
  }

  .jobs-section h4 {
    margin: 0 0 0.5rem 0;
    font-size: 0.85rem;
    color: var(--text-secondary, #666);
  }

  .job-item {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    padding: 0.5rem;
    background: var(--bg-secondary, #f9f9f9);
    border-radius: 4px;
    font-size: 0.85rem;
    margin-bottom: 0.5rem;
  }

  .job-info {
    display: flex;
    align-items: center;
    gap: 0.25rem;
    flex: 1;
    overflow: hidden;
  }

  .job-direction {
    font-weight: bold;
  }

  .job-path,
  .job-bucket {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .job-arrow {
    color: var(--text-secondary, #666);
  }

  .job-status {
    font-weight: 500;
  }

  .job-progress {
    font-size: 0.75rem;
    color: var(--text-secondary, #666);
  }

  .remove-btn {
    background: none;
    border: none;
    cursor: pointer;
    font-size: 1.2rem;
    color: var(--text-secondary, #666);
    padding: 0 0.25rem;
  }

  .remove-btn:hover {
    color: var(--error-color, #d32f2f);
  }
</style>
