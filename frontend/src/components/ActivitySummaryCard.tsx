import type { ActivitySummary } from '../api/client'

interface Props {
  summary: ActivitySummary
}

export function formatDuration(seconds: number): string {
  const h = Math.floor(seconds / 3600)
  const m = Math.floor((seconds % 3600) / 60)
  const s = Math.floor(seconds % 60)
  if (h > 0) {
    return `${h}시간 ${m}분 ${s}초`
  }
  return `${m}분 ${s}초`
}

export function formatDistance(meters: number): string {
  return `${(meters / 1000).toFixed(2)} km`
}

export function formatPace(secPerKm?: number): string {
  if (!secPerKm || secPerKm <= 0 || secPerKm > 1800) return '-'
  const m = Math.floor(secPerKm / 60)
  const s = Math.floor(secPerKm % 60)
  return `${m}'${String(s).padStart(2, '0')}" /km`
}

export default function ActivitySummaryCard({ summary }: Props) {
  return (
    <div className="summary-card" data-testid="activity-summary-card">
      <div className="summary-header">
        <span className="sport-badge">{summary.sport.toUpperCase()}</span>
        <span className="activity-date">
          {new Date(summary.start_time).toLocaleString('ko-KR', {
            dateStyle: 'medium',
            timeStyle: 'short',
          })}
        </span>
      </div>

      {/* 핵심 4대 지표 (거리 · 시간 · 평균 페이스 · 평균 심박수) */}
      <div className="summary-primary-stats">
        <div className="stat-item primary">
          <span className="stat-label">거리</span>
          <span className="stat-value primary-value">{formatDistance(summary.distance_m)}</span>
        </div>
        <div className="stat-item primary">
          <span className="stat-label">시간</span>
          <span className="stat-value primary-value">{formatDuration(summary.duration_sec)}</span>
        </div>
        <div className="stat-item primary">
          <span className="stat-label">평균 페이스</span>
          <span className="stat-value primary-value">
            {formatPace(summary.avg_pace_sec_per_km)}
          </span>
        </div>
        <div className="stat-item primary">
          <span className="stat-label">평균 심박수</span>
          <span className="stat-value primary-value">
            {summary.avg_heart_rate ? `${summary.avg_heart_rate} bpm` : '-'}
          </span>
        </div>
      </div>

      {/* 보조 지표 (최대 심박수 · 획득 고도 · 칼로리) */}
      <div className="summary-secondary-stats">
        <div className="stat-item">
          <span className="stat-label">최대 심박수</span>
          <span className="stat-value">
            {summary.max_heart_rate ? `${summary.max_heart_rate} bpm` : '-'}
          </span>
        </div>
        {summary.elevation_gain_m !== undefined && (
          <div className="stat-item">
            <span className="stat-label">획득 고도</span>
            <span className="stat-value">{Math.round(summary.elevation_gain_m)} m</span>
          </div>
        )}
        {summary.calories !== undefined && (
          <div className="stat-item">
            <span className="stat-label">칼로리</span>
            <span className="stat-value">{summary.calories} kcal</span>
          </div>
        )}
      </div>
    </div>
  )
}
