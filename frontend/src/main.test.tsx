import React from 'react'
import { beforeEach, describe, expect, it, vi } from 'vitest'

const { createRoot, render } = vi.hoisted(() => ({
  createRoot: vi.fn(),
  render: vi.fn(),
}))

vi.mock('react-dom/client', () => ({
  default: { createRoot: (...args: unknown[]) => createRoot(...args) },
}))

describe('앱 시작점', () => {
  beforeEach(() => {
    vi.resetModules()
    createRoot.mockReturnValue({ render })
    createRoot.mockClear()
    render.mockClear()
    document.body.innerHTML = '<div id="root"></div>'
  })

  it('root 요소에 StrictMode로 앱을 렌더링한다', async () => {
    await import('./main')

    expect(createRoot).toHaveBeenCalledWith(document.getElementById('root'))
    expect(render).toHaveBeenCalledOnce()
    expect(render.mock.calls[0][0].type).toBe(React.StrictMode)
  })
})
