<script lang="ts">
  import { onMount } from "svelte";
  import { fade } from "svelte/transition";
  import Logs from "./Logs.svelte";
  import { svg, RELOAD, UP_ARROW, SEARCH, CLOSE } from "@/lib/svg";
  import { directory, refresh, toParentDir } from "@/lib/fs.svelte";
  import { GetShortcutPaths } from "@wails/go/main/App";

  let shortcutPaths = $state<string[]>([]);
  onMount(async () => {
    shortcutPaths = await GetShortcutPaths();
  });

  let isSearching = $state(false);
  let timeout: number | undefined;

  const openSearch = async () => {
    isSearching = true;
  };

  const refreshDir = () => {
    clearTimeout(timeout);
    refresh();
  };

  const closeSearch = () => {
    clearTimeout(timeout);
    isSearching = false;
    directory.query = "";
  };

  const search = (e: Event) => {
    clearTimeout(timeout);
    const query = (e.target as HTMLInputElement).value.trim();
    timeout = setTimeout(() => {
      directory.query = query;
    }, 300);
  };
</script>

<header>
  <button title="refresh current directory" onclick={refreshDir}>
    {@render svg({ d: RELOAD })}
  </button>

  <button title="go to parent directory" onclick={toParentDir} disabled={directory.current == "/"}>
    {@render svg({ d: UP_ARROW })}
  </button>

  {#if isSearching}
    <div in:fade class="wrapper">
      <!-- svelte-ignore a11y_autofocus -->
      <input type="text" autofocus oninput={search} aria-label="search query" />
      <button title="close" onclick={closeSearch}>
        {@render svg({ d: CLOSE })}
      </button>
    </div>
  {:else}
    <div in:fade class="wrapper">
      <select bind:value={directory.current} aria-label="current directory path">
        <option value={directory.current}>{directory.current}</option>
        {#each shortcutPaths as path}
          {#if path !== directory.current}
            <option value={path} selected={path == directory.current}>{path}</option>
          {/if}
        {/each}
      </select>

      <button title="search entry" onclick={openSearch}>
        {@render svg({ d: SEARCH })}
      </button>
    </div>
  {/if}

  <Logs />
</header>

<style>
  header {
    position: fixed;
    top: 0;
    display: flex;
    align-items: center;
    gap: 0.5em;
    background: var(--background);
    min-width: var(--explorer-min-width);
    min-height: var(--explorer-header-height);
    margin-top: var(--explorer-header-margin);

    .wrapper {
      display: flex;
      flex: 1;
      gap: 0.5em;
    }

    .wrapper:has(> select)::after {
      top: calc(50% - 0.5em);
      right: calc(var(--btn-size) * 2);
    }

    select,
    input {
      flex: 1;
      min-width: 25ch;
      outline: none;
      box-shadow: unset;
      max-height: var(--btn-size);
    }

    input {
      cursor: text;
    }

    button {
      flex-shrink: 0;
      height: var(--btn-size);
      width: var(--btn-size);
      box-shadow: var(--shadow);
      border-radius: var(--radius);
    }
  }
</style>
