import { useState } from 'react'
import type { ActivityListItem } from '../api/client'
import { formatDistance, formatDuration, formatPace } from '../utils/formatters'

interface Props {
  activities: ActivityListItem[]
  total: number
  selectedId: number | null
  isLoading?: boolean
  onSelect: (id: number) => void
  onRename?: (id: number, name: string) => Promise<void> | void
  onDelete: (id: number) => void
}

function formatDateTime(iso: string): string {
  const date = new Date(iso)
  if (Number.isNaN(date.getTime())) return iso
  return date.toLocaleString('ko-KR', {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  })
}

export default function ActivityList({
  activities,
  total,
  selectedId,
  isLoading = false,
  onSelect,
  onRename,
  onDelete,
}: Props) {
  const [editingId, setEditingId] = useState<number | null>(null)
  const [draftName, setDraftName] = useState('')

  function startEditing(item: ActivityListItem) {
    setEditingId(item.id)
    setDraftName(item.name || item.filename)
  }

  function cancelEditing() {
    setEditingId(null)
    setDraftName('')
  }

  async function submitEditing(id: number) {
    const trimmed = draftName.trim()
    if (!trimmed) return
    await onRename?.(id, trimmed)
    setEditingId(null)
    setDraftName('')
  }

  return (
    <section
      className="card activity-list-card"
      aria-labelledby="activity-list-title"
      data-testid="activity-list"
    >
      <div className="splits-header">
        <h2 id="activity-list-title" style={{ margin: 0 }}>
          저장된 활동 목록 {total > 0 ? `(${total}개)` : ''}
        </h2>
      </div>

      {isLoading && activities.length === 0 ? (
        <p className="note">활동 목록을 불러오는 중…</p>
      ) : activities.length === 0 ? (
        <p className="note">저장된 활동 기록이 없습니다. 위에서 FIT 파일을 업로드해 보세요.</p>
      ) : (
        <div className="table-responsive">
          <table className="splits-table activity-list-table">
            <thead>
              <tr>
                <th scope="col">종목</th>
                <th scope="col">활동 이름</th>
                <th scope="col">일시</th>
                <th scope="col">거리</th>
                <th scope="col">시간</th>
                <th scope="col">평균 페이스</th>
                <th scope="col">평균 심박</th>
                <th scope="col">출처</th>
                <th scope="col">관리</th>
              </tr>
            </thead>
            <tbody>
              {activities.map((item) => {
                const isSelected = selectedId === item.id
                const isEditing = editingId === item.id
                const displayName = item.name || item.filename

                return (
                  <tr
                    key={item.id}
                    className={isSelected ? 'row-hovered activity-row-selected' : ''}
                    onClick={() => {
                      if (!isEditing) onSelect(item.id)
                    }}
                  >
                    <td>
                      <span className="sport-badge">{item.sport.toUpperCase()}</span>
                    </td>
                    <td className="font-bold">
                      {isEditing ? (
                        <div
                          className="activity-name-edit"
                          onClick={(event) => event.stopPropagation()}
                        >
                          <input
                            type="text"
                            className="activity-name-input"
                            aria-label="활동 이름 입력"
                            value={draftName}
                            maxLength={100}
                            onChange={(event) => setDraftName(event.target.value)}
                            onKeyDown={(event) => {
                              if (event.key === 'Enter') {
                                event.preventDefault()
                                void submitEditing(item.id)
                              } else if (event.key === 'Escape') {
                                cancelEditing()
                              }
                            }}
                          />
                          <button
                            type="button"
                            className="btn-save-name"
                            onClick={() => void submitEditing(item.id)}
                          >
                            저장
                          </button>
                          <button
                            type="button"
                            className="btn-cancel-name"
                            onClick={cancelEditing}
                          >
                            취소
                          </button>
                        </div>
                      ) : (
                        displayName
                      )}
                    </td>
                    <td>{formatDateTime(item.start_time)}</td>
                    <td>{formatDistance(item.distance_m)}</td>
                    <td>{formatDuration(item.duration_sec)}</td>
                    <td>{formatPace(item.avg_pace_sec_per_km)}</td>
                    <td>{item.avg_heart_rate ? `${item.avg_heart_rate} bpm` : '-'}</td>
                    <td>
                      <span className="source-badge">
                        {item.source_channel === 'dropbox' ? 'Dropbox' : '직접 업로드'}
                      </span>
                    </td>
                    <td>
                      <div
                        className="activity-actions"
                        onClick={(event) => event.stopPropagation()}
                      >
                        <button
                          type="button"
                          className="btn-edit-activity"
                          aria-label={`${displayName} 이름 변경`}
                          onClick={() => startEditing(item)}
                        >
                          이름 변경
                        </button>
                        <button
                          type="button"
                          className="btn-delete-activity"
                          aria-label={`${displayName} 삭제`}
                          onClick={() => onDelete(item.id)}
                        >
                          삭제
                        </button>
                      </div>
                    </td>
                  </tr>
                )
              })}
            </tbody>
          </table>
        </div>
      )}
    </section>
  )
}
