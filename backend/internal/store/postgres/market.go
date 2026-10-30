package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"lunax/internal/domain"
)

// лоты

const lotCols = `id, server_id, kind, name, slug, price, status, ends_at, gift, created_at`

func scanLot(row interface{ Scan(...any) error }) (domain.Lot, error) {
	var l domain.Lot
	var ends sql.NullTime
	var gift []byte
	if err := row.Scan(&l.ID, &l.ServerID, &l.Kind, &l.Name, &l.Slug, &l.Price, &l.Status, &ends, &gift, &l.CreatedAt); err != nil {
		return l, wrap(err)
	}
	if ends.Valid {
		l.EndsAt = ends.Time
	}
	if len(gift) > 0 {
		l.Gift = &domain.GiftMeta{}
		if err := json.Unmarshal(gift, l.Gift); err != nil {
			return l, err
		}
	}
	return l, nil
}

func nullTime(t time.Time) sql.NullTime { return sql.NullTime{Time: t, Valid: !t.IsZero()} }

func giftJSON(g *domain.GiftMeta) ([]byte, error) {
	if g == nil {
		return nil, nil
	}
	return json.Marshal(g)
}

func (db *DB) CreateLot(ctx context.Context, l domain.Lot) (domain.Lot, error) {
	gift, err := giftJSON(l.Gift)
	if err != nil {
		return l, err
	}
	return scanLot(db.q.QueryRow(ctx, `
		INSERT INTO lots (server_id, kind, name, slug, price, status, ends_at, gift)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING `+lotCols,
		l.ServerID, l.Kind, l.Name, l.Slug, l.Price, l.Status, nullTime(l.EndsAt), gift))
}

func (db *DB) GetLot(ctx context.Context, id int64) (domain.Lot, error) {
	return scanLot(db.q.QueryRow(ctx, `SELECT `+lotCols+` FROM lots WHERE id = $1`, id))
}

func (db *DB) LotBySlug(ctx context.Context, serverID int64, kind domain.LotKind, slug string) (domain.Lot, error) {
	return scanLot(db.q.QueryRow(ctx, `SELECT `+lotCols+` FROM lots WHERE server_id = $1 AND kind = $2 AND slug = $3`, serverID, kind, slug))
}

func (db *DB) LockLot(ctx context.Context, id int64) (domain.Lot, error) {
	return scanLot(db.q.QueryRow(ctx, `SELECT `+lotCols+` FROM lots WHERE id = $1 FOR UPDATE`, id))
}

func (db *DB) ListLots(ctx context.Context, serverID int64, kind domain.LotKind) ([]domain.Lot, error) {
	rows, err := db.q.Query(ctx, `SELECT `+lotCols+` FROM lots WHERE server_id = $1 AND kind = $2 ORDER BY price DESC, id`, serverID, kind)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Lot
	for rows.Next() {
		l, err := scanLot(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, l)
	}
	return out, rows.Err()
}

func (db *DB) UpdateLot(ctx context.Context, l domain.Lot) error {
	return affected(db.q.Exec(ctx, `UPDATE lots SET price = $2, status = $3, ends_at = $4 WHERE id = $1`,
		l.ID, l.Price, l.Status, nullTime(l.EndsAt)))
}

// удалить можно только пока лот никто не трогал
func (db *DB) DeleteLot(ctx context.Context, serverID, id int64) error {
	return affected(db.q.Exec(ctx, `
		DELETE FROM lots WHERE id = $1 AND server_id = $2 AND status <> 'sold'
		AND NOT EXISTS (SELECT 1 FROM bids WHERE lot_id = $1 AND active)
		AND NOT EXISTS (SELECT 1 FROM items WHERE lot_id = $1)`, id, serverID))
}

func (db *DB) DueAuctions(ctx context.Context, now time.Time) ([]int64, error) {
	rows, err := db.q.Query(ctx, `SELECT id FROM lots WHERE status = 'auction' AND ends_at <= $1 LIMIT 100`, now)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

// ставки

const bidCols = `id, lot_id, user_id, masked, amount, active, created_at`

func scanBids(rows interface {
	Next() bool
	Scan(...any) error
	Err() error
	Close()
}) ([]domain.Bid, error) {
	defer rows.Close()
	var out []domain.Bid
	for rows.Next() {
		var b domain.Bid
		if err := rows.Scan(&b.ID, &b.LotID, &b.UserID, &b.Masked, &b.Amount, &b.Active, &b.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, rows.Err()
}

func (db *DB) TopBid(ctx context.Context, lotID int64) (domain.Bid, error) {
	var b domain.Bid
	err := db.q.QueryRow(ctx, `SELECT `+bidCols+` FROM bids WHERE lot_id = $1 AND active ORDER BY amount DESC, id LIMIT 1`, lotID).
		Scan(&b.ID, &b.LotID, &b.UserID, &b.Masked, &b.Amount, &b.Active, &b.CreatedAt)
	return b, wrap(err)
}

func (db *DB) AddBid(ctx context.Context, b domain.Bid) error {
	_, err := db.q.Exec(ctx, `INSERT INTO bids (lot_id, user_id, masked, amount) VALUES ($1, $2, $3, $4)`, b.LotID, b.UserID, b.Masked, b.Amount)
	return wrap(err)
}

func (db *DB) DeactivateBid(ctx context.Context, id int64) error {
	return affected(db.q.Exec(ctx, `UPDATE bids SET active = false WHERE id = $1`, id))
}

func (db *DB) ListBids(ctx context.Context, lotID int64, limit int) ([]domain.Bid, error) {
	rows, err := db.q.Query(ctx, `SELECT `+bidCols+` FROM bids WHERE lot_id = $1 ORDER BY amount DESC, id LIMIT $2`, lotID, limit)
	if err != nil {
		return nil, err
	}
	return scanBids(rows)
}

func (db *DB) UserBids(ctx context.Context, userID int64) ([]domain.Bid, error) {
	rows, err := db.q.Query(ctx, `SELECT `+bidCols+` FROM bids WHERE user_id = $1 AND active ORDER BY id DESC`, userID)
	if err != nil {
		return nil, err
	}
	return scanBids(rows)
}

// избранное

func (db *DB) ToggleWatch(ctx context.Context, userID, lotID int64) (bool, error) {
	tag, err := db.q.Exec(ctx, `DELETE FROM watchlist WHERE user_id = $1 AND lot_id = $2`, userID, lotID)
	if err != nil {
		return false, wrap(err)
	}
	if tag.RowsAffected() > 0 {
		return false, nil
	}
	_, err = db.q.Exec(ctx, `INSERT INTO watchlist (user_id, lot_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`, userID, lotID)
	return true, wrap(err)
}

func (db *DB) Watching(ctx context.Context, userID int64) ([]int64, error) {
	rows, err := db.q.Query(ctx, `SELECT lot_id FROM watchlist WHERE user_id = $1`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}
