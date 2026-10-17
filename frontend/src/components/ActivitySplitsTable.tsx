import { useMemo, useState } from 'react'
import type { LapData, RecordPoint } from '../api/client'

export type SplitMode = '1km' | '5km' | '5min' | '10min' | 'interval'

export interface SplitItem {
  id: number
  label: string
  distanceM: number
  durationSec: number
  avgPaceSecPerKm: number | null
  avgHeartRate: number | null
  maxHeartRate: number | null
  elevationGainM: number
  startIndex: number
  endIndex: number
  intensity?: string
}

interface Props {
  records: RecordPoint[]
  laps?: LapData[]
  hoverIndex?: number | null
  onHoverIndex?: (index: number | null) => void
}

export function formatTime(sec: number): string {
  const m = Math.floor(sec / 60)
  const s = Math.floor(sec % 60)
  return `${m}:${String(s).padStart(2, '0')}`
}

export function formatPace(secPerKm: number | null): string {
  if (secPerKm == null || secPerKm <= 0 || secPerKm > 1800) return '-'
  const m = Math.floor(secPerKm / 60)
  const s = Math.round(secPerKm % 60)
  return `${m}'${String(s).padStart(2, '0')}"`
}

/**
 * 센서 노이즈가 심한 고도 데이터로부터 현실적인 획득 고도(Ascent)를 계산합니다.
 * 이동 평균 스무딩(5포인트 윈도우)과 2m 이상의 유의미한 언덕 상승 임계값(Threshold)을 적용합니다.
 */
export function calculateElevationGain(records: RecordPoint[]): number {
  if (!records || records.length < 2) return 0

  // 1. 유효 고도 추출
  const alts: number[] = []
  for (let i = 0; i < records.length; i++) {
    const a = records[i].altitude_m
    if (a !== undefined && a !== null && !Number.isNaN(a)) {
      alts.push(a)
    }
  }
  // 2. 포인트 수가 충분할 때만 이동 평균 스무딩 (윈도우 크기 5)
  const smoothed: number[] = []
  if (alts.length >= 5) {
    const windowSize = 5
    for (let i = 0; i < alts.length; i++) {
      let sum = 0
      let count = 0
      for (
        let w = Math.max(0, i - Math.floor(windowSize / 2));
        w <= Math.min(alts.length - 1, i + Math.floor(windowSize / 2));
        w++
      ) {
        sum += alts[w]
        count++
      }
      smoothed.push(sum / count)
    }
  } else {
    smoothed.push(...alts)
  }

  // 3. 유의미한 상승 변동(2m 임계치) 누적
  let gain = 0
  let anchor = smoothed[0]
  const threshold = alts.length >= 10 ? 2.0 : 0.5 // 포인트가 많은 실데이터는 2m, 적은 목데이터는 0.5m

  for (let i = 1; i < smoothed.length; i++) {
    const diff = smoothed[i] - anchor
    if (diff >= threshold) {
      gain += diff
      anchor = smoothed[i]
    } else if (diff <= -threshold) {
      anchor = smoothed[i]
    }
  }

  // 만약 전체 상승폭이 2m 미만이지만 구간 시작과 끝의 순상승(net elevation)이 양수인 경우 반영
  const netGain = smoothed[smoothed.length - 1] - smoothed[0]
  if (gain === 0 && netGain > 0.5) {
    return Math.round(netGain)
  }

  return Math.round(gain)
}

export function computeSplits(
  records: RecordPoint[],
  mode: SplitMode,
  laps?: LapData[],
): SplitItem[] {
  if (!records || records.length === 0) return []

  // 인터벌 모드 (FIT 장치 Lap 데이터 기준)
  if (mode === 'interval' && laps && laps.length > 0) {
    let currentRecordIdx = 0
    return laps.map((lap, idx) => {
      const lapStartSec =
        records.length > 0 && lap.start_time
          ? Math.max(
              0,
              (new Date(lap.start_time).getTime() - new Date(records[0].time).getTime()) / 1000,
            )
          : currentRecordIdx

      const startIdx = Math.max(
        0,
        records.findIndex((r) => r.elapsed_sec >= lapStartSec),
      )
      const targetEndSec = lapStartSec + lap.duration_sec
      let endIdx = records.findIndex((r) => r.elapsed_sec >= targetEndSec)
      if (endIdx === -1) endIdx = records.length - 1
      currentRecordIdx = endIdx

      let label = `Lap ${idx + 1}`
      if (lap.intensity) {
        const lower = lap.intensity.toLowerCase()
        if (lower.includes('warmup')) label = `${idx + 1} (웜업)`
        else if (lower.includes('cooldown')) label = `${idx + 1} (쿨다운)`
        else if (lower.includes('rest') || lower.includes('recovery')) label = `${idx + 1} (회복)`
        else if (lower.includes('active') || lower.includes('interval')) label = `${idx + 1} (질주)`
        else label = `${idx + 1} (${lap.intensity})`
      } else if (laps.length > 2) {
        // intensity 필드가 없는 앱(WorkOutDoors 등)의 경우
        if (idx === 0) {
          label = '1 (웜업)'
        } else if (idx === laps.length - 1) {
          label = `${idx + 1} (쿨다운)`
        } else {
          // 거리나 지속시간이 짧은 쪽이 보통 회복 랩
          const isRecovery = lap.duration_sec <= 90 || lap.distance_m < 150
          label = `${idx + 1} (${isRecovery ? '회복' : '질주'})`
        }
      }

      // Lap에 기록된 elevation_gain_m이 있으면 우선 사용, 없으면 슬라이스에서 정밀 필터링 계산
      let elevGain = lap.elevation_gain_m !== undefined ? Math.round(lap.elevation_gain_m) : 0
      if (lap.elevation_gain_m === undefined) {
        const slice = records.slice(startIdx, endIdx + 1)
        elevGain = calculateElevationGain(slice)
      }

      return {
        id: idx + 1,
        label,
        distanceM: lap.distance_m,
        durationSec: lap.duration_sec,
        avgPaceSecPerKm: lap.avg_pace_sec_per_km ?? null,
        avgHeartRate: lap.avg_heart_rate ?? null,
        maxHeartRate: lap.max_heart_rate ?? null,
        elevationGainM: elevGain,
        startIndex: startIdx >= 0 ? startIdx : 0,
        endIndex: endIdx >= 0 ? endIdx : records.length - 1,
        intensity: lap.intensity,
      }
    })
  }

  const isDistMode = mode === '1km' || mode === '5km'
  const stepDistM = mode === '1km' ? 1000 : 5000
  const stepTimeSec = mode === '5min' ? 300 : 600

  const splits: SplitItem[] = []
  let splitIndex = 1
  let splitStartIndex = 0
  let prevDistM = records[0].distance_m ?? 0
  let prevElapsedSec = records[0].elapsed_sec

  let currentThreshold = isDistMode ? stepDistM : stepTimeSec

  for (let i = 0; i < records.length; i++) {
    const r = records[i]
    const currentDistM = r.distance_m ?? 0
    const currentElapsedSec = r.elapsed_sec
    const metric = isDistMode ? currentDistM : currentElapsedSec
    const isLast = i === records.length - 1

    if (metric >= currentThreshold || isLast) {
      const slice = records.slice(splitStartIndex, i + 1)
      if (slice.length > 0) {
        const dist = Math.max(0, currentDistM - prevDistM)
        const dur = Math.max(0, currentElapsedSec - prevElapsedSec)

        // 오직 마지막 찌꺼기 구간에서 거리나 시간이 0인 무의미한 포인트는 스킵
        if (isLast && dist <= 0 && dur <= 0 && splits.length > 0) {
          continue
        }

        // 심박수 집계
        let hrSum = 0
        let hrCount = 0
        let maxHr = 0

        for (let j = 0; j < slice.length; j++) {
          const pt = slice[j]
          if (pt.heart_rate != null && pt.heart_rate > 0) {
            hrSum += pt.heart_rate
            hrCount++
            if (pt.heart_rate > maxHr) maxHr = pt.heart_rate
          }
        }

        const avgHr = hrCount > 0 ? Math.round(hrSum / hrCount) : null
        const avgPace = dist > 50 && dur > 0 ? (dur / dist) * 1000 : null
        const elevGain = calculateElevationGain(slice)

        splits.push({
          id: splitIndex,
          label: `${splitIndex}`,
          distanceM: dist,
          durationSec: dur,
          avgPaceSecPerKm: avgPace,
          avgHeartRate: avgHr,
          maxHeartRate: maxHr > 0 ? maxHr : null,
          elevationGainM: Math.round(elevGain),
          startIndex: splitStartIndex,
          endIndex: i,
        })

        splitIndex++
        splitStartIndex = i
        prevDistM = currentDistM
        prevElapsedSec = currentElapsedSec
        currentThreshold += isDistMode ? stepDistM : stepTimeSec
      }
    }
  }

  return splits
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
