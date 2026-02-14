<script lang="ts">
  import { onMount } from 'svelte';
  import api from '../api';

  interface Props {
    open?: boolean;
    onclose?: () => void;
    onsaved?: () => void;
  }

  let {
    open = false,
    onclose,
    onsaved
  }: Props = $props();

  let keyId = $state('');
  let applicationKey = $state('');
  let defaultBucket = $state('');
  let saving = $state(false);
  let error = $state('');
  let success = $state('');
  let configured = $state(false);

  onMount(async () => {
    await loadConfig();
  });

  async function loadConfig() {
    try {
      const config = await api.getConfig();
      keyId = config.key_id || '';
      defaultBucket = config.default_bucket || '';
      configured = config.configured;
      // Don't load application key - it's not returned for security
    } catch (e: any) {
      error = 'Failed to load configuration';
    }
  }

  async function handleSave() {
    error = '';
    success = '';
    saving = true;

    try {
      // Only send app key if it was entered (not empty)
      const payload: any = {
        key_id: keyId,
        default_bucket: defaultBucket
      };

      if (applicationKey) {
        payload.application_key = applicationKey;
      }

      await api.setConfig(payload);
      success = 'Settings saved successfully!';
      configured = true;
      applicationKey = ''; // Clear the password field
      onsaved?.();

      // Close modal after short delay
      setTimeout(() => {
        onclose?.();
      }, 1500);
    } catch (e: any) {
      error = e.message || 'Failed to save settings';
    } finally {
      saving = false;
    }
  }

  function handleKeydown(e: KeyboardEvent) {
    if (e.key === 'Escape') {
      onclose?.();
    }
  }
</script>

<svelte:window onkeydown={handleKeydown} />

{#if open}
  <!-- svelte-ignore a11y_no_noninteractive_element_interactions -->
  <div class="modal-overlay" onclick={() => onclose?.()} role="dialog" aria-modal="true">
    <!-- svelte-ignore a11y_click_events_have_key_events a11y_no_static_element_interactions -->
    <div class="modal" onclick={(e) => e.stopPropagation()}>
      <div class="modal-header">
        <h2>Settings</h2>
        <button class="close-btn" onclick={() => onclose?.()} aria-label="Close">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <line x1="18" y1="6" x2="6" y2="18" />
            <line x1="6" y1="6" x2="18" y2="18" />
          </svg>
        </button>
      </div>

      <div class="modal-body">
        {#if error}
          <div class="alert error">{error}</div>
        {/if}
        {#if success}
          <div class="alert success">{success}</div>
        {/if}

        <div class="section">
          <h3>Backblaze B2 Credentials</h3>
          <p class="description">
            Enter your Backblaze B2 API credentials. You can find these in the
            <a href="https://secure.backblaze.com/app_keys.htm" target="_blank" rel="noopener">
              Backblaze B2 Cloud Storage App Keys
            </a> page.
          </p>

          <div class="form-group">
            <label for="keyId">Key ID</label>
            <input
              type="text"
              id="keyId"
              bind:value={keyId}
              placeholder="Enter your B2 Key ID"
              autocomplete="username"
            />
          </div>

          <div class="form-group">
            <label for="appKey">
              Application Key
              {#if configured}
                <span class="hint">(leave blank to keep existing)</span>
              {/if}
            </label>
            <input
              type="password"
              id="appKey"
              bind:value={applicationKey}
              placeholder={configured ? '••••••••••••••••' : 'Enter your B2 Application Key'}
              autocomplete="current-password"
            />
          </div>

          <div class="form-group">
            <label for="defaultBucket">Default Bucket (optional)</label>
            <input
              type="text"
              id="defaultBucket"
              bind:value={defaultBucket}
              placeholder="Enter default bucket name"
            />
          </div>
        </div>

        <div class="status">
          {#if configured}
            <span class="status-badge configured">Configured</span>
          {:else}
            <span class="status-badge not-configured">Not Configured</span>
          {/if}
        </div>
      </div>

      <div class="modal-footer">
        <button class="btn secondary" onclick={() => onclose?.()} disabled={saving}>
          Cancel
        </button>
        <button class="btn primary" onclick={handleSave} disabled={saving || !keyId}>
          {#if saving}
            Saving...
          {:else}
            Save Settings
          {/if}
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
    z-index: 1000;
    backdrop-filter: blur(4px);
  }

  .modal {
    background: var(--bg-primary, white);
    border-radius: 12px;
    box-shadow: 0 20px 60px rgba(0, 0, 0, 0.3);
    width: 90%;
    max-width: 500px;
    max-height: 90vh;
    overflow: hidden;
    display: flex;
    flex-direction: column;
  }

  .modal-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 1.25rem 1.5rem;
    border-bottom: 1px solid var(--border-color, #eee);
  }

  .modal-header h2 {
    margin: 0;
    font-size: 1.25rem;
    font-weight: 600;
  }

  .close-btn {
    background: none;
    border: none;
    padding: 0.5rem;
    cursor: pointer;
    color: var(--text-secondary, #666);
    border-radius: 6px;
    transition: all 0.2s;
  }

  .close-btn:hover {
    background: var(--bg-hover, #f0f0f0);
    color: var(--text-primary, #333);
  }

  .close-btn svg {
    width: 20px;
    height: 20px;
  }

  .modal-body {
    padding: 1.5rem;
    overflow-y: auto;
  }

  .section {
    margin-bottom: 1.5rem;
  }

  .section h3 {
    margin: 0 0 0.5rem;
    font-size: 1rem;
    font-weight: 600;
  }

  .description {
    margin: 0 0 1rem;
    font-size: 0.875rem;
    color: var(--text-secondary, #666);
    line-height: 1.5;
  }

  .description a {
    color: var(--primary-color, #1976d2);
    text-decoration: none;
  }

  .description a:hover {
    text-decoration: underline;
  }

  .form-group {
    margin-bottom: 1rem;
  }

  .form-group label {
    display: block;
    margin-bottom: 0.5rem;
    font-size: 0.875rem;
    font-weight: 500;
    color: var(--text-primary, #333);
  }

  .hint {
    font-weight: 400;
    color: var(--text-secondary, #666);
    font-size: 0.75rem;
  }

  .form-group input {
    width: 100%;
    padding: 0.75rem;
    border: 1px solid var(--border-color, #ddd);
    border-radius: 8px;
    font-size: 0.9375rem;
    transition: all 0.2s;
    box-sizing: border-box;
  }

  .form-group input:focus {
    outline: none;
    border-color: var(--primary-color, #1976d2);
    box-shadow: 0 0 0 3px rgba(25, 118, 210, 0.1);
  }

  .form-group input::placeholder {
    color: var(--text-tertiary, #999);
  }

  .status {
    display: flex;
    justify-content: flex-end;
  }

  .status-badge {
    padding: 0.375rem 0.75rem;
    border-radius: 20px;
    font-size: 0.75rem;
    font-weight: 500;
  }

  .status-badge.configured {
    background: #e8f5e9;
    color: #2e7d32;
  }

  .status-badge.not-configured {
    background: #fff3e0;
    color: #ef6c00;
  }

  .alert {
    padding: 0.75rem 1rem;
    border-radius: 8px;
    margin-bottom: 1rem;
    font-size: 0.875rem;
  }

  .alert.error {
    background: #ffebee;
    color: #c62828;
    border: 1px solid #ffcdd2;
  }

  .alert.success {
    background: #e8f5e9;
    color: #2e7d32;
    border: 1px solid #c8e6c9;
  }

  .modal-footer {
    display: flex;
    justify-content: flex-end;
    gap: 0.75rem;
    padding: 1rem 1.5rem;
    border-top: 1px solid var(--border-color, #eee);
    background: var(--bg-secondary, #fafafa);
  }

  .btn {
    padding: 0.625rem 1.25rem;
    border-radius: 8px;
    font-size: 0.9375rem;
    font-weight: 500;
    cursor: pointer;
    transition: all 0.2s;
  }

  .btn:disabled {
    opacity: 0.6;
    cursor: not-allowed;
  }

  .btn.primary {
    background: var(--primary-color, #1976d2);
    color: white;
    border: none;
  }

  .btn.primary:hover:not(:disabled) {
    background: var(--primary-dark, #1565c0);
  }

  .btn.secondary {
    background: transparent;
    color: var(--text-primary, #333);
    border: 1px solid var(--border-color, #ddd);
  }

  .btn.secondary:hover:not(:disabled) {
    background: var(--bg-hover, #f0f0f0);
  }
</style>
