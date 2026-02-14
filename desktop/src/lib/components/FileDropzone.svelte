<script lang="ts">
  import { createEventDispatcher } from 'svelte';

  export let disabled = false;

  const dispatch = createEventDispatcher<{
    drop: { files: File[] };
  }>();

  let isDragging = false;
  let dragCounter = 0;

  function handleDragEnter(e: DragEvent) {
    e.preventDefault();
    dragCounter++;
    isDragging = true;
  }

  function handleDragLeave(e: DragEvent) {
    e.preventDefault();
    dragCounter--;
    if (dragCounter === 0) {
      isDragging = false;
    }
  }

  function handleDragOver(e: DragEvent) {
    e.preventDefault();
  }

  function handleDrop(e: DragEvent) {
    e.preventDefault();
    isDragging = false;
    dragCounter = 0;

    if (disabled) return;

    const files = Array.from(e.dataTransfer?.files || []);
    if (files.length > 0) {
      dispatch('drop', { files });
    }
  }

  function handleClick() {
    if (disabled) return;
    const input = document.createElement('input');
    input.type = 'file';
    input.multiple = true;
    input.onchange = () => {
      const files = Array.from(input.files || []);
      if (files.length > 0) {
        dispatch('drop', { files });
      }
    };
    input.click();
  }
</script>

<div
  class="dropzone"
  class:dragging={isDragging}
  class:disabled
  on:dragenter={handleDragEnter}
  on:dragleave={handleDragLeave}
  on:dragover={handleDragOver}
  on:drop={handleDrop}
  on:click={handleClick}
  on:keydown={(e) => e.key === 'Enter' && handleClick()}
  role="button"
  tabindex="0"
>
  <slot>
    <div class="dropzone-content">
      <svg class="upload-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
        <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4" />
        <polyline points="17 8 12 3 7 8" />
        <line x1="12" y1="3" x2="12" y2="15" />
      </svg>
      <p class="dropzone-text">
        {#if isDragging}
          Drop files here
        {:else}
          Drag & drop files here or click to browse
        {/if}
      </p>
    </div>
  </slot>
</div>

<style>
  .dropzone {
    border: 2px dashed var(--border-color, #ccc);
    border-radius: 8px;
    padding: 2rem;
    text-align: center;
    cursor: pointer;
    transition: all 0.2s ease;
    background: var(--bg-secondary, #f9f9f9);
  }

  .dropzone:hover:not(.disabled) {
    border-color: var(--primary-color, #4a90d9);
    background: var(--bg-hover, #f0f7ff);
  }

  .dropzone.dragging {
    border-color: var(--primary-color, #4a90d9);
    background: var(--bg-hover, #f0f7ff);
    transform: scale(1.02);
  }

  .dropzone.disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .dropzone-content {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 1rem;
  }

  .upload-icon {
    width: 48px;
    height: 48px;
    color: var(--text-secondary, #666);
  }

  .dropzone-text {
    margin: 0;
    color: var(--text-secondary, #666);
    font-size: 0.9rem;
  }
</style>
