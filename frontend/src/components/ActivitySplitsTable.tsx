import { useMemo, useState } from 'react'
import type { LapData, RecordPoint } from '../api/client'
import { formatTime, formatTablePace as formatPace } from '../utils/formatters'
import { computeSplits, type SplitMode } from '../utils/splits'

interface Props {
  records: RecordPoint[]
  laps?: LapData[]
  hoverIndex?: number | null
  onHoverIndex?: (index: number | null) => void
}

export default function ActivitySplitsTable({ records, laps, hoverIndex, onHoverIndex }: Props) {
  const hasLaps = Boolean(laps && laps.length > 0)
  const [userSelectedMode, setUserSelectedMode] = useState<SplitMode | null>(null)
  const [hoveredSplitId, setHoveredSplitId] = useState<number | null>(null)

  // 사용자가 명시적으로 선택한 모드가 없으면 laps 유무에 따라 기본값 자동 결정
  const splitMode: SplitMode = userSelectedMode ?? (hasLaps ? 'interval' : '1km')

  const splits = useMemo(() => computeSplits(records, splitMode, laps), [records, splitMode, laps])

  // 전체 평균 페이스 (비교 표시용)
  const overallAvgPace = useMemo(() => {
    if (splits.length === 0) return null
    let totalDist = 0
    let totalDur = 0
    for (const s of splits) {
      totalDist += s.distanceM
      totalDur += s.durationSec
    }
    return totalDist > 0 ? (totalDur / totalDist) * 1000 : null
  }, [splits])

  if (records.length === 0 || splits.length === 0) {
    return null
  }

  return (
    <div className="card splits-card" data-testid="activity-splits-table">
      <div className="splits-header">
        <h3>구간별 상세 분석 (스플릿)</h3>
        <div className="splits-controls">
          {hasLaps && (
            <button
              type="button"
              className={`btn-toggle btn-interval ${splitMode === 'interval' ? 'active' : ''}`}
              onClick={() => setUserSelectedMode('interval')}
            >
              인터벌 ({laps?.length}개)
            </button>
          )}
          <button
            type="button"
            className={`btn-toggle ${splitMode === '1km' ? 'active' : ''}`}
            onClick={() => setUserSelectedMode('1km')}
          >
            1km
          </button>
          <button
            type="button"
            className={`btn-toggle ${splitMode === '5km' ? 'active' : ''}`}
            onClick={() => setUserSelectedMode('5km')}
          >
            5km
          </button>
          <button
            type="button"
            className={`btn-toggle ${splitMode === '5min' ? 'active' : ''}`}
            onClick={() => setUserSelectedMode('5min')}
          >
            5분
          </button>
          <button
            type="button"
            className={`btn-toggle ${splitMode === '10min' ? 'active' : ''}`}
            onClick={() => setUserSelectedMode('10min')}
          >
            10분
          </button>
        </div>
      </div>

      <div className="table-responsive">
        <table className="splits-table">
          <thead>
            <tr>
              <th scope="col">구간</th>
              <th scope="col">거리</th>
              <th scope="col">시간</th>
              <th scope="col">평균 페이스</th>
              <th scope="col">평균 심박수</th>
              <th scope="col">최대 심박수</th>
              <th scope="col">획득 고도</th>
            </tr>
          </thead>
          <tbody>
            {splits.map((split) => {
              // 테이블에 직접 마우스를 올린 경우 해당 행만 단독 하이라이트,
              // 차트/지도에서 호버한 경우 해당 인덱스를 포함하는 구간 1개만 하이라이트
              const isHovered =
                hoveredSplitId !== null
                  ? hoveredSplitId === split.id
                  : hoverIndex !== null &&
                    hoverIndex !== undefined &&
                    hoverIndex >= split.startIndex &&
                    hoverIndex <= split.endIndex

              // 평균 대비 페이스 차이 (초/km)
              const paceDiff =
                split.avgPaceSecPerKm && overallAvgPace
                  ? Math.round(split.avgPaceSecPerKm - overallAvgPace)
                  : null

              return (
                <tr
                  key={split.id}
                  className={isHovered ? 'row-hovered' : ''}
                  onMouseEnter={() => {
                    setHoveredSplitId(split.id)
                    onHoverIndex?.(split.startIndex)
                  }}
                  onMouseLeave={() => {
                    setHoveredSplitId(null)
                    onHoverIndex?.(null)
                  }}
                >
                  <td className="col-split font-bold">{split.label}</td>
                  <td>{(split.distanceM / 1000).toFixed(2)} km</td>
                  <td>{formatTime(split.durationSec)}</td>
                  <td>
                    <span className="pace-cell">
                      <span>{formatPace(split.avgPaceSecPerKm)}</span>
                      {paceDiff !== null && (
                        <span
                          className={`pace-diff ${paceDiff < 0 ? 'pace-faster' : paceDiff > 0 ? 'pace-slower' : ''}`}
                          title={`전체 평균 대비 ${paceDiff < 0 ? `${Math.abs(paceDiff)}초 빠름` : `${paceDiff}초 느림`}`}
                        >
                          {paceDiff < 0
                            ? `-${Math.abs(paceDiff)}s`
                            : paceDiff > 0
                              ? `+${paceDiff}s`
                              : '±0s'}
                        </span>
                      )}
                    </span>
                  </td>
                  <td>{split.avgHeartRate ? `${split.avgHeartRate} bpm` : '-'}</td>
                  <td>{split.maxHeartRate ? `${split.maxHeartRate} bpm` : '-'}</td>
                  <td>{split.elevationGainM > 0 ? `+${split.elevationGainM} m` : '0 m'}</td>
                </tr>
              )
            })}
          </tbody>
        </table>
      </div>
    </div>
  )
}
