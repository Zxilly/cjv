import { useEffect, useState } from 'react'
import {
  computeBrowserPlatformResult, computePlatformResult, detectBrowserPlatformResult,
  type BrowserPlatformInput, type PlatformResult,
} from '@/lib/platform-detection'

// Pre-rendering cannot identify a visitor. Keep that state distinct from a
// completed detection that really found an unknown platform.
const DETECTING_PLATFORM = { ...computePlatformResult('', ''), state: 'detecting' as const }
type PlatformDetectionResult = PlatformResult | typeof DETECTING_PLATFORM

// Two results are equivalent for rendering purposes when their user-visible
// fields match. Comparing these lets usePlatform skip a redundant re-render
// when the async detection produces a structurally identical (but freshly
// allocated) result — e.g. on browsers that expose no UA Client Hints and so
// cannot refine the initial guess.
function samePlatformResult(a: PlatformDetectionResult, b: PlatformResult): boolean {
  return (
    a.state === b.state
    && a.info.label === b.info.label
    && a.info.reason === b.info.reason
    && a.info.harmony?.device === b.info.harmony?.device
    && a.download === b.download
  )
}

export function usePlatform(input?: BrowserPlatformInput) {
  const [platform, setPlatform] = useState<PlatformDetectionResult>(() => !input && (
    typeof document === 'undefined' || document.documentElement.dataset.prerendered === 'true'
  ) ? DETECTING_PLATFORM : computeBrowserPlatformResult(input))

  // Depend on the primitive fields rather than the input object's identity, so
  // a caller passing a fresh object literal each render does not re-trigger the
  // async detection on every render.
  const { maxTouchPoints, platform: navPlatform, userAgent, userAgentData } = input ?? {}

  useEffect(() => {
    let active = true
    // Show the synchronous UA result without waiting for Client Hints, which
    // may be slow or never settle. High-entropy hints can refine it afterwards.
    const initial = computeBrowserPlatformResult(input)
    setPlatform(prev => (samePlatformResult(prev, initial) ? prev : initial))
    detectBrowserPlatformResult(input).then(next => {
      if (!active) return
      setPlatform(prev => (samePlatformResult(prev, next) ? prev : next))
    })
    return () => {
      active = false
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps -- input is consumed via its primitive fields above
  }, [maxTouchPoints, navPlatform, userAgent, userAgentData])

  return platform
}
