package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"regexp"
	"strings"
	"time"

	"lunax/internal/domain"
	"lunax/internal/store"
)

const (
	bidStepPercent = 5
	snipeWindow    = 5 * time.Minute
	bidsShown      = 20
)

type marketStore interface {
	store.Lots
	store.Bids
	store.Watchlist
	store.TxRunner
}

// витрина, покупки и аукционы
type Market struct {
	db  marketStore
	now func() time.Time
	log *slog.Logger
}

func NewMarket(db marketStore, log *slog.Logger) *Market {
	return &Market{db: db, now: time.Now, log: log}
}

type LotView struct {
	Lot  domain.Lot
	Top  domain.Cents
	Bids []domain.Bid
}

// текущая цена аукциона это высшая ставка, если она есть
func (v LotView) Highest() domain.Cents {
	if v.Top > v.Lot.Price {
		return v.Top
	}
	return v.Lot.Price
}

// шаг 5% от высшей, но не меньше 1 грама, всё в целых грамах как на фронте
func MinBid(highest domain.Cents) domain.Cents {
	whole := (int64(highest) + 50) / 100
	step := (whole*bidStepPercent + 50) / 100
	if step < 1 {
		step = 1
	}
	return domain.Cents((whole + step) * 100)
}

func (m *Market) view(ctx context.Context, l domain.Lot) (LotView, error) {
	v := LotView{Lot: l}
	if l.Kind == domain.LotGift {
		return v, nil
	}
	bids, err := m.db.ListBids(ctx, l.ID, bidsShown)
	if err != nil {
		return v, err
	}
	v.Bids = bids
	for _, b := range bids {
		if b.Active && b.Amount > v.Top {
			v.Top = b.Amount
		}
	}
	return v, nil
}

func (m *Market) List(ctx context.Context, serverID int64, kind domain.LotKind) ([]LotView, error) {
	lots, err := m.db.ListLots(ctx, serverID, kind)
	if err != nil {
		return nil, err
	}
	out := make([]LotView, 0, len(lots))
	for _, l := range lots {
		v, err := m.view(ctx, l)
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, nil
}

func (m *Market) Get(ctx context.Context, serverID int64, kind domain.LotKind, slug string) (LotView, error) {
	l, err := m.db.LotBySlug(ctx, serverID, kind, slug)
	if err != nil {
		return LotView{}, err
	}
	return m.view(ctx, l)
}

func (m *Market) GetByID(ctx context.Context, serverID, id int64) (LotView, error) {
	l, err := m.db.GetLot(ctx, id)
	if err != nil {
		return LotView{}, err
	}
	if l.ServerID != serverID {
		return LotView{}, domain.ErrNotFound
	}
	return m.view(ctx, l)
}

// покупка по фикс цене
func (m *Market) Buy(ctx context.Context, u domain.User, lotID int64) (domain.Item, error) {
	var item domain.Item
	err := m.db.InTx(ctx, func(tx store.Tx) error {
		l, err := tx.LockLot(ctx, lotID)
		if err != nil {
			return err
		}
		if l.ServerID != u.ServerID {
			return domain.ErrNotFound
		}
		if l.Status != domain.LotSale {
			return fmt.Errorf("%w: lot is not for sale", domain.ErrConflict)
		}
		if err := tx.Debit(ctx, u.ID, l.Price, domain.HistoryBuy, "Покупка "+lotTitle(l)); err != nil {
			return err
		}
		l.Status, l.EndsAt = domain.LotSold, m.now()
		if err := tx.UpdateLot(ctx, l); err != nil {
			return err
		}
		item, err = tx.AddItem(ctx, itemFromLot(l, u.ID, l.Price, "Маркет"))
		return err
	})
	return item, err
}

// ставка держит грамы, перебили — вернули прошлому лидеру
func (m *Market) Bid(ctx context.Context, u domain.User, lotID int64, amount domain.Cents) error {
	return m.db.InTx(ctx, func(tx store.Tx) error {
		l, err := tx.LockLot(ctx, lotID)
		if err != nil {
			return err
		}
		if l.ServerID != u.ServerID {
			return domain.ErrNotFound
		}
		now := m.now()
		if l.Status != domain.LotAuction || !now.Before(l.EndsAt) {
			return fmt.Errorf("%w: auction is closed", domain.ErrConflict)
		}
		top, err := tx.TopBid(ctx, l.ID)
		hasTop := err == nil
		if err != nil && !errors.Is(err, domain.ErrNotFound) {
			return err
		}
		highest := l.Price
		if hasTop && top.Amount > highest {
			highest = top.Amount
		}
		if amount < MinBid(highest) {
			return fmt.Errorf("%w: bid must be at least %.0f", domain.ErrInvalid, MinBid(highest).Float())
		}

		title := "Ставка: " + lotTitle(l)
		charge := amount
		if hasTop && top.UserID == u.ID {
			charge = amount - top.Amount
		}
		if err := tx.Debit(ctx, u.ID, charge, domain.HistoryBid, title); err != nil {
			return err
		}
		if hasTop {
			if top.UserID != u.ID {
				if err := tx.Credit(ctx, top.UserID, top.Amount, domain.HistoryRefund, "Ставку перебили: "+lotTitle(l)); err != nil {
					return err
				}
			}
			if err := tx.DeactivateBid(ctx, top.ID); err != nil {
				return err
			}
		}
		if err := tx.AddBid(ctx, domain.Bid{LotID: l.ID, UserID: u.ID, Masked: maskUser(u), Amount: amount}); err != nil {
			return err
		}
		// ставка в последние минуты продлевает аукцион
		if l.EndsAt.Sub(now) < snipeWindow {
			l.EndsAt = now.Add(snipeWindow)
			return tx.UpdateLot(ctx, l)
		}
		return nil
	})
}

// закрываем аукцион и отдаём лот лидеру
func (m *Market) Settle(ctx context.Context, lotID int64) error {
	return m.db.InTx(ctx, func(tx store.Tx) error {
		l, err := tx.LockLot(ctx, lotID)
		if err != nil {
			return err
		}
		if l.Status != domain.LotAuction || m.now().Before(l.EndsAt) {
			return nil
		}
		top, err := tx.TopBid(ctx, l.ID)
		if errors.Is(err, domain.ErrNotFound) {
			// никто не поставил, переводим в обычную продажу
			l.Status, l.EndsAt = domain.LotSale, time.Time{}
			return tx.UpdateLot(ctx, l)
		}
		if err != nil {
			return err
		}
		if err := tx.DeactivateBid(ctx, top.ID); err != nil {
			return err
		}
		l.Status, l.Price = domain.LotSold, top.Amount
		if err := tx.UpdateLot(ctx, l); err != nil {
			return err
		}
		_, err = tx.AddItem(ctx, itemFromLot(l, top.UserID, top.Amount, "Аукцион"))
		return err
	})
}

// фоновый цикл закрытия аукционов
func (m *Market) RunSettler(ctx context.Context, every time.Duration) {
	t := time.NewTicker(every)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
		ids, err := m.db.DueAuctions(ctx, m.now())
		if err != nil {
			m.log.Error("due auctions", "err", err)
			continue
		}
		for _, id := range ids {
			if err := m.Settle(ctx, id); err != nil {
				m.log.Error("settle auction", "lot", id, "err", err)
			}
		}
	}
}

func (m *Market) ToggleWatch(ctx context.Context, u domain.User, lotID int64) (bool, error) {
	l, err := m.db.GetLot(ctx, lotID)
	if err != nil {
		return false, err
	}
	if l.ServerID != u.ServerID {
		return false, domain.ErrNotFound
	}
	return m.db.ToggleWatch(ctx, u.ID, lotID)
}

func (m *Market) Watching(ctx context.Context, userID int64) ([]int64, error) {
	return m.db.Watching(ctx, userID)
}

func (m *Market) MyBids(ctx context.Context, userID int64) ([]domain.Bid, error) {
	return m.db.UserBids(ctx, userID)
}

// лоты выставляет оператор своей приватки

type NewLot struct {
	Kind     domain.LotKind
	Name     string
	Price    domain.Cents
	Auction  bool
	Duration time.Duration
	Gift     *domain.GiftMeta
}

var (
	usernameRe = regexp.MustCompile(`^[a-z][a-z0-9_]{3,31}$`)
	slugRe     = regexp.MustCompile(`^[a-z0-9]{2,40}$`)
)

func (m *Market) CreateLot(ctx context.Context, serverID int64, in NewLot) (domain.Lot, error) {
	l := domain.Lot{ServerID: serverID, Kind: in.Kind, Price: in.Price, Status: domain.LotSale}
	if in.Price < 100 {
		return l, fmt.Errorf("%w: price must be at least 1 GRAM", domain.ErrInvalid)
	}
	switch in.Kind {
	case domain.LotUsername:
		name := strings.ToLower(strings.TrimPrefix(strings.TrimSpace(in.Name), "@"))
		if !usernameRe.MatchString(name) {
			return l, fmt.Errorf("%w: username must be 4-32 chars a-z0-9_", domain.ErrInvalid)
		}
		l.Name, l.Slug = name, name
	case domain.LotNumber:
		d := digits(in.Name)
		if len(d) < 7 || len(d) > 15 {
			return l, fmt.Errorf("%w: bad number", domain.ErrInvalid)
		}
		l.Name, l.Slug = formatNumber(d), d
	case domain.LotGift:
		g := in.Gift
		if g == nil || !slugRe.MatchString(g.Slug) || g.Num <= 0 || g.GiftID <= 0 || g.Collection == "" {
			return l, fmt.Errorf("%w: gift needs slug, collection, num and gift_id", domain.ErrInvalid)
		}
		l.Name, l.Slug, l.Gift = fmt.Sprintf("%s #%d", g.Collection, g.Num), fmt.Sprintf("%s-%d", g.Slug, g.Num), g
	default:
		return l, fmt.Errorf("%w: kind must be username, number or gift", domain.ErrInvalid)
	}
	if in.Auction {
		if in.Kind == domain.LotGift {
			return l, fmt.Errorf("%w: gifts are fixed price only", domain.ErrInvalid)
		}
		if in.Duration < time.Hour || in.Duration > 30*24*time.Hour {
			return l, fmt.Errorf("%w: auction must last 1h-30d", domain.ErrInvalid)
		}
		l.Status, l.EndsAt = domain.LotAuction, m.now().Add(in.Duration)
	}
	return m.db.CreateLot(ctx, l)
}

func (m *Market) DeleteLot(ctx context.Context, serverID, lotID int64) error {
	err := m.db.DeleteLot(ctx, serverID, lotID)
	if errors.Is(err, domain.ErrNotFound) {
		return fmt.Errorf("%w: lot not found, sold or has bids", domain.ErrConflict)
	}
	return err
}

func (m *Market) ServerLots(ctx context.Context, serverID int64) ([]domain.Lot, error) {
	var out []domain.Lot
	for _, k := range []domain.LotKind{domain.LotUsername, domain.LotNumber, domain.LotGift} {
		lots, err := m.db.ListLots(ctx, serverID, k)
		if err != nil {
			return nil, err
		}
		out = append(out, lots...)
	}
	return out, nil
}

// +888 0000 0000 как на фрагменте
func formatNumber(d string) string {
	if strings.HasPrefix(d, "888") && len(d) == 11 {
		return "+888 " + d[3:7] + " " + d[7:]
	}
	return "+" + d
}

func maskUser(u domain.User) string {
	p := digits(u.Phone)
	if len(p) < 4 {
		return "•••"
	}
	return "+" + p[:1] + "•••" + p[len(p)-2:]
}
