import { beforeEach, describe, expect, it, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import App from './App'
import { computePlatformResult } from '@/hooks/use-platform'

vi.mock('@/hooks/use-platform', async (importOriginal) => {
  const real = await importOriginal<typeof import('@/hooks/use-platform')>()
  return { ...real, usePlatform: vi.fn(() => real.computePlatformResult('Windows', 'amd64')) }
})

const platformModule = await import('@/hooks/use-platform')
const usePlatformMock = vi.mocked(platformModule.usePlatform)

function setPlatform(os: string, arch: string): void {
  usePlatformMock.mockReturnValue(computePlatformResult(os, arch))
}

describe('App (ready / Windows)', () => {
  beforeEach(() => setPlatform('Windows', 'amd64'))

  it('shows the detected platform and primary command on mount', () => {
    render(<App />)
    expect(screen.getByText(/检测到你的平台：Windows x86_64/)).toBeInTheDocument()
    expect(screen.getByText(/install\.ps1/)).toBeInTheDocument()
  })

  it('switches to the download tab and shows the primary binary', async () => {
    const user = userEvent.setup()
    render(<App />)
    await user.click(screen.getByRole('tab', { name: '下载安装' }))
    expect(screen.getByRole('link', { name: /cjv-init\.exe/ })).toBeInTheDocument()
  })

  it('switches to the source tab and shows the go install command', async () => {
    const user = userEvent.setup()
    render(<App />)
    await user.click(screen.getByRole('tab', { name: '编译安装' }))
    expect(screen.getByText(/^go install /)).toBeInTheDocument()
  })

  it('toggling the mirror switch flips the primary command to the mirror variant', async () => {
    const user = userEvent.setup()
    render(<App />)
    expect(screen.getByText('GitHub · 默认源')).toBeInTheDocument()
    await user.click(screen.getByRole('switch'))
    expect(await screen.findByText(/CJV_MIRROR/)).toBeInTheDocument()
    expect(await screen.findByText('GitCode · 镜像源')).toBeInTheDocument()
  })
})

describe('App (unsupported / iOS)', () => {
  beforeEach(() => setPlatform('iOS', 'arm64'))

  it('renders a single unsupported card with the OS name', () => {
    render(<App />)
    expect(screen.getByText(/cjv 暂不支持/)).toBeInTheDocument()
    expect(screen.getByText('iOS')).toBeInTheDocument()
    expect(screen.queryByRole('tablist')).not.toBeInTheDocument()
  })

  it('expands the install card when the user opens "查看其他平台的安装方式"', async () => {
    const user = userEvent.setup()
    render(<App />)

    expect(screen.queryByRole('tablist')).not.toBeInTheDocument()
    await user.click(screen.getByRole('button', { name: /查看其他平台的安装方式/ }))

    expect(screen.getByRole('tablist')).toBeInTheDocument()
    expect(screen.getByRole('tab', { name: '命令安装' })).toBeInTheDocument()
  })
})

describe('App (unknown OS)', () => {
  beforeEach(() => setPlatform('FreeBSD', 'amd64'))

  it('shows the unrecognized-platform hint in the command tab', () => {
    render(<App />)
    expect(screen.getByText(/无法识别你的平台/)).toBeInTheDocument()
  })

  it('shows the manual binary list in the download tab', async () => {
    const user = userEvent.setup()
    render(<App />)
    await user.click(screen.getByRole('tab', { name: '下载安装' }))
    expect(screen.getByText(/请手动选择对应平台的二进制/)).toBeInTheDocument()
  })
})

describe('App (macOS browser with hidden architecture)', () => {
  beforeEach(() => setPlatform('Mac OS', ''))

  it('keeps command install ready instead of showing the unrecognized-platform fallback', () => {
    render(<App />)
    expect(screen.getByText('检测到你的平台：macOS')).toBeInTheDocument()
    expect(screen.queryByText(/无法识别你的平台/)).not.toBeInTheDocument()
  })

  it('shows explicit Apple Silicon and Intel downloads in the download tab', async () => {
    const user = userEvent.setup()
    render(<App />)

    await user.click(screen.getByRole('tab', { name: '下载安装' }))

    expect(screen.getByRole('link', { name: /Apple Silicon/ })).toBeInTheDocument()
    expect(screen.getByRole('link', { name: /Intel/ })).toBeInTheDocument()
    expect(screen.queryByText(/请手动选择对应平台的二进制/)).not.toBeInTheDocument()
  })
})

describe('App (macOS ARM64 detected)', () => {
  beforeEach(() => setPlatform('macOS', 'arm64'))

  it('shows the single detected binary, not the chip chooser, once the arch is known', async () => {
    const user = userEvent.setup()
    render(<App />)

    await user.click(screen.getByRole('tab', { name: '下载安装' }))

    // The single-binary view shows this copy; the chip chooser says "选择对应 Mac 芯片下载".
    expect(screen.getByText(/下载并运行/)).toBeInTheDocument()
    expect(screen.queryByText(/选择对应 Mac 芯片下载/)).not.toBeInTheDocument()
    expect(screen.queryByRole('link', { name: /Apple Silicon/ })).not.toBeInTheDocument()
    expect(screen.queryByRole('link', { name: /Intel/ })).not.toBeInTheDocument()
  })
})
