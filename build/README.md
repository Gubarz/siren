# Build Directory

The build directory holds platform-specific assets and the `config.yml` file
that drives `wails3 dev`.

## Structure

* `bin/` — output directory (`make build` places binaries here)
* `darwin/` — macOS specific files
* `windows/` — Windows specific files
* `config.yml` — Wails v3 dev-mode pipeline

## Mac

The `darwin` directory holds files specific to Mac builds.
These may be customised and used as part of the build. To return these files
to the default state, simply delete them and rebuild.

The directory contains the following files:

- `Info.plist` — the main plist file used for Mac builds.
- `Info.dev.plist` — same as the main plist file but used during `wails3 dev`.

## Windows

The `windows` directory contains the manifest and rc files used during the
Windows build. These may be customised for your application. To return these
files to the default state, simply delete them and rebuild.

- `icon.ico` — The icon used for the application. If it is missing, a new
  `icon.ico` file will be created using the `appicon.png` file in the build
  directory.
- `installer/*` — The files used to create the Windows installer.
- `info.json` — Application details used for Windows builds. The data here
  will be used by the Windows installer, as well as the application itself
  (right click the exe → properties → details).
- `wails.exe.manifest` — The main application manifest file.
