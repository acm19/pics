<script>
  import { onMount } from 'svelte';
  import { EventsOn } from '../wailsjs/runtime/runtime';

  let sourceDir = '';
  let bucket = '';
  let isProcessing = false;
  let progress = { stage: '', current: 0, total: 0, message: '', file: '' };
  let error = '';
  let success = false;

  let SelectDirectory, Backup;

  onMount(async () => {
    try {
      const module = await import('../wailsjs/go/main/App');
      SelectDirectory = module.SelectDirectory;
      Backup = module.Backup;

      EventsOn('progress', (data) => {
        progress = data;
      });
    } catch (err) {
      console.error('Failed to load Wails bindings:', err);
    }
  });

  async function selectSource() {
    try {
      const dir = await SelectDirectory();
      if (dir) sourceDir = dir;
    } catch (err) {
      console.error('Failed to select directory:', err);
    }
  }

  async function startBackup() {
    if (!sourceDir || !bucket) {
      error = 'Please select source directory and enter S3 bucket name';
      return;
    }

    isProcessing = true;
    error = '';
    success = false;
    progress = { stage: '', current: 0, total: 0, message: '', file: '' };

    try {
      await Backup({ sourceDir, bucket });
      success = true;
      progress = { stage: 'completed', current: 0, total: 0, message: 'Backup completed successfully!', file: '' };
    } catch (err) {
      error = err.toString();
    } finally {
      isProcessing = false;
    }
  }

  $: progressPercent = progress.total > 0 ? Math.round((progress.current / progress.total) * 100) : 0;
</script>

<div class="backup">
  <h2>Backup to S3</h2>
  <p class="description">
    Create tar.gz archives of subdirectories and upload to S3 with MD5 deduplication.
  </p>

  <div class="form">
    <div class="form-group">
      <label for="source">Source Directory</label>
      <div class="dir-input">
        <input type="text" id="source" bind:value={sourceDir} readonly placeholder="Select source directory..." />
        <button on:click={selectSource} disabled={isProcessing}>Browse</button>
      </div>
    </div>

    <div class="form-group">
      <label for="bucket">S3 Bucket Name</label>
      <input type="text" id="bucket" bind:value={bucket} placeholder="my-backup-bucket" disabled={isProcessing} />
    </div>

    <button class="btn-primary" on:click={startBackup} disabled={isProcessing || !sourceDir || !bucket}>
      {isProcessing ? 'Backing up...' : 'Start Backup'}
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
      Backup completed successfully!
    </div>
  {/if}
</div>

<style>
  .backup {
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
</style>
