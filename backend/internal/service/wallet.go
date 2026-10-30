package service

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"strconv"
	"time"

	"lunax/internal/domain"
	"lunax/internal/network"
	"lunax/internal/secret"
	"lunax/internal/store"
)

type walletStore interface {
	GetServer(ctx context.Context, id int64) (domain.Server, error)
	store.Ledger
	store.Items
	store.TxRunner
}

// баланс, пополнение звёздами с приватки, продажа и вывод предметов
type Wallet struct {
	db   walletStore
	nets Networks
	log  *slog.Logger
}

func NewWallet(db walletStore, nets Networks, log *slog.Logger) *Wallet {
	return &Wallet{db: db, nets: nets, log: log}
}

func (w *Wallet) History(ctx context.Context, userID int64) ([]domain.HistoryEntry, error) {
	return w.db.History(ctx, userID, 200)
}

func (w *Wallet) Items(ctx context.Context, userID int64) ([]domain.Item, error) {
	return w.db.ListItems(ctx, userID)
}

// списываем звёзды на приватке и начисляем грамы по курсу сервера
func (w *Wallet) TopUp(ctx context.Context, u domain.User, gram int64) (domain.Cents, error) {
	if gram < 1 || gram > 1_000_000 {
		return 0, fmt.Errorf("%w: amount must be 1-1000000", domain.ErrInvalid)
	}
	srv, err := w.db.GetServer(ctx, u.ServerID)
	if err != nil {
		return 0, err
	}
	if srv.StarsPerGram <= 0 {
		return 0, fmt.Errorf("%w: top up is disabled on this server", domain.ErrForbidden)
	}
	net, err := w.nets.Get(ctx, u.ServerID)
	if err != nil {
		return 0, err
	}
	stars := int64(math.Ceil(float64(gram) * srv.StarsPerGram))
	op := network.Op{Key: "topup:" + strconv.FormatInt(u.ID, 10) + ":" + secret.Hex(8), Reason: "LunaX top up"}
	if err := net.DebitStars(ctx, op, u.ServerUserID, stars); err != nil {
		return 0, err
	}
	amount := domain.Cents(gram * 100)
	title := fmt.Sprintf("Пополнение · %d ⭐", stars)
	if err := w.db.Credit(ctx, u.ID, amount, domain.HistoryTopup, title); err != nil {
		// звёзды уже списаны, это надо разбирать руками
		w.log.Error("topup credit failed after debit", "user", u.ID, "stars", stars, "op", op.Key, "err", err)
		return 0, err
	}
	return amount, nil
}

// продажа обратно на витрину по цене покупки
func (w *Wallet) Sell(ctx context.Context, u domain.User, itemID int64) (domain.Cents, error) {
	var value domain.Cents
	err := w.db.InTx(ctx, func(tx store.Tx) error {
		it, err := tx.LockItem(ctx, u.ID, itemID)
		if err != nil {
			return err
		}
		if it.Status != domain.ItemOwned {
			return fmt.Errorf("%w: item is %s", domain.ErrConflict, it.Status)
		}
		if err := tx.SetItemStatus(ctx, it.ID, domain.ItemSold); err != nil {
			return err
		}
		l, err := tx.LockLot(ctx, it.LotID)
		if err != nil {
			return err
		}
		l.Status, l.Price, l.EndsAt = domain.LotSale, it.Value, time.Time{}
		if err := tx.UpdateLot(ctx, l); err != nil {
			return err
		}
		value = it.Value
		return tx.Credit(ctx, u.ID, it.Value, domain.HistorySell, "Продажа: "+itemTitle(it))
	})
	return value, err
}

// вывод на аккаунт в приватке, ключ идемпотентности не даст выдать дважды
func (w *Wallet) Withdraw(ctx context.Context, u domain.User, itemID int64) error {
	var it domain.Item
	err := w.db.InTx(ctx, func(tx store.Tx) error {
		var err error
		if it, err = tx.LockItem(ctx, u.ID, itemID); err != nil {
			return err
		}
		if it.Status != domain.ItemOwned {
			return fmt.Errorf("%w: item is %s", domain.ErrConflict, it.Status)
		}
		return tx.SetItemStatus(ctx, it.ID, domain.ItemWithdrawn)
	})
	if err != nil {
		return err
	}
	if err := w.deliver(ctx, u, it); err != nil {
		// не вышло, возвращаем предмет в инвентарь
		if rerr := w.db.SetItemStatus(ctx, it.ID, domain.ItemOwned); rerr != nil {
			w.log.Error("withdraw rollback failed", "item", it.ID, "err", rerr)
		}
		return err
	}
	return w.db.Credit(ctx, u.ID, 0, domain.HistoryWithdraw, "Вывод: "+itemTitle(it))
}

func (w *Wallet) deliver(ctx context.Context, u domain.User, it domain.Item) error {
	net, err := w.nets.Get(ctx, u.ServerID)
	if err != nil {
		return err
	}
	op := network.Op{Key: "item:" + strconv.FormatInt(it.ID, 10), Reason: "LunaX withdraw"}
	switch it.Kind {
	case domain.ItemUsername:
		return net.GrantUsername(ctx, op, u.ServerUserID, it.Name)
	case domain.ItemNumber:
		return net.GrantNumber(ctx, op, u.ServerUserID, it.Name)
	case domain.ItemGift:
		return net.GrantGift(ctx, op, u.ServerUserID, it.GiftID)
	case domain.ItemStars:
		return net.GrantStars(ctx, op, u.ServerUserID, it.Amount)
	}
	return fmt.Errorf("%w: unknown item kind %s", domain.ErrInvalid, it.Kind)
}

func itemTitle(it domain.Item) string {
	switch it.Kind {
	case domain.ItemGift:
		return fmt.Sprintf("%s #%d", it.Title, it.Num)
	case domain.ItemNumber:
		return it.Name
	}
	return it.Title
}
