<script lang="ts">
  import { fade } from "svelte/transition";
  import type { Attachment } from "svelte/attachments";
  import type { MouseEventHandler } from "svelte/elements";
  import InputDialog, { type Dialog } from "./InputDialog.svelte";
  import { svg, CREATE_DIR, DELETE, DOWNLOAD, OPEN_DIR, RENAME, UPLOAD } from "@/lib/svg";
  import {
    toDir,
    directory,
    download,
    upload,
    deleteEntry,
    basename,
    rename,
    makeDir,
  } from "@/lib/fs.svelte";

  let { selected }: { selected: string[] } = $props();

  let menu = $state({ height: 0, width: 0, isOpen: false });
  let positions = $state({ cursor: { x: 0, y: 0 }, window: { height: 0, width: 0 } });
  let currentRow = $state<HTMLTableRowElement | null>(null);
  let dialog = $state<Dialog>({
    value: "",
    title: "",
    oldValue: "",
    node: undefined,
    onSubmit: async () => {},
  });

  const currentRowId = $derived(
    currentRow?.querySelector<HTMLInputElement>("td > input")?.value || "",
  );
  const isFile = $derived(currentRowId.startsWith("1|"));
  const menuStyle = $derived.by(() => {
    const { x, y } = positions.cursor;
    return `--cursor-y:${y}; --cursor-x:${x};`;
  });

  const getContextMenuDimension: Attachment<HTMLMenuElement> = (node) => {
    menu.height = node.offsetHeight;
    menu.width = node.offsetWidth;
  };

  const closeMenu = () => {
    menu.isOpen = false;
    currentRow?.removeAttribute("data-active");
    currentRow = null;
  };

  const oncontextmenu: MouseEventHandler<Window> = async (e) => {
    e.preventDefault();
    if (menu.isOpen) return closeMenu();
    if (!(e.target instanceof HTMLElement)) return;

    const row = e.target.closest<HTMLTableRowElement>("tbody > tr");
    if (row == null || e.target.closest("td > input") != null) return;
    row.setAttribute("data-active", "");
    currentRow = row;

    positions.cursor = { x: e.clientX, y: e.clientY };
    positions.window = { height: window.innerHeight, width: window.innerWidth };

    const { window: win, cursor } = positions;
    if (win.height - cursor.y < menu.height) positions.cursor.y -= menu.height;
    if (win.width - cursor.x < menu.width) positions.cursor.x -= menu.width;
    menu.isOpen = true;
  };

  const renenameDialog = () => {
    dialog.title = `Rename ${isFile ? "File" : "Directory"}`;
    const oldName = basename(currentRowId);
    dialog.oldValue = oldName;
    dialog.value = oldName;
    dialog.node?.showModal();
    dialog.onSubmit = async (e: SubmitEvent) => {
      e.preventDefault();
      const newName = dialog.value.trim();
      if (!newName || newName == dialog.oldValue) return;
      await rename(directory.current, oldName, newName);
      dialog.node?.close();
    };
  };

  const createDirDialog = () => {
    dialog.title = "Create Directory";
    dialog.value = "";
    dialog.node?.showModal();
    dialog.onSubmit = async (e: SubmitEvent) => {
      e.preventDefault();
      const dirname = dialog.value.trim();
      if (!dirname) return;
      await makeDir(directory.current, dirname);
      dialog.node?.close();
    };
  };
</script>

<svelte:window {oncontextmenu} onclick={closeMenu} />

<InputDialog bind:dialog />
<!-- FIXME: disable unnecessary menu options based on selected row -->

{#if menu.isOpen}
  <menu transition:fade={{ duration: 100 }} {@attach getContextMenuDimension} style={menuStyle}>
    {@render item("open", OPEN_DIR, () => toDir(currentRowId))}
    {@render item("download", DOWNLOAD, download("default", [currentRowId]))}
    {@render item("download to", DOWNLOAD, download("select", [currentRowId]))}
    {@render item("delete", DELETE, deleteEntry([currentRowId]))}
    {@render item("rename", RENAME, renenameDialog)}
    <hr />
    {@render item("upload files", UPLOAD, upload("files", directory.current))}
    {@render item("upload directory", UPLOAD, upload("dir", directory.current))}
    {@render item("create directory", CREATE_DIR, createDirDialog)}
    <hr />

    {const isDisabled = selected.length === 0}
    {@render item("download selected", DOWNLOAD, download("default", selected), isDisabled)}
    {@render item("download selected to", DOWNLOAD, download("select", selected), isDisabled)}
    {@render item("delete selected", DELETE, deleteEntry(selected), isDisabled)}
  </menu>
{/if}

{#snippet item(
  name: string,
  icon: string,
  onclick: MouseEventHandler<HTMLButtonElement>,
  disabled = false,
)}
  <li>
    <button {onclick} disabled={disabled!!}>
      {@render svg({ d: icon })}
      <span>{name}</span>
    </button>
  </li>
{/snippet}

<style>
  :root {
    --cursor-y: unset;
    --cursor-x: unset;
  }

  menu {
    position: absolute;
    top: calc(var(--cursor-y) * 1px);
    left: calc(var(--cursor-x) * 1px);
    background: var(--background);
    border: 1px solid var(--background-a30);
    padding: 0.5em;
    display: grid;
    gap: 0.25em;
    z-index: 5;
    box-shadow: 0 0 4px hsla(from var(--foreground) h s l / 0.2);
  }

  li {
    list-style: none;
    margin: 0;
    padding: 0;
  }

  button {
    display: flex;
    align-items: center;
    gap: 0.5em;
    min-width: 100%;
    padding: 0.3em;
  }

  hr {
    min-width: 100%;
    background: var(--background-a30);
    border: none;
    height: 1px;
  }
</style>
