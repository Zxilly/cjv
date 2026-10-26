import { afterEach, describe, expect, it } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import App from './App'
import { HARMONY_SAMPLES } from './test-fixtures/harmony'
import { activateLang, type Lang } from './lib/i18n'

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
const IPHONE_UA =
  'Mozilla/5.0 (iPhone; CPU iPhone OS 17_5 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.5 Mobile/15E148 Safari/604.1'
const ANDROID_UA =
  'Mozilla/5.0 (Linux; Android 14; Pixel 8) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Mobile Safari/537.36'

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

describe('App platform detection integration', () => {
  it('keeps macOS command install ready without guessing x86_64 when browsers hide the CPU architecture', () => {
    stubNavigator({
      maxTouchPoints: 0,
      platform: 'MacIntel',
      userAgent: MAC_DESKTOP_UA,
      userAgentData: null,
    })

    render(<App />)

    expect(screen.getByText('检测到你的平台：macOS')).toBeInTheDocument()
    expect(screen.queryByText(/无法识别你的平台/)).not.toBeInTheDocument()
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

  it('renders iOS unsupported messaging for an iPhone browser', () => {
    stubNavigator({
      maxTouchPoints: 5,
      platform: 'iPhone',
      userAgent: IPHONE_UA,
      userAgentData: null,
    })

    render(<App />)

    expect(screen.getByText(/cjv 暂不支持/)).toBeInTheDocument()
    expect(screen.getByText('iOS')).toBeInTheDocument()
    expect(screen.queryByRole('tablist')).not.toBeInTheDocument()
  })

  it('renders Android unsupported messaging for an Android phone browser', () => {
    stubNavigator({
      maxTouchPoints: 5,
      platform: 'Linux armv8l',
      userAgent: ANDROID_UA,
      userAgentData: null,
    })

    render(<App />)

    expect(screen.getByText(/cjv 暂不支持/)).toBeInTheDocument()
    expect(screen.getByText('Android')).toBeInTheDocument()
    expect(screen.queryByRole('tablist')).not.toBeInTheDocument()
  })

  describe.each<Lang>(['zh', 'en'])('HarmonyOS guidance (%s)', lang => {
    it.each(HARMONY_SAMPLES)('renders version and device guidance for $name', async sample => {
      stubNavigator({
        maxTouchPoints: 5,
        platform: 'Win32',
        userAgent: sample.userAgent,
        userAgentData: null,
      })
      // Test platform copy independently of the page's animated language switch.
      // App.test.tsx covers switching languages and expanding other platforms.
      activateLang(lang)
      render(<App />)

      if (sample.device === 'convertible-tablet') {
        expect(screen.getByRole('tablist')).toBeInTheDocument()
        await userEvent.click(screen.getByRole('tab', { name: lang === 'zh' ? '下载安装' : 'Download' }))
        expect(await screen.findByRole('link', { name: /HarmonyOS ARM64/ })).toHaveAttribute('href', '/dl/official/openharmony_arm64/cjv-init')
        expect(screen.getByRole('link', { name: /HarmonyOS x86_64/ })).toHaveAttribute('href', '/dl/official/openharmony_amd64/cjv-init')
        expect(screen.queryByText(/cjv 暂不支持/)).not.toBeInTheDocument()
        return
      }
      expect(screen.getByText(`HarmonyOS (OpenHarmony ${sample.version})`)).toBeInTheDocument()
      expect(screen.getByText(lang === 'zh' ? sample.message : sample.englishMessage)).toBeInTheDocument()
      expect(screen.queryByText(/已列入计划|support.*planned/)).not.toBeInTheDocument()
      expect(screen.queryByText(/请在桌面设备上访问|Please open this page on a desktop/)).not.toBeInTheDocument()
      expect(screen.queryByRole('tablist')).not.toBeInTheDocument()
      expect(screen.queryByText(/install\.(ps1|sh)/)).not.toBeInTheDocument()
      expect(screen.getByRole('button', {
        name: lang === 'zh' ? '查看其他平台的安装方式' : 'Show install methods for other platforms',
      })).toBeInTheDocument()
    })
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

  it.each([
    { label: 'Linux x86_64', ua: LINUX_AMD64_UA, platform: 'Linux x86_64', supported: true },
    { label: 'Linux ARM64', ua: LINUX_ARM64_UA, platform: 'Linux aarch64', supported: true },
    { label: 'Windows x86_64', ua: WINDOWS_AMD64_UA, platform: 'Win32', supported: true },
    { label: 'Windows arm64', ua: WINDOWS_ARM64_UA, platform: 'Win32', supported: false },
  ])('renders $label from the browser platform', ({ label, ua, platform, supported }) => {
    stubNavigator({ maxTouchPoints: 0, platform, userAgent: ua, userAgentData: null })
    render(<App />)
    if (supported) {
      expect(screen.getByText(`检测到你的平台：${label}`)).toBeInTheDocument()
    } else {
      expect(screen.getByText(/cjv 暂不支持/)).toBeInTheDocument()
      expect(screen.getByText(label)).toBeInTheDocument()
    }
  })
})
