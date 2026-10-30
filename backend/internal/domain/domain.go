package domain

import (
	"errors"
	"time"
)

// деньги храним в копейках грама, чтобы не ловить ошибки float
type Cents int64

func (c Cents) Float() float64 { return float64(c) / 100 }

func CentsFromFloat(v float64) Cents {
	if v < 0 {
		return Cents(v*100 - 0.5)
	}
	return Cents(v*100 + 0.5)
}

var (
	ErrNotFound          = errors.New("not found")
	ErrNoFunds           = errors.New("not enough balance")
	ErrConflict          = errors.New("conflict")
	ErrForbidden         = errors.New("forbidden")
	ErrInvalid           = errors.New("invalid input")
	ErrUnauthorized      = errors.New("unauthorized")
	ErrServerUnavailable = errors.New("server unavailable")
)

type ServerStatus string

const (
	ServerPending  ServerStatus = "pending"
	ServerActive   ServerStatus = "active"
	ServerDisabled ServerStatus = "disabled"
)

// привязанная приватка
type Server struct {
	ID           int64
	Slug         string
	Name         string
	Kind         string
	APIURL       string
	Secret       string
	Contact      string
	StarsPerGram float64
	Status       ServerStatus
	OperatorHash []byte
	CreatedAt    time.Time
}

type User struct {
	ID           int64
	ServerID     int64
	ServerUserID int64
	Phone        string
	Name         string
	Username     string
	Balance      Cents
	CreatedAt    time.Time
}

type Session struct {
	ID        string
	UserID    int64
	Device    string
	Place     string
	CreatedAt time.Time
	LastSeen  time.Time
}

type ItemKind string

const (
	ItemGift     ItemKind = "gift"
	ItemNumber   ItemKind = "number"
	ItemUsername ItemKind = "username"
	ItemStars    ItemKind = "stars"
)

type ItemStatus string

const (
	ItemOwned     ItemStatus = "owned"
	ItemWithdrawn ItemStatus = "withdrawn"
	ItemSold      ItemStatus = "sold"
)

type Item struct {
	ID        int64
	UserID    int64
	LotID     int64
	Kind      ItemKind
	Title     string
	Name      string
	Slug      string
	Num       int
	GiftID    int64
	Amount    int64
	Value     Cents
	Source    string
	Status    ItemStatus
	CreatedAt time.Time
}

type HistoryKind string

const (
	HistoryBuy      HistoryKind = "buy"
	HistorySell     HistoryKind = "sell"
	HistoryTopup    HistoryKind = "topup"
	HistoryBid      HistoryKind = "bid"
	HistoryRefund   HistoryKind = "refund"
	HistoryWin      HistoryKind = "win"
	HistoryWithdraw HistoryKind = "withdraw"
)

type HistoryEntry struct {
	ID        int64
	UserID    int64
	Kind      HistoryKind
	Title     string
	Amount    Cents
	CreatedAt time.Time
}

type LotKind string

const (
	LotUsername LotKind = "username"
	LotNumber   LotKind = "number"
	LotGift     LotKind = "gift"
)

type LotStatus string

const (
	LotAuction LotStatus = "auction"
	LotSale    LotStatus = "sale"
	LotSold    LotStatus = "sold"
)

// атрибут гифта: модель, фон, символ
type Trait struct {
	Name   string  `json:"name"`
	Rarity float64 `json:"rarity"`
}

type GiftMeta struct {
	Slug       string `json:"slug"`
	Collection string `json:"collection"`
	Num        int    `json:"num"`
	GiftID     int64  `json:"gift_id"`
	Model      Trait  `json:"model"`
	Backdrop   Trait  `json:"backdrop"`
	Symbol     Trait  `json:"symbol"`
}

type Lot struct {
	ID        int64
	ServerID  int64
	Kind      LotKind
	Name      string
	Slug      string
	Price     Cents
	Status    LotStatus
	EndsAt    time.Time
	Gift      *GiftMeta
	CreatedAt time.Time
}

type Bid struct {
	ID        int64
	LotID     int64
	UserID    int64
	Masked    string
	Amount    Cents
	Active    bool
	CreatedAt time.Time
}

type LoginCode struct {
	ID           int64
	ServerID     int64
	Phone        string
	ServerUserID int64
	Hash         []byte
	Attempts     int
	ExpiresAt    time.Time
}
