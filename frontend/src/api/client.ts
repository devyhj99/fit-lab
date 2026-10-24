const API_BASE = '/api'

export interface ActivitySummary {
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
  distance_m: number
  avg_pace_sec_per_km?: number
  avg_heart_rate?: number
  max_heart_rate?: number
  elevation_gain_m?: number
  calories?: number
  lap_trigger?: string
  intensity?: string
}

export interface UploadReceipt {
  status: 'received'
  filename: string
  size_bytes: number
  summary?: ActivitySummary
  records?: RecordPoint[]
  laps?: LapData[]
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
