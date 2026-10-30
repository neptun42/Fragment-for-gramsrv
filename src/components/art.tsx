import type { ReactNode } from 'react'
import clsx from 'clsx'
import { AtIcon, PhoneIcon } from '@phosphor-icons/react'
import { giftImg } from '../lib/data'
import type { InvItem } from '../lib/store'
import { fmt } from '../lib/format'
import { Img, StarIcon } from './ui'

function Frame({ className, children }: { className?: string; children: ReactNode }) {
  return (
    <div className={clsx('grid aspect-square place-items-center bg-elevated', className)}>
      <div className="flex flex-col items-center">{children}</div>
    </div>
  )
}

// картинка предмета в инвентаре
export function ItemArt({ item, className, big }: { item: InvItem; className?: string; big?: boolean }) {
  if (item.kind === 'gift') return <Img src={giftImg(item.slug, item.num)} alt={item.title} className={clsx('aspect-square', className)} />

  if (item.kind === 'stars')
    return (
      <Frame className={className}>
        <StarIcon className={big ? 'size-20' : 'size-[38%] min-w-8'} />
        <span className={clsx('mt-2.5 font-bold tabular', big ? 'text-2xl' : 'text-[13px]')}>{fmt(item.amount)}</span>
      </Frame>
    )

  if (item.kind === 'number') {
    const [code, ...rest] = item.number.split(' ')
    return (
      <Frame className={className}>
        <span className={clsx('grid place-items-center rounded-full bg-accent/15 text-link', big ? 'mb-4 size-16' : 'mb-2 size-9')}>
          <PhoneIcon weight="fill" className={big ? 'size-7' : 'size-[18px]'} />
        </span>
        {rest.length > 0 && <span className={clsx('font-medium text-muted', big ? 'text-base' : 'text-[10px]')}>{code}</span>}
        <span className={clsx('font-bold tracking-wide tabular', big ? 'text-2xl' : 'text-[13px]')}>{rest.length > 0 ? rest.join(' ') : code}</span>
      </Frame>
    )
  }

  return (
    <Frame className={className}>
      <span className={clsx('grid place-items-center rounded-full bg-accent/15 text-link', big ? 'mb-4 size-16' : 'mb-2 size-9')}>
        <AtIcon weight="bold" className={big ? 'size-7' : 'size-[18px]'} />
      </span>
      <span className={clsx('font-bold', big ? 'text-2xl' : 'text-sm')}>{item.username}</span>
    </Frame>
  )
}
