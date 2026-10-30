import { useState, type ReactNode } from 'react'
import { CheckIcon, CopyIcon, PlugsConnectedIcon } from '@phosphor-icons/react'
import clsx from 'clsx'
import { api, errorText } from '../lib/api'
import { useStore } from '../lib/store'
import { Button, PageTitle, Segmented, Stagger } from '../components/ui'

type Kind = 'gramsrv' | 'webhook'

type Result = { status: string; operator_key: string; server: { slug: string; name: string } }

const HINT: Record<Kind, ReactNode> = {
  gramsrv: (
    <>
      Включите Admin API в <code>.env</code> приватки: <code>TELESRV_ADMIN_API_ADDR</code> и <code>TELESRV_ADMIN_API_TOKEN</code>. Адрес — это Admin API, например <code>https://admin.myserver.net</code>,
      токен — мастер-токен или scoped-токен с правом <code>*</code>.
    </>
  ),
  webhook: (
    <>
      Для приваток на другом движке. Поднимите один эндпоинт по контракту из <code>backend/CONNECT.md</code>, LunaX будет слать на него подписанные запросы. Секрет — от 16 символов.
    </>
  ),
}

function Field({ label, hint, children }: { label: string; hint?: string; children: ReactNode }) {
  return (
    <label className="block">
      <span className="mb-1.5 block px-1 text-[13px] font-semibold text-muted">{label}</span>
      {children}
      {hint && <span className="mt-1 block px-1 text-[11px] text-subtle">{hint}</span>}
    </label>
  )
}

const inputCls = 'h-12 w-full rounded-2xl bg-black/20 px-4 text-[15px] outline-none ring-1 ring-white/8 transition-[box-shadow] placeholder:text-subtle/60 focus:ring-accent/70'

// форма привязки своей приватки к маркету
export function ConnectPage() {
  const { toast } = useStore()
  const [kind, setKind] = useState<Kind>('gramsrv')
  const [form, setForm] = useState({ name: '', slug: '', api_url: '', secret: '', contact: '', stars_per_gram: '' })
  const [sending, setSending] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [result, setResult] = useState<Result | null>(null)

  const set = (k: keyof typeof form) => (e: React.ChangeEvent<HTMLInputElement>) => setForm((f) => ({ ...f, [k]: e.target.value }))
  const valid = form.name.trim().length >= 2 && form.slug.length >= 3 && form.api_url.startsWith('http') && form.secret.length > 0

  const submit = async () => {
    if (!valid || sending) return
    setSending(true)
    setError(null)
    try {
      const res = await api<Result>('/api/servers', {
        auth: false,
        body: { ...form, kind, stars_per_gram: Number(form.stars_per_gram) || 0 },
      })
      setResult(res)
    } catch (e) {
      setError(errorText(e))
    } finally {
      setSending(false)
    }
  }

  if (result) {
    return (
      <div className="mx-auto max-w-[560px]">
        <Stagger i={0} className="card rounded-[26px] p-6 text-center">
          <span className="mx-auto mb-4 grid size-14 place-items-center rounded-full bg-success/15 text-success">
            <CheckIcon className="size-7" />
          </span>
          <h1 className="text-[22px] font-bold">{result.server.name} подключён</h1>
          <p className="mt-1 text-sm text-muted">
            {result.status === 'active' ? 'Сервер уже на витрине, юзеры могут входить.' : 'Заявка ждёт одобрения владельца LunaX.'}
          </p>
          <div className="mt-6 rounded-[20px] bg-black/20 p-4 text-left ring-1 ring-white/8">
            <p className="text-[12px] font-semibold text-muted">Ключ оператора · показываем один раз</p>
            <p className="mt-1 font-mono text-[13px] break-all">{result.operator_key}</p>
          </div>
          <Button
            variant="glass"
            className="mt-3 w-full"
            onClick={() => {
              navigator.clipboard?.writeText(result.operator_key).catch(() => {})
              toast('Ключ скопирован', 'success')
            }}
          >
            <CopyIcon className="size-4" /> Скопировать ключ
          </Button>
          <p className="mt-4 text-[12px] leading-relaxed text-subtle">
            С этим ключом выставляйте лоты своей приватки через <code>/api/operator/lots</code>. Примеры запросов — в <code>backend/CONNECT.md</code>.
          </p>
        </Stagger>
      </div>
    )
  }

  return (
    <div className="mx-auto max-w-[560px]">
      <PageTitle title="Подключить приватку" subtitle="LunaX открытый: любой может привязать свой сервер, и его юзеры будут покупать юзернеймы, номера и подарки прямо на свой аккаунт." />

      <Stagger i={1} className="card mt-8 space-y-4 rounded-[26px] p-5">
        <Segmented
          value={kind}
          onChange={setKind}
          options={[
            { value: 'gramsrv', label: 'gramsrv / telesrv' },
            { value: 'webhook', label: 'Другой движок' },
          ]}
        />
        <p className="px-1 text-[12px] leading-relaxed text-subtle [&_code]:rounded [&_code]:bg-white/8 [&_code]:px-1 [&_code]:text-muted">{HINT[kind]}</p>

        <Field label="Название">
          <input value={form.name} onChange={set('name')} placeholder="Моя приватка" className={inputCls} />
        </Field>
        <Field label="Короткий id" hint="3-32 символа: a-z, 0-9, дефис">
          <input value={form.slug} onChange={(e) => setForm((f) => ({ ...f, slug: e.target.value.toLowerCase().replace(/[^a-z0-9-]/g, '') }))} placeholder="myserver" className={inputCls} />
        </Field>
        <Field label={kind === 'gramsrv' ? 'Адрес Admin API' : 'Адрес вебхука'}>
          <input value={form.api_url} onChange={set('api_url')} placeholder={kind === 'gramsrv' ? 'https://admin.myserver.net' : 'https://myserver.net/lunax'} className={inputCls} />
        </Field>
        <Field label={kind === 'gramsrv' ? 'Токен Admin API' : 'Секрет подписи'} hint="Храним зашифрованным, наружу не отдаём">
          <input value={form.secret} onChange={set('secret')} type="password" autoComplete="off" className={inputCls} />
        </Field>
        <Field label="Звёзд за 1 GRAM" hint="0 — пополнение баланса звёздами выключено">
          <input value={form.stars_per_gram} onChange={set('stars_per_gram')} inputMode="decimal" placeholder="0" className={inputCls} />
        </Field>
        <Field label="Контакт" hint="Телега или почта, чтобы с вами связаться">
          <input value={form.contact} onChange={set('contact')} placeholder="@admin" className={inputCls} />
        </Field>

        <div className={clsx('min-h-5 text-center text-[13px] text-danger')}>{error}</div>

        <Button size="lg" className="w-full" disabled={!valid || sending} onClick={submit}>
          <PlugsConnectedIcon className="size-4" /> {sending ? 'Проверяем сервер…' : 'Подключить'}
        </Button>
      </Stagger>
    </div>
  )
}
