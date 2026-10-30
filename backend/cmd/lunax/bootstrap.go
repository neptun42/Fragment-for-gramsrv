package main

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"lunax/internal/domain"
	"lunax/internal/service"
)

type fileServer struct {
	Slug         string  `json:"slug"`
	Name         string  `json:"name"`
	Kind         string  `json:"kind"`
	APIURL       string  `json:"api_url"`
	Secret       string  `json:"secret"`
	Contact      string  `json:"contact"`
	StarsPerGram float64 `json:"stars_per_gram"`
}

type serverRegistrar interface {
	BySlug(ctx context.Context, slug string) (domain.Server, error)
	Register(ctx context.Context, in service.RegisterInput) (domain.Server, string, error)
	SetStatus(ctx context.Context, id int64, status domain.ServerStatus) error
}

// самый лёгкий способ привязать свою приватку: вписать её в servers.json
func loadServersFile(ctx context.Context, path string, servers serverRegistrar, log *slog.Logger) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var list []fileServer
	if err := json.Unmarshal(raw, &list); err != nil {
		return err
	}
	for _, s := range list {
		s.Secret = os.ExpandEnv(s.Secret)
		if _, err := servers.BySlug(ctx, s.Slug); err == nil {
			continue
		} else if !errors.Is(err, domain.ErrNotFound) {
			return err
		}
		srv, key, err := servers.Register(ctx, service.RegisterInput(s))
		if err != nil {
			log.Warn("server from file skipped", "slug", s.Slug, "err", err)
			continue
		}
		// сервер из файла добавил сам владелец, сразу включаем
		if srv.Status != domain.ServerActive {
			if err := servers.SetStatus(ctx, srv.ID, domain.ServerActive); err != nil {
				return err
			}
		}
		log.Info("server bound from file, save the operator key, it is shown once", "slug", srv.Slug, "operator_key", key)
	}
	return nil
}

// отдаём собранный фронт, все неизвестные пути на index.html
func withStatic(api http.Handler, dir string) http.Handler {
	if dir == "" {
		return api
	}
	files := http.FileServer(http.Dir(dir))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") {
			api.ServeHTTP(w, r)
			return
		}
		p := filepath.Join(dir, filepath.Clean("/"+r.URL.Path))
		if st, err := os.Stat(p); err != nil || st.IsDir() {
			http.ServeFile(w, r, filepath.Join(dir, "index.html"))
			return
		}
		files.ServeHTTP(w, r)
	})
}
