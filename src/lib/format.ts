import { useEffect, useState } from 'react'

const nf = new Intl.NumberFormat('ru-RU')
export const fmt = (n: number) => nf.format(n)
// примерный курс GRAM в доллары для второй строки цены
export const GRAM_USD = 1.35
const usd = new Intl.NumberFormat('en-US', { style: 'currency', currency: 'USD', maximumFractionDigits: 0 })
export const fmtUsd = (gram: number) => usd.format(gram * GRAM_USD)

const df = new Intl.DateTimeFormat('ru-RU', { day: 'numeric', month: 'short', hour: '2-digit', minute: '2-digit' })
export const fmtDate = (ts: number) => df.format(ts).replace('.', '')

export const fmtLeft = (ms: number) => {
  if (ms <= 0) return 'Завершён'
  const m = Math.floor(ms / 60_000)
  const d = Math.floor(m / 1440)
  const h = Math.floor((m % 1440) / 60)
  const min = m % 60
  if (d > 0) return `${d} д ${h} ч ${min} мин`
  if (h > 0) return `${h} ч ${min} мин`
  return `${min} мин`
}

export const fmtAgo = (ts: number) => {
  const s = Math.max(1, Math.floor((Date.now() - ts) / 1000))
  if (s < 60) return `${s} с назад`
  const m = Math.floor(s / 60)
  if (m < 60) return `${m} мин назад`
  const h = Math.floor(m / 60)
  if (h < 24) return `${h} ч назад`
  return fmtDate(ts)
}

export const useNow = (interval = 30_000) => {
  const [now, setNow] = useState(Date.now())
  useEffect(() => {
    const id = setInterval(() => setNow(Date.now()), interval)
    return () => clearInterval(id)
  }, [interval])
  return now
}

export const plural = (n: number, one: string, few: string, many: string) => {
  const a = Math.abs(n) % 100
  const b = a % 10
  if (a > 10 && a < 20) return many
  if (b > 1 && b < 5) return few
  if (b === 1) return one
  return many
}
