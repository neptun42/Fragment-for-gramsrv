import { useMemo, useState } from 'react'
import { Link, Navigate, useNavigate, useParams } from 'react-router'
import { motion } from 'motion/react'
import { CaretLeftIcon, GiftIcon, MagnifyingGlassIcon, SlidersHorizontalIcon } from '@phosphor-icons/react'
import clsx from 'clsx'
import { backdropColor, COLLECTIONS, giftImg, type Gift } from '../lib/data'
import { useGift, useGifts } from '../lib/market'
import { fmtDate } from '../lib/format'
import { useRequireAuth, useStore } from '../lib/store'
import { Badge, Button, Dropdown, Empty, Gram, Img, Modal, PageTitle, SearchInput, softSpring, Stagger } from '../components/ui'

type Sort = 'price-desc' | 'price-asc' | 'new'
type StatusFilter = 'all' | 'sale' | 'sold'

export function GiftsPage() {
  const navigate = useNavigate()
  const [query, setQuery] = useState('')
  const [collection, setCollection] = useState<string | null>(null)
  const [colQuery, setColQuery] = useState('')
  const [sort, setSort] = useState<Sort>('price-desc')
  const [status, setStatus] = useState<StatusFilter>('all')
  const [filtersOpen, setFiltersOpen] = useState(false)
  const { data, loading } = useGifts()
  const all = useMemo(() => data ?? [], [data])

  const counts = useMemo(() => {
    const m: Record<string, number> = {}
    all.forEach((g) => (m[g.slug] = (m[g.slug] ?? 0) + 1))
    return m
  }, [all])

  const gifts = useMemo(() => {
    const q = query.trim().toLowerCase()
    return all.filter((g) => !collection || g.slug === collection)
      .filter((g) => status === 'all' || g.status === status)
      .filter((g) => !q || `${g.collection} #${g.num} ${g.model.name} ${g.backdrop.name} ${g.symbol.name}`.toLowerCase().includes(q))
      .sort((a, b) => (sort === 'price-asc' ? a.price - b.price : sort === 'new' ? b.date - a.date : b.price - a.price))
  }, [all, query, collection, sort, status])

  const collections = COLLECTIONS.filter((c) => c.name.toLowerCase().includes(colQuery.toLowerCase()))

  if (all.length === 0) {
    return (
      <div className="mx-auto max-w-[760px]">
        <PageTitle title="Подарки" subtitle="Коллекционные подарки Telegram." />
        <div className="mt-8">
          <Empty icon={<GiftIcon className="size-6" />} title={loading ? 'Загружаем подарки…' : 'Пока нет подарков'} text={loading ? 'Секунду' : 'Новые подарки появятся здесь.'} />
        </div>
      </div>
    )
  }

  const pickCollection = (slug: string | null) => {
    setCollection(slug)
    setFiltersOpen(false)
  }

  const sidebar = (
    <div className="space-y-3">
      <SearchInput value={query} onChange={setQuery} placeholder="Название или атрибут" />
      <div className="card overflow-hidden rounded-[24px]">
        <div className="flex items-center justify-between px-4 pt-4 pb-2 text-[14px] font-semibold">
          Коллекции <span className="text-[12px] font-medium text-subtle">{COLLECTIONS.length}</span>
        </div>
        <div className="px-2 pb-2">
          <input
            value={colQuery}
            onChange={(e) => setColQuery(e.target.value)}
            placeholder="Поиск коллекций"
            className="mb-1.5 h-9 w-full rounded-full bg-white/6 px-4 text-[13px] outline-none placeholder:text-subtle focus:bg-white/8"
          />
          <div className="max-h-[440px] overflow-y-auto">
            <CollectionRow active={!collection} onClick={() => pickCollection(null)} label="Все коллекции" count={all.length} />
            {collections.map((c) => (
              <CollectionRow key={c.slug} active={collection === c.slug} onClick={() => pickCollection(c.slug)} label={c.name} count={counts[c.slug] ?? 0} img={giftImg(c.slug, 1)} />
            ))}
          </div>
        </div>
      </div>
    </div>
  )

  return (
    <div className="grid gap-6 lg:grid-cols-[264px_1fr]">
      <aside className="hidden lg:block">
        <div className="sticky top-24">{sidebar}</div>
      </aside>

      <section>
        <div className="mb-5 flex flex-wrap items-center justify-between gap-3">
          <div>
            <motion.h1 key={collection ?? 'all'} initial={{ opacity: 0, y: 4 }} animate={{ opacity: 1, y: 0 }} className="text-[26px] font-bold tracking-tight">
              {collection ? COLLECTIONS.find((c) => c.slug === collection)?.name : 'Топ подарков'}
            </motion.h1>
            <p className="mt-0.5 text-sm text-muted">{gifts.length} предметов</p>
          </div>
          <div className="flex flex-wrap gap-2">
            <Button variant="glass" size="sm" className="h-9 lg:hidden" onClick={() => setFiltersOpen(true)}>
              <SlidersHorizontalIcon className="size-4" /> Коллекции
            </Button>
            <Dropdown
              value={status}
              options={[
                { value: 'all', label: 'Все подарки' },
                { value: 'sale', label: 'Продаются' },
                { value: 'sold', label: 'Проданы' },
              ]}
              onChange={setStatus}
            />
            <Dropdown
              value={sort}
              options={[
                { value: 'price-desc', label: 'Сначала дорогие' },
                { value: 'price-asc', label: 'Сначала дешёвые' },
                { value: 'new', label: 'Сначала новые' },
              ]}
              onChange={setSort}
            />
          </div>
        </div>

        {gifts.length === 0 ? (
          <Empty icon={<MagnifyingGlassIcon className="size-6" />} title="Ничего не найдено" text="Попробуйте другой запрос или коллекцию" />
        ) : (
          <motion.div
            key={`${collection}-${status}-${sort}-${query}`}
            initial="hidden"
            animate="show"
            variants={{ show: { transition: { staggerChildren: 0.025 } } }}
            className="grid grid-cols-2 gap-3 sm:grid-cols-3 xl:grid-cols-4"
          >
            {gifts.map((g) => (
              <motion.div key={g.id} variants={{ hidden: { opacity: 0, y: 16, scale: 0.97 }, show: { opacity: 1, y: 0, scale: 1, transition: softSpring } }}>
                <GiftCard gift={g} onClick={() => navigate(`/gifts/${g.id}`)} />
              </motion.div>
            ))}
          </motion.div>
        )}
      </section>

      <Modal open={filtersOpen} onClose={() => setFiltersOpen(false)}>
        <div className="p-5 pt-14">{sidebar}</div>
      </Modal>
    </div>
  )
}

function GiftCard({ gift, onClick }: { gift: Gift; onClick: () => void }) {
  return (
    <motion.button
      onClick={onClick}
      whileHover={{ y: -3 }}
      whileTap={{ scale: 0.97 }}
      transition={{ type: 'spring', bounce: 0.3, duration: 0.4 }}
      className="card group block w-full overflow-hidden rounded-[22px] text-left"
    >
      <div className="relative overflow-hidden">
        <Img src={giftImg(gift.slug, gift.num)} alt={gift.collection} className="aspect-square transition-transform duration-700 ease-out group-hover:scale-[1.06]" />
      </div>
      <div className="p-3">
        <p className="truncate text-[14px] font-semibold">
          {gift.collection} <span className="text-muted">#{gift.num}</span>
        </p>
        <p className="mt-0.5 truncate text-[11px] text-subtle">{fmtDate(gift.date)}</p>
        <div className="mt-2 flex items-center justify-between gap-2">
          <Gram value={gift.price} className="text-[14px] font-bold" />
          {gift.status === 'sold' ? <Badge tone="danger">Продан</Badge> : <Badge tone="success">В продаже</Badge>}
        </div>
      </div>
    </motion.button>
  )
}

function CollectionRow({ active, onClick, label, count, img }: { active: boolean; onClick: () => void; label: string; count: number; img?: string }) {
  return (
    <button
      onClick={onClick}
      className={clsx('relative flex w-full items-center gap-2.5 rounded-[14px] px-2.5 py-2 text-left text-[13px] transition-colors', active ? 'bg-accent/15 font-semibold text-link' : 'font-medium hover:bg-white/5')}
    >
      {img ? <Img src={img} alt="" className="relative size-7 rounded-[9px]" /> : <span className="relative size-7 rounded-[9px] bg-white/10" />}
      <span className="relative flex-1 truncate">{label}</span>
      <span className={clsx('relative text-xs tabular', active ? 'text-link' : 'text-subtle')}>{count}</span>
    </button>
  )
}

export function GiftPage() {
  const { id = '' } = useParams()
  const { data: gift, loading } = useGift(id)
  if (loading && !gift) return null
  if (!gift) return <Navigate to="/gifts" replace />
  return <GiftView gift={gift} />
}

function GiftView({ gift }: { gift: Gift }) {
  const { buyLot, toast, inventory } = useStore()
  const { data: same } = useGifts()
  const requireAuth = useRequireAuth()
  const navigate = useNavigate()
  const owned = inventory.some((i) => i.kind === 'gift' && i.slug === gift.slug && i.num === gift.num)
  const similar = (same ?? []).filter((g) => g.slug === gift.slug && g.id !== gift.id).slice(0, 4)

  const buy = () =>
    requireAuth(() => {
      buyLot(gift.id).then((item) => item && toast(`${gift.collection} #${gift.num} в вашем инвентаре`, 'success'))
    })

  const attrs = [
    { label: 'Модель', ...gift.model },
    { label: 'Фон', ...gift.backdrop },
    { label: 'Узор', ...gift.symbol },
  ]

  return (
    <div className="mx-auto max-w-[920px]">
      <Link to="/gifts" className="mb-5 inline-flex items-center gap-1 text-[14px] font-semibold text-link transition-opacity hover:opacity-80">
        <CaretLeftIcon className="size-4" /> Подарки
      </Link>

      <div className="grid gap-6 md:grid-cols-[1fr_1fr]">
        <Stagger i={0}>
          <div className="relative overflow-hidden rounded-[32px]" style={{ background: backdropColor(gift.backdrop.name) }}>
            <motion.div initial={{ scale: 1.08 }} animate={{ scale: 1 }} transition={{ type: 'spring', bounce: 0, duration: 1.2 }}>
              <Img src={giftImg(gift.slug, gift.num)} alt={gift.collection} className="aspect-square" />
            </motion.div>
          </div>
        </Stagger>

        <div className="flex flex-col">
          <Stagger i={1}>
            <p className="text-[13px] font-semibold text-muted">Коллекционный подарок</p>
            <h1 className="mt-1 text-[30px] font-bold tracking-tight">
              {gift.collection} <span className="text-muted">#{gift.num}</span>
            </h1>
            <div className="mt-2 flex gap-2">
              {gift.status === 'sale' ? <Badge tone="success">В продаже</Badge> : <Badge tone="danger">Продан</Badge>}
              {owned && <Badge tone="accent">У вас в инвентаре</Badge>}
            </div>
          </Stagger>

          <Stagger i={2} className="card mt-6 overflow-hidden rounded-[24px]">
            {attrs.map((a, i) => (
              <div key={a.label} className={clsx('flex items-center justify-between px-5 py-3.5 text-[14px]', i > 0 && 'border-t border-white/5')}>
                <span className="font-semibold">{a.label}</span>
                <span className="flex items-center gap-2 font-medium text-link">
                  {a.name}
                  <span className="rounded-full bg-white/8 px-2 py-0.5 text-[11px] font-semibold text-muted tabular">{a.rarity}%</span>
                </span>
              </div>
            ))}
            <div className="flex items-center justify-between border-t border-white/5 px-5 py-3.5 text-[14px]">
              <span className="font-semibold">{gift.status === 'sold' ? 'Продан' : 'Выставлен'}</span>
              <span className="font-medium text-muted">{fmtDate(gift.date)}</span>
            </div>
          </Stagger>

          <Stagger i={3} className="mt-auto pt-6">
            <div className="card flex items-center justify-between rounded-[24px] p-5">
              <div>
                <p className="text-[13px] font-semibold text-muted">{gift.status === 'sold' ? 'Продан за' : 'Цена'}</p>
                <Gram value={gift.price} usd className="mt-1.5 text-[22px] font-bold" size="size-5" />
              </div>
              {gift.status === 'sale' && (
                <Button size="lg" onClick={buy} className="px-8">
                  Купить
                </Button>
              )}
            </div>
          </Stagger>
        </div>
      </div>

      {similar.length > 0 && (
        <Stagger i={4} className="mt-12">
          <h2 className="mb-4 px-1 text-[22px] font-bold tracking-tight">Ещё из коллекции</h2>
          <div className="grid grid-cols-2 gap-3 sm:grid-cols-4">
            {similar.map((g) => (
              <GiftCard key={g.id} gift={g} onClick={() => navigate(`/gifts/${g.id}`)} />
            ))}
          </div>
        </Stagger>
      )}
    </div>
  )
}
