import './styles/main.css'
import { mount } from 'svelte'
import App from './App.svelte'
import DetachedAgentWindow from './DetachedAgentWindow.svelte'
import { applyThemePreference } from './lib/stores/ui/theme.svelte.js'

applyThemePreference()

const detachedAgentTab = new URLSearchParams(window.location.search).get('detachedAgentTab')
const Component = detachedAgentTab ? DetachedAgentWindow : App
const props = detachedAgentTab ? { token: detachedAgentTab } : {}

const app = mount(Component, {
  target: document.getElementById('app'),
  props,
})

export default app
