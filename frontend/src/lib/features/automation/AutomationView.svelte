<script>
  import {
    ClearAutomationHistory,
    DeleteAutomationRule,
    ExportAutomationRules,
    ImportAutomationRules,
    RunAutomationRule,
    SaveAutomationRule,
    SetAutomationRuleEnabled,
  } from '../../api/automation.js';
  import StarterLibraryModal from './StarterLibraryModal.svelte';
  import { toast } from '$stores/ui/toast.svelte.js';
  import { dialog } from '../../stores/ui/dialog.svelte.js';
  import { errorMessage } from '../../utils/errors.js';
  import { automation } from '$stores/resources/automation.svelte.js';
  import { automationHistory } from '$stores/resources/automationHistory.svelte.js';
  import { useResource } from '$stores/lib/createResource.svelte.js'

  useResource(automation, automationHistory)
  import { onMount } from 'svelte';
  import AutomationRuleList from './AutomationRuleList.svelte';
  import AutomationRuleEditor from './AutomationRuleEditor.svelte';
  import AutomationHistoryPanel from './AutomationHistoryPanel.svelte';
  import Button from '$components/ui/Button.svelte';
  import ErrorState from '$components/ui/ErrorState.svelte';

  let selectedID = $state('');
  let draft = $state(null);
  let selectedRun = $state(null);
  let activePanel = $state('rule');
  let manualTarget = $state('');
  let busy = $state(false);
  let starterOpen = $state(false);
  let fileInputEl = $state(null);

  let rules = $derived(Array.isArray(automation.data) ? automation.data : []);
  let history = $derived(Array.isArray(automationHistory.data) ? automationHistory.data : []);

  onMount(() => {
    automation.refresh();
    automationHistory.refresh();
  });

  $effect(() => {
    if (activePanel === 'history' && !selectedRun && history.length > 0) {
      selectedRun = history[0];
    }
  });

  function blankRule() {
    return {
      id: '', name: 'New automation', description: '', enabled: true,
      trigger: 'manual', targetKind: 'any',
      filter: { os: '', arch: '', hostname: '', username: '', name: '' },
      executionMode: 'commands', commands: [''],
      script: '// target: { id, name, hostname, username, os, arch, kind }\nconst whoami = sliver.run("whoami");\nsliver.log("Agent:", target.name || target.hostname);\nsliver.log(whoami);',
      timeoutSeconds: 300, continueOnError: false, delaySeconds: 0,
      cooldownSeconds: 0, intervalSeconds: 60, maxRuns: 0, runCount: 0,
      __dirty: true,
    };
  }

  function copyRule(rule) {
    const copy = JSON.parse(JSON.stringify(rule));
    copy.filter ||= { os: '', arch: '', hostname: '', username: '', name: '' };
    copy.commands ||= []; copy.executionMode ||= 'commands'; copy.script ||= ''; copy.timeoutSeconds ||= 300;
    copy.__dirty = false;
    return copy;
  }

  function handleNew() { selectedID = ''; draft = blankRule(); activePanel = 'rule'; manualTarget = ''; }

  function handleSelect(rule) { selectedID = rule.id; draft = copyRule(rule); activePanel = 'rule'; manualTarget = ''; }

  function handleToggle(rule) { SetAutomationRuleEnabled(rule.id, !rule.enabled).then(() => automation.refresh()).catch(() => {}); }

  function handleFilter(eventData) { if (draft) { draft.filter = { ...draft.filter, [eventData.key]: eventData.value }; markDirty(); } }

  function handleCommands(value) { if (draft) { draft.commands = value.split('\n'); markDirty(); } }

  function markDirty() { if (draft) draft.__dirty = true; }

  async function handleSave() { if (!draft || busy) return; busy = true; try { const payload = { ...draft }; delete payload.__dirty; const saved = await SaveAutomationRule(payload); selectedID = saved.id; draft = copyRule(saved); await automation.refresh(); } catch (error) { await dialog.alert(`Could not save automation: ${errorMessage(error)}`); } finally { busy = false; } }

  async function handleDelete() { if (!draft?.id || !(await dialog.confirm(`Delete "${draft.name}"?`, 'Delete Automation'))) return; busy = true; try { await DeleteAutomationRule(draft.id); selectedID = ''; draft = null; await automation.refresh(); } catch (error) { await dialog.alert(`Could not delete automation: ${errorMessage(error)}`); } finally { busy = false; } }

  async function handleRun() { if (!draft?.id) { await dialog.alert('Save the rule before running it.'); return; } busy = true; try { await RunAutomationRule(draft.id, manualTarget); activePanel = 'history'; setTimeout(() => automationHistory.refresh(), 150); } catch (error) { await dialog.alert(`Could not run automation: ${errorMessage(error)}`); } finally { busy = false; } }

  async function handleClearHistory() { if (!(await dialog.confirm('Clear all automation run history?', 'Clear History'))) return; try { await ClearAutomationHistory(); selectedRun = null; await automationHistory.refresh(); } catch {} }

  function handleHistorySelect(run) { selectedRun = run; }

  function showRuleEditor() {
    activePanel = 'rule';
  }

  function showHistory() {
    activePanel = 'history';
    if (!selectedRun && history.length > 0) selectedRun = history[0];
  }

  async function handleExport() {
    try {
      const path = await ExportAutomationRules();
      if (path) toast.push({ variant: 'success', message: `Rules exported to ${path}` });
    } catch (e) {
      await dialog.alert(`Export failed: ${errorMessage(e)}`);
    }
  }

  function handleImport() { fileInputEl?.click(); }

  async function handleFilePicked(event) {
    const file = event.currentTarget.files?.[0];
    event.currentTarget.value = '';
    if (!file) return;
    try {
      const text = await file.text();
      const result = await ImportAutomationRules(text);
      await automation.refresh();
      const parts = [`imported ${result.imported ?? 0}`];
      if (result.skipped) parts.push(`skipped ${result.skipped}`);
      toast.push({ variant: result.errors?.length ? 'error' : 'success', message: `Rules ${parts.join(', ')}` });
      if (result.errors?.length) {
        await dialog.alert(result.errors.join('\n'), 'Import errors');
      }
    } catch (e) {
      await dialog.alert(`Import failed: ${errorMessage(e)}`);
    }
  }

  function handleStartersImported() { automation.refresh(); }
</script>

<div class="flex flex-1 min-h-0 flex-col overflow-hidden bg-canvas">
  <div class="flex h-10 shrink-0 border-b border-line bg-chrome-header">
    <Button
      color="alternative"
      class={`!px-4 !border-0 !border-b-2 !rounded-none !bg-transparent focus:!ring-0 focus:!outline-none focus-visible:!ring-0 focus-visible:!outline-none ${activePanel === 'rule' ? '!border-brand !text-brand' : '!border-transparent text-fg-muted'}`}
      onclick={showRuleEditor}
    >Rule editor</Button>
    <Button
      color="alternative"
      class={`!px-4 !border-0 !border-b-2 !rounded-none !bg-transparent focus:!ring-0 focus:!outline-none focus-visible:!ring-0 focus-visible:!outline-none ${activePanel === 'history' ? '!border-brand !text-brand' : '!border-transparent text-fg-muted'}`}
      onclick={showHistory}
    >Execution history</Button>
  </div>

  <div class="flex flex-1 min-h-0 overflow-hidden">
    <AutomationRuleList {rules} {selectedID} {selectedRun} {activePanel} {history}
      onnew={handleNew} onselect={handleSelect} ontoggle={handleToggle}
      onhistory={(panel) => panel === 'rule' ? showRuleEditor() : showHistory()}
      onhistoryselect={handleHistorySelect} onclearhistory={handleClearHistory}
      onexport={handleExport} onimport={handleImport} onstarters={() => starterOpen = true} />

    <input bind:this={fileInputEl} type="file" accept="application/json,.json" class="hidden" onchange={handleFilePicked} />
    <StarterLibraryModal bind:open={starterOpen} onimported={handleStartersImported} />

    <section class="flex flex-1 min-w-0 flex-col">
      {#if automation.error || automationHistory.error}
        <ErrorState error={automation.error || automationHistory.error} title="Failed to load automation data" class="m-4" />
      {/if}

      {#if activePanel === 'rule'}
        <AutomationRuleEditor bind:draft={draft} {busy} bind:manualTarget={manualTarget}
          onnew={handleNew} ondirty={markDirty} onfilter={handleFilter} oncommands={handleCommands}
          onsave={handleSave} ondelete={handleDelete} onrun={handleRun} />
      {:else}
        <AutomationHistoryPanel {selectedRun} />
      {/if}
    </section>
  </div>
</div>
