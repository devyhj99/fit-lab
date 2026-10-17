import { afterEach, describe, expect, it, vi } from 'vitest'
import { uploadActivity } from './client'

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('활동 파일 API', () => {
  it('파일을 multipart 요청으로 전송하고 응답을 반환한다', async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      json: async () => ({ status: 'received', filename: 'run.fit', size_bytes: 3 }),
    })
    vi.stubGlobal('fetch', fetchMock)
    const file = new File(['fit'], 'run.fit')

    await expect(uploadActivity(file)).resolves.toEqual({
      status: 'received',
      filename: 'run.fit',
      size_bytes: 3,
    })

    expect(fetchMock).toHaveBeenCalledWith('/api/activities', {
      method: 'POST',
      body: expect.any(FormData),
    })
    expect(fetchMock.mock.calls[0][1].body.get('file')).toBe(file)
  })

  it('오류 응답에 설명이 없으면 상태 코드로 오류를 만든다', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue({
        ok: false,
        status: 413,
        json: async () => ({}),
      }),
    )

    await expect(uploadActivity(new File(['fit'], 'run.fit'))).rejects.toThrow('업로드 실패 (413)')
  })
})
