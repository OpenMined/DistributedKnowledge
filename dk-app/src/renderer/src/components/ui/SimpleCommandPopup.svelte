<script lang="ts">
  import {
    commandPopupVisible,
    selectedCommandIndex,
    commands,
    filterCommands,
    type Command
  } from '@lib/commands'
  import { cn } from '@lib/utils'
  import { fly } from 'svelte/transition'
  import { onMount } from 'svelte'
  import logger from '@lib/utils/logger'

  // Props
  export let inputText = ''
  export let onSelectCommand: (cmd: Command) => void = () => {}

  // Computed properties
  $: filteredCommands = inputText.startsWith('/') ? filterCommands(inputText) : []

  $: {
    logger.debug('Command state:', {
      visible: $commandPopupVisible,
      inputText,
      filtered: filteredCommands
    })
  }

  // Methods
  function handleClickCommand(cmd: Command) {
    onSelectCommand(cmd)
  }

  onMount(() => {
    logger.debug('SimpleCommandPopup mounted')
  })
</script>

{#if $commandPopupVisible && filteredCommands.length > 0}
  <div
    class="absolute left-0 bottom-full w-full max-h-[200px] overflow-y-auto bg-background border border-border rounded-md shadow-md z-10 mb-1"
    transition:fly={{ y: 10, duration: 150 }}
  >
    <div class="py-1">
      {#each filteredCommands as cmd, i}
        <button
          class={cn(
            'flex justify-between items-center w-full px-4 py-2 text-left text-sm cursor-pointer',
            i === $selectedCommandIndex
              ? 'bg-muted text-foreground'
              : 'text-foreground hover:bg-muted/50'
          )}
          on:click={() => handleClickCommand(cmd)}
        >
          <span class="font-medium">/{cmd.name}</span>
          <span class="text-muted-foreground text-xs ml-2">{cmd.description}</span>
        </button>
      {/each}
    </div>
  </div>
{/if}
