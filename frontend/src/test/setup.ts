import '@testing-library/jest-dom/vitest'
import { cleanup } from '@testing-library/react'
import { afterEach } from 'vitest'

afterEach(() => {
  cleanup()
})

// JSDOM matchMedia mock (uPlot 지원)
if (typeof window !== 'undefined' && !window.matchMedia) {
  Object.defineProperty(window, 'matchMedia', {
    writable: true,
    value: (query: string) => ({
      matches: false,
      media: query,
      onchange: null,
      addListener: () => {},
      removeListener: () => {},
      addEventListener: () => {},
      removeEventListener: () => {},
      dispatchEvent: () => false,
    }),
  })
}

// JSDOM ResizeObserver mock
if (typeof window !== 'undefined' && !window.ResizeObserver) {
  window.ResizeObserver = class ResizeObserver {
    observe() {}
    unobserve() {}
    disconnect() {}
  }
}

// JSDOM Path2D mock (uPlot 시리즈 렌더링 지원)
if (typeof globalThis !== 'undefined' && !('Path2D' in globalThis)) {
  // @ts-expect-error Mock Path2D for testing environment
  globalThis.Path2D = class Path2D {
    arc() {}
    lineTo() {}
    moveTo() {}
    closePath() {}
    rect() {}
  }
}

// JSDOM HTMLCanvasElement 2D Context mock (Proxy 기반으로 모든 Canvas 메서드 안전 지원)
if (typeof HTMLCanvasElement !== 'undefined') {
  const dummyCtx: Record<string, unknown> = {
    setLineDash: () => {},
    getLineDash: () => [],
    measureText: () => ({ width: 0 }),
  }
  const proxyCtx = new Proxy(dummyCtx, {
    get(target, prop) {
      if (prop in target) {
        return target[prop as keyof typeof target]
      }
      return () => {}
    },
  })
  HTMLCanvasElement.prototype.getContext = (() =>
    proxyCtx) as unknown as typeof HTMLCanvasElement.prototype.getContext
}
