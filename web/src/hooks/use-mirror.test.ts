import { act, renderHook } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { useMirror } from './use-mirror'

describe('useMirror', () => {
  it.each(['true', 'false', undefined, 'invalid'])('reads the injected default %s on the first render', value => {
    if (value !== undefined) document.documentElement.dataset.defaultMirror = value
    const { result } = renderHook(() => useMirror())
    expect(result.current[0]).toBe(value === 'true')
  })

  it('lets visitors override the injected mirror default', () => {
    document.documentElement.dataset.defaultMirror = 'true'
    const { result } = renderHook(() => useMirror())
    act(() => result.current[1](false))
    expect(result.current[0]).toBe(false)
    act(() => result.current[1](true))
    expect(result.current[0]).toBe(true)
  })
})
