const API_BASE = '/api'

export async function uploadActivity(file: File): Promise<unknown> {
  const body = new FormData()
  body.append('file', file)

  const response = await fetch(`${API_BASE}/activities`, {
    method: 'POST',
    body,
  })

  if (!response.ok) {
    throw new Error(`Activity upload failed (${response.status})`)
  }

  return response.json()
}
