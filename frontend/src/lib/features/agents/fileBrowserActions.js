// Every file-op the FileBrowser context menu can fire. Kept in a factory
// so the component stays under the line cap and every callback carries
// consistent error surfacing.

import {
  Chmod,
  CopyPath,
  DownloadDirectory,
  DownloadFile,
  DownloadMultipleTar,
  MakeDir,
  RemovePath,
  RenamePath,
  UploadFile,
  ViewRemoteFile,
} from '../../api/agents.js'
import { errorMessage } from '../../utils/errors.js'

function joinPath(currentPath, name) {
  const sep = currentPath.includes('\\') ? '\\' : '/'
  return currentPath.endsWith(sep) ? currentPath + name : currentPath + sep + name
}

function isBinaryPreview(raw) {
  return Array.from(raw.slice(0, 4096)).some((char) => {
    const code = char.charCodeAt(0)
    return code <= 8 || (code >= 14 && code <= 31)
  })
}

export function createFileBrowserActions({ sessionID, dialog, store, getCurrentPath, setViewerData }) {
  const join = (name) => joinPath(getCurrentPath(), name)

  async function alertOnError(err, prefix, title) {
    await dialog.alert(errorMessage(err, prefix), title)
  }

  async function downloadDir(f) {
    try {
      await DownloadDirectory(sessionID, join(f.Name || f.name))
    } catch (err) {
      await alertOnError(err, 'Download failed: ', 'Download Error')
    }
  }

  async function downloadFile(f) {
    try {
      await DownloadFile(sessionID, join(f.Name || f.name))
    } catch (err) {
      await alertOnError(err, 'Download failed: ', 'Download Error')
    }
  }

  async function uploadFile() {
    try {
      await UploadFile(sessionID, store.get().path)
      store.refresh(store.get().path)
    } catch (err) {
      await alertOnError(err, 'Upload failed: ', 'Upload Error')
    }
  }

  async function newFolder() {
    const name = await dialog.prompt('New folder name:', 'Create Folder')
    if (!name) return
    try {
      await MakeDir(sessionID, join(name))
      store.refresh(store.get().path)
    } catch (err) { await alertOnError(err, 'Create failed: ', 'Error') }
  }

  async function deleteFile(f) {
    const name = f.Name || f.name
    const isDir = f.IsDir || f.isDir
    if (!(await dialog.confirm(`Delete ${isDir ? 'folder' : 'file'} "${name}"?`, 'Confirm Delete'))) return
    try {
      await RemovePath(sessionID, join(name), !!isDir)
      store.refresh(store.get().path)
    } catch (err) { await alertOnError(err, 'Delete failed: ', 'Error') }
  }

  async function renameFile(f) {
    const name = f.Name || f.name
    const newName = await dialog.prompt(`Rename "${name}" to:`, 'Rename Item', name)
    if (!newName || newName === name) return
    try {
      await RenamePath(sessionID, join(name), join(newName))
      store.refresh(store.get().path)
    } catch (err) { await alertOnError(err, 'Rename failed: ', 'Error') }
  }

  async function viewFile(file) {
    const name = file.Name || file.name
    if (file.IsDir || file.isDir) return
    try {
      const b64 = await ViewRemoteFile(sessionID, join(name))
      if (!b64) return
      const raw = atob(b64)
      setViewerData({ filename: name, data: raw, isBinary: isBinaryPreview(raw) })
    } catch (err) {
      await alertOnError(err, 'View failed: ', 'View Error')
    }
  }

  async function editPermissions(file) {
    const name = file.Name || file.name
    const mode = await dialog.prompt(`Unix file mode for "${name}" (e.g. 0644):`, 'Permissions')
    if (!mode) return
    try {
      await Chmod(sessionID, join(name), mode, false)
      store.refresh(store.get().path)
    } catch (err) { await alertOnError(err, 'Chmod failed: ', 'Error') }
  }

  async function copyFile(file) {
    const name = file.Name || file.name
    const dst = await dialog.prompt(`Copy "${name}" to:`, 'Copy', join('copy_of_' + name))
    if (!dst || dst === join(name)) return
    try {
      await CopyPath(sessionID, join(name), dst)
      store.refresh(store.get().path)
    } catch (err) { await alertOnError(err, 'Copy failed: ', 'Error') }
  }

  async function moveFile(file) {
    const name = file.Name || file.name
    const dst = await dialog.prompt(`Move "${name}" to:`, 'Move', '')
    if (!dst || dst === join(name)) return
    try {
      await CopyPath(sessionID, join(name), dst)
      await RemovePath(sessionID, join(name), file.IsDir || file.isDir)
      store.refresh(store.get().path)
    } catch (err) { await alertOnError(err, 'Move failed: ', 'Error') }
  }

  async function downloadMultipleTar(files) {
    if (!files || files.length === 0) return
    const items = files.map((f) => ({
      remotePath: join(f.Name || f.name),
      isDirectory: !!(f.IsDir || f.isDir),
    }))
    try {
      await DownloadMultipleTar(sessionID, items)
    } catch (err) {
      await alertOnError(err, 'Bulk download failed: ', 'Download Error')
    }
  }

  return {
    downloadDir, downloadFile, downloadMultipleTar, uploadFile, newFolder,
    deleteFile, renameFile, viewFile, editPermissions,
    copyFile, moveFile, joinPath: join,
  }
}
