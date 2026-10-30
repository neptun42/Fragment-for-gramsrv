import { useEffect, useState } from 'react'
import { Link, useNavigate, useSearchParams } from 'react-router'
import { motion } from 'motion/react'
import { ArrowDownLeftIcon, ArrowUpRightIcon, ArrowUUpLeftIcon, CopyIcon, DeviceMobileIcon, GavelIcon, TrophyIcon, MonitorIcon, PackageIcon, SignInIcon, SignOutIcon, TelegramLogoIcon } from '@phosphor-icons/react'
import clsx from 'clsx'
import { fmt, fmtAgo, fmtDate, fmtUsd } from '../lib/format'
import { itemName, useStore, type InvItem } from '../lib/store'
import { Button, Empty, Gram, GramIcon, Modal, Segmented, softSpring, Stagger } from '../components/ui'
import { ItemArt } from '../components/art'
import { useServers } from '../lib/market'

function SignInGate({ text }: { text: string }) {
  const { openAuth } = useStore()
  return (
    <div className="mx-auto max-w-md pt-6">
      <Empty
        icon={<SignInIcon className="size-6" />}
        title="Требуется вход"
        text={text}
        action={
          <Button onClick={openAuth}>
            <TelegramLogoIcon className="size-4" /> Войти
          </Button>
        }
      />
    </div>
  )
}

type Tab = 'all' | 'gift' | 'number' | 'username' | 'stars'

export function InventoryPage() {
  const { user, inventory } = useStore()
  const [tab, setTab] = useState<Tab>('all')
  const [selected, setSelected] = useState<InvItem | null>(null)

  if (!user) return <SignInGate text="Войдите, чтобы увидеть свои подарки, номера и юзернеймы" />

  const items = inventory.filter((i) => tab === 'all' || i.kind === tab)
  const total = inventory.reduce((s, i) => s + i.value, 0)

  return (
    <div>
      <div className="mb-6 flex flex-wrap items-end justify-between gap-4">
        <div>
          <h1 className="text-[30px] font-bold tracking-tight">Инвентарь</h1>
          <p className="mt-1 flex items-center gap-1.5 text-sm text-muted">
            {inventory.length} предметов · оценка <Gram value={Math.round(total * 10) / 10} className="font-semibold text-fg" size="size-3" />
          </p>
        </div>
        <div className="no-scrollbar -mx-4 w-[calc(100%+2rem)] overflow-x-auto px-4 sm:mx-0 sm:w-auto sm:px-0">
          <Segmented
            value={tab}
            onChange={setTab}
            options={[
              { value: 'all', label: 'Все' },
              { value: 'gift', label: 'Подарки' },
              { value: 'number', label: 'Номера' },
              { value: 'username', label: 'Юзернеймы' },
              { value: 'stars', label: 'Звёзды' },
            ]}
          />
        </div>
      </div>

      {items.length === 0 ? (
        <Empty
          icon={<PackageIcon className="size-6" />}
          title="Здесь пока пусто"
          text="Покупайте юзернеймы, номера и подарки на маркете"
          action={
            <Link to="/usernames">
              <Button>На маркет</Button>
            </Link>
          }
        />
      ) : (
        <motion.div key={tab} initial="hidden" animate="show" variants={{ show: { transition: { staggerChildren: 0.03 } } }} className="grid grid-cols-2 gap-3 sm:grid-cols-3 lg:grid-cols-5">
          {items.map((item) => (
            <motion.button
              key={item.id}
              variants={{ hidden: { opacity: 0, y: 14, scale: 0.96 }, show: { opacity: 1, y: 0, scale: 1, transition: softSpring } }}
              whileTap={{ scale: 0.97 }}
              onClick={() => setSelected(item)}
              className="card group overflow-hidden rounded-[22px] text-left"
            >
              <ItemArt item={item} className="transition-transform duration-700 group-hover:scale-[1.05]" />
              <div className="p-3">
                <p className="truncate text-[14px] font-semibold tabular">{itemName(item)}</p>
                <div className="mt-1 flex items-center justify-between text-xs">
                  <Gram value={item.value} className="font-semibold" size="size-3" />
                  <span className="text-subtle">{item.source}</span>
                </div>
              </div>
            </motion.button>
          ))}
        </motion.div>
      )}

      <Modal open={!!selected} onClose={() => setSelected(null)}>
        {selected && <ItemDetails item={selected} onClose={() => setSelected(null)} />}
      </Modal>
    </div>
  )
}

function ItemDetails({ item, onClose }: { item: InvItem; onClose: () => void }) {
  const { sellItem, withdrawItem, toast } = useStore()
  const [busy, setBusy] = useState(false)

  const act = async (fn: () => Promise<boolean>) => {
    setBusy(true)
    const ok = await fn()
    setBusy(false)
    if (ok) onClose()
  }
  return (
    <div className="p-5 pt-12">
      <div className="mx-auto w-60 overflow-hidden rounded-[30px] shadow-[0_24px_60px_-20px_rgba(0,0,0,0.7)] ring-1 ring-white/12">
        <ItemArt item={item} big />
      </div>
      <div className="pt-6 text-center">
        <h3 className="text-[22px] font-bold tabular">{itemName(item)}</h3>
        <p className="mt-1 text-sm text-muted">
          Получен {fmtDate(item.ts)} · {item.source}
        </p>
        <div className="mt-6 grid grid-cols-2 gap-2">
          <Button
            variant="glass"
            size="lg"
            disabled={busy}
            onClick={() =>
              act(async () => {
                const res = await sellItem(item.id)
                if (res) toast(`Продано за ${fmt(res.credited)} GRAM`, 'success')
                return res !== null
              })
            }
          >
            Продать · <Gram value={item.value} />
          </Button>
          <Button
            size="lg"
            disabled={busy}
            onClick={() =>
              act(async () => {
                const ok = await withdrawItem(item.id)
                if (ok) toast('Выдано на ваш аккаунт в приватке', 'success')
                return ok
              })
            }
          >
            <ArrowUpRightIcon className="size-4" /> Вывести
          </Button>
        </div>
      </div>
    </div>
  )
}

const HISTORY_ICON = { win: TrophyIcon, buy: PackageIcon, sell: ArrowDownLeftIcon, topup: ArrowDownLeftIcon, bid: GavelIcon, refund: ArrowUUpLeftIcon, withdraw: ArrowUpRightIcon }
const TOPUP = [100, 500, 1000, 5000]

export function ProfilePage() {
  const { user, balance, inventory, history, sessions, currentSessionId, kickSession, kickOtherSessions, logout, toast } = useStore()
  const navigate = useNavigate()
  const [params, setParams] = useSearchParams()
  const [topup, setTopup] = useState(false)

  useEffect(() => {
    if (params.get('topup')) {
      setTopup(true)
      setParams({}, { replace: true })
    }
  }, [params, setParams])

  if (!user) return <SignInGate text="Войдите, чтобы открыть профиль и баланс" />

  const orderedSessions = [...sessions].sort((a, b) => {
    if (a.id === currentSessionId) return -1
    if (b.id === currentSessionId) return 1
    return b.lastSeen - a.lastSeen
  })

  const stats = [
    { label: 'Предметов', value: fmt(inventory.length) },
    { label: 'Операций', value: fmt(history.length) },
    { label: 'Сеансов', value: fmt(sessions.length) },
  ]

  return (
    <div className="mx-auto max-w-[640px] space-y-3">
      <Stagger i={0} className="flex flex-col items-center pb-4 text-center">
        <span className="grid size-24 place-items-center rounded-full bg-accent text-4xl font-bold">
          {user.username[0].toUpperCase()}
        </span>
        <h1 className="mt-4 text-[26px] font-bold tracking-tight">{user.name}</h1>
        <button
          onClick={() => {
            navigator.clipboard?.writeText(`@${user.username}`).catch(() => {})
            toast('Юзернейм скопирован')
          }}
          className="mt-0.5 flex items-center gap-1.5 text-[15px] text-link"
        >
          @{user.username} <CopyIcon className="size-3.5" />
        </button>
        <p className="mt-1 text-xs text-subtle tabular">{user.phone.replace(/\d(?=(?:\D*\d){4})/g, '•')}</p>
      </Stagger>

      <Stagger i={1} className="card relative overflow-hidden rounded-3xl p-6">
        <p className="relative text-[13px] font-semibold text-muted">Баланс</p>
        <div className="relative mt-1 flex items-end gap-3">
          <Gram value={balance} className="text-[38px] leading-none font-bold tracking-tight" size="size-8" />
        </div>
        <p className="relative mt-1.5 text-[13px] text-subtle tabular">~ {fmtUsd(balance)}</p>
        <div className="relative mt-5 flex gap-2">
          <Button onClick={() => setTopup(true)}>
            <ArrowDownLeftIcon className="size-4" /> Пополнить
          </Button>
          <Button variant="glass" onClick={() => navigate('/inventory')}>
            <PackageIcon className="size-4" /> Инвентарь
          </Button>
        </div>
      </Stagger>

      <Stagger i={2} className="grid grid-cols-3 gap-3">
        {stats.map((s) => (
          <div key={s.label} className="card rounded-[22px] px-4 py-3.5">
            <p className="text-[22px] font-bold tabular">{s.value}</p>
            <p className="text-xs text-muted">{s.label}</p>
          </div>
        ))}
      </Stagger>

      <Stagger i={3} className="card overflow-hidden rounded-[26px]">
        <h2 className="px-5 pt-5 pb-2 text-[15px] font-semibold">История операций</h2>
        {history.length === 0 ? (
          <p className="px-5 pb-5 text-sm text-muted">Операций пока нет</p>
        ) : (
          <div className="pb-2">
            {history.map((h) => {
              const Icon = HISTORY_ICON[h.kind]
              return (
                <div key={h.id} className="flex items-center gap-3 px-5 py-2.5">
                  <span className="grid size-9 shrink-0 place-items-center rounded-full bg-white/6 text-muted">
                    <Icon className="size-4" />
                  </span>
                  <div className="min-w-0 flex-1">
                    <p className="truncate text-[14px] font-medium">{h.title}</p>
                    <p className="text-xs text-subtle">{fmtAgo(h.ts)}</p>
                  </div>
                  {h.amount !== 0 && (
                    <span className={clsx('shrink-0 text-[14px] font-semibold tabular', h.amount > 0 ? 'text-success' : 'text-fg')}>
                      {h.amount > 0 ? '+' : '−'}
                      {fmt(Math.abs(h.amount))}
                    </span>
                  )}
                </div>
              )
            })}
          </div>
        )}
      </Stagger>

      <Stagger i={4} className="card overflow-hidden rounded-[26px]">
        <div className="flex items-center justify-between gap-3 px-5 pt-5 pb-3">
          <h2 className="text-[15px] font-semibold">Активные сеансы</h2>
          {orderedSessions.length > 1 && (
            <button
              onClick={() => {
                kickOtherSessions().then((r) => r !== null && toast('Другие сеансы завершены', 'success'))
              }}
              className="text-xs font-semibold text-danger transition-colors hover:text-danger/80"
            >
              Завершить другие
            </button>
          )}
        </div>
        {orderedSessions.length === 0 ? (
          <p className="px-5 pb-5 text-sm text-muted">Нет активных сеансов</p>
        ) : (
          orderedSessions.map((s) => {
            const current = s.id === currentSessionId
            const mobile = /Android|iPhone|iPad|iPod|Mobile/i.test(s.device)
            return (
              <div key={s.id} className="flex items-center gap-3 border-t border-white/5 px-5 py-3">
                <span className="grid size-9 shrink-0 place-items-center rounded-full bg-white/6 text-muted">
                  {mobile ? <DeviceMobileIcon className="size-4" /> : <MonitorIcon className="size-4" />}
                </span>
                <div className="min-w-0 flex-1">
                  <p className="truncate text-[14px] font-semibold">{s.device}</p>
                  <p className="truncate text-xs text-subtle">
                    {s.place} · {current ? 'сейчас' : fmtAgo(s.lastSeen)}
                  </p>
                </div>
                {current ? (
                  <span className="shrink-0 rounded-full bg-success/10 px-2.5 py-1 text-[11px] font-semibold text-success">Текущий</span>
                ) : (
                  <button
                    onClick={() => {
                      kickSession(s.id).then((r) => r !== null && toast('Сеанс завершён', 'success'))
                    }}
                    className="shrink-0 rounded-full px-2.5 py-1 text-[11px] font-semibold text-danger transition-colors hover:bg-danger/10"
                  >
                    Завершить
                  </button>
                )}
              </div>
            )
          })
        )}
      </Stagger>

      <Stagger i={5}>
        <Button
          variant="danger"
          size="lg"
          className="w-full"
          onClick={() => {
            logout().then(() => navigate('/usernames'))
          }}
        >
          <SignOutIcon className="size-4" /> Выйти
        </Button>
      </Stagger>

      <TopUpSheet open={topup} onClose={() => setTopup(false)} />
    </div>
  )
}

function TopUpSheet({ open, onClose }: { open: boolean; onClose: () => void }) {
  const { topUp, toast, server } = useStore()
  const { data: servers } = useServers()
  const [amount, setAmount] = useState(500)
  const [loading, setLoading] = useState(false)
  const rate = servers?.find((s) => s.slug === server)?.starsPerGram ?? 0

  const submit = async () => {
    setLoading(true)
    const res = await topUp(amount)
    setLoading(false)
    if (!res) return
    toast(`Баланс пополнен на ${fmt(res.credited)} GRAM`, 'success')
    onClose()
  }

  return (
    <Modal open={open} onClose={onClose}>
      <div className="px-6 pt-10 pb-6 text-center">
        <motion.div initial={{ scale: 0.5, rotate: -20 }} animate={{ scale: 1, rotate: 0 }} transition={{ type: 'spring', bounce: 0.5, duration: 0.7 }} className="mx-auto w-fit">
          <GramIcon className="size-16 drop-shadow-[0_10px_30px_rgba(62,164,255,0.6)]" />
        </motion.div>
        <h3 className="mt-4 text-[22px] font-bold">Пополнить баланс</h3>
        <p className="mt-1 text-sm text-muted">
          {rate > 0 ? `Спишем звёзды с вашего аккаунта в приватке · ${fmt(Math.ceil(amount * rate))} ⭐` : 'На этом сервере пополнение выключено'}
        </p>
        <div className="mt-6 grid grid-cols-2 gap-2">
          {TOPUP.map((v) => (
            <button
              key={v}
              onClick={() => setAmount(v)}
              className={clsx('relative flex h-16 flex-col items-center justify-center rounded-[20px] transition-colors', amount === v ? 'bg-accent/15 text-fg ring-[1.5px] ring-accent' : 'bg-white/5 text-muted hover:bg-white/8')}
            >
              <Gram value={v} className="relative text-[17px] font-bold" size="size-4" />
              <span className="relative text-[11px] text-subtle">~ {fmtUsd(v)}</span>
            </button>
          ))}
        </div>
        <Button size="lg" className="mt-6 w-full" onClick={submit} disabled={loading || rate <= 0}>
          {loading ? 'Обработка…' : `Пополнить на ${fmt(amount)} GRAM`}
        </Button>
      </div>
    </Modal>
  )
}
