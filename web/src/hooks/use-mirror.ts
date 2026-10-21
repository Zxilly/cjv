import { useState } from 'react'

export function useMirror() {
  return useState(() => document.documentElement.dataset.defaultMirror === 'true')
}
