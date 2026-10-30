CREATE TABLE IF NOT EXISTS servers (
    id             BIGSERIAL PRIMARY KEY,
    slug           TEXT NOT NULL UNIQUE,
    name           TEXT NOT NULL,
    kind           TEXT NOT NULL,
    api_url        TEXT NOT NULL,
    secret         TEXT NOT NULL,
    contact        TEXT NOT NULL DEFAULT '',
    stars_per_gram DOUBLE PRECISION NOT NULL DEFAULT 0,
    status         TEXT NOT NULL DEFAULT 'pending',
    operator_hash  BYTEA NOT NULL UNIQUE,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS users (
    id             BIGSERIAL PRIMARY KEY,
    server_id      BIGINT NOT NULL REFERENCES servers(id),
    server_user_id BIGINT NOT NULL,
    phone          TEXT NOT NULL,
    name           TEXT NOT NULL DEFAULT '',
    username       TEXT NOT NULL DEFAULT '',
    balance        BIGINT NOT NULL DEFAULT 0 CHECK (balance >= 0),
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (server_id, server_user_id)
);

CREATE TABLE IF NOT EXISTS login_codes (
    id             BIGSERIAL PRIMARY KEY,
    server_id      BIGINT NOT NULL REFERENCES servers(id),
    phone          TEXT NOT NULL,
    server_user_id BIGINT NOT NULL,
    hash           BYTEA NOT NULL,
    attempts       INT NOT NULL DEFAULT 0,
    expires_at     TIMESTAMPTZ NOT NULL,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS login_codes_phone ON login_codes (server_id, phone, created_at DESC);

CREATE TABLE IF NOT EXISTS sessions (
    id         TEXT PRIMARY KEY,
    user_id    BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash BYTEA NOT NULL UNIQUE,
    device     TEXT NOT NULL DEFAULT '',
    place      TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_seen  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS sessions_user ON sessions (user_id);

CREATE TABLE IF NOT EXISTS history (
    id         BIGSERIAL PRIMARY KEY,
    user_id    BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    kind       TEXT NOT NULL,
    title      TEXT NOT NULL,
    amount     BIGINT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS history_user ON history (user_id, id DESC);

CREATE TABLE IF NOT EXISTS lots (
    id         BIGSERIAL PRIMARY KEY,
    server_id  BIGINT NOT NULL REFERENCES servers(id),
    kind       TEXT NOT NULL,
    name       TEXT NOT NULL,
    slug       TEXT NOT NULL,
    price      BIGINT NOT NULL CHECK (price > 0),
    status     TEXT NOT NULL,
    ends_at    TIMESTAMPTZ,
    gift       JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (server_id, kind, slug)
);
CREATE INDEX IF NOT EXISTS lots_auctions ON lots (ends_at) WHERE status = 'auction';

CREATE TABLE IF NOT EXISTS items (
    id         BIGSERIAL PRIMARY KEY,
    user_id    BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    lot_id     BIGINT NOT NULL REFERENCES lots(id),
    kind       TEXT NOT NULL,
    title      TEXT NOT NULL,
    name       TEXT NOT NULL DEFAULT '',
    slug       TEXT NOT NULL DEFAULT '',
    num        INT NOT NULL DEFAULT 0,
    gift_id    BIGINT NOT NULL DEFAULT 0,
    amount     BIGINT NOT NULL DEFAULT 0,
    value      BIGINT NOT NULL,
    source     TEXT NOT NULL,
    status     TEXT NOT NULL DEFAULT 'owned',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS items_user ON items (user_id, id DESC);

CREATE TABLE IF NOT EXISTS bids (
    id         BIGSERIAL PRIMARY KEY,
    lot_id     BIGINT NOT NULL REFERENCES lots(id) ON DELETE CASCADE,
    user_id    BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    masked     TEXT NOT NULL,
    amount     BIGINT NOT NULL,
    active     BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS bids_lot ON bids (lot_id, amount DESC);

CREATE TABLE IF NOT EXISTS watchlist (
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    lot_id  BIGINT NOT NULL REFERENCES lots(id) ON DELETE CASCADE,
    PRIMARY KEY (user_id, lot_id)
);
