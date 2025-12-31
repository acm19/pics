<script>
  import { onMount } from 'svelte';
  import { EventsOn } from '../wailsjs/runtime/runtime';

  let sourceDir = '';
  let targetDir = '';
  let compressJPEGs = true;
  let jpegQuality = 50;
  let maxConcurrency = 100;
  let isProcessing = false;
  let progress = { stage: '', current: 0, total: 0, message: '', file: '' };
  let error = '';
  let success = false;

  let SelectDirectory, Parse;

  onMount(async () => {
    try {
      const module = await import('../wailsjs/go/main/App');
      SelectDirectory = module.SelectDirectory;
      Parse = module.Parse;

      // Listen for progress events
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

  async function selectTarget() {
    try {
      const dir = await SelectDirectory();
      if (dir) targetDir = dir;
    } catch (err) {
      console.error('Failed to select directory:', err);
    }
  }

  async function startParse() {
    if (!sourceDir || !targetDir) {
      error = 'Please select both source and target directories';
      return;
    }

    isProcessing = true;
    error = '';
    success = false;
    progress = { stage: '', current: 0, total: 0, message: '', file: '' };

    try {
      await Parse({
        sourceDir,
        targetDir,
        compressJPEGs,
        jpegQuality,
        maxConcurrency,
      });
      success = true;
      progress = { stage: 'completed', current: 0, total: 0, message: 'Processing completed successfully!', file: '' };
    } catch (err) {
      error = err.toString();
    } finally {
      isProcessing = false;
    }
  }

  $: progressPercent = progress.total > 0 ? Math.round((progress.current / progress.total) * 100) : 0;
</script>

<div class="parse">
  <h2>Parse & Organise Media Files</h2>
  <p class="description">
    Copy and organise photos and videos by date. Optionally compress JPEGs while preserving EXIF metadata.
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
      <label for="target">Target Directory</label>
      <div class="dir-input">
        <input type="text" id="target" bind:value={targetDir} readonly placeholder="Select target directory..." />
        <button on:click={selectTarget} disabled={isProcessing}>Browse</button>
      </div>
    </div>

    <div class="form-group">
      <label>
        <input type="checkbox" bind:checked={compressJPEGs} disabled={isProcessing} />
        Compress JPEG images
      </label>
    </div>

    {#if compressJPEGs}
      <div class="form-group">
        <label for="quality">JPEG Quality (1-100)</label>
        <input type="number" id="quality" bind:value={jpegQuality} min="1" max="100" disabled={isProcessing} />
      </div>
    {/if}

    <div class="form-group">
      <label for="concurrency">Max Concurrency</label>
      <input type="number" id="concurrency" bind:value={maxConcurrency} min="1" max="500" disabled={isProcessing} />
    </div>

    <button class="btn-primary" on:click={startParse} disabled={isProcessing || !sourceDir || !targetDir}>
      {isProcessing ? 'Processing...' : 'Start Processing'}
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
      Processing completed successfully!
    </div>
  {/if}
</div>

<style>
  .parse {
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

  input[type="checkbox"] {
    margin-right: 8px;
  }
</style>
