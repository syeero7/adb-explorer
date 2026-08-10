<script lang="ts">
  import { fade } from "svelte/transition";
  import type { main } from "@wails/go/models";
  import Header from "./Header.svelte";
  import ContextMenu from "./ContextMenu.svelte";
  import {
    type InfoTitle,
    getEntries,
    directory,
    sortBy,
    toDir,
    removePrefix,
  } from "@/lib/fs.svelte";
  import { svg, DOWN_ARROW, UP_ARROW, FILE, FOLDER } from "@/lib/svg";

  let selected: string[] = $state([]);

  const isSymlink = (mode: number) => (mode & 0xf000) === 0xa000;
</script>

<Header />
<ContextMenu {selected} />

<!-- TODO: sort entries dir/symlinkdir first then regular files -->

<section class="explorer" in:fade>
  {#await getEntries(directory)}
    <div data-status="loading" in:fade>
      <p>Loading...</p>
    </div>
  {:then data}
    <div in:fade>
      {#if Array.isArray(data) && data.length > 0}
        <table>
          <thead>
            <tr>
              <th></th>
              {@render th("name")}
              {@render th("size")}
              {@render th("date modified")}
            </tr>
          </thead>

          <tbody>
            {#each data as entry}
              {@render row(entry)}
            {/each}
          </tbody>
        </table>
      {:else}
        <div data-status="empty" in:fade>
          {#if directory.query != ""}
            <p>No results for "{directory.query}"</p>
          {:else}
            <p>The directory is either empty, inaccessible due to permissions or does not exist.</p>
          {/if}
        </div>
      {/if}
    </div>
  {:catch}
    <div data-status="failed" in:fade>
      <p>Oops, something went wrong. Check the logs.</p>
    </div>
  {/await}
</section>

{#snippet row(entry: main.Entry)}
  {const path = isSymlink(entry.mode) ? removePrefix(entry.id, "1|") : entry.id}
  <!-- NOTE: paths prefix with "1|" points to a symlink or regular file -->

  <tr data-active={undefined}>
    <td>
      <input type="checkbox" value={path} bind:group={selected} />
    </td>

    {const isFile = !entry.isDir && !isSymlink(entry.mode)}
    <td>
      {#if isFile}
        <div class="file">
          {const extLen = entry.ext.length}
          {const hasExt = extLen > 0 && extLen < 5}
          <span data-file-ext={hasExt ? entry.ext : undefined} style={`--ext-len:${extLen}`}>
            {@render svg({ d: FILE })}
          </span>
          <span>
            {entry.name}
          </span>
        </div>
      {:else}
        <button ondblclick={() => toDir(path)}>
          {@render svg({ d: FOLDER })}
          <span>
            {entry.name}
          </span>
        </button>
      {/if}
    </td>

    <td>{isFile ? entry.size : ""}</td>

    {const modified = new Date(entry.lastModified)}
    <td>{modified.toLocaleDateString()}</td>
  </tr>
{/snippet}

{#snippet th(title: InfoTitle)}
  {const { isActive, isAsc, handler } = sortBy(title, directory.sortBy)}
  <th>
    <button onclick={handler}>
      <span>
        {title}
      </span>
      {#if isActive}
        {#if isAsc}
          {@render svg({ d: UP_ARROW })}
        {:else}
          {@render svg({ d: DOWN_ARROW })}
        {/if}
      {/if}
    </button>
  </th>
{/snippet}

<style>
  :root {
    --btn-flex-gap: 0.5em;
    --explorer-header-height: 2.5em;
    --explorer-header-margin: 1.25em;
    --explorer-min-width: min(48em, 100vw);
    --explorer-max-height: calc(
      100vh - var(--explorer-header-height) - (var(--explorer-header-margin) * 2.5)
    );
  }

  :global body:has(div section.explorer) {
    align-content: unset;
  }

  section {
    margin-top: calc(var(--explorer-header-height) + (var(--explorer-header-margin) * 1.5));
    max-height: var(--explorer-max-height);
    min-width: var(--explorer-min-width);
    overflow-y: scroll;
    background: var(--background);
    box-shadow: 0 1px 1px var(--background-a30);
  }

  table {
    --input-col-width: 2.5em;

    table-layout: fixed;
    border-spacing: 0;
    width: var(--explorer-min-width);
    min-height: 100%;
  }

  th,
  td {
    border: 1px solid var(--background-a30);
  }

  thead {
    --max-chars: 0;

    th:first-child {
      border-color: transparent;
      border-bottom-color: var(--background-a30);
      width: var(--input-col-width);
    }

    th:not(th:nth-child(1)) {
      --btn-padding: 1em;
      --th-col-width: calc(
        (var(--max-chars) * 1ch) + var(--btn-flex-gap) + var(--svg-size) + var(--btn-padding)
      );
      min-width: var(--th-col-width);
    }

    th:nth-child(2) {
      --max-chars: 4;
    }

    th:nth-child(3) {
      --max-chars: 9;
      width: var(--th-col-width);
    }

    th:nth-child(4) {
      --max-chars: 13;
      width: var(--th-col-width);
    }

    tr {
      background: var(--background);
      position: sticky;
      top: 0;
      z-index: 10;
    }

    button {
      text-transform: capitalize;
      justify-content: center;
      min-width: 100%;
    }
  }

  .file,
  button {
    display: flex;
    align-items: center;
    gap: 0.5em;
    padding: 0.5em;
    min-height: 1.5em;
    max-width: 100%;
    background: transparent;
  }

  tbody {
    td:has(input) {
      width: var(--input-col-width);
    }

    td:nth-child(3),
    td:nth-child(4) {
      text-align: center;
      padding-inline: 0.5em;
    }

    input {
      width: 1em;
      height: 1em;
      display: block;
      margin-inline: auto;
      accent-color: var(--success);
    }

    tr {
      --row-cursor: pointer;

      position: relative;
      transform: rotate(0deg);
      transition-property: color, background-color;
      transition-duration: 300ms;

      &:hover,
      &:focus-visible,
      &[data-active] {
        --foreground: var(--accent);
        color: var(--accent);
        background: var(--background-a10);
      }

      button :global(svg),
      .file span:first-child {
        flex-shrink: 0;
      }

      button span,
      .file span:nth-child(2) {
        text-overflow: ellipsis;
        overflow-x: hidden;
        white-space: nowrap;
      }

      button,
      .file {
        &:hover,
        &:focus-visible {
          border-color: transparent;
        }

        &::after {
          content: "";
          position: fixed;
          inset: 0;
          z-index: 5;
          cursor: var(--row-cursor);
          left: var(--input-col-width);
        }
      }
    }
  }

  .file {
    --row-cursor: default;
    --ext-len: 0;

    :global svg {
      fill: var(--foreground);
      width: var(--svg-size);
      height: var(--svg-size);
    }

    span[data-file-ext] {
      position: relative;

      &::after {
        position: absolute;
        bottom: 0;
        right: 0;
        z-index: 2;
        content: attr(data-file-ext);
        color: var(--info);
        background: var(--background);
        min-width: clac(var(--ext-len) * 1ch);
        transform: translate(0.25em, 20%);
        border-radius: 2px;
        font-size: 0.5em;
        font-weight: 900;
        padding: 0.1em;
      }
    }
  }

  .explorer:has(div[data-status]) {
    overflow: unset;
    box-shadow: unset;
  }

  div[data-status] {
    min-width: 100%;
    min-height: var(--explorer-max-height);
    background: var(--background);
    align-content: center;
    text-align: center;
    font-style: oblique;
  }

  div[data-status="failed"] {
    color: var(--danger);
  }

  div[data-status="loading"] {
    color: var(--accent);
  }
</style>
