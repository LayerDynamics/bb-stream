<script lang="ts">
  import { invoke } from '@tauri-apps/api/core';
  import { resetApiPort } from '../api';

  interface Props {
    status: 'starting' | 'healthy' | 'unhealthy' | 'crashed' | 'restarting';
    error?: string;
  }

  let { status, error }: Props = $props();

  let countdown = $state(3);
  let countdownInterval: ReturnType<typeof setInterval> | null = null;

  $effect(() => {
    // Start countdown for auto-restart when crashed
    if (status === 'crashed' && countdown > 0) {
      countdownInterval = setInterval(() => {
        countdown--;
        if (countdown <= 0 && countdownInterval) {
          clearInterval(countdownInterval);
        }
      }, 1000);
    }

    return () => {
      if (countdownInterval) {
        clearInterval(countdownInterval);
      }
    };
  });

  function handleRestart() {
    resetApiPort();
    invoke('restart_backend');
  }

  function handleQuit() {
    // Close the app
    window.close();
  }
</script>

{#if status !== 'healthy'}
  <div class="overlay">
    <div class="content">
      {#if status === 'starting'}
        <div class="icon spinning">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M21 12a9 9 0 11-6.219-8.56" />
          </svg>
        </div>
        <h2>Starting BB-Stream...</h2>
        <p>Please wait while the backend initializes</p>
      {:else if status === 'unhealthy'}
        <div class="icon warning">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M10.29 3.86L1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z" />
            <line x1="12" y1="9" x2="12" y2="13" />
            <line x1="12" y1="17" x2="12.01" y2="17" />
          </svg>
        </div>
        <h2>Connection Lost</h2>
        <p>Attempting to reconnect to backend...</p>
        <div class="icon spinning small">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M21 12a9 9 0 11-6.219-8.56" />
          </svg>
        </div>
      {:else if status === 'crashed'}
        <div class="icon error">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <circle cx="12" cy="12" r="10" />
            <line x1="15" y1="9" x2="9" y2="15" />
            <line x1="9" y1="9" x2="15" y2="15" />
          </svg>
        </div>
        <h2>Backend Crashed</h2>
        {#if error}
          <p class="error-message">{error}</p>
        {/if}
        <p class="countdown">Auto-restarting in {countdown}s...</p>
        <div class="actions">
          <button class="btn primary" onclick={handleRestart}>Restart Now</button>
          <button class="btn secondary" onclick={handleQuit}>Quit</button>
        </div>
      {:else if status === 'restarting'}
        <div class="icon spinning">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M21 12a9 9 0 11-6.219-8.56" />
          </svg>
        </div>
        <h2>Restarting Backend...</h2>
        <p>Please wait while the backend restarts</p>
      {/if}
    </div>
  </div>
{/if}

<style>
  .overlay {
    position: fixed;
    top: 0;
    left: 0;
    right: 0;
    bottom: 0;
    background: rgba(0, 0, 0, 0.85);
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 10000;
    backdrop-filter: blur(4px);
  }

  .content {
    text-align: center;
    padding: 2rem;
    max-width: 400px;
  }

  .icon {
    width: 64px;
    height: 64px;
    margin: 0 auto 1.5rem;
  }

  .icon.small {
    width: 24px;
    height: 24px;
    margin-top: 1rem;
  }

  .icon svg {
    width: 100%;
    height: 100%;
  }

  .icon.spinning svg {
    animation: spin 1s linear infinite;
    stroke: var(--primary-color, #3b82f6);
  }

  .icon.warning svg {
    stroke: #f59e0b;
  }

  .icon.error svg {
    stroke: #ef4444;
  }

  @keyframes spin {
    from {
      transform: rotate(0deg);
    }
    to {
      transform: rotate(360deg);
    }
  }

  h2 {
    color: white;
    font-size: 1.5rem;
    margin: 0 0 0.5rem;
    font-weight: 600;
  }

  p {
    color: rgba(255, 255, 255, 0.7);
    margin: 0 0 0.5rem;
    line-height: 1.5;
  }

  .error-message {
    background: rgba(239, 68, 68, 0.2);
    border: 1px solid rgba(239, 68, 68, 0.3);
    border-radius: 8px;
    padding: 0.75rem 1rem;
    color: #fca5a5;
    font-family: monospace;
    font-size: 0.875rem;
    margin: 1rem 0;
    word-break: break-word;
  }

  .countdown {
    color: rgba(255, 255, 255, 0.5);
    font-size: 0.875rem;
  }

  .actions {
    display: flex;
    gap: 1rem;
    justify-content: center;
    margin-top: 1.5rem;
  }

  .btn {
    padding: 0.75rem 1.5rem;
    border-radius: 8px;
    border: none;
    font-size: 0.875rem;
    font-weight: 500;
    cursor: pointer;
    transition: all 0.15s ease;
  }

  .btn.primary {
    background: var(--primary-color, #3b82f6);
    color: white;
  }

  .btn.primary:hover {
    background: var(--primary-hover, #2563eb);
  }

  .btn.secondary {
    background: rgba(255, 255, 255, 0.1);
    color: white;
    border: 1px solid rgba(255, 255, 255, 0.2);
  }

  .btn.secondary:hover {
    background: rgba(255, 255, 255, 0.15);
  }
</style>
