const API_BASE = '/api'

export interface UploadReceipt {
  status: 'received'
  filename: string
  size_bytes: number
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
    throw new Error('error' in result && result.error ? result.error : `업로드 실패 (${response.status})`)
  }

  return result as UploadReceipt
}
