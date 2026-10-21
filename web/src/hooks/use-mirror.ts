import { useEffect, useState } from 'react'

export function useMirror() {
  const [detectedMirror, setDetectedMirror] = useState(false)
  const [selection, setSelection] = useState<boolean | null>(null)

  useEffect(() => {
    const controller = new AbortController()
    const timeout = window.setTimeout(() => controller.abort(), 3000)

    async function detectCountry() {
      try {
        const response = await fetch('https://get.geojs.io/v1/ip/country.json', {
          signal: controller.signal,
          credentials: 'omit',
          referrerPolicy: 'no-referrer',
        })
        if (!response.ok) return
        const data: unknown = await response.json()
        if (!controller.signal.aborted && data && typeof data === 'object' && 'country' in data) {
          setDetectedMirror(data.country === 'CN')
        }
      } catch {
        // Keep the default source when geolocation is unavailable.
      } finally {
        window.clearTimeout(timeout)
      }
    }

    void detectCountry()
    return () => {
      controller.abort()
      window.clearTimeout(timeout)
    }
  }, [])

  // A manual choice takes precedence even if the lookup finishes later.
  return [selection ?? detectedMirror, setSelection] as const
}
