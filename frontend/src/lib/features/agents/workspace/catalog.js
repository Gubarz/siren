// Command catalog → { category, commands } list shared by the agents
// workspace dropdown and the agent right-click menu, so both menus always
// show the same command categories.
export function catalogToCategories(catalog) {
  return (catalog?.groups ?? [])
    .map((group) => ({
      category: group.title || group.id || 'Other',
      commands: (group.commands ?? []).filter((c) => c.name),
    }))
    .filter((group) => group.commands.length > 0)
}
