package httpapi

import (
	"strconv"
	"time"

	"lunax/internal/domain"
	"lunax/internal/service"
)

// json в том виде, который ждёт фронт

func ms(t time.Time) int64 {
	if t.IsZero() {
		return 0
	}
	return t.UnixMilli()
}

func sid(id int64) string { return strconv.FormatInt(id, 10) }

type serverView struct {
	Slug  string  `json:"slug"`
	Name  string  `json:"name"`
	TopUp bool    `json:"topup"`
	Rate  float64 `json:"starsPerGram"`
}

func toServer(s domain.Server) serverView {
	return serverView{Slug: s.Slug, Name: s.Name, TopUp: s.StarsPerGram > 0, Rate: s.StarsPerGram}
}

type adminServerView struct {
	ID        int64   `json:"id"`
	Slug      string  `json:"slug"`
	Name      string  `json:"name"`
	Kind      string  `json:"kind"`
	APIURL    string  `json:"apiUrl"`
	Contact   string  `json:"contact"`
	Rate      float64 `json:"starsPerGram"`
	Status    string  `json:"status"`
	CreatedAt int64   `json:"createdAt"`
}

func toAdminServer(s domain.Server) adminServerView {
	return adminServerView{s.ID, s.Slug, s.Name, s.Kind, s.APIURL, s.Contact, s.StarsPerGram, string(s.Status), ms(s.CreatedAt)}
}

type userView struct {
	ID       int64  `json:"id"`
	Name     string `json:"name"`
	Username string `json:"username"`
	Phone    string `json:"phone"`
	Joined   int64  `json:"joined"`
}

func toUser(u domain.User) userView {
	name := u.Name
	if name == "" {
		name = "LunaX User"
	}
	// фронт рисует аватар по первой букве, пустым быть не может
	username := u.Username
	if username == "" {
		username = "user" + sid(u.ServerUserID%10000)
	}
	return userView{ID: u.ServerUserID, Name: name, Username: username, Phone: u.Phone, Joined: ms(u.CreatedAt)}
}

type sessionView struct {
	ID        string `json:"id"`
	Device    string `json:"device"`
	Place     string `json:"place"`
	CreatedAt int64  `json:"createdAt"`
	LastSeen  int64  `json:"lastSeen"`
}

func toSession(s domain.Session) sessionView {
	return sessionView{s.ID, s.Device, s.Place, ms(s.CreatedAt), ms(s.LastSeen)}
}

type itemView struct {
	ID       string  `json:"id"`
	Kind     string  `json:"kind"`
	Title    string  `json:"title"`
	Value    float64 `json:"value"`
	Ts       int64   `json:"ts"`
	Source   string  `json:"source"`
	Slug     string  `json:"slug,omitempty"`
	Num      int     `json:"num,omitempty"`
	Number   string  `json:"number,omitempty"`
	Username string  `json:"username,omitempty"`
	Amount   int64   `json:"amount,omitempty"`
}

func toItem(it domain.Item) itemView {
	v := itemView{ID: sid(it.ID), Kind: string(it.Kind), Title: it.Title, Value: it.Value.Float(), Ts: ms(it.CreatedAt), Source: it.Source}
	switch it.Kind {
	case domain.ItemGift:
		v.Slug, v.Num = it.Slug, it.Num
	case domain.ItemNumber:
		v.Number = it.Name
	case domain.ItemUsername:
		v.Username = it.Name
	case domain.ItemStars:
		v.Amount = it.Amount
	}
	return v
}

type historyView struct {
	ID     string  `json:"id"`
	Kind   string  `json:"kind"`
	Title  string  `json:"title"`
	Amount float64 `json:"amount"`
	Ts     int64   `json:"ts"`
}

func toHistory(h domain.HistoryEntry) historyView {
	return historyView{sid(h.ID), string(h.Kind), h.Title, h.Amount.Float(), ms(h.CreatedAt)}
}

type bidView struct {
	User   string  `json:"user"`
	Amount float64 `json:"amount"`
	Ts     int64   `json:"ts"`
}

type lotView struct {
	ID         string    `json:"id"`
	Kind       string    `json:"kind"`
	Name       string    `json:"name"`
	Slug       string    `json:"slug"`
	Price      float64   `json:"price"`
	StartPrice float64   `json:"startPrice"`
	MinBid     float64   `json:"minBid"`
	Status     string    `json:"status"`
	EndsAt     int64     `json:"endsAt"`
	Bids       []bidView `json:"bids"`
}

// своя ставка показывается как "Вы"
func toLot(v service.LotView, viewerID int64) lotView {
	out := lotView{
		ID:         sid(v.Lot.ID),
		Kind:       string(v.Lot.Kind),
		Name:       v.Lot.Name,
		Slug:       v.Lot.Slug,
		Price:      v.Highest().Float(),
		StartPrice: v.Lot.Price.Float(),
		MinBid:     service.MinBid(v.Highest()).Float(),
		Status:     string(v.Lot.Status),
		EndsAt:     ms(v.Lot.EndsAt),
		Bids:       []bidView{},
	}
	if v.Lot.Status == domain.LotSold {
		out.Price = v.Lot.Price.Float()
	}
	for _, b := range v.Bids {
		user := b.Masked
		if viewerID != 0 && b.UserID == viewerID {
			user = "Вы"
		}
		out.Bids = append(out.Bids, bidView{user, b.Amount.Float(), ms(b.CreatedAt)})
	}
	return out
}

type giftView struct {
	ID         string       `json:"id"`
	Slug       string       `json:"slug"`
	Collection string       `json:"collection"`
	Num        int          `json:"num"`
	Price      float64      `json:"price"`
	Status     string       `json:"status"`
	Date       int64        `json:"date"`
	Model      domain.Trait `json:"model"`
	Backdrop   domain.Trait `json:"backdrop"`
	Symbol     domain.Trait `json:"symbol"`
}

func toGift(l domain.Lot) giftView {
	g := l.Gift
	if g == nil {
		g = &domain.GiftMeta{}
	}
	date := l.CreatedAt
	if l.Status == domain.LotSold && !l.EndsAt.IsZero() {
		date = l.EndsAt
	}
	return giftView{sid(l.ID), g.Slug, g.Collection, g.Num, l.Price.Float(), string(l.Status), ms(date), g.Model, g.Backdrop, g.Symbol}
}
