<script lang="ts">
  import Toast from './Toast.svelte';
  import { toasts, removeToast } from '../stores/toasts';
</script>

<div class="toast-container">
  {#each $toasts as toast (toast.id)}
    <Toast
      id={toast.id}
      message={toast.message}
      type={toast.type}
      on:dismiss={(e) => removeToast(e.detail.id)}
    />
  {/each}
</div>

<style>
  .toast-container {
    position: fixed;
    bottom: 1.5rem;
    right: 1.5rem;
    z-index: 9999;
    display: flex;
    flex-direction: column-reverse;
    gap: 0.75rem;
    pointer-events: none;
  }
</style>
