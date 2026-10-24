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

export function formatPace(secPerKm?: number | null): string {
  if (!secPerKm || secPerKm <= 0 || secPerKm > 1800) return '-'
  const m = Math.floor(secPerKm / 60)
  const s = Math.floor(secPerKm % 60)
  return `${m}'${String(s).padStart(2, '0')}" /km`
}

export function formatTablePace(secPerKm: number | null): string {
  if (secPerKm == null || secPerKm <= 0 || secPerKm > 1800) return '-'
  const m = Math.floor(secPerKm / 60)
  const s = Math.round(secPerKm % 60)
  return `${m}'${String(s).padStart(2, '0')}"`
}

export function formatTime(sec: number): string {
  const m = Math.floor(sec / 60)
  const s = Math.floor(sec % 60)
  return `${m}:${String(s).padStart(2, '0')}`
}
