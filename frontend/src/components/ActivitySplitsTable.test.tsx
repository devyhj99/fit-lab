import { describe, expect, it } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import ActivitySplitsTable from './ActivitySplitsTable'
import { formatTablePace as formatPace, formatTime } from '../utils/formatters'
import { computeSplits } from '../utils/splits'
import type { RecordPoint } from '../api/client'

describe('ActivitySplitsTable', () => {
  const mockRecords: RecordPoint[] = [
    {
      time: '2026-10-03T11:00:00Z',
      elapsed_sec: 0,
      distance_m: 0,
      heart_rate: 140,
      altitude_m: 10,
    },
    {
      time: '2026-10-03T11:05:00Z',
      elapsed_sec: 300,
      distance_m: 1000,
      heart_rate: 150,
      altitude_m: 15,
    },
    {
      time: '2026-10-03T11:10:00Z',
      elapsed_sec: 600,
      distance_m: 2000,
      heart_rate: 160,
      altitude_m: 20,
    },
  ]

  it('포맷터 함수들이 정상 동작한다', () => {
    expect(formatTime(125)).toBe('2:05')
    expect(formatPace(300)).toBe('5\'00"')
    expect(formatPace(null)).toBe('-')
  })

  it('1km 스플릿 계산이 정확하게 분할된다', () => {
    const splits = computeSplits(mockRecords, '1km')
    expect(splits).toHaveLength(2)

    // 1구간
    expect(splits[0].id).toBe(1)
    expect(splits[0].distanceM).toBe(1000)
    expect(splits[0].durationSec).toBe(300)
    expect(splits[0].avgHeartRate).toBe(145)
    expect(splits[0].maxHeartRate).toBe(150)
    expect(splits[0].elevationGainM).toBe(5)

    // 2구간
    expect(splits[1].id).toBe(2)
    expect(splits[1].distanceM).toBe(1000)
    expect(splits[1].durationSec).toBe(300)
    expect(splits[1].avgHeartRate).toBe(155)
    expect(splits[1].maxHeartRate).toBe(160)
  })

  it('5분 스플릿 모드로 전환 시 정상 계산된다', () => {
    const splits = computeSplits(mockRecords, '5min')
    expect(splits).toHaveLength(2)
    expect(splits[0].durationSec).toBe(300)
    expect(splits[1].durationSec).toBe(300)
  })

  it('레코드가 비어있으면 렌더링하지 않는다', () => {
    const { container } = render(<ActivitySplitsTable records={[]} />)
    expect(container.firstChild).toBeNull()
  })

  it('테이블 렌더링 및 모드 전환 버튼이 동작한다', () => {
    render(<ActivitySplitsTable records={mockRecords} />)

    expect(screen.getByTestId('activity-splits-table')).toBeInTheDocument()
    expect(screen.getByText('구간별 상세 분석 (계산된 스플릿)')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: '1km' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: '5분' })).toBeInTheDocument()

    // 5분 버튼 클릭
    fireEvent.click(screen.getByRole('button', { name: '5분' }))
    expect(screen.getByRole('button', { name: '5분' })).toHaveClass('active')
  })

  it('laps 데이터가 전달되면 수동/인터벌 랩과 계산된 스플릿 테이블이 모두 함께 표시된다', () => {
    const mockLaps = [
      {
        lap_index: 1,
        start_time: '2026-10-03T11:00:00Z',
        duration_sec: 240,
        distance_m: 800,
        avg_pace_sec_per_km: 300,
        avg_heart_rate: 165,
        max_heart_rate: 175,
        lap_trigger: 'manual',
        intensity: 'active',
      },
      {
        lap_index: 2,
        start_time: '2026-10-03T11:04:00Z',
        duration_sec: 120,
        distance_m: 200,
        avg_pace_sec_per_km: 600,
        avg_heart_rate: 130,
        max_heart_rate: 140,
        lap_trigger: 'manual',
        intensity: 'rest',
      },
    ]

    render(<ActivitySplitsTable records={mockRecords} laps={mockLaps} />)

    // 수동/인터벌 랩 카드와 계산된 스플릿 카드가 동시에 렌더링됨 (Runalyze 스타일)
    expect(screen.getByTestId('manual-laps-card')).toBeInTheDocument()
    expect(screen.getByTestId('computed-splits-card')).toBeInTheDocument()
    expect(screen.getByText('수동 / 인터벌 랩 (2개)')).toBeInTheDocument()
    expect(screen.getByText('구간별 상세 분석 (계산된 스플릿)')).toBeInTheDocument()

    // 인터벌 라벨 및 심박수 표시 확인
    expect(screen.getByText('1 (질주)')).toBeInTheDocument()
    expect(screen.getByText('2 (회복)')).toBeInTheDocument()
    expect(screen.getByText('165 bpm')).toBeInTheDocument()
    expect(screen.getByText('130 bpm')).toBeInTheDocument()
  })
})
