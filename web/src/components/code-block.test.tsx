import { describe, expect, it } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { CodeBlock } from './code-block'

describe('CodeBlock', () => {
  it('renders the command text', () => {
    render(<CodeBlock command="curl -sSf http://x | sh" />)
    expect(screen.getByText('curl -sSf http://x | sh')).toBeInTheDocument()
  })

  it('shows the success state then reverts to the idle icon', async () => {
    const user = userEvent.setup()
    const { container } = render(<CodeBlock command="echo hello" />)

    expect(container.querySelector('.text-cj')).toBeNull()
    await user.click(screen.getByRole('button', { name: '复制' }))

    await waitFor(() => expect(container.querySelector('.text-cj')).not.toBeNull())
    await waitFor(() => expect(container.querySelector('.text-cj')).toBeNull(), { timeout: 2500 })
  })

  it('triggers a fresh command swap when the prop changes', async () => {
    const { rerender } = render(<CodeBlock command="first" />)
    expect(screen.getByText('first')).toBeInTheDocument()
    rerender(<CodeBlock command="second" />)
    expect(await screen.findByText('second')).toBeInTheDocument()
  })
})
