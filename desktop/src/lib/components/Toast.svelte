<script lang="ts">
  import { createEventDispatcher } from 'svelte';
  import { fly, fade } from 'svelte/transition';
  import type { ToastType } from '../stores/toasts';

  export let id: string;
  export let message: string;
  export let type: ToastType = 'info';

  const dispatch = createEventDispatcher<{
    dismiss: { id: string };
  }>();

  const icons = {
    success: `<polyline points="20 6 9 17 4 12" />`,
    error: `<circle cx="12" cy="12" r="10" /><line x1="15" y1="9" x2="9" y2="15" /><line x1="9" y1="9" x2="15" y2="15" />`,
    warning: `<path d="M10.29 3.86L1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z" /><line x1="12" y1="9" x2="12" y2="13" /><line x1="12" y1="17" x2="12.01" y2="17" />`,
    info: `<circle cx="12" cy="12" r="10" /><line x1="12" y1="16" x2="12" y2="12" /><line x1="12" y1="8" x2="12.01" y2="8" />`,
  };
</script>

<div
  class="toast {type}"
  in:fly={{ x: 300, duration: 300 }}
  out:fade={{ duration: 200 }}
  role="alert"
>
  <div class="toast-icon">
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
      {@html icons[type]}
    </svg>
  </div>
  <div class="toast-message">{message}</div>
  <button
    class="toast-dismiss"
    on:click={() => dispatch('dismiss', { id })}
    aria-label="Dismiss"
  >
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
      <line x1="18" y1="6" x2="6" y2="18" />
      <line x1="6" y1="6" x2="18" y2="18" />
    </svg>
  </button>
</div>

<style>
  .toast {
    display: flex;
    align-items: center;
    gap: 0.75rem;
    padding: 0.875rem 1rem;
    background: white;
    border-radius: 8px;
    box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
    max-width: 360px;
    pointer-events: auto;
  }

  .toast.success {
    border-left: 4px solid var(--success-color, #4caf50);
  }

  .toast.error {
    border-left: 4px solid var(--error-color, #f44336);
  }

  .toast.warning {
    border-left: 4px solid var(--warning-color, #ff9800);
  }

  .toast.info {
    border-left: 4px solid var(--info-color, #2196f3);
  }

  .toast-icon {
    flex-shrink: 0;
  }

  .toast-icon svg {
    width: 20px;
    height: 20px;
  }

  .toast.success .toast-icon {
    color: var(--success-color, #4caf50);
  }

  .toast.error .toast-icon {
    color: var(--error-color, #f44336);
  }

  .toast.warning .toast-icon {
    color: var(--warning-color, #ff9800);
  }

  .toast.info .toast-icon {
    color: var(--info-color, #2196f3);
  }

  .toast-message {
    flex: 1;
    font-size: 0.9rem;
    color: var(--text-primary, #333);
    line-height: 1.4;
  }

  .toast-dismiss {
    flex-shrink: 0;
    background: none;
    border: none;
    padding: 0.25rem;
    cursor: pointer;
    color: var(--text-secondary, #666);
    border-radius: 4px;
    opacity: 0.5;
    transition: opacity 0.2s;
  }

  .toast-dismiss:hover {
    opacity: 1;
  }

  .toast-dismiss svg {
    width: 16px;
    height: 16px;
    display: block;
  }
</style>
