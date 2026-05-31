import { afterEach, describe, expect, it } from 'vitest'
import { render, screen } from '@testing-library/react'
import { server } from 'vitest/browser'
import App from './App'
import { computePlatformResult, type PlatformResult, type PlatformState } from './hooks/use-platform'

const MAC_DESKTOP_UA =
  'Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.5 Safari/605.1.15'
const LINUX_AMD64_UA =
  'Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36'
const LINUX_ARM64_UA =
  'Mozilla/5.0 (X11; Linux aarch64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36'
const WINDOWS_AMD64_UA =
  'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36'
const WINDOWS_ARM64_UA =
  'Mozilla/5.0 (Windows NT 10.0; ARM64; rv:148.0) Gecko/20100101 Firefox/148.0'

interface NavigatorPatch {
  maxTouchPoints: number
  platform: string
  userAgent: string
  userAgentData?: {
    getHighEntropyValues?: (hints: string[]) => Promise<{
      architecture?: string
      bitness?: string
      platform?: string
    }>
    platform?: string
  } | null
}

interface ExpectedPlatformEnv {
  VITE_EXPECTED_BINARY_GOARCH?: string
  VITE_EXPECTED_BINARY_GOOS?: string
  VITE_EXPECTED_BROWSER?: string
  VITE_EXPECTED_LABEL?: string
  VITE_EXPECTED_PLATFORM_ARCH?: string
  VITE_EXPECTED_PLATFORM_OS?: string
  VITE_EXPECTED_PLAYWRIGHT_BROWSER?: string
  VITE_EXPECTED_RUNNER?: string
  VITE_EXPECTED_RUNNER_ARCH?: string
  VITE_EXPECTED_STATE?: PlatformState
}

const restoreNavigatorFns: Array<() => void> = []

function restoreNavigator() {
  while (restoreNavigatorFns.length > 0) restoreNavigatorFns.pop()?.()
}

function stubNavigator(patch: NavigatorPatch) {
  restoreNavigator()
  const nav = window.navigator as Navigator & Record<string, unknown>
  const keys = Object.keys(patch) as Array<keyof NavigatorPatch>

  for (const key of keys) {
    const original = Object.getOwnPropertyDescriptor(nav, key)
    Object.defineProperty(nav, key, {
      configurable: true,
      get: () => patch[key],
    })
    restoreNavigatorFns.push(() => {
      if (original) Object.defineProperty(nav, key, original)
      else delete nav[key]
    })
  }
}

afterEach(restoreNavigator)

function expectedEnv(): ExpectedPlatformEnv {
  return import.meta.env
}

function assertPlatformResult(result: PlatformResult, env: ExpectedPlatformEnv) {
  expect(result.state).toBe(env.VITE_EXPECTED_STATE)
  expect(result.info.label).toBe(env.VITE_EXPECTED_LABEL)

  if (result.state === 'ready') {
    expect(result.binary.goos).toBe(env.VITE_EXPECTED_BINARY_GOOS)
    expect(result.binary.goarch).toBe(env.VITE_EXPECTED_BINARY_GOARCH)
  } else {
    expect(result.binary).toBeNull()
  }
}

function navigatorPatchForExpectedPlatform(env: ExpectedPlatformEnv): NavigatorPatch {
  const os = env.VITE_EXPECTED_PLATFORM_OS
  const arch = env.VITE_EXPECTED_PLATFORM_ARCH

  if (os === 'Linux' && arch === 'amd64') {
    return { maxTouchPoints: 0, platform: 'Linux x86_64', userAgent: LINUX_AMD64_UA, userAgentData: null }
  }
  if (os === 'Linux' && arch === 'arm64') {
    return { maxTouchPoints: 0, platform: 'Linux aarch64', userAgent: LINUX_ARM64_UA, userAgentData: null }
  }
  if (os === 'Windows' && arch === 'amd64') {
    return { maxTouchPoints: 0, platform: 'Win32', userAgent: WINDOWS_AMD64_UA, userAgentData: null }
  }
  if (os === 'Windows' && arch === 'arm64') {
    return { maxTouchPoints: 0, platform: 'Win32', userAgent: WINDOWS_ARM64_UA, userAgentData: null }
  }
  if (os === 'Mac OS' && arch === '') {
    return { maxTouchPoints: 0, platform: 'MacIntel', userAgent: MAC_DESKTOP_UA, userAgentData: null }
  }

  throw new Error(`unsupported expected CI platform: ${os}/${arch}`)
}

describe('App platform detection integration', () => {
  it('does not render macOS x86_64 when macOS browsers hide the CPU architecture', () => {
    stubNavigator({
      maxTouchPoints: 0,
      platform: 'MacIntel',
      userAgent: MAC_DESKTOP_UA,
      userAgentData: null,
    })

    render(<App />)

    expect(screen.getByText(/无法识别你的平台/)).toBeInTheDocument()
    expect(screen.queryByText(/macOS x86_64/)).not.toBeInTheDocument()
  })

  it('renders iOS unsupported messaging for iPadOS desktop mode', () => {
    stubNavigator({
      maxTouchPoints: 5,
      platform: 'MacIntel',
      userAgent: MAC_DESKTOP_UA,
      userAgentData: null,
    })

    render(<App />)

    expect(screen.getByText(/cjv 暂不支持/)).toBeInTheDocument()
    expect(screen.getByText('iOS')).toBeInTheDocument()
    expect(screen.queryByText(/macOS x86_64/)).not.toBeInTheDocument()
  })

  it('updates to macOS ARM64 when Chromium UA Client Hints expose Apple Silicon', async () => {
    stubNavigator({
      maxTouchPoints: 0,
      platform: 'MacIntel',
      userAgent: MAC_DESKTOP_UA,
      userAgentData: {
        platform: 'macOS',
        getHighEntropyValues: async () => ({ architecture: 'arm', bitness: '64', platform: 'macOS' }),
      },
    })

    render(<App />)

    expect(await screen.findByText(/检测到你的平台：macOS ARM64/)).toBeInTheDocument()
    expect(screen.queryByText(/macOS x86_64/)).not.toBeInTheDocument()
  })

  it('asserts the configured CI runner browser and platform behavior', () => {
    const env = expectedEnv()
    if (!env.VITE_EXPECTED_RUNNER) return

    expect(env.VITE_EXPECTED_BROWSER).toMatch(/^(chromium|firefox|webkit)$/)
    expect(server.browser).toBe(env.VITE_EXPECTED_PLAYWRIGHT_BROWSER)
    expect(env.VITE_EXPECTED_RUNNER_ARCH).toMatch(/^(amd64|arm64)$/)

    const platformResult = computePlatformResult(
      env.VITE_EXPECTED_PLATFORM_OS || '',
      env.VITE_EXPECTED_PLATFORM_ARCH || '',
    )
    assertPlatformResult(platformResult, env)

    stubNavigator(navigatorPatchForExpectedPlatform(env))
    render(<App />)

    if (env.VITE_EXPECTED_STATE === 'ready') {
      expect(screen.getByText(`检测到你的平台：${env.VITE_EXPECTED_LABEL}`)).toBeInTheDocument()
    } else if (env.VITE_EXPECTED_STATE === 'unsupported') {
      expect(screen.getByText(/cjv 暂不支持/)).toBeInTheDocument()
      expect(screen.getByText(env.VITE_EXPECTED_LABEL || '')).toBeInTheDocument()
    } else {
      expect(screen.getByText(/无法识别你的平台/)).toBeInTheDocument()
    }
  })
})
