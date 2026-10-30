package gramsrv

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"lunax/internal/domain"
	"lunax/internal/network"
	"lunax/internal/network/jsonhttp"
)

const Kind = "gramsrv"

func init() {
	network.Register(Kind, func(apiURL, token string) (network.Network, error) {
		return New(apiURL, token)
	})
}

// адаптер для приваток на gramsrv/telesrv через их Admin API
type Client struct {
	http  *jsonhttp.Client
	actor string
}

func New(apiURL, token string) (*Client, error) {
	if !strings.HasPrefix(apiURL, "http://") && !strings.HasPrefix(apiURL, "https://") {
		return nil, fmt.Errorf("%w: api url must start with http(s)://", domain.ErrInvalid)
	}
	if token == "" {
		return nil, fmt.Errorf("%w: admin api token is empty", domain.ErrInvalid)
	}
	c := jsonhttp.New(apiURL)
	c.Headers = func([]byte) map[string]string {
		return map[string]string{"Authorization": "Bearer " + token}
	}
	return &Client{http: c, actor: "lunax"}, nil
}

// общие поля любой команды gramsrv
type meta struct {
	CommandID string `json:"command_id"`
	Actor     string `json:"actor"`
	Reason    string `json:"reason"`
	DryRun    bool   `json:"dry_run"`
}

func (c *Client) meta(op network.Op) meta {
	sum := sha256.Sum256([]byte(op.Key))
	reason := op.Reason
	if reason == "" {
		reason = "LunaX"
	}
	return meta{CommandID: "lunax-" + hex.EncodeToString(sum[:])[:40], Actor: c.actor, Reason: reason}
}

func (c *Client) Ping(ctx context.Context) error {
	return c.http.Do(ctx, "GET", "/v1/collectible-usernames?limit=1", nil, nil)
}

func (c *Client) ResolvePhone(ctx context.Context, phone string) (int64, error) {
	var out struct {
		Found  bool  `json:"found"`
		UserID int64 `json:"user_id"`
	}
	if err := c.http.Do(ctx, "POST", "/v1/accounts/resolve-by-phone", map[string]string{"phone": phone}, &out); err != nil {
		return 0, err
	}
	if !out.Found || out.UserID <= 0 {
		return 0, domain.ErrNotFound
	}
	return out.UserID, nil
}

// код уходит личным сообщением через рассылку на одного юзера
func (c *Client) SendCode(ctx context.Context, userID int64, text string) error {
	body := struct {
		meta
		Message    string  `json:"message"`
		TargetMode string  `json:"target_mode"`
		UserIDs    []int64 `json:"user_ids"`
	}{
		meta:       c.meta(network.Op{Key: fmt.Sprintf("code:%d:%d", userID, time.Now().UnixNano()), Reason: "LunaX login code"}),
		Message:    text,
		TargetMode: "selected",
		UserIDs:    []int64{userID},
	}
	return c.http.Do(ctx, "POST", "/v1/broadcasts/create", body, nil)
}

func (c *Client) GrantUsername(ctx context.Context, op network.Op, userID int64, username string) error {
	m := c.meta(op)
	mint := struct {
		meta
		Username     string `json:"username"`
		OwnerUserID  string `json:"owner_user_id"`
		Currency     string `json:"currency"`
		Amount       string `json:"amount"`
		PurchaseDate int64  `json:"purchase_date"`
	}{m, username, strconv.FormatInt(userID, 10), "TON", "0", time.Now().Unix()}
	err := c.http.Do(ctx, "POST", "/v1/collectible-usernames/mint", mint, nil)
	if !occupied(err) {
		return err
	}
	// юзернейм уже есть в хранилище сервера, просто передаём
	transfer := struct {
		meta
		Username string `json:"username"`
		ToUserID string `json:"to_user_id"`
	}{m, username, strconv.FormatInt(userID, 10)}
	transfer.CommandID += "-t"
	return c.http.Do(ctx, "POST", "/v1/collectible-usernames/transfer", transfer, nil)
}

func (c *Client) GrantNumber(ctx context.Context, op network.Op, userID int64, phone string) error {
	m := c.meta(op)
	digits := onlyDigits(phone)
	mint := struct {
		meta
		Phone        string `json:"phone"`
		Tier         string `json:"tier"`
		OwnerUserID  string `json:"owner_user_id"`
		Currency     string `json:"currency"`
		Amount       string `json:"amount"`
		PurchaseDate int64  `json:"purchase_date"`
	}{m, digits, "standard", strconv.FormatInt(userID, 10), "TON", "0", time.Now().Unix()}
	err := c.http.Do(ctx, "POST", "/v1/collectible-phones/mint", mint, nil)
	if !occupied(err) {
		return err
	}
	transfer := struct {
		meta
		Phone    string `json:"phone"`
		ToUserID string `json:"to_user_id"`
	}{m, digits, strconv.FormatInt(userID, 10)}
	transfer.CommandID += "-t"
	return c.http.Do(ctx, "POST", "/v1/collectible-phones/transfer", transfer, nil)
}

func (c *Client) GrantGift(ctx context.Context, op network.Op, userID int64, giftID int64) error {
	body := struct {
		meta
		UserID  int64  `json:"user_id"`
		GiftID  int64  `json:"gift_id"`
		Upgrade bool   `json:"upgrade"`
		Message string `json:"message"`
	}{c.meta(op), userID, giftID, true, "Покупка на LunaX"}
	return c.http.Do(ctx, "POST", "/v1/gifts/give", body, nil)
}

func (c *Client) GrantStars(ctx context.Context, op network.Op, userID int64, amount int64) error {
	body := struct {
		meta
		UserID int64 `json:"user_id"`
		Amount int64 `json:"amount"`
	}{c.meta(op), userID, amount}
	return c.http.Do(ctx, "POST", "/v1/accounts/grant-stars", body, nil)
}

func (c *Client) DebitStars(ctx context.Context, op network.Op, userID int64, amount int64) error {
	body := struct {
		meta
		UserID int64 `json:"user_id"`
		Amount int64 `json:"amount"`
	}{c.meta(op), userID, amount}
	err := c.http.Do(ctx, "POST", "/v1/accounts/debit-stars", body, nil)
	var re *jsonhttp.Error
	if errors.As(err, &re) && re.Status == 400 && strings.Contains(strings.ToLower(re.Text), "insufficient") {
		return domain.ErrNoFunds
	}
	return err
}

// сервер сказал что такой ассет уже есть
func occupied(err error) bool {
	var re *jsonhttp.Error
	if !errors.As(err, &re) {
		return false
	}
	s := strings.ToUpper(re.Code + " " + re.Text)
	return strings.Contains(s, "OCCUPIED") || strings.Contains(s, "EXIST") || strings.Contains(s, "ALREADY")
}

func onlyDigits(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}
