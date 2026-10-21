import { act, renderHook, waitFor } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'
import { useMirror } from './use-mirror'

describe('useMirror', () => {
  it.each(['US', 'HK', 'TW', ''])('keeps the default source for country %s', async country => {
    const json = vi.fn().mockResolvedValue({ country })
    vi.mocked(fetch).mockResolvedValue({ ok: true, json } as unknown as Response)
    const { result } = renderHook(() => useMirror())
    await waitFor(() => expect(json).toHaveBeenCalled())
    expect(result.current[0]).toBe(false)
  })

  it('lets CN visitors turn the mirror off after detection', async () => {
    vi.mocked(fetch).mockResolvedValue(Response.json({ country: 'CN' }))
    const { result } = renderHook(() => useMirror())
    await waitFor(() => expect(result.current[0]).toBe(true))
    act(() => result.current[1](false))
    expect(result.current[0]).toBe(false)
  })

  it('preserves a manual selection made before detection completes', async () => {
    let resolve!: (response: Response) => void
    vi.mocked(fetch).mockReturnValue(new Promise<Response>(r => { resolve = r }))
    const { result } = renderHook(() => useMirror())
    act(() => {
      result.current[1](true)
      result.current[1](false)
    })
    await act(async () => { resolve(Response.json({ country: 'CN' })) })
    expect(result.current[0]).toBe(false)
  })

  it.each([
    () => Promise.reject(new TypeError('Network unavailable')),
    () => Promise.resolve(new Response('', { status: 503 })),
    () => Promise.resolve(new Response('invalid JSON')),
    () => Promise.resolve(Response.json(null)),
  ])('keeps the page usable when detection fails (%#)', async respond => {
    vi.mocked(fetch).mockImplementation(respond)
    const { result } = renderHook(() => useMirror())
    await act(async () => {})
    expect(result.current[0]).toBe(false)
    act(() => result.current[1](true))
    expect(result.current[0]).toBe(true)
  })

  it('aborts slow detection after three seconds and ignores a late result', async () => {
    vi.useFakeTimers()
    try {
      let resolve!: (response: Response) => void
      vi.mocked(fetch).mockReturnValue(new Promise<Response>(r => { resolve = r }))
      const { result } = renderHook(() => useMirror())
      const signal = vi.mocked(fetch).mock.calls[0][1]?.signal
      await act(async () => { vi.advanceTimersByTime(3000) })
      expect(signal?.aborted).toBe(true)
      await act(async () => { resolve(Response.json({ country: 'CN' })) })
      expect(result.current[0]).toBe(false)
    } finally {
      vi.useRealTimers()
    }
  })

  it('aborts detection on unmount', () => {
    vi.mocked(fetch).mockReturnValue(new Promise(() => {}))
    const { unmount } = renderHook(() => useMirror())
    const signal = vi.mocked(fetch).mock.calls[0][1]?.signal
    unmount()
    expect(signal?.aborted).toBe(true)
  })
})
