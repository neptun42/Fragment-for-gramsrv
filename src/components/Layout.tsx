import { useEffect, useRef, useState } from 'react'
import { Link, NavLink, useLocation, useNavigate, useOutlet } from 'react-router'
import { AnimatePresence, motion } from 'motion/react'
import { AtIcon, CaretDownIcon, CheckCircleIcon, GiftIcon, InfoIcon, PackageIcon, PhoneIcon, SignOutIcon, UserIcon, WalletIcon, XCircleIcon } from '@phosphor-icons/react'
import clsx from 'clsx'
import { useStore } from '../lib/store'
import { Button, Gram, Logo } from './ui'
import { AuthModal } from './AuthModal'

const NAV = [
  { to: '/usernames', label: 'Юзернеймы', short: 'Имена', icon: AtIcon },
  { to: '/numbers', label: 'Номера', short: 'Номера', icon: PhoneIcon },
  { to: '/gifts', label: 'Подарки', short: 'Подарки', icon: GiftIcon },
]

function DesktopNav() {
  return (
    <nav className="hidden items-center gap-1 lg:flex">
      {NAV.map((n) => (
        <NavLink key={n.to} to={n.to} className={({ isActive }) => clsx('relative px-3.5 py-[19px] text-[14px] font-semibold transition-colors duration-200', isActive ? 'text-fg' : 'text-muted hover:text-fg')}>
          {({ isActive }) => (
            <>
              {n.label}
              {isActive && <motion.span layoutId="nav-line" className="absolute inset-x-3.5 bottom-0 h-[2px] rounded-full bg-accent" transition={{ type: 'spring', bounce: 0.15, duration: 0.4 }} />}
            </>
          )}
        </NavLink>
      ))}
    </nav>
  )
}

// нижнее меню для телефона
function TabBar() {
  const { user, openAuth } = useStore()
  const items = [...NAV, { to: '/profile', label: 'Профиль', short: 'Профиль', icon: UserIcon }]
  return (
    <nav className="fixed inset-x-3 bottom-[max(12px,env(safe-area-inset-bottom))] z-40 lg:hidden">
      <div className="glass mx-auto flex max-w-md items-center justify-between rounded-full p-1.5">
        {items.map((n) => (
          <NavLink
            key={n.to}
            to={n.to}
            onClick={(e) => {
              if (n.to === '/profile' && !user) {
                e.preventDefault()
                openAuth()
              }
            }}
            className={({ isActive }) =>
              clsx('flex flex-1 flex-col items-center gap-0.5 rounded-full py-1.5 text-[10px] font-semibold transition-colors duration-200', isActive ? 'bg-white/8 text-link' : 'text-muted')
            }
          >
            {({ isActive }) => (
              <>
                <n.icon weight={isActive ? 'fill' : 'regular'} className="size-[22px]" />
                {n.short}
              </>
            )}
          </NavLink>
        ))}
      </div>
    </nav>
  )
}

function UserMenu() {
  const { user, balance, logout } = useStore()
  const [open, setOpen] = useState(false)
  const ref = useRef<HTMLDivElement>(null)
  const navigate = useNavigate()
  const location = useLocation()

  useEffect(() => {
    setOpen(false)
  }, [location.pathname])

  useEffect(() => {
    if (!open) return
    const close = (e: MouseEvent) => {
      if (!ref.current?.contains(e.target as Node)) setOpen(false)
    }
    document.addEventListener('mousedown', close)
    return () => document.removeEventListener('mousedown', close)
  }, [open])

  if (!user) return null

  const items = [
    { icon: UserIcon, label: 'Профиль', to: '/profile' },
    { icon: PackageIcon, label: 'Инвентарь', to: '/inventory' },
    { icon: WalletIcon, label: 'Пополнить баланс', to: '/profile?topup=1' },
  ]

  return (
    <div ref={ref} className="relative flex items-center gap-2">
      <Link to="/profile" className="flex h-9 items-center rounded-full bg-white/6 px-3.5 text-[13px] font-semibold transition-colors hover:bg-white/10">
        <Gram value={balance} size="size-4" />
      </Link>
      <button onClick={() => setOpen((o) => !o)} className="flex h-9 items-center gap-2 rounded-full bg-white/6 pr-3 pl-1 transition-colors hover:bg-white/10">
        <span className="grid size-7 place-items-center rounded-full bg-accent text-[12px] font-bold">{user.username[0].toUpperCase()}</span>
        <span className="hidden text-[13px] font-semibold md:block">{user.username}</span>
        <CaretDownIcon weight="bold" className={clsx('size-3.5 text-muted transition-transform duration-200', open && 'rotate-180')} />
      </button>
      <AnimatePresence>
        {open && (
          <motion.div
            initial={{ opacity: 0, y: -4, scale: 0.97 }}
            animate={{ opacity: 1, y: 0, scale: 1 }}
            exit={{ opacity: 0, y: -4, scale: 0.97 }}
            transition={{ duration: 0.15, ease: 'easeOut' }}
            className="glass absolute top-12 right-0 z-40 w-60 origin-top-right rounded-2xl p-1.5"
          >
            <div className="px-3 pt-2.5 pb-3">
              <p className="text-[15px] font-semibold">@{user.username}</p>
              <Gram value={balance} className="mt-1 text-xs font-semibold text-muted" size="size-3" />
            </div>
            {items.map((i) => (
              <button key={i.label} onClick={() => navigate(i.to)} className="flex w-full items-center gap-3 rounded-xl px-3 py-2.5 text-[13px] font-medium transition-colors hover:bg-white/8">
                <i.icon weight="duotone" className="size-[18px] text-muted" />
                {i.label}
              </button>
            ))}
            <div className="mx-3 my-1 h-px bg-white/8" />
            <button
              onClick={() => {
                logout()
                navigate('/usernames')
              }}
              className="flex w-full items-center gap-3 rounded-xl px-3 py-2.5 text-[13px] font-medium text-danger transition-colors hover:bg-danger/10"
            >
              <SignOutIcon weight="duotone" className="size-[18px]" />
              Выйти
            </button>
          </motion.div>
        )}
      </AnimatePresence>
    </div>
  )
}

function Toasts() {
  const { toasts } = useStore()
  return (
    <div className="pointer-events-none fixed inset-x-0 top-4 z-[60] flex flex-col items-center gap-2 px-4">
      <AnimatePresence>
        {toasts.map((t) => (
          <motion.div
            key={t.id}
            initial={{ opacity: 0, y: -16, scale: 0.95 }}
            animate={{ opacity: 1, y: 0, scale: 1 }}
            exit={{ opacity: 0, y: -10, scale: 0.97 }}
            transition={{ type: 'spring', bounce: 0.2, duration: 0.4 }}
            className="glass flex items-center gap-2.5 rounded-full py-2.5 pr-5 pl-3 text-[13px] font-semibold"
          >
            {t.tone === 'error' ? (
              <XCircleIcon weight="fill" className="size-5 text-danger" />
            ) : t.tone === 'success' ? (
              <CheckCircleIcon weight="fill" className="size-5 text-success" />
            ) : (
              <InfoIcon weight="fill" className="size-5 text-link" />
            )}
            {t.title}
          </motion.div>
        ))}
      </AnimatePresence>
    </div>
  )
}

// держим старую страницу пока она плавно уходит
function FrozenOutlet() {
  const outlet = useOutlet()
  const [frozen, setFrozen] = useState(outlet)

  // на первом рендере редирект может прийти раньше аутлета, подхватываем его сразу
  useEffect(() => {
    if (outlet && outlet !== frozen) setFrozen(outlet)
  }, [outlet, frozen])

  return frozen
}

export function Layout() {
  const { user, openAuth } = useStore()
  const location = useLocation()

  return (
    <div className="flex min-h-dvh flex-col">
      <header className="sticky top-0 z-40 border-b border-white/6 bg-[#1f2730]/80 backdrop-blur-xl">
        <div className="mx-auto flex h-[60px] max-w-6xl items-center justify-between gap-4 px-4 sm:px-6">
          <Link to="/" aria-label="LunaX">
            <Logo />
          </Link>
          <DesktopNav />
          {user ? (
            <UserMenu />
          ) : (
            <Button size="sm" onClick={openAuth} className="h-9 px-5">
              Войти
            </Button>
          )}
        </div>
      </header>

      <main className="mx-auto w-full max-w-6xl flex-1 px-4 pt-10 pb-32 sm:px-6 lg:pb-20">
        <AnimatePresence mode="wait" initial={false} onExitComplete={() => window.scrollTo({ top: 0, behavior: 'instant' })}>
          <motion.div
            key={location.pathname}
            initial={{ opacity: 0, y: 8 }}
            animate={{ opacity: 1, y: 0, transition: { duration: 0.25, ease: [0.16, 1, 0.3, 1] } }}
            exit={{ opacity: 0, transition: { duration: 0.12, ease: 'easeIn' } }}
          >
            <FrozenOutlet />
          </motion.div>
        </AnimatePresence>
      </main>

      <footer className="hidden border-t border-white/6 lg:block">
        <div className="mx-auto flex max-w-6xl items-center justify-between gap-3 px-6 py-6 text-xs text-subtle">
          <span>© 2026 LunaX</span>
          <div className="flex gap-5">
            <Link to="/usernames" className="transition-colors hover:text-muted">Витрина</Link>
            <Link to="/inventory" className="transition-colors hover:text-muted">Инвентарь</Link>
            <Link to="/connect" className="transition-colors hover:text-muted">Подключить сервер</Link>
          </div>
        </div>
      </footer>

      <TabBar />
      <AuthModal />
      <Toasts />
    </div>
  )
}
