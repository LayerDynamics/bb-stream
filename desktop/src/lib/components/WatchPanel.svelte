<script lang="ts">
  import { watchJobs, removeWatchJob } from '../stores/jobs';

  interface Props {
    buckets?: { Name: string }[];
    onstartWatch?: (detail: {
      localPath: string;
      bucket: string;
      remotePath: string;
    }) => void;
    onstopWatch?: (detail: { jobId: string }) => void;
  }

  let {
    buckets = [],
    onstartWatch,
    onstopWatch
  }: Props = $props();

  let localPath = $state('');
  let selectedBucket = $state('');
  let remotePath = $state('');
  let expanded = $state(false);

  function handleStartWatch() {
    if (!localPath || !selectedBucket) return;

    onstartWatch?.({
      localPath,
      bucket: selectedBucket,
      remotePath,
    });

    // Clear form
    localPath = '';
    remotePath = '';
  }

  function handleStopWatch(jobId: string) {
    onstopWatch?.({ jobId });
  }

  function getStatusColor(status: string): string {
    switch (status) {
      case 'running': return 'var(--success-color, #4caf50)';
      case 'stopped': return 'var(--text-secondary, #666)';
      default: return 'var(--text-secondary, #666)';
    }
  }
</script>

<div class="watch-panel">
  <button class="panel-toggle" onclick={() => expanded = !expanded}>
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
      <circle cx="12" cy="12" r="10" />
      <polyline points="12 6 12 12 16 14" />
    </svg>
    <span>Watch</span>
    {#if $watchJobs.filter(j => j.status === 'running').length > 0}
      <span class="active-badge">{$watchJobs.filter(j => j.status === 'running').length}</span>
    {/if}
    <span class="toggle-icon">{expanded ? '-' : '+'}</span>
  </button>

  {#if expanded}
    <div class="panel-content">
      <!-- New watch form -->
      <div class="watch-form">
        <div class="form-group">
          <label for="watch-local-path">Local Folder</label>
          <input
            id="watch-local-path"
            type="text"
            bind:value={localPath}
            placeholder="/path/to/watch"
          />
        </div>

        <div class="form-group">
          <label for="watch-bucket">Bucket</label>
          <select id="watch-bucket" bind:value={selectedBucket}>
            <option value="">Select bucket...</option>
            {#each buckets as bucket}
              <option value={bucket.Name}>{bucket.Name}</option>
            {/each}
          </select>
        </div>

        <div class="form-group">
          <label for="watch-remote-path">Remote Path</label>
          <input
            id="watch-remote-path"
            type="text"
            bind:value={remotePath}
            placeholder="folder/subfolder"
          />
        </div>

        <button
          class="start-btn"
          onclick={handleStartWatch}
          disabled={!localPath || !selectedBucket}
        >
          Start Watching
        </button>
      </div>

      <!-- Active watch jobs -->
      {#if $watchJobs.length > 0}
        <div class="jobs-section">
          <h4>Active Watchers</h4>
          {#each $watchJobs as job}
            <div class="job-item">
              <div class="job-info">
                <span class="status-dot" style="background: {getStatusColor(job.status)}"></span>
                <span class="job-path">{job.localPath}</span>
                <span class="job-arrow">\u2192</span>
                <span class="job-bucket">{job.bucket}/{job.remotePath || ''}</span>
              </div>
              {#if job.recentUploads && job.recentUploads.length > 0}
                <div class="recent-uploads">
                  <span class="label">Recent:</span>
                  {#each job.recentUploads.slice(0, 3) as upload}
                    <span class="upload-item">{upload.split('/').pop()}</span>
                  {/each}
                </div>
              {/if}
              <div class="job-actions">
                {#if job.status === 'running'}
                  <button class="stop-btn" onclick={() => handleStopWatch(job.id)}>
                    Stop
                  </button>
                {:else}
                  <button class="remove-btn" onclick={() => removeWatchJob(job.id)}>
                    Remove
                  </button>
                {/if}
              </div>
            </div>
          {/each}
        </div>
      {/if}
    </div>
  {/if}
</div>

<style>
  .watch-panel {
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

  .active-badge {
    background: var(--success-color, #4caf50);
    color: white;
    font-size: 0.75rem;
    padding: 0.125rem 0.375rem;
    border-radius: 10px;
    margin-left: 0.25rem;
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

  .watch-form {
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
    flex-direction: column;
    gap: 0.25rem;
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
  }

  .status-dot {
    width: 8px;
    height: 8px;
    border-radius: 50%;
    flex-shrink: 0;
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

  .recent-uploads {
    display: flex;
    align-items: center;
    gap: 0.25rem;
    font-size: 0.75rem;
    color: var(--text-secondary, #666);
    padding-left: 1rem;
  }

  .recent-uploads .label {
    color: var(--text-muted, #999);
  }

  .upload-item {
    background: var(--bg-primary, white);
    padding: 0.125rem 0.375rem;
    border-radius: 3px;
    border: 1px solid var(--border-color, #eee);
  }

  .job-actions {
    display: flex;
    gap: 0.5rem;
    padding-left: 1rem;
  }

  .stop-btn {
    padding: 0.25rem 0.5rem;
    background: var(--error-color, #d32f2f);
    color: white;
    border: none;
    border-radius: 4px;
    cursor: pointer;
    font-size: 0.75rem;
  }

  .stop-btn:hover {
    background: var(--error-hover, #b71c1c);
  }

  .remove-btn {
    padding: 0.25rem 0.5rem;
    background: var(--bg-secondary, #e0e0e0);
    color: var(--text-primary, #333);
    border: none;
    border-radius: 4px;
    cursor: pointer;
    font-size: 0.75rem;
  }

  .remove-btn:hover {
    background: var(--bg-hover, #ccc);
  }
</style>
