package store

import (
	"context"
	"time"

	"lunax/internal/domain"
)

type Servers interface {
	CreateServer(ctx context.Context, s domain.Server) (domain.Server, error)
	GetServer(ctx context.Context, id int64) (domain.Server, error)
	ServerBySlug(ctx context.Context, slug string) (domain.Server, error)
	ServerByOperator(ctx context.Context, hash []byte) (domain.Server, error)
	ListServers(ctx context.Context, onlyActive bool) ([]domain.Server, error)
	SetServerStatus(ctx context.Context, id int64, status domain.ServerStatus) error
	SetServerSecret(ctx context.Context, id int64, apiURL, secret string) error
}

type Codes interface {
	SaveCode(ctx context.Context, c domain.LoginCode) error
	LatestCode(ctx context.Context, serverID int64, phone string) (domain.LoginCode, error)
	BumpCodeAttempts(ctx context.Context, id int64) error
	DeleteCodes(ctx context.Context, serverID int64, phone string) error
	CountCodesSince(ctx context.Context, serverID int64, phone string, since time.Time) (int, error)
}

type Users interface {
	UpsertUser(ctx context.Context, u domain.User) (domain.User, bool, error)
	GetUser(ctx context.Context, id int64) (domain.User, error)
}

type Sessions interface {
	CreateSession(ctx context.Context, s domain.Session, tokenHash []byte) error
	SessionByToken(ctx context.Context, tokenHash []byte) (domain.Session, error)
	TouchSession(ctx context.Context, id string) error
	ListSessions(ctx context.Context, userID int64) ([]domain.Session, error)
	DeleteSession(ctx context.Context, userID int64, id string) error
	DeleteOtherSessions(ctx context.Context, userID int64, keepID string) error
}

type Ledger interface {
	Credit(ctx context.Context, userID int64, amount domain.Cents, kind domain.HistoryKind, title string) error
	Debit(ctx context.Context, userID int64, amount domain.Cents, kind domain.HistoryKind, title string) error
	History(ctx context.Context, userID int64, limit int) ([]domain.HistoryEntry, error)
}

type Items interface {
	AddItem(ctx context.Context, it domain.Item) (domain.Item, error)
	LockItem(ctx context.Context, userID, id int64) (domain.Item, error)
	SetItemStatus(ctx context.Context, id int64, status domain.ItemStatus) error
	ListItems(ctx context.Context, userID int64) ([]domain.Item, error)
}

type Lots interface {
	CreateLot(ctx context.Context, l domain.Lot) (domain.Lot, error)
	GetLot(ctx context.Context, id int64) (domain.Lot, error)
	LotBySlug(ctx context.Context, serverID int64, kind domain.LotKind, slug string) (domain.Lot, error)
	LockLot(ctx context.Context, id int64) (domain.Lot, error)
	ListLots(ctx context.Context, serverID int64, kind domain.LotKind) ([]domain.Lot, error)
	UpdateLot(ctx context.Context, l domain.Lot) error
	DeleteLot(ctx context.Context, serverID, id int64) error
	DueAuctions(ctx context.Context, now time.Time) ([]int64, error)
}

type Bids interface {
	TopBid(ctx context.Context, lotID int64) (domain.Bid, error)
	AddBid(ctx context.Context, b domain.Bid) error
	DeactivateBid(ctx context.Context, id int64) error
	ListBids(ctx context.Context, lotID int64, limit int) ([]domain.Bid, error)
	UserBids(ctx context.Context, userID int64) ([]domain.Bid, error)
}

type Watchlist interface {
	ToggleWatch(ctx context.Context, userID, lotID int64) (bool, error)
	Watching(ctx context.Context, userID int64) ([]int64, error)
}

// то что можно делать внутри одной транзакции
type Tx interface {
	Users
	Ledger
	Items
	Lots
	Bids
}

type TxRunner interface {
	InTx(ctx context.Context, fn func(Tx) error) error
}

type Store interface {
	Servers
	Codes
	Users
	Sessions
	Ledger
	Items
	Lots
	Bids
	Watchlist
	TxRunner
}
