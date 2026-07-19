import { LogClient } from '../api/operatorControls.js'
import { errorMessage } from './errors.js'

function logClient(message, context = 'ui') {
  const line = `[${context}] ${String(message || '').slice(0, 4000)}`
  void LogClient(line).catch(() => {})
}

export function installClientLogHandlers() {
  if (typeof window === 'undefined') return () => {}

  const handleError = (event) => {
    const location = [event.filename, event.lineno, event.colno].filter(Boolean).join(':')
    logClient(`${event.message || 'Uncaught error'}${location ? ` (${location})` : ''}`, 'window-error')
  }
  const handleRejection = (event) => {
    logClient(errorMessage(event.reason), 'unhandled-rejection')
  }

  window.addEventListener('error', handleError)
  window.addEventListener('unhandledrejection', handleRejection)
  return () => {
    window.removeEventListener('error', handleError)
    window.removeEventListener('unhandledrejection', handleRejection)
  }
}
