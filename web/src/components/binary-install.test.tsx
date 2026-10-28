import { describe, expect, it } from 'vitest'
import { render, screen } from '../test-utils'
import userEvent from '@testing-library/user-event'
import { BinaryInstall } from './binary-install'
import { computePlatformResult } from '@/lib/platform-detection'
import { ALL_BINARIES, MAC_DOWNLOAD, HARMONY_DOWNLOAD, MANUAL_DOWNLOAD } from '@/lib/platform-catalog'

const winDownload = computePlatformResult('Windows', 'amd64').download
const macX86Download = computePlatformResult('macOS', 'amd64').download

describe('BinaryInstall (HarmonyOS choices)', () => {
  it.each([false, true])('offers both native architectures with mirror=%s', mirror => {
    render(<BinaryInstall selection={HARMONY_DOWNLOAD} mirror={mirror} />)
    const origin = mirror ? 'mirror' : 'official'
    expect(screen.getByRole('link', { name: /HarmonyOS ARM64/ })).toHaveAttribute('href', `/dl/${origin}/openharmony_arm64/cjv-init`)
    expect(screen.getByRole('link', { name: /HarmonyOS x86_64/ })).toHaveAttribute('href', `/dl/${origin}/openharmony_amd64/cjv-init`)
    expect(screen.getAllByRole('link')).toHaveLength(2)
    expect(screen.getByText(/uname -m/)).toBeInTheDocument()
    expect(screen.getByText(/chmod \+x cjv-init/)).toBeInTheDocument()
  })
})

describe('BinaryInstall (binary detected)', () => {
  it('renders the primary download for the detected binary', () => {
    render(<BinaryInstall selection={winDownload} mirror={false} />)

    const link = screen.getByRole('link', { name: /cjv-init\.exe/ })
    expect(link).toHaveAttribute('href', '/dl/official/windows_amd64/cjv-init.exe')
    expect(link).toHaveAttribute('download', 'cjv-init.exe')
    expect(screen.getByText(/检测到你的平台：Windows x86_64/)).toBeInTheDocument()
  })

  it('flips to the mirror URL when mirror=true', () => {
    render(<BinaryInstall selection={winDownload} mirror />)
    expect(screen.getByRole('link', { name: /cjv-init\.exe/ })).toHaveAttribute('href', '/dl/mirror/windows_amd64/cjv-init.exe')
  })

  it('surfaces the macOS x86_64 warning', () => {
    render(<BinaryInstall selection={macX86Download} mirror={false} />)
    expect(screen.getByText(/⚠.*x86_64/)).toBeInTheDocument()
  })

  it('hides the current binary from the "其他平台" list', async () => {
    const user = userEvent.setup()
    render(<BinaryInstall selection={winDownload} mirror={false} />)

    await user.click(screen.getByRole('button', { name: /其他平台/ }))

    const otherLinks = screen.getAllByRole('link', { name: /cjv-init/ })
    const hrefs = otherLinks.map(a => a.getAttribute('href'))
    expect(hrefs.filter(h => h?.includes('windows_amd64'))).toHaveLength(1)
    expect(hrefs.filter(h => h?.includes('darwin_arm64'))).toHaveLength(1)
    expect(hrefs.filter(h => h?.includes('linux_arm64'))).toHaveLength(1)
  })

})

describe('BinaryInstall (macOS choices)', () => {
  it('renders Apple Silicon and Intel download buttons when macOS choices are requested', () => {
    render(<BinaryInstall selection={MAC_DOWNLOAD} mirror={false} />)

    const appleSilicon = screen.getByRole('link', { name: /Apple Silicon/ })
    const intel = screen.getByRole('link', { name: /Intel/ })

    expect(appleSilicon).toHaveAttribute('href', '/dl/official/darwin_arm64/cjv-init')
    expect(appleSilicon).toHaveAttribute('download', 'cjv-init')
    expect(intel).toHaveAttribute('href', '/dl/official/darwin_amd64/cjv-init')
    expect(intel).toHaveAttribute('download', 'cjv-init')
    expect(screen.getByText(/⚠.*x86_64/)).toBeInTheDocument()
  })

  it('uses mirror URLs for both macOS choices when mirror=true', () => {
    render(<BinaryInstall selection={MAC_DOWNLOAD} mirror />)

    expect(screen.getByRole('link', { name: /Apple Silicon/ })).toHaveAttribute('href', '/dl/mirror/darwin_arm64/cjv-init')
    expect(screen.getByRole('link', { name: /Intel/ })).toHaveAttribute('href', '/dl/mirror/darwin_amd64/cjv-init')
  })

  it('omits macOS binaries from the expanded other-platform list', async () => {
    const user = userEvent.setup()
    render(<BinaryInstall selection={MAC_DOWNLOAD} mirror={false} />)

    await user.click(screen.getByRole('button', { name: /其他平台/ }))

    const links = screen.getAllByRole('link', { name: /cjv-init/ })
    const hrefs = links.map(a => a.getAttribute('href'))
    expect(hrefs.filter(h => h?.includes('darwin_'))).toHaveLength(2)
    expect(hrefs.filter(h => h?.includes('linux_'))).toHaveLength(2)
    expect(hrefs.filter(h => h?.includes('windows_'))).toHaveLength(1)
  })
})

describe('BinaryInstall (no binary)', () => {
  it('shows the manual selection prompt', () => {
    render(<BinaryInstall selection={MANUAL_DOWNLOAD} mirror={false} />)
    expect(screen.getByText(/请手动选择对应平台的二进制/)).toBeInTheDocument()
    expect(screen.getAllByRole('link', { name: /cjv-init/ })).toHaveLength(ALL_BINARIES.length)
  })

  it('uses the GitCode releases link when mirror=true', () => {
    render(<BinaryInstall selection={MANUAL_DOWNLOAD} mirror />)
    const release = screen.getByRole('link', { name: /GitCode Releases/ })
    expect(release).toHaveAttribute('href', 'https://gitcode.com/Zxilly/cjv/releases')
  })

  it('uses the GitHub releases link when mirror=false', () => {
    render(<BinaryInstall selection={MANUAL_DOWNLOAD} mirror={false} />)
    const release = screen.getByRole('link', { name: /GitHub Releases/ })
    expect(release).toHaveAttribute('href', 'https://github.com/Zxilly/cjv/releases')
  })
})
