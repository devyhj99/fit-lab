const API_BASE = '/api'

export interface ActivitySummary {
  workout_name?: string
  sport: string
  sub_sport?: string
  start_time: string
  duration_sec: number
  moving_time_sec?: number
  distance_m: number
  elevation_gain_m?: number
  avg_heart_rate?: number
  max_heart_rate?: number
  avg_speed_mps?: number
  max_speed_mps?: number
  avg_pace_sec_per_km?: number
  avg_cadence?: number
  max_cadence?: number
  avg_power?: number
  max_power?: number
  calories?: number
}

export interface RecordPoint {
  time: string
  elapsed_sec: number
  distance_m?: number
  latitude?: number
  longitude?: number
  altitude_m?: number
  speed_mps?: number
  pace_sec_per_km?: number
  heart_rate?: number
  cadence?: number
  power?: number
  temperature?: number
}

export interface LapData {
  lap_index: number
  start_time: string
  duration_sec: number
  moving_time_sec?: number
  distance_m: number
  avg_speed_mps?: number
  max_speed_mps?: number
  avg_pace_sec_per_km?: number
  avg_heart_rate?: number
  max_heart_rate?: number
  avg_cadence?: number
  avg_power?: number
  elevation_gain_m?: number
  calories?: number
  lap_trigger?: string
  intensity?: string
}

export interface UploadReceipt {
  status: 'received'
  activity_id?: number
  name?: string
  is_duplicate?: boolean
  filename: string
  size_bytes: number
  summary?: ActivitySummary
  records?: RecordPoint[]
  laps?: LapData[]
}

export interface ActivityListItem {
  id: number
  name: string
  filename: string
  source_channel: string
  sport: string
  sub_sport?: string
  start_time: string
  has_gps: boolean
  duration_sec: number
  moving_time_sec?: number
  distance_m: number
  elevation_gain_m?: number
  avg_heart_rate?: number
  max_heart_rate?: number
  avg_speed_mps?: number
  avg_pace_sec_per_km?: number
  calories?: number
  created_at: string
}

export interface ActivityListResponse {
  activities: ActivityListItem[]
  total: number
  limit: number
  offset: number
}

export interface ActivityDetailResponse {
  id: number
  name: string
  filename: string
  source_channel: string
  has_gps: boolean
  summary: ActivitySummary
  laps?: LapData[]
  records: RecordPoint[]
}

export async function uploadActivity(file: File): Promise<UploadReceipt> {
  const body = new FormData()
  body.append('file', file)

  const response = await fetch(`${API_BASE}/activities`, {
    method: 'POST',
    body,
  })
  const result = (await response.json()) as UploadReceipt | { error?: string }

  if (!response.ok) {
    throw new Error(
      'error' in result && result.error ? result.error : `업로드 실패 (${response.status})`,
    )
  }

  return result as UploadReceipt
}

export async function fetchActivities(limit = 20, offset = 0): Promise<ActivityListResponse> {
  const response = await fetch(`${API_BASE}/activities?limit=${limit}&offset=${offset}`)
  const result = (await response.json()) as ActivityListResponse | { error?: string }

  if (!response.ok) {
    throw new Error(
      'error' in result && result.error ? result.error : `활동 목록 조회 실패 (${response.status})`,
    )
  }

  return result as ActivityListResponse
}

export async function fetchActivityDetail(
  id: number,
  downsample = 1,
): Promise<ActivityDetailResponse> {
  const response = await fetch(`${API_BASE}/activities/${id}?downsample=${downsample}`)
  const result = (await response.json()) as ActivityDetailResponse | { error?: string }

  if (!response.ok) {
    throw new Error(
      'error' in result && result.error ? result.error : `활동 상세 조회 실패 (${response.status})`,
    )
  }

  return result as ActivityDetailResponse
}

export async function deleteActivity(id: number): Promise<void> {
  const response = await fetch(`${API_BASE}/activities/${id}`, {
    method: 'DELETE',
  })
  const result = (await response.json()) as { status?: string; error?: string }

  if (!response.ok) {
    throw new Error(
      'error' in result && result.error ? result.error : `활동 삭제 실패 (${response.status})`,
    )
  }
}

export async function updateActivityName(
  id: number,
  name: string,
): Promise<{ id: number; name: string }> {
  const response = await fetch(`${API_BASE}/activities/${id}`, {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ name }),
  })
  const result = (await response.json()) as { id: number; name: string } | { error?: string }

  if (!response.ok) {
    throw new Error(
      'error' in result && result.error ? result.error : `활동 이름 수정 실패 (${response.status})`,
    )
  }

  return result as { id: number; name: string }
}
