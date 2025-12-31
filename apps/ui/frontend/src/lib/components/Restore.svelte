<script>
  import { onMount } from 'svelte';
  import { EventsOn } from '../wailsjs/runtime/runtime';

  let bucket = '';
  let targetDir = '';
  let fromYear = '';
  let fromMonth = '';
  let toYear = '';
  let toMonth = '';
  let isProcessing = false;
  let progress = { stage: '', current: 0, total: 0, message: '', file: '' };
  let error = '';
  let success = false;

  let SelectDirectory, Restore;

  // Generate year options (current year - 10 to current year + 1)
  const currentYear = new Date().getFullYear();
  const years = Array.from({ length: 12 }, (_, i) => currentYear - 10 + i);
  const months = [
    { value: '01', label: 'January' },
    { value: '02', label: 'February' },
    { value: '03', label: 'March' },
    { value: '04', label: 'April' },
    { value: '05', label: 'May' },
    { value: '06', label: 'June' },
    { value: '07', label: 'July' },
    { value: '08', label: 'August' },
    { value: '09', label: 'September' },
    { value: '10', label: 'October' },
    { value: '11', label: 'November' },
    { value: '12', label: 'December' },
  ];

  // Build filter strings from selectors
  $: fromFilter = fromYear ? (fromMonth ? `${fromMonth}/${fromYear}` : `${fromYear}`) : '';
  $: toFilter = toYear ? (toMonth ? `${toMonth}/${toYear}` : `${toYear}`) : '';

  // Clear month when year is cleared
  $: if (!fromYear) fromMonth = '';
  $: if (!toYear) toMonth = '';

  onMount(async () => {
    try {
      const module = await import('../wailsjs/go/main/App');
      SelectDirectory = module.SelectDirectory;
      Restore = module.Restore;

      EventsOn('progress', (data) => {
        progress = data;
      });
    } catch (err) {
      console.error('Failed to load Wails bindings:', err);
    }
  });

  async function selectTarget() {
    try {
      const dir = await SelectDirectory();
      if (dir) targetDir = dir;
    } catch (err) {
      console.error('Failed to select directory:', err);
    }
  }

  async function startRestore() {
    if (!bucket || !targetDir) {
      error = 'Please enter S3 bucket name and select target directory';
      return;
    }

    isProcessing = true;
    error = '';
    success = false;
    progress = { stage: '', current: 0, total: 0, message: '', file: '' };

    try {
      await Restore({ bucket, targetDir, fromFilter, toFilter });
      success = true;
      progress = { stage: 'completed', current: 0, total: 0, message: 'Restore completed successfully!', file: '' };
    } catch (err) {
      error = err.toString();
    } finally {
      isProcessing = false;
    }
  }

  $: progressPercent = progress.total > 0 ? Math.round((progress.current / progress.total) * 100) : 0;
</script>

<div class="restore">
  <h2>Restore from S3</h2>
  <p class="description">
    Download and extract tar.gz archives from S3 bucket to local directory.
  </p>

  <div class="form">
    <div class="form-group">
      <label for="bucket">S3 Bucket Name</label>
      <input type="text" id="bucket" bind:value={bucket} placeholder="my-backup-bucket" disabled={isProcessing} />
    </div>

    <div class="form-group">
      <label for="target">Target Directory</label>
      <div class="dir-input">
        <input type="text" id="target" bind:value={targetDir} readonly placeholder="Select target directory..." />
        <button on:click={selectTarget} disabled={isProcessing}>Browse</button>
      </div>
    </div>

    <div class="form-group">
      <label>From (optional)</label>
      <div class="date-filter">
        <select bind:value={fromYear} disabled={isProcessing}>
          <option value="">Any year</option>
          {#each years as year}
            <option value={year}>{year}</option>
          {/each}
        </select>
        <select bind:value={fromMonth} disabled={isProcessing || !fromYear}>
          <option value="">All months</option>
          {#each months as month}
            <option value={month.value}>{month.label}</option>
          {/each}
        </select>
      </div>
      <small>Leave empty to restore from the beginning</small>
    </div>

    <div class="form-group">
      <label>To (optional)</label>
      <div class="date-filter">
        <select bind:value={toYear} disabled={isProcessing}>
          <option value="">Any year</option>
          {#each years as year}
            <option value={year}>{year}</option>
          {/each}
        </select>
        <select bind:value={toMonth} disabled={isProcessing || !toYear}>
          <option value="">All months</option>
          {#each months as month}
            <option value={month.value}>{month.label}</option>
          {/each}
        </select>
      </div>
      <small>Leave empty to restore until the end</small>
    </div>

    <button class="btn-primary" on:click={startRestore} disabled={isProcessing || !bucket || !targetDir}>
      {isProcessing ? 'Restoring...' : 'Start Restore'}
    </button>
  </div>

  {#if isProcessing || progress.stage}
    <div class="progress-section">
      <div class="progress-info">
        <strong>{progress.stage}</strong>
        {#if progress.message}
          <p>{progress.message}</p>
        {/if}
        {#if progress.file}
          <p class="file-name">{progress.file}</p>
        {/if}
      </div>
      {#if progress.total > 0}
        <div class="progress-bar">
          <div class="progress-bar-fill" style="width: {progressPercent}%"></div>
          <div class="progress-bar-text">{progressPercent}%</div>
        </div>
      {/if}
    </div>
  {/if}

  {#if error}
    <div class="alert alert-error">
      <strong>Error:</strong> {error}
    </div>
  {/if}

  {#if success}
    <div class="alert alert-success">
      Restore completed successfully!
    </div>
  {/if}
</div>

<style>
  .restore {
    max-width: 800px;
  }

  h2 {
    margin: 0 0 8px 0;
    font-size: 24px;
  }

  .description {
    margin: 0 0 24px 0;
    color: var(--text-secondary);
    font-size: 14px;
  }

  .form {
    background-color: var(--secondary-bg);
    padding: 24px;
    border-radius: 8px;
    margin-bottom: 24px;
  }

  .dir-input {
    display: flex;
    gap: 8px;
  }

  .dir-input input {
    flex: 1;
  }

  .dir-input button {
    flex-shrink: 0;
  }

  .btn-primary {
    width: 100%;
    padding: 12px;
    font-size: 16px;
    margin-top: 8px;
  }

  .progress-section {
    background-color: var(--secondary-bg);
    padding: 24px;
    border-radius: 8px;
    margin-bottom: 16px;
  }

  .progress-info {
    margin-bottom: 16px;
  }

  .progress-info strong {
    text-transform: capitalize;
    display: block;
    margin-bottom: 8px;
    color: var(--accent);
  }

  .progress-info p {
    margin: 4px 0;
    font-size: 14px;
  }

  .file-name {
    font-size: 12px;
    color: var(--text-secondary);
    font-family: monospace;
    word-break: break-all;
  }

  .alert {
    padding: 16px;
    border-radius: 8px;
    margin-bottom: 16px;
  }

  .alert-error {
    background-color: rgba(244, 67, 54, 0.1);
    border: 1px solid var(--error);
    color: var(--error);
  }

  .alert-success {
    background-color: rgba(76, 175, 80, 0.1);
    border: 1px solid var(--success);
    color: var(--success);
  }

  small {
    display: block;
    margin-top: 4px;
    font-size: 12px;
    color: var(--text-secondary);
  }

  .date-filter {
    display: flex;
    gap: 8px;
  }

  .date-filter select {
    flex: 1;
  }
</style>
