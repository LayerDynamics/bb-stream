<script lang="ts">
  import { onMount } from 'svelte';
  import api from '../api';

  interface Props {
    onconfigured?: () => void;
  }

  let { onconfigured }: Props = $props();

  let keyId = $state('');
  let applicationKey = $state('');
  let defaultBucket = $state('');
  let saving = $state(false);
  let error = $state('');
  let success = $state('');
  let step = $state(1);

  async function handleSave() {
    error = '';
    success = '';
    saving = true;

    try {
      await api.setConfig({
        key_id: keyId,
        application_key: applicationKey,
        default_bucket: defaultBucket || undefined,
      });
      success = 'Connected to Backblaze B2 successfully!';

      // Notify parent and transition to main app
      setTimeout(() => {
        onconfigured?.();
      }, 1500);
    } catch (e: any) {
      error = e.message || 'Failed to connect. Please check your credentials.';
    } finally {
      saving = false;
    }
  }

  function nextStep() {
    if (step < 3) step++;
  }

  function prevStep() {
    if (step > 1) step--;
  }
</script>

<div class="welcome-screen">
  <div class="welcome-container">
    <div class="logo">
      <svg viewBox="0 0 24 24" fill="currentColor">
        <path d="M19.35 10.04C18.67 6.59 15.64 4 12 4 9.11 4 6.6 5.64 5.35 8.04 2.34 8.36 0 10.91 0 14c0 3.31 2.69 6 6 6h13c2.76 0 5-2.24 5-5 0-2.64-2.05-4.78-4.65-4.96zM14 13v4h-4v-4H7l5-5 5 5h-3z"/>
      </svg>
      <h1>BB Stream</h1>
    </div>

    <p class="subtitle">Cloud storage streaming for Backblaze B2</p>

    <div class="steps-indicator">
      <div class="step" class:active={step >= 1} class:current={step === 1}>
        <span class="step-number">1</span>
        <span class="step-label">Welcome</span>
      </div>
      <div class="step-line" class:active={step >= 2}></div>
      <div class="step" class:active={step >= 2} class:current={step === 2}>
        <span class="step-number">2</span>
        <span class="step-label">Credentials</span>
      </div>
      <div class="step-line" class:active={step >= 3}></div>
      <div class="step" class:active={step >= 3} class:current={step === 3}>
        <span class="step-number">3</span>
        <span class="step-label">Connect</span>
      </div>
    </div>

    {#if step === 1}
      <div class="step-content">
        <h2>Welcome to BB Stream</h2>
        <p>
          BB Stream provides a seamless way to manage your Backblaze B2 cloud storage.
          Upload, download, sync, and watch files with an intuitive desktop interface.
        </p>
        <div class="features">
          <div class="feature">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4" />
              <polyline points="17 8 12 3 7 8" />
              <line x1="12" y1="3" x2="12" y2="15" />
            </svg>
            <span>Upload & Download</span>
          </div>
          <div class="feature">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M23 4v6h-6M1 20v-6h6" />
              <path d="M3.51 9a9 9 0 0 1 14.85-3.36L23 10M1 14l4.64 4.36A9 9 0 0 0 20.49 15" />
            </svg>
            <span>Bidirectional Sync</span>
          </div>
          <div class="feature">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <circle cx="12" cy="12" r="10" />
              <polyline points="12 6 12 12 16 14" />
            </svg>
            <span>Real-time Watch</span>
          </div>
        </div>
        <button class="btn primary" onclick={nextStep}>Get Started</button>
      </div>
    {:else if step === 2}
      <div class="step-content">
        <h2>Get Your API Keys</h2>
        <p>
          To connect BB Stream to your Backblaze B2 account, you'll need to create
          an application key.
        </p>
        <ol class="instructions">
          <li>
            Go to the
            <a href="https://secure.backblaze.com/app_keys.htm" target="_blank" rel="noopener">
              Backblaze B2 App Keys page
            </a>
          </li>
          <li>Click "Add a New Application Key"</li>
          <li>Give it a name like "BB Stream Desktop"</li>
          <li>Set the access permissions (all buckets recommended)</li>
          <li>Click "Create New Key"</li>
          <li>Copy the Key ID and Application Key</li>
        </ol>
        <div class="note">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <circle cx="12" cy="12" r="10" />
            <line x1="12" y1="16" x2="12" y2="12" />
            <line x1="12" y1="8" x2="12.01" y2="8" />
          </svg>
          <span>The Application Key is only shown once. Make sure to copy it!</span>
        </div>
        <div class="button-group">
          <button class="btn secondary" onclick={prevStep}>Back</button>
          <button class="btn primary" onclick={nextStep}>I Have My Keys</button>
        </div>
      </div>
    {:else if step === 3}
      <div class="step-content">
        <h2>Enter Your Credentials</h2>

        {#if error}
          <div class="alert error">{error}</div>
        {/if}
        {#if success}
          <div class="alert success">{success}</div>
        {/if}

        <form onsubmit={(e) => { e.preventDefault(); handleSave(); }}>
          <div class="form-group">
            <label for="keyId">Key ID</label>
            <input
              type="text"
              id="keyId"
              bind:value={keyId}
              placeholder="Enter your B2 Key ID"
              autocomplete="username"
              required
            />
          </div>

          <div class="form-group">
            <label for="appKey">Application Key</label>
            <input
              type="password"
              id="appKey"
              bind:value={applicationKey}
              placeholder="Enter your B2 Application Key"
              autocomplete="current-password"
              required
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

          <div class="button-group">
            <button type="button" class="btn secondary" onclick={prevStep} disabled={saving}>
              Back
            </button>
            <button type="submit" class="btn primary" disabled={saving || !keyId || !applicationKey}>
              {#if saving}
                Connecting...
              {:else}
                Connect to B2
              {/if}
            </button>
          </div>
        </form>
      </div>
    {/if}
  </div>
</div>

<style>
  .welcome-screen {
    position: fixed;
    top: 0;
    left: 0;
    right: 0;
    bottom: 0;
    background: linear-gradient(135deg, #1976d2 0%, #0d47a1 100%);
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 9999;
  }

  .welcome-container {
    background: white;
    border-radius: 16px;
    padding: 2.5rem;
    max-width: 500px;
    width: 90%;
    box-shadow: 0 20px 60px rgba(0, 0, 0, 0.3);
    text-align: center;
  }

  .logo {
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 0.75rem;
    margin-bottom: 0.5rem;
    color: #1976d2;
  }

  .logo svg {
    width: 48px;
    height: 48px;
  }

  .logo h1 {
    margin: 0;
    font-size: 2rem;
    font-weight: 700;
  }

  .subtitle {
    color: var(--text-secondary, #666);
    margin: 0 0 2rem;
    font-size: 1rem;
  }

  .steps-indicator {
    display: flex;
    align-items: center;
    justify-content: center;
    margin-bottom: 2rem;
  }

  .step {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 0.25rem;
    opacity: 0.4;
    transition: opacity 0.2s;
  }

  .step.active {
    opacity: 1;
  }

  .step-number {
    width: 32px;
    height: 32px;
    border-radius: 50%;
    background: #e0e0e0;
    display: flex;
    align-items: center;
    justify-content: center;
    font-weight: 600;
    font-size: 0.875rem;
    transition: background 0.2s, color 0.2s;
  }

  .step.current .step-number {
    background: #1976d2;
    color: white;
  }

  .step.active:not(.current) .step-number {
    background: #4caf50;
    color: white;
  }

  .step-label {
    font-size: 0.75rem;
    color: var(--text-secondary, #666);
  }

  .step-line {
    width: 60px;
    height: 2px;
    background: #e0e0e0;
    margin: 0 0.5rem;
    margin-bottom: 1.25rem;
    transition: background 0.2s;
  }

  .step-line.active {
    background: #4caf50;
  }

  .step-content {
    text-align: left;
  }

  .step-content h2 {
    margin: 0 0 1rem;
    font-size: 1.5rem;
    font-weight: 600;
    text-align: center;
  }

  .step-content p {
    color: var(--text-secondary, #666);
    line-height: 1.6;
    margin-bottom: 1.5rem;
  }

  .features {
    display: flex;
    gap: 1rem;
    margin-bottom: 2rem;
    justify-content: center;
  }

  .feature {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 0.5rem;
    padding: 1rem;
    background: #f5f7fa;
    border-radius: 8px;
    flex: 1;
  }

  .feature svg {
    width: 32px;
    height: 32px;
    color: #1976d2;
  }

  .feature span {
    font-size: 0.8rem;
    font-weight: 500;
    text-align: center;
  }

  .instructions {
    text-align: left;
    padding-left: 1.25rem;
    margin-bottom: 1.5rem;
  }

  .instructions li {
    margin-bottom: 0.5rem;
    color: var(--text-secondary, #666);
    line-height: 1.5;
  }

  .instructions a {
    color: #1976d2;
    text-decoration: none;
  }

  .instructions a:hover {
    text-decoration: underline;
  }

  .note {
    display: flex;
    align-items: flex-start;
    gap: 0.5rem;
    padding: 0.75rem 1rem;
    background: #fff3e0;
    border-radius: 8px;
    margin-bottom: 1.5rem;
    font-size: 0.875rem;
    color: #ef6c00;
  }

  .note svg {
    width: 18px;
    height: 18px;
    flex-shrink: 0;
    margin-top: 2px;
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
    border-color: #1976d2;
    box-shadow: 0 0 0 3px rgba(25, 118, 210, 0.1);
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

  .button-group {
    display: flex;
    gap: 0.75rem;
    justify-content: center;
    margin-top: 1.5rem;
  }

  .btn {
    padding: 0.75rem 1.5rem;
    border-radius: 8px;
    font-size: 0.9375rem;
    font-weight: 500;
    cursor: pointer;
    transition: all 0.2s;
    border: none;
  }

  .btn:disabled {
    opacity: 0.6;
    cursor: not-allowed;
  }

  .btn.primary {
    background: #1976d2;
    color: white;
  }

  .btn.primary:hover:not(:disabled) {
    background: #1565c0;
  }

  .btn.secondary {
    background: transparent;
    color: var(--text-primary, #333);
    border: 1px solid var(--border-color, #ddd);
  }

  .btn.secondary:hover:not(:disabled) {
    background: #f0f0f0;
  }
</style>
