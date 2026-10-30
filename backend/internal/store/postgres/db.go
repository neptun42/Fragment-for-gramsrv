package postgres

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"sort"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"lunax/internal/domain"
	"lunax/internal/store"
)

//go:embed migrations/*.sql
var migrations embed.FS

// шифрует токены приваток перед записью в бд
type Cipher interface {
	Seal(plain string) (string, error)
	Open(sealed string) (string, error)
}

type querier interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// одни и те же методы работают и на пуле и внутри транзакции
type DB struct {
	pool   *pgxpool.Pool
	q      querier
	cipher Cipher
}

var _ store.Store = (*DB)(nil)

func Open(ctx context.Context, dsn string, cipher Cipher) (*DB, error) {
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, err
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	return &DB{pool: pool, q: pool, cipher: cipher}, nil
}

func (db *DB) Close() { db.pool.Close() }

func (db *DB) Migrate(ctx context.Context) error {
	files, err := migrations.ReadDir("migrations")
	if err != nil {
		return err
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Name() < files[j].Name() })
	if _, err := db.pool.Exec(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (name TEXT PRIMARY KEY)`); err != nil {
		return err
	}
	for _, f := range files {
		var done bool
		if err := db.pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM schema_migrations WHERE name = $1)`, f.Name()).Scan(&done); err != nil {
			return err
		}
		if done {
			continue
		}
		sql, err := migrations.ReadFile("migrations/" + f.Name())
		if err != nil {
			return err
		}
		if _, err := db.pool.Exec(ctx, string(sql)); err != nil {
			return fmt.Errorf("migration %s: %w", f.Name(), err)
		}
		if _, err := db.pool.Exec(ctx, `INSERT INTO schema_migrations (name) VALUES ($1)`, f.Name()); err != nil {
			return err
		}
	}
	return nil
}

func (db *DB) InTx(ctx context.Context, fn func(store.Tx) error) error {
	tx, err := db.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if err := fn(&DB{pool: db.pool, q: tx, cipher: db.cipher}); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// переводим ошибки pgx в доменные
func wrap(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ErrNotFound
	}
	var pe *pgconn.PgError
	if errors.As(err, &pe) {
		switch pe.Code {
		case "23505":
			return fmt.Errorf("%w: %s", domain.ErrConflict, pe.ConstraintName)
		case "23514":
			return domain.ErrNoFunds
		}
	}
	return err
}
