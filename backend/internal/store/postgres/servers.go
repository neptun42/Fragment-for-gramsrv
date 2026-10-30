package postgres

import (
	"context"

	"lunax/internal/domain"
)

const serverCols = `id, slug, name, kind, api_url, secret, contact, stars_per_gram, status, operator_hash, created_at`

func (db *DB) scanServer(row interface{ Scan(...any) error }) (domain.Server, error) {
	var s domain.Server
	var sealed string
	err := row.Scan(&s.ID, &s.Slug, &s.Name, &s.Kind, &s.APIURL, &sealed, &s.Contact, &s.StarsPerGram, &s.Status, &s.OperatorHash, &s.CreatedAt)
	if err != nil {
		return s, wrap(err)
	}
	s.Secret, err = db.cipher.Open(sealed)
	return s, err
}

func (db *DB) CreateServer(ctx context.Context, s domain.Server) (domain.Server, error) {
	sealed, err := db.cipher.Seal(s.Secret)
	if err != nil {
		return s, err
	}
	row := db.q.QueryRow(ctx, `
		INSERT INTO servers (slug, name, kind, api_url, secret, contact, stars_per_gram, status, operator_hash)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING `+serverCols,
		s.Slug, s.Name, s.Kind, s.APIURL, sealed, s.Contact, s.StarsPerGram, s.Status, s.OperatorHash)
	return db.scanServer(row)
}

func (db *DB) GetServer(ctx context.Context, id int64) (domain.Server, error) {
	return db.scanServer(db.q.QueryRow(ctx, `SELECT `+serverCols+` FROM servers WHERE id = $1`, id))
}

func (db *DB) ServerBySlug(ctx context.Context, slug string) (domain.Server, error) {
	return db.scanServer(db.q.QueryRow(ctx, `SELECT `+serverCols+` FROM servers WHERE slug = $1`, slug))
}

func (db *DB) ServerByOperator(ctx context.Context, hash []byte) (domain.Server, error) {
	return db.scanServer(db.q.QueryRow(ctx, `SELECT `+serverCols+` FROM servers WHERE operator_hash = $1`, hash))
}

func (db *DB) ListServers(ctx context.Context, onlyActive bool) ([]domain.Server, error) {
	rows, err := db.q.Query(ctx, `SELECT `+serverCols+` FROM servers WHERE NOT $1 OR status = 'active' ORDER BY id`, onlyActive)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Server
	for rows.Next() {
		s, err := db.scanServer(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

func (db *DB) SetServerStatus(ctx context.Context, id int64, status domain.ServerStatus) error {
	return affected(db.q.Exec(ctx, `UPDATE servers SET status = $2 WHERE id = $1`, id, status))
}

func (db *DB) SetServerSecret(ctx context.Context, id int64, apiURL, secret string) error {
	sealed, err := db.cipher.Seal(secret)
	if err != nil {
		return err
	}
	return affected(db.q.Exec(ctx, `UPDATE servers SET api_url = $2, secret = $3 WHERE id = $1`, id, apiURL, sealed))
}
