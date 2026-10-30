import { createContext, useCallback, useContext, useEffect, useState, type ReactNode } from 'react'
import { api, errorText, getToken, setServer, setToken } from './api'

export type User = { id: number; name: string; username: string; phone: string; joined: number }

type ItemBase = { id: string; title: string; value: number; ts: number; source: string }
export type InvItem =
  | (ItemBase & { kind: 'gift'; slug: string; num: number })
  | (ItemBase & { kind: 'number'; number: string })
  | (ItemBase & { kind: 'username'; username: string })
  | (ItemBase & { kind: 'stars'; amount: number })

export type HistoryKind = 'buy' | 'sell' | 'topup' | 'bid' | 'refund' | 'win' | 'withdraw'
export type HistoryEntry = { id: string; kind: HistoryKind; title: string; amount: number; ts: number }
export type MyBid = { amount: number; ts: number }

export type Session = {
  id: string
  device: string
  place: string
  createdAt: number
  lastSeen: number
}

type Me = {
  user: User | null
  server: string
  balance: number
  inventory: InvItem[]
  history: HistoryEntry[]
  bids: Record<string, MyBid>
  watching: string[]
  sessions: Session[]
  currentSessionId: string
}

export type Toast = { id: number; title: string; tone: 'default' | 'success' | 'error' }

const EMPTY: Me = { user: null, server: '', balance: 0, inventory: [], history: [], bids: {}, watching: [], sessions: [], currentSessionId: '' }

const TZ_PLACES: Record<string, string> = {
  'Europe/Moscow': 'Москва, Россия',
  'Europe/Kaliningrad': 'Калининград, Россия',
  'Europe/Samara': 'Самара, Россия',
  'Asia/Yekaterinburg': 'Екатеринбург, Россия',
  'Asia/Omsk': 'Омск, Россия',
  'Asia/Novosibirsk': 'Новосибирск, Россия',
  'Asia/Krasnoyarsk': 'Красноярск, Россия',
  'Asia/Irkutsk': 'Иркутск, Россия',
  'Asia/Yakutsk': 'Якутск, Россия',
  'Asia/Vladivostok': 'Владивосток, Россия',
  'Asia/Magadan': 'Магадан, Россия',
  'Asia/Kamchatka': 'Петропавловск-Камчатский, Россия',
  'Europe/Kyiv': 'Киев, Украина',
  'Europe/Minsk': 'Минск, Беларусь',
  'Asia/Almaty': 'Алматы, Казахстан',
  'Asia/Tashkent': 'Ташкент, Узбекистан',
  'Asia/Bishkek': 'Бишкек, Кыргызстан',
  'Asia/Tbilisi': 'Тбилиси, Грузия',
  'Asia/Yerevan': 'Ереван, Армения',
}

const browserLabel = (ua: string) => {
  const match =
    ua.match(/Edg(?:e|A|iOS)?\/([\d.]+)/) ??
    ua.match(/OPR\/([\d.]+)/) ??
    ua.match(/Firefox\/([\d.]+)/) ??
    ua.match(/Chrome\/([\d.]+)/) ??
    ua.match(/Version\/([\d.]+).*Safari/)
  if (!match) return 'Браузер'
  const name = /Edg/.test(ua) ? 'Edge' : /OPR\//.test(ua) ? 'Opera' : /Firefox\//.test(ua) ? 'Firefox' : /Chrome\//.test(ua) ? 'Chrome' : 'Safari'
  return `${name} ${match[1].split('.')[0]}`
}

const osLabel = (ua: string) => {
  if (/Windows/.test(ua)) return 'Windows'
  if (/iPhone|iPad|iPod/.test(ua)) return 'iOS'
  if (/Android/.test(ua)) return 'Android'
  if (/Mac OS X/.test(ua)) return 'macOS'
  if (/Linux/.test(ua)) return 'Linux'
  return 'Устройство'
}

// подпись устройства и города для списка сеансов
export const deviceInfo = () => {
  const ua = typeof navigator === 'undefined' ? '' : navigator.userAgent
  const timeZone = Intl.DateTimeFormat().resolvedOptions().timeZone
  const fallback = timeZone.split('/').pop()?.replace(/_/g, ' ') || 'Неизвестно'
  return { device: [browserLabel(ua), osLabel(ua)].join(' · '), place: TZ_PLACES[timeZone] ?? fallback }
}

export const itemName = (i: InvItem) => (i.kind === 'gift' ? `${i.title} #${i.num}` : i.kind === 'number' ? i.number : i.title)

function useStoreValue() {
  const [me, setMe] = useState<Me>(EMPTY)
  const [ready, setReady] = useState(false)
  const [authOpen, setAuthOpen] = useState(false)
  const [toasts, setToasts] = useState<Toast[]>([])
  const [version, setVersion] = useState(0)

  const toast = useCallback((title: string, tone: Toast['tone'] = 'default') => {
    const id = Date.now() + Math.random()
    setToasts((t) => [...t.slice(-1), { id, title, tone }])
    setTimeout(() => setToasts((t) => t.filter((x) => x.id !== id)), 3000)
  }, [])

  // тянем всё про юзера с бэка
  const refresh = useCallback(async () => {
    if (!getToken()) {
      setMe(EMPTY)
      setReady(true)
      return
    }
    try {
      const data = await api<Me>('/api/me')
      setMe(data)
      if (data.server) setServer(data.server)
    } catch {
      setToken(null)
      setMe(EMPTY)
    } finally {
      setReady(true)
      setVersion((v) => v + 1)
    }
  }, [])

  useEffect(() => {
    refresh()
  }, [refresh])

  // сеанс могли завершить с другого устройства, проверяем раз в минуту
  useEffect(() => {
    if (!me.user) return
    const id = setInterval(refresh, 60_000)
    return () => clearInterval(id)
  }, [me.user, refresh])

  // обёртка для действий: ошибку в тост, после успеха обновляем стор
  const run = useCallback(
    async <T,>(fn: () => Promise<T>): Promise<T | null> => {
      try {
        const out = await fn()
        await refresh()
        return out
      } catch (e) {
        toast(errorText(e), 'error')
        return null
      }
    },
    [refresh, toast],
  )

  const login = useCallback(
    async (token: string) => {
      setToken(token)
      await refresh()
    },
    [refresh],
  )

  const logout = useCallback(async () => {
    await api('/api/auth/logout', { method: 'POST' }).catch(() => {})
    setToken(null)
    setMe(EMPTY)
    setVersion((v) => v + 1)
  }, [])

  // гость сменил приватку, перезагружаем витрину
  const pickServer = useCallback((slug: string) => {
    setServer(slug)
    setVersion((v) => v + 1)
  }, [])

  const kickSession = (id: string) => run(() => api(`/api/sessions/${id}`, { method: 'DELETE' }))
  const kickOtherSessions = () => run(() => api('/api/sessions/kick-others', { method: 'POST' }))
  const buyLot = (lotId: string) => run(() => api<InvItem>(`/api/lots/${lotId}/buy`, { method: 'POST' }))
  const placeBid = async (lotId: string, amount: number) => (await run(() => api(`/api/lots/${lotId}/bid`, { body: { amount } }))) !== null
  const toggleWatch = (lotId: string) => run(() => api<{ watching: boolean }>(`/api/lots/${lotId}/watch`, { method: 'POST' }))
  const topUp = (amount: number) => run(() => api<{ credited: number }>('/api/wallet/topup', { body: { amount } }))
  const sellItem = (id: string) => run(() => api<{ credited: number }>(`/api/items/${id}/sell`, { method: 'POST' }))
  const withdrawItem = async (id: string) => (await run(() => api(`/api/items/${id}/withdraw`, { method: 'POST' }))) !== null

  return {
    ...me,
    ready,
    version,
    authOpen,
    toasts,
    openAuth: () => setAuthOpen(true),
    closeAuth: () => setAuthOpen(false),
    toast,
    refresh,
    pickServer,
    login,
    logout,
    kickSession,
    kickOtherSessions,
    buyLot,
    placeBid,
    toggleWatch,
    topUp,
    sellItem,
    withdrawItem,
  }
}

type Store = ReturnType<typeof useStoreValue>
const Ctx = createContext<Store | null>(null)

export function StoreProvider({ children }: { children: ReactNode }) {
  const value = useStoreValue()
  return <Ctx.Provider value={value}>{children}</Ctx.Provider>
}

export function useStore() {
  const s = useContext(Ctx)
  if (!s) throw new Error('useStore outside provider')
  return s
}

// если не вошёл, сначала открываем вход
export function useRequireAuth() {
  const { user, openAuth } = useStore()
  return (fn: () => void) => (user ? fn() : openAuth())
}
