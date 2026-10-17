import { describe, expect, it } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import ActivityChart from './ActivityChart'
import type { RecordPoint } from '../api/client'

describe('ActivityChart', () => {
  const mockRecords: RecordPoint[] = [
    {
      time: '2026-10-03T11:08:29Z',
      elapsed_sec: 0,
      distance_m: 0,
      heart_rate: 140,
      speed_mps: 3.0,
      pace_sec_per_km: 333,
      altitude_m: 20,
    },
    {
      time: '2026-10-03T11:08:30Z',
      elapsed_sec: 1,
      distance_m: 3,
      heart_rate: 145,
      speed_mps: 3.2,
      pace_sec_per_km: 312,
      altitude_m: 21,
    },
  ]

  it('레코드가 비어있으면 렌더링하지 않는다', () => {
    const { container } = render(<ActivityChart records={[]} />)
    expect(container.firstChild).toBeNull()
  })

  it('레코드가 존재하면 차트 컨테이너와 컨트롤을 렌더링한다', () => {
    render(<ActivityChart records={mockRecords} />)

    expect(screen.getByTestId('activity-chart')).toBeInTheDocument()
    expect(screen.getByText('시계열 분석 (심박수 · 페이스 · 고도)')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: '시간 기준' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: '거리 기준' })).toBeInTheDocument()
  })

  it('기본 상태(hover 이전)에서 범례에 전체 및 평균 데이터가 표시된다', async () => {
    const { container } = render(<ActivityChart records={mockRecords} />)

    const legend = container.querySelector('.u-legend')
    expect(legend).not.toBeNull()

    // uPlot은 마이크로태스크를 거쳐 legend를 초기화하므로 waitFor로 확인
    await waitFor(() => {
      // mockRecords의 심박수 평균: (140 + 145) / 2 = 143 bpm
      expect(legend?.textContent).toContain('평균 143 bpm')
      // mockRecords의 고도 평균: (20 + 21) / 2 = 21 m (반올림)
      expect(legend?.textContent).toContain('평균 21 m')
      // 경과 시간 총계: 1초 -> 0:01
      expect(legend?.textContent).toContain('전체 0:01')
    })
  })

  it('laps 데이터가 전달되면 인터벌 구간 토글 버튼이 노출된다', () => {
    const mockLaps = [
      {
        lap_index: 1,
        start_time: '2026-10-03T11:08:29Z',
        duration_sec: 1,
        distance_m: 3,
      },
    ]

    render(<ActivityChart records={mockRecords} laps={mockLaps} />)
    expect(screen.getByRole('button', { name: '인터벌 구간 ON' })).toBeInTheDocument()
  })
})
