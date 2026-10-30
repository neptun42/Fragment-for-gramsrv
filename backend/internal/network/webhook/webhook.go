package webhook

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
	"time"

	"lunax/internal/domain"
	"lunax/internal/network"
	"lunax/internal/network/jsonhttp"
)

const Kind = "webhook"

func init() {
	network.Register(Kind, func(apiURL, secret string) (network.Network, error) {
		return New(apiURL, secret)
	})
}

// адаптер для любых других приваток, им надо поднять один эндпоинт по контракту из CONNECT.md
type Client struct {
	http *jsonhttp.Client
}

func New(apiURL, secret string) (*Client, error) {
	if !strings.HasPrefix(apiURL, "http://") && !strings.HasPrefix(apiURL, "https://") {
		return nil, fmt.Errorf("%w: webhook url must start with http(s)://", domain.ErrInvalid)
	}
	if len(secret) < 16 {
		return nil, fmt.Errorf("%w: webhook secret must be at least 16 chars", domain.ErrInvalid)
	}
	c := jsonhttp.New(apiURL)
	c.Headers = func(body []byte) map[string]string {
		ts := strconv.FormatInt(time.Now().Unix(), 10)
		return map[string]string{
			"X-LunaX-Timestamp": ts,
			"X-LunaX-Signature": "sha256=" + Sign(secret, ts, body),
		}
	}
	return &Client{http: c}, nil
}

// подпись hmac от "timestamp.body", приватка сверяет её у себя
func Sign(secret, ts string, body []byte) string {
	m := hmac.New(sha256.New, []byte(secret))
	m.Write([]byte(ts))
	m.Write([]byte("."))
	m.Write(body)
	return hex.EncodeToString(m.Sum(nil))
}

type request struct {
	Action string `json:"action"`
	Key    string `json:"idempotency_key,omitempty"`
	Reason string `json:"reason,omitempty"`
	UserID int64  `json:"user_id,omitempty"`
	Phone  string `json:"phone,omitempty"`
	Text   string `json:"text,omitempty"`
	Name   string `json:"name,omitempty"`
	GiftID int64  `json:"gift_id,omitempty"`
	Amount int64  `json:"amount,omitempty"`
}

type response struct {
	OK     bool   `json:"ok"`
	UserID int64  `json:"user_id"`
	Error  string `json:"error"`
}

func (c *Client) call(ctx context.Context, req request) (response, error) {
	var out response
	if err := c.http.Do(ctx, "POST", "", req, &out); err != nil {
		return out, err
	}
	if !out.OK {
		switch out.Error {
		case "not_found":
			return out, domain.ErrNotFound
		case "no_funds":
			return out, domain.ErrNoFunds
		}
		return out, fmt.Errorf("webhook %s: %s", req.Action, out.Error)
	}
	return out, nil
}

func (c *Client) Ping(ctx context.Context) error {
	_, err := c.call(ctx, request{Action: "ping"})
	return err
}

func (c *Client) ResolvePhone(ctx context.Context, phone string) (int64, error) {
	out, err := c.call(ctx, request{Action: "resolve_phone", Phone: phone})
	if err != nil {
		return 0, err
	}
	if out.UserID <= 0 {
		return 0, domain.ErrNotFound
	}
	return out.UserID, nil
}

func (c *Client) SendCode(ctx context.Context, userID int64, text string) error {
	_, err := c.call(ctx, request{Action: "send_message", UserID: userID, Text: text})
	return err
}

func (c *Client) GrantUsername(ctx context.Context, op network.Op, userID int64, username string) error {
	_, err := c.call(ctx, request{Action: "grant_username", Key: op.Key, Reason: op.Reason, UserID: userID, Name: username})
	return err
}

func (c *Client) GrantNumber(ctx context.Context, op network.Op, userID int64, phone string) error {
	_, err := c.call(ctx, request{Action: "grant_number", Key: op.Key, Reason: op.Reason, UserID: userID, Phone: phone})
	return err
}

func (c *Client) GrantGift(ctx context.Context, op network.Op, userID int64, giftID int64) error {
	_, err := c.call(ctx, request{Action: "grant_gift", Key: op.Key, Reason: op.Reason, UserID: userID, GiftID: giftID})
	return err
}

func (c *Client) GrantStars(ctx context.Context, op network.Op, userID int64, amount int64) error {
	_, err := c.call(ctx, request{Action: "grant_stars", Key: op.Key, Reason: op.Reason, UserID: userID, Amount: amount})
	return err
}

func (c *Client) DebitStars(ctx context.Context, op network.Op, userID int64, amount int64) error {
	_, err := c.call(ctx, request{Action: "debit_stars", Key: op.Key, Reason: op.Reason, UserID: userID, Amount: amount})
	return err
}
