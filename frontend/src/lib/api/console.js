export {
  CloseShell,
  GetCommandCatalog,
  GetShellOutput,
  InterruptShell,
  ListCommands,
  ResizeShell,
  RunSessionCommand,
  StartShell,
  WriteShell,
} from '../../../wailsjs/go/gui/App.js';

// Subprocess-console methods aren't in the generated wailsjs bindings
// until the next `wails dev`/`build`. Route through the runtime global
// so a static import doesn't break the build.
function app() {
  return globalThis.go?.gui?.App;
}

export function StartConsole(sessionID) {
  return app().StartConsole(sessionID);
}

export function WriteConsole(jobID, data) {
  return app().WriteConsole(jobID, data);
}

export function ResizeConsole(jobID, cols, rows) {
  return app().ResizeConsole(jobID, cols, rows);
}

export function StopConsole(jobID) {
  return app().StopConsole(jobID);
}

export function SendToSessionConsole(sessionID, line) {
  return app().SendToSessionConsole(sessionID, line);
}
