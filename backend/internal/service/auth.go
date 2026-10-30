package service

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"errors"
	"fmt"
	"math/big"
	"strconv"
	"time"

	"lunax/internal/domain"
	"lunax/internal/secret"
	"lunax/internal/store"
)

const (
	codeLen      = 5
	codeTTL      = 5 * time.Minute
	codeAttempts = 5
	codesPer10m  = 3
)

var ErrTooManyCodes = errors.New("too many codes, try later")

type authStore interface {
	ServerBySlug(ctx context.Context, slug string) (domain.Server, error)
	store.Codes
	store.Users
	store.Sessions
	store.TxRunner
}

// вход по коду, который приходит в чат на приватке
type Auth struct {
	db    authStore
	nets  Networks
	bonus domain.Cents
	now   func() time.Time
}

func NewAuth(db authStore, nets Networks, bonus domain.Cents) *Auth {
	return &Auth{db: db, nets: nets, bonus: bonus, now: time.Now}
}

type LoginResult struct {
	Token   string
	User    domain.User
	Session domain.Session
	Bonus   domain.Cents
}

func codeHash(serverID int64, phone, code string) []byte {
	return secret.Hash(strconv.FormatInt(serverID, 10) + ":" + phone + ":" + code)
}

func randomCode() string {
	n, err := rand.Int(rand.Reader, big.NewInt(100000))
	if err != nil {
		panic(err)
	}
	return fmt.Sprintf("%0*d", codeLen, n.Int64())
}

func (a *Auth) RequestCode(ctx context.Context, serverSlug, rawPhone string) error {
	phone := digits(rawPhone)
	if len(phone) < 7 || len(phone) > 15 {
		return fmt.Errorf("%w: bad phone", domain.ErrInvalid)
	}
	srv, err := a.db.ServerBySlug(ctx, serverSlug)
	if err != nil {
		return err
	}
	n, err := a.db.CountCodesSince(ctx, srv.ID, phone, a.now().Add(-10*time.Minute))
	if err != nil {
		return err
	}
	if n >= codesPer10m {
		return ErrTooManyCodes
	}
	net, err := a.nets.Get(ctx, srv.ID)
	if err != nil {
		return err
	}
	userID, err := net.ResolvePhone(ctx, "+"+phone)
	if err != nil {
		return err
	}
	code := randomCode()
	err = a.db.SaveCode(ctx, domain.LoginCode{
		ServerID:     srv.ID,
		Phone:        phone,
		ServerUserID: userID,
		Hash:         codeHash(srv.ID, phone, code),
		ExpiresAt:    a.now().Add(codeTTL),
	})
	if err != nil {
		return err
	}
	text := fmt.Sprintf("Код для входа в LunaX: %s\n\nНикому его не сообщайте. Если это были не вы, просто проигнорируйте сообщение.", code)
	return net.SendCode(ctx, userID, text)
}

func (a *Auth) VerifyCode(ctx context.Context, serverSlug, rawPhone, code, device, place string) (LoginResult, error) {
	var res LoginResult
	phone := digits(rawPhone)
	srv, err := a.db.ServerBySlug(ctx, serverSlug)
	if err != nil {
		return res, err
	}
	c, err := a.db.LatestCode(ctx, srv.ID, phone)
	if errors.Is(err, domain.ErrNotFound) {
		return res, fmt.Errorf("%w: code expired", domain.ErrInvalid)
	}
	if err != nil {
		return res, err
	}
	if a.now().After(c.ExpiresAt) || c.Attempts >= codeAttempts {
		return res, fmt.Errorf("%w: code expired", domain.ErrInvalid)
	}
	if subtle.ConstantTimeCompare(c.Hash, codeHash(srv.ID, phone, code)) != 1 {
		_ = a.db.BumpCodeAttempts(ctx, c.ID)
		return res, fmt.Errorf("%w: wrong code", domain.ErrInvalid)
	}
	if err := a.db.DeleteCodes(ctx, srv.ID, phone); err != nil {
		return res, err
	}

	var user domain.User
	err = a.db.InTx(ctx, func(tx store.Tx) error {
		var created bool
		user, created, err = tx.UpsertUser(ctx, domain.User{ServerID: srv.ID, ServerUserID: c.ServerUserID, Phone: "+" + phone})
		if err != nil || !created || a.bonus <= 0 {
			return err
		}
		// бонус только новым юзерам
		if err := tx.Credit(ctx, user.ID, a.bonus, domain.HistoryTopup, "Приветственный бонус"); err != nil {
			return err
		}
		user.Balance += a.bonus
		res.Bonus = a.bonus
		return nil
	})
	if err != nil {
		return res, err
	}

	token := secret.Token("lx_", 32)
	sess := domain.Session{ID: secret.Hex(8), UserID: user.ID, Device: clip(device, 80), Place: clip(place, 80), CreatedAt: a.now(), LastSeen: a.now()}
	if err := a.db.CreateSession(ctx, sess, secret.Hash(token)); err != nil {
		return res, err
	}
	res.Token, res.User, res.Session = token, user, sess
	return res, nil
}

// проверка токена на каждый запрос
func (a *Auth) Authenticate(ctx context.Context, token string) (domain.Session, domain.User, error) {
	sess, err := a.db.SessionByToken(ctx, secret.Hash(token))
	if errors.Is(err, domain.ErrNotFound) {
		return sess, domain.User{}, domain.ErrUnauthorized
	}
	if err != nil {
		return sess, domain.User{}, err
	}
	if a.now().Sub(sess.LastSeen) > time.Minute {
		_ = a.db.TouchSession(ctx, sess.ID)
	}
	user, err := a.db.GetUser(ctx, sess.UserID)
	return sess, user, err
}

func (a *Auth) Sessions(ctx context.Context, userID int64) ([]domain.Session, error) {
	return a.db.ListSessions(ctx, userID)
}

func (a *Auth) Logout(ctx context.Context, userID int64, sessionID string) error {
	return a.db.DeleteSession(ctx, userID, sessionID)
}

func (a *Auth) Kick(ctx context.Context, userID int64, current, target string) error {
	if current == target {
		return fmt.Errorf("%w: use logout for current session", domain.ErrInvalid)
	}
	return a.db.DeleteSession(ctx, userID, target)
}

func (a *Auth) KickOthers(ctx context.Context, userID int64, current string) error {
	return a.db.DeleteOtherSessions(ctx, userID, current)
}

func clip(s string, n int) string {
	r := []rune(s)
	if len(r) > n {
		return string(r[:n])
	}
	return s
}
