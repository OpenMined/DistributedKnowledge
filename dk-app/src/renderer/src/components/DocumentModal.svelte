<script>
  import { createEventDispatcher, onMount } from 'svelte';
  import { X, Copy, CheckCircle } from 'lucide-svelte';
  import { cn } from '@lib/utils';
  import { marked } from 'marked';

  export let title = '';
  export let content = '';
  export let show = false;

  const dispatch = createEventDispatcher();

  // For Markdown rendering
  let renderedContent = '';

  // For copy functionality
  let copied = false;

  // Render markdown content using marked library
  $: if (content) {
    try {
      renderedContent = marked(content);
    } catch(e) {
      console.error('Error parsing markdown:', e);
      renderedContent = `<pre>${content}</pre>`;
    }
  }

  function copyContent() {
    navigator.clipboard.writeText(content).then(() => {
      copied = true;
      setTimeout(() => (copied = false), 2000);
    });
  }

  function close() {
    show = false;
    dispatch('close');
  }

  // Close modal when clicking outside
  function handleBackdropClick(e) {
    if (e.target === e.currentTarget) {
      close();
    }
  }

  // Close on Escape key
  function handleKeydown(e) {
    if (e.key === 'Escape' && show) {
      close();
    }
  }

  // Apply styling to the generated markdown
  onMount(() => {
    // Add any markdown-specific styling setup if needed
  });
</script>

<svelte:window on:keydown={handleKeydown} />

{#if show}
  <div
    class="fixed inset-0 z-50 bg-black/50 backdrop-blur-sm flex items-center justify-center p-4 opacity-100 transition-opacity duration-300"
    on:click={handleBackdropClick}
  >
    <div
      class="bg-background border border-border rounded-lg shadow-lg max-w-2xl w-full max-h-[80vh] flex flex-col animate-in fade-in slide-in-from-bottom-5 duration-300"
    >
      <!-- Header -->
      <div class="flex items-center justify-between p-4 border-b border-border">
        <h3 class="text-lg font-semibold truncate pr-2">{title}</h3>
        <div class="flex items-center gap-2">
          <button
            class="p-1.5 rounded-md text-muted-foreground hover:bg-muted hover:text-foreground transition-colors"
            title="Copy content"
            on:click={copyContent}
          >
            {#if copied}
              <CheckCircle size={18} class="text-success" />
            {:else}
              <Copy size={18} />
            {/if}
          </button>
          <button
            class="p-1.5 rounded-md text-muted-foreground hover:bg-muted hover:text-foreground transition-colors"
            title="Close"
            on:click={close}
          >
            <X size={18} />
          </button>
        </div>
      </div>

      <!-- Content -->
      <div class="flex-1 p-4 overflow-y-auto">
        <!-- Render markdown content -->
        <div class="prose prose-sm dark:prose-invert max-w-none">
          {@html renderedContent}
        </div>
      </div>

      <!-- Footer -->
      <div class="p-4 border-t border-border flex justify-end">
        <button
          class="px-4 py-2 rounded-md bg-muted hover:bg-muted/80 transition-colors font-medium text-sm"
          on:click={close}
        >
          Close
        </button>
      </div>
    </div>
  </div>
{/if}

<style>
  /* Markdown styling */
  :global(.prose) {
    color: var(--foreground);
  }

  :global(.prose h1),
  :global(.prose h2),
  :global(.prose h3),
  :global(.prose h4),
  :global(.prose h5),
  :global(.prose h6) {
    color: var(--foreground);
    margin-top: 1.5em;
    margin-bottom: 0.5em;
    font-weight: 600;
  }

  :global(.prose h1) {
    font-size: 1.5em;
  }

  :global(.prose h2) {
    font-size: 1.25em;
  }

  :global(.prose h3) {
    font-size: 1.125em;
  }

  :global(.prose p) {
    margin-top: 0.75em;
    margin-bottom: 0.75em;
  }

  :global(.prose a) {
    color: var(--primary);
    text-decoration: underline;
    font-weight: 500;
  }

  :global(.prose ul),
  :global(.prose ol) {
    margin-top: 0.5em;
    margin-bottom: 0.5em;
    padding-left: 1.5em;
  }

  :global(.prose code) {
    font-family: monospace;
    font-size: 0.9em;
    background-color: var(--muted);
    padding: 0.2em 0.4em;
    border-radius: 0.25em;
  }

  :global(.prose pre) {
    background-color: var(--muted);
    padding: 1em;
    border-radius: 0.25em;
    overflow-x: auto;
    margin: 1em 0;
  }

  :global(.prose pre code) {
    background-color: transparent;
    padding: 0;
    border-radius: 0;
  }

  :global(.prose blockquote) {
    border-left: 3px solid var(--border);
    padding-left: 1em;
    font-style: italic;
    color: var(--muted-foreground);
    margin: 1em 0;
  }

  :global(.prose table) {
    width: 100%;
    border-collapse: collapse;
    margin: 1em 0;
  }

  :global(.prose th),
  :global(.prose td) {
    border: 1px solid var(--border);
    padding: 0.5em;
  }

  :global(.prose th) {
    background-color: var(--muted);
    font-weight: 600;
  }

  :global(.prose hr) {
    border: none;
    border-top: 1px solid var(--border);
    margin: 1.5em 0;
  }
</style>