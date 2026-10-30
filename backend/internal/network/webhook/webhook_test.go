package webhook

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"lunax/internal/domain"
	"lunax/internal/network"
)

const testSecret = "0123456789abcdef"

// приватка сверяет подпись и отвечает по контракту
func TestSignedCalls(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		want := "sha256=" + Sign(testSecret, r.Header.Get("X-LunaX-Timestamp"), body)
		if r.Header.Get("X-LunaX-Signature") != want {
			w.WriteHeader(401)
			return
		}
		var req request
		_ = json.Unmarshal(body, &req)
		switch req.Action {
		case "resolve_phone":
			_ = json.NewEncoder(w).Encode(response{OK: true, UserID: 7})
		case "debit_stars":
			_ = json.NewEncoder(w).Encode(response{OK: false, Error: "no_funds"})
		default:
			_ = json.NewEncoder(w).Encode(response{OK: true})
		}
	}))
	defer srv.Close()

	c, err := New(srv.URL, testSecret)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if err := c.Ping(ctx); err != nil {
		t.Fatal(err)
	}
	if id, err := c.ResolvePhone(ctx, "+79990000000"); err != nil || id != 7 {
		t.Fatalf("got %d %v", id, err)
	}
	if err := c.DebitStars(ctx, network.Op{Key: "k"}, 7, 10); !errors.Is(err, domain.ErrNoFunds) {
		t.Fatalf("want no funds, got %v", err)
	}
}

func TestShortSecretRejected(t *testing.T) {
	if _, err := New("https://x", "short"); err == nil {
		t.Fatal("short secret must be rejected")
	}
}
