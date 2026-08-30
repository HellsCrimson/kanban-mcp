import { useEffect, useRef } from 'react'

/**
 * Subscribes to the server's SSE change stream and calls onChange whenever the
 * data this client cares about was modified — by this browser, another tab, or
 * an MCP client.
 *
 * projectId scopes the stream to one project; pass null to receive every event.
 * paused lets the caller hold refreshes back (during a drag, for instance); a
 * change that arrives while paused is replayed as soon as it is unpaused.
 * enabled: false opens no connection at all.
 */
export function useLiveUpdates(projectId, onChange, { paused = false, enabled = true } = {}) {
  const handler = useRef(onChange)
  handler.current = onChange

  const pausedRef = useRef(paused)
  const pendingRef = useRef(false)

  useEffect(() => {
    const wasPaused = pausedRef.current
    pausedRef.current = paused
    if (wasPaused && !paused && pendingRef.current) {
      pendingRef.current = false
      handler.current()
    }
  }, [paused])

  useEffect(() => {
    if (!enabled) return
    const url = projectId ? `/api/events?project_id=${encodeURIComponent(projectId)}` : '/api/events'
    const source = new EventSource(url)
    let timer = null

    // Coalesce bursts: a single user action can fan out into several events.
    const fire = () => {
      if (pausedRef.current) {
        pendingRef.current = true
        return
      }
      clearTimeout(timer)
      timer = setTimeout(() => handler.current(), 80)
    }

    source.addEventListener('board', fire)
    source.addEventListener('projects', fire)

    return () => {
      clearTimeout(timer)
      source.close()
    }
  }, [projectId, enabled])
}
