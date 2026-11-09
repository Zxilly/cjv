import { useEffect, useState } from 'react'

export function useMirror() {
  const state = useState(() => typeof document !== 'undefined'
    && document.documentElement.dataset.prerendered !== 'true'
    && document.documentElement.dataset.defaultMirror === 'true')
  const [, setMirror] = state
  useEffect(() => {
    setMirror(document.documentElement.dataset.defaultMirror === 'true')
  }, [setMirror])
  return state
}
