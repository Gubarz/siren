# State Management Rules

All global state in this app lives under `src/lib/stores/` and uses **Svelte 5 runes only** — no `svelte/store`, no `writable`, no `readable`, no `derived` from `svelte/store`.

## Rules

1. **Every store file uses the `.svelte.js` extension.** Runes (`$state`, `$derived`, `$effect`) only work in `.svelte`, `.svelte.js`, or `.svelte.ts` files. A plain `.js` file cannot host state.
2. **No `svelte/store` imports anywhere in the frontend.** If you find yourself reaching for `writable` / `readable` / `get` / `derived` (from `svelte/store`) / `fromStore`, stop and use a rune module instead. The one legal use of `derived` is the *runes* `$derived` — that is a rune, not the store import.
3. **Export a single reactive instance, not a `subscribe` function.** Use a class (or a factory returning a plain object) whose public fields are declared with `$state`. Consumers read those fields directly (`config.zoom`), not via `$config`.
4. **Use `$state` for data, `$derived` for computed values, `$effect` for side effects.** The old `store.update(s => ...)` idiom becomes a plain mutation (`this.foo = next`) or a reassignment (`this.state = { ...this.state, foo: next }`).
5. **Resource stores (fetched, event-driven, polled) use `createResource` from `./lib/createResource.svelte.js`.** It exposes reactive `.data` / `.loading` / `.error` / `.lastFetched` and requires callers to `.acquire()` / `.release()` for lifecycle. In components, prefer the `useResource(resource)` helper — it wires acquire/release into an `$effect` for you.
6. **Per-agent scoped stores use `createPerKeyStore` from `./lib/createPerKeyStore.svelte.js`.** Same acquire/release contract, keyed by session ID.

## Anti-patterns to reject in review

- `import { writable } from 'svelte/store'` — banned.
- Adding a `subscribe` method to a store to make it "auto-subscribe-compatible" — banned; consumers should read `.field` directly.
- Using `get(store)` — the value is a reactive field, just read it.
- Global mutable module-level `let` that isn't wrapped in `$state` — reactivity will silently not track it.

## Example — simple store

```js
// src/lib/stores/myThing.svelte.js
class MyThing {
  count = $state(0)
  label = $state('hello')

  increment() {
    this.count += 1
  }
}

export const myThing = new MyThing()
```

```svelte
<!-- consumer.svelte -->
<script>
  import { myThing } from '$stores/myThing.svelte.js'
</script>

<button onclick={() => myThing.increment()}>
  {myThing.label}: {myThing.count}
</button>
```

## Example — resource store

```js
// src/lib/stores/resources/widgets.svelte.js
import { createResource } from '../lib/createResource.svelte.js'
import { listWidgets } from '../../api/widgets.js'

export const widgets = createResource({
  name: 'widgets',
  fetch: () => listWidgets(),
  events: ['widget-created', 'widget-deleted'],
})
```

```svelte
<script>
  import { widgets } from '$stores/resources/widgets.svelte.js'
  import { useResource } from '$stores/lib/createResource.svelte.js'

  useResource(widgets) // acquires on mount, releases on destroy
</script>

{#if widgets.loading}Loading…{/if}
{#each widgets.data as w}<div>{w.name}</div>{/each}
```
