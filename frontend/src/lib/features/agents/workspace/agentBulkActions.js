// Bulk-action handlers for the SessionsTable multi-select. Each entry
// returns a per-agent Promise so the caller can drive a shared progress
// panel and surface partial-failure detail. Kept factory-shaped for
// consistency with agentActions.js.

import { KillAgent, RenameAgent } from '../../../api/agents.js'
import { GetAgentTags, SetAgentTags } from '../../../api/tags.js'
import { errorMessage } from '../../../utils/errors.js'

export function createBulkActions({ dialog }) {
  async function runBatch(ids, label, fn) {
    const failures = []
    let succeeded = 0
    for (const id of ids) {
      try {
        await fn(id)
        succeeded++
      } catch (err) {
        failures.push({ id, message: errorMessage(err) })
      }
    }
    if (failures.length > 0) {
      const lines = failures.map((f) => `${f.id}: ${f.message}`).join('\n')
      await dialog.alert(
        `${label} completed for ${succeeded}/${ids.length}. Failures:\n${lines}`,
        `Bulk ${label}`,
      )
    } else {
      await dialog.alert(`${label} completed for ${ids.length} agent${ids.length === 1 ? '' : 's'}.`, `Bulk ${label}`)
    }
  }

  async function bulkKill(agents) {
    if (agents.length === 0) return
    if (!(await dialog.confirm(
      `Kill ${agents.length} agent${agents.length === 1 ? '' : 's'}? This cannot be undone.`,
      'Confirm Bulk Kill',
    ))) return
    await runBatch(agents.map((a) => a.ID), 'Kill', (id) => KillAgent(id))
  }

  async function bulkRenamePrefix(agents) {
    if (agents.length === 0) return
    const prefix = await dialog.prompt(
      `Prepend a prefix to ${agents.length} agent name${agents.length === 1 ? '' : 's'}:`,
      'Bulk Rename',
      'prod-',
    )
    if (!prefix) return
    await runBatch(agents.map((a) => a.ID), 'Rename', (id) => {
      const agent = agents.find((a) => a.ID === id)
      return RenameAgent(id, `${prefix}${agent?.Name || id}`)
    })
  }

  async function bulkAddTag(agents) {
    if (agents.length === 0) return
    const tag = await dialog.prompt(
      `Add a tag to ${agents.length} agent${agents.length === 1 ? '' : 's'}:`,
      'Bulk Add Tag',
      '',
    )
    const normalized = String(tag || '').trim().toLowerCase()
    if (!normalized) return
    await runBatch(agents.map((a) => a.ID), 'Add tag', async (id) => {
      const current = await GetAgentTags(id) || []
      if (current.includes(normalized)) return
      await SetAgentTags(id, [...current, normalized])
    })
  }

  async function bulkRemoveTag(agents) {
    if (agents.length === 0) return
    const tag = await dialog.prompt(
      `Remove a tag from ${agents.length} agent${agents.length === 1 ? '' : 's'}:`,
      'Bulk Remove Tag',
      '',
    )
    const normalized = String(tag || '').trim().toLowerCase()
    if (!normalized) return
    await runBatch(agents.map((a) => a.ID), 'Remove tag', async (id) => {
      const current = await GetAgentTags(id) || []
      if (!current.includes(normalized)) return
      await SetAgentTags(id, current.filter((t) => t !== normalized))
    })
  }

  return { bulkKill, bulkRenamePrefix, bulkAddTag, bulkRemoveTag }
}
