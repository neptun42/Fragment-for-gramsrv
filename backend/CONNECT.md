# Как привязать свою приватку к LunaX

LunaX — открытый маркет. Любой владелец приватного Telegram-сервера может
подключить его, и юзеры этого сервера будут входить по своему номеру,
покупать юзернеймы, +888 номера и подарки и выводить их прямо на свой аккаунт.

LunaX не лезет в базу вашего сервера напрямую. Всё идёт через API вашего
сервера, поэтому кэши, апдейты клиентам и аудит работают как обычно.

Есть два типа привязки:

| kind | для кого | что нужно |
| --- | --- | --- |
| `gramsrv` | приватки на gramsrv / telesrv | включить Admin API и дать токен |
| `webhook` | любой другой движок | поднять один эндпоинт по контракту ниже |

## Способ 1. Через сайт

Откройте `/connect` на LunaX, заполните форму, нажмите «Подключить». LunaX
сразу проверит, что сервер отвечает с этим токеном. В ответ придёт **ключ
оператора** `lxo_...` — он показывается один раз, сохраните его.

Если на LunaX выключено автоодобрение (`LUNAX_AUTO_APPROVE=false`), сервер
появится на витрине после одобрения владельцем.

## Способ 2. Через servers.json (если LunaX ваш)

Скопируйте `servers.example.json` в `servers.json`, впишите свой сервер и
укажите `LUNAX_SERVERS_FILE=servers.json`. При старте LunaX привяжет его и
сразу включит. Ключ оператора выведется в лог один раз.

```json
[
  {
    "slug": "myserver",
    "name": "Моя приватка",
    "kind": "gramsrv",
    "api_url": "http://127.0.0.1:2402",
    "secret": "${MYSERVER_ADMIN_TOKEN}",
    "stars_per_gram": 10
  }
]
```

`${...}` подставляется из переменных окружения, так токен не лежит в файле.

## gramsrv / telesrv

В `.env` приватки:

```env
TELESRV_ADMIN_API_ADDR=127.0.0.1:2402
TELESRV_ADMIN_API_TOKEN=длинный-случайный-токен
```

Если LunaX на другой машине, пробросьте Admin API через reverse proxy с HTTPS
и закройте доступ всем, кроме IP LunaX. Admin API даёт полный контроль над
сервером, держите токен в секрете.

Нужен мастер-токен или scoped-токен с правом `*`:
маршруты, которые использует LunaX, в gramsrv требуют полного набора прав.

```env
TELESRV_ADMIN_SCOPED_TOKENS=lunax:другой-случайный-токен:*
```

Что LunaX вызывает на вашем сервере:

| зачем | маршрут |
| --- | --- |
| проверка токена | `GET /v1/collectible-usernames?limit=1` |
| найти юзера по номеру | `POST /v1/accounts/resolve-by-phone` |
| прислать код входа | `POST /v1/broadcasts/create` (`target_mode: selected`, один юзер) |
| выдать юзернейм | `POST /v1/collectible-usernames/mint`, если занят — `.../transfer` |
| выдать номер | `POST /v1/collectible-phones/mint`, если занят — `.../transfer` |
| выдать подарок | `POST /v1/gifts/give` с `upgrade: true` |
| выдать звёзды | `POST /v1/accounts/grant-stars` |
| пополнение баланса | `POST /v1/accounts/debit-stars` |

Каждая команда идёт со стабильным `command_id`, повтор не выдаст предмет
дважды.

## webhook (любой другой движок)

LunaX шлёт `POST` на ваш `api_url` с JSON и заголовками:

```http
Content-Type: application/json
X-LunaX-Timestamp: 1784275200
X-LunaX-Signature: sha256=<hex hmac>
```

Подпись — HMAC-SHA256 вашим секретом от строки `<timestamp>.<тело запроса>`.
Проверяйте её и отбрасывайте запросы старше пары минут.

Тело всегда содержит `action`, остальные поля зависят от него:

| action | поля | ответ |
| --- | --- | --- |
| `ping` | — | `{"ok": true}` |
| `resolve_phone` | `phone` (`+79991234567`) | `{"ok": true, "user_id": 123}` или `{"ok": false, "error": "not_found"}` |
| `send_message` | `user_id`, `text` | `{"ok": true}` |
| `grant_username` | `user_id`, `name`, `idempotency_key` | `{"ok": true}` |
| `grant_number` | `user_id`, `phone`, `idempotency_key` | `{"ok": true}` |
| `grant_gift` | `user_id`, `gift_id`, `idempotency_key` | `{"ok": true}` |
| `grant_stars` | `user_id`, `amount`, `idempotency_key` | `{"ok": true}` |
| `debit_stars` | `user_id`, `amount`, `idempotency_key` | `{"ok": true}` или `{"ok": false, "error": "no_funds"}` |

`idempotency_key` одинаковый при повторе — второй раз ничего не выдавайте,
просто верните `ok: true`.

Пример проверки подписи на Go:

```go
func verify(secret string, r *http.Request, body []byte) bool {
	ts := r.Header.Get("X-LunaX-Timestamp")
	m := hmac.New(sha256.New, []byte(secret))
	m.Write([]byte(ts + "."))
	m.Write(body)
	want := "sha256=" + hex.EncodeToString(m.Sum(nil))
	return hmac.Equal([]byte(want), []byte(r.Header.Get("X-LunaX-Signature")))
}
```

## Выставить лоты

Ключ оператора даёт доступ только к лотам вашего сервера.

```bash
# юзернейм по фикс цене
curl -X POST https://lunax.example/api/operator/lots \
  -H "Authorization: Bearer lxo_..." -H "Content-Type: application/json" \
  -d '{"kind":"username","name":"lunar","price":1500}'

# номер на аукцион на 24 часа
curl -X POST https://lunax.example/api/operator/lots \
  -H "Authorization: Bearer lxo_..." -H "Content-Type: application/json" \
  -d '{"kind":"number","name":"+888 0000 0000","price":5000,"auction_hours":24}'

# подарок, gift_id — id подарка в каталоге вашего сервера
curl -X POST https://lunax.example/api/operator/lots \
  -H "Authorization: Bearer lxo_..." -H "Content-Type: application/json" \
  -d '{"kind":"gift","price":120,"gift":{"slug":"plushpepe","collection":"Plush Pepe","num":17,"gift_id":5170145012310081615,"model":{"name":"Gold","rarity":1.2},"backdrop":{"name":"Onyx Black","rarity":2},"symbol":{"name":"Halo","rarity":0.8}}}'
```

| метод | путь | что делает |
| --- | --- | --- |
| `GET` | `/api/operator/server` | данные вашего сервера |
| `GET` | `/api/operator/lots` | все ваши лоты |
| `POST` | `/api/operator/lots` | выставить лот |
| `DELETE` | `/api/operator/lots/{id}` | снять лот, пока его не купили и на нём нет ставок |
| `POST` | `/api/operator/secret` | сменить токен или адрес: `{"api_url": "...", "secret": "..."}` |

## Как устроены деньги

- Баланс юзера в GRAM хранится в LunaX, отдельно на каждого юзера каждой приватки.
- Пополнение списывает звёзды с аккаунта на приватке по курсу `stars_per_gram`.
  `0` — пополнение выключено.
- Ставка сразу замораживает GRAM. Перебили — GRAM вернулись. Ставка в последние
  5 минут продлевает аукцион на 5 минут.
- Купленное лежит в инвентаре LunaX, пока юзер не нажмёт «Вывести». Тогда LunaX
  выдаёт предмет на его аккаунт через API приватки.
