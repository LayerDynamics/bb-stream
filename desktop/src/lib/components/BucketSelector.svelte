<script lang="ts">
  import { createEventDispatcher } from 'svelte';
  import type { BucketInfo } from '../api';

  export let buckets: BucketInfo[] = [];
  export let selected: string | null = null;
  export let loading = false;

  const dispatch = createEventDispatcher<{
    select: { bucket: string };
    refresh: void;
  }>();

  function handleSelect(bucket: string) {
    dispatch('select', { bucket });
  }
</script>

<div class="bucket-selector">
  <div class="header">
    <h3>Buckets</h3>
    <button
      class="refresh-btn"
      title="Refresh buckets"
      disabled={loading}
      on:click={() => dispatch('refresh')}
    >
      <svg
        class:spinning={loading}
        viewBox="0 0 24 24"
        fill="none"
        stroke="currentColor"
        stroke-width="2"
      >
        <polyline points="23 4 23 10 17 10" />
        <polyline points="1 20 1 14 7 14" />
        <path d="M3.51 9a9 9 0 0 1 14.85-3.36L23 10M1 14l4.64 4.36A9 9 0 0 0 20.49 15" />
      </svg>
    </button>
  </div>

  <div class="bucket-list">
    {#if loading && buckets.length === 0}
      <div class="loading">Loading buckets...</div>
    {:else if buckets.length === 0}
      <div class="empty">No buckets found</div>
    {:else}
      {#each buckets as bucket (bucket.Name)}
        <button
          class="bucket-item"
          class:selected={selected === bucket.Name}
          on:click={() => handleSelect(bucket.Name)}
        >
          <svg class="bucket-icon" viewBox="0 0 24 24" fill="currentColor">
            <path d="M18 2H6c-1.1 0-2 .9-2 2v16c0 1.1.9 2 2 2h12c1.1 0 2-.9 2-2V4c0-1.1-.9-2-2-2zM6 4h5v8l-2.5-1.5L6 12V4z" />
          </svg>
          <span class="bucket-name">{bucket.Name}</span>
          <span class="bucket-type">{bucket.Type}</span>
        </button>
      {/each}
    {/if}
  </div>
</div>

<style>
  .bucket-selector {
    display: flex;
    flex-direction: column;
    height: 100%;
  }

  .header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 1rem;
    border-bottom: 1px solid var(--border-color, #ddd);
  }

  h3 {
    margin: 0;
    font-size: 1rem;
    font-weight: 600;
  }

  .refresh-btn {
    background: none;
    border: none;
    padding: 0.25rem;
    cursor: pointer;
    color: var(--text-secondary, #666);
    border-radius: 4px;
    transition: all 0.2s;
  }

  .refresh-btn:hover:not(:disabled) {
    background: var(--bg-hover, #e0e0e0);
    color: var(--primary-color, #1976d2);
  }

  .refresh-btn:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .refresh-btn svg {
    width: 18px;
    height: 18px;
    display: block;
  }

  .refresh-btn svg.spinning {
    animation: spin 1s linear infinite;
  }

  @keyframes spin {
    to {
      transform: rotate(360deg);
    }
  }

  .bucket-list {
    flex: 1;
    overflow-y: auto;
    padding: 0.5rem;
  }

  .bucket-item {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    width: 100%;
    padding: 0.75rem;
    border: none;
    background: none;
    border-radius: 6px;
    cursor: pointer;
    text-align: left;
    transition: background 0.2s;
  }

  .bucket-item:hover {
    background: var(--bg-hover, #f0f0f0);
  }

  .bucket-item.selected {
    background: var(--bg-selected, #e3f2fd);
    color: var(--primary-color, #1976d2);
  }

  .bucket-icon {
    width: 20px;
    height: 20px;
    flex-shrink: 0;
    color: var(--primary-color, #1976d2);
  }

  .bucket-name {
    flex: 1;
    font-weight: 500;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .bucket-type {
    font-size: 0.75rem;
    color: var(--text-secondary, #999);
    text-transform: lowercase;
  }

  .loading,
  .empty {
    padding: 1rem;
    text-align: center;
    color: var(--text-secondary, #666);
    font-size: 0.9rem;
  }
</style>
