import { describe, expect, it } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { CollapsibleSection } from './collapsible-section'

describe('CollapsibleSection', () => {
  it('starts collapsed by default', () => {
    render(
      <CollapsibleSection title="More">
        <p>hidden body</p>
      </CollapsibleSection>,
    )
    expect(screen.getByRole('button', { name: /More/ })).toHaveAttribute('aria-expanded', 'false')
  })

  it('honors initial=true to start expanded', () => {
    render(
      <CollapsibleSection title="More" initial>
        <p>visible body</p>
      </CollapsibleSection>,
    )
    expect(screen.getByRole('button', { name: /More/ })).toHaveAttribute('aria-expanded', 'true')
    expect(screen.getByText('visible body')).toBeInTheDocument()
  })

  it('toggles on trigger click', async () => {
    const user = userEvent.setup()
    render(
      <CollapsibleSection title="More">
        <p>body</p>
      </CollapsibleSection>,
    )
    const trigger = screen.getByRole('button', { name: /More/ })
    expect(trigger).toHaveAttribute('aria-expanded', 'false')
    await user.click(trigger)
    expect(trigger).toHaveAttribute('aria-expanded', 'true')
  })
})
