// клиент к go бэкенду, адрес берём из VITE_API_URL или с того же домена
const BASE = (import.meta.env.VITE_API_URL ?? '').replace(/\/$/, '')
const TOKEN_KEY = 'lunax:token'
const SERVER_KEY = 'lunax:server'

export class ApiError extends Error {
  code: string
  status: number
  constructor(status: number, code: string, message: string) {
    super(message)
    this.status = status
    this.code = code
  }
}

const read = (key: string) => {
  try {
    return localStorage.getItem(key)
  } catch {
    return null
  }
}

const write = (key: string, value: string | null) => {
  try {
    if (value === null) localStorage.removeItem(key)
    else localStorage.setItem(key, value)
  } catch {
    // приватный режим, живём без сохранения
  }
}

export const getToken = () => read(TOKEN_KEY)
export const setToken = (t: string | null) => write(TOKEN_KEY, t)
export const getServer = () => read(SERVER_KEY)
export const setServer = (s: string | null) => write(SERVER_KEY, s)

export async function api<T>(path: string, init: { method?: string; body?: unknown; auth?: boolean } = {}): Promise<T> {
  const headers: Record<string, string> = {}
  const token = getToken()
  if (token && init.auth !== false) headers.Authorization = `Bearer ${token}`
  if (init.body !== undefined) headers['Content-Type'] = 'application/json'

  const res = await fetch(BASE + path, {
    method: init.method ?? (init.body !== undefined ? 'POST' : 'GET'),
    headers,
    body: init.body !== undefined ? JSON.stringify(init.body) : undefined,
  })
  if (res.status === 204) return undefined as T
  const data = await res.json().catch(() => ({}))
  if (!res.ok) throw new ApiError(res.status, data.error ?? 'internal', data.message ?? res.statusText)
  return data as T
}

// гостю витрину показываем по выбранной приватке
export const withServer = (path: string) => {
  const s = getServer()
  if (!s || getToken()) return path
  return path + (path.includes('?') ? '&' : '?') + 'server=' + encodeURIComponent(s)
}

// понятный текст ошибки для тоста
export function errorText(e: unknown) {
  if (!(e instanceof ApiError)) return 'Нет связи с сервером'
  switch (e.code) {
    case 'no_funds':
      return 'Недостаточно GRAM на балансе'
    case 'rate_limited':
      return 'Слишком много попыток, подождите пару минут'
    case 'server_unavailable':
      return 'Приватка не отвечает, попробуйте позже'
    case 'unauthorized':
      return 'Сессия завершена, войдите снова'
    case 'conflict':
      return 'Лот уже недоступен'
    case 'forbidden':
      return 'На этом сервере это отключено'
    default:
      return e.message
  }
}
