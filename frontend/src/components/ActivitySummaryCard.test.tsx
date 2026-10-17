import { describe, expect, it } from 'vitest'
import { render, screen } from '@testing-library/react'
import ActivitySummaryCard, {
  formatDistance,
  formatDuration,
  formatPace,
} from './ActivitySummaryCard'
import type { ActivitySummary } from '../api/client'

describe('ActivitySummaryCard', () => {
  const mockSummary: ActivitySummary = {
    sport: 'running',
    start_time: '2026-10-03T11:08:29Z',
    duration_sec: 4167,
    distance_m: 13466,
    avg_heart_rate: 151,
    max_heart_rate: 172,
    avg_speed_mps: 3.23,
    avg_pace_sec_per_km: 309,
    elevation_gain_m: 45,
    calories: 950,
  }

  it('시간, 거리, 페이스 포맷터를 정확하게 계산한다', () => {
    expect(formatDuration(3665)).toBe('1시간 1분 5초')
    expect(formatDuration(59)).toBe('0분 59초')
    expect(formatDistance(5000)).toBe('5.00 km')
    expect(formatPace(330)).toBe('5\'30" /km')
    expect(formatPace(undefined)).toBe('-')
  })

  it('활동 요약 정보를 렌더링한다', () => {
    render(<ActivitySummaryCard summary={mockSummary} />)

    expect(screen.getByText('RUNNING')).toBeInTheDocument()
    expect(screen.getByText('13.47 km')).toBeInTheDocument()
    expect(screen.getByText('1시간 9분 27초')).toBeInTheDocument()
    expect(screen.getByText('151 bpm')).toBeInTheDocument()
    expect(screen.getByText('172 bpm')).toBeInTheDocument()
    expect(screen.getByText('5\'09" /km')).toBeInTheDocument()
    expect(screen.getByText('45 m')).toBeInTheDocument()
    expect(screen.getByText('950 kcal')).toBeInTheDocument()
  })
})
