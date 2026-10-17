import { describe, expect, it } from 'vitest'
import { render, screen } from '@testing-library/react'
import ActivityMap from './ActivityMap'
import type { RecordPoint } from '../api/client'

describe('ActivityMap', () => {
  it('GPS 좌표가 없는 실내 활동일 경우 안내 문구를 렌더링한다', () => {
    const indoorRecords: RecordPoint[] = [
      { time: '2026-10-03T11:08:29Z', elapsed_sec: 0, heart_rate: 140 },
      { time: '2026-10-03T11:08:30Z', elapsed_sec: 1, heart_rate: 142 },
    ]

    render(<ActivityMap records={indoorRecords} />)

    expect(screen.getByTestId('activity-map-empty')).toBeInTheDocument()
    expect(screen.getByText(/GPS 위치 정보가 없는 실내 활동/)).toBeInTheDocument()
  })

  it('유효한 GPS 좌표가 있으면 지도 뷰를 렌더링한다', () => {
    const outdoorRecords: RecordPoint[] = [
      { time: '2026-10-03T11:08:29Z', elapsed_sec: 0, latitude: 37.55, longitude: 127.12 },
      { time: '2026-10-03T11:08:30Z', elapsed_sec: 1, latitude: 37.56, longitude: 127.13 },
    ]

    render(<ActivityMap records={outdoorRecords} />)

    expect(screen.getByTestId('activity-map')).toBeInTheDocument()
    expect(screen.getByText('운동 경로 (GPS 트랙)')).toBeInTheDocument()
  })
})
