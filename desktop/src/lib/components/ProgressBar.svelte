<script lang="ts">
  interface Props {
    progress?: number;
    label?: string;
    showPercent?: boolean;
    variant?: 'default' | 'success' | 'error';
    size?: 'sm' | 'md' | 'lg';
  }

  let {
    progress = 0,
    label = '',
    showPercent = true,
    variant = 'default',
    size = 'md'
  }: Props = $props();

  let clampedProgress = $derived(Math.min(100, Math.max(0, progress)));
</script>

<div class="progress-container {size}">
  {#if label}
    <div class="progress-label">
      <span class="label-text">{label}</span>
      {#if showPercent}
        <span class="label-percent">{Math.round(clampedProgress)}%</span>
      {/if}
    </div>
  {/if}
  <div class="progress-bar">
    <div
      class="progress-fill {variant}"
      style="width: {clampedProgress}%"
    ></div>
  </div>
</div>

<style>
  .progress-container {
    width: 100%;
  }

  .progress-label {
    display: flex;
    justify-content: space-between;
    margin-bottom: 0.25rem;
    font-size: 0.85rem;
  }

  .label-text {
    color: var(--text-primary, #333);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .label-percent {
    color: var(--text-secondary, #666);
    flex-shrink: 0;
    margin-left: 0.5rem;
  }

  .progress-bar {
    width: 100%;
    background: var(--bg-secondary, #e0e0e0);
    border-radius: 4px;
    overflow: hidden;
  }

  .sm .progress-bar {
    height: 4px;
  }

  .md .progress-bar {
    height: 8px;
  }

  .lg .progress-bar {
    height: 12px;
  }

  .progress-fill {
    height: 100%;
    transition: width 0.3s ease;
    border-radius: 4px;
  }

  .progress-fill.default {
    background: var(--primary-color, #1976d2);
  }

  .progress-fill.success {
    background: var(--success-color, #4caf50);
  }

  .progress-fill.error {
    background: var(--error-color, #f44336);
  }
</style>
