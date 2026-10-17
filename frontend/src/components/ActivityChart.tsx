import { useEffect, useRef, useState } from 'react'
import uPlot from 'uplot'
import 'uplot/dist/uPlot.min.css'
import type { LapData, RecordPoint } from '../api/client'

interface Props {
  records: RecordPoint[]
  laps?: LapData[]
  hoverIndex?: number | null
  onHoverIndex?: (index: number | null) => void
}

type XAxisMode = 'time' | 'distance'

function formatSeconds(sec: number): string {
  const m = Math.floor(sec / 60)
  const s = Math.floor(sec % 60)
  return `${m}:${String(s).padStart(2, '0')}`
}

export default function ActivityChart({ records, laps, hoverIndex, onHoverIndex }: Props) {
  const containerRef = useRef<HTMLDivElement>(null)
  const chartInstanceRef = useRef<uPlot | null>(null)
  const [xAxisMode, setXAxisMode] = useState<XAxisMode>('time')
  const [showIntervals, setShowIntervals] = useState<boolean>(true)
  const hasLaps = Boolean(laps && laps.length > 0)

  useEffect(() => {
    if (!containerRef.current || records.length === 0) return

    // 데이터 정렬 및 추출
    const xVals: number[] = []
    const hrVals: (number | null)[] = []
    const paceVals: (number | null)[] = []
    const altVals: (number | null)[] = []

    for (let i = 0; i < records.length; i++) {
      const r = records[i]
      const x = xAxisMode === 'time' ? r.elapsed_sec : r.distance_m ? r.distance_m / 1000 : 0
      xVals.push(x)
      hrVals.push(r.heart_rate ?? null)

      // 페이스(초/km) -> 분/km 소수로 변환 (예: 5분 30초 = 5.5분)
      if (r.pace_sec_per_km && r.pace_sec_per_km > 120 && r.pace_sec_per_km < 900) {
        paceVals.push(Number((r.pace_sec_per_km / 60).toFixed(2)))
      } else {
        paceVals.push(null)
      }

      altVals.push(r.altitude_m !== undefined ? Math.round(r.altitude_m) : null)
    }

    const data: uPlot.AlignedData = [xVals, hrVals, paceVals, altVals]

    const width = containerRef.current.clientWidth || 800
    const height = 320

    // 기본 상태(hover 해제 시) 표시를 위한 통계치 계산
    let hrSum = 0
    let hrCount = 0
    let paceSum = 0
    let paceCount = 0
    let altSum = 0
    let altCount = 0

    for (let i = 0; i < records.length; i++) {
      const r = records[i]
      if (r.heart_rate != null && r.heart_rate > 0) {
        hrSum += r.heart_rate
        hrCount++
      }
      if (r.pace_sec_per_km != null && r.pace_sec_per_km > 120 && r.pace_sec_per_km < 900) {
        paceSum += r.pace_sec_per_km
        paceCount++
      }
      if (r.altitude_m !== undefined && r.altitude_m !== null) {
        altSum += r.altitude_m
        altCount++
      }
    }

    const avgHr = hrCount > 0 ? Math.round(hrSum / hrCount) : null
    const avgPaceSec = paceCount > 0 ? paceSum / paceCount : null
    const avgAlt = altCount > 0 ? Math.round(altSum / altCount) : null

    const formatPaceVal = (valInMinutes: number): string => {
      const m = Math.floor(valInMinutes)
      const s = Math.round((valInMinutes - m) * 60)
      return `${m}'${String(s).padStart(2, '0')}" /km`
    }

    const totalTimeSec = records.length > 0 ? records[records.length - 1].elapsed_sec : 0
    const lastRecordDist = records.length > 0 ? records[records.length - 1].distance_m : 0
    const totalDistKm = lastRecordDist ? lastRecordDist / 1000 : 0

    const opts: uPlot.Options = {
      width,
      height,
      cursor: {
        drag: { x: true, y: false },
        points: { size: 6, fill: '#1f2937' },
      },
      hooks: {
        draw: [
          (u) => {
            if (!showIntervals || !laps || laps.length === 0 || records.length === 0) return

            const ctx = u.ctx
            const plotLeft = u.bbox.left
            const plotTop = u.bbox.top
            const plotWidth = u.bbox.width
            const plotHeight = u.bbox.height

            ctx.save()

            let currentLapStartSec = 0
            for (let idx = 0; idx < laps.length; idx++) {
              const lap = laps[idx]
              const lapStartSec =
                lap.start_time && records[0]?.time
                  ? Math.max(
                      0,
                      (new Date(lap.start_time).getTime() - new Date(records[0].time).getTime()) /
                        1000,
                    )
                  : currentLapStartSec

              const lapEndSec = lapStartSec + lap.duration_sec
              currentLapStartSec = lapEndSec

              // X축 값 변환
              let startXVal: number
              let endXVal: number

              if (xAxisMode === 'time') {
                startXVal = lapStartSec
                endXVal = lapEndSec
              } else {
                // 거리 기준일 경우 해당 시간의 distance_m 탐색
                const startPt = records.find((r) => r.elapsed_sec >= lapStartSec)
                const endPt = records.find((r) => r.elapsed_sec >= lapEndSec)
                startXVal = startPt?.distance_m ? startPt.distance_m / 1000 : 0
                endXVal = endPt?.distance_m
                  ? endPt.distance_m / 1000
                  : (records[records.length - 1].distance_m ?? 0) / 1000
              }

              const leftPos = Math.max(
                plotLeft,
                Math.min(plotLeft + plotWidth, u.valToPos(startXVal, 'x', true)),
              )
              const rightPos = Math.max(
                plotLeft,
                Math.min(plotLeft + plotWidth, u.valToPos(endXVal, 'x', true)),
              )
              const bandWidth = rightPos - leftPos

              if (bandWidth <= 0) continue

              // 강도(질주/회복/웜업/쿨다운)에 따른 배경색
              const isRecovery =
                lap.intensity?.toLowerCase().includes('rest') ||
                lap.intensity?.toLowerCase().includes('recovery') ||
                (idx > 0 &&
                  idx < laps.length - 1 &&
                  (lap.duration_sec <= 90 || lap.distance_m < 150))

              const isWork =
                lap.intensity?.toLowerCase().includes('active') ||
                lap.intensity?.toLowerCase().includes('interval') ||
                (!isRecovery && idx > 0 && idx < laps.length - 1)

              if (isWork) {
                // 질주/인터벌 구간: 시원한 연하늘/연청색 밴드
                ctx.fillStyle = 'rgba(59, 130, 246, 0.08)'
                ctx.fillRect(leftPos, plotTop, bandWidth, plotHeight)
              } else if (isRecovery) {
                // 휴식/회복 구간: 연한 주황/노란색 밴드
                ctx.fillStyle = 'rgba(245, 158, 11, 0.06)'
                ctx.fillRect(leftPos, plotTop, bandWidth, plotHeight)
              }

              // 구간 구분 세로 점선 (첫 번째 제외)
              if (idx > 0 && leftPos >= plotLeft && leftPos <= plotLeft + plotWidth) {
                ctx.beginPath()
                ctx.setLineDash([4, 4])
                ctx.strokeStyle = 'rgba(148, 163, 184, 0.5)'
                ctx.lineWidth = 1
                ctx.moveTo(leftPos, plotTop)
                ctx.lineTo(leftPos, plotTop + plotHeight)
                ctx.stroke()
              }

              // 상단 미니 라벨 텍스트 (폭이 20px 이상 확보될 때만 표시)
              if (bandWidth >= 22) {
                ctx.fillStyle = isWork ? '#2563eb' : isRecovery ? '#d97706' : '#64748b'
                ctx.font = '600 9px system-ui, sans-serif'
                ctx.textAlign = 'center'
                ctx.fillText(`L${idx + 1}`, leftPos + bandWidth / 2, plotTop + 12)
              }
            }

            ctx.restore()
          },
        ],
        setCursor: [
          (u) => {
            const idx = u.cursor.idx
            if (idx !== undefined && idx !== null && idx >= 0 && idx < records.length) {
              onHoverIndex?.(idx)
            } else {
              onHoverIndex?.(null)
            }
          },
        ],
      },
      scales: {
        x: { time: false },
        hr: { auto: true, range: (_u, min, max) => [Math.max(60, min - 10), max + 10] },
        pace: {
          auto: true,
          // 페이스는 낮을수록 빠르므로 상하 반전 (낮은 숫자가 위)
          range: (_u, min, max) => [Math.max(2, min - 0.5), Math.min(12, max + 0.5)],
          dir: -1,
        },
        alt: { auto: true, range: (_u, min, max) => [Math.max(0, min - 5), max + 10] },
      },
      axes: [
        {
          scale: 'x',
          label: xAxisMode === 'time' ? '시간' : '거리 (km)',
          values: (_u, vals) =>
            vals.map((v) => (xAxisMode === 'time' ? formatSeconds(v) : `${v.toFixed(1)}k`)),
        },
        {
          scale: 'hr',
          label: '심박수 (bpm)',
          stroke: '#ef4444',
          size: 50,
          grid: { show: true },
        },
        {
          scale: 'pace',
          label: '페이스 (min/km)',
          stroke: '#3b82f6',
          side: 1,
          size: 50,
          grid: { show: false },
          values: (_u, vals) =>
            vals.map((v) => {
              const m = Math.floor(v)
              const s = Math.round((v - m) * 60)
              return `${m}'${String(s).padStart(2, '0')}"`
            }),
        },
      ],
      series: [
        {
          label: xAxisMode === 'time' ? '시간' : '거리',
          value: (_u, v) => {
            if (v != null) {
              return xAxisMode === 'time' ? formatSeconds(v) : `${v.toFixed(2)} km`
            }
            return xAxisMode === 'time'
              ? `전체 ${formatSeconds(totalTimeSec)}`
              : `전체 ${totalDistKm.toFixed(2)} km`
          },
        },
        {
          label: '심박수',
          scale: 'hr',
          stroke: '#ef4444',
          width: 1.5,
          value: (_u, v) => {
            if (v != null) {
              return `${Math.round(v)} bpm`
            }
            return avgHr != null ? `평균 ${avgHr} bpm` : '-'
          },
        },
        {
          label: '페이스',
          scale: 'pace',
          stroke: '#3b82f6',
          width: 1.5,
          value: (_u, v) => {
            if (v != null) {
              return formatPaceVal(v)
            }
            return avgPaceSec != null ? `평균 ${formatPaceVal(avgPaceSec / 60)}` : '-'
          },
        },
        {
          label: '고도',
          scale: 'alt',
          stroke: '#10b981',
          width: 1,
          show: true,
          value: (_u, v) => {
            if (v != null) {
              return `${Math.round(v)} m`
            }
            return avgAlt != null ? `평균 ${avgAlt} m` : '-'
          },
        },
      ],
    }

    // 기존 인스턴스 정리
    if (chartInstanceRef.current) {
      chartInstanceRef.current.destroy()
    }

    containerRef.current.innerHTML = ''
    const chart = new uPlot(opts, data, containerRef.current)
    chartInstanceRef.current = chart

    const resizeObserver = new ResizeObserver((entries) => {
      for (const entry of entries) {
        if (entry.contentRect.width > 0) {
          chart.setSize({
            width: entry.contentRect.width,
            height,
          })
        }
      }
    })
    resizeObserver.observe(containerRef.current)

    return () => {
      resizeObserver.disconnect()
      chart.destroy()
      chartInstanceRef.current = null
    }
  }, [records, laps, xAxisMode, showIntervals, onHoverIndex])

  // 부모(지도 등)로부터 전달받은 hoverIndex 동기화
  useEffect(() => {
    const chart = chartInstanceRef.current
    if (!chart || hoverIndex === undefined) return

    if (hoverIndex === null || hoverIndex < 0 || hoverIndex >= records.length) {
      chart.setCursor({ left: -10, top: -10 })
      return
    }

    const r = records[hoverIndex]
    const xVal = xAxisMode === 'time' ? r.elapsed_sec : r.distance_m ? r.distance_m / 1000 : 0
    const left = chart.valToPos(xVal, 'x')
    chart.setCursor({ left, top: 40 })
  }, [hoverIndex, records, xAxisMode])

  if (records.length === 0) {
    return null
  }

  return (
    <div className="chart-card" data-testid="activity-chart">
      <div className="chart-header">
        <h3>시계열 분석 (심박수 · 페이스 · 고도)</h3>
        <div className="chart-controls">
          {hasLaps && (
            <button
              type="button"
              className={`btn-toggle ${showIntervals ? 'active' : ''}`}
              onClick={() => setShowIntervals((prev) => !prev)}
            >
              인터벌 구간 {showIntervals ? 'ON' : 'OFF'}
            </button>
          )}
          <button
            type="button"
            className={`btn-toggle ${xAxisMode === 'time' ? 'active' : ''}`}
            onClick={() => setXAxisMode('time')}
          >
            시간 기준
          </button>
          <button
            type="button"
            className={`btn-toggle ${xAxisMode === 'distance' ? 'active' : ''}`}
            onClick={() => setXAxisMode('distance')}
          >
            거리 기준
          </button>
        </div>
      </div>
      <div className="uplot-wrapper" ref={containerRef} />
    </div>
  )
}
