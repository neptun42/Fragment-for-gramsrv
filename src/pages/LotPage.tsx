import { useEffect, useRef, useState, type ReactNode } from 'react'
import { Link, Navigate, useParams } from 'react-router'
import { AnimatePresence, motion } from 'motion/react'
import { AtIcon, BellIcon, BellRingingIcon, CaretLeftIcon, CheckIcon, GavelIcon, PhoneIcon, QuestionIcon } from '@phosphor-icons/react'
import clsx from 'clsx'
import type { Lot } from '../lib/data'
import { useLot } from '../lib/market'
import { fmt, fmtDate, fmtUsd, useNow } from '../lib/format'
import { useRequireAuth, useStore } from '../lib/store'
import { Badge, Button, Countdown, Gram, GramIcon, Modal, Stagger } from '../components/ui'
import { lotTitle } from './Lots'

export function LotPage({ kind }: { kind: Lot['kind'] }) {
  const { slug = '' } = useParams()
  const { data: lot, loading } = useLot(kind, slug)
  if (loading && !lot) return null
  if (!lot) return <Navigate to={kind === 'username' ? '/usernames' : '/numbers'} replace />
  return <LotView lot={lot} />
}

function LotView({ lot }: { lot: Lot }) {
  const { bids, watching, toggleWatch, toast, buyLot } = useStore()
  const requireAuth = useRequireAuth()
  const now = useNow(1000)
  const [bidOpen, setBidOpen] = useState(false)
  const [help, setHelp] = useState(false)
  const mine = bids[lot.id]
  const highest = lot.price
  const minBid = lot.minBid
  const step = minBid - highest
  const watched = watching.includes(lot.id)
  const back = lot.kind === 'username' ? { to: '/usernames', label: 'Юзернеймы' } : { to: '/numbers', label: 'Номера' }

  const history = lot.bids

  const info =
    lot.kind === 'username'
      ? [
          { label: 'Юзернейм Telegram', value: `@${lot.name}` },
          { label: 'Веб-адрес', value: `t.me/${lot.name}` },
          { label: 'Адрес LunaX', value: `${lot.name}.lunax.me` },
        ]
      : [
          { label: 'Номер', value: lot.name },
          { label: 'Тип', value: 'Анонимный, без SIM' },
          { label: 'Вход', value: 'Код приходит в LunaX' },
        ]

  const buy = () =>
    requireAuth(() => {
      buyLot(lot.id).then((item) => item && toast(`${lotTitle(lot)} теперь ваш`, 'success'))
    })

  return (
    <div className="mx-auto max-w-[760px]">
      <Link to={back.to} className="mb-5 inline-flex items-center gap-1 text-[14px] font-semibold text-link transition-opacity hover:opacity-80">
        <CaretLeftIcon className="size-4" />
        {back.label}
      </Link>

      <Stagger i={0} className="mb-6 flex flex-wrap items-center justify-between gap-3">
        <div className="flex min-w-0 items-center gap-3">
          <span className="grid size-12 bg-accent/15 shrink-0 place-items-center rounded-[16px] text-link">{lot.kind === 'username' ? <AtIcon className="size-5" /> : <PhoneIcon className="size-5" />}</span>
          <h1 className="truncate text-[26px] font-bold tracking-tight tabular sm:text-[30px]">{lot.kind === 'username' ? `${lot.name}.t.me` : lot.name}</h1>
          {lot.status === 'auction' && <Badge tone="success">На аукционе</Badge>}
          {lot.status === 'sale' && <Badge tone="accent">Продаётся</Badge>}
          {lot.status === 'sold' && <Badge>Продан</Badge>}
        </div>
        {lot.status === 'auction' && (
          <button
            onClick={() =>
              requireAuth(() => {
                toggleWatch(lot.id).then((res) => res && toast(res.watching ? 'Уведомим о новых ставках' : 'Вы отписались от обновлений', 'success'))
              })
            }
            className="flex items-center gap-2 text-[14px] font-semibold text-link transition-opacity hover:opacity-80"
          >
            <motion.span key={String(watched)} initial={{ rotate: -25, scale: 0.6 }} animate={{ rotate: 0, scale: 1 }} transition={{ type: 'spring', bounce: 0.6 }}>
              {watched ? <BellRingingIcon className="size-4" /> : <BellIcon className="size-4" />}
            </motion.span>
            {watched ? 'Вы подписаны' : 'Подписаться на обновления'}
          </button>
        )}
      </Stagger>

      <div className="grid gap-3 md:grid-cols-2">
        <Stagger i={1} className="card rounded-[24px] p-5">
          {lot.status === 'auction' ? (
            <div className="grid grid-cols-3 gap-3">
              <Stat label="Высшая ставка" value={highest} usd />
              <Stat label="Шаг ставки" value={step} sub="5%" />
              <Stat label="Мин. ставка" value={minBid} usd />
            </div>
          ) : (
            <div className="grid grid-cols-2 gap-3">
              <Stat label={lot.status === 'sale' ? 'Цена' : 'Продан за'} value={lot.price} usd />
              <div>
                <p className="text-[13px] font-semibold text-muted">{lot.status === 'sale' ? 'Тип продажи' : 'Дата продажи'}</p>
                <p className="mt-2 text-[16px] font-bold">{lot.status === 'sale' ? 'Фикс. цена' : fmtDate(lot.endsAt)}</p>
              </div>
            </div>
          )}
          <button onClick={() => setHelp((h) => !h)} className="mt-4 inline-flex items-center gap-1.5 text-[13px] font-semibold text-link">
            <QuestionIcon className="size-3.5" /> Как это работает?
          </button>
          <AnimatePresence initial={false}>
            {help && (
              <motion.p
                initial={{ height: 0, opacity: 0 }}
                animate={{ height: 'auto', opacity: 1 }}
                exit={{ height: 0, opacity: 0 }}
                transition={{ type: 'spring', bounce: 0, duration: 0.4 }}
                className="overflow-hidden text-[13px] leading-relaxed text-muted"
              >
                <span className="block pt-2">
                  Каждая новая ставка должна быть выше текущей минимум на шаг. Если вашу ставку перебьют, GRAM вернутся на баланс. Когда таймер закончится, лот достанется лидеру.
                </span>
              </motion.p>
            )}
          </AnimatePresence>
        </Stagger>

        <Stagger i={2} className="card overflow-hidden rounded-[24px]">
          {info.map((r, i) => (
            <div key={r.label} className={clsx('flex items-center justify-between gap-4 px-5 py-[15px] text-[14px]', i > 0 && 'border-t border-white/5')}>
              <span className="font-semibold">{r.label}</span>
              <span className="truncate font-medium text-link tabular">{r.value}</span>
            </div>
          ))}
        </Stagger>
      </div>

      <Stagger i={3} className="mt-3 grid items-center gap-4 md:grid-cols-2">
        {lot.status === 'auction' && (
          <>
            <Button size="lg" className="w-full" onClick={() => requireAuth(() => setBidOpen(true))}>
              <GavelIcon weight="fill" className="size-[18px]" /> {mine ? 'Повысить ставку' : 'Сделать ставку'}
            </Button>
            <div className="flex items-center justify-center gap-3 md:justify-end">
              <span className="text-[14px] font-medium whitespace-nowrap text-muted">Завершится через</span>
              <Countdown ms={lot.endsAt - now} />
            </div>
          </>
        )}
        {lot.status === 'sale' && (
          <Button size="lg" className="w-full md:col-span-2" onClick={buy}>
            Купить за <Gram value={lot.price} size="size-4" />
          </Button>
        )}
      </Stagger>

      {history.length > 0 && (
        <Stagger i={4} className="mt-10">
          <h2 className="mb-3 px-1 text-[22px] font-bold tracking-tight">История ставок</h2>
          <div className="card overflow-hidden rounded-[24px]">
            <div className="grid grid-cols-[1fr_1.2fr_1fr] gap-4 bg-elevated/70 px-5 py-3 text-[13px] font-semibold text-muted">
              <span>Ставка</span>
              <span>Дата</span>
              <span>От кого</span>
            </div>
            <AnimatePresence initial={false}>
              {history.map((b) => (
                <motion.div
                  key={`${b.user}-${b.amount}-${b.ts}`}
                  layout
                  initial={{ opacity: 0, backgroundColor: 'rgba(46,155,255,0.18)' }}
                  animate={{ opacity: 1, backgroundColor: 'rgba(46,155,255,0)' }}
                  transition={{ duration: 1.2 }}
                  className="grid grid-cols-[1fr_1.2fr_1fr] items-center gap-4 border-t border-white/5 px-5 py-3.5 text-[14px]"
                >
                  <Gram value={b.amount} className="font-bold" />
                  <span className="text-muted">{fmtDate(b.ts)}</span>
                  <span className={clsx('truncate font-medium', b.user === 'Вы' ? 'text-success' : 'text-link')}>{b.user}</span>
                </motion.div>
              ))}
            </AnimatePresence>
          </div>
        </Stagger>
      )}

      <BidSheet open={bidOpen} onClose={() => setBidOpen(false)} lot={lot} minBid={minBid} step={step} />
    </div>
  )
}

function Stat({ label, value, usd, sub }: { label: string; value: number; usd?: boolean; sub?: string }) {
  return (
    <div className="min-w-0">
      <p className="text-[13px] font-semibold text-muted">{label}</p>
      <Gram value={value} usd={usd} className="mt-2 text-[16px] font-bold" size="size-4" />
      {sub && <p className="mt-0.5 text-[11px] font-medium text-subtle">{sub}</p>}
    </div>
  )
}

function BidSheet({ open, onClose, lot, minBid, step }: { open: boolean; onClose: () => void; lot: Lot; minBid: number; step: number }) {
  const { balance, bids, placeBid, toast } = useStore()
  const [value, setValue] = useState(String(minBid))
  const [done, setDone] = useState(false)
  const inputRef = useRef<HTMLInputElement>(null)
  const prev = bids[lot.id]?.amount ?? 0
  const amount = Number(value) || 0
  const charge = Math.max(0, amount - prev)
  const tooLow = amount < minBid
  const tooHigh = balance + prev < amount

  const minRef = useRef(minBid)
  minRef.current = minBid

  // сбрасываем только при открытии, minBid меняется сразу после ставки
  useEffect(() => {
    if (!open) return
    setValue(String(minRef.current))
    setDone(false)
    const t = setTimeout(() => inputRef.current?.select(), 350)
    return () => clearTimeout(t)
  }, [open])

  const chips = [
    { label: 'Мин.', v: minBid },
    { label: '+10%', v: Math.round(minBid * 1.1) },
    { label: '+25%', v: Math.round(minBid * 1.25) },
    { label: '+50%', v: Math.round(minBid * 1.5) },
  ]

  const [sending, setSending] = useState(false)

  const submit = async () => {
    if (tooLow || tooHigh || sending) return
    setSending(true)
    const ok = await placeBid(lot.id, amount)
    setSending(false)
    if (!ok) return
    setDone(true)
    setTimeout(() => {
      onClose()
      toast(`Ставка ${fmt(amount)} GRAM принята`, 'success')
    }, 1100)
  }

  return (
    <Modal open={open} onClose={onClose}>
      <div className="px-6 pt-8 pb-6">
        <AnimatePresence mode="popLayout" initial={false}>
          {done ? (
            <motion.div key="done" initial={{ opacity: 0, scale: 0.9 }} animate={{ opacity: 1, scale: 1 }} className="flex flex-col items-center py-12 text-center">
              <motion.span initial={{ scale: 0 }} animate={{ scale: 1 }} transition={{ type: 'spring', bounce: 0.55, duration: 0.6 }} className="mb-5 grid size-16 place-items-center rounded-full bg-success/15 text-success">
                <CheckIcon className="size-8" />
              </motion.span>
              <p className="text-[20px] font-bold">Ставка принята</p>
              <p className="mt-1 text-sm text-muted">Вы лидируете в аукционе {lotTitle(lot)}</p>
            </motion.div>
          ) : (
            <motion.div key="form" exit={{ opacity: 0 }}>
              <p className="text-center text-[13px] font-semibold text-muted">Ставка на</p>
              <h3 className="mt-1 text-center text-[22px] font-bold tabular">{lotTitle(lot)}</h3>

              <label className={clsx('mt-6 flex h-[72px] items-center justify-center gap-3 rounded-[22px] bg-black/20 px-5 ring-1 transition-[box-shadow]', tooLow || tooHigh ? 'ring-danger/50' : 'ring-white/8 focus-within:ring-accent/70')}>
                <GramIcon className="size-8" />
                <input
                  ref={inputRef}
                  value={value}
                  inputMode="numeric"
                  onChange={(e) => setValue(e.target.value.replace(/\D/g, '').slice(0, 9))}
                  onKeyDown={(e) => e.key === 'Enter' && submit()}
                  className="min-w-0 bg-transparent text-[32px] font-bold tracking-tight outline-none tabular"
                  style={{ width: `${Math.max(2, value.length) + 0.5}ch` }}
                />
              </label>
              <p className="mt-2 text-center text-[12px] font-medium text-subtle tabular">~ {fmtUsd(amount)}</p>

              <div className="mt-4 grid grid-cols-4 gap-2">
                {chips.map((c) => (
                  <button
                    key={c.label}
                    onClick={() => setValue(String(c.v))}
                    className={clsx('h-9 rounded-full text-[13px] font-semibold transition-colors', amount === c.v ? 'bg-accent/20 text-link ring-1 ring-accent/50' : 'bg-white/6 text-muted hover:bg-white/10 hover:text-fg')}
                  >
                    {c.label}
                  </button>
                ))}
              </div>

              <div className="mt-5 overflow-hidden rounded-[20px] bg-black/15 text-[14px] ring-1 ring-white/6">
                <Row label="Минимальная ставка" value={<Gram value={minBid} />} />
                <Row label="Шаг" value={<Gram value={step} />} />
                {prev > 0 && <Row label="Ваша прошлая ставка" value={<Gram value={prev} />} />}
                <Row label="Будет списано" value={<Gram value={charge} className="font-bold" />} />
                <Row label="Баланс" value={<Gram value={balance} className={tooHigh ? 'text-danger' : ''} />} />
              </div>

              <AnimatePresence>
                {(tooLow || tooHigh) && (
                  <motion.p initial={{ opacity: 0, height: 0 }} animate={{ opacity: 1, height: 'auto' }} exit={{ opacity: 0, height: 0 }} className="overflow-hidden text-center text-[13px] font-medium text-danger">
                    <span className="block pt-3">{tooLow ? `Ставка должна быть не меньше ${fmt(minBid)} GRAM` : 'Недостаточно GRAM на балансе'}</span>
                  </motion.p>
                )}
              </AnimatePresence>

              <Button size="lg" className="mt-5 w-full" disabled={tooLow || tooHigh || sending} onClick={submit}>
                Поставить <Gram value={amount} size="size-4" />
              </Button>
            </motion.div>
          )}
        </AnimatePresence>
      </div>
    </Modal>
  )
}

function Row({ label, value }: { label: string; value: ReactNode }) {
  return (
    <div className="flex items-center justify-between border-t border-white/5 px-4 py-3 first:border-0">
      <span className="text-muted">{label}</span>
      <span className="font-semibold">{value}</span>
    </div>
  )
}
