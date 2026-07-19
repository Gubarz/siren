<script>
  import {
    minimizeWindow,
    quitApplication,
    toggleMaximizeWindow,
  } from '../../api/runtime.js';
  import PrimaryNavigation from './PrimaryNavigation.svelte';
  import { navigation } from '$stores/ui/navigation.svelte.js';

  let activeView = $derived(navigation.activeView);

  function minimize() { minimizeWindow(); }
  function maximize() { toggleMaximizeWindow(); }
  function closeApp() { quitApplication(); }

  // When the window is unfocused, the OS consumes the activating click before
  // it can turn into a DOM `click` event — so the user has to click twice.
  // `pointerdown` fires ahead of that focus handoff, so the first press lands
  // as expected. Only respond to the primary button.
  function handlePress(action) {
    return (e) => {
      if (e.button !== 0) return;
      e.preventDefault();
      action();
    };
  }
</script>

<div class="relative z-titlebar h-10 bg-chrome-header flex justify-between items-stretch border-b border-line select-none shrink-0" style="--wails-draggable:drag">
  <div class="flex-1 h-full flex items-stretch">
    <PrimaryNavigation active={activeView} onSelect={(v) => navigation.setView(v)} />
  </div>

  <div class="flex h-full">
    <button type="button" style="--wails-draggable:no-drag" class="w-12 h-full p-0 border-0 bg-transparent flex justify-center items-center cursor-default transition-colors text-fg-muted hover:bg-row-hover hover:text-fg" aria-label="Minimize window" onpointerdown={handlePress(minimize)}>
      <svg width="12" height="12" viewBox="0 0 12 12"><rect fill="currentColor" width="10" height="1" x="1" y="6"></rect></svg>
    </button>
    <button type="button" style="--wails-draggable:no-drag" class="w-12 h-full p-0 border-0 bg-transparent flex justify-center items-center cursor-default transition-colors text-fg-muted hover:bg-row-hover hover:text-fg" aria-label="Maximize window" onpointerdown={handlePress(maximize)}>
      <svg width="12" height="12" viewBox="0 0 12 12"><rect width="9" height="9" x="1.5" y="1.5" fill="none" stroke="currentColor"></rect></svg>
    </button>
    <button type="button" style="--wails-draggable:no-drag" class="w-12 h-full p-0 border-0 bg-transparent flex justify-center items-center cursor-default transition-colors text-fg-muted hover:bg-danger-500! hover:text-white!" aria-label="Close window" onpointerdown={handlePress(closeApp)}>
      <svg width="12" height="12" viewBox="0 0 12 12"><polygon fill="currentColor" points="11 1.5 10.5 1 6 5.5 1.5 1 1 1.5 5.5 6 1 10.5 1.5 11 6 6.5 10.5 11 11 10.5 6.5 6"></polygon></svg>
    </button>
  </div>
</div>