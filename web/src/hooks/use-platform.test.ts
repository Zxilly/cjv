import { describe, expect, it } from 'vitest'
import { renderHook, waitFor } from '@testing-library/react'
import { HARMONY_SAMPLES } from '../test-fixtures/harmony'
import { usePlatform } from './use-platform'

const MAC_DESKTOP_UA =
  'Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.5 Safari/605.1.15'
const WINDOWS_DESKTOP_UA =
  'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36'

describe('usePlatform', () => {
  it('updates the device hint when a HarmonyOS device changes mode at the same version', async () => {
    const userAgent = HARMONY_SAMPLES[1].userAgent
    const { result, rerender } = renderHook(({ ua }) => usePlatform({ userAgent: ua }), {
      initialProps: { ua: userAgent as string },
    })
    rerender({ ua: userAgent.replace('; Windows NT 10.0; Win64; x64', '') })
    await waitFor(() => {
      if (result.current.state !== 'unsupported') throw new Error('expected unsupported')
      expect(result.current.info.harmony?.device).toBe('tablet')
    })
  })

  it('exposes the same shape as computePlatformResult', () => {
    const input = {
      maxTouchPoints: 0,
      platform: 'Win32',
      userAgent: WINDOWS_DESKTOP_UA,
    }
    const { result } = renderHook(() => usePlatform(input))

    const r = result.current
    expect(r).toMatchObject({
      state: expect.any(String),
      info: expect.any(Object),
      methods: expect.any(Array),
      otherMethods: expect.any(Array),
      sourceMethod: expect.any(Object),
      download: expect.objectContaining({ kind: 'detected' }),
    })
  })

  it('updates when asynchronous UA Client Hints refine a macOS architecture', async () => {
    const input = {
      maxTouchPoints: 0,
      platform: 'MacIntel',
      userAgent: MAC_DESKTOP_UA,
      userAgentData: {
        platform: 'macOS',
        getHighEntropyValues: async () => ({ architecture: 'arm', bitness: '64', platform: 'macOS' }),
      },
    }
    const { result } = renderHook(() => usePlatform(input))

    expect(result.current.state).toBe('ready')
    expect(result.current.info.label).toBe('macOS')
    expect(result.current.download).toMatchObject({ kind: 'choose-architecture', goos: 'darwin' })
    await waitFor(() => expect(result.current.info.label).toBe('macOS ARM64'))
    expect(result.current.state).toBe('ready')
    expect(result.current.download).toMatchObject({ kind: 'detected', binary: { goarch: 'arm64' } })
  })
})
