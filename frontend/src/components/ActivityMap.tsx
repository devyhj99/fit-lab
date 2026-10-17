import { memo, useEffect, useMemo, useRef } from 'react'
import L from 'leaflet'
import 'leaflet/dist/leaflet.css'
import type { RecordPoint } from '../api/client'

interface Props {
  records: RecordPoint[]
  hoverIndex?: number | null
  onHoverIndex?: (index: number | null) => void
}

function ActivityMapComponent({ records, hoverIndex, onHoverIndex }: Props) {
  const mapContainerRef = useRef<HTMLDivElement>(null)
  const mapInstanceRef = useRef<L.Map | null>(null)
  const hoverMarkerRef = useRef<L.CircleMarker | null>(null)
  const onHoverIndexRef = useRef(onHoverIndex)
  const recordsRef = useRef(records)

  useEffect(() => {
    onHoverIndexRef.current = onHoverIndex
  }, [onHoverIndex])

  useEffect(() => {
    recordsRef.current = records
  }, [records])

  // GPS 좌표가 존재하는 레코드만 필터링 (records 배열 참조가 동일하면 재계산 방지)
  const gpsPoints = useMemo(
    () =>
      records.filter(
        (r): r is RecordPoint & { latitude: number; longitude: number } =>
          typeof r.latitude === 'number' &&
          typeof r.longitude === 'number' &&
          !Number.isNaN(r.latitude) &&
          !Number.isNaN(r.longitude) &&
          (r.latitude !== 0 || r.longitude !== 0),
      ),
    [records],
  )

  // 지도 인스턴스 초기화 및 고정된 경로 레이어 (gpsPoints가 바뀔 때만 지도 생성)
  useEffect(() => {
    if (!mapContainerRef.current || gpsPoints.length === 0) return

    // 지도 인스턴스 생성
    const map = L.map(mapContainerRef.current, {
      zoomControl: true,
      scrollWheelZoom: true,
    })
    mapInstanceRef.current = map

    // OpenStreetMap 표준 무료 타일 레이어 (API Key 불필요)
    L.tileLayer('https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png', {
      maxZoom: 19,
      attribution: '&copy; OpenStreetMap contributors',
    }).addTo(map)

    // 경로 폴리라인
    const latlngs: L.LatLngExpression[] = gpsPoints.map((p) => [p.latitude, p.longitude])
    const polyline = L.polyline(latlngs, {
      color: '#2563eb',
      weight: 4,
      opacity: 0.85,
      lineJoin: 'round',
    }).addTo(map)

    // 경로 바운드에 맞춰 자동 줌/이동
    if (latlngs.length > 0) {
      map.fitBounds(polyline.getBounds(), { padding: [30, 30] })
    }

    // 출발 마커 (초록색)
    const startPt = gpsPoints[0]
    L.circleMarker([startPt.latitude, startPt.longitude], {
      radius: 7,
      color: '#15803d',
      fillColor: '#22c55e',
      fillOpacity: 1,
      weight: 2,
    })
      .bindTooltip('출발', { permanent: false, direction: 'top' })
      .addTo(map)

    // 도착 마커 (빨간색)
    const endPt = gpsPoints[gpsPoints.length - 1]
    L.circleMarker([endPt.latitude, endPt.longitude], {
      radius: 7,
      color: '#b91c1c',
      fillColor: '#ef4444',
      fillOpacity: 1,
      weight: 2,
    })
      .bindTooltip('도착', { permanent: false, direction: 'top' })
      .addTo(map)

    // 호버 인디케이터 마커 (주황색)
    const hoverMarker = L.circleMarker([0, 0], {
      radius: 9,
      color: '#d97706',
      fillColor: '#f59e0b',
      fillOpacity: 0.9,
      weight: 3,
    })
    hoverMarkerRef.current = hoverMarker

    // 지도 경로 호버 시 가장 가까운 레코드 인덱스 탐색
    polyline.on('mousemove', (e: L.LeafletMouseEvent) => {
      const clickLatLng = e.latlng
      const currentRecords = recordsRef.current
      let closestIdx = -1
      let minDist = Number.POSITIVE_INFINITY

      for (let i = 0; i < currentRecords.length; i++) {
        const r = currentRecords[i]
        if (r.latitude !== undefined && r.longitude !== undefined) {
          const d = clickLatLng.distanceTo([r.latitude, r.longitude])
          if (d < minDist) {
            minDist = d
            closestIdx = i
          }
        }
      }

      if (closestIdx !== -1) {
        onHoverIndexRef.current?.(closestIdx)
      }
    })

    polyline.on('mouseout', () => {
      onHoverIndexRef.current?.(null)
    })

    return () => {
      map.remove()
      mapInstanceRef.current = null
      hoverMarkerRef.current = null
    }
  }, [gpsPoints])

  // hoverIndex 변경 시 지도 재렌더링 없이 오직 hoverMarker 위치만 즉각 업데이트
  useEffect(() => {
    const map = mapInstanceRef.current
    const hoverMarker = hoverMarkerRef.current
    if (!map || !hoverMarker) return

    const currentRecords = recordsRef.current
    if (
      hoverIndex === undefined ||
      hoverIndex === null ||
      hoverIndex < 0 ||
      hoverIndex >= currentRecords.length
    ) {
      if (map.hasLayer(hoverMarker)) {
        map.removeLayer(hoverMarker)
      }
      return
    }

    const r = currentRecords[hoverIndex]
    if (
      r.latitude !== undefined &&
      r.longitude !== undefined &&
      !Number.isNaN(r.latitude) &&
      !Number.isNaN(r.longitude) &&
      (r.latitude !== 0 || r.longitude !== 0)
    ) {
      hoverMarker.setLatLng([r.latitude, r.longitude])
      if (!map.hasLayer(hoverMarker)) {
        hoverMarker.addTo(map)
      }
    } else {
      if (map.hasLayer(hoverMarker)) {
        map.removeLayer(hoverMarker)
      }
    }
  }, [hoverIndex])

  if (gpsPoints.length === 0) {
    return (
      <div className="card map-empty-card" data-testid="activity-map-empty">
        <p className="note">📍 GPS 위치 정보가 없는 실내 활동(트레드밀 등)입니다.</p>
      </div>
    )
  }

  return (
    <div className="card map-card" data-testid="activity-map">
      <h3>운동 경로 (GPS 트랙)</h3>
      <div className="map-view" ref={mapContainerRef} />
    </div>
  )
}

const ActivityMap = memo(ActivityMapComponent)
export default ActivityMap
