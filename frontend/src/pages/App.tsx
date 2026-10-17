import { useEffect, useState } from 'react'
import type { FormEvent } from 'react'
import { uploadActivity } from '../api/client'
import type { UploadReceipt } from '../api/client'
import ActivitySummaryCard from '../components/ActivitySummaryCard'
import ActivityChart from '../components/ActivityChart'
import ActivityMap from '../components/ActivityMap'
import ActivitySplitsTable from '../components/ActivitySplitsTable'

type ServiceStatus = '확인 중' | '연결됨' | '연결할 수 없음' | 'DB 연결 확인 필요'

export default function App() {
  const [file, setFile] = useState<File | null>(null)
  const [serviceStatus, setServiceStatus] = useState<ServiceStatus>('확인 중')
  const [isDragging, setIsDragging] = useState(false)
  const [isUploading, setIsUploading] = useState(false)
  const [fileError, setFileError] = useState('')
  const [uploadMessage, setUploadMessage] = useState('')
  const [activityData, setActivityData] = useState<UploadReceipt | null>(null)
  const [hoverIndex, setHoverIndex] = useState<number | null>(null)

  useEffect(() => {
    let active = true

    fetch('/readyz')
      .then((response) => {
        if (active) setServiceStatus(response.ok ? '연결됨' : 'DB 연결 확인 필요')
      })
      .catch(() => {
        if (active) setServiceStatus('연결할 수 없음')
      })

    return () => {
      active = false
    }
  }, [])

  function selectFile(candidate?: File) {
    if (!candidate) return
    if (!candidate.name.toLowerCase().endsWith('.fit')) {
      setFile(null)
      setFileError('FIT 파일(.fit)만 선택할 수 있습니다.')
      setUploadMessage('')
      return
    }
    setFile(candidate)
    setFileError('')
    setUploadMessage('')
  }

  async function handleUpload(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    if (!file || isUploading) return

    setIsUploading(true)
    setFileError('')
    setUploadMessage('')
    try {
      const receipt = await uploadActivity(file)
      const sizeMB = (receipt.size_bytes / (1024 * 1024)).toFixed(2)
      setUploadMessage(`서버가 ${receipt.filename} 파일을 받았습니다 (${sizeMB} MB).`)
      setActivityData(receipt)
      setHoverIndex(null)
    } catch (error) {
      setFileError(error instanceof Error ? error.message : '파일 업로드에 실패했습니다.')
      setActivityData(null)
    } finally {
      setIsUploading(false)
    }
  }

  return (
    <main className="container">
      <header className="header">
        <h1>FIT Lab</h1>
        <p>FIT 운동 기록을 분석하고 시각화합니다.</p>
      </header>

      <section className="card" aria-labelledby="upload-title">
        <h2 id="upload-title">활동 파일 업로드</h2>
        <form onSubmit={handleUpload}>
          <div
            className={`dropzone${isDragging ? ' dropzone-active' : ''}`}
            onDragOver={(event) => {
              event.preventDefault()
              setIsDragging(true)
            }}
            onDragLeave={() => setIsDragging(false)}
            onDrop={(event) => {
              event.preventDefault()
              setIsDragging(false)
              selectFile(event.dataTransfer.files[0])
            }}
          >
            <p>FIT 파일을 여기에 끌어 놓으세요</p>
            <label className="file-picker">
              또는 파일 선택
              <input
                type="file"
                accept=".fit,application/octet-stream"
                onChange={(event) => selectFile(event.target.files?.[0])}
              />
            </label>
          </div>
          {file && <p className="file-name">선택한 파일: {file.name}</p>}
          <button type="submit" disabled={!file || isUploading}>
            {isUploading ? '업로드 중…' : '서버로 업로드'}
          </button>
        </form>
        {fileError && (
          <p className="error" role="alert">
            {fileError}
          </p>
        )}
        {uploadMessage && <p role="status">{uploadMessage}</p>}
        <p className="note">
          업로드된 파일은 서버에서 파싱되어 심박수, 페이스, 고도 시계열 및 GPS 경로로 시각화됩니다.
        </p>
      </section>

      {/* 활동 요약 정보 */}
      {activityData?.summary && (
        <section className="card" aria-labelledby="summary-title">
          <h2 id="summary-title">활동 요약</h2>
          <ActivitySummaryCard summary={activityData.summary} />
        </section>
      )}

      {/* 시계열 차트 및 지도 시각화 */}
      {activityData?.records && activityData.records.length > 0 && (
        <>
          <ActivityChart
            records={activityData.records}
            laps={activityData.laps}
            hoverIndex={hoverIndex}
            onHoverIndex={setHoverIndex}
          />
          <ActivityMap
            records={activityData.records}
            hoverIndex={hoverIndex}
            onHoverIndex={setHoverIndex}
          />
          <ActivitySplitsTable
            records={activityData.records}
            laps={activityData.laps}
            hoverIndex={hoverIndex}
            onHoverIndex={setHoverIndex}
          />
        </>
      )}

      <section className="card" aria-labelledby="status-title">
        <h2 id="status-title">서버 상태</h2>
        <p>
          API 및 DB: <strong>{serviceStatus}</strong>
        </p>
      </section>
    </main>
  )
}
