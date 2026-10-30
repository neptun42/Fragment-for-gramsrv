# LunaX

Открытый маркет для приватных Telegram-серверов: юзернеймы, анонимные номера
+888 и коллекционные подарки. Любой владелец приватки может привязать свой
сервер, и его юзеры будут покупать прямо на свой аккаунт.

- Фронт: React 19, TypeScript, Vite 8, Tailwind CSS 4, Motion, React Router.
- Бэкенд: Go, PostgreSQL, лежит в [`backend/`](backend/).
- Как подключить свою приватку: [`backend/CONNECT.md`](backend/CONNECT.md).

## Быстрый старт в Docker

```bash
echo "LUNAX_SECRET_KEY=$(openssl rand -hex 32)" > .env
docker compose up --build
```

Откройте http://localhost:8080, затем `/connect` — подключите свою приватку.

## Разработка

Нужны Go 1.25+, Node 22+ и PostgreSQL.

```bash
cd backend
export LUNAX_SECRET_KEY=dev-secret-key-change-me
export LUNAX_AUTO_APPROVE=true
go run ./cmd/lunax
```

В другом терминале:

```bash
npm install
npm run dev
```

Vite проксирует `/api` на `localhost:8080`.

## Как устроен бэкенд

```
backend/
  cmd/lunax/            main и привязка серверов из servers.json
  internal/domain/      сущности и доменные ошибки
  internal/network/     интерфейс приватки и реестр адаптеров
    gramsrv/            адаптер под Admin API gramsrv/telesrv
    webhook/            адаптер под любой движок по подписанному webhook
  internal/store/       интерфейсы хранилища
    postgres/           реализация на pgx + миграции
  internal/service/     вход, маркет, кошелёк, сервера
  internal/httpapi/     http хендлеры
```

- Сервисы зависят от узких интерфейсов (`store.*`, `network.*`), а не от pgx
  или конкретного сервера.
- Новый тип приватки — новый пакет с `network.Register("kind", ...)` в `init`.
  Сервисы и хендлеры трогать не нужно.
- Деньги хранятся в копейках GRAM (`int64`), баланс не уходит в минус за счёт
  `CHECK` в базе, все списания идут в транзакции.

## Лицензия

[MIT](LICENSE). Pull request'ы и новые адаптеры приваток приветствуются.
