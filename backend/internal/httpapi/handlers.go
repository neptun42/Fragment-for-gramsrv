package httpapi

import (
	"errors"
	"net/http"
	"time"

	"lunax/internal/domain"
	"lunax/internal/service"
)

// сервера

func (a *API) listServers(w http.ResponseWriter, r *http.Request) {
	list, err := a.Servers.Public(r.Context())
	if err != nil {
		writeErr(w, a.Log, err)
		return
	}
	out := make([]serverView, 0, len(list))
	for _, s := range list {
		out = append(out, toServer(s))
	}
	writeJSON(w, 200, out)
}

func (a *API) registerServer(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Slug         string  `json:"slug"`
		Name         string  `json:"name"`
		Kind         string  `json:"kind"`
		APIURL       string  `json:"api_url"`
		Secret       string  `json:"secret"`
		Contact      string  `json:"contact"`
		StarsPerGram float64 `json:"stars_per_gram"`
	}
	if !readJSON(w, r, &in) {
		return
	}
	srv, key, err := a.Servers.Register(r.Context(), service.RegisterInput{
		Slug: in.Slug, Name: in.Name, Kind: in.Kind, APIURL: in.APIURL, Secret: in.Secret, Contact: in.Contact, StarsPerGram: in.StarsPerGram,
	})
	if err != nil {
		writeErr(w, a.Log, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{
		"server":       toServer(srv),
		"status":       srv.Status,
		"operator_key": key,
	})
}

// вход

func (a *API) requestCode(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Server string `json:"server"`
		Phone  string `json:"phone"`
	}
	if !readJSON(w, r, &in) {
		return
	}
	if err := a.Auth.RequestCode(r.Context(), in.Server, in.Phone); err != nil {
		writeErr(w, a.Log, err)
		return
	}
	writeJSON(w, 200, map[string]bool{"sent": true})
}

func (a *API) verifyCode(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Server string `json:"server"`
		Phone  string `json:"phone"`
		Code   string `json:"code"`
		Device string `json:"device"`
		Place  string `json:"place"`
	}
	if !readJSON(w, r, &in) {
		return
	}
	res, err := a.Auth.VerifyCode(r.Context(), in.Server, in.Phone, in.Code, in.Device, in.Place)
	if err != nil {
		writeErr(w, a.Log, err)
		return
	}
	writeJSON(w, 200, map[string]any{"token": res.Token, "bonus": res.Bonus.Float()})
}

func (a *API) logout(w http.ResponseWriter, r *http.Request) {
	u, _ := userFrom(r.Context())
	if err := a.Auth.Logout(r.Context(), u.ID, sessionFrom(r.Context()).ID); err != nil {
		writeErr(w, a.Log, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *API) kickSession(w http.ResponseWriter, r *http.Request) {
	u, _ := userFrom(r.Context())
	if err := a.Auth.Kick(r.Context(), u.ID, sessionFrom(r.Context()).ID, r.PathValue("id")); err != nil {
		writeErr(w, a.Log, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *API) kickOthers(w http.ResponseWriter, r *http.Request) {
	u, _ := userFrom(r.Context())
	if err := a.Auth.KickOthers(r.Context(), u.ID, sessionFrom(r.Context()).ID); err != nil {
		writeErr(w, a.Log, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// всё про юзера одним запросом, фронт держит это в сторе
func (a *API) me(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	u, _ := userFrom(ctx)
	sess := sessionFrom(ctx)

	items, err := a.Wallet.Items(ctx, u.ID)
	if err != nil {
		writeErr(w, a.Log, err)
		return
	}
	history, err := a.Wallet.History(ctx, u.ID)
	if err != nil {
		writeErr(w, a.Log, err)
		return
	}
	sessions, err := a.Auth.Sessions(ctx, u.ID)
	if err != nil {
		writeErr(w, a.Log, err)
		return
	}
	bids, err := a.Market.MyBids(ctx, u.ID)
	if err != nil {
		writeErr(w, a.Log, err)
		return
	}
	watching, err := a.Market.Watching(ctx, u.ID)
	if err != nil {
		writeErr(w, a.Log, err)
		return
	}

	out := struct {
		User             userView                  `json:"user"`
		Server           string                    `json:"server"`
		Balance          float64                   `json:"balance"`
		Inventory        []itemView                `json:"inventory"`
		History          []historyView             `json:"history"`
		Bids             map[string]map[string]any `json:"bids"`
		Watching         []string                  `json:"watching"`
		Sessions         []sessionView             `json:"sessions"`
		CurrentSessionID string                    `json:"currentSessionId"`
	}{
		User:             toUser(u),
		Balance:          u.Balance.Float(),
		Inventory:        []itemView{},
		History:          []historyView{},
		Bids:             map[string]map[string]any{},
		Watching:         []string{},
		Sessions:         []sessionView{},
		CurrentSessionID: sess.ID,
	}
	if srv, err := a.serverByID(r, u.ServerID); err == nil {
		out.Server = srv.Slug
	}
	for _, it := range items {
		out.Inventory = append(out.Inventory, toItem(it))
	}
	for _, h := range history {
		out.History = append(out.History, toHistory(h))
	}
	for _, b := range bids {
		out.Bids[sid(b.LotID)] = map[string]any{"amount": b.Amount.Float(), "ts": ms(b.CreatedAt)}
	}
	for _, id := range watching {
		out.Watching = append(out.Watching, sid(id))
	}
	for _, s := range sessions {
		out.Sessions = append(out.Sessions, toSession(s))
	}
	writeJSON(w, 200, out)
}

func (a *API) serverByID(r *http.Request, id int64) (domain.Server, error) {
	list, err := a.Servers.Public(r.Context())
	if err != nil {
		return domain.Server{}, err
	}
	for _, s := range list {
		if s.ID == id {
			return s, nil
		}
	}
	return domain.Server{}, domain.ErrNotFound
}

// витрина

// залогиненный видит свою приватку, гость выбирает через ?server=
func (a *API) viewerServer(r *http.Request) (int64, int64, error) {
	if u, ok := userFrom(r.Context()); ok {
		return u.ServerID, u.ID, nil
	}
	if slug := r.URL.Query().Get("server"); slug != "" {
		srv, err := a.Servers.BySlug(r.Context(), slug)
		if err != nil || srv.Status != domain.ServerActive {
			return 0, 0, domain.ErrNotFound
		}
		return srv.ID, 0, nil
	}
	list, err := a.Servers.Public(r.Context())
	if err != nil || len(list) == 0 {
		return 0, 0, err
	}
	return list[0].ID, 0, nil
}

func lotKind(s string) (domain.LotKind, bool) {
	switch domain.LotKind(s) {
	case domain.LotUsername, domain.LotNumber, domain.LotGift:
		return domain.LotKind(s), true
	}
	return "", false
}

func (a *API) listLots(w http.ResponseWriter, r *http.Request) {
	kind, ok := lotKind(r.URL.Query().Get("kind"))
	if !ok || kind == domain.LotGift {
		writeErr(w, a.Log, errors.Join(domain.ErrInvalid, errors.New("kind must be username or number")))
		return
	}
	serverID, viewer, err := a.viewerServer(r)
	if err != nil {
		writeErr(w, a.Log, err)
		return
	}
	out := []lotView{}
	if serverID != 0 {
		list, err := a.Market.List(r.Context(), serverID, kind)
		if err != nil {
			writeErr(w, a.Log, err)
			return
		}
		for _, v := range list {
			out = append(out, toLot(v, viewer))
		}
	}
	writeJSON(w, 200, out)
}

func (a *API) getLot(w http.ResponseWriter, r *http.Request) {
	kind, ok := lotKind(r.PathValue("kind"))
	if !ok || kind == domain.LotGift {
		writeErr(w, a.Log, domain.ErrNotFound)
		return
	}
	serverID, viewer, err := a.viewerServer(r)
	if err != nil {
		writeErr(w, a.Log, err)
		return
	}
	v, err := a.Market.Get(r.Context(), serverID, kind, r.PathValue("slug"))
	if err != nil {
		writeErr(w, a.Log, err)
		return
	}
	writeJSON(w, 200, toLot(v, viewer))
}

func (a *API) listGifts(w http.ResponseWriter, r *http.Request) {
	serverID, _, err := a.viewerServer(r)
	if err != nil {
		writeErr(w, a.Log, err)
		return
	}
	out := []giftView{}
	if serverID != 0 {
		list, err := a.Market.List(r.Context(), serverID, domain.LotGift)
		if err != nil {
			writeErr(w, a.Log, err)
			return
		}
		for _, v := range list {
			out = append(out, toGift(v.Lot))
		}
	}
	writeJSON(w, 200, out)
}

func (a *API) getGift(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r, "id")
	if !ok {
		writeErr(w, a.Log, domain.ErrNotFound)
		return
	}
	serverID, _, err := a.viewerServer(r)
	if err != nil {
		writeErr(w, a.Log, err)
		return
	}
	v, err := a.Market.GetByID(r.Context(), serverID, id)
	if err != nil || v.Lot.Kind != domain.LotGift {
		writeErr(w, a.Log, domain.ErrNotFound)
		return
	}
	writeJSON(w, 200, toGift(v.Lot))
}

// действия юзера

func (a *API) buy(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r, "id")
	if !ok {
		writeErr(w, a.Log, domain.ErrNotFound)
		return
	}
	u, _ := userFrom(r.Context())
	it, err := a.Market.Buy(r.Context(), u, id)
	if err != nil {
		writeErr(w, a.Log, err)
		return
	}
	writeJSON(w, 200, toItem(it))
}

func (a *API) bid(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r, "id")
	if !ok {
		writeErr(w, a.Log, domain.ErrNotFound)
		return
	}
	var in struct {
		Amount float64 `json:"amount"`
	}
	if !readJSON(w, r, &in) {
		return
	}
	u, _ := userFrom(r.Context())
	if err := a.Market.Bid(r.Context(), u, id, domain.CentsFromFloat(in.Amount)); err != nil {
		writeErr(w, a.Log, err)
		return
	}
	writeJSON(w, 200, map[string]bool{"ok": true})
}

func (a *API) watch(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r, "id")
	if !ok {
		writeErr(w, a.Log, domain.ErrNotFound)
		return
	}
	u, _ := userFrom(r.Context())
	on, err := a.Market.ToggleWatch(r.Context(), u, id)
	if err != nil {
		writeErr(w, a.Log, err)
		return
	}
	writeJSON(w, 200, map[string]bool{"watching": on})
}

func (a *API) topUp(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Amount int64 `json:"amount"`
	}
	if !readJSON(w, r, &in) {
		return
	}
	u, _ := userFrom(r.Context())
	got, err := a.Wallet.TopUp(r.Context(), u, in.Amount)
	if err != nil {
		writeErr(w, a.Log, err)
		return
	}
	writeJSON(w, 200, map[string]float64{"credited": got.Float()})
}

func (a *API) sell(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r, "id")
	if !ok {
		writeErr(w, a.Log, domain.ErrNotFound)
		return
	}
	u, _ := userFrom(r.Context())
	got, err := a.Wallet.Sell(r.Context(), u, id)
	if err != nil {
		writeErr(w, a.Log, err)
		return
	}
	writeJSON(w, 200, map[string]float64{"credited": got.Float()})
}

func (a *API) withdraw(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r, "id")
	if !ok {
		writeErr(w, a.Log, domain.ErrNotFound)
		return
	}
	u, _ := userFrom(r.Context())
	if err := a.Wallet.Withdraw(r.Context(), u, id); err != nil {
		writeErr(w, a.Log, err)
		return
	}
	writeJSON(w, 200, map[string]bool{"ok": true})
}

// оператор

func (a *API) operatorServer(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, toAdminServer(serverFrom(r.Context())))
}

func (a *API) operatorLots(w http.ResponseWriter, r *http.Request) {
	lots, err := a.Market.ServerLots(r.Context(), serverFrom(r.Context()).ID)
	if err != nil {
		writeErr(w, a.Log, err)
		return
	}
	out := make([]map[string]any, 0, len(lots))
	for _, l := range lots {
		out = append(out, map[string]any{
			"id": sid(l.ID), "kind": l.Kind, "name": l.Name, "slug": l.Slug,
			"price": l.Price.Float(), "status": l.Status, "endsAt": ms(l.EndsAt), "gift": l.Gift,
		})
	}
	writeJSON(w, 200, out)
}

func (a *API) createLot(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Kind         string           `json:"kind"`
		Name         string           `json:"name"`
		Price        float64          `json:"price"`
		AuctionHours int              `json:"auction_hours"`
		Gift         *domain.GiftMeta `json:"gift"`
	}
	if !readJSON(w, r, &in) {
		return
	}
	l, err := a.Market.CreateLot(r.Context(), serverFrom(r.Context()).ID, service.NewLot{
		Kind:     domain.LotKind(in.Kind),
		Name:     in.Name,
		Price:    domain.CentsFromFloat(in.Price),
		Auction:  in.AuctionHours > 0,
		Duration: time.Duration(in.AuctionHours) * time.Hour,
		Gift:     in.Gift,
	})
	if err != nil {
		writeErr(w, a.Log, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"id": sid(l.ID), "slug": l.Slug, "status": l.Status})
}

func (a *API) deleteLot(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r, "id")
	if !ok {
		writeErr(w, a.Log, domain.ErrNotFound)
		return
	}
	if err := a.Market.DeleteLot(r.Context(), serverFrom(r.Context()).ID, id); err != nil {
		writeErr(w, a.Log, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *API) rotateSecret(w http.ResponseWriter, r *http.Request) {
	var in struct {
		APIURL string `json:"api_url"`
		Secret string `json:"secret"`
	}
	if !readJSON(w, r, &in) {
		return
	}
	if err := a.Servers.RotateSecret(r.Context(), serverFrom(r.Context()), in.APIURL, in.Secret); err != nil {
		writeErr(w, a.Log, err)
		return
	}
	writeJSON(w, 200, map[string]bool{"ok": true})
}

// админка владельца

func (a *API) adminServers(w http.ResponseWriter, r *http.Request) {
	list, err := a.Servers.All(r.Context())
	if err != nil {
		writeErr(w, a.Log, err)
		return
	}
	out := make([]adminServerView, 0, len(list))
	for _, s := range list {
		out = append(out, toAdminServer(s))
	}
	writeJSON(w, 200, out)
}

func (a *API) adminSetStatus(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r, "id")
	if !ok {
		writeErr(w, a.Log, domain.ErrNotFound)
		return
	}
	var in struct {
		Status string `json:"status"`
	}
	if !readJSON(w, r, &in) {
		return
	}
	if err := a.Servers.SetStatus(r.Context(), id, domain.ServerStatus(in.Status)); err != nil {
		writeErr(w, a.Log, err)
		return
	}
	writeJSON(w, 200, map[string]bool{"ok": true})
}
