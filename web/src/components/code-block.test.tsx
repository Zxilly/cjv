import { afterEach, describe, expect, it, vi } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { CodeBlock } from './code-block'

const COPY_BUTTON = { name: '复制命令' }

describe('CodeBlock', () => {
  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('renders the command text', () => {
    render(<CodeBlock command="curl -sSf http://x | sh" />)
    expect(screen.getByText('curl -sSf http://x | sh')).toBeInTheDocument()
  })

  it('exposes an accessible label on the copy button', () => {
    render(<CodeBlock command="echo hello" />)
    expect(screen.getByRole('button', COPY_BUTTON)).toBeInTheDocument()
  })

  it('shows the success state then reverts to the idle icon', async () => {
    const user = userEvent.setup()
    const { container } = render(<CodeBlock command="echo hello" />)

    expect(container.querySelector('.text-cj')).toBeNull()
    await user.click(screen.getByRole('button', COPY_BUTTON))

    await waitFor(() => expect(container.querySelector('.text-cj')).not.toBeNull())
    await waitFor(() => expect(screen.getByText('已复制')).toBeInTheDocument())
    await waitFor(() => expect(container.querySelector('.text-cj')).toBeNull(), { timeout: 2500 })
  })

  it('falls back to execCommand when the async clipboard write rejects', async () => {
    const user = userEvent.setup()
    const writeText = vi.fn().mockRejectedValue(new Error('blocked'))
    vi.spyOn(navigator, 'clipboard', 'get').mockReturnValue({ writeText } as unknown as Clipboard)
    const execCommand = vi.spyOn(document, 'execCommand').mockReturnValue(true)

    const { container } = render(<CodeBlock command="echo fallback" />)
    await user.click(screen.getByRole('button', COPY_BUTTON))

    await waitFor(() => expect(execCommand).toHaveBeenCalledWith('copy'))
    expect(writeText).toHaveBeenCalledWith('echo fallback')
    // Success via the legacy path still shows the copied state.
    await waitFor(() => expect(container.querySelector('.text-cj')).not.toBeNull())
    expect(screen.getByText('已复制')).toBeInTheDocument()
  })

  it('shows the failure state and announces it when every copy path fails', async () => {
    const user = userEvent.setup()
    const writeText = vi.fn().mockRejectedValue(new Error('blocked'))
    vi.spyOn(navigator, 'clipboard', 'get').mockReturnValue({ writeText } as unknown as Clipboard)
    vi.spyOn(document, 'execCommand').mockReturnValue(false)

    const { container } = render(<CodeBlock command="echo nope" />)
    await user.click(screen.getByRole('button', COPY_BUTTON))

    await waitFor(() => expect(screen.getByText('复制失败，请手动复制')).toBeInTheDocument())
    expect(container.querySelector('.text-cj')).toBeNull()
    await waitFor(() => expect(container.querySelector('.text-red-500')).not.toBeNull())
  })

  it('does not throw when navigator.clipboard is unavailable (non-secure context)', async () => {
    const user = userEvent.setup()
    vi.spyOn(navigator, 'clipboard', 'get').mockReturnValue(undefined as unknown as Clipboard)
    const execCommand = vi.spyOn(document, 'execCommand').mockReturnValue(true)

    render(<CodeBlock command="echo insecure" />)
    await user.click(screen.getByRole('button', COPY_BUTTON))

    await waitFor(() => expect(execCommand).toHaveBeenCalledWith('copy'))
    await waitFor(() => expect(screen.getByText('已复制')).toBeInTheDocument())
  })

  it('triggers a fresh command swap when the prop changes', async () => {
    const { rerender } = render(<CodeBlock command="first" />)
    expect(screen.getByText('first')).toBeInTheDocument()
    rerender(<CodeBlock command="second" />)
    expect(await screen.findByText('second')).toBeInTheDocument()
  })

  it('clears the copied state when the command changes', async () => {
    const user = userEvent.setup()
    const { container, rerender } = render(<CodeBlock command="official" />)

    await user.click(screen.getByRole('button', COPY_BUTTON))
    await waitFor(() => expect(container.querySelector('.text-cj')).not.toBeNull())

    rerender(<CodeBlock command="mirror" />)

    expect(await screen.findByText('mirror')).toBeInTheDocument()
    await waitFor(() => expect(container.querySelector('.text-cj')).toBeNull())
  })
})
