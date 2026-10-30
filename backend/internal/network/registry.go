package network

import (
	"context"
	"fmt"
	"sync"

	"lunax/internal/domain"
)

type ServerSource interface {
	GetServer(ctx context.Context, id int64) (domain.Server, error)
}

// кэш адаптеров по id сервера
type Registry struct {
	src   ServerSource
	mu    sync.Mutex
	cache map[int64]Network
}

func NewRegistry(src ServerSource) *Registry {
	return &Registry{src: src, cache: map[int64]Network{}}
}

func (r *Registry) Get(ctx context.Context, serverID int64) (Network, error) {
	r.mu.Lock()
	n, ok := r.cache[serverID]
	r.mu.Unlock()
	if ok {
		return n, nil
	}
	s, err := r.src.GetServer(ctx, serverID)
	if err != nil {
		return nil, err
	}
	if s.Status != domain.ServerActive {
		return nil, fmt.Errorf("%w: server %s is %s", domain.ErrServerUnavailable, s.Slug, s.Status)
	}
	n, err = Build(s.Kind, s.APIURL, s.Secret)
	if err != nil {
		return nil, err
	}
	r.mu.Lock()
	r.cache[serverID] = n
	r.mu.Unlock()
	return n, nil
}

// сбрасываем после смены токена или статуса
func (r *Registry) Forget(serverID int64) {
	r.mu.Lock()
	delete(r.cache, serverID)
	r.mu.Unlock()
}
