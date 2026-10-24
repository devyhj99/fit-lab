import { afterEach, describe, expect, it, vi } from 'vitest'
import {
  deleteActivity,
  fetchActivities,
  fetchActivityDetail,
  uploadActivity,
} from './client'

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

  it('활동 목록을 조회한다', async () => {
    const mockList = {
      activities: [
        {
          id: 1,
          name: 'morning-run',
          filename: 'morning-run.fit',
          source_channel: 'upload',
          sport: 'running',
          start_time: '2026-10-03T11:00:00Z',
          has_gps: true,
          duration_sec: 1800,
          distance_m: 5000,
          created_at: '2026-10-03T12:00:00Z',
        },
      ],
      total: 1,
      limit: 20,
      offset: 0,
    }
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      json: async () => mockList,
    })
    vi.stubGlobal('fetch', fetchMock)

    await expect(fetchActivities(20, 0)).resolves.toEqual(mockList)
    expect(fetchMock).toHaveBeenCalledWith('/api/activities?limit=20&offset=0')
  })

  it('활동 상세 정보를 조회하고 삭제 요청을 수행한다', async () => {
    const mockDetail = {
      id: 1,
      name: 'morning-run',
      filename: 'morning-run.fit',
      source_channel: 'upload',
      has_gps: true,
      summary: {
        sport: 'running',
        start_time: '2026-10-03T11:00:00Z',
        duration_sec: 1800,
        distance_m: 5000,
      },
      records: [],
    }
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce({
        ok: true,
        json: async () => mockDetail,
      })
      .mockResolvedValueOnce({
        ok: true,
        json: async () => ({ status: 'deleted', id: 1 }),
      })
    vi.stubGlobal('fetch', fetchMock)

    await expect(fetchActivityDetail(1, 2)).resolves.toEqual(mockDetail)
    expect(fetchMock).toHaveBeenCalledWith('/api/activities/1?downsample=2')

    await expect(deleteActivity(1)).resolves.toBeUndefined()
    expect(fetchMock).toHaveBeenCalledWith('/api/activities/1', { method: 'DELETE' })
  })
})
