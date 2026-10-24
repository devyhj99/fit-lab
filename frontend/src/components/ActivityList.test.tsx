import { describe, expect, it, vi } from 'vitest'
import { fireEvent, render, screen } from '@testing-library/react'
import ActivityList from './ActivityList'
import type { ActivityListItem } from '../api/client'

describe('ActivityList', () => {
  const sampleItems: ActivityListItem[] = [
    {
      id: 10,
      name: '저녁 러닝',
      filename: 'hangang.fit',
      source_channel: 'upload',
      sport: 'running',
      start_time: '2026-10-03T11:00:00Z',
      has_gps: true,
      duration_sec: 1800,
      distance_m: 5000,
      avg_pace_sec_per_km: 360,
      avg_heart_rate: 152,
      created_at: '2026-10-03T12:00:00Z',
    },
    {
      id: 11,
      name: '800m 인터벌',
      filename: 'track.fit',
      source_channel: 'dropbox',
      sport: 'running',
      start_time: '2026-10-04T11:00:00Z',
      has_gps: true,
      duration_sec: 2400,
      distance_m: 8000,
      avg_pace_sec_per_km: 300,
      avg_heart_rate: 168,
      created_at: '2026-10-04T12:00:00Z',
    },
  ]

  it('목록이 비어있으면 안내 문구를 표시한다', () => {
    render(
      <ActivityList
        activities={[]}
        total={0}
        selectedId={null}
        onSelect={vi.fn()}
        onDelete={vi.fn()}
      />,
    )

    expect(
      screen.getByText('저장된 활동 기록이 없습니다. 위에서 FIT 파일을 업로드해 보세요.'),
    ).toBeInTheDocument()
  })

  it('활동 목록을 렌더링하고 선택, 이름 변경, 삭제 이벤트를 전달한다', () => {
    const onSelect = vi.fn()
    const onRename = vi.fn()
    const onDelete = vi.fn()

    render(
      <ActivityList
        activities={sampleItems}
        total={2}
        selectedId={10}
        onSelect={onSelect}
        onRename={onRename}
        onDelete={onDelete}
      />,
    )

    expect(screen.getByText('저장된 활동 목록 (2개)')).toBeInTheDocument()
    expect(screen.getByText('저녁 러닝')).toBeInTheDocument()
    expect(screen.getByText('800m 인터벌')).toBeInTheDocument()
    expect(screen.getByText('직접 업로드')).toBeInTheDocument()
    expect(screen.getByText('Dropbox')).toBeInTheDocument()

    // 행 클릭 시 onSelect 호출
    fireEvent.click(screen.getByText('800m 인터벌'))
    expect(onSelect).toHaveBeenCalledWith(11)

    // 이름 변경 버튼 클릭 -> 입력 후 저장
    fireEvent.click(screen.getByRole('button', { name: '저녁 러닝 이름 변경' }))
    const input = screen.getByLabelText('활동 이름 입력')
    fireEvent.change(input, { target: { value: '석촌호수 저녁 조깅' } })
    fireEvent.click(screen.getByRole('button', { name: '저장' }))
    expect(onRename).toHaveBeenCalledWith(10, '석촌호수 저녁 조깅')

    // 삭제 버튼 클릭 시 행 클릭으로 전파되지 않고 onDelete만 호출
    onSelect.mockClear()
    fireEvent.click(screen.getByRole('button', { name: '저녁 러닝 삭제' }))
    expect(onDelete).toHaveBeenCalledWith(10)
    expect(onSelect).not.toHaveBeenCalled()
  })
})
