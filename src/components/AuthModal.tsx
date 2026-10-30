import { useEffect, useRef, useState, type ReactNode } from 'react'
import { Link } from 'react-router'
import { AnimatePresence, motion, useAnimate } from 'motion/react'
import { ArrowLeftIcon, CaretDownIcon, CheckIcon, CircleNotchIcon } from '@phosphor-icons/react'
import clsx from 'clsx'
import { COUNTRIES, type Server } from '../lib/data'
import { api, ApiError, errorText, getServer } from '../lib/api'
import { useServers } from '../lib/market'
import { deviceInfo, useStore } from '../lib/store'
import { fmt } from '../lib/format'
import { Button, Modal } from './ui'

const CODE_LEN = 5

const applyMask = (digits: string, mask: string) => {
  let out = ''
  let i = 0
  for (const ch of mask) {
    if (i >= digits.length) break
    if (ch === 'X') out += digits[i++]
    else out += ch
  }
  return out
}

type Step = 'phone' | 'code' | 'done'

const sendCode = (server: string, phone: string) => api('/api/auth/code', { body: { server, phone }, auth: false })

export function AuthModal() {
  const { authOpen, closeAuth, login, toast, pickServer } = useStore()
  const { data: servers } = useServers()
  const [step, setStep] = useState<Step>('phone')
  const [dir, setDir] = useState(1)
  const [country, setCountry] = useState(COUNTRIES[0])
  const [phone, setPhone] = useState('')
  const [server, setServer] = useState<Server | null>(null)

  useEffect(() => {
    if (!authOpen) return
    setStep('phone')
    setDir(1)
  }, [authOpen])

  // по умолчанию последняя выбранная приватка
  useEffect(() => {
    if (server || !servers?.length) return
    setServer(servers.find((s) => s.slug === getServer()) ?? servers[0])
  }, [servers, server])

  const go = (s: Step, d: number) => {
    setDir(d)
    setStep(s)
  }

  const fullPhone = `+${country.code} ${applyMask(phone, country.mask)}`

  return (
    <Modal open={authOpen} onClose={closeAuth}>
      <div className="relative overflow-hidden px-6 pt-10 pb-7 sm:px-8">
        <AnimatePresence mode="wait" initial={false} custom={dir}>
          <motion.div
            key={step}
            custom={dir}
            variants={{
              enter: (d: number) => ({ opacity: 0, x: d * 24 }),
              center: { opacity: 1, x: 0 },
              exit: (d: number) => ({ opacity: 0, x: d * -24 }),
            }}
            initial="enter"
            animate="center"
            exit="exit"
            transition={{ duration: 0.22, ease: [0.16, 1, 0.3, 1] }}
          >
            {step === 'phone' && (
              <PhoneStep
                servers={servers ?? []}
                server={server}
                setServer={(s) => {
                  setServer(s)
                  pickServer(s.slug)
                }}
                country={country}
                setCountry={setCountry}
                phone={phone}
                setPhone={setPhone}
                onSubmit={async () => {
                  if (!server) return 'Выберите сервер'
                  try {
                    await sendCode(server.slug, fullPhone)
                    go('code', 1)
                    return null
                  } catch (e) {
                    if (e instanceof ApiError && e.code === 'not_found') return `На «${server.name}» нет аккаунта с этим номером`
                    return errorText(e)
                  }
                }}
              />
            )}
            {step === 'code' && server && (
              <CodeStep
                phone={fullPhone}
                server={server}
                onBack={() => go('phone', -1)}
                onResend={() => sendCode(server.slug, fullPhone).catch((e) => toast(errorText(e), 'error'))}
                onDone={async (token, bonus) => {
                  await login(token)
                  go('done', 1)
                  setTimeout(() => {
                    closeAuth()
                    toast(bonus > 0 ? `Вы вошли · начислено ${fmt(bonus)} GRAM` : 'Вы вошли', 'success')
                  }, 1300)
                }}
              />
            )}
            {step === 'done' && <DoneStep />}
          </motion.div>
        </AnimatePresence>
      </div>
    </Modal>
  )
}

function Header({ title, text }: { title: string; text: ReactNode }) {
  return (
    <div className="mb-7 flex flex-col items-center text-center">
      <motion.img
        src="/logo-mark.png"
        alt=""
        className="mb-5 size-16"
        initial={{ scale: 0.8, opacity: 0 }}
        animate={{ scale: 1, opacity: 1 }}
        transition={{ type: 'spring', bounce: 0.4, duration: 0.6 }}
      />
      <h2 className="text-xl font-bold tracking-tight">{title}</h2>
      <p className="mt-2 max-w-72 text-sm leading-relaxed text-muted">{text}</p>
    </div>
  )
}

// выпадашка как у выбора страны
function Picker<T>({ label, value, items, keyOf, render, onPick }: { label: string; value: string; items: T[]; keyOf: (t: T) => string; render: (t: T) => ReactNode; onPick: (t: T) => void }) {
  const [open, setOpen] = useState(false)
  return (
    <div className="relative mb-3">
      <button onClick={() => setOpen((o) => !o)} className="flex h-13 w-full items-center justify-between rounded-2xl bg-black/20 ring-1 ring-white/8 px-4 text-left transition-colors hover:bg-black/25">
        <span>
          <span className="block text-[11px] font-medium text-subtle">{label}</span>
          <span className="text-[15px] font-medium">{value}</span>
        </span>
        <CaretDownIcon className={clsx('size-4 text-muted transition-transform', open && 'rotate-180')} />
      </button>
      <AnimatePresence>
        {open && (
          <motion.div
            initial={{ opacity: 0, y: -4 }}
            animate={{ opacity: 1, y: 0 }}
            exit={{ opacity: 0, y: -4 }}
            transition={{ duration: 0.14 }}
            className="glass absolute inset-x-0 top-full z-10 mt-2 max-h-60 overflow-y-auto rounded-[20px] p-1.5"
          >
            {items.map((t) => (
              <button
                key={keyOf(t)}
                onClick={() => {
                  onPick(t)
                  setOpen(false)
                }}
                className="flex w-full items-center justify-between rounded-[14px] px-3 py-2.5 text-left text-sm hover:bg-white/8"
              >
                {render(t)}
              </button>
            ))}
          </motion.div>
        )}
      </AnimatePresence>
    </div>
  )
}

function PhoneStep({
  servers,
  server,
  setServer,
  country,
  setCountry,
  phone,
  setPhone,
  onSubmit,
}: {
  servers: Server[]
  server: Server | null
  setServer: (s: Server) => void
  country: (typeof COUNTRIES)[number]
  setCountry: (c: (typeof COUNTRIES)[number]) => void
  phone: string
  setPhone: (p: string) => void
  onSubmit: () => Promise<string | null>
}) {
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const inputRef = useRef<HTMLInputElement>(null)
  const maxLen = country.mask.split('X').length - 1
  const valid = phone.length === maxLen && !!server

  useEffect(() => {
    const t = setTimeout(() => inputRef.current?.focus(), 250)
    return () => clearTimeout(t)
  }, [])

  const submit = async () => {
    if (!valid || loading) return
    setLoading(true)
    setError(null)
    const err = await onSubmit()
    setLoading(false)
    setError(err)
  }

  if (servers.length === 0) {
    return (
      <div>
        <Header title="Вход в LunaX" text="Пока не привязано ни одной приватки. Подключите свой сервер, и вход заработает." />
        <Link to="/connect">
          <Button size="lg" className="w-full">
            Подключить сервер
          </Button>
        </Link>
      </div>
    )
  }

  return (
    <div>
      <Header title="Вход в LunaX" text="Выберите свою приватку и номер, код придёт туда личным сообщением" />

      <Picker
        label="Сервер"
        value={server?.name ?? 'Выберите'}
        items={servers}
        keyOf={(s) => s.slug}
        render={(s) => <span className="font-medium">{s.name}</span>}
        onPick={setServer}
      />

      <Picker
        label="Страна"
        value={country.name}
        items={COUNTRIES}
        keyOf={(c) => c.name}
        render={(c) => (
          <>
            <span className="font-medium">{c.name}</span>
            <span className="tabular text-muted">+{c.code}</span>
          </>
        )}
        onPick={(c) => {
          setCountry(c)
          setPhone('')
          inputRef.current?.focus()
        }}
      />

      <label className="flex h-13 items-center rounded-2xl bg-black/20 ring-1 ring-white/8 px-4 transition-colors focus-within:ring-accent/70">
        <span className="mr-2 text-[15px] font-medium text-muted tabular">+{country.code}</span>
        <input
          ref={inputRef}
          inputMode="numeric"
          autoComplete="tel-national"
          value={applyMask(phone, country.mask)}
          placeholder={country.mask.replace(/X/g, '0')}
          onChange={(e) => {
            setError(null)
            setPhone(e.target.value.replace(/\D/g, '').slice(0, maxLen))
          }}
          onKeyDown={(e) => e.key === 'Enter' && submit()}
          className="h-full flex-1 bg-transparent text-[15px] font-medium tracking-wide outline-none tabular placeholder:text-subtle/60"
        />
      </label>

      <div className="mt-3 min-h-5 text-center text-[13px] text-danger">{error}</div>

      <Button size="lg" className="mt-3 w-full" disabled={!valid} onClick={submit}>
        {loading ? <CircleNotchIcon className="size-5 animate-spin" /> : 'Получить код'}
      </Button>

      <p className="mt-4 text-center text-[11px] leading-relaxed text-subtle">
        Своя приватка? <Link to="/connect" className="font-semibold text-link">Подключите её к LunaX</Link>
      </p>
    </div>
  )
}

function CodeStep({
  phone,
  server,
  onBack,
  onResend,
  onDone,
}: {
  phone: string
  server: Server
  onBack: () => void
  onResend: () => void
  onDone: (token: string, bonus: number) => void
}) {
  const [code, setCode] = useState('')
  const [error, setError] = useState<string | null>(null)
  const [checking, setChecking] = useState(false)
  const [left, setLeft] = useState(59)
  const inputRef = useRef<HTMLInputElement>(null)
  const [scope, animate] = useAnimate()

  useEffect(() => {
    const t = setTimeout(() => inputRef.current?.focus(), 250)
    return () => clearTimeout(t)
  }, [])

  useEffect(() => {
    if (left <= 0) return
    const id = setTimeout(() => setLeft((l) => l - 1), 1000)
    return () => clearTimeout(id)
  }, [left])

  const update = (v: string) => {
    const digits = v.replace(/\D/g, '').slice(0, CODE_LEN)
    setError(null)
    setCode(digits)
    if (digits.length === CODE_LEN) verify(digits)
  }

  const verify = async (value: string) => {
    setChecking(true)
    try {
      const res = await api<{ token: string; bonus: number }>('/api/auth/verify', {
        body: { server: server.slug, phone, code: value, ...deviceInfo() },
        auth: false,
      })
      onDone(res.token, res.bonus)
    } catch (e) {
      setError(e instanceof ApiError && e.code === 'invalid' ? 'Неверный или просроченный код' : errorText(e))
      animate(scope.current, { x: [0, -10, 10, -7, 7, -3, 0] }, { duration: 0.45 })
      setTimeout(() => setCode(''), 450)
    } finally {
      setChecking(false)
    }
  }

  return (
    <div>
      <button onClick={onBack} className="absolute top-0 left-0 grid size-8 place-items-center rounded-full text-muted transition-colors hover:bg-white/8 hover:text-fg" aria-label="Назад">
        <ArrowLeftIcon className="size-[18px]" />
      </button>

      <Header
        title={phone}
        text={
          <>
            Мы отправили код личным сообщением на <span className="font-semibold text-fg">{server.name}</span>. Введите его ниже.
          </>
        }
      />

      <div ref={scope} className="relative mx-auto flex max-w-72 justify-between gap-2" onClick={() => inputRef.current?.focus()}>
        {Array.from({ length: CODE_LEN }, (_, i) => {
          const ch = code[i]
          const active = i === code.length && !checking
          return (
            <div
              key={i}
              className={clsx(
                'relative grid h-14 flex-1 place-items-center rounded-2xl bg-black/20 text-xl font-semibold tabular ring-1 transition-[box-shadow] duration-150',
                error ? 'ring-danger/70' : active ? 'ring-2 ring-accent' : ch ? 'ring-white/20' : 'ring-white/8',
              )}
            >
              <AnimatePresence>
                {ch && (
                  <motion.span initial={{ opacity: 0, y: 8, scale: 0.6 }} animate={{ opacity: 1, y: 0, scale: 1 }} exit={{ opacity: 0, scale: 0.6 }} transition={{ type: 'spring', bounce: 0.4, duration: 0.35 }}>
                    {ch}
                  </motion.span>
                )}
              </AnimatePresence>
              {active && <span className="absolute h-6 w-[2px] animate-pulse rounded-full bg-accent" />}
            </div>
          )
        })}
        <input
          ref={inputRef}
          value={code}
          onChange={(e) => update(e.target.value)}
          inputMode="numeric"
          autoComplete="one-time-code"
          className="absolute inset-0 opacity-0"
          aria-label="Код подтверждения"
          disabled={checking}
        />
      </div>

      <div className="mt-4 h-5 text-center text-[13px]">
        {checking ? (
          <span className="inline-flex items-center gap-2 text-muted">
            <CircleNotchIcon className="size-4 animate-spin" /> Проверяем код
          </span>
        ) : error ? (
          <span className="text-danger">{error}</span>
        ) : left > 0 ? (
          <span className="text-subtle tabular">Отправить повторно через 0:{String(left).padStart(2, '0')}</span>
        ) : (
          <button
            onClick={() => {
              setLeft(59)
              onResend()
            }}
            className="font-semibold text-accent hover:text-accent-hover"
          >
            Отправить код повторно
          </button>
        )}
      </div>
    </div>
  )
}

function DoneStep() {
  return (
    <div className="flex flex-col items-center py-10 text-center">
      <motion.div
        initial={{ scale: 0 }}
        animate={{ scale: 1 }}
        transition={{ type: 'spring', bounce: 0.5, duration: 0.6 }}
        className="mb-5 grid size-16 place-items-center rounded-full bg-success/15 text-success"
      >
        <motion.span initial={{ scale: 0, rotate: -30 }} animate={{ scale: 1, rotate: 0 }} transition={{ delay: 0.15, type: 'spring', bounce: 0.5 }}>
          <CheckIcon className="size-8" />
        </motion.span>
      </motion.div>
      <h2 className="text-xl font-bold">Готово</h2>
      <p className="mt-2 text-sm text-muted">Добро пожаловать в LunaX</p>
    </div>
  )
}
