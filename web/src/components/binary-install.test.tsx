import { describe, expect, it, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { BinaryInstall } from './binary-install'
import { computePlatformResult } from '@/hooks/use-platform'

const { allBinaries } = computePlatformResult('Windows', 'amd64')
const winBinary = allBinaries.find(b => b.goos === 'windows')!
const macX86Binary = allBinaries.find(b => b.goos === 'darwin' && b.goarch === 'amd64')!

describe('BinaryInstall (binary detected)', () => {
  it('renders the primary download for the detected binary', () => {
    render(<BinaryInstall binary={winBinary} allBinaries={allBinaries} mirror={false} />)

    const link = screen.getByRole('link', { name: /cjv-init\.exe/ })
    expect(link).toHaveAttribute('href', '/dl/official/windows_amd64/cjv-init.exe')
    expect(link).toHaveAttribute('download', 'cjv-init.exe')
    expect(screen.getByText(/检测到你的平台：Windows x86_64/)).toBeInTheDocument()
  })

  it('flips to the mirror URL when mirror=true', () => {
    render(<BinaryInstall binary={winBinary} allBinaries={allBinaries} mirror />)
    expect(screen.getByRole('link', { name: /cjv-init\.exe/ })).toHaveAttribute('href', '/dl/mirror/windows_amd64/cjv-init.exe')
  })

  it('surfaces the macOS x86_64 warning', () => {
    render(<BinaryInstall binary={macX86Binary} allBinaries={allBinaries} mirror={false} />)
    expect(screen.getByText(/⚠.*x86_64/)).toBeInTheDocument()
  })

  it('hides the current binary from the "其他平台" list', async () => {
    const user = userEvent.setup()
    render(<BinaryInstall binary={winBinary} allBinaries={allBinaries} mirror={false} />)

    await user.click(screen.getByRole('button', { name: /其他平台/ }))

    const otherLinks = screen.getAllByRole('link', { name: /cjv-init/ })
    const hrefs = otherLinks.map(a => a.getAttribute('href'))
    expect(hrefs.filter(h => h?.includes('windows_amd64'))).toHaveLength(1)
    expect(hrefs.filter(h => h?.includes('darwin_arm64'))).toHaveLength(1)
    expect(hrefs.filter(h => h?.includes('linux_arm64'))).toHaveLength(1)
  })

  it('invokes the command-install callback when clicked', async () => {
    const user = userEvent.setup()
    const onSwitch = vi.fn()
    render(<BinaryInstall binary={winBinary} allBinaries={allBinaries} mirror={false} onUseCommandInstall={onSwitch} />)

    await user.click(screen.getByRole('button', { name: /命令安装/ }))
    expect(onSwitch).toHaveBeenCalledOnce()
  })
})

describe('BinaryInstall (no binary)', () => {
  it('shows the manual selection prompt', () => {
    render(<BinaryInstall binary={null} allBinaries={allBinaries} mirror={false} />)
    expect(screen.getByText(/请手动选择对应平台的二进制/)).toBeInTheDocument()
    expect(screen.getAllByRole('link', { name: /cjv-init/ })).toHaveLength(allBinaries.length)
  })

  it('falls back to the command-install hint button', async () => {
    const user = userEvent.setup()
    const onSwitch = vi.fn()
    render(<BinaryInstall binary={null} allBinaries={allBinaries} mirror={false} onUseCommandInstall={onSwitch} />)

    await user.click(screen.getByRole('button', { name: /切换到命令安装/ }))
    expect(onSwitch).toHaveBeenCalledOnce()
  })

  it('uses the GitCode releases link when mirror=true', () => {
    render(<BinaryInstall binary={null} allBinaries={allBinaries} mirror />)
    const release = screen.getByRole('link', { name: /GitCode Releases/ })
    expect(release).toHaveAttribute('href', 'https://gitcode.com/Zxilly/cjv/releases')
  })

  it('uses the GitHub releases link when mirror=false', () => {
    render(<BinaryInstall binary={null} allBinaries={allBinaries} mirror={false} />)
    const release = screen.getByRole('link', { name: /GitHub Releases/ })
    expect(release).toHaveAttribute('href', 'https://github.com/Zxilly/cjv/releases')
  })
})
