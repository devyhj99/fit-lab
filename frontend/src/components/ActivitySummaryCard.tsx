import type { ActivitySummary } from '../api/client'
import { formatDistance, formatDuration, formatPace } from '../utils/formatters'

interface Props {
  summary: ActivitySummary
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
