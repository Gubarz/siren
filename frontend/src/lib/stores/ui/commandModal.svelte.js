const DEFAULTS = {
  command: null,
  targetIDs: [],
  useSession: false,
  initialValues: {},
  sourceContext: '',
}

class CommandModal {
  command = $state(DEFAULTS.command)
  targetIDs = $state(DEFAULTS.targetIDs)
  useSession = $state(DEFAULTS.useSession)
  initialValues = $state(DEFAULTS.initialValues)
  sourceContext = $state(DEFAULTS.sourceContext)

  open({ command, targetIDs = [], useSession = true, initialValues = {}, sourceContext = '' }) {
    if (!command) return
    this.command = command
    this.targetIDs = targetIDs
    this.useSession = useSession
    this.initialValues = initialValues
    this.sourceContext = sourceContext
  }

  close() {
    this.command = DEFAULTS.command
    this.targetIDs = DEFAULTS.targetIDs
    this.useSession = DEFAULTS.useSession
    this.initialValues = DEFAULTS.initialValues
    this.sourceContext = DEFAULTS.sourceContext
  }
}

export const commandModal = new CommandModal()
