import { useMemo, useState } from 'react'
import type { LapData, RecordPoint } from '../api/client'
import { formatTime, formatTablePace as formatPace } from '../utils/formatters'
import { computeSplits, type SplitItem, type SplitMode } from '../utils/splits'

interface Props {
  records: RecordPoint[]
  laps?: LapData[]
  hoverIndex?: number | null
  onHoverIndex?: (index: number | null) => void
}

type ComputedSplitMode = Exclude<SplitMode, 'interval'>

function computeOverallAvgPace(splits: SplitItem[]): number | null {
  if (splits.length === 0) return null
  let totalDist = 0
  let totalDur = 0
  for (const s of splits) {
    totalDist += s.distanceM
    totalDur += s.durationSec
  }
  return totalDist > 0 ? (totalDur / totalDist) * 1000 : null
}

function SplitsDataTable({
  splits,
  tableKey,
   hoveredRowKey,
  setHoveredRowKey,
  hoverIndex,
  onHoverIndex,
}: {
  splits: SplitItem[]
  tableKey: string
  hoveredRowKey: string | null
  setHoveredRowKey: (key: string | null) => void
  hoverIndex?: number | null
  onHoverIndex?: (index: number | null) => void
}) {
  const overallAvgPace = useMemo(() => computeOverallAvgPace(splits), [splits])

  return (
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
            const rowKey = `${tableKey}-${split.id}`
            const isHovered =
              hoveredRowKey !== null
                ? hoveredRowKey === rowKey
                : hoverIndex !== null &&
                  hoverIndex !== undefined &&
                  hoverIndex >= split.startIndex &&
                  hoverIndex <= split.endIndex

            const paceDiff =
              split.avgPaceSecPerKm && overallAvgPace
                ? Math.round(split.avgPaceSecPerKm - overallAvgPace)
                : null

            return (
              <tr
                key={rowKey}
                className={isHovered ? 'row-hovered' : ''}
                onMouseEnter={() => {
                  setHoveredRowKey(rowKey)
                  onHoverIndex?.(split.startIndex)
                }}
                onMouseLeave={() => {
                  setHoveredRowKey(null)
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
  )
}

export default function ActivitySplitsTable({ records, laps, hoverIndex, onHoverIndex }: Props) {
  const hasLaps = Boolean(laps && laps.length > 0)
  const [computedMode, setComputedMode] = useState<ComputedSplitMode>('1km')
  const [hoveredRowKey, setHoveredRowKey] = useState<string | null>(null)

  const manualSplits = useMemo(
    () => (hasLaps ? computeSplits(records, 'interval', laps) : []),
    [records, hasLaps, laps],
  )

  const computedSplits = useMemo(
    () => computeSplits(records, computedMode),
    [records, computedMode],
  )

  if (records.length === 0 || (manualSplits.length === 0 && computedSplits.length === 0)) {
    return null
  }

  return (
    <div data-testid="activity-splits-table">
      {manualSplits.length > 0 && (
        <div className="card splits-card" data-testid="manual-laps-card">
          <div className="splits-header">
            <h3>수동 / 인터벌 랩 ({manualSplits.length}개)</h3>
          </div>
          <SplitsDataTable
            splits={manualSplits}
            tableKey="manual"
            hoveredRowKey={hoveredRowKey}
            setHoveredRowKey={setHoveredRowKey}
            hoverIndex={hoverIndex}
            onHoverIndex={onHoverIndex}
          />
        </div>
      )}

      {computedSplits.length > 0 && (
        <div className="card splits-card" data-testid="computed-splits-card">
          <div className="splits-header">
            <h3>구간별 상세 분석 (계산된 스플릿)</h3>
            <div className="splits-controls">
              <button
                type="button"
                className={`btn-toggle ${computedMode === '1km' ? 'active' : ''}`}
                onClick={() => setComputedMode('1km')}
              >
                1km
              </button>
              <button
                type="button"
                className={`btn-toggle ${computedMode === '5km' ? 'active' : ''}`}
                onClick={() => setComputedMode('5km')}
              >
                5km
              </button>
              <button
                type="button"
                className={`btn-toggle ${computedMode === '5min' ? 'active' : ''}`}
                onClick={() => setComputedMode('5min')}
              >
                5분
              </button>
              <button
                type="button"
                className={`btn-toggle ${computedMode === '10min' ? 'active' : ''}`}
                onClick={() => setComputedMode('10min')}
              >
                10분
              </button>
            </div>
          </div>
          <SplitsDataTable
            splits={computedSplits}
            tableKey="computed"
            hoveredRowKey={hoveredRowKey}
            setHoveredRowKey={setHoveredRowKey}
            hoverIndex={hoverIndex}
            onHoverIndex={onHoverIndex}
          />
        </div>
      )}
    </div>
  )
}
