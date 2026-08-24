import { Application, Events, Window } from '@wailsio/runtime';
import { OpenFileDialog as WailsOpenFileDialog } from '../../../bindings/siren/cmd/gui/app.js';

// Compatibility alias — matches the original App-binding name so callers
// swapping from the generated bindings only change the import path, not the
// call site.
export const OpenFileDialog = WailsOpenFileDialog;

// v3 event callbacks receive a WailsEvent object; unwrap the payload so
// subscribers keep the v2 callback shape (raw data only).
function subscribe(name, callback) {
  return Events.On(name, (event) => callback(event?.data));
}

export function onSliverEvent(callback) {
  return subscribe('sliver-event', callback);
}

// BloodHound backend events. Payload: {type, payload} where payload is the
// JSON-serialized bus event body (status / enrichment / synced).
export function onBloodhoundEvent(callback) {
  return subscribe('bloodhound-event', callback);
}

export function onShellOutput(callback) {
  return subscribe('shell-output', callback);
}

// Console subprocess events. Payload: {jobID, data (base64)} for output,
// {jobID, exitCode?} for exit. Emitted by internal/console/subproc.go.
export function onConsoleOutput(callback) {
  return subscribe('console-output', callback);
}

export function onConsoleExit(callback) {
  return subscribe('console-exit', callback);
}

export function onConsoleOpenShell(callback) {
  return subscribe('console-open-shell', callback);
}

// Generic named-event subscription — wrap the runtime so store code
// doesn't reach into the generated bindings directly.
export function onWailsEvent(name, callback) {
  return subscribe(name, callback);
}

const fileDropListeners = new Set();
let fileDropRegistered = false;
let currentWindowName = '';

// v3 only delivers drops that land on elements carrying the
// data-file-drop-target attribute; the Go backend re-emits them as the
// 'files-dropped' custom event (see cmd/gui/lifecycle.go).
export function onFileDrop(callback) {
  fileDropListeners.add(callback);
  if (!fileDropRegistered) {
    Window.Name().then((name) => { currentWindowName = name }).catch(() => {});
    Events.On('files-dropped', (event) => {
      if (event?.sender && currentWindowName && event.sender !== currentWindowName) return;
      const data = event?.data;
      if (!data) return;
      for (const listener of fileDropListeners) listener(data.x, data.y, data.files);
    });
    fileDropRegistered = true;
  }

  return () => {
    fileDropListeners.delete(callback);
    // The underlying Events.On subscription is process-lifetime; only the
    // listener set is drained. (Events.Off would nuke every listener for the
    // channel, so we deliberately don't call it.)
  };
}

export function minimizeWindow() {
  Window.Minimise();
}

export function toggleMaximizeWindow() {
  Window.ToggleMaximise();
}

export function closeWindow() {
  Window.Close();
}

export function quitApplication() {
  Application.Quit();
}
