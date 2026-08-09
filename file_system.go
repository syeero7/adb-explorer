package main

import (
	"bytes"
	"cmp"
	"errors"
	"io/fs"
	"log"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode"
)

type Entry struct {
	Id           string      `json:"id"`
	IsDir        bool        `json:"isDir"`
	Name         string      `json:"name"`
	Size         string      `json:"size"`
	Ext          string      `json:"ext"`
	Mode         os.FileMode `json:"mode"`
	LastModified time.Time   `json:"lastModified"`
	_size        uint32
}

type DirEntries struct {
	Parent  string  `json:"parent"`
	Path    string  `json:"path"`
	Entries []Entry `json:"entries"`
}

const DefaultFileMode = 0644
const DefaultDirMode = 0755

func (a *App) List(path, query, sortBy string, refresh int) DirEntries {
	dirpath, err := cleanPath(path)
	if err != nil {
		a.sendLogMsg(LogErr, err.Error())
		return DirEntries{}
	}

	if entries, ok := a.cache.get(dirpath); ok {
		if refresh == 0 {
			a.currentPath = dirpath
			return a.sortFilterDir(&entries, query, sortBy)
		}

		a.cache.invalidate(dirpath)
	}

	entries, err := a.getEntries(dirpath)
	if err != nil {
		a.sendLogMsg(LogErr, err.Error())
		return DirEntries{}
	}

	a.currentPath = dirpath
	return a.sortFilterDir(&entries, query, sortBy)
}

// TODO: pull/push tar compressed multiple files/directories

func (a *App) Download(downloadDir string, paths []string) {
	dirpath := a.getDownloadDirPath(downloadDir)
	if dirpath == "" {
		a.sendLogMsg(LogWarn, "no download directory found")
		return
	}

	for _, fpath := range paths {
		isDir, entryPath := parseEntryId(fpath)
		remotePath, err := cleanPath(entryPath)
		if err != nil {
			a.sendLogMsg(LogErr, err.Error())
			return
		}

		localPath := filepath.Join(dirpath, path.Base(remotePath))
		if isDir {
			if err := a.pullDir(remotePath, localPath); err != nil {
				a.sendLogMsg(LogErr, err.Error())
				return
			}
			continue
		}

		if err := a.pullFile(remotePath, localPath); err != nil {
			a.sendLogMsg(LogErr, err.Error())
			return
		}
	}
}

func (a *App) Upload(kind, remote string) {
	remoteDir, err := cleanPath(remote)
	if err != nil {
		a.sendLogMsg(LogErr, err.Error())
		return
	}

	switch kind {

	case "dir":
		localPath := a.selectDirToUpload()
		if localPath == "" {
			a.sendLogMsg(LogWarn, "no directory is selected to upload")
			return
		}

		remotePath := path.Join(remoteDir, filepath.Base(localPath))
		if err := a.pushDir(localPath, remotePath); err != nil {
			a.sendLogMsg(LogErr, err.Error())
			return
		}

	case "files":
		files := a.selectFilesToUpload()
		if len(files) == 0 {
			a.sendLogMsg(LogWarn, "no files selected to upload")
			return
		}

		for _, localPath := range files {
			remotePath := path.Join(remoteDir, filepath.Base(localPath))
			if err := a.pushFile(localPath, remotePath); err != nil {
				a.sendLogMsg(LogErr, err.Error())
				return
			}
		}
	default:
		panic("unkown kind: " + kind)
	}

}

func (a *App) Delete(paths []string) {
	var toDelete []string
	for _, fpath := range paths {
		_, entryPath := parseEntryId(fpath)
		remotePath, err := cleanPath(entryPath)
		if err != nil {
			a.sendLogMsg(LogErr, err.Error())
			return
		}

		toDelete = append(toDelete, strconv.Quote(remotePath))
	}

	if _, err := a.device.RunShellCommand("rm -rf", strings.Join(toDelete, " ")); err != nil {
		a.sendLogMsg(LogErr, err.Error())
		return
	}
}

func (a *App) Rename(dir, oldName, newName string) {
	if a.isEntryExist(dir, newName) || !isValidName(newName) {
		a.sendLogMsg(LogWarn, "name already exists or invalid name")
		return
	}

	oldPath, err := cleanPath(path.Join(dir, oldName))
	if err != nil {
		a.sendLogMsg(LogErr, err.Error())
		return
	}

	newPath, err := cleanPath(path.Join(dir, newName))
	if err != nil {
		a.sendLogMsg(LogErr, err.Error())
		return
	}

	if dir != path.Dir(oldPath) || dir != path.Dir(newPath) {
		a.sendLogMsg(LogErr, "cannot move")
		return
	}

	defer a.cache.invalidateRec(dir)
	if _, err := a.device.RunShellCommand("mv", strconv.Quote(oldPath), strconv.Quote(newPath)); err != nil {
		a.sendLogMsg(LogErr, err.Error())
		return
	}
}

func (a *App) MakeDir(dir, dirname string) {
	if a.isEntryExist(dir, dirname) || !isValidName(dirname) {
		a.sendLogMsg(LogWarn, "directory name is already exists or its invalid")
		return
	}

	dirPath, err := cleanPath(path.Join(dir, dirname))
	if err != nil {
		a.sendLogMsg(LogErr, err.Error())
		return
	}

	if !strings.HasPrefix(dirPath, dir) {
		a.sendLogMsg(LogErr, "not in current dir")
		return
	}

	defer a.cache.invalidate(dir)
	if _, err := a.device.RunShellCommand("mkdir", strconv.Quote(dirPath)); err != nil {
		a.sendLogMsg(LogErr, err.Error())
		return
	}
}

func (a *App) setIgnoreDirs() {
	if a.ignoreDirs == nil {
		a.ignoreDirs = make(map[string]struct{})
	}

	dirNames := []string{".", "..", "emulated", "self"}
	for _, key := range dirNames {
		a.ignoreDirs[key] = struct{}{}
	}
}

func (a *App) getEntries(dirpath string) (DirEntries, error) {
	items, err := a.device.List(dirpath)
	if err != nil {
		return DirEntries{}, err
	}

	entries := make([]Entry, 0, len(items))
	for _, item := range items {
		if _, ok := a.ignoreDirs[item.Name]; ok {
			continue
		}

		entry := Entry{
			IsDir:        item.IsDir(),
			Name:         item.Name,
			Mode:         item.Mode,
			LastModified: item.LastModified,
			Ext:          getFileExt(item.Name, item.IsDir()),
			Id:           createEntryId(path.Join(dirpath, item.Name), item.IsDir()),
			Size:         toReadableSize(int64(item.Size)),
			_size:        item.Size,
		}

		entries = append(entries, entry)
	}

	return DirEntries{Path: dirpath, Parent: path.Dir(dirpath), Entries: entries}, nil
}

func (a *App) sortFilterDir(dir *DirEntries, query, sortBy string) DirEntries {
	sorted := sortEntries(dir, sortBy)
	entries, filtered := filterEntries(sorted, strings.TrimSpace(query))
	a.cache.set(a.currentPath, *entries)
	entries.Entries = filtered
	return *entries
}

func (a *App) pullDir(remote, local string) error {
	if err := os.MkdirAll(local, DefaultDirMode); err != nil {
		return err
	}

	entries, err := a.device.List(remote)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if entry.Name == "." || entry.Name == ".." {
			continue
		}

		remotePath := path.Join(remote, entry.Name)
		localPath := filepath.Join(local, entry.Name)

		if entry.IsDir() {
			if err := a.pullDir(remotePath, localPath); err != nil {
				return err
			}
			continue
		}

		if err := a.pullFile(remotePath, localPath); err != nil {
			return err
		}
	}
	return nil
}

func (a *App) pushDir(local, remote string) error {
	type F2Push struct {
		local  string
		remote string
	}

	var dirsToMake []string
	var filesToPush []F2Push

	err := filepath.WalkDir(local, func(fpath string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(local, fpath)
		if err != nil {
			return err
		}

		remotePath := path.Join(remote, filepath.ToSlash(relPath))
		if entry.IsDir() {
			dirsToMake = append(dirsToMake, strconv.Quote(remotePath))
		} else {
			filesToPush = append(filesToPush, F2Push{local: fpath, remote: remotePath})
		}

		return nil
	})

	if err != nil {
		return err
	}

	if _, err := a.device.RunShellCommand("mkdir -p", strings.Join(dirsToMake, " ")); err != nil {
		return err
	}

	for _, f2p := range filesToPush {
		if err := a.pushFile(f2p.local, f2p.remote); err != nil {
			return err
		}
	}

	return nil
}

func (a *App) pullFile(remote, local string) error {
	dest, err := os.OpenFile(local, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, DefaultFileMode)
	if err != nil {
		log.Println("pull file | main", remote, local)
		return err
	}

	defer closeIO(dest)
	return a.device.Pull(remote, dest)
}

func (a *App) pushFile(local, remote string) error {
	file, err := os.Open(local)
	if err != nil {
		return err
	}

	defer closeIO(file)
	stat, err := file.Stat()
	if err != nil {
		return err
	}

	return a.device.Push(file, remote, stat.ModTime(), DefaultFileMode)
}

func (a *App) getDownloadDirPath(downloadDir string) string {
	if downloadDir == "select" {
		return a.SelectDownloadDir()
	}

	return a.settings.DownloadDir
}

func (a *App) isEntryExist(dir, name string) bool {
	if entris, ok := a.cache.get(dir); ok {
		for _, entry := range entris.Entries {
			if entry.Name == name {
				return true
			}
		}
	}

	return false
}

func isValidName(name string) bool {
	if l := len(name); l == 0 || l > 255 {
		return false
	}

	if name == ".." || name == "." || filepath.Base(name) != name {
		return false
	}

	for _, c := range name {
		if !unicode.IsLetter(c) && !unicode.IsDigit(c) && !bytes.ContainsRune([]byte{'-', ' ', '_', '.'}, c) {
			return false
		}
	}

	return true
}

func filterEntries(dir *DirEntries, query string) (*DirEntries, []Entry) {
	if query == "" {
		return dir, dir.Entries
	}

	entries := make([]Entry, 0, len(dir.Entries))
	for _, entry := range dir.Entries {
		if strings.Contains(entry.Name, query) {
			entries = append(entries, entry)
		}
	}

	return dir, entries
}

func sortEntries(dir *DirEntries, sortBy string) *DirEntries {
	parts := strings.Split(sortBy, ":")
	slices.SortFunc(dir.Entries, func(a, b Entry) int {
		if a.IsDir != b.IsDir {
			if a.IsDir {
				return -1
			}
			return 1
		}

		switch parts[0] {
		case "name":
			aName, bName := strings.ToLower(a.Name), strings.ToLower(b.Name)
			if parts[1] == "asc" {
				return cmp.Compare(aName, bName)
			}

			return cmp.Compare(bName, aName)
		case "size":
			if parts[1] == "asc" {
				return cmp.Compare(a._size, b._size)
			}

			return cmp.Compare(b._size, a._size)
		default:
			if parts[1] == "asc" {
				return a.LastModified.Compare(b.LastModified)
			}

			return b.LastModified.Compare(a.LastModified)
		}
	})

	return dir
}

func cleanPath(fpath string) (string, error) {
	if strings.ContainsAny(";&|", fpath) {
		return "", errors.New("invalid characters")
	}

	return path.Clean(fpath), nil
}

func toReadableSize(size int64) string {
	const unit int64 = 1000

	if size < unit {
		return strconv.FormatInt(size, 10) + " B"
	}

	division := unit
	var exponent int64 = 0
	for i := size / unit; i >= unit; i /= unit {
		division *= unit
		exponent++
	}

	const sizes = "kMGTPE"
	str := strconv.FormatFloat(float64(size)/float64(division), 'f', 2, 64)
	return strings.Join([]string{str, " ", string(sizes[exponent]), "B"}, "")
}

func getFileExt(name string, isDir bool) string {
	if !isDir {
		ext := path.Ext(name)
		if ext != "" {
			return strings.ToUpper(ext[1:])
		}
	}

	return ""
}

// prefix path with "0|" if its a directory and prefix with "1|" if its a symlink or regular file
func createEntryId(fpath string, isDir bool) string {
	prefix := '0'
	if !isDir {
		prefix = '1'
	}

	return strings.Join([]string{string(prefix), "|", fpath}, "")
}

// returns true if prefix is "0|" (directory). remove path prefixes "0|" and "1|"
func parseEntryId(fpath string) (bool, string) {
	trimmed := strings.TrimSpace(fpath)
	if entryPath, ok := strings.CutPrefix(trimmed, "0|"); ok {
		return true, entryPath
	}

	if entryPath, ok := strings.CutPrefix(trimmed, "1|"); ok {
		return false, entryPath
	}

	panic("unkown entry id prefix")
}
