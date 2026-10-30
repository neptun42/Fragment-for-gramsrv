package service

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"
	"time"

	"lunax/internal/domain"
	"lunax/internal/network"
	"lunax/internal/secret"
	"lunax/internal/store"
)

var serverSlugRe = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{2,31}$`)

type cacheForget interface {
	Forget(serverID int64)
}

// привязка приваток: регистрация, модерация, смена токена
type Servers struct {
	db          store.Servers
	cache       cacheForget
	autoApprove bool
	build       func(kind, apiURL, secret string) (network.Network, error)
}

func NewServers(db store.Servers, cache cacheForget, autoApprove bool) *Servers {
	return &Servers{db: db, cache: cache, autoApprove: autoApprove, build: network.Build}
}

type RegisterInput struct {
	Slug         string
	Name         string
	Kind         string
	APIURL       string
	Secret       string
	Contact      string
	StarsPerGram float64
}

// проверяем что сервер реально отвечает с этим токеном
func (s *Servers) check(ctx context.Context, kind, apiURL, sec string) error {
	net, err := s.build(kind, apiURL, sec)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if err := net.Ping(ctx); err != nil {
		return fmt.Errorf("%w: server did not answer: %v", domain.ErrServerUnavailable, err)
	}
	return nil
}

// возвращает сервер и ключ оператора, ключ показываем один раз
func (s *Servers) Register(ctx context.Context, in RegisterInput) (domain.Server, string, error) {
	in.Slug = strings.ToLower(strings.TrimSpace(in.Slug))
	in.Name = strings.TrimSpace(in.Name)
	in.APIURL = strings.TrimRight(strings.TrimSpace(in.APIURL), "/")
	if !serverSlugRe.MatchString(in.Slug) {
		return domain.Server{}, "", fmt.Errorf("%w: slug must be 3-32 chars a-z0-9-", domain.ErrInvalid)
	}
	if l := len([]rune(in.Name)); l < 2 || l > 40 {
		return domain.Server{}, "", fmt.Errorf("%w: name must be 2-40 chars", domain.ErrInvalid)
	}
	if !slices.Contains(network.Kinds(), in.Kind) {
		return domain.Server{}, "", fmt.Errorf("%w: kind must be one of %v", domain.ErrInvalid, network.Kinds())
	}
	if in.StarsPerGram < 0 || in.StarsPerGram > 10000 {
		return domain.Server{}, "", fmt.Errorf("%w: stars_per_gram must be 0-10000", domain.ErrInvalid)
	}
	if err := s.check(ctx, in.Kind, in.APIURL, in.Secret); err != nil {
		return domain.Server{}, "", err
	}
	status := domain.ServerPending
	if s.autoApprove {
		status = domain.ServerActive
	}
	key := secret.Token("lxo_", 32)
	srv, err := s.db.CreateServer(ctx, domain.Server{
		Slug:         in.Slug,
		Name:         in.Name,
		Kind:         in.Kind,
		APIURL:       in.APIURL,
		Secret:       in.Secret,
		Contact:      strings.TrimSpace(in.Contact),
		StarsPerGram: in.StarsPerGram,
		Status:       status,
		OperatorHash: secret.Hash(key),
	})
	if errors.Is(err, domain.ErrConflict) {
		return srv, "", fmt.Errorf("%w: slug is taken", domain.ErrConflict)
	}
	return srv, key, err
}

func (s *Servers) Public(ctx context.Context) ([]domain.Server, error) {
	return s.db.ListServers(ctx, true)
}

func (s *Servers) All(ctx context.Context) ([]domain.Server, error) {
	return s.db.ListServers(ctx, false)
}

func (s *Servers) BySlug(ctx context.Context, slug string) (domain.Server, error) {
	return s.db.ServerBySlug(ctx, slug)
}

func (s *Servers) SetStatus(ctx context.Context, id int64, status domain.ServerStatus) error {
	switch status {
	case domain.ServerActive, domain.ServerDisabled, domain.ServerPending:
	default:
		return fmt.Errorf("%w: bad status", domain.ErrInvalid)
	}
	if err := s.db.SetServerStatus(ctx, id, status); err != nil {
		return err
	}
	s.cache.Forget(id)
	return nil
}

func (s *Servers) Operator(ctx context.Context, key string) (domain.Server, error) {
	srv, err := s.db.ServerByOperator(ctx, secret.Hash(key))
	if errors.Is(err, domain.ErrNotFound) {
		return srv, domain.ErrUnauthorized
	}
	return srv, err
}

func (s *Servers) RotateSecret(ctx context.Context, srv domain.Server, apiURL, sec string) error {
	apiURL = strings.TrimRight(strings.TrimSpace(apiURL), "/")
	if apiURL == "" {
		apiURL = srv.APIURL
	}
	if err := s.check(ctx, srv.Kind, apiURL, sec); err != nil {
		return err
	}
	if err := s.db.SetServerSecret(ctx, srv.ID, apiURL, sec); err != nil {
		return err
	}
	s.cache.Forget(srv.ID)
	return nil
}
