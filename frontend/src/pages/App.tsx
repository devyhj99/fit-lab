import { useState } from 'react'

export default function App() {
  const [fileName, setFileName] = useState('')

  return (
    <main className="shell">
      <header className="topbar">
        <a className="brand" href="/" aria-label="FIT Lab 홈">
          <span className="brand-mark">F</span>
          <span>FIT<span className="brand-light">LAB</span></span>
        </a>
        <span className="status"><i /> 활동 분석 워크스페이스</span>
      </header>

      <section className="hero">
        <p className="eyebrow">TRAINING DATA, CLEARLY</p>
        <h1>운동 기록을<br /><em>한눈에</em> 살펴보세요.</h1>
        <p className="intro">FIT 파일을 업로드해 거리, 페이스, 심박과 경로를 분석하세요.</p>
      </section>

      <section className="upload-card" aria-labelledby="upload-title">
        <div className="card-heading">
          <div>
            <p className="eyebrow">START WITH A FIT FILE</p>
            <h2 id="upload-title">활동 기록 가져오기</h2>
          </div>
          <span className="file-tag">.FIT</span>
        </div>
        <label className="dropzone">
          <input
            type="file"
            accept=".fit,application/octet-stream"
            onChange={(event) => setFileName(event.target.files?.[0]?.name ?? '')}
          />
          <span className="upload-icon" aria-hidden="true">↑</span>
          <strong>{fileName || 'FIT 파일을 선택하세요'}</strong>
          <span className="muted">기기에서 내보낸 .fit 파일을 업로드할 수 있습니다.</span>
          <span className="choose-button">파일 선택</span>
        </label>
        <p className="privacy-note"><span aria-hidden="true">⌑</span> 운동 기록과 위치 정보는 안전하게 다뤄집니다.</p>
      </section>

      <footer>FIT LAB <span>·</span> YOUR TRAINING, IN CONTEXT</footer>
    </main>
  )
}
