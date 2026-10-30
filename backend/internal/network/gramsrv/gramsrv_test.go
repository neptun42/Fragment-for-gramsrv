package gramsrv

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"lunax/internal/domain"
	"lunax/internal/network"
)

// фейковый gramsrv, запоминает какие ручки дёргали
func fake(t *testing.T, handle func(path string, body map[string]any) (int, any)) (*Client, *[]string) {
	t.Helper()
	var calls []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer tok" {
			w.WriteHeader(401)
			return
		}
		calls = append(calls, r.URL.Path)
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		status, out := handle(r.URL.Path, body)
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(out)
	}))
	t.Cleanup(srv.Close)
	c, err := New(srv.URL, "tok")
	if err != nil {
		t.Fatal(err)
	}
	return c, &calls
}

func TestResolvePhone(t *testing.T) {
	c, _ := fake(t, func(_ string, body map[string]any) (int, any) {
		if body["phone"] == "+79990000000" {
			return 200, map[string]any{"found": true, "user_id": 42}
		}
		return 200, map[string]any{"found": false, "user_id": 0}
	})
	id, err := c.ResolvePhone(context.Background(), "+79990000000")
	if err != nil || id != 42 {
		t.Fatalf("got %d %v", id, err)
	}
	if _, err := c.ResolvePhone(context.Background(), "+70000000000"); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("want not found, got %v", err)
	}
}

// если юзернейм уже есть на сервере, делаем transfer вместо mint
func TestGrantUsernameFallsBackToTransfer(t *testing.T) {
	c, calls := fake(t, func(path string, _ map[string]any) (int, any) {
		if path == "/v1/collectible-usernames/mint" {
			return 400, map[string]any{"status": "failed", "error": "username occupied", "code": "USERNAME_OCCUPIED"}
		}
		return 200, map[string]any{"status": "executed"}
	})
	err := c.GrantUsername(context.Background(), network.Op{Key: "item:1"}, 42, "lunar")
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"/v1/collectible-usernames/mint", "/v1/collectible-usernames/transfer"}
	if len(*calls) != 2 || (*calls)[0] != want[0] || (*calls)[1] != want[1] {
		t.Fatalf("calls = %v", *calls)
	}
}

func TestCommandIDIsStable(t *testing.T) {
	c, _ := New("http://x", "tok")
	a := c.meta(network.Op{Key: "item:7"}).CommandID
	b := c.meta(network.Op{Key: "item:7"}).CommandID
	if a != b || a == c.meta(network.Op{Key: "item:8"}).CommandID {
		t.Fatal("command id must depend only on op key")
	}
}
