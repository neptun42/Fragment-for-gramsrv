package httpapi

import (
	"context"
	"crypto/subtle"
	"log/slog"
	"net/http"
	"time"

	"lunax/internal/domain"
	"lunax/internal/service"
)

// хендлеры зависят от этих интерфейсов, а не от конкретных сервисов

type Auth interface {
	RequestCode(ctx context.Context, serverSlug, phone string) error
	VerifyCode(ctx context.Context, serverSlug, phone, code, device, place string) (service.LoginResult, error)
	Authenticate(ctx context.Context, token string) (domain.Session, domain.User, error)
	Sessions(ctx context.Context, userID int64) ([]domain.Session, error)
	Logout(ctx context.Context, userID int64, sessionID string) error
	Kick(ctx context.Context, userID int64, current, target string) error
	KickOthers(ctx context.Context, userID int64, current string) error
}

type Market interface {
	List(ctx context.Context, serverID int64, kind domain.LotKind) ([]service.LotView, error)
	Get(ctx context.Context, serverID int64, kind domain.LotKind, slug string) (service.LotView, error)
	GetByID(ctx context.Context, serverID, id int64) (service.LotView, error)
	Buy(ctx context.Context, u domain.User, lotID int64) (domain.Item, error)
	Bid(ctx context.Context, u domain.User, lotID int64, amount domain.Cents) error
	ToggleWatch(ctx context.Context, u domain.User, lotID int64) (bool, error)
	Watching(ctx context.Context, userID int64) ([]int64, error)
	MyBids(ctx context.Context, userID int64) ([]domain.Bid, error)
	CreateLot(ctx context.Context, serverID int64, in service.NewLot) (domain.Lot, error)
	DeleteLot(ctx context.Context, serverID, lotID int64) error
	ServerLots(ctx context.Context, serverID int64) ([]domain.Lot, error)
}

type Wallet interface {
	History(ctx context.Context, userID int64) ([]domain.HistoryEntry, error)
	Items(ctx context.Context, userID int64) ([]domain.Item, error)
	TopUp(ctx context.Context, u domain.User, gram int64) (domain.Cents, error)
	Sell(ctx context.Context, u domain.User, itemID int64) (domain.Cents, error)
	Withdraw(ctx context.Context, u domain.User, itemID int64) error
}

type Servers interface {
	Register(ctx context.Context, in service.RegisterInput) (domain.Server, string, error)
	Public(ctx context.Context) ([]domain.Server, error)
	All(ctx context.Context) ([]domain.Server, error)
	BySlug(ctx context.Context, slug string) (domain.Server, error)
	SetStatus(ctx context.Context, id int64, status domain.ServerStatus) error
	Operator(ctx context.Context, key string) (domain.Server, error)
	RotateSecret(ctx context.Context, srv domain.Server, apiURL, secret string) error
}

type Deps struct {
	Auth       Auth
	Market     Market
	Wallet     Wallet
	Servers    Servers
	AdminToken string
	CORSOrigin string
	Log        *slog.Logger
}

type API struct{ Deps }

func New(d Deps) http.Handler {
	a := &API{d}
	mux := http.NewServeMux()

	// публичное
	mux.HandleFunc("GET /api/health", func(w http.ResponseWriter, _ *http.Request) { writeJSON(w, 200, map[string]bool{"ok": true}) })
	mux.HandleFunc("GET /api/servers", a.listServers)
	mux.HandleFunc("POST /api/servers", a.registerServer)
	mux.HandleFunc("POST /api/auth/code", a.requestCode)
	mux.HandleFunc("POST /api/auth/verify", a.verifyCode)
	mux.HandleFunc("GET /api/lots", a.maybeUser(a.listLots))
	mux.HandleFunc("GET /api/lots/{kind}/{slug}", a.maybeUser(a.getLot))
	mux.HandleFunc("GET /api/gifts", a.maybeUser(a.listGifts))
	mux.HandleFunc("GET /api/gifts/{id}", a.maybeUser(a.getGift))

	// юзер
	mux.HandleFunc("GET /api/me", a.user(a.me))
	mux.HandleFunc("POST /api/auth/logout", a.user(a.logout))
	mux.HandleFunc("DELETE /api/sessions/{id}", a.user(a.kickSession))
	mux.HandleFunc("POST /api/sessions/kick-others", a.user(a.kickOthers))
	mux.HandleFunc("POST /api/lots/{id}/buy", a.user(a.buy))
	mux.HandleFunc("POST /api/lots/{id}/bid", a.user(a.bid))
	mux.HandleFunc("POST /api/lots/{id}/watch", a.user(a.watch))
	mux.HandleFunc("POST /api/wallet/topup", a.user(a.topUp))
	mux.HandleFunc("POST /api/items/{id}/sell", a.user(a.sell))
	mux.HandleFunc("POST /api/items/{id}/withdraw", a.user(a.withdraw))

	// оператор приватки
	mux.HandleFunc("GET /api/operator/server", a.operator(a.operatorServer))
	mux.HandleFunc("GET /api/operator/lots", a.operator(a.operatorLots))
	mux.HandleFunc("POST /api/operator/lots", a.operator(a.createLot))
	mux.HandleFunc("DELETE /api/operator/lots/{id}", a.operator(a.deleteLot))
	mux.HandleFunc("POST /api/operator/secret", a.operator(a.rotateSecret))

	// владелец lunax
	mux.HandleFunc("GET /api/admin/servers", a.admin(a.adminServers))
	mux.HandleFunc("POST /api/admin/servers/{id}/status", a.admin(a.adminSetStatus))

	return a.cors(a.logRequests(mux))
}

type ctxKey int

const (
	keyUser ctxKey = iota
	keySession
	keyServer
)

func userFrom(ctx context.Context) (domain.User, bool) {
	u, ok := ctx.Value(keyUser).(domain.User)
	return u, ok
}

func sessionFrom(ctx context.Context) domain.Session {
	s, _ := ctx.Value(keySession).(domain.Session)
	return s
}

func serverFrom(ctx context.Context) domain.Server {
	s, _ := ctx.Value(keyServer).(domain.Server)
	return s
}

func (a *API) withUser(r *http.Request) (*http.Request, error) {
	tok := bearer(r)
	if tok == "" {
		return r, domain.ErrUnauthorized
	}
	sess, u, err := a.Auth.Authenticate(r.Context(), tok)
	if err != nil {
		return r, err
	}
	ctx := context.WithValue(r.Context(), keyUser, u)
	ctx = context.WithValue(ctx, keySession, sess)
	return r.WithContext(ctx), nil
}

func (a *API) user(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r, err := a.withUser(r)
		if err != nil {
			writeErr(w, a.Log, err)
			return
		}
		next(w, r)
	}
}

// витрину можно смотреть и без входа
func (a *API) maybeUser(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if bearer(r) != "" {
			if authed, err := a.withUser(r); err == nil {
				r = authed
			}
		}
		next(w, r)
	}
}

func (a *API) operator(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		srv, err := a.Servers.Operator(r.Context(), bearer(r))
		if err != nil {
			writeErr(w, a.Log, err)
			return
		}
		next(w, r.WithContext(context.WithValue(r.Context(), keyServer, srv)))
	}
}

func (a *API) admin(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tok := bearer(r)
		if a.AdminToken == "" || subtle.ConstantTimeCompare([]byte(tok), []byte(a.AdminToken)) != 1 {
			writeErr(w, a.Log, domain.ErrUnauthorized)
			return
		}
		next(w, r)
	}
}

func (a *API) cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if a.CORSOrigin != "" {
			w.Header().Set("Access-Control-Allow-Origin", a.CORSOrigin)
			w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
			w.Header().Set("Vary", "Origin")
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (s *statusWriter) WriteHeader(code int) {
	s.status = code
	s.ResponseWriter.WriteHeader(code)
}

func (a *API) logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		sw := &statusWriter{ResponseWriter: w, status: 200}
		next.ServeHTTP(sw, r)
		a.Log.Info("http", "method", r.Method, "path", r.URL.Path, "status", sw.status, "took", time.Since(start).Round(time.Millisecond))
	})
}
