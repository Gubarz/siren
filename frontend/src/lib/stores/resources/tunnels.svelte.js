import { createResource } from '../lib/createResource.svelte.js'
import { ListProxies, ListRportfwds } from '../../api/agents.js'

// Server-wide tunnel state (all sessions). Combines proxy list (SOCKS +
// portfwd) with reverse portfwds so the tunnels tab renders from a single
// reactive slice.
export const tunnels = createResource({
  name: 'tunnels',
  fetch: async () => {
    const [proxies, rportfwds] = await Promise.all([
      ListProxies().then((r) => r || []),
      ListRportfwds().then((r) => r || []),
    ])
    return { proxies, rportfwds }
  },
  pollInterval: 5000,
  events: ['tunnels-changed'],
})
