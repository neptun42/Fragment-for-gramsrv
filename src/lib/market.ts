import { useEffect, useState } from 'react'
import { api, withServer } from './api'
import { useStore } from './store'
import type { Gift, Lot, Server } from './data'

type Remote<T> = { data: T | null; loading: boolean; failed: boolean }

// грузим с бэка и перезапрашиваем после входа или покупки
function useRemote<T>(path: string | null): Remote<T> {
  const { version } = useStore()
  const [state, setState] = useState<Remote<T>>({ data: null, loading: true, failed: false })

  useEffect(() => {
    if (!path) return
    let alive = true
    setState((s) => ({ ...s, loading: true }))
    api<T>(withServer(path))
      .then((data) => alive && setState({ data, loading: false, failed: false }))
      .catch(() => alive && setState({ data: null, loading: false, failed: true }))
    return () => {
      alive = false
    }
  }, [path, version])

  return state
}

export const useServers = () => useRemote<Server[]>('/api/servers')
export const useLots = (kind: Lot['kind']) => useRemote<Lot[]>(`/api/lots?kind=${kind}`)
export const useLot = (kind: Lot['kind'], slug: string) => useRemote<Lot>(`/api/lots/${kind}/${encodeURIComponent(slug)}`)
export const useGifts = () => useRemote<Gift[]>('/api/gifts')
export const useGift = (id: string) => useRemote<Gift>(`/api/gifts/${encodeURIComponent(id)}`)
