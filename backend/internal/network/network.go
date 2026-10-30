package network

import (
	"context"
	"fmt"
	"sync"

	"lunax/internal/domain"
)

// поиск юзера на приватке по номеру
type Directory interface {
	ResolvePhone(ctx context.Context, phone string) (int64, error)
}

// отправка кода входа юзеру на приватке
type Messenger interface {
	SendCode(ctx context.Context, userID int64, text string) error
}

// выдача купленного на приватке
type Assets interface {
	GrantUsername(ctx context.Context, op Op, userID int64, username string) error
	GrantNumber(ctx context.Context, op Op, userID int64, phone string) error
	GrantGift(ctx context.Context, op Op, userID int64, giftID int64) error
	GrantStars(ctx context.Context, op Op, userID int64, amount int64) error
}

// списание звёзд с юзера на приватке
type Wallet interface {
	DebitStars(ctx context.Context, op Op, userID int64, amount int64) error
}

type Pinger interface {
	Ping(ctx context.Context) error
}

// всё что умеет привязанный сервер
type Network interface {
	Directory
	Messenger
	Assets
	Wallet
	Pinger
}

// ключ идемпотентности и причина, чтобы повтор не выдал дважды
type Op struct {
	Key    string
	Reason string
}

// конструктор адаптера по типу сервера
type Factory func(apiURL, secret string) (Network, error)

var (
	mu        sync.RWMutex
	factories = map[string]Factory{}
)

// новый тип приватки регистрируется тут, сервисы трогать не надо
func Register(kind string, f Factory) {
	mu.Lock()
	defer mu.Unlock()
	factories[kind] = f
}

func Kinds() []string {
	mu.RLock()
	defer mu.RUnlock()
	out := make([]string, 0, len(factories))
	for k := range factories {
		out = append(out, k)
	}
	return out
}

func Build(kind, apiURL, secret string) (Network, error) {
	mu.RLock()
	f, ok := factories[kind]
	mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("%w: unknown server kind %q", domain.ErrInvalid, kind)
	}
	return f(apiURL, secret)
}
