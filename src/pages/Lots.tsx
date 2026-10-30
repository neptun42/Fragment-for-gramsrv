import { useMemo, useState } from 'react'
import { useNavigate } from 'react-router'
import { motion } from 'motion/react'
import { AtIcon, CaretRightIcon, PhoneIcon } from '@phosphor-icons/react'
import { lotUrl, type Lot, type LotStatus } from '../lib/data'
import { useLots } from '../lib/market'
import { fmtDate, fmtLeft, useNow } from '../lib/format'
import { useStore } from '../lib/store'
import { Badge, Dropdown, Empty, Gram, PageTitle, SearchInput, softSpring } from '../components/ui'

type Sort = 'price-desc' | 'price-asc'

const STATUS_FILTER: { value: LotStatus; label: string }[] = [
  { value: 'auction', label: 'Аукционы' },
  { value: 'sale', label: 'Продаются' },
  { value: 'sold', label: 'Проданы' },
]

const SORTS: { value: Sort; label: string }[] = [
  { value: 'price-desc', label: 'Сначала дорогие' },
  { value: 'price-asc', label: 'Сначала дешёвые' },
]

const COPY = {
  username: {
    title: 'Покупка и продажа юзернеймов',
    subtitle: (
      <>
        <b className="font-semibold text-fg">Закрепите имя</b> — короткое и запоминающееся, для аккаунта, канала или группы.
      </>
    ),
    search: 'Введите юзернейм…',
  },
  number: {
    title: 'Анонимные номера',
    subtitle: (
      <>
        Номера <b className="font-semibold text-fg">+888</b> для входа в Telegram без SIM-карты. Полностью анонимно.
      </>
    ),
    search: 'Введите номер +888…',
  },
}

export const lotTitle = (lot: Lot) => (lot.kind === 'username' ? `@${lot.name}` : lot.name)

export function LotsPage({ kind }: { kind: Lot['kind'] }) {
  const copy = COPY[kind]
  const now = useNow()
  const navigate = useNavigate()
  const { bids } = useStore()
  const { data, loading } = useLots(kind)
  const [query, setQuery] = useState('')
  const [status, setStatus] = useState<LotStatus>('sale')
  const [sort, setSort] = useState<Sort>('price-desc')
  const all = data ?? []

  const lots = useMemo(() => {
    const q = query.trim().toLowerCase().replace(/^@/, '').replace(/\s/g, '')
    return all
      .filter((l) => l.status === status)
      .filter((l) => !q || l.name.toLowerCase().replace(/\s/g, '').includes(q))
      .sort((a, b) => (sort === 'price-asc' ? a.price - b.price : b.price - a.price))
  }, [all, query, status, sort])

  const Icon = kind === 'username' ? AtIcon : PhoneIcon

  return (
    <div className="mx-auto max-w-[760px]">
      <PageTitle title={copy.title} subtitle={copy.subtitle} />
      <SearchInput value={query} onChange={setQuery} placeholder={copy.search} />

      <div className={all.length === 0 ? 'hidden' : 'mt-9 mb-3 flex flex-wrap items-center justify-between gap-3 px-1'}>
        <h2 className="text-[20px] font-bold tracking-tight">{status === 'auction' ? 'Топ аукционов' : status === 'sale' ? 'Продаются' : 'Недавно проданы'}</h2>
        <div className="flex gap-2">
          <Dropdown value={status} options={STATUS_FILTER} onChange={setStatus} />
          <Dropdown value={sort} options={SORTS} onChange={setSort} />
        </div>
      </div>

      {lots.length === 0 ? (
        <div className="mt-8">
          <Empty icon={<Icon className="size-6" />} title={loading ? 'Загружаем лоты…' : 'Пока нет лотов'} text={loading ? 'Секунду' : 'Новые предложения появятся здесь.'} />
        </div>
      ) : (
        <div className="card overflow-hidden rounded-[24px]">
          <div className="grid grid-cols-[1fr_auto] gap-4 bg-elevated/70 px-4 py-3 text-[12px] font-semibold text-muted sm:grid-cols-[1.3fr_1fr_1.3fr_16px] sm:px-5">
            <span>{kind === 'username' ? 'Юзернейм' : 'Номер'}</span>
            <span className="text-right sm:text-left">{status === 'auction' ? 'Мин. ставка' : 'Цена'}</span>
            <span className="hidden sm:block">{status === 'auction' ? 'До окончания' : status === 'sale' ? 'Статус' : 'Дата продажи'}</span>
          </div>
          <motion.div key={`${status}-${sort}-${query}`} initial="hidden" animate="show" variants={{ show: { transition: { staggerChildren: 0.025 } } }}>
            {lots.map((lot) => (
              <motion.button
                key={lot.id}
                variants={{ hidden: { opacity: 0, y: 6 }, show: { opacity: 1, y: 0, transition: softSpring } }}
                onClick={() => navigate(lotUrl(lot))}
                whileTap={{ scale: 0.985 }}
                className="group grid w-full grid-cols-[1fr_auto] items-center gap-4 border-t border-white/5 px-4 py-3.5 text-left transition-colors duration-200 hover:bg-white/[0.035] sm:grid-cols-[1.3fr_1fr_1.3fr_16px] sm:px-5"
              >
                <span className="flex min-w-0 items-center gap-3">
                  <span className="grid size-10 shrink-0 place-items-center rounded-[14px] bg-white/6 text-muted transition-all duration-300 group-hover:bg-accent/15 group-hover:text-link">
                    <Icon className="size-[18px]" />
                  </span>
                  <span className="min-w-0">
                    <span className="flex items-center gap-2">
                      <span className="truncate text-[15px] font-semibold tabular">{lotTitle(lot)}</span>
                      {bids[lot.id] && <span className="size-1.5 shrink-0 rounded-full bg-success" title="Есть ваша ставка" />}
                    </span>
                    <span className="block truncate text-[12px] text-link">{kind === 'username' ? `t.me/${lot.name}` : 'анонимный номер'}</span>
                  </span>
                </span>
                <span className="text-right sm:text-left">
                  <Gram value={lot.status === 'auction' ? lot.minBid : lot.price} usd className="text-[14px] font-semibold" />
                </span>
                <span className="hidden text-[13px] sm:block">
                  {status === 'auction' ? (
                    <>
                      <span className="block font-medium">{fmtLeft(lot.endsAt - now)}</span>
                      <span className="text-[12px] text-subtle">{fmtDate(lot.endsAt)}</span>
                    </>
                  ) : status === 'sale' ? (
                    <Badge tone="success">Купить сейчас</Badge>
                  ) : (
                    <span className="text-muted">{fmtDate(lot.endsAt)}</span>
                  )}
                </span>
                <CaretRightIcon className="hidden size-4 text-subtle transition-transform duration-300 group-hover:translate-x-1 group-hover:text-muted sm:block" />
              </motion.button>
            ))}
          </motion.div>
        </div>
      )}
    </div>
  )
}
