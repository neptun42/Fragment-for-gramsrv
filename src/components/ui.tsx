import { useEffect, useId, useRef, useState, type ButtonHTMLAttributes, type ReactNode } from 'react'
import { createPortal } from 'react-dom'
import { AnimatePresence, motion, type Transition } from 'motion/react'
import { CaretDownIcon, CheckIcon, MagnifyingGlassIcon, XIcon } from '@phosphor-icons/react'
import clsx from 'clsx'
import { fmt, fmtUsd } from '../lib/format'

export const spring: Transition = { type: 'spring', bounce: 0.2, duration: 0.45 }
export const softSpring: Transition = { type: 'spring', bounce: 0, duration: 0.4 }

// логотип и валюта

export function Logo({ className }: { className?: string }) {
  return (
    <span className={clsx('flex items-center gap-2.5', className)}>
      <img src="/logo-mark.png" alt="" className="size-8 select-none object-contain" draggable={false} />
      <span className="font-brand text-[19px] font-black tracking-[0.06em]">LunaX</span>
    </span>
  )
}

// значок GRAM, синий кружок
export function GramIcon({ className }: { className?: string }) {
  return (
    <svg viewBox="0 0 56 56" className={clsx('shrink-0', className)} aria-label="GRAM" role="img">
      <circle cx="28" cy="28" r="28" fill="#0098EA" />
      <path
        fill="#fff"
        d="M37.56 15.63H18.44c-3.52 0-5.75 3.79-3.98 6.86l11.8 20.45a2 2 0 0 0 3.47 0l11.8-20.45c1.77-3.06-.46-6.86-3.97-6.86Zm-11.3 21.18-2.58-4.98-6.2-11.09a1.08 1.08 0 0 1 .95-1.62h7.82v17.69Zm12.25-16.07-6.2 11.1-2.57 4.97V19.12h7.82c.86 0 1.36.91.95 1.62Z"
      />
    </svg>
  )
}

export function Gram({ value, className, size = 'size-3.5', usd }: { value: number; className?: string; size?: string; usd?: boolean }) {
  return (
    <span className={clsx('inline-flex flex-col', className)}>
      <span className="inline-flex items-center gap-1.5 tabular">
        <GramIcon className={size} />
        {fmt(value)}
      </span>
      {usd && <span className="mt-0.5 text-[11px] font-medium text-subtle tabular">~ {fmtUsd(value)}</span>}
    </span>
  )
}

export function StarIcon({ className }: { className?: string }) {
  const id = useId()
  return (
    <svg viewBox="0 0 24 24" className={clsx('shrink-0', className)} aria-hidden>
      <defs>
        <linearGradient id={id} x1="0" y1="0" x2="1" y2="1">
          <stop offset="0" stopColor="#ffd45e" />
          <stop offset="1" stopColor="#f5a234" />
        </linearGradient>
      </defs>
      <path
        fill={`url(#${id})`}
        d="M12 2.6c.4 0 .76.23.93.6l2.3 4.86 5.3.7c.84.1 1.18 1.14.56 1.72l-3.88 3.68.98 5.26c.16.83-.72 1.47-1.46 1.06L12 17.94l-4.73 2.54c-.74.4-1.62-.23-1.46-1.06l.98-5.26L2.9 10.48c-.62-.58-.28-1.61.56-1.72l5.3-.7 2.3-4.86c.17-.37.53-.6.94-.6Z"
      />
    </svg>
  )
}

// кнопки

type BtnProps = Omit<ButtonHTMLAttributes<HTMLButtonElement>, 'onDrag' | 'onDragStart' | 'onDragEnd' | 'onAnimationStart'> & {
  variant?: 'primary' | 'glass' | 'ghost' | 'danger'
  size?: 'sm' | 'md' | 'lg'
}

export function Button({ variant = 'primary', size = 'md', className, ...rest }: BtnProps) {
  return (
    <motion.button
      whileTap={rest.disabled ? undefined : { scale: 0.97 }}
      transition={{ type: 'spring', bounce: 0.3, duration: 0.25 }}
      {...rest}
      className={clsx(
        'inline-flex items-center justify-center gap-2 rounded-full font-semibold whitespace-nowrap transition-colors duration-200 disabled:opacity-40',
        size === 'sm' && 'h-8 px-3.5 text-[13px]',
        size === 'md' && 'h-10 px-5 text-sm',
        size === 'lg' && 'h-[52px] px-6 text-[15px]',
        variant === 'primary' && 'bg-accent text-white hover:bg-accent-hover',
        variant === 'glass' && 'bg-white/8 text-fg hover:bg-white/12',
        variant === 'ghost' && 'text-muted hover:bg-white/6 hover:text-fg',
        variant === 'danger' && 'bg-danger/12 text-danger hover:bg-danger/18',
        className,
      )}
    />
  )
}

// поля ввода

export function SearchInput({ value, onChange, placeholder, className }: { value: string; onChange: (v: string) => void; placeholder: string; className?: string }) {
  return (
    <label className={clsx('group flex h-12 items-center gap-3 rounded-2xl bg-surface px-4 ring-1 ring-line transition-[box-shadow] focus-within:ring-accent/60', className)}>
      <MagnifyingGlassIcon weight="bold" className="size-[18px] text-subtle transition-colors group-focus-within:text-accent" />
      <input value={value} onChange={(e) => onChange(e.target.value)} placeholder={placeholder} className="h-full min-w-0 flex-1 bg-transparent text-[15px] outline-none placeholder:text-subtle" />
      {value && (
        <button onClick={() => onChange('')} className="grid size-6 place-items-center rounded-full bg-white/10 text-muted hover:text-fg" aria-label="Очистить">
          <XIcon weight="bold" className="size-3" />
        </button>
      )}
    </label>
  )
}

export function Dropdown<T extends string>({ value, options, onChange }: { value: T; options: { value: T; label: string }[]; onChange: (v: T) => void }) {
  const [open, setOpen] = useState(false)
  const ref = useRef<HTMLDivElement>(null)

  useEffect(() => {
    if (!open) return
    const close = (e: MouseEvent) => {
      if (!ref.current?.contains(e.target as Node)) setOpen(false)
    }
    document.addEventListener('mousedown', close)
    return () => document.removeEventListener('mousedown', close)
  }, [open])

  return (
    <div ref={ref} className="relative">
      <button onClick={() => setOpen((o) => !o)} className="flex h-9 items-center gap-1.5 rounded-full bg-surface px-3.5 text-[13px] font-semibold ring-1 ring-line transition-colors hover:bg-elevated">
        {options.find((o) => o.value === value)?.label}
        <CaretDownIcon weight="bold" className={clsx('size-3.5 text-muted transition-transform duration-200', open && 'rotate-180')} />
      </button>
      <AnimatePresence>
        {open && (
          <motion.div
            initial={{ opacity: 0, y: -4, scale: 0.97 }}
            animate={{ opacity: 1, y: 0, scale: 1 }}
            exit={{ opacity: 0, y: -4, scale: 0.97 }}
            transition={{ duration: 0.15, ease: 'easeOut' }}
            className="glass absolute right-0 z-30 mt-2 min-w-52 origin-top-right rounded-2xl p-1.5"
          >
            {options.map((o) => (
              <button
                key={o.value}
                onClick={() => {
                  onChange(o.value)
                  setOpen(false)
                }}
                className="flex w-full items-center justify-between gap-4 rounded-xl px-3 py-2.5 text-left text-[13px] font-medium transition-colors hover:bg-white/8"
              >
                {o.label}
                {o.value === value && <CheckIcon weight="bold" className="size-4 text-accent" />}
              </button>
            ))}
          </motion.div>
        )}
      </AnimatePresence>
    </div>
  )
}

export function Segmented<T extends string>({ value, options, onChange, className }: { value: T; options: { value: T; label: ReactNode }[]; onChange: (v: T) => void; className?: string }) {
  return (
    <div className={clsx('flex gap-1 rounded-full bg-surface p-1 ring-1 ring-line', className)}>
      {options.map((o) => (
        <button
          key={o.value}
          onClick={() => onChange(o.value)}
          className={clsx(
            'h-8 flex-1 rounded-full px-3.5 text-[13px] font-semibold whitespace-nowrap transition-colors duration-200',
            o.value === value ? 'bg-white/12 text-fg' : 'text-muted hover:text-fg',
          )}
        >
          {o.label}
        </button>
      ))}
    </div>
  )
}

// модалка снизу

export function Modal({ open, onClose, children, className }: { open: boolean; onClose: () => void; children: ReactNode; className?: string }) {
  useEffect(() => {
    if (!open) return
    const onKey = (e: KeyboardEvent) => e.key === 'Escape' && onClose()
    document.addEventListener('keydown', onKey)
    document.body.style.overflow = 'hidden'
    return () => {
      document.removeEventListener('keydown', onKey)
      document.body.style.overflow = ''
    }
  }, [open, onClose])

  return createPortal(
    <AnimatePresence>
      {open && (
        <div className="fixed inset-0 z-50 flex items-end justify-center sm:items-center sm:p-6">
          <motion.div
            className="absolute inset-0 bg-[#0b0f14]/60 backdrop-blur-sm"
            onClick={onClose}
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            exit={{ opacity: 0 }}
            transition={{ duration: 0.2 }}
          />
          <motion.div
            role="dialog"
            initial={{ opacity: 0, y: 24, scale: 0.97 }}
            animate={{ opacity: 1, y: 0, scale: 1 }}
            exit={{ opacity: 0, y: 16, scale: 0.98 }}
            transition={{ type: 'spring', bounce: 0.1, duration: 0.4 }}
            className={clsx(
              'relative max-h-[92dvh] w-full overflow-y-auto rounded-t-[28px] bg-surface shadow-[0_24px_64px_-16px_rgba(0,0,0,0.6)] ring-1 ring-white/8 sm:rounded-[28px]',
              className?.includes('max-w') ? className : ['sm:max-w-[420px]', className],
            )}
          >
            <div className="mx-auto mt-2.5 h-1 w-9 rounded-full bg-white/15 sm:hidden" />
            <button onClick={onClose} className="absolute top-4 right-4 z-10 grid size-8 place-items-center rounded-full bg-white/6 text-muted transition-colors hover:bg-white/10 hover:text-fg" aria-label="Закрыть">
              <XIcon weight="bold" className="size-3.5" />
            </button>
            {children}
          </motion.div>
        </div>
      )}
    </AnimatePresence>,
    document.body,
  )
}

// всякое мелкое

export function Badge({ tone = 'default', children }: { tone?: 'default' | 'accent' | 'success' | 'danger'; children: ReactNode }) {
  return (
    <span
      className={clsx(
        'inline-flex h-6 items-center rounded-full px-2.5 text-[12px] font-semibold whitespace-nowrap',
        tone === 'default' && 'bg-white/8 text-muted',
        tone === 'accent' && 'bg-accent/15 text-link',
        tone === 'success' && 'bg-success/14 text-success',
        tone === 'danger' && 'bg-danger/14 text-danger',
      )}
    >
      {children}
    </span>
  )
}

export function PageTitle({ title, subtitle }: { title: string; subtitle?: ReactNode }) {
  return (
    <div className="mb-8 text-center">
      <h1 className="text-[28px] font-bold tracking-[-0.02em] sm:text-[32px]">{title}</h1>
      {subtitle && <p className="mx-auto mt-2 max-w-md text-[15px] leading-relaxed text-muted">{subtitle}</p>}
    </div>
  )
}

export function Empty({ icon, title, text, action }: { icon: ReactNode; title: string; text: string; action?: ReactNode }) {
  return (
    <div className="card flex flex-col items-center rounded-3xl px-6 py-16 text-center">
      <div className="mb-4 grid size-14 place-items-center rounded-2xl bg-white/6 text-muted">{icon}</div>
      <p className="text-[17px] font-semibold">{title}</p>
      <p className="mt-1 max-w-xs text-sm text-muted">{text}</p>
      {action && <div className="mt-6">{action}</div>}
    </div>
  )
}

export function Img({ src, alt, className }: { src: string; alt: string; className?: string }) {
  const [loaded, setLoaded] = useState(false)
  return (
    <span className={clsx('relative block overflow-hidden', className)}>
      {!loaded && <span className="skeleton absolute inset-0" />}
      <img
        src={src}
        alt={alt}
        loading="lazy"
        decoding="async"
        draggable={false}
        onLoad={() => setLoaded(true)}
        className={clsx('size-full object-cover transition-opacity duration-500 select-none', loaded ? 'opacity-100' : 'opacity-0')}
      />
    </span>
  )
}

function Digit({ d }: { d: string }) {
  return (
    <span className="relative grid h-9 w-7 place-items-center overflow-hidden rounded-[10px] bg-surface text-[15px] font-bold tabular ring-1 ring-line">
      <AnimatePresence initial={false}>
        <motion.span
          key={d}
          className="absolute"
          initial={{ y: -16, opacity: 0 }}
          animate={{ y: 0, opacity: 1 }}
          exit={{ y: 16, opacity: 0 }}
          transition={{ duration: 0.3, ease: [0.16, 1, 0.3, 1] }}
        >
          {d}
        </motion.span>
      </AnimatePresence>
    </span>
  )
}

// таймер как на фрагменте, цифры переворачиваются
export function Countdown({ ms }: { ms: number }) {
  const s = Math.max(0, Math.floor(ms / 1000))
  const d = Math.floor(s / 86400)
  const parts = [Math.floor((s % 86400) / 3600), Math.floor((s % 3600) / 60), s % 60].map((n) => String(n).padStart(2, '0'))
  return (
    <div className="flex items-center gap-1">
      {d > 0 && <span className="grid h-9 place-items-center rounded-[10px] bg-surface px-2.5 text-[15px] font-bold whitespace-nowrap ring-1 ring-line">{d} д</span>}
      {parts.map((p, i) => (
        <span key={i} className="flex items-center gap-1">
          {(i > 0 || d > 0) && <span className="px-0.5 font-bold text-subtle">:</span>}
          <Digit d={p[0]} />
          <Digit d={p[1]} />
        </span>
      ))}
    </div>
  )
}

export function Stagger({ i, children, className }: { i: number; children: ReactNode; className?: string }) {
  return (
    <motion.div initial={{ opacity: 0, y: 8 }} animate={{ opacity: 1, y: 0 }} transition={{ ...softSpring, delay: Math.min(i, 10) * 0.04 }} className={className}>
      {children}
    </motion.div>
  )
}
