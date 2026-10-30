package jsonhttp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// ошибка от чужого сервера с кодом, если он его прислал
type Error struct {
	Status int
	Code   string
	Text   string
}

func (e *Error) Error() string {
	if e.Code != "" {
		return fmt.Sprintf("remote %d %s: %s", e.Status, e.Code, e.Text)
	}
	return fmt.Sprintf("remote %d: %s", e.Status, e.Text)
}

// мелкий клиент для json апи приваток
type Client struct {
	Base    string
	HTTP    *http.Client
	Headers func(body []byte) map[string]string
}

func New(base string) *Client {
	return &Client{
		Base: strings.TrimRight(base, "/"),
		HTTP: &http.Client{
			Timeout: 15 * time.Second,
			CheckRedirect: func(*http.Request, []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
	}
}

func (c *Client) Do(ctx context.Context, method, path string, in, out any) error {
	var body []byte
	if in != nil {
		var err error
		if body, err = json.Marshal(in); err != nil {
			return err
		}
	}
	req, err := http.NewRequestWithContext(ctx, method, c.Base+path, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	if in != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.Headers != nil {
		for k, v := range c.Headers(body) {
			req.Header.Set(k, v)
		}
	}
	res, err := c.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if err != nil {
		return err
	}
	if res.StatusCode >= 300 {
		return parseError(res.StatusCode, raw)
	}
	if out == nil || len(raw) == 0 {
		return nil
	}
	return json.Unmarshal(raw, out)
}

func parseError(status int, raw []byte) error {
	e := &Error{Status: status, Text: strings.TrimSpace(string(raw))}
	var body struct {
		Error   string `json:"error"`
		Message string `json:"message"`
		Code    string `json:"code"`
	}
	if json.Unmarshal(raw, &body) == nil {
		e.Code = body.Code
		switch {
		case body.Error != "":
			e.Text = body.Error
		case body.Message != "":
			e.Text = body.Message
		}
	}
	if len(e.Text) > 300 {
		e.Text = e.Text[:300]
	}
	return e
}
