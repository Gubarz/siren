import {
  EventsOn,
  OnFileDrop,
  OnFileDropOff,
  Quit,
  WindowMinimise,
  WindowToggleMaximise,
} from '../../../wailsjs/runtime/runtime.js';
import { OpenFileDialog as WailsOpenFileDialog } from '../../../wailsjs/go/main/App.js';

// Compatibility alias — matches the original App-binding name so callers
// swapping from `import { OpenFileDialog } from wailsjs` only change the
// import path, not the call site.
export const OpenFileDialog = WailsOpenFileDialog;

const fileDropListeners = new Set();
let fileDropRegistered = false;

export function onSliverEvent(callback) {
  return EventsOn('sliver-event', callback);
}

export function onShellOutput(callback) {
  return EventsOn('shell-output', callback);
}

// Console subprocess events. Payload: {jobID, data (base64)} for output,
// {jobID, exitCode?} for exit. Emitted by internal/console/subproc.go.
export function onConsoleOutput(callback) {
  return EventsOn('console-output', callback);
}

export function onConsoleExit(callback) {
  return EventsOn('console-exit', callback);
}

export function onConsoleOpenShell(callback) {
  return EventsOn('console-open-shell', callback);
}

// Generic named-event subscription — wrap wails' EventsOn so store code
// doesn't reach into the wailsjs bindings directly.
export function onWailsEvent(name, callback) {
  return EventsOn(name, callback);
}

export function onFileDrop(callback) {
  fileDropListeners.add(callback);
  if (!fileDropRegistered) {
    OnFileDrop((x, y, paths) => {
      for (const listener of fileDropListeners) listener(x, y, paths);
    }, true);
    fileDropRegistered = true;
  }

  return () => {
    fileDropListeners.delete(callback);
    if (fileDropListeners.size === 0 && fileDropRegistered) {
      OnFileDropOff();
      fileDropRegistered = false;
    }
  };
}

export function minimizeWindow() {
  WindowMinimise();
}

export function toggleMaximizeWindow() {
  WindowToggleMaximise();
}

export function quitApplication() {
  Quit();
}
