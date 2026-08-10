import { Delete, Download, List, MakeDir, Rename, Upload } from "@wails/go/main/App";

export type InfoTitle = "name" | "size" | "date modified";
type SortBy = `${InfoTitle}:${"asc" | "desc"}`;

type Dir = {
  current: string;
  query: string;
  sortBy: SortBy;
  refresh: number;
};

export const directory = $state<Dir>({
  current: "/sdcard",
  query: "",
  sortBy: "name:asc",
  refresh: 0,
});

let parentDir = $state("/");

export async function getEntries(dir: Dir) {
  const { entries, parent } = await List(dir.current, dir.query, dir.sortBy, dir.refresh);
  parentDir = parent;
  return entries;
}

export function toParentDir() {
  directory.current = parentDir;
  directory.refresh = 0;
}

export function toDir(path: string) {
  // NOTE: paths prefix with "0|" points to a directory
  directory.current = removePrefix(path, "0|");
  directory.refresh = 0;
}

export function refresh() {
  directory.refresh = directory.refresh < 1 ? 1 : directory.refresh + 1;
}

export function removePrefix(str: string, prefix: string) {
  return str.startsWith(prefix) ? str.slice(prefix.length) : str;
}

export function download(dir: "default" | "select", paths: string[]) {
  return async () => await Download(dir, paths);
}

export function upload(kind: "dir" | "files", dir: string) {
  return async () => {
    await Upload(kind, dir);
    refresh();
  };
}

export function deleteEntry(paths: string[]) {
  return async () => {
    await Delete(paths);
    refresh();
  };
}

export async function rename(dir: string, oldName: string, newName: string) {
  await Rename(dir, oldName, newName);
  refresh();
}

export async function makeDir(dir: string, name: string) {
  await MakeDir(dir, name);
  refresh();
}

export function sortBy(title: InfoTitle, sortBy: SortBy) {
  const isActive = sortBy.startsWith(title);
  const isAsc = sortBy.endsWith("asc");
  const handler = () => {
    if (!isActive) {
      directory.sortBy = `${title}:asc`;
      return;
    }

    directory.sortBy = `${title}:${isAsc ? "desc" : "asc"}`;
  };

  return { isActive, isAsc, handler };
}

export function basename(path: string) {
  path = path.endsWith("/") ? path.slice(0, -1) : path;
  const idx = path.lastIndexOf("/");
  if (idx > 0) path = path.slice(idx + 1);
  return path;
}
