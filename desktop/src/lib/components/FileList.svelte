<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  import type { ObjectInfo } from '../api';
  import api from '../api';

  interface Props {
    files?: ObjectInfo[];
    loading?: boolean;
    selectedFiles?: Set<string>;
    bucket?: string;
    onselect?: (detail: { file: ObjectInfo }) => void;
    ondownload?: (detail: { file: ObjectInfo }) => void;
    ondelete?: (detail: { file: ObjectInfo }) => void;
    onnavigate?: (detail: { path: string }) => void;
    oncopyUrl?: (detail: { url: string }) => void;
  }

  let {
    files = [],
    loading = false,
    selectedFiles = new Set<string>(),
    bucket = '',
    onselect,
    ondownload,
    ondelete,
    onnavigate,
    oncopyUrl
  }: Props = $props();

  // Context menu state
  let contextMenu = $state({ show: false, x: 0, y: 0, file: null as ObjectInfo | null });
  let contextMenuJustOpened = $state(false);

  function formatSize(bytes: number): string {
    if (bytes === 0) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i];
  }

  function formatDate(timestamp: number): string {
    return new Date(timestamp * 1000).toLocaleDateString(undefined, {
      year: 'numeric',
      month: 'short',
      day: 'numeric',
      hour: '2-digit',
      minute: '2-digit',
    });
  }

  function getFileName(path: string): string {
    return path.split('/').pop() || path;
  }

  function isFolder(file: ObjectInfo): boolean {
    return file.Name.endsWith('/');
  }

  function handleClick(file: ObjectInfo) {
    if (isFolder(file)) {
      onnavigate?.({ path: file.Name });
    } else {
      onselect?.({ file });
    }
  }

  function handleDoubleClick(file: ObjectInfo) {
    if (!isFolder(file)) {
      ondownload?.({ file });
    }
  }

  function handleContextMenu(e: MouseEvent, file: ObjectInfo) {
    e.preventDefault();
    e.stopPropagation();
    contextMenuJustOpened = true;
    contextMenu = {
      show: true,
      x: e.clientX,
      y: e.clientY,
      file
    };
    // Reset flag after a short delay to allow click handler to ignore immediate clicks
    setTimeout(() => {
      contextMenuJustOpened = false;
    }, 100);
  }

  function closeContextMenu() {
    contextMenu = { show: false, x: 0, y: 0, file: null };
  }

  function handleContextDownload() {
    if (contextMenu.file) {
      ondownload?.({ file: contextMenu.file });
    }
    closeContextMenu();
  }

  function handleContextDelete() {
    if (contextMenu.file) {
      ondelete?.({ file: contextMenu.file });
    }
    closeContextMenu();
  }

  function handleContextCopyUrl() {
    if (contextMenu.file && bucket) {
      const url = api.getDownloadUrl(bucket, contextMenu.file.Name);
      navigator.clipboard.writeText(url).then(() => {
        oncopyUrl?.({ url });
      });
    }
    closeContextMenu();
  }

  // Close context menu on click outside
  function handleWindowClick(e: MouseEvent) {
    // Only close on left click and not immediately after opening
    if (contextMenu.show && !contextMenuJustOpened && e.button === 0) {
      closeContextMenu();
    }
  }

  onMount(() => {
    window.addEventListener('click', handleWindowClick);
  });

  onDestroy(() => {
    window.removeEventListener('click', handleWindowClick);
  });
</script>

<div class="file-list">
  {#if loading}
    <div class="loading">
      <div class="spinner"></div>
      <span>Loading files...</span>
    </div>
  {:else if !files || files.length === 0}
    <div class="empty">
      <svg class="empty-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
        <path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z" />
      </svg>
      <p>No files found</p>
    </div>
  {:else}
    <table>
      <thead>
        <tr>
          <th class="col-name">Name</th>
          <th class="col-size">Size</th>
          <th class="col-type">Type</th>
          <th class="col-modified">Modified</th>
          <th class="col-actions">Actions</th>
        </tr>
      </thead>
      <tbody>
        {#each files || [] as file (file.Name)}
          <!-- svelte-ignore a11y_click_events_have_key_events a11y_no_noninteractive_element_interactions -->
          <tr
            class:selected={selectedFiles.has(file.Name)}
            onclick={() => handleClick(file)}
            ondblclick={() => handleDoubleClick(file)}
            oncontextmenu={(e) => handleContextMenu(e, file)}
          >
            <td class="col-name">
              <div class="file-name">
                {#if isFolder(file)}
                  <svg class="file-icon folder" viewBox="0 0 24 24" fill="currentColor">
                    <path d="M10 4H4c-1.1 0-2 .9-2 2v12c0 1.1.9 2 2 2h16c1.1 0 2-.9 2-2V8c0-1.1-.9-2-2-2h-8l-2-2z" />
                  </svg>
                {:else}
                  <svg class="file-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z" />
                    <polyline points="14 2 14 8 20 8" />
                  </svg>
                {/if}
                <span>{getFileName(file.Name)}</span>
              </div>
            </td>
            <td class="col-size">{isFolder(file) ? '-' : formatSize(file.Size)}</td>
            <td class="col-type">{file.ContentType || '-'}</td>
            <td class="col-modified">{formatDate(file.Timestamp)}</td>
            <td class="col-actions">
              {#if !isFolder(file)}
                <button
                  class="action-btn"
                  title="Download"
                  onclick={(e) => { e.stopPropagation(); ondownload?.({ file }); }}
                >
                  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4" />
                    <polyline points="7 10 12 15 17 10" />
                    <line x1="12" y1="15" x2="12" y2="3" />
                  </svg>
                </button>
                <button
                  class="action-btn delete"
                  title="Delete"
                  onclick={(e) => { e.stopPropagation(); ondelete?.({ file }); }}
                >
                  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <polyline points="3 6 5 6 21 6" />
                    <path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2" />
                  </svg>
                </button>
              {/if}
            </td>
          </tr>
        {/each}
      </tbody>
    </table>
  {/if}
</div>

<!-- Context Menu -->
{#if contextMenu.show && contextMenu.file}
  <!-- svelte-ignore a11y_no_noninteractive_tabindex -->
  <div
    class="context-menu"
    role="menu"
    tabindex="-1"
    style="left: {contextMenu.x}px; top: {contextMenu.y}px;"
    onclick={(e) => e.stopPropagation()}
    onkeydown={(e) => e.key === 'Escape' && closeContextMenu()}
  >
    {#if !isFolder(contextMenu.file)}
      <div class="context-menu-item" role="menuitem" tabindex="0" onclick={handleContextDownload} onkeydown={(e) => e.key === 'Enter' && handleContextDownload()}>
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4" />
          <polyline points="7 10 12 15 17 10" />
          <line x1="12" y1="15" x2="12" y2="3" />
        </svg>
        <span>Download</span>
      </div>
      <div class="context-menu-item" role="menuitem" tabindex="0" onclick={handleContextCopyUrl} onkeydown={(e) => e.key === 'Enter' && handleContextCopyUrl()}>
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <rect x="9" y="9" width="13" height="13" rx="2" ry="2" />
          <path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1" />
        </svg>
        <span>Copy URL</span>
      </div>
      <div class="context-menu-divider" role="separator"></div>
      <div class="context-menu-item delete" role="menuitem" tabindex="0" onclick={handleContextDelete} onkeydown={(e) => e.key === 'Enter' && handleContextDelete()}>
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <polyline points="3 6 5 6 21 6" />
          <path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2" />
        </svg>
        <span>Delete</span>
      </div>
    {:else}
      <div class="context-menu-item" role="menuitem" tabindex="0" onclick={() => { onnavigate?.({ path: contextMenu.file?.Name || '' }); closeContextMenu(); }} onkeydown={(e) => { if (e.key === 'Enter') { onnavigate?.({ path: contextMenu.file?.Name || '' }); closeContextMenu(); } }}>
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z" />
        </svg>
        <span>Open folder</span>
      </div>
      <div class="context-menu-divider" role="separator"></div>
      <div class="context-menu-item delete" role="menuitem" tabindex="0" onclick={handleContextDelete} onkeydown={(e) => e.key === 'Enter' && handleContextDelete()}>
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <polyline points="3 6 5 6 21 6" />
          <path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2" />
        </svg>
        <span>Delete folder</span>
      </div>
    {/if}
  </div>
{/if}

<style>
  .file-list {
    border: 1px solid var(--border-color, #ddd);
    border-radius: 8px;
    overflow: hidden;
  }

  table {
    width: 100%;
    border-collapse: collapse;
    font-size: 0.9rem;
  }

  thead {
    background: var(--bg-secondary, #f5f5f5);
    position: sticky;
    top: 0;
  }

  th {
    text-align: left;
    padding: 0.75rem 1rem;
    font-weight: 600;
    color: var(--text-secondary, #666);
    border-bottom: 1px solid var(--border-color, #ddd);
  }

  td {
    padding: 0.75rem 1rem;
    border-bottom: 1px solid var(--border-color, #eee);
  }

  tr:hover {
    background: var(--bg-hover, #f9f9f9);
  }

  tr.selected {
    background: var(--bg-selected, #e3f2fd);
  }

  .col-name {
    width: 40%;
  }

  .col-size {
    width: 15%;
  }

  .col-type {
    width: 20%;
  }

  .col-modified {
    width: 20%;
  }

  .col-actions {
    width: 5%;
    text-align: right;
  }

  .file-name {
    display: flex;
    align-items: center;
    gap: 0.5rem;
  }

  .file-icon {
    width: 20px;
    height: 20px;
    flex-shrink: 0;
    color: var(--text-secondary, #666);
  }

  .file-icon.folder {
    color: var(--folder-color, #ffc107);
  }

  .action-btn {
    background: none;
    border: none;
    padding: 0.25rem;
    cursor: pointer;
    color: var(--text-secondary, #666);
    border-radius: 4px;
    transition: all 0.2s;
  }

  .action-btn:hover {
    background: var(--bg-hover, #e0e0e0);
    color: var(--primary-color, #1976d2);
  }

  .action-btn.delete:hover {
    color: var(--error-color, #d32f2f);
  }

  .action-btn svg {
    width: 18px;
    height: 18px;
  }

  .loading,
  .empty {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    padding: 3rem;
    color: var(--text-secondary, #666);
  }

  .spinner {
    width: 32px;
    height: 32px;
    border: 3px solid var(--border-color, #ddd);
    border-top-color: var(--primary-color, #1976d2);
    border-radius: 50%;
    animation: spin 1s linear infinite;
  }

  @keyframes spin {
    to {
      transform: rotate(360deg);
    }
  }

  .empty-icon {
    width: 48px;
    height: 48px;
    margin-bottom: 1rem;
  }

  /* Context menu styles */
  .context-menu {
    position: fixed;
    background: var(--bg-primary, white);
    border: 1px solid var(--border-color, #ddd);
    border-radius: 8px;
    box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
    min-width: 160px;
    z-index: 1000;
    padding: 0.5rem 0;
  }

  .context-menu-item {
    display: flex;
    align-items: center;
    gap: 0.75rem;
    padding: 0.5rem 1rem;
    cursor: pointer;
    transition: background 0.15s;
    color: var(--text-primary, #333);
  }

  .context-menu-item:hover {
    background: var(--bg-hover, #f5f5f5);
  }

  .context-menu-item.delete:hover {
    background: var(--error-bg, #ffebee);
    color: var(--error-color, #d32f2f);
  }

  .context-menu-item svg {
    width: 16px;
    height: 16px;
    color: var(--text-secondary, #666);
  }

  .context-menu-item.delete svg {
    color: inherit;
  }

  .context-menu-divider {
    height: 1px;
    background: var(--border-color, #eee);
    margin: 0.5rem 0;
  }
</style>
