<script>
  import Modal from "../../components/patterns/Modal.svelte";
  import Button from "../../components/ui/Button.svelte";
  import TextArea from "../../components/ui/TextArea.svelte";
  import IconButton from "../../components/ui/IconButton.svelte";
  import {
    GetEntityComments,
    AddEntityComment,
    DeleteEntityComment,
  } from "../../api/comments.js";
  import { errorMessage } from "../../utils/errors.js";
  import { now } from "../../stores/ui/now.svelte.js";
  import { connection } from "../../stores/connection.svelte.js";
  import { cleanUsername } from "../../utils/text.js";
  import { formatRelativeTime } from "../../utils/formats.js";

  let {
    open = $bindable(false),
    onclose = () => {},
    entityType = "",
    entityID = "",
    entityLabel = "",
  } = $props();

  let items = $state([]);
  let textDraft = $state("");
  let authorName = $derived(cleanUsername(connection?.profile));
  let loading = $state(false);
  let submitting = $state(false);
  let error = $state("");

  $effect(() => {
    if (!open || !entityType || !entityID) return;
    loadComments();
  });

  async function loadComments() {
    loading = true;
    error = "";
    try {
      const res = await GetEntityComments(entityType, entityID);
      items = res || [];
    } catch (err) {
      error = errorMessage(err, "Failed to load comments: ");
    } finally {
      loading = false;
    }
  }

  async function handleAdd() {
    const text = textDraft.trim();
    if (!text) return;
    submitting = true;
    error = "";
    try {
      await AddEntityComment(entityType, entityID, authorName, text);
      textDraft = "";
      await loadComments();
    } catch (err) {
      error = errorMessage(err, "Failed to post comment: ");
    } finally {
      submitting = false;
    }
  }

  async function handleDelete(id) {
    try {
      await DeleteEntityComment(id);
      await loadComments();
    } catch (err) {
      error = errorMessage(err, "Failed to delete comment: ");
    }
  }

  function onKeydown(e) {
    if (e.key === "Enter" && (e.ctrlKey || e.metaKey)) {
      e.preventDefault();
      handleAdd();
    }
  }
</script>

<Modal
  bind:open
  title={`Comments — ${entityType.toUpperCase()}: ${entityLabel || entityID}`}
  size="lg"
  {onclose}
>
  <div class="space-y-4">
    <div
      class="flex items-center justify-between text-xs text-fg-muted border-b border-line pb-2"
    >
      <span class="capitalize font-medium text-fg"
        >Entity: {entityType} / <span class="font-mono">{entityID}</span></span
      >
      <span>{items.length} comment{items.length === 1 ? "" : "s"}</span>
    </div>

    {#if loading}
      <div class="text-sm text-fg-muted py-4 text-center">
        Loading comments…
      </div>
    {:else if items.length === 0}
      <div
        class="py-6 text-center text-fg-muted text-sm border border-dashed border-line rounded"
      >
        No comments yet for this {entityType}. Add one below!
      </div>
    {:else}
      <div class="space-y-3 max-h-80 overflow-y-auto pr-1">
        {#each items as item (item.id)}
          <div
            class="p-3 rounded bg-surface-100 border border-line text-sm space-y-1"
          >
            <div class="flex items-center justify-between">
              <div class="flex items-center gap-2">
                <span
                  class="font-semibold text-xs text-primary-400 bg-primary-950/60 px-2 py-1 rounded border border-primary-800/40"
                >
                  {item.author || "Operator"}
                </span>
                <span class="text-3xs font-mono text-fg-muted"
                  >{formatRelativeTime(item.createdAt, now.value)}</span
                >
              </div>
              <IconButton
                icon="trash"
                label="Delete comment"
                size="xs"
                onclick={() => handleDelete(item.id)}
                class="text-fg-muted hover:text-danger-500"
              />
            </div>
            <p
              class="text-fg whitespace-pre-wrap text-xs leading-relaxed font-sans"
            >
              {item.text}
            </p>
          </div>
        {/each}
      </div>
    {/if}

    {#if error}
      <div class="text-sm text-danger-500">{error}</div>
    {/if}

    <div class="border-t border-line pt-3 space-y-2">
      <TextArea
        bind:value={textDraft}
        rows={3}
        placeholder={`Add a comment to this ${entityType}...`}
        onkeydown={onKeydown}
      />
    </div>
  </div>

  {#snippet footer()}
    <div class="flex justify-end gap-2">
      <Button color="dark" onclick={() => (open = false)}>Close</Button>
      <Button
        color="primary"
        icon="message-square"
        onclick={handleAdd}
        disabled={submitting || !textDraft.trim()}
      >
        {submitting ? "Posting…" : "Post Comment"}
      </Button>
    </div>
  {/snippet}
</Modal>
