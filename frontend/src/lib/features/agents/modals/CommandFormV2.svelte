<script>
  import { stripTerminalFormatting } from '../../../utils/text.js';
  import { quote } from '../../../utils/shell.js';
  import Modal from '../../../components/patterns/Modal.svelte';
  import CollapsibleGroup from '../../../components/forms/CollapsibleGroup.svelte';
  import Button from '../../../components/ui/Button.svelte';
  import Checkbox from '../../../components/ui/Checkbox.svelte';
  import TextInput from '../../../components/ui/TextInput.svelte';
  import PidPickerField from './pickers/PidPickerField.svelte';
  import RemoteFilePickerField from './pickers/RemoteFilePickerField.svelte';
  import FilePickerField from '../../../components/forms/FilePickerField.svelte';
  import PresetPicker from '../../../components/forms/PresetPicker.svelte';
  import { flagOverrides, GROUP_ORDER, GROUP_LABELS, COLLAPSED_GROUPS } from './flagOverrides.js';

  let {
    sessionID = '',
    firstSessionID = '',
    command = null,
    open = $bindable(false),
    onexecute,
    onclose,
  } = $props();

  let argumentValues = $state({});
  let flagValues = $state({});
  let advancedLine = $state('');
  let advancedLineOpen = $state(false);
  let overrides = $derived(command && flagOverrides[command.name]);

  $effect.pre(() => {
    if (command) initializeForm(command);
  });

  let commandPreview = $derived(command ? buildCommand(command, flagValues, argumentValues, advancedLine) : '');
  let formValid = $derived(command ? isValid(command, flagValues, argumentValues) : false);

  // Build the preset payload — flat `{ args: {...}, flags: {...} }` shape so
  // apply can restore each map independently. Passwords are dropped so saved
  // presets never carry plaintext credentials on disk.
  let presetValues = $derived.by(() => {
    if (!command) return { args: {}, flags: {} };
    const flags = {};
    for (const flag of command.flags ?? []) {
      if (inputType(flag) === 'password') continue;
      flags[flag.name] = flagValues[flag.name];
    }
    return { args: { ...argumentValues }, flags, advanced: advancedLine };
  });

  function applyPreset(values) {
    if (!values) return;
    // Nested shape.
    if (values.args && values.flags) {
      for (const [k, v] of Object.entries(values.args)) {
        if (k in argumentValues) argumentValues[k] = v;
      }
      for (const [k, v] of Object.entries(values.flags)) {
        if (k in flagValues) flagValues[k] = v;
      }
      if (values.advanced != null) {
        advancedLine = values.advanced;
        if (values.advanced.trim()) advancedLineOpen = true;
      }
      return;
    }
    // Flat shape: try each key against both maps.
    for (const [k, v] of Object.entries(values)) {
      if (k in argumentValues) argumentValues[k] = v;
      else if (k in flagValues) flagValues[k] = v;
    }
  }

  function initializeForm(currentCommand) {
    const nextArgs = {};
    const nextFlags = {};

    const o = flagOverrides[currentCommand.name];
    for (const arg of currentCommand.arguments ?? []) {
      const override = o?.args?.find((a) => a.name === arg.name);
      nextArgs[arg.name] = override?.default ?? '';
    }
    for (const flag of currentCommand.flags ?? []) {
      const override = o?.flags?.[flag.name];
      if (flag.boolean) {
        nextFlags[flag.name] = false;
      } else {
        nextFlags[flag.name] = override?.default ?? (flag.default ?? '');
      }
    }
    argumentValues = nextArgs;
    flagValues = nextFlags;
  }

  function buildCommand(currentCommand, currentFlags, currentArguments, extraLine) {
    const parts = [currentCommand.path];
    for (const flag of currentCommand.flags ?? []) {
      const value = currentFlags[flag.name];
      if (flag.boolean) { if (value) parts.push(`--${flag.name}`); }
      else if (value !== '' && value !== null && value !== undefined) { parts.push(`--${flag.name}`, quote(value)); }
    }
    for (const argument of currentCommand.arguments ?? []) {
      const value = currentArguments[argument.name];
      if (!value) continue;
      parts.push(argument.variadic ? value : quote(value));
    }
    if (extraLine.trim()) parts.push(extraLine.trim());
    return parts.join(' ');
  }

  function execute() {
    if (!formValid) return;
    onexecute?.({ cmd: commandPreview });
  }

  function isValid(currentCommand, currentFlags, currentArguments) {
    if (!currentCommand.supported) return false;
    return (currentCommand.arguments ?? []).every(
      (a) => !a.required || Boolean(currentArguments[a.name]?.trim()),
    ) && (currentCommand.flags ?? []).every(
      (f) => !f.required || (f.boolean ? Boolean(currentFlags[f.name]) : Boolean(String(currentFlags[f.name] ?? '').trim())),
    );
  }

  function inputType(flag) {
    const name = flag.name.toLowerCase();
    if (name.includes('password') || name === 'pass') return 'password';
    if (/^(int|int32|int64|uint|uint32|uint64|float)/.test(flag.type)) return 'number';
    return 'text';
  }

  function flagDescription(flag) {
    if (!overrides?.flags?.[flag.name]) return flag.usage;
    return flag.usage;
  }

  function overrideFor(flag) {
    return overrides?.flags?.[flag.name];
  }

  function argumentOverride(arg) {
    return overrides?.args?.find((a) => a.name === arg.name);
  }

  function labelFor(flag) {
    const o = overrideFor(flag);
    if (o?.label) return o.label;
    return flag.name.replaceAll('-', ' ').replace(/\b\w/g, (c) => c.toUpperCase());
  }

  function argLabel(arg) {
    const o = argumentOverride(arg);
    if (o?.label) return o.label;
    return arg.name.replaceAll('-', ' ').replace(/\b\w/g, (c) => c.toUpperCase());
  }

  function argGroup(arg) {
    return argumentOverride(arg)?.group ?? 'payload';
  }

  let groupedFields = $derived.by(() => {
    if (!command) return [];
    const groups = {};
    for (const g of GROUP_ORDER) groups[g] = [];

    for (const arg of command.arguments ?? []) {
      const g = argGroup(arg);
      groups[g]?.push({ kind: 'arg', arg });
    }
    for (const flag of command.flags ?? []) {
      const o = overrideFor(flag);
      const g = o?.group ?? 'other';
      groups[g]?.push({ kind: 'flag', flag });
    }

    return GROUP_ORDER
      .filter((g) => groups[g]?.length > 0)
      .map((g) => ({
        key: g,
        label: GROUP_LABELS[g] || g,
        collapsed: COLLAPSED_GROUPS.has(g),
        fields: groups[g],
      }));
  });

</script>

{#if command}
  <Modal bind:open title="{command.name}{sessionID ? ` - ${sessionID}` : ''}" size="2xl" {onclose}>
    
      {#if command.description}
        <p class="text-fg-muted text-sm mb-4 whitespace-pre-line">{stripTerminalFormatting(command.description)}</p>
      {/if}

      {#if !command.supported}
        <div class="mb-4 p-2 border border-danger-500 rounded bg-danger-500/10 text-danger-500">{command.unavailable}</div>
      {/if}

      {#each groupedFields as section}
        <CollapsibleGroup title={section.label} open={!section.collapsed}>
          {#each section.fields as field}
            {#if field.kind === 'arg'}
              <div class="mb-2">
                {#if argumentOverride(field.arg)?.widgetHint === 'file'}
                  <FilePickerField
                    bind:value={argumentValues[field.arg.name]}
                    label={argLabel(field.arg)}
                  />
                {:else if argumentOverride(field.arg)?.widgetHint === 'remoteFile'}
                  <RemoteFilePickerField
                    bind:value={argumentValues[field.arg.name]}
                    sessionID={firstSessionID || sessionID}
                    label={argLabel(field.arg)}
                    mode="file"
                  />
                {:else if argumentOverride(field.arg)?.widgetHint === 'remoteDir'}
                  <RemoteFilePickerField
                    bind:value={argumentValues[field.arg.name]}
                    sessionID={firstSessionID || sessionID}
                    label={argLabel(field.arg)}
                    mode="dir"
                  />
                {:else}
                  <label class="mb-1 block text-sm font-medium text-fg" for="arg-{field.arg.name}">{argLabel(field.arg)}</label>
                  <TextInput
                    id="arg-{field.arg.name}"
                    type="text"
                    size="sm"
                    bind:value={argumentValues[field.arg.name]}
                    placeholder={field.arg.name}
                  />
                {/if}
                {#if field.arg.required}
                  <span class="mt-1 block text-xs text-danger-500">Required</span>
                {/if}
              </div>
            {:else}
              <div class="mb-2">
                {#if !['pidPicker', 'file', 'remoteFile', 'remoteDir'].includes(overrideFor(field.flag)?.widgetHint)}
                  <label class="mb-1 block text-sm font-medium text-fg" for="flag-{field.flag.name}">{labelFor(field.flag)}</label>
                {/if}
                {#if field.flag.boolean}
                  <Checkbox bind:checked={flagValues[field.flag.name]} label={flagDescription(field.flag)} />
                {:else if overrideFor(field.flag)?.widgetHint === 'file'}
                  <FilePickerField
                    bind:value={flagValues[field.flag.name]}
                    label={labelFor(field.flag)}
                  />
                {:else if overrideFor(field.flag)?.widgetHint === 'remoteFile'}
                  <RemoteFilePickerField
                    bind:value={flagValues[field.flag.name]}
                    sessionID={firstSessionID || sessionID}
                    label={labelFor(field.flag)}
                    mode="file"
                  />
                {:else if overrideFor(field.flag)?.widgetHint === 'remoteDir'}
                  <RemoteFilePickerField
                    bind:value={flagValues[field.flag.name]}
                    sessionID={firstSessionID || sessionID}
                    label={labelFor(field.flag)}
                    mode="dir"
                  />
                {:else if overrideFor(field.flag)?.widgetHint === 'pidPicker'}
                  <PidPickerField
                    bind:value={flagValues[field.flag.name]}
                    label={labelFor(field.flag)}
                    sessionID={firstSessionID}
                  />
                {:else}
                  <TextInput
                    id="flag-{field.flag.name}"
                    type={inputType(field.flag)}
                    size="sm"
                    bind:value={flagValues[field.flag.name]}
                    placeholder={field.flag.usage}
                  />
                {/if}
                <span class="mt-1 block text-xs text-fg-muted">{flagDescription(field.flag)}</span>
              </div>
            {/if}
          {/each}
        </CollapsibleGroup>
      {/each}

      <div class="mb-3">
        <Button
          color="alternative"
          size="xs"
          icon={advancedLineOpen ? 'chevron-down' : 'chevron-right'}
          onclick={() => advancedLineOpen = !advancedLineOpen}
        >
          Advanced command-line arguments
        </Button>
      </div>
      {#if advancedLineOpen}
        <div class="mb-2">
          <label class="mb-1 block text-sm font-medium text-fg" for="advanced-command-line">Additional arguments</label>
          <TextInput id="advanced-command-line" size="sm" bind:value={advancedLine} placeholder="Passed through exactly as entered" onkeydown={(e) => e.key === 'Enter' && execute()} />
        </div>
      {/if}

      <div class="mb-4">
        <span class="block text-sm font-semibold text-fg mb-1">Command preview</span>
        <code class="block p-2 border border-line rounded bg-chrome text-fg break-all">{commandPreview}</code>
      </div>
    
  {#snippet footer()}
    <div class="flex justify-between items-center">
      <PresetPicker
        commandPath={command.path || command.name}
        currentValues={presetValues}
        onapply={applyPreset}
      />
      <div class="flex gap-2">
        <Button color="dark" onclick={() => open = false}>Cancel</Button>
        <Button color="primary" onclick={execute} disabled={!formValid}>Execute</Button>
      </div>
    </div>
  {/snippet}
</Modal>
{/if}
