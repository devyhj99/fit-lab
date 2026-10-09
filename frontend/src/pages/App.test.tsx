import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { cleanup, fireEvent, render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import App from './App'

const fetchMock = vi.fn()

beforeEach(() => {
  fetchMock.mockReset()
  fetchMock.mockResolvedValue({ ok: true, json: async () => ({}) })
  vi.stubGlobal('fetch', fetchMock)
})

afterEach(() => {
  cleanup()
  vi.unstubAllGlobals()
})

describe('FIT 파일 업로드 화면', () => {
  it('FIT 파일을 선택하면 파일명을 표시한다', async () => {
    const user = userEvent.setup()
    render(<App />)

    const file = new File(['fit data'], 'morning-run.fit', { type: 'application/octet-stream' })
    await user.upload(screen.getByLabelText('또는 파일 선택'), file)

    expect(screen.getByText('선택한 파일: morning-run.fit')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: '서버로 업로드' })).toBeEnabled()
  })

  it('FIT 이외 파일은 선택하지 않고 오류를 표시한다', () => {
    render(<App />)

    const file = new File(['text'], 'notes.txt', { type: 'text/plain' })
    fireEvent.change(screen.getByLabelText('또는 파일 선택'), { target: { files: [file] } })

    expect(screen.getByRole('alert')).toHaveTextContent('FIT 파일(.fit)만 선택할 수 있습니다.')
    expect(screen.getByRole('button', { name: '서버로 업로드' })).toBeDisabled()
  })

  it('FIT 파일 선택이 취소되면 기존 선택 상태를 유지한다', async () => {
    const user = userEvent.setup()
    render(<App />)
    const input = screen.getByLabelText('또는 파일 선택')

    await user.upload(input, new File(['fit'], 'ride.fit'))
    fireEvent.change(input, { target: { files: [] } })

    expect(screen.getByText('선택한 파일: ride.fit')).toBeInTheDocument()
  })

  it('확장자의 대소문자를 구분하지 않고 FIT 파일로 처리한다', async () => {
    const user = userEvent.setup()
    render(<App />)

    await user.upload(
      screen.getByLabelText('또는 파일 선택'),
      new File(['fit'], 'RIDE.FIT'),
    )

    expect(screen.getByText('선택한 파일: RIDE.FIT')).toBeInTheDocument()
    expect(screen.queryByRole('alert')).not.toBeInTheDocument()
  })

  it('드롭한 FIT 파일을 선택한다', () => {
    render(<App />)

    const file = new File(['fit data'], 'trail.fit', { type: 'application/octet-stream' })
    const dropzone = screen.getByText('FIT 파일을 여기에 끌어 놓으세요').parentElement
    if (!dropzone) throw new Error('파일 드롭 영역을 찾지 못했습니다.')

    fireEvent.drop(dropzone, { dataTransfer: { files: [file] } })

    expect(screen.getByText('선택한 파일: trail.fit')).toBeInTheDocument()
  })

  it('드래그 중 드롭 영역 스타일을 바꾸고 드래그가 끝나면 되돌린다', () => {
    render(<App />)
    const dropzone = screen.getByText('FIT 파일을 여기에 끌어 놓으세요').parentElement
    if (!dropzone) throw new Error('파일 드롭 영역을 찾지 못했습니다.')

    fireEvent.dragOver(dropzone)
    expect(dropzone).toHaveClass('dropzone-active')

    fireEvent.dragLeave(dropzone)
    expect(dropzone).not.toHaveClass('dropzone-active')
  })

  it('준비 상태 확인에서 HTTP 오류를 표시한다', async () => {
    fetchMock.mockResolvedValueOnce({ ok: false })
    render(<App />)

    expect(await screen.findByText('DB 연결 확인 필요')).toBeInTheDocument()
  })

  it('준비 상태 확인에 실패하면 연결할 수 없다고 표시한다', async () => {
    fetchMock.mockRejectedValueOnce(new Error('network error'))
    render(<App />)

    expect(await screen.findByText('연결할 수 없음')).toBeInTheDocument()
  })

  it('선택한 FIT 파일을 API로 전송하고 응답을 표시한다', async () => {
    const user = userEvent.setup()
    fetchMock
      .mockResolvedValueOnce({ ok: true })
      .mockResolvedValueOnce({
        ok: true,
        json: async () => ({ status: 'received', filename: 'morning-run.fit', size_bytes: 8 }),
      })
    render(<App />)

    const file = new File(['fit data'], 'morning-run.fit', { type: 'application/octet-stream' })
    await user.upload(screen.getByLabelText('또는 파일 선택'), file)
    await user.click(screen.getByRole('button', { name: '서버로 업로드' }))

    expect(await screen.findByRole('status')).toHaveTextContent('서버가 morning-run.fit 파일을 받았습니다 (0.00 MB).')
    expect(fetchMock).toHaveBeenNthCalledWith(2, '/api/activities', expect.objectContaining({ method: 'POST' }))
  })

  it('업로드 API 오류를 사용자에게 표시한다', async () => {
    const user = userEvent.setup()
    fetchMock
      .mockResolvedValueOnce({ ok: true })
      .mockResolvedValueOnce({ ok: false, json: async () => ({ error: '업로드 파일은 25MB 이하여야 합니다.' }) })
    render(<App />)

    const file = new File(['fit data'], 'large.fit', { type: 'application/octet-stream' })
    await user.upload(screen.getByLabelText('또는 파일 선택'), file)
    await user.click(screen.getByRole('button', { name: '서버로 업로드' }))

    expect(await screen.findByRole('alert')).toHaveTextContent('업로드 파일은 25MB 이하여야 합니다.')
  })
})
