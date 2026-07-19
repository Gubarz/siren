import './styles/main.css'
import { mount } from 'svelte'
import App from './App.svelte'
import { applyThemePreference } from './lib/stores/ui/theme.svelte.js'

applyThemePreference()

const app = mount(App, {
  target: document.getElementById('app'),
})

export default app
