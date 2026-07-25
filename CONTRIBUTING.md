# Contributing to sliver-gui

This project moves fastest when the rules stay small and visible.

## Size limits

Keep files and functions short enough that someone can review them without
having to build a map in their head.

| Language | File limit | Function limit | Checked by |
| --- | --- | --- | --- |
| Go | **350 lines** | **45 lines** | `funlen` in [.golangci.yml](.golangci.yml) and [scripts/check-go-file-length.sh](scripts/check-go-file-length.sh) |
| JavaScript | 400 lines | 45 lines | `max-lines` and `max-lines-per-function` in [frontend/eslint.config.js](frontend/eslint.config.js) |
| Svelte | 300 lines | - | `max-lines` for `.svelte` |

`cmd/gui/app.go` is the composition root: keep the `App` type, service 
ownership, and construction there. Put Wails-facing methods in the
domain-oriented `cmd/gui/bindings_*.go` file that matches the service
they delegate to. Lifecycle and connection state belong in
`cmd/gui/lifecycle.go` and `cmd/gui/connection.go`.

Frontend size checks currently warn instead of failing, mostly so older code
doesn't block unrelated work. For new code, treat the warning as a stop sign.

Run the full set with:

```sh
make analyze
```

## Frontend boundaries

The frontend is split into layers. Imports should move down the stack:

```text
features/  ->  api/, stores/, patterns/, ui/, utils/
patterns/  ->  api/, stores/, ui/, utils/
system/    ->  api/, stores/, patterns/, ui/
ui/        ->  utils/
stores/    ->  api/, utils/, stores/
api/       ->  utils/ and wailsjs/
```

Only `src/lib/api/` should import from `wailsjs`. The lint rules enforce this
with `eslint-plugin-boundaries` and `no-restricted-imports`.

So if a feature needs a new App method, add a wrapper in the right
`src/lib/api/*.js` file first. Feature code should import that wrapper, not the
generated Wails binding.

## Adding RPC bindings

When you expose a new backend call to the UI, update all of the places that
describe or wrap it:

1. Add the backend wrapper in `internal/<pkg>/*.go`.
2. Add the thin App method to the matching `cmd/gui/bindings_*.go` file.
3. Re-export it from `frontend/src/lib/api/*.js`.

## Project habits

- Polling resources live in `frontend/src/lib/stores/resources/` and use the
  `{data, loading, error, startPolling, stopPolling}` shape.
- Modals with reusable settings should include a `PresetPicker` slot so
  operators can save and apply presets.
- RPC bindings should surface the raw server error message. Do not swallow it.
  `annotateGenerateError` is the pattern to follow.
- Do not add new `<style>` blocks in `.svelte` files. Use Tailwind utilities,
  or put unavoidable global rules in `src/styles/main.css`.
- Do not duplicate UI primitives inside feature code. Raw `<button>`,
  `<input>`, `<textarea>`, `<select>`, and `<input type="checkbox">` are banned
  in `src/lib/features/`; use the `$components/ui/*` wrappers.
