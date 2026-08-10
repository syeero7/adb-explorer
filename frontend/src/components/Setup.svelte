<script lang="ts">
  import type { MouseEventHandler } from "svelte/elements";
  import { onMount } from "svelte";
  import Logs from "./Logs.svelte";
  import { router } from "@/lib/router.svelte";
  import { svg, RELOAD, TARGET } from "@/lib/svg";
  import {
    NewADBClient,
    SelectDevice,
    KillServer,
    DownloadADB,
    GetDeviceList,
    ConnectToServer,
    SelectDownloadDir,
    SelectADBExecutable,
    GetDefaultSettings,
  } from "@wails/go/main/App";

  let port = $state(5037);
  let paths = $state({ downloadDir: "loading...", adb: "loading..." });
  let selectedDevice = $state<number | null>();
  let devices = $state<string[]>([]);

  onMount(async () => {
    paths = await GetDefaultSettings();
  });

  const onADBSubmit = async (e: SubmitEvent) => {
    e.preventDefault();
    if (paths.adb.startsWith("loading") || !(e.submitter instanceof HTMLButtonElement)) return;

    switch (e.submitter.value) {
      case "kill": {
        await KillServer(paths.adb, port);
        devices = [];
        selectedDevice = null;
        return;
      }
      case "start": {
        await NewADBClient(paths.adb, port);
        break;
      }
      case "connect": {
        await ConnectToServer(port);
        break;
      }
      default:
        throw new Error("unkown action");
    }

    await refreshDevices();
  };

  const onSelectDevice = async (e: SubmitEvent) => {
    e.preventDefault();
    if (selectedDevice == null) return;
    await SelectDevice(selectedDevice, paths.downloadDir);
    router.current = "explore";
  };

  const downloadADB = async () => {
    paths.adb = await DownloadADB();
  };

  const refreshDevices = async () => {
    devices = await GetDeviceList();
    if (devices == null || devices.length === 0) return;
    selectedDevice = 0;
  };

  const selectADBExecutable = async () => {
    const path = await SelectADBExecutable();
    if (path) paths.adb = path;
  };

  const selectDownloadDir = async () => {
    const path = await SelectDownloadDir();
    if (path) paths.downloadDir = path;
  };
</script>

<form onsubmit={onADBSubmit}>
  <label>
    <span>Port</span>
    <input required bind:value={port} type="number" min={1024} max={65535} />
  </label>

  {@render pathInput(paths, "adb", "ADB executable path", selectADBExecutable)}

  <div class="adb-actions">
    <button name="action" value="connect" type="submit">Connect</button>
    <button name="action" value="start" type="submit">Start Server</button>
    <button name="action" value="kill" type="submit">Kill Server</button>
  </div>
</form>

<form onsubmit={onSelectDevice}>
  {@render pathInput(paths, "downloadDir", "Download directory path", selectDownloadDir)}

  <div class="field-w-btn">
    <label>
      <span>Device</span>
      <select required bind:value={selectedDevice}>
        {#if devices == null || devices.length === 0}
          <option>No device</option>
        {/if}

        {#each devices as device, i}
          <option value={i}>{device}</option>
        {/each}
      </select>
    </label>

    <button type="button" title="refresh" onclick={refreshDevices}>
      {@render svg({ d: RELOAD })}
    </button>
  </div>

  <button class="exp" type="submit" disabled={typeof selectedDevice !== "number"}>Explore</button>
</form>

<section>
  <!-- TODO: download progress bar -->
  <!-- NOTE: use resp.ContentLength with io.TeeReader -->
  <!-- TODO: check server is running on given port or any onsubmit errors -->
  <button onclick={downloadADB}>Download ADB</button>
  <Logs />
</section>

{#snippet pathInput(
  values: typeof paths,
  key: keyof typeof paths,
  label: string,
  onclick: MouseEventHandler<HTMLButtonElement>,
)}
  <div class="field-w-btn">
    <label>
      <span>{label}</span>
      <input required type="text" bind:value={values[key]} />
    </label>

    <button
      {onclick}
      type="button"
      title={`select ${key === "adb" ? "adb executable" : "download directory"}`}>
      {@render svg({ d: TARGET })}
    </button>
  </div>
{/snippet}

<style>
  :root {
    --flex-gap: 0.6em;
    --field-width: calc(100% - var(--btn-size) - var(--flex-gap));
  }

  form {
    display: grid;
    gap: 0.6em;
    padding-block: 1em;
  }

  label {
    display: grid;
    min-width: var(--field-width);
    gap: 0.25em;

    span {
      font-weight: 500;
      font-size: 0.9em;
      text-transform: capitalize;
    }

    input {
      min-width: 26em;
      max-width: var(--field-width);
    }

    select {
      min-height: 2.25em;
    }
  }

  button[type="submit"] {
    margin-block: 0.6em 0.25em;
    padding: 0.4em 0.8em;
    border-radius: var(--radius);
    box-shadow: var(--shadow);

    &.exp {
      max-width: var(--field-width);
      margin-top: 1.5em;
    }
  }

  .adb-actions {
    display: flex;
    justify-content: space-around;
    max-width: var(--field-width);
  }

  .field-w-btn {
    display: flex;
    gap: var(--flex-gap);

    button {
      width: var(--btn-size);
      height: var(--btn-size);
      margin-top: auto;
      border-radius: 50%;
    }
  }

  section {
    display: flex;
    justify-content: space-between;
    padding: 0.5em 0.25em;
    padding-right: calc(100% - var(--field-width) + 0.25em);

    button {
      border-radius: var(--radius);
      box-shadow: var(--shadow);
      font-size: 0.75em;
    }
  }
</style>
