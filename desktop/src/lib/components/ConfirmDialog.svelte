<script lang="ts">
  interface Props {
    open?: boolean;
    title?: string;
    message?: string;
    confirmLabel?: string;
    cancelLabel?: string;
    variant?: 'default' | 'danger';
    onconfirm?: () => void;
    oncancel?: () => void;
  }

  let {
    open = false,
    title = 'Confirm',
    message = 'Are you sure?',
    confirmLabel = 'Confirm',
    cancelLabel = 'Cancel',
    variant = 'default',
    onconfirm,
    oncancel
  }: Props = $props();

  function handleKeydown(e: KeyboardEvent) {
    if (e.key === 'Escape') {
      oncancel?.();
    }
  }

  function handleConfirm() {
    onconfirm?.();
  }

  function handleCancel() {
    oncancel?.();
  }
</script>

<svelte:window onkeydown={handleKeydown} />

{#if open}
  <!-- svelte-ignore a11y_no_noninteractive_element_interactions -->
  <div class="modal-overlay" onclick={handleCancel} role="dialog" aria-modal="true" aria-labelledby="confirm-title">
    <!-- svelte-ignore a11y_click_events_have_key_events a11y_no_static_element_interactions -->
    <div class="modal" onclick={(e) => e.stopPropagation()}>
      <div class="modal-header">
        <div class="icon-container" class:danger={variant === 'danger'}>
          {#if variant === 'danger'}
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M10.29 3.86L1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z" />
              <line x1="12" y1="9" x2="12" y2="13" />
              <line x1="12" y1="17" x2="12.01" y2="17" />
            </svg>
          {:else}
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <circle cx="12" cy="12" r="10" />
              <path d="M9.09 9a3 3 0 0 1 5.83 1c0 2-3 3-3 3" />
              <line x1="12" y1="17" x2="12.01" y2="17" />
            </svg>
          {/if}
        </div>
        <h2 id="confirm-title">{title}</h2>
      </div>

      <div class="modal-body">
        <p>{message}</p>
      </div>

      <div class="modal-footer">
        <button class="btn secondary" onclick={handleCancel}>
          {cancelLabel}
        </button>
        <button class="btn" class:danger={variant === 'danger'} class:primary={variant !== 'danger'} onclick={handleConfirm}>
          {confirmLabel}
        </button>
      </div>
    </div>
  </div>
{/if}

<style>
  .modal-overlay {
    position: fixed;
    top: 0;
    left: 0;
    right: 0;
    bottom: 0;
    background: rgba(0, 0, 0, 0.5);
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 1100;
    backdrop-filter: blur(4px);
  }

  .modal {
    background: var(--bg-primary, white);
    border-radius: 12px;
    box-shadow: 0 20px 60px rgba(0, 0, 0, 0.3);
    width: 90%;
    max-width: 400px;
    overflow: hidden;
    display: flex;
    flex-direction: column;
    animation: modalIn 0.2s ease-out;
  }

  @keyframes modalIn {
    from {
      opacity: 0;
      transform: scale(0.95);
    }
    to {
      opacity: 1;
      transform: scale(1);
    }
  }

  .modal-header {
    display: flex;
    flex-direction: column;
    align-items: center;
    padding: 1.5rem 1.5rem 1rem;
    text-align: center;
  }

  .icon-container {
    width: 48px;
    height: 48px;
    border-radius: 50%;
    display: flex;
    align-items: center;
    justify-content: center;
    margin-bottom: 1rem;
    background: var(--bg-secondary, #f5f5f5);
    color: var(--text-secondary, #666);
  }

  .icon-container.danger {
    background: #ffebee;
    color: #d32f2f;
  }

  .icon-container svg {
    width: 24px;
    height: 24px;
  }

  .modal-header h2 {
    margin: 0;
    font-size: 1.125rem;
    font-weight: 600;
    color: var(--text-primary, #333);
  }

  .modal-body {
    padding: 0 1.5rem 1.5rem;
    text-align: center;
  }

  .modal-body p {
    margin: 0;
    color: var(--text-secondary, #666);
    font-size: 0.9375rem;
    line-height: 1.5;
  }

  .modal-footer {
    display: flex;
    gap: 0.75rem;
    padding: 1rem 1.5rem;
    border-top: 1px solid var(--border-color, #eee);
    background: var(--bg-secondary, #fafafa);
  }

  .btn {
    flex: 1;
    padding: 0.75rem 1rem;
    border-radius: 8px;
    font-size: 0.9375rem;
    font-weight: 500;
    cursor: pointer;
    transition: all 0.2s;
  }

  .btn.primary {
    background: var(--primary-color, #1976d2);
    color: white;
    border: none;
  }

  .btn.primary:hover {
    background: var(--primary-dark, #1565c0);
  }

  .btn.danger {
    background: #d32f2f;
    color: white;
    border: none;
  }

  .btn.danger:hover {
    background: #c62828;
  }

  .btn.secondary {
    background: transparent;
    color: var(--text-primary, #333);
    border: 1px solid var(--border-color, #ddd);
  }

  .btn.secondary:hover {
    background: var(--bg-hover, #f0f0f0);
  }
</style>
